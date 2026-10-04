package report

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testStore(t *testing.T) (*Store, *time.Time) {
	t.Helper()
	store := NewStore(filepath.Join(t.TempDir(), "reports", "state.json"))
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	return store, &now
}

func applyReport(t *testing.T, store *Store, r Request) Receipt {
	t.Helper()
	receipt, err := store.Apply(context.Background(), r)
	require.NoError(t, err)
	return receipt
}

func loadReports(t *testing.T, store *Store) Snapshot {
	t.Helper()
	state, err := store.Load(context.Background())
	require.NoError(t, err)
	return state
}

func TestReportLifecycleSeparatesInputCompletionAndAcknowledgment(t *testing.T) {
	store, now := testStore(t)
	request := openRequest()
	request.PendingInput = []string{"approval-1"}
	opened := applyReport(t, store, request)
	require.EqualValues(t, 1, opened.Sequence)
	require.EqualValues(t, 1, opened.Revision)
	complete := Request{Version: 1, Op: "complete", StreamID: opened.StreamID, Sequence: 2, EventID: "turn-1", Outcome: "finished"}
	receipt := applyReport(t, store, complete)
	require.True(t, receipt.EventRecorded)
	state := loadReports(t, store)
	require.Len(t, state.Streams, 1)
	require.Equal(t, Working, state.Streams[0].Activity)
	require.Equal(t, []string{"approval-1"}, state.Streams[0].PendingInput)
	require.True(t, state.Streams[0].Live(*now))
	require.Len(t, state.Events, 1)
	require.False(t, state.Events[0].Acknowledged)
	ack := Request{Version: 1, Op: "ack", Source: request.Source, Session: request.Session, EventID: "turn-1"}
	applyReport(t, store, ack)
	state = loadReports(t, store)
	require.True(t, state.Events[0].Acknowledged)
	require.Equal(t, []string{"approval-1"}, state.Streams[0].PendingInput)
	require.True(t, applyReport(t, store, ack).Duplicate)
	update := Request{Version: 1, Op: "update", StreamID: opened.StreamID, Sequence: 3, Activity: Idle, PendingInput: []string{}}
	applyReport(t, store, update)
	state = loadReports(t, store)
	require.Empty(t, state.Streams[0].PendingInput)
	require.Equal(t, Idle, state.Streams[0].Activity)
	applyReport(t, store, Request{Version: 1, Op: "close", StreamID: opened.StreamID, Sequence: 4})
	state = loadReports(t, store)
	require.False(t, state.Streams[0].Live(*now))
	require.Len(t, state.Events, 1, "close retains completion history")
	update.Sequence = 5
	_, err := store.Apply(context.Background(), update)
	require.ErrorContains(t, err, "closed or expired")
}

func TestReportActivityAndKeepalivePreservePendingInput(t *testing.T) {
	store, _ := testStore(t)
	r := openRequest()
	r.PendingInput = []string{"approval", "question"}
	opened := applyReport(t, store, r)
	update := Request{Version: 1, Op: "update", StreamID: opened.StreamID, Sequence: 2, Activity: Idle}
	applyReport(t, store, update)
	require.Equal(t, []string{"approval", "question"}, loadReports(t, store).Streams[0].PendingInput)
	update.PendingInput = []string{}
	_, err := store.Apply(context.Background(), update)
	require.ErrorContains(t, err, "conflicting", "omitted and explicit empty sets are not identical retries")
	update.Sequence++
	applyReport(t, store, update)
	require.Empty(t, loadReports(t, store).Streams[0].PendingInput)
	// Explicit clearing survives Request JSON serialization, unlike omitempty.
	data, err := json.Marshal(update)
	require.NoError(t, err)
	decoded, err := Decode(strings.NewReader(string(data)))
	require.NoError(t, err)
	require.NotNil(t, decoded.PendingInput)
	require.Empty(t, decoded.PendingInput)
}

