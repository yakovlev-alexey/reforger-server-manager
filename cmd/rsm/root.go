package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/systemd"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3"
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "rsm",
	Short: "Reforger Server Manager",
	Long: `rsm manages Arma Reforger dedicated server instances.

Each instance is a separate server installation that can have multiple
named configurations (config.json + profile directory pairs).`,
	RunE: runRoot,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print rsm version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.RedString("Error: %v", err))
		os.Exit(1)
	}
}

func runRoot(_ *cobra.Command, _ []string) error {
	fmt.Println(color.HiCyanString("rsm") + " — Reforger Server Manager " + color.HiBlackString("("+version+")"))
	fmt.Println()

	// If CWD is inside an instance, show a focused view of that instance.
	if inst, ok := instance.LoadFromCWD(); ok {
		return printInstanceView(inst)
	}

	// Otherwise show the global registry list (or getting-started).
	names, _ := instance.List()
	if len(names) == 0 {
		return printGettingStarted()
	}
	return printRegistryView(names)
}

// printGettingStarted is shown when no instances are registered anywhere.
func printGettingStarted() error {
	fmt.Println(color.HiWhiteString("Getting started"))
	fmt.Println()
	fmt.Println("  No server instances found. Create your first one:")
	fmt.Println()
	fmt.Println("  " + color.HiCyanString("rsm init") + "   — guided setup wizard")
	fmt.Println()
	fmt.Println(color.HiBlackString("  The wizard will walk you through:"))
	fmt.Println(color.HiBlackString("    1. Naming the instance and choosing an install directory"))
	fmt.Println(color.HiBlackString("    2. Generating a server configuration"))
	fmt.Println(color.HiBlackString("    3. Downloading the server via steamcmd"))
	fmt.Println(color.HiBlackString("    4. Setting up autostart and launching"))
	fmt.Println()
	return nil
}

// printRegistryView is shown when outside any instance directory.
func printRegistryView(names []string) error {
	fmt.Println(color.HiWhiteString("Instances"))
	fmt.Println()
	printInstanceSummary(names)
	fmt.Println()
	fmt.Println(color.HiWhiteString("Common commands"))
	fmt.Println()
	printCommonCommands(false)
	fmt.Println()
	fmt.Println(color.HiBlackString("  Run 'rsm <command> --help' for details on any command."))
	fmt.Println()
	return nil
}

// printInstanceView is shown when the CWD is inside a known instance directory.
func printInstanceView(inst *instance.Instance) error {
	// ── Instance status row ──────────────────────────────────────────────────
	status := "stopped"
	if systemd.IsActive(inst) {
		status = "running"
	}
	autostart := "off"
	if systemd.IsEnabled(inst) {
		autostart = "on"
	}
	activeCfg := inst.ActiveConfig
	if activeCfg == "" {
		activeCfg = "(no config)"
	}

	fmt.Println(color.HiWhiteString("Instance"))
	fmt.Println()

	// Single-row instance table
	{
		namePlain, cfgPlain, statusPlain, autostartPlain :=
			inst.Name, activeCfg, status, autostart

		w0 := max(len("NAME"), len(namePlain))
		w1 := max(len("CONFIG"), len(cfgPlain))
		w2 := max(len("STATUS"), len(statusPlain))
		w3 := max(len("AUTOSTART"), len(autostartPlain))
		gap := 3

		header := fmt.Sprintf("  %-*s%-*s%-*s%-*s",
			w0+gap, "NAME", w1+gap, "CONFIG", w2+gap, "STATUS", w3, "AUTOSTART")
		fmt.Println(color.HiWhiteString(header))
		fmt.Println("  " + strings.Repeat("-", w0+w1+w2+w3+gap*3))

		statusStr := color.RedString(statusPlain)
		if status == "running" {
			statusStr = color.GreenString(statusPlain)
		}
		autostartStr := color.RedString(autostartPlain)
		if autostart == "on" {
			autostartStr = color.GreenString(autostartPlain)
		}
		nameStr := color.HiCyanString(namePlain)

		fmt.Printf("  %-*s%-*s%-*s%s\n",
			w0+gap+len(nameStr)-len(namePlain), nameStr,
			w1+gap, cfgPlain,
			w2+gap+len(statusStr)-len(statusPlain), statusStr,
			autostartStr,
		)
	}

	fmt.Println()

	// ── Configurations table ─────────────────────────────────────────────────
	configs, _ := inst.ListConfigs()
	if len(configs) == 0 {
		fmt.Println(color.HiWhiteString("Configurations"))
		fmt.Println()
		fmt.Println("  (none — run 'rsm config new' to create one)")
		fmt.Println()
	} else {
		fmt.Println(color.HiWhiteString("Configurations"))
		fmt.Println()
		printConfigsTable(inst, configs)
		fmt.Println()
	}

	// ── Common commands ──────────────────────────────────────────────────────
	fmt.Println(color.HiWhiteString("Common commands"))
	fmt.Println()
	printCommonCommands(true)
	fmt.Println()
	fmt.Println(color.HiBlackString("  Run 'rsm <command> --help' for details on any command."))
	fmt.Println()
	return nil
}

