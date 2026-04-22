package tui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
)

func typeThemePickerFilter(t *testing.T, m Model, query string) Model {
	t.Helper()
	for _, r := range query {
		m = updateModel(m, keyMsg(string(r)))
	}
	return m
}

func TestThemePickerOpensOnKey(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("ctrl+y"))
	require.True(t, m.showThemePicker, "expected showThemePicker after ctrl+y")
}

func TestThemePickerCursorStartsOnCurrentTheme(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Theme.Name = "dracula"
	m = updateModel(m, keyMsg("ctrl+y"))

	themes := BuiltinThemes()
	require.Equal(t, "dracula", themes[m.themePickerCursor].Name,
		"cursor should land on the currently configured theme")
}

func TestThemePickerCursorDefaultsToFirstWhenThemeUnrecognised(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Theme.Name = "nonexistent"
	m = updateModel(m, keyMsg("ctrl+y"))
	require.Equal(t, 0, m.themePickerCursor,
		"cursor should default to index 0 for an unrecognised theme name")
}

func TestThemePickerEscClosesModalAndRestoresTheme(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Theme.Name = "nord"
	ApplyTheme(m.cfg)

	m = updateModel(m, keyMsg("ctrl+y"))
	require.True(t, m.showThemePicker)

	// Navigate away to change the live preview theme.
	m = updateModel(m, keyMsg("j"))
	require.True(t, m.showThemePicker)

	// ESC should restore the original theme and close the modal.
	m = updateModel(m, keyMsg("esc"))
	require.False(t, m.showThemePicker, "modal should be closed after esc")
	require.Equal(t, "nord", m.cfg.Theme.Name,
		"cfg.Theme.Name must not be mutated by cancel")
	require.Equal(t, "theme unchanged", m.status)
}

func TestThemePickerNavigationMovesCursor(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("ctrl+y"))
	initial := m.themePickerCursor

	m = updateModel(m, keyMsg("j"))
	require.Equal(t, initial+1, m.themePickerCursor, "j should move cursor down")

	m = updateModel(m, keyMsg("k"))
	require.Equal(t, initial, m.themePickerCursor, "k should move cursor back up")
}

func TestThemePickerNavigationWraps(t *testing.T) {
	m := newTestModel(t)
	// Position cursor at the first theme.
	m.cfg.Theme.Name = BuiltinThemes()[0].Name
	m = updateModel(m, keyMsg("ctrl+y"))
	require.Equal(t, 0, m.themePickerCursor)

	// k from first should wrap to last.
	m = updateModel(m, keyMsg("k"))
	require.Equal(t, len(BuiltinThemes())-1, m.themePickerCursor,
		"k from first theme should wrap to last")
}

func TestThemePickerEnterConfirmsTheme(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NOTEUI_CONFIG", dir+"/config.toml")

	m := newTestModel(t)
	m.cfg.Theme.Name = "default"
	m = updateModel(m, keyMsg("ctrl+y"))

	// Navigate to "nord" (index 1 in BuiltinThemes).
	themes := BuiltinThemes()
	nordIdx := -1
	for i, th := range themes {
		if th.Name == "nord" {
			nordIdx = i
			break
		}
	}
	require.NotEqual(t, -1, nordIdx)

	for m.themePickerCursor != nordIdx {
		m = updateModel(m, keyMsg("j"))
	}

	m = updateModel(m, keyMsg("enter"))
	require.False(t, m.showThemePicker, "modal should be closed after enter")
	require.Equal(t, "nord", m.cfg.Theme.Name,
		"cfg.Theme.Name should be updated to the selected theme")

	// Config file should reflect the change.
	saved, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "nord", saved.Theme.Name)
}

func TestThemePickerScrollOffsetCentersOnOpen(t *testing.T) {
	m := newTestModel(t)
	themes := BuiltinThemes()
	// Pick a theme near the end so centering is non-trivial.
	last := themes[len(themes)-1]
	m.cfg.Theme.Name = last.Name
	m = updateModel(m, keyMsg("ctrl+y"))

	require.Equal(t, len(themes)-1, m.themePickerCursor,
		"cursor should be on the last theme")
	require.Greater(t, m.themePickerScrollOffset, 0,
		"scroll offset should be positive when cursor is near the end")
}

