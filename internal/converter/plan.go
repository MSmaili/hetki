package converter

import (
	"github.com/MSmaili/tms/internal/plan"
	"github.com/MSmaili/tms/internal/tmux"
)

func PlanActionsToTmux(actions []plan.Action) []tmux.Action {
	result := make([]tmux.Action, 0, len(actions))
	for _, a := range actions {
		if ta := planActionToTmux(a); ta != nil {
			result = append(result, ta)
		}
	}
	return result
}

func planActionToTmux(a plan.Action) tmux.Action {
	switch action := a.(type) {
	case plan.CreateSessionAction:
		return tmux.CreateSession{Name: action.Name, WindowName: action.WindowName, Path: action.Path}
	case plan.CreateWindowAction:
		return tmux.CreateWindow{Session: action.Session, Name: action.Name, Path: action.Path}
	case plan.SplitPaneAction:
		return tmux.SplitPane{Target: action.Target, Path: action.Path}
	case plan.SendKeysAction:
		return tmux.SendKeys{Target: action.Target, Keys: action.Command}
	case plan.SelectLayoutAction:
		return tmux.SelectLayout{Target: action.Target, Layout: action.Layout}
	case plan.ZoomPaneAction:
		return tmux.ZoomPane{Target: action.Target}
	case plan.KillSessionAction:
		return tmux.KillSession{Name: action.Name}
	case plan.KillWindowAction:
		return tmux.KillWindow{Target: action.Target}
	default:
		return nil
	}
}
