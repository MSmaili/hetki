package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MSmaili/muxie/internal/logger"
	"github.com/spf13/cobra"
)

const (
	modulePath       = "github.com/MSmaili/muxie@latest"
	modulePathSource = "github.com/MSmaili/muxie@main"
)

var (
	updateFromSource bool
	updateDryRun     bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update muxie to the latest version",
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateFromSource, "source", false, "Build from source instead of using release")
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Show what would be done without updating")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	updater, err := determineUpdater(exePath)
	if err != nil {
		return err
	}

	logger.Info("Detected installation method", "method", updater.Name())

	if updateDryRun {
		updater.DryRun()
		return nil
	}

	if err := updater.Update(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	logger.Success("Update completed successfully")
	return nil
}

type Updater interface {
	Name() string
	Update() error
	DryRun()
}

func determineUpdater(exePath string) (Updater, error) {
	if installedViaGo(exePath) {
		return &GoUpdater{}, nil
	}

	return nil, errors.New(
		"muxie was not installed via `go install`; updates for script installs are not supported yet",
	)
}

type GoUpdater struct{}

func (g *GoUpdater) Name() string { return "go install" }

func (g *GoUpdater) DryRun() {
	module := modulePath
	if updateFromSource {
		module = modulePathSource
	}

	logger.Info("Dry run", "command", "go install "+module)
}

func (g *GoUpdater) Update() error {
	if _, err := exec.LookPath("go"); err != nil {
		return errors.New("go binary not found in PATH")
	}

	module := modulePath
	if updateFromSource {
		module = modulePathSource
	}

	cmd := exec.Command("go", "install", module)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logger.Info("Running", "command", strings.Join(cmd.Args, " "))
	return cmd.Run()
}

func installedViaGo(exePath string) bool {
	exeReal, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return false
	}

	for _, dir := range goBinDirs() {
		dirReal, err := filepath.EvalSymlinks(dir)
		if err != nil {
			continue
		}

		if isWithinDir(exeReal, dirReal) {
			return true
		}
	}

	return false
}

func goBinDirs() []string {
	var dirs []string

	if gobin := os.Getenv("GOBIN"); gobin != "" {
		dirs = append(dirs, gobin)
	}

	if gopath := os.Getenv("GOPATH"); gopath != "" {
		for _, p := range filepath.SplitList(gopath) {
			dirs = append(dirs, filepath.Join(p, "bin"))
		}
	}

	if len(dirs) == 0 {
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs, filepath.Join(home, "go", "bin"))
		}
	}

	return dirs
}

func isWithinDir(file, dir string) bool {
	rel, err := filepath.Rel(dir, file)
	return err == nil && !strings.HasPrefix(rel, "..")
}
