package instance

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yakovlev-alex/reforger-server-manager/internal/config"
	"gopkg.in/yaml.v3"
)

// instanceDir returns the metadata directory for a named instance.
func instanceDir(name string) (string, error) {
	base, err := config.InstancesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, name), nil
}

// Load reads an instance from disk by name.
func Load(name string) (*Instance, error) {
	dir, err := instanceDir(name)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "instance.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("instance %q not found", name)
	}
	if err != nil {
		return nil, fmt.Errorf("reading instance %q: %w", name, err)
	}
	var inst Instance
	if err := yaml.Unmarshal(data, &inst); err != nil {
		return nil, fmt.Errorf("parsing instance %q: %w", name, err)
	}
	return &inst, nil
}

// List returns all instance names.
func List() ([]string, error) {
	base, err := config.InstancesDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading instances dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Check that instance.yaml exists
		metaPath := filepath.Join(base, e.Name(), "instance.yaml")
		if _, err := os.Stat(metaPath); err == nil {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ResolveInstance resolves the target instance name:
// - If name is provided, use it.
// - If only one instance exists, use it automatically.
// - Otherwise return an error asking the user to specify.
func ResolveInstance(name string) (string, error) {
	if name != "" {
		return name, nil
	}
	names, err := List()
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no instances found; run 'rsm instance new' to create one")
	}
	if len(names) == 1 {
		return names[0], nil
	}
	return "", fmt.Errorf("multiple instances exist; specify one with --instance <name>")
}

// Delete removes an instance's metadata directory.
// Set wipeInstallDir=true to also remove the server binary directory.
func Delete(name string, wipeInstallDir bool) error {
	inst, err := Load(name)
	if err != nil {
		return err
	}

	dir, err := instanceDir(name)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing instance dir: %w", err)
	}

	if wipeInstallDir && inst.InstallDir != "" {
		if err := os.RemoveAll(inst.InstallDir); err != nil {
			return fmt.Errorf("removing install dir %s: %w", inst.InstallDir, err)
		}
	}
	return nil
}

// EnsureConfigDirs creates the config and profile directories for a named config.
func EnsureConfigDirs(inst *Instance, configName string) error {
	profileDir, err := inst.ProfileDir(configName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return fmt.Errorf("creating profile dir: %w", err)
	}
	return nil
}
