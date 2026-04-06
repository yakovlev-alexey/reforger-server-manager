package instance_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
)

func setupHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	return tmpDir
}

func makeInstance(t *testing.T, name string) *instance.Instance {
	t.Helper()
	inst := &instance.Instance{
		Name:            name,
		InstallDir:      "/home/steam/reforger",
		ActiveConfig:    "vanilla",
		UpdateOnRestart: false,
		MaxFPS:          60,
		ExtraFlags:      []string{"-loadSessionSave", "-backendLocalStorage"},
		SystemdUser:     "steam",
	}
	if err := inst.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return inst
}

func TestSaveAndLoad(t *testing.T) {
	setupHome(t)
	orig := makeInstance(t, "test-server")

	loaded, err := instance.Load("test-server")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Name != orig.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, orig.Name)
	}
	if loaded.ActiveConfig != orig.ActiveConfig {
		t.Errorf("ActiveConfig = %q, want %q", loaded.ActiveConfig, orig.ActiveConfig)
	}
	if loaded.MaxFPS != orig.MaxFPS {
		t.Errorf("MaxFPS = %d, want %d", loaded.MaxFPS, orig.MaxFPS)
	}
	if len(loaded.ExtraFlags) != len(orig.ExtraFlags) {
		t.Errorf("ExtraFlags len = %d, want %d", len(loaded.ExtraFlags), len(orig.ExtraFlags))
	}
}

func TestLoadNotFound(t *testing.T) {
	setupHome(t)
	_, err := instance.Load("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent instance")
	}
}

func TestList(t *testing.T) {
	setupHome(t)

	names, err := instance.List()
	if err != nil {
		t.Fatalf("List (empty): %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0 instances, got %d", len(names))
	}

	makeInstance(t, "alpha")
	makeInstance(t, "beta")

	names, err = instance.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("expected 2 instances, got %d", len(names))
	}
}

func TestResolveInstanceSingle(t *testing.T) {
	setupHome(t)
	makeInstance(t, "only")

	resolved, err := instance.ResolveInstance("")
	if err != nil {
		t.Fatalf("ResolveInstance: %v", err)
	}
	if resolved != "only" {
		t.Errorf("resolved = %q, want 'only'", resolved)
	}
}

func TestResolveInstanceExplicit(t *testing.T) {
	setupHome(t)
	makeInstance(t, "alpha")
	makeInstance(t, "beta")

	resolved, err := instance.ResolveInstance("beta")
	if err != nil {
		t.Fatalf("ResolveInstance: %v", err)
	}
	if resolved != "beta" {
		t.Errorf("resolved = %q, want 'beta'", resolved)
	}
}

func TestResolveInstanceAmbiguous(t *testing.T) {
	setupHome(t)
	makeInstance(t, "alpha")
	makeInstance(t, "beta")

	_, err := instance.ResolveInstance("")
	if err == nil {
		t.Error("expected error when multiple instances and no name given")
	}
}

func TestResolveInstanceNone(t *testing.T) {
	setupHome(t)
	_, err := instance.ResolveInstance("")
	if err == nil {
		t.Error("expected error when no instances exist")
	}
}

func TestSystemdServiceName(t *testing.T) {
	inst := &instance.Instance{Name: "main"}
	want := "rsm-main.service"
	if got := inst.SystemdServiceName(); got != want {
		t.Errorf("SystemdServiceName = %q, want %q", got, want)
	}
}

func TestListConfigs(t *testing.T) {
	setupHome(t)
	inst := makeInstance(t, "srv")

	// No configs yet
	configs, err := inst.ListConfigs()
	if err != nil {
		t.Fatalf("ListConfigs: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("expected 0 configs, got %d", len(configs))
	}

	// Create config dirs manually
	if err := instance.EnsureConfigDirs(inst, "vanilla"); err != nil {
		t.Fatalf("EnsureConfigDirs: %v", err)
	}
	if err := instance.EnsureConfigDirs(inst, "modded"); err != nil {
		t.Fatalf("EnsureConfigDirs: %v", err)
	}

	configs, err = inst.ListConfigs()
	if err != nil {
		t.Fatalf("ListConfigs after creation: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(configs))
	}
}

func TestDelete(t *testing.T) {
	setupHome(t)
	makeInstance(t, "deleteme")

	if err := instance.Delete("deleteme", false); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	names, _ := instance.List()
	for _, n := range names {
		if n == "deleteme" {
			t.Error("instance still listed after delete")
		}
	}
}

func TestConfigJSONPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	inst := makeInstance(t, "pathtest")

	path, err := inst.ConfigJSONPath("vanilla")
	if err != nil {
		t.Fatalf("ConfigJSONPath: %v", err)
	}
	if filepath.Base(path) != "config.json" {
		t.Errorf("expected config.json, got %q", filepath.Base(path))
	}
	if filepath.Base(filepath.Dir(path)) != "vanilla" {
		t.Errorf("expected parent dir 'vanilla', got %q", filepath.Base(filepath.Dir(path)))
	}
}

func TestProfileDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	inst := makeInstance(t, "proftest")

	if err := instance.EnsureConfigDirs(inst, "vanilla"); err != nil {
		t.Fatalf("EnsureConfigDirs: %v", err)
	}

	profileDir, err := inst.ProfileDir("vanilla")
	if err != nil {
		t.Fatalf("ProfileDir: %v", err)
	}
	if _, err := os.Stat(profileDir); err != nil {
		t.Errorf("profile dir not created: %v", err)
	}
}
