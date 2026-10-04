# Reporting work status

Run the command **inside the tmux pane whose status you want to report**:

```sh
hetki report --status working
hetki report --status needs-input
hetki report --status idle
hetki report --status unknown
```

That's the complete public interface. No JSON, pane/session IDs, registration,
sequence numbers or lease settings are required. Success is quiet; invalid
statuses and failures return a nonzero exit code and an error on stderr.
`--status` supports shell completion. Names are exact and case-sensitive.

| Status | Meaning |
|---|---|
| `working` | The caller explicitly reports ongoing work. |
| `needs-input` | The caller explicitly reports waiting for input/approval. |
| `idle` | Not currently working; **not** proof of successful completion. |
| `unknown` | The caller cannot establish current activity. |

Each call asserts the current status of this pane-scoped reporting slot. Thus
`working`/`idle`/`unknown` replace this slot's earlier `needs-input` indication.
Only report that change when it is actually established. This cannot approve a
tool permission or clear a different integration's pending request records.

## Automatic context

Hetki uses the invoking terminal's `TMUX` and `TMUX_PANE`, queries that exact
socket/pane and checks process ancestry. It does **not** guess the active pane,
match directories/program names, or scrape terminal output. The command errors
without saving if context is absent, the pane is missing/dead, the environment
points at another server, or the caller cannot be associated with that pane.

Internal identity incorporates the tmux server's lifetime and the pane process's
PID/birth time, so pane respawn and server restart do not reuse previous status
slots. Context observation is best-effort and non-atomic; future readers must
still verify that the saved binding is current. Automatic attachment is supported
on macOS and Linux. No tmux user options, hooks or configuration are modified.

For agent integrations, run the report hook in the **local terminal frontend**.
A shared agent server's inherited environment does not identify whichever client
or agent session produced an event. Ancestry is not proof of semantic session
association. The OpenCode CLI integration must filter/associate its actual
sessions; that integration is not implemented yet.

## What stays internal

Hetki creates/reuses a reporting stream and allocates ordering under the same
writer lock, including simultaneous calls. A successful call renews a 60-second
lease. Expired evidence is unknown/stale, **never** automatic completion. An
event-based adapter must renew while healthy, even during a long quiet operation;
one call is not a permanent "working" or "needs input" status.

Reports use one private local JSON snapshot at
`$XDG_STATE_HOME/hetki/reports/state.json`, falling back to
`~/.local/state/hetki/reports/state.json`. No daemon is needed. The command
does not read stdin or print protocol receipts. It uses a two-second cancellable
deadline, although in-flight native file/process calls cannot be interrupted.
Adapters should also bound their subprocess and avoid failing the agent when
reporting is unavailable.

Storage uses a stable sidecar lock and synced atomic replacement. It refuses
corrupt, unknown-version, public, symlinked, hard-linked or nonregular state.
No prompts/transcripts, full command arguments, outputs or credentials are
stored. Reports are same-user assertions, not authenticated status.

The generic producer API reserves source `pane` for the simple-status facade.
Other producers cannot claim or mutate its streams through `Store.Apply`;
likewise the facade never owns another producer's pending requests. This prevents
accidental cross-API updates, not intentional same-user file modification.

Capacity is bounded: 128 live streams, 256 retired receipts, 256 retained
completion events and a 2 MiB snapshot. Live records are not evicted for capacity.
Successful writes lazily prune retired/history records after seven days, or
oldest history when capacity is exceeded (including unread history). Nothing
needs to continuously delete the file; readers must honor expiry even if old
bytes remain on disk without new writes.

## Separate future capabilities

The internal model keeps current activity, real pending requests, completion
occurrences and acknowledgment separate. They are not all one "done" status.
The public status command never creates or acknowledges a completion event.

Agent plugins, TUI status indicators/readers, event-reporting CLI ergonomics,
notifications/sound and live watching are separate slices and are not enabled
by this command. The internal schema remains experimental, not a public JSON
protocol integrations must construct.
