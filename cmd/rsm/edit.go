package main

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/systemd"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit instance options (flags, periodic restart, max FPS, etc.)",
	Long: `Interactive wizard to change instance-level settings.

Editable options:
  - Max FPS (server tick rate)
  - Extra launch flags
  - Periodic automatic restarts
  - Update on restart flag
  - Experimental branch
  - Systemd service user

After saving, the systemd unit (and restart timer if applicable) are
reinstalled automatically so the changes take effect on the next start.`,
	RunE: runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func runEdit(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance("")
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	fmt.Println(color.CyanString("=== Edit Instance: %s ===", inst.Name))
	fmt.Println()

	// Which fields to change — multi-select menu
	allOptions := []string{
		"Max FPS",
		"Extra launch flags",
		"Periodic restart",
		"Update on restart",
		"Experimental branch",
		"Systemd service user",
	}
	selectedOptions := []string{}
	if err := survey.AskOne(&survey.MultiSelect{
		Message: "Select options to edit:",
		Options: allOptions,
		Help:    "Space to select, Enter to confirm.",
	}, &selectedOptions); err != nil {
		return err
	}
	if len(selectedOptions) == 0 {
		printInfo("Nothing selected — no changes made.")
		return nil
	}

	changed := false

	for _, opt := range selectedOptions {
		fmt.Println()
		switch opt {
		case "Max FPS":
			if err := editMaxFPS(inst); err != nil {
				return err
			}
			changed = true

		case "Extra launch flags":
			if err := editExtraFlags(inst); err != nil {
				return err
			}
			changed = true

		case "Periodic restart":
			if err := editPeriodicRestart(inst); err != nil {
				return err
			}
			changed = true

		case "Update on restart":
			if err := editUpdateOnRestart(inst); err != nil {
				return err
			}
			changed = true

		case "Experimental branch":
			if err := editExperimental(inst); err != nil {
				return err
			}
			changed = true

		case "Systemd service user":
			if err := editSystemdUser(inst); err != nil {
				return err
			}
			changed = true
		}
	}

	if !changed {
		return nil
	}

	// Persist
	fmt.Println()
	if err := inst.Save(); err != nil {
		return fmt.Errorf("saving instance: %w", err)
	}
	printSuccess("Instance %q updated.", inst.Name)

	// Regenerate unit + sync timer if the unit is already installed
	if systemd.IsInstalled(inst) {
		printInfo("Reinstalling systemd unit to apply changes...")
		if err := regenerateUnit(inst, findSteamCMD()); err != nil {
			printWarning("Could not reinstall systemd unit: %v", err)
			printNextStep("Reinstall it manually with:", "rsm enable")
		} else {
			printSuccess("systemd unit reinstalled.")
		}
	} else {
		printInfo("systemd unit is not installed yet — run 'rsm enable' to install it.")
	}

	if isInstanceRunning(inst) {
		printNextStep("Restart the server to apply changes:", "rsm restart")
	}
	return nil
}

func editMaxFPS(inst *instance.Instance) error {
	current := fmt.Sprintf("%d", inst.MaxFPS)
	result := current
	if err := survey.AskOne(&survey.Input{
		Message: "Max FPS (server tick rate):",
		Default: current,
		Help:    "Server tick rate passed as -maxFPS flag.",
	}, &result); err != nil {
		return err
	}
	val := inst.MaxFPS
	fmt.Sscanf(result, "%d", &val)
	if val != inst.MaxFPS {
		printInfo("Max FPS: %d → %d", inst.MaxFPS, val)
		inst.MaxFPS = val
	}
	return nil
}

func editExtraFlags(inst *instance.Instance) error {
	knownOptions := []string{"-loadSessionSave", "-backendLocalStorage", "-logStats 3000"}

	// Pre-select flags that are already set and also appear in knownOptions.
	// Any custom flags (not in knownOptions) are shown separately.
	knownSet := make(map[string]bool, len(knownOptions))
	for _, o := range knownOptions {
		knownSet[o] = true
	}

	currentKnown := []string{}
	customFlags := []string{}
	for _, f := range inst.ExtraFlags {
		if knownSet[f] {
			currentKnown = append(currentKnown, f)
		} else {
			customFlags = append(customFlags, f)
		}
	}

	selected := currentKnown
	if err := survey.AskOne(&survey.MultiSelect{
		Message: "Extra launch flags:",
		Options: knownOptions,
		Default: currentKnown,
		Help:    "These flags are appended to the server launch command.",
	}, &selected); err != nil {
		return err
	}

	// Preserve any custom flags that aren't in the known list
	newFlags := append(selected, customFlags...)
	printInfo("Extra flags: [%s] → [%s]",
		strings.Join(inst.ExtraFlags, " "),
		strings.Join(newFlags, " "))
	inst.ExtraFlags = newFlags
	return nil
}

