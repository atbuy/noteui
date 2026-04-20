package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/notes"
	notesync "atbuy/noteui/internal/sync"
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

func TestRenderStatusUsesErrorStyleForSyncFailures(t *testing.T) {
	m := newTestModel(t)
	m.status = "sync failed: network down"
	line := strings.Join([]string{
		m.renderModeSegment(),
		m.renderFocusSegment(),
		m.renderSelectionSegment(),
		m.renderPrivacySegment(),
		m.renderSortSegment(),
		m.status,
	}, "  •  ")
	rendered := m.renderStatus()
	assert.Equal(t, statusErrStyle.Render(line), rendered)
}

func TestViewEmbedsEditorInPreviewWhenConfigured(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Preview.EditInPreview = true
	m = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	editorModel := NewEditorModel("", "draft.md", m.rootDir, "body", m.preview.Width, m.preview.Height, false, "", false)
	editorModel.markLoaded(editorHashContent(editorModel.Content()), time.Now())
	m.editorActive = true
	m.editorModel = &editorModel
	m.focus = focusPreview

	rendered := stripANSI(m.View())
	require.Contains(t, rendered, "Tree (")
	require.Contains(t, rendered, "Editor")
	require.Contains(t, rendered, "-- NORMAL -- draft.md")
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
	m := Model{sortMethod: sortModified}
	if got := m.renderSortSegment(); got != "sort: modified" {
		require.Failf(t, "assertion failed", "expected 'sort: modified', got %q", got)
	}
}

func TestRenderSortSegmentAlpha(t *testing.T) {
	m := Model{sortMethod: sortAlpha}
	if got := m.renderSortSegment(); got != "sort: alpha" {
		require.Failf(t, "assertion failed", "expected 'sort: alpha', got %q", got)
	}
}

func TestRenderSortSegmentReverse(t *testing.T) {
	m := Model{sortMethod: sortModified, sortReverse: true}
	got := m.renderSortSegment()
	if !strings.Contains(got, "modified") || !strings.Contains(got, "↑") {
		require.Failf(t, "assertion failed", "expected 'modified' and reverse marker in segment, got %q", got)
	}
}

func TestRenderStatusLineTreatsSyncImportFailureAsError(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.status = "sync import failed: remote unavailable"
	rendered := m.renderStatus()
	plain := stripANSI(rendered)
	require.Contains(t, plain, "sync import failed: remote unavailable")
}

func TestRenderStatusLineTreatsRemoteDeleteFailureAsError(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.status = "remote delete failed: remote unavailable"
	rendered := m.renderStatus()
	plain := stripANSI(rendered)
	require.Contains(t, plain, "remote delete failed: remote unavailable")
}

func TestRenderStatusShowsConflictHintForSelectedConflictedNote(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{Path: m.rootDir + "/work/note.md", RelPath: "work/note.md", Name: "note.md", TitleText: "Note", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", LastSyncAt: time.Now(), Conflict: &notesync.ConflictInfo{CopyPath: "work/note.conflict-20260403-120000.md", OccurredAt: time.Now()}},
	}
	rendered := m.renderStatus()
	plain := stripANSI(rendered)
	require.Contains(t, plain, "conflict: press O to resolve")
}

func TestRenderModeSegmentSyncDebug(t *testing.T) {
	m := newTestModel(t)
	m.showSyncDebugModal = true
	require.Equal(t, "SYNC DEBUG", m.renderModeSegment())
}

func TestRenderStatusShowsSyncDebugHintForSelectedErroredNote(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{Path: m.rootDir + "/work/note.md", RelPath: "work/note.md", Name: "note.md", TitleText: "Note", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", LastSyncError: "remote unavailable"},
	}

	rendered := m.renderStatus()
	plain := stripANSI(rendered)
	require.Contains(t, plain, "sync issue: press ctrl+e for details")
}

func TestRenderSyncDebugModalContainsRawError(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	n := notes.Note{Path: m.rootDir + "/work/note.md", RelPath: "work/note.md", Name: "note.md", TitleText: "Note", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", ID: "n1", LastSyncError: "dial tcp timeout", LastSyncAttemptAt: time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC)},
	}

	rendered := m.renderSyncDebugModal()
	plain := stripANSI(rendered)
	require.Contains(t, plain, "Sync details")
	require.Contains(t, plain, "dial tcp timeout")
	require.Contains(t, plain, "Remote ID")
	require.Contains(t, plain, "y copy detail")
}

