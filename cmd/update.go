package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/MSmaili/tms/internal/logger"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update tms to the latest version",
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	logger.Info("Updating tms...")

	installCmd := exec.Command("bash", "-c",
		"curl -fsSL https://raw.githubusercontent.com/MSmaili/tms/main/install.sh | bash")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}
