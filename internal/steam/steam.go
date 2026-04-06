package steam

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const reforgerAppID = "1874900"

// ExperimentalBranch is the Steam beta branch name for the experimental server build.
const ExperimentalBranch = "experiment"

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
// Set experimental=true to use the "experiment" beta branch.
func Install(steamcmdPath, installDir string, experimental bool) error {
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("creating install dir %s: %w", installDir, err)
	}

	args := []string{
		"+force_install_dir", installDir,
		"+login", "anonymous",
		"+app_update", reforgerAppID,
	}
	if experimental {
		args = append(args, "-beta", ExperimentalBranch)
	}
	args = append(args, "validate", "+quit")

	cmd := exec.Command(steamcmdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	branch := "stable"
	if experimental {
		branch = "experimental"
	}
	fmt.Printf("Running steamcmd to install Arma Reforger Server (AppID %s, branch: %s)...\n", reforgerAppID, branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("steamcmd install failed: %w", err)
	}
	return nil
}

// Update runs steamcmd to update the server, respecting the experimental flag.
func Update(steamcmdPath, installDir string, experimental bool) error {
	branch := "stable"
	if experimental {
		branch = "experimental"
	}
	fmt.Printf("Running steamcmd to update Arma Reforger Server (AppID %s, branch: %s)...\n", reforgerAppID, branch)
	return Install(steamcmdPath, installDir, experimental)
}

// BuildUpdateCommand returns the shell command string used in ExecStartPre.
func BuildUpdateCommand(steamcmdPath, installDir string, experimental bool) string {
	betaArgs := ""
	if experimental {
		betaArgs = " -beta " + ExperimentalBranch
	}
	return fmt.Sprintf(
		"%s +force_install_dir %s +login anonymous +app_update %s%s validate +quit",
		steamcmdPath, installDir, reforgerAppID, betaArgs,
	)
}
