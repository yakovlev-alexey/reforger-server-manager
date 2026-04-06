package steam_test

import (
	"strings"
	"testing"

	"github.com/yakovlev-alex/reforger-server-manager/internal/steam"
)

func TestBuildUpdateCommand(t *testing.T) {
	cmd := steam.BuildUpdateCommand("/usr/games/steamcmd", "/home/steam/reforger")
	if !strings.Contains(cmd, "/usr/games/steamcmd") {
		t.Error("command should contain steamcmd path")
	}
	if !strings.Contains(cmd, "/home/steam/reforger") {
		t.Error("command should contain install dir")
	}
	if !strings.Contains(cmd, "1874900") {
		t.Error("command should contain Reforger app ID")
	}
	if !strings.Contains(cmd, "validate") {
		t.Error("command should include validate flag")
	}
	if !strings.Contains(cmd, "+login anonymous") {
		t.Error("command should use anonymous login")
	}
}

func TestDetectSteamCMDReturnsString(t *testing.T) {
	// Just verify it doesn't panic and returns a string (may be empty in CI)
	result := steam.DetectSteamCMD()
	_ = result // empty on systems without steamcmd is fine
}

func TestBuildUpdateCommandFormat(t *testing.T) {
	cmd := steam.BuildUpdateCommand("/path/to/steamcmd", "/data/reforger")
	expected := "/path/to/steamcmd +force_install_dir /data/reforger +login anonymous +app_update 1874900 validate +quit"
	if cmd != expected {
		t.Errorf("BuildUpdateCommand =\n  %q\nwant\n  %q", cmd, expected)
	}
}
