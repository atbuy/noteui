package tui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
)

func TestActiveWorkspaceSyncRemoteRootReturnsConfiguredOverride(t *testing.T) {
	m := Model{
		workspaceName: "work",
		cfg: config.Config{
			Workspaces: map[string]config.WorkspaceConfig{
				"work": {SyncRemoteRoot: " /srv/noteui/work "},
			},
		},
	}

	require.Equal(t, "/srv/noteui/work", m.activeWorkspaceSyncRemoteRoot())
}

func TestActiveWorkspaceSyncRemoteRootReturnsEmptyWithoutWorkspace(t *testing.T) {
	m := Model{
		cfg: config.Config{
			Workspaces: map[string]config.WorkspaceConfig{
				"work": {SyncRemoteRoot: "/srv/noteui/work"},
			},
		},
	}

	require.Empty(t, m.activeWorkspaceSyncRemoteRoot())
}
