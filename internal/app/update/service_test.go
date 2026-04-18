package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubUpdater struct {
	dryRunCalls int
	updateCalls int
	lastVersion string
}

func (s *stubUpdater) Name() string { return "stub" }
func (s *stubUpdater) DryRun()      { s.dryRunCalls++ }
func (s *stubUpdater) Update(version string) error {
	s.updateCalls++
	s.lastVersion = version
	return nil
}

func TestServiceRunDryRunUsesUpdaterDryRun(t *testing.T) {
	updater := &stubUpdater{}
	service := Service{
		SetVerbose:       func(bool) {},
		Executable:       func() (string, error) { return "/tmp/hetki", nil },
		DetermineUpdater: func(string) (Updater, error) { return updater, nil },
		GetLatestVersion: func() (string, error) {
			t.Fatal("GetLatestVersion should not be called in dry-run mode")
			return "", nil
		},
	}

	err := service.Run(Options{DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, 1, updater.dryRunCalls)
	assert.Zero(t, updater.updateCalls)
}

func TestServiceRunSkipsUpdateWhenAlreadyLatest(t *testing.T) {
	updater := &stubUpdater{}
	service := Service{
		SetVerbose:       func(bool) {},
		Executable:       func() (string, error) { return "/tmp/hetki", nil },
		DetermineUpdater: func(string) (Updater, error) { return updater, nil },
		GetLatestVersion: func() (string, error) { return "v1.2.3", nil },
	}

	err := service.Run(Options{CurrentVersion: "v1.2.3"})
	require.NoError(t, err)
	assert.Zero(t, updater.dryRunCalls)
	assert.Zero(t, updater.updateCalls)
	assert.Empty(t, updater.lastVersion)
}