func TestReportOrderingAndIdempotency(t *testing.T) {
	store, now := testStore(t)
	opened := applyReport(t, store, openRequest())
	request := Request{Version: 1, Op: "update", StreamID: opened.StreamID, Sequence: 5, Activity: Working, PendingInput: []string{"q2", "q1"}}
	receipt := applyReport(t, store, request)
	before, err := os.ReadFile(store.path)
	require.NoError(t, err)
	*now = now.Add(time.Second)
	request.PendingInput = []string{"q1", "q2"}
	request.LeaseSeconds = DefaultLeaseSeconds
	duplicate := applyReport(t, store, request)
	require.True(t, duplicate.Duplicate)
	require.Equal(t, receipt.Revision, duplicate.Revision)
	after, err := os.ReadFile(store.path)
	require.NoError(t, err)
	require.Equal(t, before, after, "duplicate does not extend lease or rewrite snapshot")
	request.Sequence = 4
	_, err = store.Apply(context.Background(), request)
	require.ErrorContains(t, err, "outdated")
	request.Sequence, request.Activity = 5, Idle
	_, err = store.Apply(context.Background(), request)
	require.ErrorContains(t, err, "conflicting")
	request.Sequence = 6
	applyReport(t, store, request)
}

func TestReportRegistrationRetriesNeverResetOrReviveStream(t *testing.T) {
	store, now := testStore(t)
	request := openRequest()
	first := applyReport(t, store, request)
	applyReport(t, store, Request{Version: 1, Op: "update", StreamID: first.StreamID, Sequence: 2, Activity: Idle})
	before := loadReports(t, store)
	*now = now.Add(time.Second)
	retry := applyReport(t, store, request)
	require.True(t, retry.Duplicate)
	require.Equal(t, first.StreamID, retry.StreamID)
	require.EqualValues(t, 2, retry.Sequence)
	require.Equal(t, before.Revision, retry.Revision)
	require.Equal(t, before, loadReports(t, store))
	conflict := request
	conflict.Activity = Idle
	_, err := store.Apply(context.Background(), conflict)
	require.ErrorContains(t, err, "conflicting registration")
	*now = now.Add(time.Minute)
	retry = applyReport(t, store, request)
	require.True(t, retry.Duplicate)
	require.False(t, loadReports(t, store).Streams[0].Live(*now))
	_, err = store.Apply(context.Background(), Request{Version: 1, Op: "update", StreamID: retry.StreamID, Sequence: 3, Activity: Working})
	require.ErrorContains(t, err, "expired")
}

func TestReportExpiredOwnerCannotResurrectOrClearNewOwner(t *testing.T) {
	store, now := testStore(t)
	request := openRequest()
	request.PendingInput = []string{"approval"}
	first := applyReport(t, store, request)
	request.RegistrationID = "registration-2"
	_, err := store.Apply(context.Background(), request)
	require.ErrorContains(t, err, "already owns")
	*now = now.Add(DefaultLeaseSeconds * time.Second)
	state := loadReports(t, store)
	require.False(t, state.Streams[0].Live(*now))
	require.Empty(t, state.Events, "lease expiry is never completion")
	update := Request{Version: 1, Op: "update", StreamID: first.StreamID, Sequence: 2, Activity: Idle}
	_, err = store.Apply(context.Background(), update)
	require.ErrorContains(t, err, "expired")
	second := applyReport(t, store, request)
	require.NotEqual(t, first.StreamID, second.StreamID)
	_, err = store.Apply(context.Background(), Request{Version: 1, Op: "close", StreamID: first.StreamID, Sequence: 3})
	require.Error(t, err)
	state = loadReports(t, store)
	require.Len(t, state.Streams, 2)
	require.Equal(t, second.StreamID, state.Streams[0].ID)
	require.True(t, state.Streams[0].Live(*now))
	// Updates retain a full snapshot and renew only the current stream lease.
	*now = now.Add(30 * time.Second)
	applyReport(t, store, Request{Version: 1, Op: "update", StreamID: second.StreamID, Sequence: 2, Activity: Working, PendingInput: []string{"approval"}, LeaseSeconds: 120})
	require.Equal(t, now.Add(120*time.Second), loadReports(t, store).Streams[0].ExpiresAt)
}

