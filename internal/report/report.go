// Package report stores explicit, unauthenticated work observations from local
// integrations. It does not detect programs, interpret tool events, or validate
// live tmux membership. Its version-1 storage schema is experimental.
package report

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

const (
	Version             = 1
	MaxRequestBytes     = 16 << 10
	DefaultLeaseSeconds = 60
	maxLeaseSeconds     = 300
	maxStreams          = 128
	maxRetired          = 256
	maxEvents           = 256
	maxPending          = 16
	retention           = 7 * 24 * time.Hour
)

type Activity string

const (
	Unknown Activity = "unknown"
	Working Activity = "working"
	Idle    Activity = "idle"
)

// Binding is an explicit producer-supplied attachment, not validated navigation
// identity. Server distinguishes tmux servers; generation must distinguish pane
// respawns/replacements. Adapters and future consumers must verify these facts.
type Binding struct {
	Server     string `json:"server"`
	Pane       string `json:"pane"`
	Generation string `json:"generation"`
}

// Request is one operation. Open declares a new producer/session lifetime;
// update replaces activity and optionally the pending-input set. Complete never
// implies idle or successful work. Sequence covers every mutation of a stream.
type Request struct {
	Version        int      `json:"version"`
	Op             string   `json:"op"`
	Source         string   `json:"source,omitempty"`
	Session        string   `json:"session,omitempty"`
	Binding        *Binding `json:"binding,omitempty"`
	RegistrationID string   `json:"registration_id,omitempty"`
	StreamID       string   `json:"stream_id,omitempty"`
	Sequence       uint64   `json:"sequence,omitempty"`
	LeaseSeconds   int      `json:"lease_seconds,omitempty"`
	Activity       Activity `json:"activity,omitempty"`
	PendingInput   []string `json:"pending_input,omitzero"`
	EventID        string   `json:"event_id,omitempty"`
	Outcome        string   `json:"outcome,omitempty"`
}

type Receipt struct {
	Version       int    `json:"version"`
	Revision      uint64 `json:"revision"`
	StreamID      string `json:"stream_id,omitempty"`
	Sequence      uint64 `json:"sequence,omitempty"`
	Duplicate     bool   `json:"duplicate"`
	EventRecorded bool   `json:"event_recorded"`
}

type Stream struct {
	ID             string     `json:"id"`
	Source         string     `json:"source"`
	Session        string     `json:"session"`
	Binding        Binding    `json:"binding"`
	RegistrationID string     `json:"registration_id"`
	OpenDigest     string     `json:"open_digest"`
	Sequence       uint64     `json:"sequence"`
	LastDigest     string     `json:"last_digest"`
	Activity       Activity   `json:"activity"`
	PendingInput   []string   `json:"pending_input"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	RetiredAt      *time.Time `json:"retired_at,omitempty"`
}

// Live checks only the reporting lease, not process health or pane membership.
// A future consumer must also validate the binding. Expired evidence must not
// be displayed as current work/input or interpreted as a completed occurrence.
func (s Stream) Live(now time.Time) bool {
	return s.RetiredAt == nil && now.Before(s.ExpiresAt)
}

type Event struct {
	Source       string    `json:"source"`
	Session      string    `json:"session"`
	ID           string    `json:"id"`
	StreamID     string    `json:"stream_id"`
	Binding      Binding   `json:"binding"`
	Outcome      string    `json:"outcome"`
	RecordedAt   time.Time `json:"recorded_at"`
	Acknowledged bool      `json:"acknowledged"`
}

type Snapshot struct {
	Version  int      `json:"version"`
	Revision uint64   `json:"revision"`
	Streams  []Stream `json:"streams"`
	Events   []Event  `json:"events"`
}

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)
var paneID = regexp.MustCompile(`^%[0-9]{1,20}$`)

func validID(value string) bool { return len(value) <= 128 && identifier.MatchString(value) }

func validBinding(b Binding) bool {
	return validID(b.Server) && paneID.MatchString(b.Pane) && validID(b.Generation)
}

func validActivity(a Activity) bool { return a == Unknown || a == Working || a == Idle }

func validOutcome(o string) bool {
	return o == "finished" || o == "succeeded" || o == "failed" || o == "cancelled" || o == "unknown"
}

func validatePending(ids []string) error {
	if len(ids) > maxPending {
		return fmt.Errorf("pending_input exceeds %d requests", maxPending)
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !validID(id) || seen[id] {
			return errors.New("pending_input requires distinct bounded identifiers")
		}
		seen[id] = true
	}
	return nil
}

func (r Request) validate() error {
	if r.Version != Version {
		return errors.New("unsupported report version")
	}
	if r.LeaseSeconds != 0 && (r.LeaseSeconds < 10 || r.LeaseSeconds > maxLeaseSeconds) {
		return fmt.Errorf("lease_seconds must be between 10 and %d", maxLeaseSeconds)
	}
	if err := validatePending(r.PendingInput); err != nil {
		return err
	}
	switch r.Op {
	case "open":
		if !validID(r.Source) || !validID(r.Session) || !validID(r.RegistrationID) || r.Binding == nil || !validBinding(*r.Binding) || !validActivity(r.Activity) {
			return errors.New("open requires source, session, registration_id, binding and activity")
		}
		if r.StreamID != "" || r.Sequence != 0 || r.EventID != "" || r.Outcome != "" {
			return errors.New("open cannot supply stream_id, sequence or completion fields")
		}
	case "update", "complete", "close":
		if !validID(r.StreamID) || r.Sequence < 2 {
			return errors.New("stream mutation requires stream_id and sequence >= 2")
		}
		if r.Source != "" || r.Session != "" || r.Binding != nil || r.RegistrationID != "" {
			return errors.New("stream identity and binding are immutable")
		}
		if r.Op == "update" {
			if !validActivity(r.Activity) || r.EventID != "" || r.Outcome != "" {
				return errors.New("update requires activity and no completion fields")
			}
		} else if r.Activity != "" || r.PendingInput != nil || r.LeaseSeconds != 0 {
			return errors.New("complete/close cannot change activity, input requests or lease")
		}
		if r.Op == "complete" && (!validID(r.EventID) || !validOutcome(r.Outcome)) {
			return errors.New("complete requires event_id and explicit outcome")
		}
		if r.Op == "close" && (r.EventID != "" || r.Outcome != "") {
			return errors.New("close cannot supply completion fields")
		}
	case "ack":
		if !validID(r.Source) || !validID(r.Session) || !validID(r.EventID) {
			return errors.New("ack requires source, session and event_id")
		}
		if r.Binding != nil || r.RegistrationID != "" || r.StreamID != "" || r.Sequence != 0 || r.Activity != "" || r.PendingInput != nil || r.Outcome != "" || r.LeaseSeconds != 0 {
			return errors.New("ack cannot mutate stream state")
		}
	default:
		return errors.New("op must be open, update, complete, close or ack")
	}
	return nil
}

func (r Request) lease() time.Duration {
	seconds := r.LeaseSeconds
	if seconds == 0 {
		seconds = DefaultLeaseSeconds
	}
	return time.Duration(seconds) * time.Second
}
