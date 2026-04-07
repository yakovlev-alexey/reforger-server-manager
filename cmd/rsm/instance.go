package main

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/steam"
	"github.com/yakovlev-alex/reforger-server-manager/internal/systemd"
)

var flagNewExperimental bool

// initCmd replaces the old "rsm instance new" — it is the primary entry point
// for setting up a new Arma Reforger server instance.
var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Set up a new Arma Reforger server instance",
	Long: `Guided wizard to create a new server instance.

Walks through:
  1. Instance name and install directory
  2. Generating a server configuration
  3. Downloading server files via steamcmd
  4. Installing a systemd service unit
  5. Enabling autostart and starting the server`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		return runInstanceNew(name)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Remove this instance's registration (optionally wipe server files)",
	RunE:  runInstanceDelete,
}

func init() {
	initCmd.Flags().BoolVar(&flagNewExperimental, "experimental", false, "Use the experimental (beta) branch of the Arma Reforger Server")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(deleteCmd)
}

func runInstanceNew(nameArg string) error {
	fmt.Println(color.CyanString("=== Set Up New Server Instance ==="))
	fmt.Println()

	// --- Step 1: instance name ---
	instanceName := nameArg
	if instanceName == "" {
		if err := survey.AskOne(&survey.Input{
			Message: "Instance name (used as systemd service identifier):",
			Default: "main",
			Help:    "Lowercase letters, numbers, and hyphens only. E.g. 'main', 'modded'",
		}, &instanceName, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}

	// --- Step 2: defaults that depend on name ---
	currentUser := ""
	if u, err := user.Current(); err == nil {
		currentUser = u.Username
	}

	cwd, err := filepath.Abs(".")
	if err != nil {
		cwd = "."
	}
	defaultInstallDir := filepath.Join(cwd, instanceName)

	// --- Step 3: remaining instance questions ---
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

	maxFPSStr := "60"
	if err := survey.AskOne(&survey.Input{
		Message: "Maximum FPS (server tick rate):",
		Default: "60",
	}, &maxFPSStr); err != nil {
		return err
	}
	maxFPS := 60
	fmt.Sscanf(maxFPSStr, "%d", &maxFPS)

	extraFlagsChoices := []string{}
	if err := survey.AskOne(&survey.MultiSelect{
		Message: "Enable extra launch flags:",
		Options: []string{"-loadSessionSave", "-backendLocalStorage", "-logStats 3000"},
		Default: []string{"-loadSessionSave", "-backendLocalStorage"},
	}, &extraFlagsChoices); err != nil {
		return err
	}

	periodicRestart, err := promptPeriodicRestart()
	if err != nil {
		return err
	}

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

	// --- Validate: no duplicate name in registry ---
	if existing, err := instance.Load(instanceName); err == nil && existing != nil {
		return fmt.Errorf("instance %q already exists (install dir: %s)", instanceName, existing.InstallDir)
	}

	inst := &instance.Instance{
		Name:            instanceName,
		InstallDir:      installDir,
		ActiveConfig:    "",
		UpdateOnRestart: false,
		Experimental:    useExperimental,
		MaxFPS:          maxFPS,
		ExtraFlags:      extraFlagsChoices,
		SystemdUser:     systemUser,
		PeriodicRestart: periodicRestart,
	}

	// --- Save rsm.yaml + register ---
	if err := inst.Save(); err != nil {
		return fmt.Errorf("saving instance: %w", err)
	}
	if err := instance.Register(inst); err != nil {
		return fmt.Errorf("registering instance: %w", err)
	}
	printSuccess("Instance %q created.", inst.Name)

	// ── Step: create first configuration ────────────────────────────────────
	// Config is asked before install because it's fast (just questions) and
	// gives the user something productive to do before the slow download.
	fmt.Println()
	doConfig := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Configure the server now?",
		Default: true,
		Help:    "Sets up config.json (server name, ports, scenario, passwords, etc.)",
	}, &doConfig); err != nil {
		return err
	}

	if doConfig {
		fmt.Println()
		if err := createConfigWizard(inst, ""); err != nil {
			return err
		}
		if updated, err := instance.Load(inst.Name); err == nil {
			inst = updated
		}
	} else {
		fmt.Println()
		printNextStep("When ready to configure, run:", "rsm config new")
		return nil
	}

	// ── Step: install server files ──────────────────────────────────────────
	fmt.Println()
	doInstall := false
	if err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Download and install server files into %s now?", installDir),
		Default: true,
		Help:    "Runs steamcmd to download the Arma Reforger dedicated server. This may take a while.",
	}, &doInstall); err != nil {
		return err
	}

	if doInstall {
		steamcmdPath, steamErr := steam.Require()
		if steamErr != nil {
			fmt.Println()
			printNextStep("Once steamcmd is installed, run:", "rsm install")
			return steamErr
		}
		fmt.Println()
		if err := runInstallForInstance(inst, steamcmdPath); err != nil {
			printWarning("Installation failed: %v", err)
			fmt.Println()
			printNextStep("Retry the install with:", "rsm install")
			return nil
		}
	} else {
		fmt.Println()
		printNextStep("When ready to install the server, run:", "rsm install")
		return nil
	}

	// ── Step: install systemd unit ───────────────────────────────────────────
	fmt.Println()
	doUnit := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Install systemd service unit for autostart management? (requires sudo)",
		Default: true,
	}, &doUnit); err != nil {
		return err
	}

	steamcmdPath := findSteamCMD()
	if doUnit {
		if err := systemd.InstallUnit(inst, steamcmdPath); err != nil {
			printWarning("Could not install systemd unit: %v", err)
			unitPath := inst.ServiceUnitPath()
			printInfo("Unit file saved locally at: %s", unitPath)
			fmt.Println()
			printNextStep("Install it manually with:",
				fmt.Sprintf("sudo cp %s /etc/systemd/system/%s && sudo systemctl daemon-reload",
					unitPath, inst.SystemdServiceName()))
			return nil
		}
		printSuccess("systemd unit installed: %s", inst.SystemdServiceName())

		if inst.PeriodicRestart != "" {
			if err := systemd.InstallRestartTimer(inst); err != nil {
				printWarning("Could not install periodic restart timer: %v", err)
			} else {
				printSuccess("Periodic restart timer installed (%s interval).", inst.PeriodicRestart)
			}
		}
	} else {
		fmt.Println()
		printNextStep("To set up autostart later, run:", "rsm enable")
		return nil
	}

	// ── Step: enable autostart ───────────────────────────────────────────────
	fmt.Println()
	doEnable := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Enable autostart on boot? (systemctl enable)",
		Default: true,
	}, &doEnable); err != nil {
		return err
	}

	if doEnable {
		if err := systemd.Enable(inst); err != nil {
			printWarning("Enable failed: %v", err)
		} else {
			printSuccess("Autostart enabled.")
		}
	} else {
		fmt.Println()
		printNextStep("To enable autostart later, run:", "rsm enable")
	}

	// ── Step: start now ──────────────────────────────────────────────────────
	fmt.Println()
	doStart := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Start the server now?",
		Default: true,
	}, &doStart); err != nil {
		return err
	}

	if doStart {
		if err := systemd.Start(inst); err != nil {
			printWarning("Failed to start: %v", err)
			fmt.Println()
			printNextStep("Start it manually with:", "rsm start")
		} else {
			printSuccess("Server started.")
			fmt.Println()
			printInfo("Follow logs with: rsm logs -i %s -f", inst.Name)
		}
	} else {
		fmt.Println()
		printNextStep("Start the server when ready with:", "rsm start")
	}

	fmt.Println()
	return nil
}

