package core

import (
	"strings"

	"github.com/MSmaili/hetki/internal/tui/contracts"
)

func findRowIndexByID(rows []row, nodeID string) int {
	if nodeID == "" {
		return -1
	}
	for i := range rows {
		if rows[i].Node.ID == nodeID {
			return i
		}
	}
	return -1
}

func preferredSelectionID(snapshot contracts.Snapshot, intent *contracts.Intent, previousID string) string {
	if intent == nil {
		return previousID
	}
	switch intent.Type {
	case contracts.IntentCreateSession, contracts.IntentRenameSession:
		name := strings.TrimSpace(intent.Payload["name"])
		if name != "" {
			return "session:" + name
		}
	case contracts.IntentCreateWindow:
		session := strings.TrimSpace(intent.Payload["session"])
		if session == "" {
			session = sessionFromNodeTarget(intent.Target)
		}
		name := strings.TrimSpace(intent.Payload["name"])
		if session != "" && name != "" {
			if id := findWindowNodeIDByName(snapshot.Nodes, session, name); id != "" {
				return id
			}
		}
	case contracts.IntentRenameWindow:
		return nodeIDFromTarget(intent.Target)
	case contracts.IntentDeleteWindow:
		return "session:" + sessionFromNodeTarget(intent.Target)
	}
	return previousID
}

func sessionFromNodeTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	session, _, hasWindow := strings.Cut(target, ":")
	if hasWindow {
		return strings.TrimSpace(session)
	}
	return target
}

func nodeIDFromTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if session, window, ok := sessionWindowFromNodeTarget(target); ok {
		return "window:" + session + ":" + window
	}
	return "session:" + target
}

func sessionWindowFromNodeTarget(target string) (string, string, bool) {
	session, rest, hasWindow := strings.Cut(strings.TrimSpace(target), ":")
	if !hasWindow || session == "" || rest == "" {
		return "", "", false
	}
	window, _, _ := strings.Cut(rest, ".")
	window = strings.TrimSpace(window)
	if window == "" {
		return "", "", false
	}
	return strings.TrimSpace(session), window, true
}

func findWindowNodeIDByName(nodes []contracts.Node, sessionName, windowName string) string {
	for _, session := range nodes {
		if session.Kind != contracts.NodeKindSession || session.Label != sessionName {
			continue
		}
		for _, window := range session.Children {
			if windowDisplayName(window.Label) == windowName {
				return window.ID
			}
		}
	}
	return ""
}

func windowDisplayName(label string) string {
	parts := strings.Fields(strings.TrimSpace(label))
	if len(parts) <= 1 {
		return strings.TrimSpace(label)
	}
	return strings.Join(parts[1:], " ")
}
