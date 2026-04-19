package cmd

import (
	apptui "github.com/MSmaili/hetki/internal/app/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open an interactive TUI for live tmux sessions",
	Long: `Open an interactive terminal UI for browsing live sessions, windows, and panes.

This command is live-first and reads from the running tmux server.`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	service := apptui.NewService(detectBackend)
	return service.Run(cmd.Context())
}
