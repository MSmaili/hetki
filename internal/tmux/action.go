package tmux

type Action interface {
	Args() []string
}

type CreateSession struct {
	Name string
	Path string
}

func (a CreateSession) Args() []string {
	args := []string{"new-session", "-d", "-s", a.Name}
	if a.Path != "" {
		args = append(args, "-c", a.Path)
	}
	return args
}

type CreateWindow struct {
	Session string
	Name    string
	Path    string
}

func (a CreateWindow) Args() []string {
	args := []string{"new-window", "-t", a.Session, "-n", a.Name}
	if a.Path != "" {
		args = append(args, "-c", a.Path)
	}
	return args
}

type SplitPane struct {
	Target string
	Path   string
}

func (a SplitPane) Args() []string {
	args := []string{"split-window", "-t", a.Target}
	if a.Path != "" {
		args = append(args, "-c", a.Path)
	}
	return args
}

type SendKeys struct {
	Target string
	Keys   string
}

func (a SendKeys) Args() []string {
	return []string{"send-keys", "-t", a.Target, a.Keys, "Enter"}
}

type KillSession struct {
	Name string
}

func (a KillSession) Args() []string {
	return []string{"kill-session", "-t", a.Name}
}

type KillWindow struct {
	Target string
}

func (a KillWindow) Args() []string {
	return []string{"kill-window", "-t", a.Target}
}
