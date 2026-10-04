package report

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func openRequest() Request {
	return Request{Version: Version, Op: "open", Source: "test-tool", Session: "session-1", RegistrationID: "registration-1",
		Binding: &Binding{Server: "server-1", Pane: "%1", Generation: "birth-1"}, Activity: Working}
}

func TestDecodeReportRejectsInvalidInput(t *testing.T) {
	valid, err := json.Marshal(openRequest())
	require.NoError(t, err)
	request, err := Decode(strings.NewReader(string(valid)))
	require.NoError(t, err)
	require.Equal(t, openRequest(), request)
	for name, input := range map[string]string{
		"empty": "", "null": "null", "array": "[]", "malformed": "{",
		"trailing JSON": string(valid) + " {}", "trailing garbage": string(valid) + " garbage",
		"duplicate":        strings.Replace(string(valid), `"version":1`, `"version":1,"version":1`, 1),
		"case alias":       strings.Replace(string(valid), `"version":1`, `"version":1,"Version":2`, 1),
		"nested duplicate": strings.Replace(string(valid), `"pane":"%1"`, `"pane":"%1","pane":"%2"`, 1),
		"unknown":          strings.Replace(string(valid), `"version":1`, `"version":1,"transcript":"private"`, 1),
		"nested unknown":   strings.Replace(string(valid), `"pane":"%1"`, `"pane":"%1","path":"private"`, 1),
		"null version":     strings.Replace(string(valid), `"version":1`, `"version":null`, 1),
		"null activity":    strings.Replace(string(valid), `"activity":"working"`, `"activity":null`, 1),
		"invalid unicode":  strings.Replace(string(valid), `"session-1"`, `"\ud800"`, 1),
		"invalid UTF8":     string(valid) + "\xff",
		"oversized":        strings.Repeat(" ", MaxRequestBytes+1),
		"wrong type":       strings.Replace(string(valid), `"version":1`, `"version":"1"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Decode(strings.NewReader(input))
			require.Error(t, err)
			require.NotContains(t, err.Error(), "private")
		})
	}
}

func TestReportValidation(t *testing.T) {
	for name, mutate := range map[string]func(*Request){
		"version":              func(r *Request) { r.Version++ },
		"operation":            func(r *Request) { r.Op = "done" },
		"missing source":       func(r *Request) { r.Source = "" },
		"missing registration": func(r *Request) { r.RegistrationID = "" },
		"long source":          func(r *Request) { r.Source = strings.Repeat("x", 129) },
		"control":              func(r *Request) { r.Session = "\x1b[31m" },
		"spaces":               func(r *Request) { r.Session = "secret prompt" },
		"missing binding":      func(r *Request) { r.Binding = nil },
		"invalid pane":         func(r *Request) { r.Binding.Pane = "1" },
		"missing generation":   func(r *Request) { r.Binding.Generation = "" },
		"missing server":       func(r *Request) { r.Binding.Server = "" },
		"unknown activity":     func(r *Request) { r.Activity = "finished" },
		"missing activity":     func(r *Request) { r.Activity = "" },
		"short lease":          func(r *Request) { r.LeaseSeconds = 9 },
		"long lease":           func(r *Request) { r.LeaseSeconds = 301 },
		"negative lease":       func(r *Request) { r.LeaseSeconds = -1 },
		"duplicate requests":   func(r *Request) { r.PendingInput = []string{"q1", "q1"} },
		"invalid request":      func(r *Request) { r.PendingInput = []string{""} },
		"too many requests":    func(r *Request) { r.PendingInput = make([]string, maxPending+1) },
		"injected sequence":    func(r *Request) { r.Sequence = 1 },
		"injected completion":  func(r *Request) { r.EventID = "event-1" },
	} {
		t.Run(name, func(t *testing.T) { request := openRequest(); mutate(&request); require.Error(t, request.validate()) })
	}
	for _, r := range []Request{
		{Version: 1, Op: "update", StreamID: "stream", Sequence: 2, Activity: Idle},
		{Version: 1, Op: "complete", StreamID: "stream", Sequence: 2, EventID: "e1", Outcome: "finished"},
		{Version: 1, Op: "close", StreamID: "stream", Sequence: 2},
		{Version: 1, Op: "ack", Source: "source", Session: "session", EventID: "e1"},
	} {
		require.NoError(t, r.validate())
	}
	for _, r := range []Request{
		{Version: 1, Op: "update", StreamID: "stream", Sequence: 1, Activity: Idle},
		{Version: 1, Op: "update", StreamID: "stream", Sequence: 2},
		{Version: 1, Op: "update", StreamID: "stream", Sequence: 2, Activity: Idle, Source: "source"},
		{Version: 1, Op: "complete", StreamID: "stream", Sequence: 2, EventID: "e1"},
		{Version: 1, Op: "complete", StreamID: "stream", Sequence: 2, EventID: "e1", Outcome: "finished", Activity: Idle},
		{Version: 1, Op: "close", StreamID: "stream", Sequence: 2, LeaseSeconds: 60},
		{Version: 1, Op: "close", StreamID: "stream", Sequence: 2, PendingInput: []string{"q1"}},
		{Version: 1, Op: "ack", Source: "source", Session: "session", EventID: "e1", Activity: Idle},
	} {
		require.Error(t, r.validate())
	}
}

func FuzzDecodeReport(f *testing.F) {
	valid, _ := json.Marshal(openRequest())
	f.Add(string(valid))
	f.Add(`{"version":1,"op":"update","stream_id":"stream","sequence":2,"activity":"idle"}`)
	f.Add(`null`)
	f.Fuzz(func(t *testing.T, input string) {
		r, err := Decode(strings.NewReader(input))
		if err == nil {
			require.NoError(t, r.validate())
			require.LessOrEqual(t, len(input), MaxRequestBytes)
		}
	})
}

type failingReportReader struct{ err error }

func (r failingReportReader) Read([]byte) (int, error) { return 0, r.err }

func TestDecodePreservesReaderError(t *testing.T) {
	want := errors.New("input unavailable")
	_, err := Decode(failingReportReader{err: want})
	require.ErrorIs(t, err, want)
}