func TestReportCompletionDeduplicationAcrossStreams(t *testing.T) {
	store, _ := testStore(t)
	r := openRequest()
	first := applyReport(t, store, r)
	r.Binding.Pane = "%2"
	r.RegistrationID = "registration-2"
	second := applyReport(t, store, r)
	complete := Request{Version: 1, Op: "complete", StreamID: first.StreamID, Sequence: 2, EventID: "turn-1", Outcome: "finished"}
	require.True(t, applyReport(t, store, complete).EventRecorded)
	require.True(t, applyReport(t, store, complete).Duplicate)
	ack := Request{Version: 1, Op: "ack", Source: r.Source, Session: r.Session, EventID: "turn-1"}
	applyReport(t, store, ack)
	complete.StreamID = second.StreamID
	require.False(t, applyReport(t, store, complete).EventRecorded)
	state := loadReports(t, store)
	require.Len(t, state.Events, 1)
	require.True(t, state.Events[0].Acknowledged, "duplicate does not make acknowledged event unread")
	complete.Sequence, complete.Outcome = 3, "failed"
	_, err := store.Apply(context.Background(), complete)
	require.ErrorContains(t, err, "conflicting completion")
	require.EqualValues(t, 2, loadReports(t, store).Streams[1].Sequence)
}

func TestReportRetentionAndCapacity(t *testing.T) {
	store, now := testStore(t)
	opened := applyReport(t, store, openRequest())
	for i := 0; i < maxEvents+2; i++ {
		applyReport(t, store, Request{Version: 1, Op: "complete", StreamID: opened.StreamID, Sequence: uint64(i + 2), EventID: fmt.Sprintf("event-%d", i), Outcome: "finished"})
	}
	state := loadReports(t, store)
	require.Len(t, state.Events, maxEvents)
	require.Equal(t, "event-257", state.Events[0].ID, "newest retained even when timestamps tie")
	for _, event := range state.Events {
		require.NotEqual(t, "event-0", event.ID)
	}
	*now = now.Add(retention + time.Minute)
	applyReport(t, store, openRequest())
	state = loadReports(t, store)
	require.Empty(t, state.Events)
	require.Len(t, state.Streams, 1)
	_, err := store.Apply(context.Background(), Request{Version: 1, Op: "update", StreamID: opened.StreamID, Sequence: 1000, Activity: Idle})
	require.ErrorContains(t, err, "not found")

	// Capacity never evicts a live stream or pending request to admit another.
	for i := 1; i < maxStreams; i++ {
		r := openRequest()
		r.Session = fmt.Sprintf("session-%d", i+1)
		applyReport(t, store, r)
	}
	before, err := os.ReadFile(store.path)
	require.NoError(t, err)
	r := openRequest()
	r.Session = "overflow"
	_, err = store.Apply(context.Background(), r)
	require.ErrorContains(t, err, "capacity")
	after, err := os.ReadFile(store.path)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestReportRetiredReceiptBound(t *testing.T) {
	store, _ := testStore(t)
	var latest Receipt
	for i := 0; i < maxRetired+2; i++ {
		r := openRequest()
		r.Session = fmt.Sprintf("session-%d", i)
		opened := applyReport(t, store, r)
		latest = applyReport(t, store, Request{Version: 1, Op: "close", StreamID: opened.StreamID, Sequence: 2})
	}
	state := loadReports(t, store)
	require.Len(t, state.Streams, maxRetired)
	require.Equal(t, latest.StreamID, state.Streams[0].ID)
	for _, stream := range state.Streams {
		require.NotNil(t, stream.RetiredAt)
	}
}

func TestReportReaderMayKeepReplacedSnapshotInode(t *testing.T) {
	store, _ := testStore(t)
	applyReport(t, store, openRequest())
	f, err := os.Open(store.path)
	require.NoError(t, err)
	defer f.Close()
	r := openRequest()
	r.Session = "another"
	applyReport(t, store, r)
	require.NoError(t, privateRegular(f, true))
	require.Error(t, privateRegular(f, false))
}

func TestReportStorePreservesInvalidFiles(t *testing.T) {
	for name, contents := range map[string]string{
		"corrupt": "not json", "unknown version": `{"version":2,"revision":1,"streams":[],"events":[]}`,
		"missing arrays": `{"version":1,"revision":0}`, "null arrays": `{"version":1,"revision":0,"streams":null,"events":[]}`,
		"missing revision": `{"version":1,"streams":[],"events":[]}`,
		"oversized":        strings.Repeat(" ", maxStateBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			store, _ := testStore(t)
			require.NoError(t, store.directory())
			require.NoError(t, os.WriteFile(store.path, []byte(contents), 0600))
			_, err := store.Apply(context.Background(), openRequest())
			require.Error(t, err)
			got, err := os.ReadFile(store.path)
			require.NoError(t, err)
			require.Equal(t, contents, string(got))
		})
	}
}

