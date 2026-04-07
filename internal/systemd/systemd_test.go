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
	if !strings.Contains(content, "1874900") && !strings.Contains(content, "1890870") {
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

	if !strings.Contains(content, "1890870") {
		t.Error("experimental unit should use the experimental app ID (1890870) in ExecStartPre")
	}
	if strings.Contains(content, "1874900") {
		t.Error("experimental unit should not use the stable app ID (1874900) in ExecStartPre")
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

	if strings.Contains(content, "1890870") {
		t.Error("stable unit should not use the experimental app ID (1890870) in ExecStartPre")
	}
	if !strings.Contains(content, "1874900") {
		t.Error("stable unit should use the stable app ID (1874900) in ExecStartPre")
	}
}

func TestGenerateRestartTimer_Basic(t *testing.T) {
	inst := setupInstance(t)
	inst.PeriodicRestart = "6h"

	content, err := systemd.GenerateRestartTimer(inst)
	if err != nil {
		t.Fatalf("GenerateRestartTimer: %v", err)
	}

	if !strings.Contains(content, "[Timer]") {
		t.Error("timer unit should contain [Timer] section")
	}
	if !strings.Contains(content, "OnUnitActiveSec=6h") {
		t.Error("timer unit should contain OnUnitActiveSec=6h")
	}
	if !strings.Contains(content, "rsm-testserver.service") {
		t.Error("timer unit should reference the service")
	}
	if !strings.Contains(content, "[Install]") {
		t.Error("timer unit should contain [Install] section")
	}
	if !strings.Contains(content, "timers.target") {
		t.Error("timer unit should be wanted by timers.target")
	}
}

func TestGenerateRestartService_Basic(t *testing.T) {
	inst := setupInstance(t)
	inst.PeriodicRestart = "12h"

	content, err := systemd.GenerateRestartService(inst)
	if err != nil {
		t.Fatalf("GenerateRestartService: %v", err)
	}

	if !strings.Contains(content, "[Service]") {
		t.Error("restart service unit should contain [Service] section")
	}
	if !strings.Contains(content, "Type=oneshot") {
		t.Error("restart service should be Type=oneshot")
	}
	if !strings.Contains(content, "rsm-testserver.service") {
		t.Error("restart service should reference the main service")
	}
	if !strings.Contains(content, "systemctl restart") {
		t.Error("restart service should call systemctl restart")
	}
}

func TestGenerateRestartTimer_DifferentIntervals(t *testing.T) {
	cases := []string{"6h", "12h", "1d", "2d", "3h30min"}
	inst := setupInstance(t)
	for _, interval := range cases {
		inst.PeriodicRestart = interval
		content, err := systemd.GenerateRestartTimer(inst)
		if err != nil {
			t.Fatalf("GenerateRestartTimer(%s): %v", interval, err)
		}
		if !strings.Contains(content, "OnUnitActiveSec="+interval) {
			t.Errorf("timer unit should contain OnUnitActiveSec=%s", interval)
		}
	}
}

func TestSystemdTimerName(t *testing.T) {
	inst := setupInstance(t)
	if inst.SystemdTimerName() != "rsm-testserver-restart.timer" {
		t.Errorf("unexpected timer name: %s", inst.SystemdTimerName())
	}
	if inst.SystemdTimerServiceName() != "rsm-testserver-restart.service" {
		t.Errorf("unexpected timer service name: %s", inst.SystemdTimerServiceName())
	}
}
