package state

import (
	"fmt"
	"testing"

	"github.com/MSmaili/tmx/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestWindowsEqual(t *testing.T) {
	t.Helper()

	tests := []struct {
		name string
		a, b domain.Window
		want bool
	}{
		{
			name: "Equal windows",
			a:    domain.Window{Index: 1, Name: "a", Path: "/x", Layout: "h", Command: "ls"},
			b:    domain.Window{Index: 1, Name: "a", Path: "/x", Layout: "h", Command: "ls"},
			want: true,
		},
		{
			name: "Different name",
			a:    domain.Window{Index: 1, Name: "a", Path: "/x"},
			b:    domain.Window{Index: 1, Name: "b", Path: "/x"},
			want: false,
		},
		{
			name: "Different path",
			a:    domain.Window{Index: 1, Name: "a", Path: "/x"},
			b:    domain.Window{Index: 1, Name: "a", Path: "/y"},
			want: false,
		},
		{
			name: "Different index",
			a:    domain.Window{Index: 1},
			b:    domain.Window{Index: 2},
			want: false,
		},
		{
			name: "Different layout",
			a:    domain.Window{Index: 1, Layout: "h"},
			b:    domain.Window{Index: 1, Layout: "v"},
			want: false,
		},
		{
			name: "Different command",
			a:    domain.Window{Index: 1, Command: "ls"},
			b:    domain.Window{Index: 1, Command: "top"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, windowsEqual(tt.a, tt.b, domain.CompareStrict))
		})
	}
}

func TestCompareMixedDiffInSameSession(t *testing.T) {
	desired := map[string][]domain.Window{
		"s": {
			{Index: 0, Name: "A2", Path: "/C"}, // mismatched
			{Index: 1, Name: "B", Path: "/B"},  // missing
			{Index: 2, Name: "C", Path: "/C"},  // match
		},
	}

	actual := map[string][]domain.Window{
		"s": {
			{Index: 0, Name: "A2", Path: "/A"}, // mismatched
			{Index: 2, Name: "C", Path: "/C"},  // match
			{Index: 3, Name: "D", Path: "/D"},  // extra
		},
	}

	diff := Compare(desired, actual, domain.CompareStrict)

	fmt.Println("missing")
	fmt.Println(diff.MissingWindows)
	assert.Len(t, diff.MissingWindows["s"], 1)
	assert.Equal(t, "B", diff.MissingWindows["s"][0].Name)

	assert.Len(t, diff.Mismatched["s"], 1)
	assert.Equal(t, "/C", diff.Mismatched["s"][0].Desired.Path)
	assert.Equal(t, "/A", diff.Mismatched["s"][0].Actual.Path)

	assert.Len(t, diff.ExtraWindows["s"], 1)
	assert.Equal(t, "D", diff.ExtraWindows["s"][0].Name)
}

func TestCompareKeyCollisionOverridesEarlier(t *testing.T) {
	desired := map[string][]domain.Window{
		"s": {
			{Index: 1, Name: "same"}, // will be overwritten
			{Index: 1, Name: "same"}, // overwrite
		},
	}

	actual := map[string][]domain.Window{
		"s": {},
	}

	diff := Compare(desired, actual, domain.CompareStrict)

	assert.Len(t, diff.MissingWindows["s"], 1)
	assert.Equal(t, "same", diff.MissingWindows["s"][0].Name)
}

func TestCompareMultipleMissingExtra(t *testing.T) {
	desired := map[string][]domain.Window{
		"s": {
			{Index: 0, Name: "A", Path: "/A"},
			{Index: 1, Name: "B", Path: "/B"},
		},
	}

	actual := map[string][]domain.Window{
		"s": {
			{Index: 2, Name: "C", Path: "/C"},
			{Index: 3, Name: "D", Path: "/D"},
		},
	}

	diff := Compare(desired, actual, domain.CompareStrict)

	assert.Len(t, diff.MissingWindows["s"], 2)
	assert.Len(t, diff.ExtraWindows["s"], 2)
}
