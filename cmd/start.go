package cmd

import (
	"fmt"
	"os"

	"github.com/MSmaili/tmx/internal/domain"
	"github.com/MSmaili/tmx/internal/manifest"
	"github.com/MSmaili/tmx/internal/tmux"
)

func Start(args []string) {
	if len(args) > 1 {
		fmt.Println("You should give max one session")
		os.Exit(1)
	}

	c := manifest.NewFileLoader("~/.tmux-sessions-2.json")

	sessions, err := c.Load()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	tmx, err := tmux.New()

	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	f, err := createSessions(sessions, tmx)

	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	err = tmx.Attach(f)

	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func createSessions(c *manifest.Config, tmx *tmux.TmuxClient) (string, error) {
	var defaultSession string
	for session, windows := range c.Sessions {
		if len(defaultSession) == 0 {
			defaultSession = session
		}

		var window domain.Window
		if len(windows) > 0 {
			window = windows[0]
		}

		err := tmx.CreateSession(session, &window)
		if err != nil {
			return "", err
		}

		for i := 1; i < len(windows); i++ {
			wo := windows[i]
			fmt.Println(wo.Name)
			err := tmx.CreateWindow(session, wo.Name, wo)
			if err != nil {
				return "", err
			}
		}

	}

	return defaultSession, nil
}
