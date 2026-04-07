package steam

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// StableAppID is the Steam App ID for the stable Arma Reforger dedicated server.
const StableAppID = "1874900"

// ExperimentalAppID is the Steam App ID for the experimental Arma Reforger dedicated server.
const ExperimentalAppID = "1890870"

// commonPaths lists typical steamcmd locations on Linux.
var commonPaths = []string{
	"/usr/games/steamcmd",
	"/usr/bin/steamcmd",
	"/usr/local/bin/steamcmd",
}

// Find searches PATH and common install locations for a working steamcmd binary.
// Returns the absolute path if found, or an empty string if not found.
func Find() string {
	// PATH first
	if path, err := exec.LookPath("steamcmd"); err == nil {
		return path
	}

	// Home-directory manual installs
	candidates := commonPaths
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "Steam", "steamcmd.sh"),
			filepath.Join(home, "steamcmd", "steamcmd.sh"),
			filepath.Join(home, ".steam", "steamcmd", "steamcmd.sh"),
		)
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// Require returns the steamcmd path or an error with installation instructions.
func Require() (string, error) {
	path := Find()
	if path != "" {
		return path, nil
	}

	return "", fmt.Errorf(`steamcmd not found on this system.

Install it first:

  Ubuntu:
    sudo add-apt-repository multiverse
    sudo apt update && sudo apt install steamcmd

  Debian:
    sudo apt update; sudo apt install software-properties-common; sudo apt-add-repository non-free; sudo dpkg --add-architecture i386; sudo apt update
    sudo apt install steamcmd

  For other distributions see official documentation: https://developer.valvesoftware.com/wiki/SteamCMD#Package_From_Repositories

Then re-run this command.`)
}

// Install runs steamcmd to install or update the Reforger dedicated server.
// Set experimental=true to use the experimental app (AppID 1890870) instead of stable (1874900).
func Install(steamcmdPath, installDir string, experimental bool) error {
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("creating install dir %s: %w", installDir, err)
	}

	appID := StableAppID
	branch := "stable"
	if experimental {
		appID = ExperimentalAppID
		branch = "experimental"
	}

	args := []string{
		"+force_install_dir", installDir,
		"+login", "anonymous",
		"+app_update", appID,
		"validate", "+quit",
	}

	cmd := exec.Command(steamcmdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Printf("Running steamcmd — installing Arma Reforger Server (AppID %s, branch: %s)...\n", appID, branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("steamcmd install failed: %w", err)
	}
	return nil
}

// Update runs steamcmd to update the server, respecting the experimental flag.
func Update(steamcmdPath, installDir string, experimental bool) error {
	return Install(steamcmdPath, installDir, experimental)
}

// BuildUpdateCommand returns the shell command string used in ExecStartPre.
func BuildUpdateCommand(steamcmdPath, installDir string, experimental bool) string {
	appID := StableAppID
	if experimental {
		appID = ExperimentalAppID
	}
	return fmt.Sprintf(
		"%s +force_install_dir %s +login anonymous +app_update %s validate +quit",
		steamcmdPath, installDir, appID,
	)
}

// DetectSteamCMD is an alias for Find, kept for backward compatibility with tests.
func DetectSteamCMD() string { return Find() }