func runInstanceDelete(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance("")
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	if systemd.IsActive(inst) {
		stopFirst := false
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Instance %q is running. Stop it first?", inst.Name),
			Default: true,
		}, &stopFirst); err != nil {
			return err
		}
		if stopFirst {
			_ = systemd.Stop(inst)
		}
	}

	wipeFiles := false
	if err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Also delete server files in %s? (irreversible)", inst.InstallDir),
		Default: false,
	}, &wipeFiles); err != nil {
		return err
	}

	confirmed := false
	if err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Are you sure you want to delete instance %q?", inst.Name),
		Default: false,
	}, &confirmed); err != nil {
		return err
	}
	if !confirmed {
		printInfo("Aborted.")
		return nil
	}

	_ = systemd.Disable(inst)
	_ = systemd.RemoveUnit(inst)
	if inst.PeriodicRestart != "" || systemd.IsRestartTimerInstalled(inst) {
		_ = systemd.RemoveRestartTimer(inst)
	}

	if err := instance.Delete(inst.Name, wipeFiles); err != nil {
		return err
	}
	printSuccess("Instance %q removed.", inst.Name)
	return nil
}

// promptPeriodicRestart asks the user whether to enable periodic restarts and,
// if so, on what interval. Returns the interval string (e.g. "6h") or "".
func promptPeriodicRestart() (string, error) {
	enablePeriodic := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Enable periodic automatic restarts?",
		Default: false,
		Help:    "Installs a systemd timer that restarts the server on a fixed schedule.",
	}, &enablePeriodic); err != nil {
		return "", err
	}
	if !enablePeriodic {
		return "", nil
	}

	intervalOptions := []string{
		"6h  — every 6 hours",
		"12h — every 12 hours",
		"1d  — once a day",
		"2d  — every 2 days",
		"Custom (enter manually)",
	}
	choice := intervalOptions[1] // default: 12h
	if err := survey.AskOne(&survey.Select{
		Message: "Restart interval:",
		Options: intervalOptions,
		Default: intervalOptions[1],
		Help:    "How often the server should be automatically restarted.",
	}, &choice); err != nil {
		return "", err
	}

	switch {
	case strings.HasPrefix(choice, "6h"):
		return "6h", nil
	case strings.HasPrefix(choice, "12h"):
		return "12h", nil
	case strings.HasPrefix(choice, "1d"):
		return "1d", nil
	case strings.HasPrefix(choice, "2d"):
		return "2d", nil
	default:
		interval := ""
		if err := survey.AskOne(&survey.Input{
			Message: "Restart interval (systemd time span, e.g. 6h, 12h, 1d):",
			Help:    "Use systemd time span format: Nh for hours, Nd for days. E.g. '6h', '12h', '1d'.",
		}, &interval, survey.WithValidator(survey.Required)); err != nil {
			return "", err
		}
		return strings.TrimSpace(interval), nil
	}
}

