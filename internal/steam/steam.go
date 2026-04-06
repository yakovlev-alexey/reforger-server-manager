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

  Debian / Ubuntu:
    sudo add-apt-repository multiverse
    sudo apt update && sudo apt install steamcmd

  Other Linux (manual):
    mkdir ~/steamcmd && cd ~/steamcmd
    curl -O https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz
    tar -xzf steamcmd_linux.tar.gz
    ./steamcmd.sh +quit

Then re-run this command.`)
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
	fmt.Printf("Running steamcmd — installing Arma Reforger Server (AppID %s, branch: %s)...\n", reforgerAppID, branch)
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
	fmt.Printf("Running steamcmd — updating Arma Reforger Server (AppID %s, branch: %s)...\n", reforgerAppID, branch)
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

// DetectSteamCMD is an alias for Find, kept for backward compatibility with tests.
func DetectSteamCMD() string { return Find() }
