package state

import "sort"

func compareSessions(diff *Diff, desired, actual *State) {
	for _, name := range sortedSessionNames(desired) {
		if _, ok := actual.Sessions[name]; !ok {
			diff.Sessions.Missing = append(diff.Sessions.Missing, name)
		}
	}

	for _, name := range sortedSessionNames(actual) {
		if _, ok := desired.Sessions[name]; !ok {
			diff.Sessions.Extra = append(diff.Sessions.Extra, name)
		}
	}
}

func CommonSessions(desired, actual *State) []string {
	common := make([]string, 0, len(desired.Sessions))
	for name := range desired.Sessions {
		if _, exists := actual.Sessions[name]; exists {
			common = append(common, name)
		}
	}
	sort.Strings(common)
	return common
}

func sortedSessionNames(s *State) []string {
	names := make([]string, 0, len(s.Sessions))
	for name := range s.Sessions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
