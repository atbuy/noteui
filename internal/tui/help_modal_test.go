package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	cfg := config.Default()
	cfg.Dashboard = false
	return New(t.TempDir(), "", cfg, "test")
}

func TestFilteredHelpSectionsAllReturned(t *testing.T) {
	m := newTestModel(t)
	sections := m.filteredHelpSections()
	if len(sections) == 0 {
		require.FailNow(t, "expected non-empty sections for empty query")
	}
	names := make(map[string]bool)
	for _, s := range sections {
		names[s.title] = true
	}
	for _, want := range []string{"Filter", "Tree", "Notes", "Preview", "Global"} {
		if !names[want] {
			assert.Failf(t, "assertion failed", "expected section %q to be present", want)
		}
	}
}

func TestFilteredHelpSectionsFiltersQuery(t *testing.T) {
	m := newTestModel(t)
	m.helpInput.SetValue("scroll")
	sections := m.filteredHelpSections()
	for _, s := range sections {
		for _, e := range s.entries {
			blob := strings.ToLower(s.title + " " + e.key + " " + e.desc)
			if !strings.Contains(blob, "scroll") {
				assert.Failf(t, "assertion failed", "entry %q/%q should not appear for query 'scroll'", s.title, e.desc)
			}
		}
	}
	if len(sections) == 0 {
		require.FailNow(t, "expected at least one section to match 'scroll'")
	}
}

func TestFilteredHelpSectionsNoMatch(t *testing.T) {
	m := newTestModel(t)
	m.helpInput.SetValue("xyzzynotacommand")
	sections := m.filteredHelpSections()
	if len(sections) != 0 {
		require.Failf(t, "assertion failed", "expected empty sections for nonsense query, got %d", len(sections))
	}
}

func TestFilteredHelpSectionsSectionOrder(t *testing.T) {
	m := newTestModel(t)
	sections := m.filteredHelpSections()
	order := []string{"Filter", "Tree", "Notes", "Todos", "Preview", "Global"}
	orderIdx := 0
	for _, s := range sections {
		for orderIdx < len(order) && order[orderIdx] != s.title {
			orderIdx++
		}
		if orderIdx >= len(order) {
			assert.Failf(t, "assertion failed", "section %q appeared out of expected order", s.title)
		}
	}
}

func TestHelpRowCountMatchesManualCalc(t *testing.T) {
	m := newTestModel(t)
	count := m.helpRowCount()
	sections := m.filteredHelpSections()
	expected := len(sections) - 1 // blank separators between sections
	for _, s := range sections {
		expected += 1 + len(s.entries) // title + entries
	}
	if count != expected {
		require.Failf(t, "assertion failed", "helpRowCount = %d, want %d", count, expected)
	}
}

func TestHelpRowCountNoMatch(t *testing.T) {
	m := newTestModel(t)
	m.helpInput.SetValue("xyzzynotacommand")
	count := m.helpRowCount()
	if count != 1 {
		require.Failf(t, "assertion failed", "expected 1 placeholder row when no match, got %d", count)
	}
}

func TestMoveHelpScrollDelta(t *testing.T) {
	m := newTestModel(t)
	m.helpScroll = 0
	// Use small maxRows so there's room to scroll
	maxRows := 3
	changed := m.moveHelpScroll(1, maxRows)
	if !changed {
		require.FailNow(t, "expected scroll to change with small maxRows")
	}
	if m.helpScroll <= 0 {
		require.Failf(t, "assertion failed", "expected scroll to increase, got %d", m.helpScroll)
	}
}

func TestMoveHelpScrollClampsLow(t *testing.T) {
	m := newTestModel(t)
	m.helpScroll = 0
	changed := m.moveHelpScroll(-5, 100)
	if changed || m.helpScroll != 0 {
		require.Failf(t, "assertion failed", "expected no change at min scroll, got %d (changed=%v)", m.helpScroll, changed)
	}
}

func TestMoveHelpScrollClampsHigh(t *testing.T) {
	m := newTestModel(t)
	maxRows := 3
	totalRows := m.helpRowCount()
	maxScroll := max(0, totalRows-maxRows)
	m.helpScroll = maxScroll
	changed := m.moveHelpScroll(1000, maxRows)
	if changed {
		require.Failf(t, "assertion failed", "expected no change when already at max scroll (scroll=%d, maxScroll=%d)", m.helpScroll, maxScroll)
	}
}

func TestClampHelpScrollZeroMaxRows(t *testing.T) {
	m := newTestModel(t)
	m.helpScroll = 10
	m.clampHelpScroll(0)
	if m.helpScroll != 0 {
		require.Failf(t, "assertion failed", "expected scroll reset to 0 with maxRows=0, got %d", m.helpScroll)
	}
}

