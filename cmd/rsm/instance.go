package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yakovlev-alex/reforger-server-manager/internal/config"
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/systemd"
)

var instanceCmd = &cobra.Command{
	Use:   "instance",
	Short: "Manage server instances",
}

var flagNewExperimental bool

var instanceNewCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new server instance",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = args[0]
		}
		return runInstanceNew(cmd, []string{name})
	},
}

var instanceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all server instances",
	RunE:  runInstanceList,
}

var instanceDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a server instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstanceDelete,
}

var instanceStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show detailed status of an instance",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInstanceStatus,
}

func init() {
	instanceNewCmd.Flags().BoolVar(&flagNewExperimental, "experimental", false, "Use the experimental (beta) branch of the Arma Reforger Server")
	instanceCmd.AddCommand(instanceNewCmd)
	instanceCmd.AddCommand(instanceListCmd)
	instanceCmd.AddCommand(instanceDeleteCmd)
	instanceCmd.AddCommand(instanceStatusCmd)
	rootCmd.AddCommand(instanceCmd)
}

func runInstanceNew(_ *cobra.Command, args []string) error {
	fmt.Println(color.CyanString("=== Create New Server Instance ==="))
	fmt.Println()

	// --- Step 1: resolve instance name first so we can derive the install dir default ---
	instanceName := ""
	if len(args) > 0 && args[0] != "" {
		instanceName = args[0]
	}
	if instanceName == "" {
		if err := survey.AskOne(&survey.Input{
			Message: "Instance name (used as systemd service identifier):",
			Default: "main",
			Help:    "Lowercase letters, numbers, and hyphens only. E.g. 'main', 'modded'",
		}, &instanceName, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}

	// --- Step 2: derive defaults that depend on the name ---

	// Default system user: the currently logged-in user (never hardcode "steam")
	currentUser := ""
	if u, err := user.Current(); err == nil {
		currentUser = u.Username
	}

	// Default install dir: <cwd>/<instance-name>
	// This is intuitive — the user is likely already in the right parent directory.
	cwd, err := filepath.Abs(".")
	if err != nil {
		cwd = "."
	}
	defaultInstallDir := filepath.Join(cwd, instanceName)

	// --- Step 3: ask the remaining questions ---
	installDir := ""
	if err := survey.AskOne(&survey.Input{
		Message: "Server installation directory:",
		Default: defaultInstallDir,
		Help:    "Where ArmaReforgerServer binary will be installed",
	}, &installDir, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	systemUser := ""
	if err := survey.AskOne(&survey.Input{
		Message: "System user to run the server as:",
		Default: currentUser,
		Help:    "The OS user account the server process will run under",
	}, &systemUser, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	var answers struct {
		Name       string
		InstallDir string
		SystemUser string
	}
	answers.Name = instanceName
	answers.InstallDir = installDir
	answers.SystemUser = systemUser

	// MaxFPS
	maxFPSStr := "60"
	if err := survey.AskOne(&survey.Input{
		Message: "Maximum FPS (server tick rate):",
		Default: "60",
	}, &maxFPSStr); err != nil {
		return err
	}
	maxFPS := 60
	fmt.Sscanf(maxFPSStr, "%d", &maxFPS)

	// Extra flags
	extraFlagsChoices := []string{}
	extraFlagsPrompt := &survey.MultiSelect{
		Message: "Enable extra launch flags:",
		Options: []string{
			"-loadSessionSave",
			"-backendLocalStorage",
			"-logStats 3000",
		},
		Default: []string{"-loadSessionSave", "-backendLocalStorage"},
	}
	if err := survey.AskOne(extraFlagsPrompt, &extraFlagsChoices); err != nil {
		return err
	}

	// Experimental branch — use flag value if set, otherwise ask
	useExperimental := flagNewExperimental
	if !flagNewExperimental {
		if err := survey.AskOne(&survey.Confirm{
			Message: "Use experimental (beta) server branch?",
			Default: false,
			Help:    "Installs the 'experiment' Steam beta branch. May be unstable.",
		}, &useExperimental); err != nil {
			return err
		}
	}
	if useExperimental {
		printWarning("Experimental branch selected — this build may be unstable.")
	}

	// Validate instance name doesn't already exist
	if _, err := instance.Load(answers.Name); err == nil {
		return fmt.Errorf("instance %q already exists", answers.Name)
	}

	// Create instance
	inst := &instance.Instance{
		Name:            answers.Name,
		InstallDir:      answers.InstallDir,
		ActiveConfig:    "",
		UpdateOnRestart: false,
		Experimental:    useExperimental,
		MaxFPS:          maxFPS,
		ExtraFlags:      extraFlagsChoices,
		SystemdUser:     answers.SystemUser,
	}

	// Install server?
	doInstall := false
	cfg, err := config.LoadGlobal()
	if err != nil {
		return err
	}

	if cfg.SteamCMDPath != "" {
		installPrompt := &survey.Confirm{
			Message: fmt.Sprintf("Install/verify server files in %s now?", answers.InstallDir),
			Default: true,
		}
		if err := survey.AskOne(installPrompt, &doInstall); err != nil {
			return err
		}
	} else {
		printWarning("steamcmd not configured — skipping server installation.")
		printInfo("Run 'rsm init' to configure steamcmd, then 'rsm install' to install server files.")
	}

	// Save instance metadata (directories created by Save)
	if err := inst.Save(); err != nil {
		return fmt.Errorf("saving instance: %w", err)
	}
	printSuccess("Instance %q created.", inst.Name)

	// Run install
	if doInstall {
		fmt.Println()
		if err := runInstallForInstance(inst, cfg.SteamCMDPath); err != nil {
			printWarning("Installation failed: %v", err)
			printInfo("You can retry with: rsm install --instance %s", inst.Name)
		}
	}

	// Create first configuration
	fmt.Println()
	printInfo("Now let's create the first configuration for this instance.")
	if err := createConfigWizard(inst, ""); err != nil {
		return err
	}

	// Generate and install systemd unit
	fmt.Println()
	installUnit := false
	unitPrompt := &survey.Confirm{
		Message: "Generate and install systemd service unit? (requires sudo)",
		Default: true,
	}
	if err := survey.AskOne(unitPrompt, &installUnit); err != nil {
		return err
	}

	if installUnit {
		if err := systemd.InstallUnit(inst, cfg.SteamCMDPath); err != nil {
			printWarning("Could not install systemd unit: %v", err)
			printInfo("You can install it manually — unit file saved to:")
			unitPath, _ := inst.ServiceUnitPath()
			fmt.Println(" ", unitPath)
		} else {
			printSuccess("systemd unit installed: rsm-%s.service", inst.Name)
		}
	}

	// Offer to enable + start
	if installUnit {
		fmt.Println()
		enableNow := false
		enablePrompt := &survey.Confirm{
			Message: "Enable autostart (systemctl enable)?",
			Default: true,
		}
		if err := survey.AskOne(enablePrompt, &enableNow); err != nil {
			return err
		}
		if enableNow {
			if err := systemd.Enable(inst); err != nil {
				printWarning("Enable failed: %v", err)
			} else {
				printSuccess("Autostart enabled.")
			}
		}

		startNow := false
		startPrompt := &survey.Confirm{
			Message: "Start the server now?",
			Default: false,
		}
		if err := survey.AskOne(startPrompt, &startNow); err != nil {
			return err
		}
		if startNow {
			if err := systemd.Start(inst); err != nil {
				return fmt.Errorf("starting server: %w", err)
			}
			printSuccess("Server started.")
		}
	}

	fmt.Println()
	printSuccess("Instance %q is ready.", inst.Name)
	printInstanceQuickHelp(inst.Name)
	return nil
}

func runInstanceList(_ *cobra.Command, _ []string) error {
	names, err := instance.List()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		printInfo("No instances found. Run 'rsm instance new' to create one.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, color.HiWhiteString("NAME\tACTIVE CONFIG\tSTATUS\tAUTOSTART\tINSTALL DIR"))
	fmt.Fprintln(w, strings.Repeat("-", 80))

	for _, name := range names {
		inst, err := instance.Load(name)
		if err != nil {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", name, "?", "error", "?", "?")
			continue
		}

		status := color.RedString("stopped")
		if systemd.IsActive(inst) {
			status = color.GreenString("running")
		}

		autostart := color.RedString("disabled")
		if systemd.IsEnabled(inst) {
			autostart = color.GreenString("enabled")
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			color.HiCyanString(inst.Name),
			inst.ActiveConfig,
			status,
			autostart,
			inst.InstallDir,
		)
	}
	w.Flush()
	return nil
}

func runInstanceDelete(_ *cobra.Command, args []string) error {
	name := args[0]
	inst, err := instance.Load(name)
	if err != nil {
		return err
	}

	// Stop if running
	if systemd.IsActive(inst) {
		stopFirst := false
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Instance %q is running. Stop it first?", name),
			Default: true,
		}, &stopFirst); err != nil {
			return err
		}
		if stopFirst {
			_ = systemd.Stop(inst)
		}
	}

	// Ask about install dir
	wipeFiles := false
	if err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Also delete server files in %s? (this is irreversible)", inst.InstallDir),
		Default: false,
	}, &wipeFiles); err != nil {
		return err
	}

	// Confirm
	confirmed := false
	if err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Are you sure you want to delete instance %q?", name),
		Default: false,
	}, &confirmed); err != nil {
		return err
	}
	if !confirmed {
		printInfo("Aborted.")
		return nil
	}

	// Remove systemd unit
	_ = systemd.Disable(inst)
	_ = systemd.RemoveUnit(inst)

	if err := instance.Delete(name, wipeFiles); err != nil {
		return err
	}
	printSuccess("Instance %q deleted.", name)
	return nil
}

