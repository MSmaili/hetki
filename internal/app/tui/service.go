package tui

import (
	"context"
	"errors"
	"fmt"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/MSmaili/hetki/internal/backend"
	ui "github.com/MSmaili/hetki/internal/tui"
	"github.com/MSmaili/hetki/internal/tui/list"
)

type Driver interface {
	Load(context.Context) (list.Snapshot, error)
	Execute(context.Context, ui.ActionRequest) (ui.ActionResult, error)
	Preview(context.Context, list.ItemID) (string, error)
	Navigate(context.Context, ui.BackendTarget) error
}

type RunUIFunc func(context.Context, list.Snapshot, ui.KeyMap, ui.StartMode, ui.DispatchFunc, ui.PreviewOptions) (ui.BackendTarget, error)

type Service struct {
	Driver       Driver
	Keys         ui.KeyMap
	StartMode    ui.StartMode
	PreviewWidth int
	RunUI        RunUIFunc
}

func NewService(detectBackend func(...string) (backend.Backend, error)) Service {
	return Service{
		Driver:    NewLiveAdapter(detectBackend),
		Keys:      ui.DefaultKeyMap(),
		StartMode: ui.DefaultStartMode(),
		RunUI:     ui.Run,
	}
}

func (s Service) Run(ctx context.Context) error {
	if s.Driver == nil {
		return fmt.Errorf("tui driver is not configured")
	}
	runUI := s.RunUI
	if runUI == nil {
		runUI = ui.Run
	}
	keys := s.Keys
	if keys.IsZero() {
		keys = ui.DefaultKeyMap()
	}
	startMode := s.StartMode
	if startMode == "" {
		startMode = ui.DefaultStartMode()
	}

	width := s.PreviewWidth
	if width == 0 {
		width = ui.DefaultPreviewWidth
	}
	if width < 1 || width > 99 {
		return fmt.Errorf("preview width must be between 1 and 99 percent")
	}

	effectsCtx, cancelEffects := context.WithCancel(ctx)
	defer cancelEffects()
	var effects sync.WaitGroup
	var effectsMu sync.Mutex

	initial, err := s.Driver.Load(effectsCtx)
	if err != nil {
		return err
	}

	beginEffect := func() error {
		effectsMu.Lock()
		defer effectsMu.Unlock()
		if err := effectsCtx.Err(); err != nil {
			return err
		}
		effects.Add(1)
		return nil
	}
	dispatch := func(request ui.ActionRequest) (ui.ActionResult, error) {
		if err := beginEffect(); err != nil {
			return ui.ActionResult{}, err
		}
		defer effects.Done()
		return s.Driver.Execute(effectsCtx, request)
	}
	preview := func(ctx context.Context, id list.ItemID) (string, error) {
		if err := beginEffect(); err != nil {
			return "", err
		}
		defer effects.Done()
		if err := ctx.Err(); err != nil {
			return "", err
		}
		readCtx, cancel := context.WithCancel(effectsCtx)
		defer cancel()
		stop := context.AfterFunc(ctx, cancel)
		defer stop()
		return s.Driver.Preview(readCtx, id)
	}

	navigation, err := runUI(effectsCtx, initial, keys, startMode, dispatch, ui.PreviewOptions{Width: width, Read: preview})
	effectsMu.Lock()
	cancelEffects()
	effectsMu.Unlock()
	effects.Wait()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if errors.Is(err, tea.ErrProgramKilled) || errors.Is(err, tea.ErrInterrupted) {
			return nil
		}
		return err
	}
	if navigation != "" {
		return s.Driver.Navigate(ctx, navigation)
	}
	return nil
}
