package tmux

import "strings"

type Query[T any] interface {
	Args() []string
	Parse(output string) (T, error)
}

type Session struct {
	Name    string
	Windows []Window
}

type ListSessionsQuery struct{}

func (q ListSessionsQuery) Args() []string {
	return []string{"list-sessions", "-F", "#{session_name}"}
}

func (q ListSessionsQuery) Parse(output string) ([]Session, error) {
	if output == "" {
		return []Session{}, nil
	}

	lines := strings.Split(output, "\n")
	sessions := make([]Session, 0, len(lines))

	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name != "" {
			sessions = append(sessions, Session{Name: name})
		}
	}

	return sessions, nil
}

type ListWindowsQuery struct {
	Session string
}

func (q ListWindowsQuery) Args() []string {
	return []string{"list-windows", "-t", q.Session, "-F", "#{window_name}|#{pane_current_path}"}
}

func (q ListWindowsQuery) Parse(output string) ([]Window, error) {
	if output == "" {
		return []Window{}, nil
	}

	lines := strings.Split(output, "\n")
	windows := make([]Window, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 2 {
			windows = append(windows, Window{
				Name: parts[0],
				Path: parts[1],
			})
		}
	}

	return windows, nil
}

type ListPanesQuery struct {
	Target string // session:window
}

func (q ListPanesQuery) Args() []string {
	return []string{"list-panes", "-t", q.Target, "-F", "#{pane_current_path}|#{pane_current_command}"}
}

func (q ListPanesQuery) Parse(output string) ([]Pane, error) {
	if output == "" {
		return []Pane{}, nil
	}

	lines := strings.Split(output, "\n")
	panes := make([]Pane, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 2 {
			panes = append(panes, Pane{
				Path:    parts[0],
				Command: parts[1],
			})
		}
	}

	return panes, nil
}