// printConfigsTable renders the configuration list for an instance.
func printConfigsTable(inst *instance.Instance, configs []string) {
	type row struct{ name, status, path string }
	rows := make([]row, 0, len(configs))
	for _, name := range configs {
		status := ""
		if name == inst.ActiveConfig {
			status = "active"
		}
		rows = append(rows, row{name, status, inst.ConfigJSONPath(name)})
	}

	w0, w1 := len("NAME"), len("STATUS")
	for _, r := range rows {
		if len(r.name) > w0 {
			w0 = len(r.name)
		}
		if len(r.status) > w1 {
			w1 = len(r.status)
		}
	}
	gap := 3
	header := fmt.Sprintf("  %-*s%-*s%s", w0+gap, "NAME", w1+gap, "STATUS", "CONFIG PATH")
	fmt.Println(color.HiWhiteString(header))
	fmt.Println("  " + strings.Repeat("-", w0+w1+gap*2+20))
	for _, r := range rows {
		nameStr := color.HiCyanString(r.name)
		statusStr := r.status
		if r.status == "active" {
			statusStr = color.GreenString(r.status)
		}
		fmt.Printf("  %-*s%-*s%s\n",
			w0+gap+len(nameStr)-len(r.name), nameStr,
			w1+gap+len(statusStr)-len(r.status), statusStr,
			r.path,
		)
	}
}

// printCommonCommands prints the quick-reference command list.
// inInstance=true adds instance-specific commands (config new, enable, etc.).
func printCommonCommands(inInstance bool) {
	type entry struct{ cmd, desc string }
	cmds := []entry{
		{"rsm start", "start the server"},
		{"rsm stop", "stop the server"},
		{"rsm restart", "restart the server"},
		{"rsm logs -f", "follow live logs"},
		{"rsm status", "show detailed status"},
	}
	if inInstance {
		cmds = append(cmds,
			entry{"rsm config new", "create a new configuration"},
			entry{"rsm config use <name>", "switch active configuration"},
			entry{"rsm config edit", "edit active config.json in $EDITOR"},
			entry{"rsm edit", "edit instance options (FPS, restarts, flags, etc.)"},
			entry{"rsm enable", "enable autostart on boot"},
			entry{"rsm update", "schedule a server update"},
		)
	} else {
		cmds = append(cmds,
			entry{"rsm config use <name>", "switch active configuration"},
			entry{"rsm init", "set up another instance"},
		)
	}

	// Compute padding from the longest command string
	maxLen := 0
	for _, e := range cmds {
		if len(e.cmd) > maxLen {
			maxLen = len(e.cmd)
		}
	}
	for _, e := range cmds {
		cmdStr := color.HiCyanString(e.cmd)
		// Compensate for ANSI bytes in cmdStr when computing padding
		pad := maxLen - len(e.cmd) + 4
		fmt.Printf("  %s%s— %s\n", cmdStr, strings.Repeat(" ", pad), e.desc)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// ── print helpers ────────────────────────────────────────────────────────────

func printSuccess(format string, args ...interface{}) {
	fmt.Println(color.GreenString("✓ "+format, args...))
}

func printInfo(format string, args ...interface{}) {
	fmt.Println(color.CyanString("→ "+format, args...))
}

func printWarning(format string, args ...interface{}) {
	fmt.Println(color.YellowString("! "+format, args...))
}

func printError(format string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, color.RedString("✗ "+format, args...))
}

func fatal(format string, args ...interface{}) {
	printError(format, args...)
	os.Exit(1)
}

func printNextStep(message, command string) {
	fmt.Println(color.HiWhiteString(message))
	fmt.Println("  " + color.HiCyanString(command))
	fmt.Println()
}

// printInstanceSummary renders the compact instance table used by printRegistryView.
func printInstanceSummary(names []string) {
	type row struct {
		name, cfg, status, autostart string
	}
	rows := make([]row, 0, len(names))
	for _, name := range names {
		inst, err := instance.Load(name)
		if err != nil {
			rows = append(rows, row{name, "?", "error", "?"})
			continue
		}
		cfg := inst.ActiveConfig
		if cfg == "" {
			cfg = "(no config)"
		}
		status := "stopped"
		if systemd.IsActive(inst) {
			status = "running"
		}
		autostart := "off"
		if systemd.IsEnabled(inst) {
			autostart = "on"
		}
		rows = append(rows, row{inst.Name, cfg, status, autostart})
	}

	w0, w1, w2, w3 := len("NAME"), len("CONFIG"), len("STATUS"), len("AUTOSTART")
	for _, r := range rows {
		if len(r.name) > w0 {
			w0 = len(r.name)
		}
		if len(r.cfg) > w1 {
			w1 = len(r.cfg)
		}
		if len(r.status) > w2 {
			w2 = len(r.status)
		}
		if len(r.autostart) > w3 {
			w3 = len(r.autostart)
		}
	}
	gap := 3

	header := fmt.Sprintf("  %-*s%-*s%-*s%-*s",
		w0+gap, "NAME", w1+gap, "CONFIG", w2+gap, "STATUS", w3, "AUTOSTART")
	fmt.Println(color.HiWhiteString(header))
	fmt.Println("  " + strings.Repeat("-", w0+w1+w2+w3+gap*3))

	for _, r := range rows {
		statusStr := color.RedString(r.status)
		if r.status == "running" {
			statusStr = color.GreenString(r.status)
		}
		autostartStr := color.RedString(r.autostart)
		if r.autostart == "on" {
			autostartStr = color.GreenString(r.autostart)
		}
		nameStr := color.HiCyanString(r.name)
		fmt.Printf("  %-*s%-*s%-*s%s\n",
			w0+gap+len(nameStr)-len(r.name), nameStr,
			w1+gap, r.cfg,
			w2+gap+len(statusStr)-len(r.status), statusStr,
			autostartStr,
		)
	}
}

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
