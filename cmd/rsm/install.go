package main

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/yakovlev-alex/reforger-server-manager/internal/config"
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/steam"
)

var flagInstallExperimental bool

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install or verify the Arma Reforger server files via steamcmd",
	Long: `Runs steamcmd to install or verify the Arma Reforger dedicated server files.
If the files are already installed, steamcmd will verify and update them.

Use --experimental to install the experimental beta branch instead of stable.
This overrides the branch stored in the instance for this run only.`,
	RunE: runInstall,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Schedule a steamcmd update on next restart, or update immediately if stopped",
	RunE:  runUpdate,
}

func init() {
	installCmd.Flags().BoolVar(&flagInstallExperimental, "experimental", false, "Install the experimental (beta) branch for this run")
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(updateCmd)
}

func runInstall(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	// --experimental flag on the command overrides the stored instance setting
	if flagInstallExperimental {
		inst.Experimental = true
	}

	cfg, err := config.LoadGlobal()
	if err != nil {
		return err
	}

	steamcmdPath, err := requireSteamCMD(cfg)
	if err != nil {
		return err
	}

	return runInstallForInstance(inst, steamcmdPath)
}

func runInstallForInstance(inst *instance.Instance, steamcmdPath string) error {
	branch := "stable"
	if inst.Experimental {
		branch = "experimental"
	}
	fmt.Printf("Installing server to: %s (branch: %s)\n", inst.InstallDir, branch)
	if err := steam.Install(steamcmdPath, inst.InstallDir, inst.Experimental); err != nil {
		return err
	}
	printSuccess("Server files installed/verified in %s", inst.InstallDir)
	return nil
}

func runUpdate(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	cfg, err := config.LoadGlobal()
	if err != nil {
		return err
	}

	steamcmdPath, err := requireSteamCMD(cfg)
	if err != nil {
		return err
	}

	// If server is running, schedule update-on-restart
	if isRunning := isInstanceRunning(inst); isRunning {
		inst.UpdateOnRestart = true
		if err := inst.Save(); err != nil {
			return fmt.Errorf("saving instance: %w", err)
		}

		// Regenerate systemd unit to include the update command in ExecStartPre
		if err := regenerateUnit(inst, steamcmdPath); err != nil {
			printWarning("Could not update systemd unit: %v", err)
		}

		printSuccess("Update scheduled for next restart of instance %q.", inst.Name)
		printInfo("Run 'rsm restart -i %s' to apply now.", inst.Name)
		return nil
	}

	// Server is stopped — update immediately
	printInfo("Server is not running. Updating now...")
	if err := steam.Update(steamcmdPath, inst.InstallDir, inst.Experimental); err != nil {
		return err
	}

	// Clear update-on-restart flag since we just updated
	inst.UpdateOnRestart = false
	if err := inst.Save(); err != nil {
		return fmt.Errorf("saving instance: %w", err)
	}

	// Regenerate unit without the update pre-command
	if err := regenerateUnit(inst, steamcmdPath); err != nil {
		printWarning("Could not update systemd unit: %v", err)
	}

	printSuccess("Update complete.")
	return nil
}

// requireSteamCMD ensures steamcmd is configured, prompting the user if not.
func requireSteamCMD(cfg *config.GlobalConfig) (string, error) {
	if cfg.SteamCMDPath != "" {
		return cfg.SteamCMDPath, nil
	}

	printWarning("steamcmd is not configured. Run 'rsm init' first.")

	var path string
	if err := survey.AskOne(&survey.Input{
		Message: "Enter path to steamcmd:",
	}, &path, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	cfg.SteamCMDPath = path
	if err := config.SaveGlobal(cfg); err != nil {
		return "", fmt.Errorf("saving config: %w", err)
	}
	return path, nil
}
