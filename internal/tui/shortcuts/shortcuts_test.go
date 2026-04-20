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
	requireHelpEntry(t, entries, "Global", m.ShowThemePicker.Help().Key, "Open theme picker (live preview; saves theme.name only)")
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

func requireHelpEntry(t *testing.T, entries []HelpEntry, section, key, desc string) {
	t.Helper()

	for _, entry := range entries {
		if entry.Section == section && entry.Key == key && entry.Desc == desc {
			return
		}
	}
	require.Failf(t, "missing help entry", "section=%q key=%q desc=%q", section, key, desc)
}
