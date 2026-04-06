package systemd_test

import (
	"os"
	"strings"
	"testing"

	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/systemd"
)

func setupInstance(t *testing.T) *instance.Instance {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	installDir := t.TempDir()
	inst := &instance.Instance{
		Name:            "testserver",
		InstallDir:      installDir,
		ActiveConfig:    "vanilla",
		UpdateOnRestart: false,
		MaxFPS:          60,
		ExtraFlags:      []string{"-loadSessionSave", "-backendLocalStorage"},
		SystemdUser:     "steam",
	}
	if err := inst.Save(); err != nil {
		t.Fatalf("Save instance: %v", err)
	}

	// Create config dirs so paths resolve
	if err := instance.EnsureConfigDirs(inst, "vanilla"); err != nil {
		t.Fatalf("EnsureConfigDirs: %v", err)
	}
	// Write a dummy config.json
	configPath := inst.ConfigJSONPath("vanilla")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}
	return inst
}

func TestGenerateUnit_Basic(t *testing.T) {
	inst := setupInstance(t)

	content, err := systemd.GenerateUnit(inst, "")
	if err != nil {
		t.Fatalf("GenerateUnit: %v", err)
	}

	if !strings.Contains(content, "rsm-testserver") {
		t.Error("unit should contain service name")
	}
	if !strings.Contains(content, "ArmaReforgerServer") {
		t.Error("unit should contain server binary")
	}
	if !strings.Contains(content, "-maxFPS 60") {
		t.Error("unit should contain maxFPS flag")
	}
	if !strings.Contains(content, "-loadSessionSave") {
		t.Error("unit should contain extra flag")
	}
	if !strings.Contains(content, "User=steam") {
		t.Error("unit should contain User=steam")
	}
	if !strings.Contains(content, "[Unit]") {
		t.Error("unit should contain [Unit] section")
	}
	if !strings.Contains(content, "[Service]") {
		t.Error("unit should contain [Service] section")
	}
	if !strings.Contains(content, "[Install]") {
		t.Error("unit should contain [Install] section")
	}
}

func TestGenerateUnit_NoUpdateOnRestart(t *testing.T) {
	inst := setupInstance(t)
	inst.UpdateOnRestart = false

	content, err := systemd.GenerateUnit(inst, "/usr/games/steamcmd")
	if err != nil {
		t.Fatalf("GenerateUnit: %v", err)
	}

	// When update-on-restart is false, ExecStartPre should be /bin/true
	if !strings.Contains(content, "ExecStartPre=/bin/true") {
		t.Error("expected ExecStartPre=/bin/true when UpdateOnRestart=false")
	}
}

func TestGenerateUnit_WithUpdateOnRestart(t *testing.T) {
	inst := setupInstance(t)
	inst.UpdateOnRestart = true

	content, err := systemd.GenerateUnit(inst, "/usr/games/steamcmd")
	if err != nil {
		t.Fatalf("GenerateUnit: %v", err)
	}

	if strings.Contains(content, "ExecStartPre=/bin/true") {
		t.Error("ExecStartPre should not be /bin/true when UpdateOnRestart=true")
	}
	if !strings.Contains(content, "steamcmd") {
		t.Error("ExecStartPre should contain steamcmd when UpdateOnRestart=true")
	}
	if !strings.Contains(content, "1874900") {
		t.Error("ExecStartPre should contain Reforger app ID")
	}
}

func TestGenerateUnit_ActiveConfig(t *testing.T) {
	inst := setupInstance(t)

	content, err := systemd.GenerateUnit(inst, "")
	if err != nil {
		t.Fatalf("GenerateUnit: %v", err)
	}

	if !strings.Contains(content, "vanilla") {
		t.Error("unit should reference active config name")
	}
	if !strings.Contains(content, "config.json") {
		t.Error("unit should reference config.json path")
	}
	if !strings.Contains(content, "profile") {
		t.Error("unit should reference profile path")
	}
}

func TestGenerateUnit_SyslogIdentifier(t *testing.T) {
	inst := setupInstance(t)

	content, err := systemd.GenerateUnit(inst, "")
	if err != nil {
		t.Fatalf("GenerateUnit: %v", err)
	}

	if !strings.Contains(content, "SyslogIdentifier=rsm-testserver") {
		t.Error("unit should set SyslogIdentifier for journald filtering")
	}
}

func TestGenerateUnit_RestartPolicy(t *testing.T) {
	inst := setupInstance(t)

	content, err := systemd.GenerateUnit(inst, "")
	if err != nil {
		t.Fatalf("GenerateUnit: %v", err)
	}

	if !strings.Contains(content, "Restart=on-failure") {
		t.Error("unit should have Restart=on-failure")
	}
}

func TestGenerateUnit_ExperimentalBranch(t *testing.T) {
	inst := setupInstance(t)
	inst.UpdateOnRestart = true
	inst.Experimental = true

	content, err := systemd.GenerateUnit(inst, "/usr/games/steamcmd")
	if err != nil {
		t.Fatalf("GenerateUnit: %v", err)
	}

	if !strings.Contains(content, "-beta") {
		t.Error("experimental unit should include -beta in ExecStartPre")
	}
	if !strings.Contains(content, "experiment") {
		t.Error("experimental unit should include 'experiment' branch name in ExecStartPre")
	}
}

func TestGenerateUnit_StableNoBetaFlag(t *testing.T) {
	inst := setupInstance(t)
	inst.UpdateOnRestart = true
	inst.Experimental = false

	content, err := systemd.GenerateUnit(inst, "/usr/games/steamcmd")
	if err != nil {
		t.Fatalf("GenerateUnit: %v", err)
	}

	if strings.Contains(content, "-beta") {
		t.Error("stable unit should not include -beta in ExecStartPre")
	}
}