func runInstanceStatus(_ *cobra.Command, args []string) error {
	name := flagInstance
	if len(args) > 0 && args[0] != "" {
		name = args[0]
	}

	resolved, err := instance.ResolveInstance(name)
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	branch := "stable"
	if inst.Experimental {
		branch = color.YellowString("experimental")
	}

	fmt.Println(color.HiCyanString("Instance: %s", inst.Name))
	fmt.Printf("  Install dir:     %s\n", inst.InstallDir)
	fmt.Printf("  Branch:          %s\n", branch)
	fmt.Printf("  Active config:   %s\n", inst.ActiveConfig)
	fmt.Printf("  Max FPS:         %d\n", inst.MaxFPS)
	fmt.Printf("  Extra flags:     %s\n", strings.Join(inst.ExtraFlags, " "))
	fmt.Printf("  Update on start: %v\n", inst.UpdateOnRestart)
	fmt.Printf("  Systemd user:    %s\n", inst.SystemdUser)
	fmt.Printf("  Service name:    %s\n", inst.SystemdServiceName())
	fmt.Println()

	configs, _ := inst.ListConfigs()
	fmt.Printf("  Configurations:  %s\n", strings.Join(configs, ", "))
	fmt.Println()

	status, _ := systemd.Status(inst)
	fmt.Println(color.HiWhiteString("systemd status:"))
	fmt.Println(status)
	return nil
}

func printInstanceQuickHelp(name string) {
	fmt.Println(color.HiWhiteString("Quick commands:"))
	fmt.Printf("  rsm start -i %s         — start the server\n", name)
	fmt.Printf("  rsm stop -i %s          — stop the server\n", name)
	fmt.Printf("  rsm logs -i %s          — view logs\n", name)
	fmt.Printf("  rsm config list -i %s   — list configurations\n", name)
	fmt.Printf("  rsm config use <name> -i %s — switch configuration\n", name)
}
