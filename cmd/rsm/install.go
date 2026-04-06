package main

import (
	"fmt"

	"github.com/fatih/color"
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

// requireSteamCMD returns the steamcmd path, detecting it automatically if not
// already stored. Saves the detected path for future runs. Fails with a clear
// install instruction if steamcmd is not found anywhere on the system.
func requireSteamCMD(cfg *config.GlobalConfig) (string, error) {
	// Already known — use it.
	if cfg.SteamCMDPath != "" {
		return cfg.SteamCMDPath, nil
	}

	// Try to detect automatically.
	printInfo("Looking for steamcmd...")
	detected := steam.DetectSteamCMD()
	if detected != "" {
		printSuccess("Found steamcmd at: %s", detected)
		cfg.SteamCMDPath = detected
		if err := config.SaveGlobal(cfg); err != nil {
			return "", fmt.Errorf("saving config: %w", err)
		}
		return detected, nil
	}

	// Not found — print install instructions and fail.
	fmt.Println()
	fmt.Println(color.RedString("✗ steamcmd not found."))
	fmt.Println()
	fmt.Println("  Install it first, then re-run this command:")
	fmt.Println()
	fmt.Println("  " + color.HiWhiteString("Debian / Ubuntu:"))
	fmt.Println("    sudo add-apt-repository multiverse")
	fmt.Println("    sudo apt update && sudo apt install steamcmd")
	fmt.Println()
	fmt.Println("  " + color.HiWhiteString("Other Linux (manual):"))
	fmt.Println("    mkdir ~/steamcmd && cd ~/steamcmd")
	fmt.Println("    curl -O https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz")
	fmt.Println("    tar -xzf steamcmd_linux.tar.gz")
	fmt.Println("    ./steamcmd.sh +quit")
	fmt.Println()
	return "", fmt.Errorf("steamcmd is required but was not found; install it and try again")
}
