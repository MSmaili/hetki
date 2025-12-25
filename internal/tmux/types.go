package tmux

type Window struct {
	Name  string
	Path  string
	Panes []Pane
}

type Pane struct {
	Path    string
	Command string
}
