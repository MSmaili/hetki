package save

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MSmaili/hetki/internal/backend"
	"github.com/MSmaili/hetki/internal/manifest"
)

type Options struct {
	Path  string
	Name  string
	Local bool
	All   bool
}

type Service struct {
	DetectBackend func(...string) (backend.Backend, error)
}

func NewService(detectBackend func(...string) (backend.Backend, error)) Service {
	return Service{DetectBackend: detectBackend}
}

func (s Service) Run(opts Options) (string, error) {
	if err := validateOptions(opts); err != nil {
		return "", err
	}

	b, err := s.detectBackend()
	if err != nil {
		return "", fmt.Errorf("failed to detect backend: %w\nHint: Make sure a supported multiplexer is running", err)
	}

	sessions, err := getTargetSessions(b, opts.All)
	if err != nil {
		return "", err
	}

	outputPath, err := determineSavePath(opts)
	if err != nil {
		return "", err
	}

	return saveWorkspace(sessions, outputPath, opts.All)
}

func (s Service) detectBackend() (backend.Backend, error) {
	if s.DetectBackend != nil {
		return s.DetectBackend()
	}
	return backend.Detect()
}

func validateOptions(opts Options) error {
	if opts.Path != "" && opts.Name != "" {
		return fmt.Errorf("cannot use both -p and -n flags\nUse either: muxie save -p <path> OR muxie save -n <name>")
	}
	return nil
}

func getTargetSessions(b backend.Backend, saveAll bool) ([]backend.Session, error) {
	result, err := b.QueryState()
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}

	if len(result.Sessions) == 0 {
		return nil, fmt.Errorf("no sessions found\nHint: Create a session first")
	}

	if saveAll {
		return result.Sessions, nil
	}

	return findCurrentSession(result)
}

func findCurrentSession(result backend.StateResult) ([]backend.Session, error) {
	if result.Active.Session == "" {
		return nil, fmt.Errorf("not in a session\nHint: Run this command from inside a multiplexer session, or use --all with -p/-n/.")
	}

	for _, s := range result.Sessions {
		if s.Name == result.Active.Session {
			return []backend.Session{s}, nil
		}
	}

	return nil, fmt.Errorf("session %q not found", result.Active.Session)
}

func determineSavePath(opts Options) (string, error) {
	if opts.Path != "" {
		return opts.Path, nil
	}

	resolver := manifest.NewResolver()

	if opts.Name != "" {
		return resolver.NamedPath(opts.Name)
	}

	if opts.Local {
		return resolver.LocalPath()
	}

	if opts.All {
		return "", fmt.Errorf("--all requires a destination\nUse: muxie save --all -p <path>, muxie save --all -n <name>, or muxie save --all .")
	}

	return "", fmt.Errorf("no save target specified\nHint: Use -p <path>, -n <name>, or . to specify where to save")
}

func saveWorkspace(sessions []backend.Session, outputPath string, saveAll bool) (string, error) {
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}

	workspace := convertToWorkspace(sessions)

	if !saveAll {
		loader := manifest.NewFileLoader(absPath)
		if existing, err := loader.Load(); err == nil {
			workspace = mergeWorkspaces(existing, workspace)
		}
	}

	if err := manifest.Write(workspace, absPath); err != nil {
		return "", fmt.Errorf("writing workspace: %w", err)
	}

	return absPath, nil
}

func mergeWorkspaces(existing, new *manifest.Workspace) *manifest.Workspace {
	seen := make(map[string]int, len(existing.Sessions))
	for i, sess := range existing.Sessions {
		seen[sess.Name] = i
	}

	for _, sess := range new.Sessions {
		if idx, ok := seen[sess.Name]; ok {
			existing.Sessions[idx] = sess
		} else {
			existing.Sessions = append(existing.Sessions, sess)
		}
	}
	return existing
}

func convertToWorkspace(sessions []backend.Session) *manifest.Workspace {
	ws := &manifest.Workspace{Sessions: make([]manifest.Session, len(sessions))}

	for i, sess := range sessions {
		ws.Sessions[i] = manifest.Session{
			Name:    sess.Name,
			Windows: convertWindows(sess.Windows),
		}
	}

	return ws
}

func convertWindows(windows []backend.Window) []manifest.Window {
	result := make([]manifest.Window, len(windows))
	for i, w := range windows {
		result[i] = manifest.Window{
			Name: w.Name,
			Path: contractHomePath(w.Path),
		}
		if len(w.Panes) > 1 {
			result[i].Panes = convertPanes(w.Panes)
		}
	}
	return result
}

func contractHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + strings.TrimPrefix(path, home)
	}
	if path == home {
		return "~"
	}
	return path
}

func convertPanes(panes []backend.Pane) []manifest.Pane {
	result := make([]manifest.Pane, len(panes))
	for i, p := range panes {
		result[i] = manifest.Pane{Path: contractHomePath(p.Path)}
	}
	return result
}
