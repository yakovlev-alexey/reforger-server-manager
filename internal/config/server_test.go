package config_test

import (
	"encoding/json"
	"testing"

	"github.com/yakovlev-alex/reforger-server-manager/internal/config"
)

func TestDefaultServerConfig(t *testing.T) {
	cfg := config.DefaultServerConfig(
		"Test Server",
		"0.0.0.0",
		"1.2.3.4",
		2001, 17777, 64,
		"adminpass", "",
	)

	if cfg.BindAddress != "0.0.0.0" {
		t.Errorf("BindAddress = %q, want 0.0.0.0", cfg.BindAddress)
	}
	if cfg.PublicAddress != "1.2.3.4" {
		t.Errorf("PublicAddress = %q, want 1.2.3.4", cfg.PublicAddress)
	}
	if cfg.BindPort != 2001 {
		t.Errorf("BindPort = %d, want 2001", cfg.BindPort)
	}
	if cfg.A2S.Port != 17777 {
		t.Errorf("A2S.Port = %d, want 17777", cfg.A2S.Port)
	}
	if cfg.Game.MaxPlayers != 64 {
		t.Errorf("MaxPlayers = %d, want 64", cfg.Game.MaxPlayers)
	}
	if cfg.Game.PasswordAdmin != "adminpass" {
		t.Errorf("PasswordAdmin = %q, want adminpass", cfg.Game.PasswordAdmin)
	}
	if cfg.Game.Password != "" {
		t.Errorf("Password = %q, want empty", cfg.Game.Password)
	}
	if cfg.Game.ScenarioID != config.EveronGameMasterScenarioID {
		t.Errorf("ScenarioID = %q, want %q", cfg.Game.ScenarioID, config.EveronGameMasterScenarioID)
	}
}

func TestDefaultServerConfigModsIsEmptyArray(t *testing.T) {
	cfg := config.DefaultServerConfig("S", "0.0.0.0", "", 2001, 17777, 10, "pw", "")

	// Mods must marshal as [] not null
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	game, ok := raw["game"].(map[string]interface{})
	if !ok {
		t.Fatal("missing 'game' key in JSON")
	}
	mods, ok := game["mods"].([]interface{})
	if !ok {
		t.Fatalf("'mods' is not an array in JSON, got %T", game["mods"])
	}
	if len(mods) != 0 {
		t.Errorf("mods len = %d, want 0", len(mods))
	}
}

func TestDefaultServerConfigVisibleByDefault(t *testing.T) {
	cfg := config.DefaultServerConfig("S", "0.0.0.0", "", 2001, 17777, 10, "pw", "")
	if !cfg.Game.Visible {
		t.Error("expected Visible=true by default")
	}
}

func TestDefaultServerConfigBattlEyeEnabled(t *testing.T) {
	cfg := config.DefaultServerConfig("S", "0.0.0.0", "", 2001, 17777, 10, "pw", "")
	if !cfg.Game.GameProperties.BattlEye {
		t.Error("expected BattlEye=true by default")
	}
}

func TestEveronScenarioID(t *testing.T) {
	want := "{ECC61978EDCC2B5A}Missions/23_Campaign.conf"
	if config.EveronGameMasterScenarioID != want {
		t.Errorf("EveronGameMasterScenarioID = %q, want %q", config.EveronGameMasterScenarioID, want)
	}
}
