package cmd

import (
	"fmt"

	"github.com/MSmaili/tms/internal/manifest"
	"github.com/MSmaili/tms/internal/tmux"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [workspace-name-or-path]",
	Short: "Start a tmux workspace",
	RunE:  runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	var nameOrPath string
	if len(args) > 0 {
		nameOrPath = args[0]
	}

	resolver := manifest.NewResolver()
	workspacePath, err := resolver.Resolve(nameOrPath)
	if err != nil {
		return err
	}

	loader := manifest.NewFileLoader(workspacePath)
	workspace, err := loader.Load()
	if err != nil {
		return fmt.Errorf("loading workspace: %w", err)
	}

	client, err := tmux.New()
	if err != nil {
		return fmt.Errorf("initializing tmux client: %w", err)
	}

	actions := buildActions(workspace)
	for _, action := range actions {
		if err := client.Execute(action); err != nil {
			return fmt.Errorf("executing action: %w", err)
		}
	}

	for sessionName := range workspace.Sessions {
		return client.Attach(sessionName)
	}

	return nil
}

func buildActions(workspace *manifest.Workspace) []tmux.Action {
	var actions []tmux.Action

	for sessionName, windows := range workspace.Sessions {
		if len(windows) == 0 {
			continue
		}

		first := windows[0]
		actions = append(actions, tmux.CreateSession{
			Name: sessionName,
			Path: first.Path,
		})

		actions = append(actions, buildWindowActions(sessionName, first)...)

		for _, w := range windows[1:] {
			actions = append(actions, tmux.CreateWindow{
				Session: sessionName,
				Name:    w.Name,
				Path:    w.Path,
			})
			actions = append(actions, buildWindowActions(sessionName, w)...)
		}
	}

	return actions
}

func buildWindowActions(sessionName string, window manifest.Window) []tmux.Action {
	var actions []tmux.Action
	target := fmt.Sprintf("%s:%s", sessionName, window.Name)

	if len(window.Panes) > 0 {
		if window.Panes[0].Command != "" {
			actions = append(actions, tmux.SendKeys{
				Target: fmt.Sprintf("%s.0", target),
				Keys:   window.Panes[0].Command,
			})
		}

		for i, pane := range window.Panes[1:] {
			actions = append(actions, tmux.SplitPane{
				Target: target,
				Path:   pane.Path,
			})
			if pane.Command != "" {
				actions = append(actions, tmux.SendKeys{
					Target: fmt.Sprintf("%s.%d", target, i+1),
					Keys:   pane.Command,
				})
			}
		}
	} else if window.Command != "" {
		actions = append(actions, tmux.SendKeys{
			Target: fmt.Sprintf("%s.0", target),
			Keys:   window.Command,
		})
	}

	return actions
}
