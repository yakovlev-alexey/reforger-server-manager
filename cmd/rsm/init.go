package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yakovlev-alex/reforger-server-manager/internal/config"
	"github.com/yakovlev-alex/reforger-server-manager/internal/steam"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "First-time setup: detect steamcmd and configure rsm",
	Long: `Initializes rsm by detecting or configuring steamcmd.
Run this once before creating your first instance.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println(color.CyanString("=== rsm — Reforger Server Manager Setup ==="))
	fmt.Println()

	// Load existing config (may be empty)
	cfg, err := config.LoadGlobal()
	if err != nil {
		return err
	}

	// --- steamcmd detection ---
	steamcmdPath := cfg.SteamCMDPath

	if steamcmdPath == "" {
		printInfo("Searching for steamcmd...")
		steamcmdPath = steam.DetectSteamCMD()
	}

	if steamcmdPath != "" {
		printSuccess("Found steamcmd at: %s", steamcmdPath)
		useFound := true
		confirm := &survey.Confirm{
			Message: fmt.Sprintf("Use this steamcmd path? (%s)", steamcmdPath),
			Default: true,
		}
		if err := survey.AskOne(confirm, &useFound); err != nil {
			return err
		}
		if !useFound {
			steamcmdPath = ""
		}
	}

	if steamcmdPath == "" {
		printWarning("steamcmd not found automatically.")
		fmt.Println("You can install it via:")
		fmt.Println("  Debian/Ubuntu: sudo apt install steamcmd")
		fmt.Println("  Or download from: https://developer.valvesoftware.com/wiki/SteamCMD")
		fmt.Println()

		pathPrompt := &survey.Input{
			Message: "Enter the full path to steamcmd (or press Enter to skip):",
		}
		if err := survey.AskOne(pathPrompt, &steamcmdPath); err != nil {
			return err
		}
		steamcmdPath = strings.TrimSpace(steamcmdPath)
	}

	// Validate steamcmd if provided
	if steamcmdPath != "" {
		if _, err := os.Stat(steamcmdPath); err != nil {
			printWarning("Path does not exist: %s (saving anyway)", steamcmdPath)
		} else {
			printInfo("Validating steamcmd...")
			testCmd := exec.Command(steamcmdPath, "+quit")
			testCmd.Stdout = os.Stdout
			testCmd.Stderr = os.Stderr
			if err := testCmd.Run(); err != nil {
				printWarning("steamcmd test run failed (it may still work): %v", err)
			} else {
				printSuccess("steamcmd is working.")
			}
		}
		cfg.SteamCMDPath = steamcmdPath
	}

	// Save global config
	if err := config.SaveGlobal(cfg); err != nil {
		return fmt.Errorf("saving global config: %w", err)
	}

	cfgPath, _ := config.GlobalConfigPath()
	printSuccess("Configuration saved to %s", cfgPath)
	fmt.Println()

	// Offer to create first instance
	createNow := false
	createPrompt := &survey.Confirm{
		Message: "Would you like to create your first server instance now?",
		Default: true,
	}
	if err := survey.AskOne(createPrompt, &createNow); err != nil {
		return err
	}
	if createNow {
		fmt.Println()
		return runInstanceNew(cmd, args)
	}

	fmt.Println()
	printInfo("Next steps:")
	fmt.Println("  rsm instance new <name>   — create a server instance")
	fmt.Println("  rsm instance list          — list all instances")
	return nil
}
