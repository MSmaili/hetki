package tui

import (
	"context"
	"fmt"

	"github.com/MSmaili/hetki/internal/backend"
	"github.com/MSmaili/hetki/internal/tui/contracts"
	"github.com/MSmaili/hetki/internal/tui/core"
)

type Provider interface {
	Load(ctx context.Context) (contracts.Snapshot, error)
	Refresh(ctx context.Context) (contracts.Snapshot, error)
}

type Executor interface {
	Execute(ctx context.Context, intent contracts.Intent) (contracts.ActionResult, error)
}

type Service struct {
	Provider Provider
	Executor Executor
	RunUI    func(initial contracts.Snapshot, dispatch core.DispatchFunc) error
}

func NewService(detectBackend func(...string) (backend.Backend, error)) Service {
	live := NewLiveAdapter(detectBackend)
	return Service{
		Provider: live,
		Executor: live,
		RunUI:    core.Run,
	}
}

func (s Service) Run(ctx context.Context) error {
	if s.Provider == nil {
		return fmt.Errorf("tui provider is not configured")
	}
	if s.Executor == nil {
		return fmt.Errorf("tui executor is not configured")
	}
	runUI := s.RunUI
	if runUI == nil {
		runUI = core.Run
	}

	initial, err := s.Provider.Load(ctx)
	if err != nil {
		return err
	}

	dispatch := func(ctx context.Context, intent contracts.Intent) (contracts.ActionResult, error) {
		if intent.Type == contracts.IntentRefresh {
			snapshot, err := s.Provider.Refresh(ctx)
			if err != nil {
				return contracts.ActionResult{}, err
			}
			return contracts.ActionResult{Message: "refreshed", Snapshot: &snapshot}, nil
		}

		result, err := s.Executor.Execute(ctx, intent)
		if err != nil {
			return contracts.ActionResult{}, err
		}

		if result.NeedsRefresh && result.Snapshot == nil {
			snapshot, err := s.Provider.Refresh(ctx)
			if err != nil {
				return contracts.ActionResult{}, err
			}
			result.Snapshot = &snapshot
		}

		return result, nil
	}

	return runUI(initial, dispatch)
}
