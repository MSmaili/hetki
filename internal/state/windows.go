package state

import (
	"github.com/MSmaili/tmx/internal/domain"
)

func compareWindows(diff *domain.Diff, desired, actual map[string][]domain.Window) *domain.Diff {
	for session, desiredWindows := range desired {
		processSession(diff, session, desiredWindows, actual[session])
	}

	//missing session windows
	for session, actualWindows := range actual {
		_, ok := desired[session]
		if !ok {
			diff.ExtraWindows[session] = append(diff.ExtraWindows[session], actualWindows...)
		}
	}

	return diff
}

func processSession(diff *domain.Diff, session string, desiredWindows []domain.Window, actualWindows []domain.Window) {

	desiredMap := windowsKey(desiredWindows)
	actualMap := windowsKey(actualWindows)

	missing := missingWindows(desiredMap, actualMap)
	mismatched := mismatchedWindows(desiredMap, actualMap)
	extra := extraWindows(desiredMap, actualMap)

	if len(missing) > 0 {
		diff.MissingWindows[session] = missing
	}

	if len(mismatched) > 0 {
		diff.Mismatched[session] = mismatched
	}

	if len(extra) > 0 {
		diff.ExtraWindows[session] = extra
	}
}

func missingWindows(desiredMap map[string]domain.Window, actualMap map[string]domain.Window) []domain.Window {
	var missing []domain.Window

	for key, dw := range desiredMap {
		_, exist := actualMap[key]
		if !exist {
			missing = append(missing, dw)
		}
	}
	return missing
}

func mismatchedWindows(desiredMap map[string]domain.Window, actualMap map[string]domain.Window) []domain.WindowMismatch {
	var mismatched []domain.WindowMismatch

	for key, dw := range desiredMap {
		aw, ok := actualMap[key]
		if !ok {
			continue // handled as missing
		}
		if !windowsEqual(dw, aw) {
			mismatched = append(mismatched, domain.WindowMismatch{
				Actual:  aw,
				Desired: dw,
			})
		}
	}

	return mismatched
}

func extraWindows(desiredMap map[string]domain.Window, actualMap map[string]domain.Window) []domain.Window {
	var extraWindows []domain.Window

	for key, aw := range actualMap {
		_, exist := desiredMap[key]
		if !exist {
			extraWindows = append(extraWindows, aw)
		}
	}
	return extraWindows
}

func windowsKey(windows []domain.Window) map[string]domain.Window {
	m := make(map[string]domain.Window, len(windows))
	for _, w := range windows {
		m[w.Path] = w // TODO: consider if Path is the unique key or we should combine?
	}
	return m
}

func windowsEqual(w1, w2 domain.Window) bool {
	return w1.Name == w2.Name &&
		w1.Path == w2.Path &&
		intPtrEqual(w1.Index, w2.Index) &&
		w1.Layout == w2.Layout &&
		w1.Command == w2.Command
}

func intPtrEqual(a, b *int) bool {
	if a == b {
		return true // covers both nil
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
