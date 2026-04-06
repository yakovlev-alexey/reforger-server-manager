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
		execStartPre = steam.BuildUpdateCommand(steamcmdPath, inst.InstallDir)
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

// InstallUnit writes the unit file to /etc/systemd/system/ and reloads the daemon.
// This requires root privileges (or sudo).
func InstallUnit(inst *instance.Instance, steamcmdPath string) error {
	content, err := GenerateUnit(inst, steamcmdPath)
	if err != nil {
		return err
	}

	// Save a reference copy next to instance.yaml
	localPath, err := inst.ServiceUnitPath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing local unit file: %w", err)
	}

	systemdPath := "/etc/systemd/system/" + inst.SystemdServiceName()
	if err := writeFileWithSudo(systemdPath, content); err != nil {
		return fmt.Errorf("installing unit to %s: %w", systemdPath, err)
	}

	return daemonReload()
}

// RemoveUnit removes the systemd unit file and reloads the daemon.
func RemoveUnit(inst *instance.Instance) error {
	systemdPath := "/etc/systemd/system/" + inst.SystemdServiceName()
	if err := runWithSudo("rm", "-f", systemdPath); err != nil {
		return fmt.Errorf("removing unit file: %w", err)
	}
	return daemonReload()
}

// Start starts the systemd service.
func Start(inst *instance.Instance) error {
	return systemctl("start", inst.SystemdServiceName())
}

// Stop stops the systemd service.
func Stop(inst *instance.Instance) error {
	return systemctl("stop", inst.SystemdServiceName())
}

// Restart restarts the systemd service.
func Restart(inst *instance.Instance) error {
	return systemctl("restart", inst.SystemdServiceName())
}

// Enable enables autostart for the systemd service.
func Enable(inst *instance.Instance) error {
	return systemctl("enable", inst.SystemdServiceName())
}

// Disable disables autostart for the systemd service.
func Disable(inst *instance.Instance) error {
	return systemctl("disable", inst.SystemdServiceName())
}

// Status returns the raw output of systemctl status.
func Status(inst *instance.Instance) (string, error) {
	cmd := exec.Command("systemctl", "status", inst.SystemdServiceName())
	out, _ := cmd.CombinedOutput() // exit code is non-zero when stopped; that's ok
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
	cmd.Stdout = os.Stdout
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