func TestReportStoreRefusesUnsafeFiles(t *testing.T) {
	for _, target := range []string{"state", "lock"} {
		for _, kind := range []string{"symlink", "hardlink", "public", "fifo", "directory"} {
			t.Run(target+"/"+kind, func(t *testing.T) {
				store, _ := testStore(t)
				require.NoError(t, store.directory())
				path := store.path
				if target == "lock" {
					path += ".lock"
				}
				other := filepath.Join(t.TempDir(), "unrelated")
				require.NoError(t, os.WriteFile(other, []byte("untouched"), 0600))
				switch kind {
				case "symlink":
					require.NoError(t, os.Symlink(other, path))
				case "hardlink":
					require.NoError(t, os.Link(other, path))
				case "public":
					require.NoError(t, os.WriteFile(path, []byte("public"), 0644))
					require.NoError(t, os.Chmod(path, 0644))
				case "fifo":
					require.NoError(t, syscall.Mkfifo(path, 0600))
				case "directory":
					require.NoError(t, os.Mkdir(path, 0700))
				}
				_, err := store.Apply(context.Background(), openRequest())
				require.Error(t, err)
				got, err := os.ReadFile(other)
				require.NoError(t, err)
				require.Equal(t, "untouched", string(got))
			})
		}
	}
}

func TestReportStoreRejectsPublicOrSymlinkDirectory(t *testing.T) {
	for _, kind := range []string{"public", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			store, _ := testStore(t)
			dir := filepath.Dir(store.path)
			if kind == "public" {
				require.NoError(t, os.Mkdir(dir, 0755))
				require.NoError(t, os.Chmod(dir, 0755))
			} else {
				require.NoError(t, os.Symlink(t.TempDir(), dir))
			}
			_, err := store.Apply(context.Background(), openRequest())
			require.Error(t, err)
			_, err = os.Lstat(store.path)
			require.True(t, os.IsNotExist(err))
		})
	}
}

func TestReportLoadMissingAndCancelledIsNonmutating(t *testing.T) {
	store, _ := testStore(t)
	require.Equal(t, emptySnapshot(), loadReports(t, store))
	_, err := os.Stat(filepath.Dir(store.path))
	require.True(t, os.IsNotExist(err))
	_, err = store.Load(nil)
	require.Error(t, err)
	_, err = store.Apply(nil, openRequest())
	require.Error(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.Load(ctx)
	require.ErrorIs(t, err, context.Canceled)
	var missing *Store
	_, err = missing.Load(context.Background())
	require.Error(t, err)
	_, err = missing.Apply(context.Background(), openRequest())
	require.Error(t, err)
}

func TestReportStoreCancellationAndFailedWrite(t *testing.T) {
	store, _ := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.Apply(ctx, openRequest())
	require.ErrorIs(t, err, context.Canceled)
	_, err = os.Stat(filepath.Dir(store.path))
	require.True(t, os.IsNotExist(err))
	require.NoError(t, store.directory())
	f, err := store.lock(context.Background())
	require.NoError(t, err)
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = store.Apply(ctx, openRequest())
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, f.Close())
	opened := applyReport(t, store, openRequest())
	before, err := os.ReadFile(store.path)
	require.NoError(t, err)
	store.rename = func(string, string) error { return errors.New("rename failure") }
	_, err = store.Apply(context.Background(), Request{Version: 1, Op: "close", StreamID: opened.StreamID, Sequence: 2})
	require.ErrorContains(t, err, "rename failure")
	after, err := os.ReadFile(store.path)
	require.NoError(t, err)
	require.Equal(t, before, after)
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(store.path), ".report.tmp-*"))
	require.NoError(t, err)
	require.Empty(t, paths)
}

