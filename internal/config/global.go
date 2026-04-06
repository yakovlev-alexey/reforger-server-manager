package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const globalConfigFileName = "config.yaml"

// GlobalConfig holds rsm-wide settings persisted to ~/.config/rsm/config.yaml
type GlobalConfig struct {
	SteamCMDPath string `yaml:"steamcmd_path"`
}

// GlobalConfigDir returns the rsm config directory (~/.config/rsm)
func GlobalConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "rsm"), nil
}

// GlobalConfigPath returns the full path to the global config file
func GlobalConfigPath() (string, error) {
	dir, err := GlobalConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, globalConfigFileName), nil
}

// LoadGlobal reads the global config from disk. Returns empty config if file doesn't exist.
func LoadGlobal() (*GlobalConfig, error) {
	path, err := GlobalConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &GlobalConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading global config: %w", err)
	}

	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing global config: %w", err)
	}
	return &cfg, nil
}

// SaveGlobal writes the global config to disk, creating directories as needed.
func SaveGlobal(cfg *GlobalConfig) error {
	dir, err := GlobalConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	path := filepath.Join(dir, globalConfigFileName)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling global config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing global config: %w", err)
	}
	return nil
}
