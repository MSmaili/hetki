package tmux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Client interface {
	Run(args ...string) (string, error)
	Execute(action Action) error
	ExecuteBatch(actions []Action) error
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

	err := cmd.Run()
	output := strings.TrimSpace(out.String())

	if err != nil {
		return output, fmt.Errorf("tmux %v failed: %v (%s)", args, err, stderr.String())
	}

	return output, nil
}

func (c *client) Execute(action Action) error {
	cmd := exec.Command(c.bin, action.Args()...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if s := strings.TrimSpace(stderr.String()); s != "" {
			return fmt.Errorf("%s", s)
		}
		return err
	}
	return nil
}

func (c *client) ExecuteBatch(actions []Action) error {
	if len(actions) == 0 {
		return nil
	}
	return c.executeBatch(actions)
}

func (c *client) executeBatch(actions []Action) error {
	cmd := exec.Command(c.bin, buildBatchArgs(actions)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux batch failed: %v (%s)", err, stderr.String())
	}
	return nil
}

func buildBatchArgs(actions []Action) []string {
	args := make([]string, 0, len(actions)*4)
	for i, action := range actions {
		if i > 0 {
			args = append(args, ";")
		}
		args = append(args, action.Args()...)
	}
	return args
}
