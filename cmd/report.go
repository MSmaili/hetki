package cmd

import (
	"context"

	appreport "github.com/MSmaili/hetki/internal/app/report"
	"github.com/MSmaili/hetki/internal/backend/tmux"
	"github.com/MSmaili/hetki/internal/report"
	"github.com/spf13/cobra"
)

func newReportCommand() *cobra.Command {
	service := appreport.NewService(func(ctx context.Context) (report.Binding, error) {
		attachment, err := tmux.ResolveReportContext(ctx)
		if err != nil {
			return report.Binding{}, err
		}
		return report.Binding{Server: attachment.Server, Pane: attachment.Pane, Generation: attachment.Generation}, nil
	})
	return newReportCommandWithRunner(service.Run)
}

// Dependencies are per-command, so tests do not mutate production globals.
func newReportCommandWithRunner(run func(context.Context, report.Status) error) *cobra.Command {
	var value string
	command := &cobra.Command{
		Use:           "report --status <status>",
		Short:         "Report current work status from inside a tmux pane",
		SilenceUsage:  true,
		SilenceErrors: true,
		Long: `Report explicit current status for the invoking tmux pane.

Statuses: working, needs-input, idle, unknown.
Pane/server identity, reporting lifetimes and ordering are managed internally.
The command does not read JSON, require a session ID, or select an active pane.
It must run inside the local tmux terminal process, not a shared agent server.

Status is not a completion event. No daemon, notifications, agent plugin or
TUI status display is started. See docs/reporting.md for lifecycle limits.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, err := report.ParseStatus(value)
			if err != nil {
				return err
			}
			return run(cmd.Context(), status)
		},
	}
	command.Flags().StringVar(&value, "status", "", "Current status: working, needs-input, idle, unknown")
	_ = command.MarkFlagRequired("status")
	_ = command.RegisterFlagCompletionFunc("status", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return report.Statuses(), cobra.ShellCompDirectiveNoFileComp
	})
	return command
}

func init() { rootCmd.AddCommand(newReportCommand()) }
