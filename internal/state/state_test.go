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
		PinnedNotes:         []string{"inbox/today.md", "ideas.md"},
		PinnedCategories:    []string{"inbox", "work/projects"},
		CollapsedCategories: []string{"archive"},
		RecentCommands:      []string{"show_help", "refresh"},
		SortByModTime:       true,
	}

	require.NoError(t, Save(want))

	path := filepath.Join(xdg, "noteui", "state.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	text := string(data)
	for _, fragment := range []string{
		`"pinned_notes"`,
		`"pinned_categories"`,
		`"collapsed_categories"`,
		`"recent_commands"`,
		`"sort_by_mod_time": true`,
	} {
		require.Contains(t, text, fragment)
	}

	got, err := Load()
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(want, got))
}