func TestRenderConflictResolutionModalShowsBothSides(t *testing.T) {
	m := newTestModel(t)
	m.width = 140
	m.height = 42
	notePath := m.rootDir + "/work/note.md"
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, []byte("local body"), 0o644))
	conflictPath := m.rootDir + "/work/note.conflict-20260403-120000.md"
	require.NoError(t, os.WriteFile(conflictPath, []byte("remote body"), 0o644))
	n := notes.Note{Path: notePath, RelPath: "work/note.md", Name: "note.md", TitleText: "Note", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", ID: "n1", LastSyncAt: time.Now(), Conflict: &notesync.ConflictInfo{CopyPath: "work/note.conflict-20260403-120000.md", OccurredAt: time.Now()}},
	}

	rendered := m.renderSyncDebugModal()
	plain := stripANSI(rendered)
	require.Contains(t, plain, "Resolve conflict")
	require.Contains(t, plain, "Keep local")
	require.Contains(t, plain, "Keep remote")
	require.Contains(t, plain, "local body")
	require.Contains(t, plain, "remote body")
}

func TestRenderSearchBarShowsResultMeta(t *testing.T) {
	m := newTestModel(t)
	m.width = 220
	m.height = 40
	m.searchMode = true
	m.searchInput.Focus()
	m.searchInput.SetValue("notes")
	m.treeItems = []treeItem{
		{Kind: treeCategory, RelPath: "", Name: "/"},
		{Kind: treeNote, RelPath: "work/notes.md", Name: "Notes"},
	}
	rendered := m.renderSearchBar()
	plain := stripANSI(rendered)
	require.Contains(t, plain, "Search active")
	require.Contains(t, plain, "1 result")
	require.Contains(t, plain, "esc clears")
}

func TestRenderFilterSegmentShowsResultAndPreviewCounts(t *testing.T) {
	m := newTestModel(t)
	m.searchInput.SetValue("alpha")
	m.treeItems = []treeItem{
		{Kind: treeCategory, RelPath: "", Name: "/"},
		{Kind: treeNote, RelPath: "alpha.md", Name: "Alpha"},
		{Kind: treeNote, RelPath: "beta.md", Name: "Beta"},
	}
	m.previewPath = "/tmp/alpha.md"
	m.previewMatches = []previewMatch{{line: 1, occurrIdx: 0}, {line: 3, occurrIdx: 0}}
	rendered := m.renderFilterSegment()
	plain := stripANSI(rendered)
	require.Contains(t, plain, "filter: alpha")
	require.Contains(t, plain, "2 results")
	require.Contains(t, plain, "2 preview matches")
}

func TestRenderFocusSegmentPreview(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	require.Equal(t, "focus: preview", m.renderFocusSegment())
}

func TestRightPanelTitleFocusedShowsMatchCount(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	m.searchInput.SetValue("alpha")
	m.previewPath = "/tmp/alpha.md"
	m.previewMatches = []previewMatch{{line: 1, occurrIdx: 0}, {line: 2, occurrIdx: 0}}
	title := m.rightPanelTitle()
	require.Contains(t, title, "focused")
	require.Contains(t, title, "2 matches")
}

func TestRenderTreeEmptyStateIsActionable(t *testing.T) {
	m := newTestModel(t)
	m.treeItems = []treeItem{{Kind: treeCategory, RelPath: "", Name: "/"}}
	rendered := m.renderTreeView()
	plain := stripANSI(rendered)
	require.Contains(t, plain, "Press n")
	require.Contains(t, plain, "T")
}

func TestRenderFilterSegmentTagSearchSkipsPreviewCounts(t *testing.T) {
	m := newTestModel(t)
	m.searchInput.SetValue("#demo")
	m.treeItems = []treeItem{
		{Kind: treeCategory, RelPath: "", Name: "/"},
		{Kind: treeNote, RelPath: "alpha.md", Name: "Alpha"},
	}
	m.previewPath = "/tmp/alpha.md"
	m.previewMatches = []previewMatch{{line: 1, occurrIdx: 0}}
	plain := stripANSI(m.renderFilterSegment())
	require.Contains(t, plain, "1 result")
	require.NotContains(t, plain, "preview match")
}
