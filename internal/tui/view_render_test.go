package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderStatusContainsMessage(t *testing.T) {
	m := newTestModel(t)
	m.status = "hello test message"
	rendered := m.renderStatus()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "hello test message") {
		require.Failf(t, "assertion failed", "expected status to contain message, got %q", plain)
	}
}

func TestRenderModeSegmentHelp(t *testing.T) {
	m := newTestModel(t)
	m.showDashboard = false
	m.showHelp = true
	rendered := m.renderModeSegment()
	if rendered != "HELP" {
		require.Failf(t, "assertion failed", "expected 'HELP' mode segment, got %q", rendered)
	}
}

func TestRenderModeSegmentNotes(t *testing.T) {
	m := newTestModel(t)
	m.showDashboard = false
	m.showHelp = false
	m.listMode = listModeNotes
	rendered := m.renderModeSegment()
	if rendered == "" {
		require.FailNow(t, "expected non-empty mode segment in notes mode")
	}
}

func TestRenderModeSegmentTemporary(t *testing.T) {
	m := newTestModel(t)
	m.showDashboard = false
	m.listMode = listModeTemporary
	rendered := m.renderModeSegment()
	if !strings.Contains(strings.ToLower(rendered), "temp") {
		require.Failf(t, "assertion failed", "expected mode segment to indicate temporary mode, got %q", rendered)
	}
}

func TestRenderModeSegmentPins(t *testing.T) {
	m := newTestModel(t)
	m.showDashboard = false
	m.listMode = listModePins
	rendered := m.renderModeSegment()
	if !strings.Contains(strings.ToLower(rendered), "pin") {
		require.Failf(t, "assertion failed", "expected mode segment to indicate pins mode, got %q", rendered)
	}
}

func TestRenderModeSegmentDashboard(t *testing.T) {
	m := newTestModel(t)
	m.showDashboard = true
	rendered := m.renderModeSegment()
	if rendered != "DASHBOARD" {
		require.Failf(t, "assertion failed", "expected 'DASHBOARD' mode segment, got %q", rendered)
	}
}

func TestRenderModeSegmentSearch(t *testing.T) {
	m := newTestModel(t)
	m.showDashboard = false
	m.searchMode = true
	m.listMode = listModeNotes
	rendered := m.renderModeSegment()
	if rendered != "SEARCH" {
		require.Failf(t, "assertion failed", "expected 'SEARCH' mode segment, got %q", rendered)
	}
}

func TestRenderModeSegmentPreviewFocus(t *testing.T) {
	m := newTestModel(t)
	m.showDashboard = false
	m.focus = focusPreview
	rendered := m.renderModeSegment()
	if rendered != "PREVIEW" {
		require.Failf(t, "assertion failed", "expected 'PREVIEW' mode segment, got %q", rendered)
	}
}

func TestRenderFilterSegmentActive(t *testing.T) {
	m := newTestModel(t)
	m.searchMode = false
	m.searchInput.SetValue("myquery")
	rendered := m.renderFilterSegment()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "myquery") {
		require.Failf(t, "assertion failed", "expected filter segment to contain search query, got %q", plain)
	}
}

func TestRenderFilterSegmentInactive(t *testing.T) {
	m := newTestModel(t)
	m.searchMode = false
	m.searchInput.SetValue("")
	rendered := m.renderFilterSegment()
	plain := stripANSI(rendered)
	if plain != "" {
		require.Failf(t, "assertion failed", "expected empty filter segment when no query, got %q", plain)
	}
}

func TestRenderHelpModalContainsEntries(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 50
	m.showHelp = true
	m.helpInput.SetValue("")
	m.rebuildHelpRowsCache()

	// Scroll to the end so the Global section (with Quit) is visible
	maxRows := max(8, min(20, m.height-16))
	m.helpScroll = m.maxHelpScroll(maxRows)
	rendered := m.renderHelpModal()
	plain := stripANSI(rendered)

	// "Quit" is in the Global section - visible at the bottom
	if !strings.Contains(plain, "Quit") {
		assert.Failf(t, "assertion failed", "expected help modal to contain 'Quit' when scrolled to end")
	}
}

func TestRenderHelpModalContainsTreeEntries(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 50
	m.showHelp = true
	m.helpInput.SetValue("")
	m.rebuildHelpRowsCache()

	// At scroll=0, Tree/Filter sections are visible
	m.helpScroll = 0
	rendered := m.renderHelpModal()
	plain := stripANSI(rendered)

	// Tree section is near the top
	if !strings.Contains(plain, "Tree") {
		assert.Failf(t, "assertion failed", "expected help modal to contain 'Tree' section at scroll=0")
	}
}

func TestRenderHelpModalFilteredShowsMatch(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 50
	m.showHelp = true
	m.helpInput.SetValue("quit")
	m.rebuildHelpRowsCache()

	rendered := m.renderHelpModal()
	plain := stripANSI(rendered)

	if !strings.Contains(strings.ToLower(plain), "quit") {
		require.Failf(t, "assertion failed", "expected filtered help to contain 'quit'")
	}
}

func TestRenderHelpModalNoMatchShowsPlaceholder(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 50
	m.showHelp = true
	m.helpInput.SetValue("xyzzynotacommand")
	m.rebuildHelpRowsCache()

	rendered := m.renderHelpModal()
	plain := stripANSI(rendered)

	if !strings.Contains(plain, "No matching commands") {
		require.Failf(t, "assertion failed", "expected 'No matching commands' placeholder, got (first 300): %q", plain[:min(len(plain), 300)])
	}
}

func TestRenderHelpSectionTitle(t *testing.T) {
	m := newTestModel(t)
	_, innerWidth := m.modalDimensions(60, 96)
	rendered := m.renderHelpSectionTitle("TestSection", innerWidth)
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "TestSection") {
		require.Failf(t, "assertion failed", "expected section title to contain 'TestSection', got %q", plain)
	}
}

func TestRenderSearchBar(t *testing.T) {
	m := newTestModel(t)
	// Should not panic and should return a string
	rendered := m.renderSearchBar()
	_ = rendered
}

func TestRenderSearchBarActive(t *testing.T) {
	m := newTestModel(t)
	m.searchMode = true
	m.searchInput.Focus()
	m.searchInput.SetValue("notes")
	rendered := m.renderSearchBar()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "notes") {
		require.Failf(t, "assertion failed", "expected search bar to contain query text, got %q", plain)
	}
}

func TestRenderSortSegmentModified(t *testing.T) {
	m := Model{sortByModTime: true}
	if got := m.renderSortSegment(); got != "sort: modified" {
		require.Failf(t, "assertion failed", "expected 'sort: modified', got %q", got)
	}
}

func TestRenderSortSegmentAlpha(t *testing.T) {
	m := Model{sortByModTime: false}
	if got := m.renderSortSegment(); got != "sort: alpha" {
		require.Failf(t, "assertion failed", "expected 'sort: alpha', got %q", got)
	}
}