func TestReportConcurrentWritersAndSnapshotReaders(t *testing.T) {
	store, _ := testStore(t)
	const writers = 24
	errors := make(chan error, writers)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := openRequest()
			r.Session = fmt.Sprintf("session-%d", i)
			_, err := store.Apply(context.Background(), r)
			if err == nil {
				_, err = store.Load(context.Background())
			}
			errors <- err
		}(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	state := loadReports(t, store)
	require.Len(t, state.Streams, writers)
	require.EqualValues(t, writers, state.Revision)
	for _, path := range []string{store.path, store.path + ".lock"} {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
	info, err := os.Stat(filepath.Dir(store.path))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

func TestReportProcessWriter(t *testing.T) {
	path, session := os.Getenv("HETKI_REPORT_TEST_PATH"), os.Getenv("HETKI_REPORT_TEST_SESSION")
	if path == "" || session == "" {
		t.Skip("child-process fixture")
	}
	r := openRequest()
	r.Session = session
	applyReport(t, NewStore(path), r)
}

func TestReportCrossProcessWriters(t *testing.T) {
	store, _ := testStore(t)
	const count = 8
	commands := make([]*exec.Cmd, count)
	for i := range commands {
		commands[i] = exec.Command(os.Args[0], "-test.run=^TestReportProcessWriter$")
		commands[i].Env = append(os.Environ(), "HETKI_REPORT_TEST_PATH="+store.path, fmt.Sprintf("HETKI_REPORT_TEST_SESSION=process-%d", i))
		require.NoError(t, commands[i].Start())
	}
	for _, command := range commands {
		require.NoError(t, command.Wait())
	}
	require.Len(t, loadReports(t, store).Streams, count)
}

func TestReportDefaultPathAndRevisionExhaustion(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	path, err := DefaultPath()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(base, "hetki", "reports", "state.json"), path)
	t.Setenv("XDG_STATE_HOME", "relative")
	t.Setenv("HOME", base)
	path, err = DefaultPath()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(base, ".local", "state", "hetki", "reports", "state.json"), path)
	store, _ := testStore(t)
	opened := applyReport(t, store, openRequest())
	state := loadReports(t, store)
	state.Revision = math.MaxUint64
	require.NoError(t, store.write(context.Background(), state))
	_, err = store.Apply(context.Background(), Request{Version: 1, Op: "close", StreamID: opened.StreamID, Sequence: 2})
	require.ErrorContains(t, err, "revision exhausted")
}

func TestReportSnapshotValidation(t *testing.T) {
	store, _ := testStore(t)
	opened := applyReport(t, store, openRequest())
	applyReport(t, store, Request{Version: 1, Op: "complete", StreamID: opened.StreamID, Sequence: 2, EventID: "e1", Outcome: "finished"})
	data, err := os.ReadFile(store.path)
	require.NoError(t, err)
	for _, mutate := range []func(*Snapshot){
		func(s *Snapshot) { s.Revision = 0 },
		func(s *Snapshot) { s.Streams[0].Sequence = 0 },
		func(s *Snapshot) { s.Streams[0].LastDigest = "invalid" },
		func(s *Snapshot) { s.Streams[0].UpdatedAt = time.Time{} },
		func(s *Snapshot) { s.Streams[0].ExpiresAt = time.Time{} },
		func(s *Snapshot) { s.Streams[0].Activity = "done" },
		func(s *Snapshot) { s.Streams[0].PendingInput = nil },
		func(s *Snapshot) { s.Streams[0].RetiredAt = &time.Time{} },
		func(s *Snapshot) { s.Streams = append(s.Streams, s.Streams[0]) },
		func(s *Snapshot) { s.Events[0].Outcome = "invalid" },
		func(s *Snapshot) { s.Events[0].RecordedAt = time.Time{} },
		func(s *Snapshot) { s.Events = append(s.Events, s.Events[0]) },
	} {
		var snapshot Snapshot
		require.NoError(t, json.Unmarshal(data, &snapshot))
		mutate(&snapshot)
		require.Error(t, snapshot.validate())
	}
}
