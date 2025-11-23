package state

import (
	"github.com/MSmaili/tmx/internal/domain"
)

type Diff interface {
	Compare(desiredConfig map[string][]domain.Window, currentSession map[string][]domain.Window) domain.Diff
}

func Compare(desired map[string][]domain.Window, actual map[string][]domain.Window) domain.Diff {
	diff := domain.Diff{
		MissingWindows: make(map[string][]domain.Window),
		ExtraWindows:   make(map[string][]domain.Window),
		Mismatched:     make(map[string][]domain.WindowMismatch),
	}

	compareSessions(&diff, desired, actual)
	compareWindows(&diff, desired, actual)

	return diff
}
