package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yakovlev-alex/reforger-server-manager/internal/config"
)

func TestSaveAndLoadGlobal(t *testing.T) {
	// Use a temp dir as home
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := &config.GlobalConfig{
		SteamCMDPath: "/usr/games/steamcmd",
	}
	if err := config.SaveGlobal(cfg); err != nil {
		t.Fatalf("SaveGlobal: %v", err)
	}

	loaded, err := config.LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal: %v", err)
	}
	if loaded.SteamCMDPath != cfg.SteamCMDPath {
		t.Errorf("SteamCMDPath = %q, want %q", loaded.SteamCMDPath, cfg.SteamCMDPath)
	}
}

func TestLoadGlobalMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := config.LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal on missing file: %v", err)
	}
	if cfg.SteamCMDPath != "" {
		t.Errorf("expected empty SteamCMDPath, got %q", cfg.SteamCMDPath)
	}
}

func TestGlobalConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	path, err := config.GlobalConfigPath()
	if err != nil {
		t.Fatalf("GlobalConfigPath: %v", err)
	}
	want := filepath.Join(tmpDir, ".config", "rsm", "config.yaml")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestSaveGlobalCreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := &config.GlobalConfig{SteamCMDPath: "/test/steamcmd"}
	if err := config.SaveGlobal(cfg); err != nil {
		t.Fatalf("SaveGlobal: %v", err)
	}

	cfgDir, _ := config.GlobalConfigDir()
	if _, err := os.Stat(cfgDir); err != nil {
		t.Errorf("config dir not created: %v", err)
	}
}