func runInstanceStatus(_ *cobra.Command, args []string) error {
	name := ""
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

	periodicRestartStr := "disabled"
	if inst.PeriodicRestart != "" {
		timerStatus := "not installed"
		if systemd.IsRestartTimerActive(inst) {
			timerStatus = color.GreenString("active")
		} else if systemd.IsRestartTimerInstalled(inst) {
			timerStatus = color.YellowString("installed, not active")
		}
		periodicRestartStr = fmt.Sprintf("every %s (%s)", inst.PeriodicRestart, timerStatus)
	}

	fmt.Println(color.HiCyanString("Instance: %s", inst.Name))
	fmt.Printf("  Install dir:      %s\n", inst.InstallDir)
	fmt.Printf("  Branch:           %s\n", branch)
	fmt.Printf("  Active config:    %s\n", inst.ActiveConfig)
	fmt.Printf("  Max FPS:          %d\n", inst.MaxFPS)
	fmt.Printf("  Extra flags:      %s\n", strings.Join(inst.ExtraFlags, " "))
	fmt.Printf("  Update on start:  %v\n", inst.UpdateOnRestart)
	fmt.Printf("  Periodic restart: %s\n", periodicRestartStr)
	fmt.Printf("  Systemd user:     %s\n", inst.SystemdUser)
	fmt.Printf("  Service name:     %s\n", inst.SystemdServiceName())
	fmt.Println()

	configs, _ := inst.ListConfigs()
	fmt.Printf("  Configurations:  %s\n", strings.Join(configs, ", "))
	fmt.Println()

	status, _ := systemd.Status(inst)
	fmt.Println(color.HiWhiteString("systemd status:"))
	fmt.Println(status)
	return nil
}
