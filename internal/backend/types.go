package backend

type StateResult struct {
	Sessions []Session
	Active   ActiveContext
}

type Session struct {
	Name          string
	WorkspacePath string
	Windows       []Window
}

type Window struct {
	Name   string
	Index  int
	Path   string
	Layout string
	Panes  []Pane
}

type Pane struct {
	Index   int
	Path    string
	Command string
	Zoom    bool
}

type ActiveContext struct {
	Session     string
	Window      string
	WindowIndex int
	Pane        int
	Path        string
}
