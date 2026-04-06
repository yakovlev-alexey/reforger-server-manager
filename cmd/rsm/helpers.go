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
func regenerateUnit(inst *instance.Instance, steamcmdPath string) error {
	// InstallUnit generates, writes the local copy, installs to /etc/systemd/system/,
	// and calls daemon-reload.
	return systemd.InstallUnit(inst, steamcmdPath)
}
