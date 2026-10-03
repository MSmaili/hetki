package tmux

type Window struct {
	ID     string
	Name   string
	Index  int
	Path   string
	Layout string
	Active bool
	Panes  []Pane
}

type Pane struct {
	ID      string
	Index   int
	Path    string
	Command string
	PID     int
	Dead    bool
	Zoom    bool
	Active  bool
}