func TestThemePickerScrollOffsetUpdatesOnNavigate(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Theme.Name = BuiltinThemes()[0].Name
	m = updateModel(m, keyMsg("ctrl+y"))
	require.Equal(t, 0, m.themePickerScrollOffset)

	// Navigate past the visible window.
	for range themePickerVisible + 1 {
		m = updateModel(m, keyMsg("j"))
	}
	require.Greater(t, m.themePickerScrollOffset, 0,
		"scroll offset should increase as cursor moves past visible window")
}

func TestThemePickerDoesNotMutateOrigThemeOnCancel(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Theme.Name = "gruvbox"
	origCfg := m.cfg

	m = updateModel(m, keyMsg("ctrl+y"))
	// Move cursor several times.
	for range 5 {
		m = updateModel(m, keyMsg("j"))
	}
	m = updateModel(m, keyMsg("esc"))

	require.Equal(t, origCfg.Theme.Name, m.cfg.Theme.Name,
		"cfg.Theme.Name must not change after cancel")
}

func TestThemePickerTabTogglesFilterFocus(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("ctrl+y"))
	require.False(t, m.themePickerInput.Focused())

	m = updateModel(m, keyMsg("tab"))
	require.True(t, m.themePickerInput.Focused())

	m = updateModel(m, keyMsg("tab"))
	require.False(t, m.themePickerInput.Focused())
}

func TestThemePickerFilterMatchesAliases(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("ctrl+y"))
	m = updateModel(m, keyMsg("tab"))
	m = typeThemePickerFilter(t, m, "rosepine")

	filtered := m.filteredThemePickerIndices()
	require.Len(t, filtered, 1, "alias filter should narrow to a single theme")
	require.Equal(t, "rose-pine", BuiltinThemes()[filtered[0]].Name)
	require.Equal(t, filtered[0], m.themePickerCursor, "cursor should move to the first matching theme")
}

func TestThemePickerFocusedFilterUsesArrowKeysForNavigation(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("ctrl+y"))
	start := m.themePickerCursor

	m = updateModel(m, keyMsg("tab"))
	require.True(t, m.themePickerInput.Focused())

	m = updateModel(m, keyMsg("down"))
	require.Equal(t, start+1, m.themePickerCursor, "down should navigate even while filter is focused")

	current := m.themePickerCursor
	m = updateModel(m, keyMsg("n"))
	require.Equal(t, current, m.themePickerCursor, "typing should update the filter without treating the key as list navigation")
	require.Equal(t, "n", m.themePickerInput.Value())
}

func TestThemePickerKeyBlockedWhenOpen(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("ctrl+y"))
	require.True(t, m.showThemePicker)

	// Keys that would normally do things (e.g. new note) should be swallowed.
	m = updateModel(m, keyMsg("n"))
	require.True(t, m.showThemePicker,
		"unrelated keys should not close the theme picker or trigger actions")
}

func TestThemePickerEnterKeepsModalOpenWhenFilterHasNoMatches(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("ctrl+y"))
	m = updateModel(m, keyMsg("tab"))
	m = typeThemePickerFilter(t, m, "zzzzzz")

	m = updateModel(m, keyMsg("enter"))
	require.True(t, m.showThemePicker, "enter should not close the picker when there are no matching themes")
	require.Equal(t, "no matching themes", m.status)
}

func TestRenderThemePickerModalContainsThemeNames(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	m = updateModel(m, keyMsg("ctrl+y"))

	rendered := stripANSI(m.renderThemePickerModal())
	require.Contains(t, rendered, "Theme Picker")
	// The hovered theme name should appear in the header.
	require.Contains(t, rendered, BuiltinThemes()[m.themePickerCursor].Name)
	require.Contains(t, rendered, "Filter:")
	require.Contains(t, rendered, "theme.name")
}

func TestRenderThemePickerModalShowsThemeGuardrails(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	m.cfg.Theme.TextColor = "#777777"
	m.cfg.Theme.PanelBgColor = "#777777"
	m = updateModel(m, keyMsg("ctrl+y"))

	rendered := stripANSI(m.renderThemePickerModal())
	require.Contains(t, rendered, "Preview keeps current theme color overrides")
	require.Contains(t, rendered, "Low-contrast colors auto-adjusted for readability")
}
