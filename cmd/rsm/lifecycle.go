package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yakovlev-alex/reforger-server-manager/internal/config"
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/systemd"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server",
	RunE:  runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the server",
	RunE:  runStop,
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the server (runs steamcmd update first if update-on-restart is set)",
	RunE:  runRestart,
}

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable autostart on boot (installs systemd unit if needed)",
	RunE:  runEnable,
}

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable autostart (systemctl disable)",
	RunE:  runDisable,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show server status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInstanceStatus(nil, args)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(disableCmd)
	rootCmd.AddCommand(statusCmd)
}

// loadSteamcmdPath returns the configured steamcmd path, or empty string.
func loadSteamcmdPath() string {
	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.SteamCMDPath
}

func runStart(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance("")
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}
	if inst.ActiveConfig == "" {
		return fmt.Errorf("no active configuration set; run 'rsm config new' first")
	}

	// Ensure the unit file is installed before trying to start.
	// This handles the case where the user declined during rsm init or is
	// running rsm start for the first time on an existing install.
	if !systemd.IsInstalled(inst) {
		printInfo("systemd unit not found — installing (requires sudo)...")
		if err := systemd.InstallUnit(inst, loadSteamcmdPath()); err != nil {
			return fmt.Errorf("installing systemd unit: %w", err)
		}
		printSuccess("systemd unit installed: %s", inst.SystemdServiceName())
	}

	printInfo("Starting %s...", inst.SystemdServiceName())
	if err := systemd.Start(inst); err != nil {
		return err
	}
	printSuccess("Server started.")
	printInfo("View logs with: rsm logs -f")
	return nil
}

func runStop(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance("")
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	printInfo("Stopping %s...", inst.SystemdServiceName())
	if err := systemd.Stop(inst); err != nil {
		return err
	}
	printSuccess("Server stopped.")
	return nil
}

func runRestart(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance("")
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}
	if inst.ActiveConfig == "" {
		return fmt.Errorf("no active configuration set; run 'rsm config new' first")
	}

	steamcmdPath := loadSteamcmdPath()

	// Ensure unit exists before restarting.
	if !systemd.IsInstalled(inst) {
		printInfo("systemd unit not found — installing (requires sudo)...")
		if err := systemd.InstallUnit(inst, steamcmdPath); err != nil {
			return fmt.Errorf("installing systemd unit: %w", err)
		}
		printSuccess("systemd unit installed: %s", inst.SystemdServiceName())
	} else if inst.UpdateOnRestart && steamcmdPath != "" {
		// Unit exists but needs updating for the steamcmd pre-command.
		printInfo("Update-on-restart is enabled — refreshing unit...")
		if err := regenerateUnit(inst, steamcmdPath); err != nil {
			printWarning("Could not refresh systemd unit: %v", err)
		}
	}

	printInfo("Restarting %s...", inst.SystemdServiceName())
	if err := systemd.Restart(inst); err != nil {
		return err
	}
	printSuccess("Server restarted.")

	// Clear update-on-restart flag after a successful restart.
	if inst.UpdateOnRestart {
		inst.UpdateOnRestart = false
		if err := inst.Save(); err != nil {
			printWarning("Could not clear update-on-restart flag: %v", err)
		} else if steamcmdPath != "" {
			if err := regenerateUnit(inst, steamcmdPath); err != nil {
				printWarning("Could not reset systemd unit: %v", err)
			}
		}
	}
	return nil
}

func runEnable(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance("")
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	// Install the unit if not already present — enabling a non-existent unit
	// would succeed silently but do nothing useful.
	if !systemd.IsInstalled(inst) {
		printInfo("systemd unit not found — installing (requires sudo)...")
		if err := systemd.InstallUnit(inst, loadSteamcmdPath()); err != nil {
			return fmt.Errorf("installing systemd unit: %w", err)
		}
		printSuccess("systemd unit installed: %s", inst.SystemdServiceName())
	}

	if err := systemd.Enable(inst); err != nil {
		return err
	}
	printSuccess("Autostart enabled for %s.", inst.SystemdServiceName())
	return nil
}

func runDisable(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance("")
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	if err := systemd.Disable(inst); err != nil {
		return err
	}
	printSuccess("Autostart disabled for %s.", inst.SystemdServiceName())
	return nil
}
