package systemd

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"text/template"

	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/steam"
)

//go:embed service.unit.tmpl
var templateFS embed.FS

// unitParams holds template data for service.unit.tmpl
type unitParams struct {
	InstanceName string
	ActiveConfig string
	User         string
	InstallDir   string
	ExecStartPre string
	ConfigPath   string
	ProfilePath  string
	MaxFPS       int
	ExtraFlags   []string
}

// GenerateUnit renders the systemd unit file content for an instance.
func GenerateUnit(inst *instance.Instance, steamcmdPath string) (string, error) {
	configPath, err := inst.ActiveConfigJSONPath()
	if err != nil {
		return "", err
	}
	profilePath, err := inst.ActiveProfileDir()
	if err != nil {
		return "", err
	}

	execStartPre := "/bin/true"
	if inst.UpdateOnRestart && steamcmdPath != "" {
		execStartPre = steam.BuildUpdateCommand(steamcmdPath, inst.InstallDir, inst.Experimental)
	}

	params := unitParams{
		InstanceName: inst.Name,
		ActiveConfig: inst.ActiveConfig,
		User:         inst.SystemdUser,
		InstallDir:   inst.InstallDir,
		ExecStartPre: execStartPre,
		ConfigPath:   configPath,
		ProfilePath:  profilePath,
		MaxFPS:       inst.MaxFPS,
		ExtraFlags:   inst.ExtraFlags,
	}

	tmplData, err := templateFS.ReadFile("service.unit.tmpl")
	if err != nil {
		return "", fmt.Errorf("reading unit template: %w", err)
	}

	tmpl, err := template.New("unit").Parse(string(tmplData))
	if err != nil {
		return "", fmt.Errorf("parsing unit template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("rendering unit template: %w", err)
	}
	return buf.String(), nil
}

// InstallUnit writes the unit file to /etc/systemd/system/ and reloads the
// daemon. Requires sudo — the caller should print a notice before calling.
func InstallUnit(inst *instance.Instance, steamcmdPath string) error {
	content, err := GenerateUnit(inst, steamcmdPath)
	if err != nil {
		return err
	}

	// Save a reference copy inside the install directory (no privilege needed).
	localPath := inst.ServiceUnitPath()
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing local unit file: %w", err)
	}

	// sudo tee writes to /etc/systemd/system/
	systemdPath := "/etc/systemd/system/" + inst.SystemdServiceName()
	sudoNotice("writing %s", systemdPath)
	if err := writeFileWithSudo(systemdPath, content); err != nil {
		return fmt.Errorf("installing unit to %s: %w", systemdPath, err)
	}

	sudoNotice("running systemctl daemon-reload")
	return daemonReload()
}

// ReinstallUnit regenerates and re-installs the unit file unconditionally.
// Use this after changing configuration, active config, or launch flags.
func ReinstallUnit(inst *instance.Instance, steamcmdPath string) error {
	return InstallUnit(inst, steamcmdPath)
}

// IsInstalled reports whether the unit file exists in /etc/systemd/system/.
func IsInstalled(inst *instance.Instance) bool {
	_, err := os.Stat("/etc/systemd/system/" + inst.SystemdServiceName())
	return err == nil
}

// EnsureInstalled generates and installs the unit if it is not already present.
func EnsureInstalled(inst *instance.Instance, steamcmdPath string) error {
	if IsInstalled(inst) {
		return nil
	}
	return InstallUnit(inst, steamcmdPath)
}

// RemoveUnit removes the systemd unit file and reloads the daemon.
// Requires sudo.
func RemoveUnit(inst *instance.Instance) error {
	systemdPath := "/etc/systemd/system/" + inst.SystemdServiceName()
	sudoNotice("removing %s", systemdPath)
	if err := runWithSudo("rm", "-f", systemdPath); err != nil {
		return fmt.Errorf("removing unit file: %w", err)
	}
	sudoNotice("running systemctl daemon-reload")
	return daemonReload()
}

// Start starts the systemd service. Requires sudo.
func Start(inst *instance.Instance) error {
	sudoNotice("systemctl start %s", inst.SystemdServiceName())
	return systemctl("start", inst.SystemdServiceName())
}

// Stop stops the systemd service. Requires sudo.
func Stop(inst *instance.Instance) error {
	sudoNotice("systemctl stop %s", inst.SystemdServiceName())
	return systemctl("stop", inst.SystemdServiceName())
}

// Restart restarts the systemd service. Requires sudo.
func Restart(inst *instance.Instance) error {
	sudoNotice("systemctl restart %s", inst.SystemdServiceName())
	return systemctl("restart", inst.SystemdServiceName())
}

// Enable enables autostart for the systemd service. Requires sudo.
func Enable(inst *instance.Instance) error {
	sudoNotice("systemctl enable %s", inst.SystemdServiceName())
	return systemctl("enable", inst.SystemdServiceName())
}

// Disable disables autostart for the systemd service. Requires sudo.
func Disable(inst *instance.Instance) error {
	sudoNotice("systemctl disable %s", inst.SystemdServiceName())
	return systemctl("disable", inst.SystemdServiceName())
}

// Status returns the raw output of systemctl status.
func Status(inst *instance.Instance) (string, error) {
	cmd := exec.Command("systemctl", "status", inst.SystemdServiceName())
	out, _ := cmd.CombinedOutput() // non-zero exit when stopped is normal
	return string(out), nil
}

// IsActive returns true if the service is currently active (running).
func IsActive(inst *instance.Instance) bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", inst.SystemdServiceName())
	return cmd.Run() == nil
}

// IsEnabled returns true if the service is enabled for autostart.
func IsEnabled(inst *instance.Instance) bool {
	cmd := exec.Command("systemctl", "is-enabled", "--quiet", inst.SystemdServiceName())
	return cmd.Run() == nil
}

// sudoNotice prints a short message explaining what sudo will be used for.
// It goes to stderr so it doesn't mix with structured output.
func sudoNotice(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "  [sudo] %s\n", fmt.Sprintf(format, args...))
}

func systemctl(action, service string) error {
	cmd := exec.Command("sudo", "systemctl", action, service)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl %s %s: %w", action, service, err)
	}
	return nil
}

func daemonReload() error {
	cmd := exec.Command("sudo", "systemctl", "daemon-reload")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	return nil
}

func writeFileWithSudo(path, content string) error {
	cmd := exec.Command("sudo", "tee", path)
	cmd.Stdin = bytes.NewBufferString(content)
	// Suppress tee's stdout echo (it would print the entire unit file)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("writing %s via sudo tee: %w", path, err)
	}
	return nil
}

func runWithSudo(args ...string) error {
	cmd := exec.Command("sudo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
