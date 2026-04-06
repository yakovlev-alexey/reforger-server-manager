package main

import (
	"github.com/spf13/cobra"
	"github.com/yakovlev-alex/reforger-server-manager/internal/instance"
	rsmLogs "github.com/yakovlev-alex/reforger-server-manager/internal/logs"
)

var (
	flagLogsFollow bool
	flagLogsLines  int
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View server logs via journalctl",
	Long: `Streams server logs from systemd journal.

Use -f to follow logs in real time (like 'tail -f').
Use -n to control how many previous lines to show.`,
	RunE: runLogs,
}

func init() {
	logsCmd.Flags().BoolVarP(&flagLogsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&flagLogsLines, "lines", "n", 50, "Number of previous lines to show (0 = all)")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(_ *cobra.Command, _ []string) error {
	resolved, err := instance.ResolveInstance(flagInstance)
	if err != nil {
		return err
	}
	inst, err := instance.Load(resolved)
	if err != nil {
		return err
	}

	return rsmLogs.Stream(inst, flagLogsFollow, flagLogsLines)
}
