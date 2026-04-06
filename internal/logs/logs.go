package logs

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
)

// Stream streams logs for an instance via journalctl.
// If follow is true, tails the log continuously (-f).
// lines controls how many prior lines to show (0 = journalctl default).
func Stream(inst *instance.Instance, follow bool, lines int) error {
	args := []string{
		"-u", inst.SystemdServiceName(),
		"--no-pager",
	}
	if follow {
		args = append(args, "-f")
	}
	if lines > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", lines))
	}

	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		// journalctl exits non-zero when interrupted (Ctrl-C), which is normal
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil
		}
		return fmt.Errorf("journalctl: %w", err)
	}
	return nil
}
