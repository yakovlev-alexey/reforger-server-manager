package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	rsmconfig "github.com/yakovlev-alex/reforger-server-manager/internal/config"
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/systemd"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage server configurations",
}

var configNewCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new named configuration via wizard",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runConfigNew,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configurations for an instance",
	RunE:  runConfigList,
}

var configEditCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Open a configuration's config.json in $EDITOR (defaults to active config)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runConfigEdit,
}

var configUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch the active configuration (restarts server if running)",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigUse,
}

var configDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a named configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigDelete,
}

func init() {
	configCmd.AddCommand(configNewCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configUseCmd)
	configCmd.AddCommand(configDeleteCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigNew(_ *cobra.Command, args []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	configName := ""
	if len(args) > 0 {
		configName = args[0]
	}
	return createConfigWizard(inst, configName)
}

// createConfigWizard runs the interactive wizard to create a new named configuration.
func createConfigWizard(inst *instance.Instance, configName string) error {
	fmt.Println(color.CyanString("=== New Server Configuration ==="))
	fmt.Println()

	// Config name
	if configName == "" {
		if err := survey.AskOne(&survey.Input{
			Message: "Configuration name:",
			Default: "default",
			Help:    "A short name like 'vanilla', 'modded', 'coop'",
		}, &configName, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}

	// Check for conflict
	existingConfigs, _ := inst.ListConfigs()
	for _, existing := range existingConfigs {
		if existing == configName {
			return fmt.Errorf("configuration %q already exists in instance %q", configName, inst.Name)
		}
	}

	// Server display name
	serverName := ""
	if err := survey.AskOne(&survey.Input{
		Message: "Server display name (shown in server browser):",
		Default: fmt.Sprintf("Reforger Server — %s", configName),
	}, &serverName, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// Bind address
	bindAddress := ""
	if err := survey.AskOne(&survey.Input{
		Message: "Bind IP address:",
		Default: "0.0.0.0",
		Help:    "Use 0.0.0.0 to listen on all interfaces",
	}, &bindAddress); err != nil {
		return err
	}

	// Public address
	publicAddress := ""
	if err := survey.AskOne(&survey.Input{
		Message: "Public IP address (visible to players, leave blank to auto-detect):",
		Default: "",
		Help:    "Your server's public/external IP. Leave blank if unsure.",
	}, &publicAddress); err != nil {
		return err
	}

	// Game port
	gamePortStr := "2001"
	if err := survey.AskOne(&survey.Input{
		Message: "Game port (UDP):",
		Default: "2001",
	}, &gamePortStr); err != nil {
		return err
	}
	gamePort := 2001
	fmt.Sscanf(gamePortStr, "%d", &gamePort)

	// Query port
	queryPortStr := "17777"
	if err := survey.AskOne(&survey.Input{
		Message: "Steam query port (UDP, for A2S queries):",
		Default: "17777",
	}, &queryPortStr); err != nil {
		return err
	}
	queryPort := 17777
	fmt.Sscanf(queryPortStr, "%d", &queryPort)

	// Max players
	maxPlayersStr := "64"
	if err := survey.AskOne(&survey.Input{
		Message: "Max players:",
		Default: "64",
	}, &maxPlayersStr); err != nil {
		return err
	}
	maxPlayers := 64
	fmt.Sscanf(maxPlayersStr, "%d", &maxPlayers)

	// Admin password
	adminPassword := ""
	if err := survey.AskOne(&survey.Password{
		Message: "Admin password:",
	}, &adminPassword, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// Game password (optional)
	gamePassword := ""
	isPrivate := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Make server private (require password to join)?",
		Default: false,
	}, &isPrivate); err != nil {
		return err
	}
	if isPrivate {
		if err := survey.AskOne(&survey.Password{
			Message: "Game password:",
		}, &gamePassword, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}

	// Scenario
	scenarioOptions := []string{
		"{ECC61978EDCC2B5A}Missions/23_Campaign.conf (Everon Game Master)",
		"{59AD59368755F41A}Missions/21_GM_Eden.conf (Eden Game Master)",
		"{90F086877C27B6F6}Missions/99_Tutorial.conf (Tutorial)",
		"Custom (enter manually)",
	}
	scenarioChoice := scenarioOptions[0]
	if err := survey.AskOne(&survey.Select{
		Message: "Scenario:",
		Options: scenarioOptions,
		Default: scenarioOptions[0],
	}, &scenarioChoice); err != nil {
		return err
	}

	scenarioID := "{ECC61978EDCC2B5A}Missions/23_Campaign.conf"
	switch {
	case strings.HasPrefix(scenarioChoice, "{ECC61978EDCC2B5A}"):
		scenarioID = "{ECC61978EDCC2B5A}Missions/23_Campaign.conf"
	case strings.HasPrefix(scenarioChoice, "{59AD59368755F41A}"):
		scenarioID = "{59AD59368755F41A}Missions/21_GM_Eden.conf"
	case strings.HasPrefix(scenarioChoice, "{90F086877C27B6F6}"):
		scenarioID = "{90F086877C27B6F6}Missions/99_Tutorial.conf"
	case strings.HasPrefix(scenarioChoice, "Custom"):
		if err := survey.AskOne(&survey.Input{
			Message: "Enter scenario ID:",
			Help:    "Format: {MODID}Missions/MissionName.conf",
		}, &scenarioID, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	}

	// Build config
	serverCfg := rsmconfig.DefaultServerConfig(
		serverName, bindAddress, publicAddress,
		gamePort, queryPort, maxPlayers,
		adminPassword, gamePassword,
	)
	serverCfg.Game.ScenarioID = scenarioID

	// Write config.json
	if err := instance.EnsureConfigDirs(inst, configName); err != nil {
		return fmt.Errorf("creating config directories: %w", err)
	}

	configPath := inst.ConfigJSONPath(configName)

	jsonData, err := json.MarshalIndent(serverCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	if err := os.WriteFile(configPath, jsonData, 0o644); err != nil {
		return fmt.Errorf("writing config.json: %w", err)
	}

	// Set as active config if this is the first one
	if inst.ActiveConfig == "" {
		inst.ActiveConfig = configName
		if err := inst.Save(); err != nil {
			return fmt.Errorf("saving instance: %w", err)
		}
		printSuccess("Configuration %q created and set as active.", configName)
	} else {
		printSuccess("Configuration %q created.", configName)
	}

	fmt.Printf("  Config file: %s\n", configPath)
	fmt.Println()

	// Offer to open in editor
	openEditor := false
	if err := survey.AskOne(&survey.Confirm{
		Message: "Open config.json in $EDITOR for further customization?",
		Default: false,
	}, &openEditor); err != nil {
		return err
	}
	if openEditor {
		return openInEditor(configPath)
	}

	return nil
}

func runConfigList(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	configs, err := inst.ListConfigs()
	if err != nil {
		return err
	}

	if len(configs) == 0 {
		printInfo("No configurations found. Run 'rsm config new' to create one.")
		return nil
	}

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
	header := fmt.Sprintf("%-*s%-*s%s", w0+gap, "NAME", w1+gap, "STATUS", "CONFIG PATH")
	fmt.Println(color.HiWhiteString(header))
	fmt.Println(strings.Repeat("-", w0+w1+gap*2+20))
	for _, r := range rows {
		nameStr := color.HiCyanString(r.name)
		statusStr := r.status
		if r.status == "active" {
			statusStr = color.GreenString(r.status)
		}
		fmt.Printf("%-*s%-*s%s\n",
			w0+gap+len(nameStr)-len(r.name), nameStr,
			w1+gap+len(statusStr)-len(r.status), statusStr,
			r.path,
		)
	}
	return nil
}

func runConfigEdit(_ *cobra.Command, args []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	// Default to active config when no name is given
	configName := inst.ActiveConfig
	if len(args) > 0 && args[0] != "" {
		configName = args[0]
	}
	if configName == "" {
		return fmt.Errorf("no active configuration set; run 'rsm config new' first")
	}

	configPath := inst.ConfigJSONPath(configName)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config %q not found (no config.json at %s)", configName, configPath)
	}

	return openInEditor(configPath)
}

func runConfigUse(_ *cobra.Command, args []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	newConfig := args[0]

	// Verify config exists
	configPath := inst.ConfigJSONPath(newConfig)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration %q not found", newConfig)
	}

	if inst.ActiveConfig == newConfig {
		printInfo("Configuration %q is already active.", newConfig)
		return nil
	}

	wasRunning := isInstanceRunning(inst)
	if wasRunning {
		printInfo("Server is running. It will be restarted to apply the new configuration.")
		confirm := false
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Switch to config %q and restart?", newConfig),
			Default: true,
		}, &confirm); err != nil {
			return err
		}
		if !confirm {
			printInfo("Aborted.")
			return nil
		}
		printInfo("Stopping server...")
		if err := systemd.Stop(inst); err != nil {
			return fmt.Errorf("stopping server: %w", err)
		}
	}

	oldConfig := inst.ActiveConfig
	inst.ActiveConfig = newConfig
	if err := inst.Save(); err != nil {
		return fmt.Errorf("saving instance: %w", err)
	}

	// Regenerate systemd unit with new config/profile paths
	globalCfg, _ := rsmconfig.LoadGlobal()
	steamcmdPath := ""
	if globalCfg != nil {
		steamcmdPath = globalCfg.SteamCMDPath
	}
	if err := regenerateUnit(inst, steamcmdPath); err != nil {
		printWarning("Could not update systemd unit: %v", err)
		printInfo("Run 'rsm instance new --instance %s' to reinstall the unit manually.", inst.Name)
	}

	printSuccess("Active configuration changed: %s → %s", oldConfig, newConfig)

	if wasRunning {
		printInfo("Starting server with new configuration...")
		if err := systemd.Start(inst); err != nil {
			return fmt.Errorf("starting server: %w", err)
		}
		printSuccess("Server restarted with configuration %q.", newConfig)
	}

	return nil
}

func runConfigDelete(_ *cobra.Command, args []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	configName := args[0]

	if inst.ActiveConfig == configName {
		return fmt.Errorf("cannot delete the active configuration %q; switch to another config first", configName)
	}

	configDir := inst.ConfigDir(configName)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("configuration %q not found", configName)
	}

	confirm := false
	if err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Delete configuration %q and its profile directory?", configName),
		Default: false,
	}, &confirm); err != nil {
		return err
	}
	if !confirm {
		printInfo("Aborted.")
		return nil
	}

	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("deleting config dir: %w", err)
	}
	printSuccess("Configuration %q deleted.", configName)
	return nil
}

// openInEditor opens a file in the user's $EDITOR (falling back to vi/nano).
func openInEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try common editors in order
		for _, e := range []string{"nano", "vi", "vim"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found; set $EDITOR environment variable")
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
