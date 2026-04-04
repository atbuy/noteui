package tui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
)

func TestSortedSyncProfileNames(t *testing.T) {
	// nil profiles returns nil
	if got := sortedSyncProfileNames(config.SyncConfig{}); got != nil {
		require.Failf(t, "assertion failed", "expected nil for empty profiles, got %v", got)
	}

	// whitespace-only keys are excluded
	cfg := config.SyncConfig{
		Profiles: map[string]config.SyncProfile{
			"  ": {},
		},
	}
	if got := sortedSyncProfileNames(cfg); got != nil {
		require.Failf(t, "assertion failed", "expected nil when all keys are whitespace, got %v", got)
	}

	// names are sorted and trimmed
	cfg = config.SyncConfig{
		Profiles: map[string]config.SyncProfile{
			"zebra":  {},
			"alpha":  {},
			"middle": {},
		},
	}
	got := sortedSyncProfileNames(cfg)
	require.Equal(t, []string{"alpha", "middle", "zebra"}, got)
}
