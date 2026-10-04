package report

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// apply implements lifecycle transitions in memory. The caller validates the
// request, prunes expired records, and publishes only successful transitions.
func (s *Snapshot) apply(r Request, now time.Time) (Receipt, error) {
	receipt := Receipt{Version: Version, StreamID: r.StreamID, Sequence: r.Sequence}
	if r.Op == "ack" {
		for i := range s.Events {
			e := &s.Events[i]
			if e.Source == r.Source && e.Session == r.Session && e.ID == r.EventID {
				receipt.Duplicate = e.Acknowledged
				e.Acknowledged = true
				return receipt, nil
			}
		}
		return Receipt{}, errors.New("completion event not found")
	}
	if r.Op == "open" {
		hash := digest(r)
		for _, stream := range s.Streams {
			if stream.Source == r.Source && stream.Session == r.Session && stream.RegistrationID == r.RegistrationID {
				if stream.OpenDigest != hash {
					return Receipt{}, errors.New("conflicting registration_id")
				}
				receipt.StreamID, receipt.Sequence, receipt.Duplicate = stream.ID, stream.Sequence, true
				return receipt, nil
			}
		}
		live := 0
		for _, stream := range s.Streams {
			if !stream.Live(now) {
				continue
			}
			live++
			if stream.Source == r.Source && stream.Session == r.Session && stream.Binding == *r.Binding {
				return Receipt{}, errors.New("a live stream already owns this source/session/binding; close it or wait for expiry")
			}
		}
		if live >= maxStreams {
			return Receipt{}, errors.New("report stream capacity reached")
		}
		id := make([]byte, 16)
		if _, err := rand.Read(id); err != nil {
			return Receipt{}, fmt.Errorf("generate report stream ID: %w", err)
		}
		receipt.StreamID, receipt.Sequence = hex.EncodeToString(id), 1
		s.Streams = append(s.Streams, Stream{ID: receipt.StreamID, Source: r.Source, Session: r.Session,
			Binding: *r.Binding, RegistrationID: r.RegistrationID, OpenDigest: hash, Sequence: 1, LastDigest: hash, Activity: r.Activity,
			PendingInput: copyPending(r.PendingInput), UpdatedAt: now, ExpiresAt: now.Add(r.lease())})
		return receipt, nil
	}
	index := -1
	for i := range s.Streams {
		if s.Streams[i].ID == r.StreamID {
			index = i
			break
		}
	}
	if index < 0 {
		return Receipt{}, errors.New("report stream not found; open a new lifetime")
	}
	stream := &s.Streams[index]
	hash := digest(r)
	if r.Sequence == stream.Sequence && hash == stream.LastDigest {
		receipt.Duplicate = true
		return receipt, nil
	}
	if !stream.Live(now) {
		return Receipt{}, errors.New("report stream is closed or expired; open a new lifetime")
	}
	if r.Sequence <= stream.Sequence {
		return Receipt{}, errors.New("outdated or conflicting report sequence")
	}
	switch r.Op {
	case "update":
		stream.Activity = r.Activity
		if r.PendingInput != nil {
			stream.PendingInput = copyPending(r.PendingInput)
		}
		stream.ExpiresAt = now.Add(r.lease())
	case "complete":
		found := false
		for _, event := range s.Events {
			if event.Source == stream.Source && event.Session == stream.Session && event.ID == r.EventID {
				if event.Outcome != r.Outcome {
					return Receipt{}, errors.New("conflicting completion outcome")
				}
				found = true
				break
			}
		}
		if !found {
			event := Event{Source: stream.Source, Session: stream.Session, ID: r.EventID,
				StreamID: stream.ID, Binding: stream.Binding, Outcome: r.Outcome, RecordedAt: now}
			s.Events = append([]Event{event}, s.Events...)
			receipt.EventRecorded = true
		}
	case "close":
		stream.RetiredAt = &now
	}
	stream.Sequence, stream.LastDigest, stream.UpdatedAt = r.Sequence, hash, now
	return receipt, nil
}

func copyPending(ids []string) []string {
	result := append([]string{}, ids...)
	sort.Strings(result)
	return result
}

