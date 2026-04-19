package tmux

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MSmaili/hetki/internal/backend"
)

type TmuxBackend struct {
	client          Client
	paneBaseIndex   int
	windowBaseIndex int
}

func init() {
	backend.Register("tmux", func() (backend.Backend, error) {
		return NewBackend()
	})
}

func NewBackend() (*TmuxBackend, error) {
	c, err := New()
	if err != nil {
		return nil, err
	}
	return &TmuxBackend{client: c}, nil
}

func (b *TmuxBackend) Name() string {
	return "tmux"
}

func (b *TmuxBackend) QueryState() (backend.StateResult, error) {
	result, err := RunQuery(b.client, LoadStateQuery{})

	b.paneBaseIndex = result.PaneBaseIndex
	b.windowBaseIndex = result.WindowBaseIndex

	if err != nil {
		return backend.StateResult{}, err
	}

	sessions := make([]backend.Session, len(result.Sessions))
	for i, s := range result.Sessions {
		windows := make([]backend.Window, len(s.Windows))
		for j, w := range s.Windows {
			panes := make([]backend.Pane, len(w.Panes))
			for k, p := range w.Panes {
				panes[k] = backend.Pane{
					Index:   k,
					Path:    p.Path,
					Command: p.Command,
					Zoom:    p.Zoom,
				}
			}
			windows[j] = backend.Window{
				Name:   w.Name,
				Index:  w.Index,
				Path:   w.Path,
				Layout: w.Layout,
				Panes:  panes,
			}
		}
		sessions[i] = backend.Session{
			Name:          s.Name,
			WorkspacePath: s.WorkspacePath,
			Windows:       windows,
		}
	}

	return backend.StateResult{
		Sessions: sessions,
		Active: backend.ActiveContext{
			Session:     result.Active.Session,
			Window:      result.Active.Window,
			WindowIndex: result.Active.WindowIndex,
			Pane:        result.Active.Pane,
			Path:        result.Active.Path,
		},
	}, nil
}

func (b *TmuxBackend) Apply(actions []backend.Action) error {
	tmuxActions := b.mapActions(actions)
	return b.client.ExecuteBatch(tmuxActions)
}

func (b *TmuxBackend) DryRun(actions []backend.Action) []string {
	tmuxActions := b.mapActions(actions)
	lines := make([]string, len(tmuxActions))
	for i, a := range tmuxActions {
		lines[i] = "tmux " + strings.Join(a.Args(), " ")
	}
	return lines
}

func (b *TmuxBackend) Attach(session string) error {
	return b.switchTo(session)
}

func (b *TmuxBackend) Switch(target string) error {
	session, rest, hasWindow := strings.Cut(target, ":")
	if !hasWindow {
		return b.switchTo(target)
	}

	state, err := RunQuery(b.client, LoadStateQuery{})
	if err != nil {
		return err
	}

	window, paneStr, hasPane := strings.Cut(rest, ".")

	winIndex, err := resolveWindowIndex(state.Sessions, session, window)
	if err != nil {
		return err
	}

	resolved := fmt.Sprintf("%s:%d", session, winIndex)
	if hasPane {
		var pane int
		fmt.Sscanf(paneStr, "%d", &pane)
		resolved = fmt.Sprintf("%s.%d", resolved, pane+state.PaneBaseIndex)
	}

	return b.switchTo(resolved)
}

func (b *TmuxBackend) switchTo(target string) error {
	if isInsideTmux() {
		return b.client.Execute(SwitchClient{Target: target})
	}
	return b.client.Execute(AttachSession{Target: target})
}

func isInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

func findWindowIndex(sessions []Session, sessionName, windowName string) (int, error) {
	for _, s := range sessions {
		if s.Name != sessionName {
			continue
		}
		for _, w := range s.Windows {
			if w.Name == windowName {
				return w.Index, nil
			}
		}
		return 0, fmt.Errorf("window %q not found in session %q", windowName, sessionName)
	}
	return 0, fmt.Errorf("session %q not found", sessionName)
}

func resolveWindowIndex(sessions []Session, sessionName, window string) (int, error) {
	if idx, err := strconv.Atoi(strings.TrimSpace(window)); err == nil {
		if windowIndexExists(sessions, sessionName, idx) {
			return idx, nil
		}
		return 0, fmt.Errorf("window index %d not found in session %q", idx, sessionName)
	}
	return findWindowIndex(sessions, sessionName, window)
}

func windowIndexExists(sessions []Session, sessionName string, index int) bool {
	for _, s := range sessions {
		if s.Name != sessionName {
			continue
		}
		for _, w := range s.Windows {
			if w.Index == index {
				return true
			}
		}
		return false
	}
	return false
}

func (b *TmuxBackend) mapActions(actions []backend.Action) []Action {
	result := make([]Action, 0, len(actions))
	windowIndex := make(map[string]int)
	for _, a := range actions {
		if ta := b.mapAction(a, windowIndex); ta != nil {
			result = append(result, ta)
		}
	}
	return result
}

func (b *TmuxBackend) mapAction(a backend.Action, windowIndex map[string]int) Action {
	base := b.windowBaseIndex
	switch action := a.(type) {
	case backend.CreateSessionAction:
		windowIndex[action.Name] = base
		return CreateSession{Name: action.Name, WindowName: action.WindowName, Path: action.Path}
	case backend.CreateWindowAction:
		windowIndex[action.Session]++
		return CreateWindow{Session: action.Session, Name: action.Name, Path: action.Path}
	case backend.RenameSessionAction:
		return RenameSession{Target: action.Current, Name: action.New}
	case backend.RenameWindowAction:
		return RenameWindow{Target: fmt.Sprintf("%s:%s", action.Session, action.Window), Name: action.New}
	case backend.SplitPaneAction:
		return SplitPane{Target: fmt.Sprintf("%s:%d", action.Session, windowIndex[action.Session]), Path: action.Path}
	case backend.SendKeysAction:
		return SendKeys{Target: fmt.Sprintf("%s:%d.%d", action.Session, windowIndex[action.Session], action.Pane+b.paneBaseIndex), Keys: action.Command}
	case backend.SelectLayoutAction:
		return SelectLayout{Target: fmt.Sprintf("%s:%d", action.Session, windowIndex[action.Session]), Layout: action.Layout}
	case backend.ZoomPaneAction:
		return ZoomPane{Target: fmt.Sprintf("%s:%d.%d", action.Session, windowIndex[action.Session], action.Pane+b.paneBaseIndex)}
	case backend.KillSessionAction:
		return KillSession{Name: action.Name}
	case backend.KillWindowAction:
		return KillWindow{Target: fmt.Sprintf("%s:%s", action.Session, action.Window)}
	case backend.SetSessionOptionAction:
		return SetSessionOption{Session: action.Session, Key: action.Key, Value: action.Value}
	default:
		return nil
	}
}
