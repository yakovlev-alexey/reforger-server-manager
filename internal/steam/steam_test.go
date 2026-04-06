package steam_test

import (
	"strings"
	"testing"

	"github.com/yakovlev-alex/reforger-server-manager/internal/steam"
)

func TestBuildUpdateCommand_Stable(t *testing.T) {
	cmd := steam.BuildUpdateCommand("/usr/games/steamcmd", "/home/steam/reforger", false)
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
	if strings.Contains(cmd, "-beta") {
		t.Error("stable command should not contain -beta flag")
	}
}

func TestBuildUpdateCommand_StableFormat(t *testing.T) {
	cmd := steam.BuildUpdateCommand("/path/to/steamcmd", "/data/reforger", false)
	expected := "/path/to/steamcmd +force_install_dir /data/reforger +login anonymous +app_update 1874900 validate +quit"
	if cmd != expected {
		t.Errorf("BuildUpdateCommand (stable) =\n  %q\nwant\n  %q", cmd, expected)
	}
}

func TestBuildUpdateCommand_Experimental(t *testing.T) {
	cmd := steam.BuildUpdateCommand("/usr/games/steamcmd", "/home/steam/reforger", true)
	if !strings.Contains(cmd, "-beta") {
		t.Error("experimental command should contain -beta flag")
	}
	if !strings.Contains(cmd, steam.ExperimentalBranch) {
		t.Errorf("experimental command should contain branch name %q", steam.ExperimentalBranch)
	}
}

func TestBuildUpdateCommand_ExperimentalFormat(t *testing.T) {
	cmd := steam.BuildUpdateCommand("/path/to/steamcmd", "/data/reforger", true)
	expected := "/path/to/steamcmd +force_install_dir /data/reforger +login anonymous +app_update 1874900 -beta experiment validate +quit"
	if cmd != expected {
		t.Errorf("BuildUpdateCommand (experimental) =\n  %q\nwant\n  %q", cmd, expected)
	}
}

func TestExperimentalBranchConstant(t *testing.T) {
	if steam.ExperimentalBranch != "experiment" {
		t.Errorf("ExperimentalBranch = %q, want 'experiment'", steam.ExperimentalBranch)
	}
}

func TestFindReturnsString(t *testing.T) {
	// Verify Find() doesn't panic; may return empty on systems without steamcmd
	result := steam.Find()
	_ = result
}

func TestRequireErrorMessage(t *testing.T) {
	// If steamcmd is not installed, Require() should return an error with
	// installation instructions. We can only test the error shape, not the
	// happy path, without steamcmd present.
	if steam.Find() != "" {
		t.Skip("steamcmd is present on this system")
	}
	_, err := steam.Require()
	if err == nil {
		t.Fatal("expected error when steamcmd not found")
	}
	msg := err.Error()
	for _, want := range []string{"steamcmd", "apt", "steamcmd_linux.tar.gz"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message should mention %q, got: %s", want, msg)
		}
	}
}
