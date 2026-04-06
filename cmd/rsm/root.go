package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
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
