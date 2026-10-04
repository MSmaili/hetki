package report

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"
)

// Status is an explicit current-state assertion, not a completion outcome.
type Status string

const (
	StatusWorking    Status = "working"
	StatusNeedsInput Status = "needs-input"
	StatusIdle       Status = "idle"
	StatusUnknown    Status = "unknown"
)

// Reserved for the simple status facade. Generic producers must not own this
// source: its synthetic pending indication is replaced on every status call.
const paneStatusSource = "pane"

func Statuses() []string {
	return []string{string(StatusWorking), string(StatusNeedsInput), string(StatusIdle), string(StatusUnknown)}
}

func ParseStatus(value string) (Status, error) {
	switch Status(value) {
	case StatusWorking, StatusNeedsInput, StatusIdle, StatusUnknown:
		return Status(value), nil
	default:
		return "", errors.New("status must be working, needs-input, idle or unknown")
	}
}

// RecordStatus maintains a pane-scoped reporting slot without exposing stream
// IDs, registration or sequence numbers. Binding must be resolved by the caller;
// storage remains independent of tmux. Allocation/update is one locked mutation.
// This explicit status replaces only this slot's synthetic input indication,
// never another integration's real pending request IDs or completion events.
func (s *Store) RecordStatus(ctx context.Context, binding Binding, status Status) error {
	if _, err := ParseStatus(string(status)); err != nil {
		return err
	}
	if !validBinding(binding) {
		return errors.New("invalid report attachment")
	}
	activity := Activity(status)
	pending := []string{}
	if status == StatusNeedsInput {
		activity, pending = Idle, []string{"reported-input"}
	}
	_, err := s.mutate(ctx, func(state *Snapshot, now time.Time) (Receipt, error) {
		for _, stream := range state.Streams {
			if stream.Source != paneStatusSource || stream.Session != binding.Generation || stream.Binding != binding || !stream.Live(now) {
				continue
			}
			if stream.Sequence == math.MaxUint64 {
				return Receipt{}, errors.New("report sequence exhausted")
			}
			return state.apply(Request{Version: Version, Op: "update", StreamID: stream.ID, Sequence: stream.Sequence + 1,
				Activity: activity, PendingInput: pending}, now)
		}
		// Registration uses the current published revision, generated under the
		// same lock. A renewed slot after expiry gets a fresh reporting lifetime.
		return state.apply(Request{Version: Version, Op: "open", Source: paneStatusSource, Session: binding.Generation,
			RegistrationID: "status-" + strconv.FormatUint(state.Revision, 10), Binding: &binding, Activity: activity, PendingInput: pending}, now)
	})
	return err
}
