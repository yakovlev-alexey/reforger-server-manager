package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/systemd"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3"
var version = "dev"

var (
	flagInstance string
)

var rootCmd = &cobra.Command{
	Use:   "rsm",
	Short: "Reforger Server Manager",
	Long: `rsm manages Arma Reforger dedicated server instances.

Each instance is a separate server installation that can have multiple
named configurations (config.json + profile directory pairs).`,
	// When called with no subcommand, print a guided getting-started message
	// instead of the bare usage block.
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

func runRoot(cmd *cobra.Command, _ []string) error {
	instances, _ := instance.List()

	fmt.Println(color.HiCyanString("rsm") + " — Reforger Server Manager " + color.HiBlackString("("+version+")"))
	fmt.Println()

	if len(instances) == 0 {
		// First-run experience
		fmt.Println(color.HiWhiteString("Getting started"))
		fmt.Println()
		fmt.Println("  No server instances found. Create your first one:")
		fmt.Println()
		fmt.Println("  " + color.HiCyanString("rsm instance new") + "   — guided setup wizard")
		fmt.Println()
		fmt.Println(color.HiBlackString("  The wizard will walk you through:"))
		fmt.Println(color.HiBlackString("    1. Naming the instance and choosing an install directory"))
		fmt.Println(color.HiBlackString("    2. Downloading the server via steamcmd"))
		fmt.Println(color.HiBlackString("    3. Generating a server configuration"))
		fmt.Println(color.HiBlackString("    4. Setting up autostart and launching"))
	} else {
		// Returning user — show instance table and common commands
		fmt.Println(color.HiWhiteString("Instances"))
		fmt.Println()
		printInstanceSummary(instances)
		fmt.Println()
		fmt.Println(color.HiWhiteString("Common commands"))
		fmt.Println()
		fmt.Println("  " + color.HiCyanString("rsm start") + "                     — start the server")
		fmt.Println("  " + color.HiCyanString("rsm stop") + "                      — stop the server")
		fmt.Println("  " + color.HiCyanString("rsm restart") + "                   — restart the server")
		fmt.Println("  " + color.HiCyanString("rsm logs -f") + "                   — follow live logs")
		fmt.Println("  " + color.HiCyanString("rsm status") + "                    — show status")
		fmt.Println("  " + color.HiCyanString("rsm config use <name>") + "         — switch active configuration")
		fmt.Println("  " + color.HiCyanString("rsm instance new") + "              — create another instance")
		fmt.Println()
		fmt.Println(color.HiBlackString("  Run 'rsm <command> --help' for details on any command."))
	}
	fmt.Println()
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagInstance, "instance", "i", "", "Instance name (optional when only one exists)")
	rootCmd.AddCommand(versionCmd)
}

// printSuccess prints a green success message.
func printSuccess(format string, args ...interface{}) {
	fmt.Println(color.GreenString("✓ "+format, args...))
}

// printInfo prints a cyan info message.
func printInfo(format string, args ...interface{}) {
	fmt.Println(color.CyanString("→ "+format, args...))
}

// printWarning prints a yellow warning message.
func printWarning(format string, args ...interface{}) {
	fmt.Println(color.YellowString("! "+format, args...))
}

// printError prints a red error message.
func printError(format string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, color.RedString("✗ "+format, args...))
}

// fatal prints an error and exits.
func fatal(format string, args ...interface{}) {
	printError(format, args...)
	os.Exit(1)
}

// printNextStep prints a "what to do next" hint after a user declines a step.
func printNextStep(message, command string) {
	fmt.Println(color.HiWhiteString(message))
	fmt.Println("  " + color.HiCyanString(command))
	fmt.Println()
}

// printInstanceSummary renders a compact instance table for the root command output.
func printInstanceSummary(names []string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "  "+color.HiWhiteString("NAME\tCONFIG\tSTATUS\tAUTOSTART"))
	fmt.Fprintln(w, "  "+strings.Repeat("-", 60))
	for _, name := range names {
		inst, err := instance.Load(name)
		if err != nil {
			fmt.Fprintf(w, "  %s\t?\terror\t?\n", name)
			continue
		}
		status := color.RedString("stopped")
		if systemd.IsActive(inst) {
			status = color.GreenString("running")
		}
		autostart := color.RedString("off")
		if systemd.IsEnabled(inst) {
			autostart = color.GreenString("on")
		}
		cfg := inst.ActiveConfig
		if cfg == "" {
			cfg = color.YellowString("(no config)")
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
			color.HiCyanString(name), cfg, status, autostart)
	}
	w.Flush()
}
