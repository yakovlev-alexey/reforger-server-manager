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
	Short: "Enable autostart (systemctl enable)",
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

func runStart(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
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

	printInfo("Starting %s...", inst.SystemdServiceName())
	if err := systemd.Start(inst); err != nil {
		return err
	}
	printSuccess("Server started.")
	printInfo("View logs with: rsm logs -i %s -f", inst.Name)
	return nil
}

func runStop(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
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
	resolved, err := instance.ResolveInstance(flagInstance)
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

	cfg, _ := config.LoadGlobal()
	steamcmdPath := ""
	if cfg != nil {
		steamcmdPath = cfg.SteamCMDPath
	}

	// If update-on-restart is set, update the unit to include ExecStartPre
	if inst.UpdateOnRestart && steamcmdPath != "" {
		printInfo("Update-on-restart is enabled — steamcmd will run before starting.")
		if err := regenerateUnit(inst, steamcmdPath); err != nil {
			printWarning("Could not refresh systemd unit: %v", err)
		}
	}

	printInfo("Restarting %s...", inst.SystemdServiceName())
	if err := systemd.Restart(inst); err != nil {
		return err
	}
	printSuccess("Server restarted.")

	// After successful restart, clear update-on-restart flag
	if inst.UpdateOnRestart {
		inst.UpdateOnRestart = false
		if err := inst.Save(); err != nil {
			printWarning("Could not clear update-on-restart flag: %v", err)
		} else {
			// Regenerate unit without the steamcmd pre-command
			if err := regenerateUnit(inst, steamcmdPath); err != nil {
				printWarning("Could not reset systemd unit: %v", err)
			}
		}
	}
	return nil
}

func runEnable(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	if err := systemd.Enable(inst); err != nil {
		return err
	}
	printSuccess("Autostart enabled for %s.", inst.SystemdServiceName())
	return nil
}

func runDisable(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
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
