package state

type windowKey struct {
	Name string
	Path string
}

func compareWindows(diff *Diff, desired, actual *State) {
	common := CommonSessions(desired, actual)

	for _, sessionName := range common {
		desiredSession := desired.Sessions[sessionName]
		actualSession := actual.Sessions[sessionName]

		windowDiff := compareSessionWindows(desiredSession.Windows, actualSession.Windows)
		if !windowDiff.IsEmpty() {
			diff.Windows[sessionName] = windowDiff
		}
	}
}

func compareSessionWindows(desired, actual []*Window) ItemDiff[Window] {
	actualMap := windowsByKey(actual)

	windowDiff := ItemDiff[Window]{
		Missing:    make([]Window, 0, len(desired)),
		Extra:      make([]Window, 0, len(actual)),
		Mismatched: make([]Mismatch[Window], 0),
	}

	for _, desiredWindow := range desired {
		key := keyForWindow(desiredWindow)
		actualWindow, exists := actualMap[key]
		if !exists {
			windowDiff.Missing = append(windowDiff.Missing, *desiredWindow)
		} else {
			if !windowsMatch(desiredWindow, actualWindow) {
				windowDiff.Mismatched = append(windowDiff.Mismatched, Mismatch[Window]{
					Desired: *desiredWindow,
					Actual:  *actualWindow,
				})
			}
			delete(actualMap, key)
		}
	}

	for _, actualWindow := range actual {
		key := keyForWindow(actualWindow)
		if _, exists := actualMap[key]; exists {
			windowDiff.Extra = append(windowDiff.Extra, *actualWindow)
			delete(actualMap, key)
		}
	}

	return windowDiff
}

func windowsMatch(desired, actual *Window) bool {
	return len(desired.Panes) == len(actual.Panes)
}

func windowsByKey(windows []*Window) map[windowKey]*Window {
	m := make(map[windowKey]*Window, len(windows))
	for _, w := range windows {
		m[keyForWindow(w)] = w
	}
	return m
}

func keyForWindow(w *Window) windowKey {
	return windowKey{Name: w.Name, Path: w.Path}
}

func cloneWindows(ws []*Window) []Window {
	out := make([]Window, len(ws))
	for i, w := range ws {
		out[i] = *w
	}
	return out
}
