package report

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const maxStateBytes = 2 << 20

// Store follows a stable sidecar lock -> reload -> mutate -> atomic replacement
// protocol. Readers see a single published snapshot. Corrupt/unknown-version
// files are never reset or overwritten. Same-user reports are not authenticated.
type Store struct {
	path   string
	now    func() time.Time
	rename func(string, string) error
}

func NewStore(path string) *Store { return &Store{path: path, now: time.Now, rename: os.Rename} }

func DefaultPath() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if !filepath.IsAbs(base) {
		home, err := os.UserHomeDir()
		if err != nil || !filepath.IsAbs(home) {
			return "", errors.New("cannot determine absolute state directory")
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "hetki", "reports", "state.json"), nil
}

func emptySnapshot() Snapshot {
	return Snapshot{Version: Version, Streams: []Stream{}, Events: []Event{}}
}

// Load does not create files, prune, or imply that stored leases/bindings are
// live. Consumers must check Stream.Live and independently validate membership.
func (s *Store) Load(ctx context.Context) (Snapshot, error) {
	if ctx == nil {
		return Snapshot{}, errors.New("nil report context")
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if s == nil || s.path == "" {
		return Snapshot{}, errors.New("empty report store path")
	}
	f, err := os.OpenFile(s.path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, os.ErrNotExist) {
		return emptySnapshot(), nil
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("open report state: %w", err)
	}
	defer f.Close()
	if err := privateRegular(f, true); err != nil {
		return Snapshot{}, err
	}
	data, err := io.ReadAll(io.LimitReader(f, maxStateBytes+1))
	if err != nil {
		return Snapshot{}, fmt.Errorf("read report state: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if len(data) > maxStateBytes {
		return Snapshot{}, errors.New("report state exceeds byte limit")
	}
	var state Snapshot
	if err := decode(data, &state); err != nil {
		return Snapshot{}, fmt.Errorf("decode report state: %w", err)
	}
	if err := state.validate(); err != nil {
		return Snapshot{}, err
	}
	return state, ctx.Err()
}

// Apply records a generic producer operation. The pane source and its streams
// are reserved for RecordStatus; this is an API boundary, not authentication.
func (s *Store) Apply(ctx context.Context, r Request) (Receipt, error) {
	if err := r.validate(); err != nil {
		return Receipt{}, err
	}
	if r.Source == paneStatusSource {
		return Receipt{}, errors.New("pane source is reserved for RecordStatus")
	}
	return s.mutate(ctx, func(state *Snapshot, now time.Time) (Receipt, error) {
		for _, stream := range state.Streams {
			if stream.ID == r.StreamID && stream.Source == paneStatusSource {
				return Receipt{}, errors.New("pane status streams can only be changed through RecordStatus")
			}
		}
		return state.apply(r, now)
	})
}

func (s *Store) mutate(ctx context.Context, apply func(*Snapshot, time.Time) (Receipt, error)) (Receipt, error) {
	if ctx == nil {
		return Receipt{}, errors.New("nil report context")
	}
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	if s == nil || s.path == "" {
		return Receipt{}, errors.New("empty report store path")
	}
	if err := s.directory(); err != nil {
		return Receipt{}, err
	}
	lock, err := s.lock(ctx)
	if err != nil {
		return Receipt{}, err
	}
	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN); _ = lock.Close() }()
	state, err := s.Load(ctx)
	if err != nil {
		return Receipt{}, err
	}
	now := s.now().UTC()
	state.prune(now)
	receipt, err := apply(&state, now)
	if err != nil {
		return Receipt{}, err
	}
	if receipt.Duplicate {
		receipt.Revision = state.Revision
		return receipt, nil
	}
	if state.Revision == math.MaxUint64 {
		return Receipt{}, errors.New("report revision exhausted")
	}
	state.Revision++
	// Prune closed receipts/history after a successful transition, never live
	// streams to make room. A rejected report does not modify the store.
	state.prune(now)
	if err := state.validate(); err != nil {
		return Receipt{}, err
	}
	if err := s.write(ctx, state); err != nil {
		return Receipt{}, err
	}
	receipt.Revision = state.Revision
	return receipt, nil
}

func (s *Store) directory() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 || !ok || stat.Uid != uint32(os.Geteuid()) {
		return errors.New("report directory must be a private, owned directory (0700)")
	}
	return nil
}

func privateRegular(f *os.File, allowUnlinked bool) error {
	info, err := f.Stat()
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	// A reader may hold the old published inode after atomic replacement
	// unlinks it. That snapshot is still valid; lock inodes must stay linked.
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || !ok || stat.Uid != uint32(os.Geteuid()) || stat.Nlink > 1 || (!allowUnlinked && stat.Nlink != 1) {
		return errors.New("report state/lock must be a private, owned regular file (0600), without hard links")
	}
	return nil
}

func (s *Store) lock(ctx context.Context) (*os.File, error) {
	f, err := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, fmt.Errorf("open report lock: %w", err)
	}
	if err := privateRegular(f, false); err != nil {
		_ = f.Close()
		return nil, err
	}
	for {
		if err := ctx.Err(); err != nil {
			_ = f.Close()
			return nil, err
		}
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			_ = f.Close()
			return nil, err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			_ = f.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (s *Store) write(ctx context.Context, state Snapshot) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode report state: %w", err)
	}
	data = append(data, '\n')
	if len(data) > maxStateBytes {
		return errors.New("report state exceeds byte limit")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".report.tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := s.rename(f.Name(), s.path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(s.path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