func digest(r Request) string {
	if r.Op == "open" || r.PendingInput != nil {
		r.PendingInput = copyPending(r.PendingInput)
	}
	if r.Op == "open" || r.Op == "update" {
		r.LeaseSeconds = int(r.lease() / time.Second)
	}
	// Request contains only JSON-safe scalar fields and slices; marshaling
	// cannot fail. Canonical pending order/default lease make retries equivalent.
	data, _ := json.Marshal(r)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (s *Snapshot) prune(now time.Time) {
	kept := s.Streams[:0]
	for _, stream := range s.Streams {
		if stream.RetiredAt == nil && !stream.Live(now) {
			retired := stream.ExpiresAt
			stream.RetiredAt = &retired
		}
		if stream.RetiredAt == nil || now.Sub(*stream.RetiredAt) < retention {
			kept = append(kept, stream)
		}
	}
	s.Streams = kept
	sort.SliceStable(s.Streams, func(i, j int) bool {
		a, b := s.Streams[i], s.Streams[j]
		if (a.RetiredAt == nil) != (b.RetiredAt == nil) {
			return a.RetiredAt == nil
		}
		if a.RetiredAt != nil && !a.RetiredAt.Equal(*b.RetiredAt) {
			return a.RetiredAt.After(*b.RetiredAt)
		}
		// Newly retired streams precede older receipts before this stable
		// sort. Preserve that order when wall-clock timestamps tie.
		if a.RetiredAt != nil {
			return false
		}
		return a.ID < b.ID
	})
	live := 0
	for _, stream := range s.Streams {
		if stream.RetiredAt == nil {
			live++
		}
	}
	if len(s.Streams) > live+maxRetired {
		s.Streams = s.Streams[:live+maxRetired]
	}
	events := s.Events[:0]
	for _, event := range s.Events {
		if now.Sub(event.RecordedAt) < retention {
			events = append(events, event)
		}
	}
	s.Events = events
	sort.SliceStable(s.Events, func(i, j int) bool { return s.Events[i].RecordedAt.After(s.Events[j].RecordedAt) })
	if len(s.Events) > maxEvents {
		s.Events = s.Events[:maxEvents]
	}
}

func (s Snapshot) validate() error {
	if s.Version != Version || s.Streams == nil || s.Events == nil || len(s.Streams) > maxStreams+maxRetired || len(s.Events) > maxEvents {
		return errors.New("invalid report snapshot version, arrays or capacity")
	}
	seen := make(map[string]bool)
	bindings := make(map[string]bool)
	if s.Revision == 0 && (len(s.Streams) != 0 || len(s.Events) != 0) {
		return errors.New("nonempty report snapshot requires a positive revision")
	}
	registrations := make(map[string]bool)
	active, retired := 0, 0
	for _, stream := range s.Streams {
		_, hashErr := hex.DecodeString(stream.LastDigest)
		_, openHashErr := hex.DecodeString(stream.OpenDigest)
		if !validID(stream.ID) || !validID(stream.Source) || !validID(stream.Session) || !validID(stream.RegistrationID) || !validBinding(stream.Binding) ||
			len(stream.OpenDigest) != 64 || openHashErr != nil ||
			stream.Sequence == 0 || len(stream.LastDigest) != 64 || hashErr != nil || !validActivity(stream.Activity) ||
			stream.PendingInput == nil || stream.UpdatedAt.IsZero() || stream.ExpiresAt.IsZero() ||
			!stream.ExpiresAt.After(stream.UpdatedAt) || stream.ExpiresAt.Sub(stream.UpdatedAt) > maxLeaseSeconds*time.Second || seen[stream.ID] {
			return errors.New("invalid report stream")
		}
		if err := validatePending(stream.PendingInput); err != nil {
			return err
		}
		seen[stream.ID] = true
		registration := stream.Source + "/" + stream.Session + "/" + stream.RegistrationID
		if registrations[registration] {
			return errors.New("duplicate report registration")
		}
		registrations[registration] = true
		if stream.RetiredAt == nil {
			key := stream.Source + "/" + stream.Session + "/" + stream.Binding.Server + "/" + stream.Binding.Pane + "/" + stream.Binding.Generation
			if bindings[key] {
				return errors.New("duplicate active report binding")
			}
			bindings[key] = true
			active++
		} else {
			if stream.RetiredAt.IsZero() {
				return errors.New("invalid retirement time")
			}
			retired++
		}
	}
	if active > maxStreams || retired > maxRetired {
		return errors.New("invalid report stream capacity")
	}
	seen = make(map[string]bool)
	for _, event := range s.Events {
		key := event.Source + "/" + event.Session + "/" + event.ID
		if !validID(event.Source) || !validID(event.Session) || !validID(event.ID) || !validID(event.StreamID) ||
			!validBinding(event.Binding) || !validOutcome(event.Outcome) || event.RecordedAt.IsZero() || seen[key] {
			return errors.New("invalid report completion")
		}
		seen[key] = true
	}
	return nil
}
