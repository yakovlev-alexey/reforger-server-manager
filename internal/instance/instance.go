package instance

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Instance represents a named Arma Reforger server installation.
type Instance struct {
	Name            string   `yaml:"name"`
	InstallDir      string   `yaml:"install_dir"`
	ActiveConfig    string   `yaml:"active_config"`
	UpdateOnRestart bool     `yaml:"update_on_restart"`
	MaxFPS          int      `yaml:"max_fps"`
	ExtraFlags      []string `yaml:"extra_flags"`
	SystemdUser     string   `yaml:"systemd_user"`
}

// Dir returns the rsm metadata directory for this instance.
func (i *Instance) Dir() (string, error) {
	return instanceDir(i.Name)
}

// ConfigsDir returns the directory containing all named configurations.
func (i *Instance) ConfigsDir() (string, error) {
	dir, err := instanceDir(i.Name)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "configs"), nil
}

// ConfigDir returns the directory for a specific named configuration.
func (i *Instance) ConfigDir(configName string) (string, error) {
	dir, err := i.ConfigsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configName), nil
}

// ConfigJSONPath returns the path to config.json for a named configuration.
func (i *Instance) ConfigJSONPath(configName string) (string, error) {
	dir, err := i.ConfigDir(configName)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// ProfileDir returns the profile directory for a named configuration.
func (i *Instance) ProfileDir(configName string) (string, error) {
	dir, err := i.ConfigDir(configName)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "profile"), nil
}

// ServiceUnitPath returns the path to the generated systemd unit file copy.
func (i *Instance) ServiceUnitPath() (string, error) {
	dir, err := instanceDir(i.Name)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "service.unit"), nil
}

// SystemdServiceName returns the systemd service name for this instance.
func (i *Instance) SystemdServiceName() string {
	return fmt.Sprintf("rsm-%s.service", i.Name)
}

// ActiveConfigJSONPath returns the config.json path for the active configuration.
func (i *Instance) ActiveConfigJSONPath() (string, error) {
	return i.ConfigJSONPath(i.ActiveConfig)
}

// ActiveProfileDir returns the profile directory for the active configuration.
func (i *Instance) ActiveProfileDir() (string, error) {
	return i.ProfileDir(i.ActiveConfig)
}

// ListConfigs returns all configuration names for this instance.
func (i *Instance) ListConfigs() ([]string, error) {
	dir, err := i.ConfigsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading configs dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// Save persists the instance metadata to disk.
func (i *Instance) Save() error {
	dir, err := instanceDir(i.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating instance dir: %w", err)
	}
	path := filepath.Join(dir, "instance.yaml")
	data, err := yaml.Marshal(i)
	if err != nil {
		return fmt.Errorf("marshalling instance: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
