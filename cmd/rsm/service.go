package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/systemd"
)

// serviceCmd groups operations on the systemd service unit itself.
// rsm enable / rsm disable / rsm start already handle day-to-day use;
// this command exists for explicit reinstalls and status checks.
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the systemd service unit",
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Generate and install (or reinstall) the systemd unit file",
	Long: `Writes the systemd service unit to /etc/systemd/system/ and reloads
the daemon. Run this after changing launch settings (max FPS, extra flags,
active configuration) or if the unit file was deleted.

Requires sudo to write to /etc/systemd/system/.`,
	RunE: runServiceInstall,
}

var serviceEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable autostart on boot (alias for rsm enable)",
	RunE:  runEnable,
}

var serviceDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable autostart on boot (alias for rsm disable)",
	RunE:  runDisable,
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the raw systemd unit file that would be installed",
	RunE:  runServiceStatus,
}

func init() {
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceEnableCmd)
	serviceCmd.AddCommand(serviceDisableCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	rootCmd.AddCommand(serviceCmd)
}

func runServiceInstall(_ *cobra.Command, _ []string) error {
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

	steamcmdPath := findSteamCMD()

	wasInstalled := systemd.IsInstalled(inst)
	action := "Installing"
	if wasInstalled {
		action = "Reinstalling"
	}
	printInfo("%s systemd unit (requires sudo)...", action)

	if err := systemd.ReinstallUnit(inst, steamcmdPath); err != nil {
		return fmt.Errorf("installing unit: %w", err)
	}

	printSuccess("systemd unit installed: %s", inst.SystemdServiceName())

	if !wasInstalled {
		printInfo("Run 'rsm enable' to enable autostart on boot.")
		printInfo("Run 'rsm start' to start the server now.")
	}
	return nil
}

func runServiceStatus(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance("")
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	steamcmdPath := findSteamCMD()
	content, err := systemd.GenerateUnit(inst, steamcmdPath)
	if err != nil {
		return fmt.Errorf("generating unit: %w", err)
	}

	installed := systemd.IsInstalled(inst)
	enabled := systemd.IsEnabled(inst)

	if installed {
		printSuccess("Unit installed at: /etc/systemd/system/%s", inst.SystemdServiceName())
	} else {
		printWarning("Unit NOT installed in /etc/systemd/system/")
	}
	if enabled {
		printSuccess("Autostart: enabled")
	} else {
		printWarning("Autostart: disabled")
	}
	fmt.Println()
	fmt.Println("Generated unit file:")
	fmt.Println("---")
	fmt.Print(content)
	fmt.Println("---")
	return nil
}
