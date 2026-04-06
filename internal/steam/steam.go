package steam

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const reforgerAppID = "1874900"

// CommonPaths lists typical steamcmd locations on Linux.
var CommonPaths = []string{
	"/usr/games/steamcmd",
	"/usr/bin/steamcmd",
	"/usr/local/bin/steamcmd",
}

// DetectSteamCMD searches PATH and common locations for steamcmd.
// Returns the path if found, or empty string if not found.
func DetectSteamCMD() string {
	// Try PATH first
	if path, err := exec.LookPath("steamcmd"); err == nil {
		return path
	}

	// Try home directory (common manual install)
	if home, err := os.UserHomeDir(); err == nil {
		candidates := []string{
			filepath.Join(home, "Steam", "steamcmd.sh"),
			filepath.Join(home, "steamcmd", "steamcmd.sh"),
			filepath.Join(home, ".steam", "steamcmd", "steamcmd.sh"),
		}
		CommonPaths = append(CommonPaths, candidates...)
	}

	for _, p := range CommonPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// Validate runs steamcmd +quit to verify the binary works.
func Validate(steamcmdPath string) error {
	cmd := exec.Command(steamcmdPath, "+quit")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("steamcmd validation failed: %w", err)
	}
	return nil
}

// Install runs steamcmd to install or update the Reforger dedicated server.
func Install(steamcmdPath, installDir string) error {
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("creating install dir %s: %w", installDir, err)
	}

	cmd := exec.Command(
		steamcmdPath,
		"+force_install_dir", installDir,
		"+login", "anonymous",
		"+app_update", reforgerAppID, "validate",
		"+quit",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Printf("Running steamcmd to install Arma Reforger Server (AppID %s)...\n", reforgerAppID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("steamcmd install failed: %w", err)
	}
	return nil
}

// Update is an alias for Install (steamcmd +app_update with validate handles updates).
func Update(steamcmdPath, installDir string) error {
	fmt.Printf("Running steamcmd to update Arma Reforger Server (AppID %s)...\n", reforgerAppID)
	return Install(steamcmdPath, installDir)
}

// BuildUpdateCommand returns the shell command string used in ExecStartPre.
func BuildUpdateCommand(steamcmdPath, installDir string) string {
	return fmt.Sprintf(
		"%s +force_install_dir %s +login anonymous +app_update %s validate +quit",
		steamcmdPath, installDir, reforgerAppID,
	)
}