func TestIsMouseEscapeFragmentMultiRune(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[64;127;31M"), Paste: false}
	if !isMouseEscapeFragment(msg) {
		require.FailNow(t, "expected multi-rune non-paste KeyRunes to be detected as escape fragment")
	}
}

func TestIsMouseEscapeFragmentSingleRune(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a"), Paste: false}
	if isMouseEscapeFragment(msg) {
		require.FailNow(t, "expected single-rune KeyRunes not to be escape fragment")
	}
}

func TestIsMouseEscapeFragmentPasteMultiRune(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hello world"), Paste: true}
	if isMouseEscapeFragment(msg) {
		require.FailNow(t, "expected pasted multi-rune KeyRunes not to be escape fragment")
	}
}

func TestIsMouseEscapeFragmentNonRunes(t *testing.T) {
	for _, msgType := range []tea.KeyType{tea.KeyUp, tea.KeyDown, tea.KeyEsc} {
		msg := tea.KeyMsg{Type: msgType}
		if isMouseEscapeFragment(msg) {
			require.Failf(t, "assertion failed", "expected key type %v not to be escape fragment", msgType)
		}
	}
}

func TestShouldUpdateHelpInputAcceptsLetters(t *testing.T) {
	input := textinput.New()
	for _, r := range []rune{'a', 'z', 'A', 'Z', '0', '9'} {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		if !shouldUpdateHelpInput(msg, input) {
			assert.Failf(t, "assertion failed", "expected rune %q to be accepted", r)
		}
	}
}

func TestShouldUpdateHelpInputRejectsBracket(t *testing.T) {
	input := textinput.New()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
	if shouldUpdateHelpInput(msg, input) {
		require.FailNow(t, "expected single '[' rune to be rejected (escape sequence start)")
	}
}

func TestShouldUpdateHelpInputRejectsMouseFragment(t *testing.T) {
	input := textinput.New()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[64;127;31M"), Paste: false}
	if shouldUpdateHelpInput(msg, input) {
		require.FailNow(t, "expected multi-rune KeyRunes to be rejected")
	}
}

func TestShouldUpdateHelpInputAcceptsPaste(t *testing.T) {
	input := textinput.New()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hello world"), Paste: true}
	if !shouldUpdateHelpInput(msg, input) {
		require.FailNow(t, "expected paste event to be accepted")
	}
}

// alt+z is not a recognized editing key so it should be rejected by the Alt check.
func TestShouldUpdateHelpInputRejectsNonEditAlt(t *testing.T) {
	input := textinput.New()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}, Alt: true}
	if shouldUpdateHelpInput(msg, input) {
		require.FailNow(t, "expected non-edit Alt key to be rejected")
	}
}

func TestShouldUpdateHelpInputRejectsControlChars(t *testing.T) {
	input := textinput.New()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\x01'}} // ctrl-A as rune
	if shouldUpdateHelpInput(msg, input) {
		require.FailNow(t, "expected control character rune to be rejected")
	}
}

func TestRebuildHelpRowsCache(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	if m.helpRowsCache != nil {
		require.FailNow(t, "expected nil cache before rebuild")
	}
	m.rebuildHelpRowsCache()
	if m.helpRowsCache == nil {
		require.FailNow(t, "expected non-nil cache after rebuild")
	}
	firstLen := len(m.helpRowsCache)

	// Second call with same state should be a no-op (cache hit)
	m.rebuildHelpRowsCache()
	if len(m.helpRowsCache) != firstLen {
		require.FailNow(t, "expected cache to be stable on second call")
	}
}

func TestRebuildHelpRowsCacheCacheInvalidatesOnQueryChange(t *testing.T) {
	m := newTestModel(t)
	m.rebuildHelpRowsCache()
	allRows := len(m.helpRowsCache)

	// Change filter to something with fewer results
	m.helpInput.SetValue("quit")
	m.rebuildHelpRowsCache()
	if len(m.helpRowsCache) >= allRows {
		require.Failf(t, "assertion failed", "expected fewer rows after filtering, got %d (was %d)", len(m.helpRowsCache), allRows)
	}
}

func TestRenderedHelpRowsNoMatch(t *testing.T) {
	m := newTestModel(t)
	_, innerWidth := m.modalDimensions(60, 96)
	m.helpInput.SetValue("xyzzynotacommand")
	rows := m.renderedHelpRows(innerWidth, false)
	if len(rows) != 1 {
		require.Failf(t, "assertion failed", "expected 1 placeholder row, got %d", len(rows))
	}
}

func TestRenderedHelpRowsContainsSections(t *testing.T) {
	m := newTestModel(t)
	_, innerWidth := m.modalDimensions(60, 96)
	rows := m.renderedHelpRows(innerWidth, false)
	// Should have multiple rows for all sections
	if len(rows) < 5 {
		require.Failf(t, "assertion failed", "expected at least 5 rows for all sections, got %d", len(rows))
	}
}

func TestHelpEntriesNotEmpty(t *testing.T) {
	m := newTestModel(t)
	entries := m.helpEntries()
	if len(entries) == 0 {
		require.FailNow(t, "expected non-empty help entries")
	}
	// Verify each entry has non-empty fields
	for _, e := range entries {
		if e.section == "" {
			assert.Failf(t, "assertion failed", "entry %q has empty section", e.desc)
		}
		if e.key == "" && e.desc == "" {
			assert.Failf(t, "assertion failed", "entry has both empty key and desc")
		}
	}
}

func TestHelpEntriesIncludeSyncImport(t *testing.T) {
	m := newTestModel(t)
	entries := m.helpEntries()
	for _, entry := range entries {
		if entry.key == keys.SyncImport.Help().Key && strings.Contains(strings.ToLower(entry.desc), "import") {
			return
		}
	}
	require.FailNow(t, "expected help entries to include sync import")
}
