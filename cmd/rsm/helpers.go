package main

import (
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	"github.com/yakovlev-alex/reforger-server-manager/internal/systemd"
)

// isInstanceRunning returns true if the instance's systemd service is active.
func isInstanceRunning(inst *instance.Instance) bool {
	return systemd.IsActive(inst)
}

// regenerateUnit re-renders and reinstalls the systemd unit file for an instance.
// It also syncs the periodic restart timer: installs it if PeriodicRestart is set,
// removes it otherwise.
func regenerateUnit(inst *instance.Instance, steamcmdPath string) error {
	// InstallUnit generates, writes the local copy, installs to /etc/systemd/system/,
	// and calls daemon-reload.
	if err := systemd.InstallUnit(inst, steamcmdPath); err != nil {
		return err
	}
	return syncRestartTimer(inst)
}

// syncRestartTimer installs or removes the periodic restart timer based on
// the instance's PeriodicRestart setting.
func syncRestartTimer(inst *instance.Instance) error {
	if inst.PeriodicRestart != "" {
		return systemd.InstallRestartTimer(inst)
	}
	if systemd.IsRestartTimerInstalled(inst) {
		return systemd.RemoveRestartTimer(inst)
	}
	return nil
}
