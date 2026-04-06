package instance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFromDir reads an instance from a given install directory.
// The metadata file is expected at <dir>/rsm.yaml.
func LoadFromDir(dir string) (*Instance, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving path %s: %w", dir, err)
	}
	path := filepath.Join(absDir, metaFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no rsm.yaml found in %s; run 'rsm init' to create an instance", absDir)
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var inst Instance
	if err := yaml.Unmarshal(data, &inst); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	// Always keep InstallDir canonical
	inst.InstallDir = absDir
	return &inst, nil
}

// Load reads an instance by name from the global instances registry.
// It searches all registered install directories for a matching name.
// If name is empty it falls back to CWD-based resolution.
func Load(name string) (*Instance, error) {
	// If name looks like a path, load directly
	if filepath.IsAbs(name) || strings.HasPrefix(name, ".") {
		return LoadFromDir(name)
	}

	names, err := List()
	if err != nil {
		return nil, err
	}
	for _, n := range names {
		if n == name {
			dir, err := registeredDir(name)
			if err != nil {
				return nil, err
			}
			return LoadFromDir(dir)
		}
	}
	return nil, fmt.Errorf("instance %q not found", name)
}

// ResolveInstance resolves the target instance directory using the following priority:
//  1. Explicit name provided via --instance flag → look up in registry.
//  2. CWD contains rsm.yaml → load from CWD.
//  3. CWD is inside a registered install_dir → load that instance.
//  4. Only one registered instance exists → use it.
//  5. Error.
func ResolveInstance(name string) (string, error) {
	if name != "" {
		return name, nil
	}

	// Check CWD for rsm.yaml
	if inst, err := resolveFromCWD(); err == nil && inst != "" {
		return inst, nil
	}

	names, err := List()
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no instances found; run 'rsm init' to create one")
	}
	if len(names) == 1 {
		return names[0], nil
	}
	return "", fmt.Errorf("multiple instances exist; specify one with --instance <name>")
}

// resolveFromCWD returns the instance name if the CWD (or a parent) contains rsm.yaml,
// or if the CWD is inside a registered instance's install_dir.
func resolveFromCWD() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil
	}
	cwd, _ = filepath.EvalSymlinks(cwd)

	// Walk up from CWD looking for rsm.yaml
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, metaFileName)); err == nil {
			inst, err := LoadFromDir(dir)
			if err == nil {
				return inst.Name, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fall back to registry: check if CWD is inside a known install_dir
	names, err := List()
	if err != nil || len(names) == 0 {
		return "", nil
	}
	for _, n := range names {
		instDir, err := registeredDir(n)
		if err != nil {
			continue
		}
		installDir, _ := filepath.EvalSymlinks(instDir)
		if installDir == "" {
			installDir = instDir
		}
		if cwd == installDir || isSubPath(cwd, installDir) {
			return n, nil
		}
	}
	return "", nil
}

// List returns all registered instance names from the global registry file.
func List() ([]string, error) {
	reg, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(reg))
	for name := range reg {
		names = append(names, name)
	}
	return names, nil
}

// Register adds an instance to the global registry (name → install_dir).
func Register(inst *Instance) error {
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	reg[inst.Name] = inst.InstallDir
	return saveRegistry(reg)
}

// Unregister removes an instance from the global registry.
func Unregister(name string) error {
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	delete(reg, name)
	return saveRegistry(reg)
}

// Delete unregisters an instance and optionally wipes its install directory.
func Delete(name string, wipeInstallDir bool) error {
	inst, err := Load(name)
	if err != nil {
		return err
	}
	if err := Unregister(name); err != nil {
		return err
	}
	if wipeInstallDir && inst.InstallDir != "" {
		if err := os.RemoveAll(inst.InstallDir); err != nil {
			return fmt.Errorf("removing install dir %s: %w", inst.InstallDir, err)
		}
	}
	return nil
}

// ── registry ──────────────────────────────────────────────────────────────────
// The registry is a simple YAML map[name]installDir stored in
// ~/.config/rsm/registry.yaml. It is the only file that lives outside
// the install directory.

type registry map[string]string

func registryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "rsm", "registry.yaml"), nil
}

func registeredDir(name string) (string, error) {
	reg, err := loadRegistry()
	if err != nil {
		return "", err
	}
	dir, ok := reg[name]
	if !ok {
		return "", fmt.Errorf("instance %q not found in registry", name)
	}
	return dir, nil
}

func loadRegistry() (registry, error) {
	path, err := registryPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return registry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading registry: %w", err)
	}
	var reg registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}
	if reg == nil {
		reg = registry{}
	}
	return reg, nil
}

func saveRegistry(reg registry) error {
	path, err := registryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := yaml.Marshal(reg)
	if err != nil {
		return fmt.Errorf("marshalling registry: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// isSubPath reports whether child is strictly inside parent.
func isSubPath(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	if rel == "." || rel == ".." {
		return false
	}
	parts := strings.SplitN(rel, string(filepath.Separator), 2)
	return parts[0] != ".."
}
