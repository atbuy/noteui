package shortcuts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
)

func TestDefaultMapAndHelpEntries(t *testing.T) {
	m := DefaultMap()

	require.Equal(t, []string{"enter", "o"}, m.Open.Keys())
	require.Equal(t, "enter/o", m.Open.Help().Key)
	require.Equal(t, []string{"e"}, m.EditInApp.Keys())
	require.Equal(t, []string{"ctrl+p", ":"}, m.CommandPalette.Keys())
	require.Equal(t, []string{"ctrl+y"}, m.ShowThemePicker.Keys())

	entries := HelpEntries(m)
	require.Greater(t, len(entries), 20)
	requireHelpEntry(t, entries, "Tree", m.CommandPalette.Help().Key, "Command palette: notes, actions, and workspace switch")
	requireHelpEntry(t, entries, "Tree", m.EditInApp.Help().Key, "Edit current note in app")
	requireHelpEntry(t, entries, "Global", m.ShowThemePicker.Help().Key, "Open theme picker (searchable live preview; / or tab filters; saves theme.name only)")
	requireHelpEntry(t, entries, "Preview", m.TodoKey.Help().Key+m.TodoPriority.Help().Key, "Set or clear current todo priority")
}

func TestApplyConfigOverridesOnlyProvidedBindings(t *testing.T) {
	m := DefaultMap()

	ApplyConfig(&m, config.KeysConfig{
		Open:            []string{"ctrl+o"},
		Quit:            []string{"x", "ctrl+x"},
		ShowThemePicker: []string{"alt+t"},
	})

	require.Equal(t, []string{"ctrl+o"}, m.Open.Keys())
	require.Equal(t, "ctrl+o", m.Open.Help().Key)
	require.Equal(t, "Open in editor", m.Open.Help().Desc)
	require.Equal(t, []string{"x", "ctrl+x"}, m.Quit.Keys())
	require.Equal(t, []string{"alt+t"}, m.ShowThemePicker.Keys())
	require.Equal(t, []string{"/"}, m.Search.Keys())
}

func TestValidateCollisions(t *testing.T) {
	m := DefaultMap()
	require.Empty(t, ValidateCollisions(m))

	ApplyConfig(&m, config.KeysConfig{
		Refresh: []string{"q"},
	})

	collisions := ValidateCollisions(m)
	require.Len(t, collisions, 1)
	require.Contains(t, collisions[0], `key "q"`)
	require.True(t, strings.Contains(collisions[0], "refresh") && strings.Contains(collisions[0], "quit"))
}

func TestValidateCollisionsDetectsSortMenuConflicts(t *testing.T) {
	m := DefaultMap()

	ApplyConfig(&m, config.KeysConfig{
		SortByModified: []string{"n"},
	})

	collisions := ValidateCollisions(m)
	require.Len(t, collisions, 1)
	require.Contains(t, collisions[0], `sort menu key "n"`)
	require.Contains(t, collisions[0], "sort_by_name")
	require.Contains(t, collisions[0], "sort_by_modified")
}

func TestValidateCollisionsDetectsPreviewChordConflicts(t *testing.T) {
	m := DefaultMap()

	ApplyConfig(&m, config.KeysConfig{
		LinkKey: []string{"h"},
	})

	collisions := ValidateCollisions(m)
	require.Len(t, collisions, 1)
	require.Contains(t, collisions[0], `preview bracket chord second key "h"`)
	require.Contains(t, collisions[0], "heading_jump_key")
	require.Contains(t, collisions[0], "link_key")
}

func TestValidateCollisionsDetectsTodoChordConflicts(t *testing.T) {
	m := DefaultMap()

	ApplyConfig(&m, config.KeysConfig{
		TodoAdd: []string{"t"},
	})

	collisions := ValidateCollisions(m)
	require.Len(t, collisions, 1)
	require.Contains(t, collisions[0], `todo chord second key "t"`)
	require.Contains(t, collisions[0], "todo_key")
	require.Contains(t, collisions[0], "todo_add")
}

func TestValidateCollisionsAllowsFollowLinkToShareLinkKey(t *testing.T) {
	m := DefaultMap()
	require.Empty(t, ValidateCollisions(m))

	ApplyConfig(&m, config.KeysConfig{
		LinkKey:    []string{"x"},
		FollowLink: []string{"x"},
	})

	require.Empty(t, ValidateCollisions(m))
}

func requireHelpEntry(t *testing.T, entries []HelpEntry, section, key, desc string) {
	t.Helper()

	for _, entry := range entries {
		if entry.Section == section && entry.Key == key && entry.Desc == desc {
			return
		}
	}
	require.Failf(t, "missing help entry", "section=%q key=%q desc=%q", section, key, desc)
}
