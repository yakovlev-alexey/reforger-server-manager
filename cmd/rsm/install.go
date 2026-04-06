package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/steam"
)

var flagInstallExperimental bool

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install or verify the Arma Reforger server files via steamcmd",
	Long: `Runs steamcmd to install or verify the Arma Reforger dedicated server files.
If the files are already installed, steamcmd will verify and update them.

steamcmd must be installed and available on PATH or in a standard location.

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
	resolved, err := instance.ResolveInstance("")
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	if flagInstallExperimental {
		inst.Experimental = true
	}

	steamcmdPath, err := steam.Require()
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
	resolved, err := instance.ResolveInstance("")
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	steamcmdPath, err := steam.Require()
	if err != nil {
		return err
	}

	// If server is running, schedule update-on-restart
	if isInstanceRunning(inst) {
		inst.UpdateOnRestart = true
		if err := inst.Save(); err != nil {
			return fmt.Errorf("saving instance: %w", err)
		}
		if err := regenerateUnit(inst, steamcmdPath); err != nil {
			printWarning("Could not update systemd unit: %v", err)
		}
		printSuccess("Update scheduled for next restart.")
		printInfo("Run 'rsm restart' to apply now.")
		return nil
	}

	// Server is stopped — update immediately
	printInfo("Server is not running. Updating now...")
	if err := steam.Update(steamcmdPath, inst.InstallDir, inst.Experimental); err != nil {
		return err
	}

	inst.UpdateOnRestart = false
	if err := inst.Save(); err != nil {
		return fmt.Errorf("saving instance: %w", err)
	}
	if err := regenerateUnit(inst, steamcmdPath); err != nil {
		printWarning("Could not update systemd unit: %v", err)
	}

	printSuccess("Update complete.")
	return nil
}

// findSteamCMD returns the steamcmd path if found, or empty string.
// Used by commands that benefit from steamcmd but don't require it (e.g. unit generation).
func findSteamCMD() string {
	return steam.Find()
}