func editPeriodicRestart(inst *instance.Instance) error {
	current := inst.PeriodicRestart
	currentDisplay := "disabled"
	if current != "" {
		currentDisplay = current
	}
	printInfo("Current periodic restart: %s", currentDisplay)

	enabled := current != ""
	if err := survey.AskOne(&survey.Confirm{
		Message: "Enable periodic automatic restarts?",
		Default: enabled,
		Help:    "Installs a systemd timer that restarts the server on a fixed schedule.",
	}, &enabled); err != nil {
		return err
	}

	if !enabled {
		if inst.PeriodicRestart != "" {
			printInfo("Periodic restart: %s → disabled", inst.PeriodicRestart)
		}
		inst.PeriodicRestart = ""
		return nil
	}

	// Pick or enter an interval
	intervalOptions := []string{
		"6h  — every 6 hours",
		"12h — every 12 hours",
		"1d  — once a day",
		"2d  — every 2 days",
		"Custom (enter manually)",
	}
	defaultChoice := intervalOptions[1] // 12h
	for i, opt := range intervalOptions {
		if strings.HasPrefix(opt, current) {
			defaultChoice = intervalOptions[i]
			break
		}
	}

	choice := defaultChoice
	if err := survey.AskOne(&survey.Select{
		Message: "Restart interval:",
		Options: intervalOptions,
		Default: defaultChoice,
	}, &choice); err != nil {
		return err
	}

	var newInterval string
	switch {
	case strings.HasPrefix(choice, "6h"):
		newInterval = "6h"
	case strings.HasPrefix(choice, "12h"):
		newInterval = "12h"
	case strings.HasPrefix(choice, "1d"):
		newInterval = "1d"
	case strings.HasPrefix(choice, "2d"):
		newInterval = "2d"
	default:
		if err := survey.AskOne(&survey.Input{
			Message: "Restart interval (systemd time span, e.g. 6h, 12h, 1d):",
			Default: current,
			Help:    "Use systemd time span format: Nh for hours, Nd for days.",
		}, &newInterval, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
		newInterval = strings.TrimSpace(newInterval)
	}

	if newInterval != current {
		printInfo("Periodic restart: %s → %s", currentDisplay, newInterval)
	}
	inst.PeriodicRestart = newInterval
	return nil
}

func editUpdateOnRestart(inst *instance.Instance) error {
	result := inst.UpdateOnRestart
	if err := survey.AskOne(&survey.Confirm{
		Message: "Run steamcmd update before each restart?",
		Default: inst.UpdateOnRestart,
		Help:    "When enabled, steamcmd will update the server files before ExecStart on each restart.",
	}, &result); err != nil {
		return err
	}
	if result != inst.UpdateOnRestart {
		printInfo("Update on restart: %v → %v", inst.UpdateOnRestart, result)
		inst.UpdateOnRestart = result
	}
	return nil
}

func editExperimental(inst *instance.Instance) error {
	result := inst.Experimental
	if err := survey.AskOne(&survey.Confirm{
		Message: "Use experimental (beta) server branch?",
		Default: inst.Experimental,
		Help:    "Installs the 'experiment' Steam beta branch. May be unstable.",
	}, &result); err != nil {
		return err
	}
	if result != inst.Experimental {
		printInfo("Experimental: %v → %v", inst.Experimental, result)
		inst.Experimental = result
		if result {
			printWarning("Experimental branch selected — run 'rsm update' to switch branches.")
		}
	}
	return nil
}

func editSystemdUser(inst *instance.Instance) error {
	result := inst.SystemdUser
	if err := survey.AskOne(&survey.Input{
		Message: "System user to run the server as:",
		Default: inst.SystemdUser,
		Help:    "The OS user account the server process will run under.",
	}, &result, survey.WithValidator(survey.Required)); err != nil {
		return err
	}
	result = strings.TrimSpace(result)
	if result != inst.SystemdUser {
		printInfo("Systemd user: %s → %s", inst.SystemdUser, result)
		inst.SystemdUser = result
	}
	return nil
}
