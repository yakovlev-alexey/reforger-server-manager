package instance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// ResolveInstance resolves the target instance name using the following priority:
//  1. Explicit name provided via --instance flag.
//  2. CWD matches an instance's install_dir (or is beneath it).
//  3. Only one instance exists — use it automatically.
//  4. Error: ask the user to specify.
func ResolveInstance(name string) (string, error) {
	if name != "" {
		return name, nil
	}

	// Try CWD-based resolution before falling back to single-instance logic.
	if cwdName, err := resolveFromCWD(); err == nil && cwdName != "" {
		return cwdName, nil
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

// resolveFromCWD attempts to identify which instance the current working
// directory belongs to by comparing the CWD against each instance's install_dir.
// Returns the instance name if found, or ("", nil) if no match.
func resolveFromCWD() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil // non-fatal; fall through to other strategies
	}
	// Resolve symlinks so path comparisons are reliable.
	cwd, _ = filepath.EvalSymlinks(cwd)

	names, err := List()
	if err != nil || len(names) == 0 {
		return "", nil
	}

	for _, n := range names {
		inst, err := Load(n)
		if err != nil || inst.InstallDir == "" {
			continue
		}
		installDir, _ := filepath.EvalSymlinks(inst.InstallDir)
		if installDir == "" {
			installDir = inst.InstallDir
		}
		// Match if CWD equals installDir or is a subdirectory of it.
		if cwd == installDir || isSubPath(cwd, installDir) {
			return n, nil
		}
	}
	return "", nil
}

// isSubPath reports whether child is strictly inside parent (not equal, not outside).
func isSubPath(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	// "." means equal (already handled by the == check in the caller).
	// ".." or paths starting with ".." mean child is outside parent.
	if rel == "." || rel == ".." {
		return false
	}
	// Any path component starting with ".." means child escaped parent.
	parts := strings.SplitN(rel, string(filepath.Separator), 2)
	return parts[0] != ".."
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
