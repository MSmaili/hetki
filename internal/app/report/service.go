// Package report coordinates local attachment and generic report persistence.
package report

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MSmaili/hetki/internal/report"
)

const reportingTimeout = 2 * time.Second

// Service depends on attachment resolution, not a specific multiplexer. The
// command supplies the tmux adapter; the report model never imports a backend.
type Service struct {
	ResolveBinding func(context.Context) (report.Binding, error)
	StatePath      func() (string, error)
}

func NewService(resolve func(context.Context) (report.Binding, error)) Service {
	return Service{ResolveBinding: resolve, StatePath: report.DefaultPath}
}

func (s Service) Run(ctx context.Context, status report.Status) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := report.ParseStatus(string(status)); err != nil {
		return err
	}
	if s.ResolveBinding == nil || s.StatePath == nil {
		return errors.New("report service is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, reportingTimeout)
	defer cancel()
	binding, err := s.ResolveBinding(ctx)
	if err != nil {
		return fmt.Errorf("resolve report attachment: %w", err)
	}
	path, err := s.StatePath()
	if err != nil {
		return fmt.Errorf("resolve report state path: %w", err)
	}
	if err := report.NewStore(path).RecordStatus(ctx, binding, status); err != nil {
		return fmt.Errorf("record report status: %w", err)
	}
	return nil
}
