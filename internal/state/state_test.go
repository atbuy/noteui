package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestStatePathUsesXDGStateHomeWhenSet(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdg)

	path, err := statePath()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(xdg, "noteui", "state.json"), path)
}

func TestStatePathFallsBackToHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", home)

	path, err := statePath()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".local", "state", "noteui", "state.json"), path)
}

func TestLoadReturnsZeroValueWhenFileMissing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	s, err := Load()
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(State{}, s))
}

func TestLoadReturnsZeroValueWhenFileEmpty(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdg)

	path := filepath.Join(xdg, "noteui", "state.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, nil, 0o644))

	s, err := Load()
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(State{}, s))
}

func TestLoadRejectsInvalidJSON(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdg)

	path := filepath.Join(xdg, "noteui", "state.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("{not-json"), 0o644))

	_, err := Load()
	require.Error(t, err)
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdg)

	want := State{
		Workspaces: map[string]WorkspaceState{
			"work": {
				PinnedNotes:         []string{"inbox/today.md", "ideas.md"},
				PinnedCategories:    []string{"inbox", "work/projects"},
				CollapsedCategories: []string{"archive"},
				RecentCommands:      []string{"show_help", "refresh"},
				SortMethod:          "modified",
			},
		},
	}

	require.NoError(t, Save(want))

	path := filepath.Join(xdg, "noteui", "state.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	text := string(data)
	for _, fragment := range []string{
		`"workspaces"`,
		`"work"`,
		`"pinned_notes"`,
		`"pinned_categories"`,
		`"collapsed_categories"`,
		`"recent_commands"`,
		`"sort_method": "modified"`,
	} {
		require.Contains(t, text, fragment)
	}

	got, err := Load()
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(want, got))
}

func TestLoadMigratesLegacyFlatState(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdg)

	path := filepath.Join(xdg, "noteui", "state.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	legacy := `{
	  "pinned_notes": ["ideas.md"],
	  "pinned_categories": ["work"],
	  "collapsed_categories": ["archive"],
	  "recent_commands": ["refresh"],
	  "sort_by_mod_time": true
	}`
	require.NoError(t, os.WriteFile(path, []byte(legacy), 0o644))

	got, err := Load()
	require.NoError(t, err)
	require.Equal(t, WorkspaceState{
		PinnedNotes:         []string{"ideas.md"},
		PinnedCategories:    []string{"work"},
		CollapsedCategories: []string{"archive"},
		RecentCommands:      []string{"refresh"},
		SortMethod:          "modified",
	}, got.Workspace("default"))
}

func TestSetWorkspaceRemovesZeroValueWorkspace(t *testing.T) {
	var s State
	s.SetWorkspace("demo", WorkspaceState{RecentCommands: []string{"refresh"}})
	require.Contains(t, s.Workspaces, "demo")
	s.SetWorkspace("demo", WorkspaceState{})
	require.NotContains(t, s.Workspaces, "demo")
}
