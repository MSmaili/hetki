package tmux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Client interface {
	Execute(action Action) error
	ExecuteBatch(actions []Action) error
	Run(args ...string) (string, error)
	Attach(session string) error
}

func RunQuery[T any](c Client, q Query[T]) (T, error) {
	output, err := c.Run(q.Args()...)
	if err != nil {
		var zero T
		return zero, err
	}
	return q.Parse(output)
}

type client struct {
	bin string
}

func New() (Client, error) {
	bin, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux not found in PATH")
	}
	return &client{bin: bin}, nil
}

func (c *client) Run(args ...string) (string, error) {
	cmd := exec.Command(c.bin, args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux %v failed: %v (%s)", args, err, stderr.String())
	}

	return strings.TrimSpace(out.String()), nil
}

func (c *client) Execute(action Action) error {
	_, err := c.Run(action.Args()...)
	return err
}

func (c *client) ExecuteBatch(actions []Action) error {
	if len(actions) == 0 {
		return nil
	}

	var script strings.Builder
	for _, action := range actions {
		script.WriteString(strings.Join(action.Args(), " "))
		script.WriteString("\n")
	}

	cmd := exec.Command(c.bin, "source", "-")
	cmd.Stdin = strings.NewReader(script.String())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux source failed: %v (%s)", err, stderr.String())
	}

	return nil
}

func (c *client) Attach(session string) error {
	cmd := exec.Command(c.bin, "attach-session", "-t", session)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
