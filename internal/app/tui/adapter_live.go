package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/MSmaili/hetki/internal/backend"
	"github.com/MSmaili/hetki/internal/tui/contracts"
)

type LiveAdapter struct {
	DetectBackend func(...string) (backend.Backend, error)
}

func NewLiveAdapter(detectBackend func(...string) (backend.Backend, error)) LiveAdapter {
	return LiveAdapter{DetectBackend: detectBackend}
}

func (a LiveAdapter) Load(ctx context.Context) (contracts.Snapshot, error) {
	return a.snapshotFromBackend(ctx)
}

func (a LiveAdapter) Refresh(ctx context.Context) (contracts.Snapshot, error) {
	return a.snapshotFromBackend(ctx)
}

func (a LiveAdapter) Execute(ctx context.Context, intent contracts.Intent) (contracts.ActionResult, error) {
	b, err := a.detectBackend()
	if err != nil {
		return contracts.ActionResult{}, fmt.Errorf("failed to detect backend: %w", err)
	}

	switch intent.Type {
	case contracts.IntentSwitch:
		target := strings.TrimSpace(intent.Target)
		if target == "" {
			return contracts.ActionResult{}, fmt.Errorf("empty switch target")
		}
		if err := b.Switch(target); err != nil {
			return contracts.ActionResult{}, fmt.Errorf("switch to %q: %w", target, err)
		}
		return contracts.ActionResult{Message: "switched to " + target, NeedsRefresh: true}, nil
	default:
		return contracts.ActionResult{}, fmt.Errorf("intent %q is not implemented yet", intent.Type)
	}
}

func (a LiveAdapter) snapshotFromBackend(ctx context.Context) (contracts.Snapshot, error) {
	b, err := a.detectBackend()
	if err != nil {
		return contracts.Snapshot{}, fmt.Errorf("failed to detect backend: %w", err)
	}

	result, err := b.QueryState()
	if err != nil {
		return contracts.Snapshot{}, fmt.Errorf("failed to query sessions: %w", err)
	}

	activeTarget := buildActiveTarget(result.Active)
	activeWorkspace := workspacePathForSession(result.Sessions, result.Active.Session)
	if activeWorkspace == "" {
		activeWorkspace = "unmanaged"
	}
	snapshot := contracts.Snapshot{
		Nodes:        make([]contracts.Node, 0, len(result.Sessions)),
		ActiveNodeID: activeTarget,
		ContextBars: map[string]string{
			"source":    "live",
			"active":    activeTarget,
			"workspace": activeWorkspace,
		},
		Capabilities: map[contracts.Capability]bool{
			contracts.CapabilityRefresh: true,
			contracts.CapabilitySwitch:  true,
		},
	}

	for _, sess := range result.Sessions {
		sessionNode := contracts.Node{
			ID:     "session:" + sess.Name,
			Kind:   contracts.NodeKindSession,
			Label:  sess.Name,
			Target: sess.Name,
			Active: result.Active.Session == sess.Name,
		}

		for _, win := range sess.Windows {
			windowTarget := fmt.Sprintf("%s:%d", sess.Name, win.Index)
			windowLabel := fmt.Sprintf("%d", win.Index)
			if win.Name != "" {
				windowLabel = fmt.Sprintf("%d %s", win.Index, win.Name)
			}
			isActiveWindow := result.Active.Session == sess.Name && result.Active.WindowIndex == win.Index
			windowNode := contracts.Node{
				ID:       fmt.Sprintf("window:%s:%d", sess.Name, win.Index),
				ParentID: sessionNode.ID,
				Kind:     contracts.NodeKindWindow,
				Label:    windowLabel,
				Target:   windowTarget,
				Active:   isActiveWindow,
			}

			sessionNode.Children = append(sessionNode.Children, windowNode)
		}

		snapshot.Nodes = append(snapshot.Nodes, sessionNode)
	}

	return snapshot, nil
}

func (a LiveAdapter) detectBackend() (backend.Backend, error) {
	if a.DetectBackend != nil {
		return a.DetectBackend()
	}
	return backend.Detect()
}

func buildActiveTarget(active backend.ActiveContext) string {
	if active.Session == "" {
		return "none"
	}
	if active.Window == "" {
		return active.Session
	}
	windowRef := active.Window
	if active.WindowIndex >= 0 {
		windowRef = fmt.Sprintf("%d", active.WindowIndex)
	}
	if active.Pane >= 0 {
		return fmt.Sprintf("%s:%s.%d", active.Session, windowRef, active.Pane)
	}
	return fmt.Sprintf("%s:%s", active.Session, windowRef)
}

func workspacePathForSession(sessions []backend.Session, sessionName string) string {
	for _, sess := range sessions {
		if sess.Name == sessionName {
			return sess.WorkspacePath
		}
	}
	return ""
}
