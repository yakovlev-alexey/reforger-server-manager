package instance

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	metaFileName      = "rsm.yaml"
	configurationsDir = "configuration"
	serviceUnitFile   = "service.unit"
)

// Instance represents a named Arma Reforger server installation.
// All metadata and configurations live inside InstallDir.
type Instance struct {
	Name            string   `yaml:"name"`
	InstallDir      string   `yaml:"install_dir"`
	ActiveConfig    string   `yaml:"active_config"`
	UpdateOnRestart bool     `yaml:"update_on_restart"`
	Experimental    bool     `yaml:"experimental"`
	MaxFPS          int      `yaml:"max_fps"`
	ExtraFlags      []string `yaml:"extra_flags"`
	SystemdUser     string   `yaml:"systemd_user"`
}

// MetaPath returns the path to rsm.yaml inside the install directory.
func (i *Instance) MetaPath() string {
	return filepath.Join(i.InstallDir, metaFileName)
}

// ConfigsDir returns <install_dir>/configuration/
func (i *Instance) ConfigsDir() string {
	return filepath.Join(i.InstallDir, configurationsDir)
}

// ConfigDir returns <install_dir>/configuration/<configName>/
func (i *Instance) ConfigDir(configName string) string {
	return filepath.Join(i.ConfigsDir(), configName)
}

// ConfigJSONPath returns the path to config.json for a named configuration.
func (i *Instance) ConfigJSONPath(configName string) string {
	return filepath.Join(i.ConfigDir(configName), "config.json")
}

// ProfileDir returns the profile directory for a named configuration.
func (i *Instance) ProfileDir(configName string) string {
	return filepath.Join(i.ConfigDir(configName), "profile")
}

// ServiceUnitPath returns the path to the generated systemd unit file copy.
func (i *Instance) ServiceUnitPath() string {
	return filepath.Join(i.InstallDir, serviceUnitFile)
}

// SystemdServiceName returns the systemd service name for this instance.
func (i *Instance) SystemdServiceName() string {
	return fmt.Sprintf("rsm-%s.service", i.Name)
}

// ActiveConfigJSONPath returns the config.json path for the active configuration.
func (i *Instance) ActiveConfigJSONPath() (string, error) {
	if i.ActiveConfig == "" {
		return "", fmt.Errorf("no active configuration set")
	}
	return i.ConfigJSONPath(i.ActiveConfig), nil
}

// ActiveProfileDir returns the profile directory for the active configuration.
func (i *Instance) ActiveProfileDir() (string, error) {
	if i.ActiveConfig == "" {
		return "", fmt.Errorf("no active configuration set")
	}
	return i.ProfileDir(i.ActiveConfig), nil
}

// ListConfigs returns all configuration names for this instance.
func (i *Instance) ListConfigs() ([]string, error) {
	dir := i.ConfigsDir()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading configurations dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// Save persists the instance metadata to <install_dir>/rsm.yaml.
func (i *Instance) Save() error {
	if err := os.MkdirAll(i.InstallDir, 0o755); err != nil {
		return fmt.Errorf("creating install dir: %w", err)
	}
	data, err := yaml.Marshal(i)
	if err != nil {
		return fmt.Errorf("marshalling instance: %w", err)
	}
	return os.WriteFile(i.MetaPath(), data, 0o644)
}

// EnsureConfigDirs creates the configuration and profile directories for a named config.
func EnsureConfigDirs(inst *Instance, configName string) error {
	profileDir := inst.ProfileDir(configName)
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return fmt.Errorf("creating profile dir: %w", err)
	}
	return nil
}
