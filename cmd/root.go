package cmd

import (
	"fmt"
	"os"

	"github.com/MSmaili/tms/internal/logger"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "tms",
	Short: "TMS - Tmux Session Manager",
	Long: `TMS is a powerful tmux session manager that helps you manage complex tmux sessions with ease.

It supports:
- Multiple sessions and windows with panes
- YAML and JSON configuration files
- Named and local workspaces
- Templates for reusable configurations`,
	Version: Version,
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("tms version %s\ncommit: %s\nbuilt: %s\n", Version, GitCommit, BuildDate))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}
}
