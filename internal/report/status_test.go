package report

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSimpleStatusTransitionsAndHiddenRegistration(t *testing.T) {
	store, now := testStore(t)
	binding := *openRequest().Binding
	for _, status := range []Status{StatusWorking, StatusNeedsInput, StatusIdle, StatusUnknown} {
		require.NoError(t, store.RecordStatus(context.Background(), binding, status))
		state := loadReports(t, store)
		require.Len(t, state.Streams, 1)
		require.Empty(t, state.Events)
		stream := state.Streams[0]
		require.True(t, stream.Live(*now))
		if status == StatusNeedsInput {
			require.Equal(t, []string{"reported-input"}, stream.PendingInput)
		} else {
			require.Equal(t, Activity(status), stream.Activity)
			require.Empty(t, stream.PendingInput)
		}
	}
	first := loadReports(t, store).Streams[0].ID
	*now = now.Add(DefaultLeaseSeconds * time.Second)
	require.NoError(t, store.RecordStatus(context.Background(), binding, StatusWorking))
	state := loadReports(t, store)
	require.Len(t, state.Streams, 2)
	require.NotEqual(t, first, state.Streams[0].ID)
	require.True(t, state.Streams[0].Live(*now))
}

func TestSimpleStatusDoesNotClearOtherProducerInputOrCompletions(t *testing.T) {
	store, _ := testStore(t)
	r := openRequest()
	r.PendingInput = []string{"real-approval"}
	opened := applyReport(t, store, r)
	applyReport(t, store, Request{Version: 1, Op: "complete", StreamID: opened.StreamID, Sequence: 2, EventID: "turn", Outcome: "finished"})
	require.NoError(t, store.RecordStatus(context.Background(), *r.Binding, StatusWorking))
	require.NoError(t, store.RecordStatus(context.Background(), *r.Binding, StatusIdle))
	state := loadReports(t, store)
	require.Len(t, state.Streams, 2)
	for _, stream := range state.Streams {
		if stream.ID == opened.StreamID {
			require.Equal(t, []string{"real-approval"}, stream.PendingInput)
		}
	}
	require.Len(t, state.Events, 1)
	require.False(t, state.Events[0].Acknowledged)
}

func TestSimpleStatusConcurrentCallsAllocateSequenceUnderLock(t *testing.T) {
	store, _ := testStore(t)
	binding := *openRequest().Binding
	const count = 24
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- store.RecordStatus(context.Background(), binding, StatusWorking) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	state := loadReports(t, store)
	require.Len(t, state.Streams, 1)
	require.EqualValues(t, count, state.Streams[0].Sequence)
	require.EqualValues(t, count, state.Revision)
}

func TestSimpleStatusBindingIsolationAndValidation(t *testing.T) {
	store, _ := testStore(t)
	binding := *openRequest().Binding
	for i := 0; i < 3; i++ {
		b := binding
		b.Pane = fmt.Sprintf("%%%d", i)
		require.NoError(t, store.RecordStatus(context.Background(), b, StatusWorking))
	}
	binding.Generation = "respawn"
	require.NoError(t, store.RecordStatus(context.Background(), binding, StatusIdle))
	require.Len(t, loadReports(t, store).Streams, 4)
	require.Error(t, store.RecordStatus(context.Background(), binding, "invalid"))
	require.Error(t, store.RecordStatus(context.Background(), Binding{}, StatusWorking))
}

func TestSimpleStatusSequenceOverflowDoesNotReset(t *testing.T) {
	store, _ := testStore(t)
	binding := *openRequest().Binding
	require.NoError(t, store.RecordStatus(context.Background(), binding, StatusWorking))
	state := loadReports(t, store)
	state.Streams[0].Sequence = math.MaxUint64
	require.NoError(t, store.write(context.Background(), state))
	require.ErrorContains(t, store.RecordStatus(context.Background(), binding, StatusIdle), "sequence exhausted")
}

func TestSimpleStatusSourceCannotBeClaimedOrMutatedByGenericProducer(t *testing.T) {
	store, _ := testStore(t)
	r := openRequest()
	r.Source, r.Session = paneStatusSource, r.Binding.Generation
	r.PendingInput = []string{"real-approval"}
	_, err := store.Apply(context.Background(), r)
	require.ErrorContains(t, err, "reserved")
	require.Empty(t, loadReports(t, store).Streams)
	require.NoError(t, store.RecordStatus(context.Background(), *r.Binding, StatusNeedsInput))
	before := loadReports(t, store)
	for _, op := range []string{"update", "close", "complete"} {
		request := Request{Version: Version, Op: op, StreamID: before.Streams[0].ID, Sequence: 2}
		switch op {
		case "update":
			request.Activity, request.PendingInput = Working, []string{}
		case "complete":
			request.EventID, request.Outcome = "turn", "finished"
		}
		_, err := store.Apply(context.Background(), request)
		require.ErrorContains(t, err, "only be changed through RecordStatus")
		require.Equal(t, before, loadReports(t, store))
	}
}
