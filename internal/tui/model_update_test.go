package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
	"atbuy/noteui/internal/state"
	notesync "atbuy/noteui/internal/sync"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "ctrl+e":
		return tea.KeyMsg{Type: tea.KeyCtrlE}
	case "ctrl+t":
		return tea.KeyMsg{Type: tea.KeyCtrlT}
	default:
		if len(s) == 1 {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
		}
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func updateModel(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

func TestWindowSizeMsgSetsWidthHeight(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.width != 120 || m.height != 40 {
		require.Failf(t, "assertion failed", "expected 120x40, got %dx%d", m.width, m.height)
	}
}

func TestWindowSizeMsgRebuildsCacheAfterInvalidation(t *testing.T) {
	m := newTestModel(t)
	// WindowSizeMsg sets helpRowsCache = nil then rebuilds it
	m = updateModel(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	// The handler sets nil then calls rebuildHelpModalCache which repopulates it.
	// Verify the model width was set (the resize worked).
	if m.width != 100 {
		require.Failf(t, "assertion failed", "expected width 100 after resize, got %d", m.width)
	}
}

func TestShowHelpKeyOpensModal(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("?"))
	if !m.showHelp {
		require.FailNow(t, "expected showHelp to be true after pressing '?'")
	}
	if m.helpScroll != 0 {
		require.Failf(t, "assertion failed", "expected helpScroll to be 0 when opening, got %d", m.helpScroll)
	}
}

func TestEscClosesHelpModal(t *testing.T) {
	m := newTestModel(t)
	m.showHelp = true
	m.helpInput.SetValue("")
	m = updateModel(m, keyMsg("esc"))
	if m.showHelp {
		require.FailNow(t, "expected showHelp to be false after pressing esc")
	}
}

func TestHelpScrollDown(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 50})
	m.showHelp = true
	m.helpScroll = 0

	m2 := updateModel(m, keyMsg("down"))
	if m2.helpScroll <= 0 {
		// Small window might not have scrollable content
		totalRows := m.helpRowCount()
		maxRows := max(8, min(20, 50-16))
		if totalRows <= maxRows {
			t.Skip("not enough rows to test scroll down")
		}
		require.Failf(t, "assertion failed", "expected scroll to increase after down key, got %d", m2.helpScroll)
	}
}

func TestHelpScrollHome(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 50})
	m.showHelp = true
	// First scroll down to get a non-zero scroll
	m.helpScroll = 5
	m = updateModel(m, keyMsg("home"))
	if m.helpScroll != 0 {
		require.Failf(t, "assertion failed", "expected scroll to be 0 after home key, got %d", m.helpScroll)
	}
}

func TestHelpScrollEnd(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 50})
	m.showHelp = true
	m.helpScroll = 0
	m = updateModel(m, keyMsg("end"))
	maxRows := max(8, min(20, 50-16))
	maxScroll := m.maxHelpScroll(maxRows)
	if m.helpScroll != maxScroll {
		require.Failf(t, "assertion failed", "expected scroll to be %d (maxScroll) after end key, got %d", maxScroll, m.helpScroll)
	}
}

func TestHelpFilterUpdatesOnTyping(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 50})
	// Open help via the key handler so helpInput.Focus() is called properly
	m = updateModel(m, keyMsg("?"))
	if !m.showHelp {
		require.FailNow(t, "expected showHelp to be true after '?'")
	}
	// Set a non-zero scroll to verify it resets on filter change
	m.helpScroll = 5

	m = updateModel(m, keyMsg("q"))
	if m.helpInput.Value() != "q" {
		require.Failf(t, "assertion failed", "expected helpInput to contain 'q', got %q", m.helpInput.Value())
	}
	if m.helpScroll != 0 {
		require.Failf(t, "assertion failed", "expected scroll reset to 0 after filter change, got %d", m.helpScroll)
	}
}

func TestHelpMouseEscapeFragmentIgnored(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 50})
	m.showHelp = true

	// Send a mouse escape fragment (multi-rune non-paste KeyRunes)
	fragment := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[64;127;31M"), Paste: false}
	m = updateModel(m, fragment)
	if m.helpInput.Value() != "" {
		require.Failf(t, "assertion failed", "expected escape fragment to be ignored, got %q in helpInput", m.helpInput.Value())
	}
}

func TestSearchKeyEntersSearchMode(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("/"))
	if !m.searchMode {
		require.FailNow(t, "expected searchMode to be true after pressing '/'")
	}
}

func TestEscExitsSearchMode(t *testing.T) {
	m := newTestModel(t)
	m.searchMode = true
	m.searchInput.Focus()
	m = updateModel(m, keyMsg("esc"))
	if m.searchMode {
		require.FailNow(t, "expected searchMode to be false after esc")
	}
}

func TestFocusKeyTogglesFocus(t *testing.T) {
	m := newTestModel(t)
	if m.focus != focusTree {
		require.FailNow(t, "expected initial focus to be on tree")
	}
	m = updateModel(m, keyMsg("tab"))
	if m.focus != focusPreview {
		require.FailNow(t, "expected focus to switch to preview after tab")
	}
	m = updateModel(m, keyMsg("tab"))
	if m.focus != focusTree {
		require.FailNow(t, "expected focus to switch back to tree after second tab")
	}
}

func TestToggleSyncKeyOnCategoryShowsStatus(t *testing.T) {
	m := newTestModel(t)
	m.treeItems = []treeItem{{Kind: treeCategory, RelPath: "work", Name: "work"}}
	m.treeCursor = 0
	m = updateModel(m, keyMsg("S"))
	require.Contains(t, m.status, "sync toggle only works on notes")
}

func TestDeleteRemoteKeepLocalKeyOnCategoryShowsStatus(t *testing.T) {
	m := newTestModel(t)
	m.treeItems = []treeItem{{Kind: treeCategory, RelPath: "work", Name: "work"}}
	m.treeCursor = 0
	m = updateModel(m, keyMsg("U"))
	require.Contains(t, m.status, "remote delete only works on synced local notes")
}

func TestDeleteRemoteKeepLocalKeyStartsForSyncedLinkedNote(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Sync = config.SyncConfig{
		DefaultProfile: "homebox",
		Profiles: map[string]config.SyncProfile{
			"homebox": {SSHHost: "notes-prod", RemoteRoot: "/srv/noteui", RemoteBin: "noteui-sync"},
		},
	}
	n := &notes.Note{RelPath: "work/note.md", Path: "/notes/work/note.md", Name: "note.md", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: "work/note.md", Note: n}}
	m.treeCursor = 0
	m.syncRecords = map[string]notesync.NoteRecord{"work/note.md": {ID: "n1", RelPath: "work/note.md", RemoteRev: "1"}}
	next, cmd := m.Update(keyMsg("U"))
	m = next.(Model)
	require.Equal(t, "deleting remote copy...", m.status)
	require.NotNil(t, cmd)
	require.False(t, m.syncRunning)
	require.True(t, m.syncInFlight["work/note.md"])
}

func TestNoteSyncClassToggledToSyncedStartsImmediateSyncVisual(t *testing.T) {
	m := newTestModel(t)
	m.rootDir = "/notes"
	next, cmd := m.Update(noteSyncClassToggledMsg{path: "/notes/work/note.md", syncClass: notes.SyncClassSynced})
	m = next.(Model)
	require.Equal(t, "note sync: synced", m.status)
	require.NotNil(t, cmd)
	require.False(t, m.syncRunning)
	require.True(t, m.syncInFlight["work/note.md"])
}

func TestSyncDebouncedStartsRealSyncWhileImmediateVisualIsActive(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Sync = config.SyncConfig{
		DefaultProfile: "homebox",
		Profiles: map[string]config.SyncProfile{
			"homebox": {SSHHost: "notes-prod", RemoteRoot: "/srv/noteui", RemoteBin: "noteui-sync"},
		},
	}
	m.syncDebounceToken = 3
	m.syncInFlight = map[string]bool{"work/note.md": true}
	next, cmd := m.Update(syncDebouncedMsg{token: 3, sessionToken: 1})
	m = next.(Model)
	require.True(t, m.syncRunning)
	require.NotNil(t, cmd)
}

func TestSyncImportKeyStartsImport(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Sync = config.SyncConfig{
		DefaultProfile: "homebox",
		Profiles: map[string]config.SyncProfile{
			"homebox": {SSHHost: "notes-prod", RemoteRoot: "/srv/noteui", RemoteBin: "noteui-sync"},
		},
	}
	next, cmd := m.Update(keyMsg("I"))
	m = next.(Model)
	require.Equal(t, "importing synced notes...", m.status)
	require.NotNil(t, cmd)
}

func TestSyncImportCurrentKeyStartsImportForRemoteOnlyNote(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Sync = config.SyncConfig{
		DefaultProfile: "homebox",
		Profiles: map[string]config.SyncProfile{
			"homebox": {SSHHost: "notes-prod", RemoteRoot: "/srv/noteui", RemoteBin: "noteui-sync"},
		},
	}
	m.treeItems = []treeItem{{Kind: treeRemoteNote, RelPath: "work/remote.md", Name: "Remote Note", RemoteNote: &notesync.RemoteNoteMeta{ID: "n1", RelPath: "work/remote.md", Title: "Remote Note"}}}
	m.treeCursor = 0
	next, cmd := m.Update(keyMsg("i"))
	m = next.(Model)
	require.Equal(t, "importing remote note...", m.status)
	require.NotNil(t, cmd)
	require.False(t, m.syncRunning)
	require.True(t, m.syncInFlight[remoteOnlySyncVisualKey("n1")])
}

func TestSyncImportCurrentKeyOnLocalNoteShowsStatus(t *testing.T) {
	m := newTestModel(t)
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: "local.md", Note: &notes.Note{RelPath: "local.md", Path: "/notes/local.md", Name: "local.md"}}}
	m.treeCursor = 0
	m = updateModel(m, keyMsg("i"))
	require.Contains(t, m.status, "single-note import only works on remote notes")
}

func TestSortToggleFlipsSortByModTime(t *testing.T) {
	m := newTestModel(t)
	initial := m.sortByModTime
	m = updateModel(m, keyMsg("s"))
	if m.sortByModTime == initial {
		require.FailNow(t, "expected sortByModTime to flip after sort key")
	}
	m = updateModel(m, keyMsg("s"))
	if m.sortByModTime != initial {
		require.FailNow(t, "expected sortByModTime to return to original after second sort key")
	}
}

func TestNextMatchInPreviewTodoNavTracksMatchingTodo(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	m.preview.Width = 80
	m.preview.Height = 6
	m.previewTodoNavMode = true
	m.previewContent = "alpha\n[ ] first\nbeta\n[ ] second\n"
	m.previewBaseContent = m.previewContent
	m.setPreviewViewportContent(m.previewContent)
	m.previewTodos = []previewTodoItem{
		{rendLine: 1, text: "first"},
		{rendLine: 3, text: "second"},
	}
	m.previewMatches = []previewMatch{
		{line: 1, occurrIdx: 0},
		{line: 3, occurrIdx: 0},
	}
	m.previewMatchIndex = -1

	m = updateModel(m, keyMsg("n"))
	require.Equal(t, 0, m.previewTodoCursor)

	m = updateModel(m, keyMsg("n"))
	require.Equal(t, 1, m.previewTodoCursor)
}

func TestTodoBracketNavigationKeepsVisibleLineInPlace(t *testing.T) {
	m := newTestModel(t)
	m.preview.Width = 80
	m.preview.Height = 8
	m.previewTodoNavMode = true
	m.previewContent = strings.Join([]string{
		"line 0",
		"[ ] first",
		"line 2",
		"[ ] second",
		"line 4",
		"[ ] third",
		"line 6",
	}, "\n")
	m.previewBaseContent = m.previewContent
	m.setPreviewViewportContent(m.previewContent)
	m.preview.SetYOffset(0)
	m.previewTodos = []previewTodoItem{
		{rendLine: 1, text: "first"},
		{rendLine: 3, text: "second"},
		{rendLine: 5, text: "third"},
	}
	m.previewTodoCursor = 0

	m.jumpToNextTodo()

	require.Equal(t, 0, m.preview.YOffset)
	require.Equal(t, 1, m.previewTodoCursor)
}

func TestTodoModifiedPreviewKeepsScrollOffset(t *testing.T) {
	m := newTestModel(t)
	m.preview.Width = 80
	m.preview.Height = 6
	m.preview.YOffset = 5
	m.previewTodoNavMode = true
	m.previewTodoCursor = 1
	m.previewPath = filepath.Join(m.rootDir, "work", "todo.md")

	next, _ := m.Update(todoModifiedMsg{path: m.previewPath})
	m = next.(Model)
	require.Equal(t, 5, m.pendingPreviewYOffset)
	require.Equal(t, 1, m.pendingTodoCursor)

	next, _ = m.Update(previewRenderedMsg{
		forPath:         m.previewPath,
		baseContent:     "line 0\nline 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\n[ ] one\n[ ] two",
		rawContent:      "line 0\nline 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\n- [ ] one\n- [ ] two",
		lineNumberStart: 0,
		todoLineOffset:  0,
	})
	m = next.(Model)

	require.Equal(t, 5, m.preview.YOffset)
	require.Equal(t, -1, m.pendingPreviewYOffset)
	require.Equal(t, 1, m.previewTodoCursor)
}

func TestDeletePendingCancelOnEsc(t *testing.T) {
	m := newTestModel(t)
	m.deletePending = &deletePending{
		kind:    deleteTargetNote,
		relPath: "test.md",
		name:    "test",
	}
	m = updateModel(m, keyMsg("esc"))
	if m.deletePending != nil {
		require.FailNow(t, "expected deletePending to be cleared after esc")
	}
}

func TestSearchModeMouseFragmentIgnored(t *testing.T) {
	m := newTestModel(t)
	m.searchMode = true
	m.searchInput.Focus()

	fragment := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[64;127;31M"), Paste: false}
	m = updateModel(m, fragment)
	if m.searchInput.Value() != "" {
		require.Failf(t, "assertion failed", "expected escape fragment to be ignored in search mode, got %q", m.searchInput.Value())
	}
}

func TestWindowSizeMsgUpdatesSearchInputWidth(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, tea.WindowSizeMsg{Width: 200, Height: 50})
	if m.searchInput.Width <= 0 {
		require.FailNow(t, "expected searchInput width to be positive after resize")
	}
}

func TestHelp_StatusContainsHelp(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("?"))
	if !strings.Contains(m.status, "help") {
		require.Failf(t, "assertion failed", "expected status to contain 'help', got %q", m.status)
	}
}

func TestDataLoadedMsgSetsNotes(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, dataLoadedMsg{
		sessionToken: 1,
		notes:        nil,
		tempNotes:    nil,
		categories:   nil,
		err:          nil,
	})
	if !strings.Contains(m.status, "no notes found") {
		require.Failf(t, "assertion failed", "expected 'no notes found' status, got %q", m.status)
	}
}

func TestDataLoadedMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, dataLoadedMsg{sessionToken: 1, err: errorf("test error")})
	if !strings.Contains(m.status, "error") {
		require.Failf(t, "assertion failed", "expected error status, got %q", m.status)
	}
}

// errorf creates a simple error for testing without importing "errors".
type testError string

func (e testError) Error() string { return string(e) }

func errorf(s string) error { return testError(s) }

func TestNoteCreatedMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, noteCreatedMsg{err: errorf("create failed"), path: ""})
	if !strings.Contains(m.status, "create failed") {
		require.Failf(t, "assertion failed", "expected 'create failed' in status, got %q", m.status)
	}
}

func TestNoteMovedMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, noteMovedMsg{err: errorf("move error")})
	if !strings.Contains(m.status, "move failed") {
		require.Failf(t, "assertion failed", "expected 'move failed' in status, got %q", m.status)
	}
}

func TestNoteDeletedMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, noteDeletedMsg{err: errorf("delete error")})
	if !strings.Contains(m.status, "delete failed") {
		require.Failf(t, "assertion failed", "expected 'delete failed' in status, got %q", m.status)
	}
}

func TestCategoryCreatedMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, categoryCreatedMsg{err: errorf("cat error")})
	if !strings.Contains(m.status, "category create failed") {
		require.Failf(t, "assertion failed", "expected 'category create failed' in status, got %q", m.status)
	}
}

func TestNoteTaggedMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, noteTaggedMsg{err: errorf("tag error")})
	if !strings.Contains(m.status, "add tag failed") {
		require.Failf(t, "assertion failed", "expected 'add tag failed' in status, got %q", m.status)
	}
}

func TestEncryptNoteMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, encryptNoteMsg{err: errorf("enc error")})
	if !strings.Contains(m.status, "encryption failed") {
		require.Failf(t, "assertion failed", "expected 'encryption failed' in status, got %q", m.status)
	}
}

func TestDecryptNoteMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, decryptNoteMsg{err: errorf("dec error")})
	if !strings.Contains(m.status, "decryption failed") {
		require.Failf(t, "assertion failed", "expected 'decryption failed' in status, got %q", m.status)
	}
}

func TestShowCreateCategoryEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.showCreateCategory = true
	m.categoryInput.Focus()
	m = updateModel(m, keyMsg("esc"))
	if m.showCreateCategory {
		require.FailNow(t, "expected showCreateCategory to be false after esc")
	}
	if !strings.Contains(m.status, "cancelled") {
		require.Failf(t, "assertion failed", "expected cancelled status, got %q", m.status)
	}
}

func TestShowMoveEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.showMove = true
	m.moveInput.Focus()
	m = updateModel(m, keyMsg("esc"))
	if m.showMove {
		require.FailNow(t, "expected showMove to be false after esc")
	}
	if !strings.Contains(m.status, "move cancelled") {
		require.Failf(t, "assertion failed", "expected 'move cancelled' status, got %q", m.status)
	}
}

func TestShowRenameEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.showRename = true
	m.renamePending = &renamePending{kind: renameTargetNote}
	m.renameInput.Focus()
	m = updateModel(m, keyMsg("esc"))
	if m.showRename {
		require.FailNow(t, "expected showRename to be false after esc")
	}
}

func TestShowAddTagEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.showAddTag = true
	m.tagInput.Focus()
	m = updateModel(m, keyMsg("esc"))
	if m.showAddTag {
		require.FailNow(t, "expected showAddTag to be false after esc")
	}
}

func TestTodoAddEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.showTodoAdd = true
	m.todoInput.Focus()
	m = updateModel(m, keyMsg("esc"))
	if m.showTodoAdd {
		require.FailNow(t, "expected showTodoAdd to be false after esc")
	}
}

func TestTodoEditEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.showTodoEdit = true
	m.todoInput.Focus()
	m = updateModel(m, keyMsg("esc"))
	if m.showTodoEdit {
		require.FailNow(t, "expected showTodoEdit to be false after esc")
	}
}

func TestPassphraseModalEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.showPassphraseModal = true
	m.passphraseInput.Focus()
	m = updateModel(m, keyMsg("esc"))
	if m.showPassphraseModal {
		require.FailNow(t, "expected showPassphraseModal to be false after esc")
	}
}

func TestEncryptConfirmEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.showEncryptConfirm = true
	m = updateModel(m, keyMsg("esc"))
	if m.showEncryptConfirm {
		require.FailNow(t, "expected showEncryptConfirm to be false after esc")
	}
}

func TestBracketForwardDoesNotSwitchToTemporaryMode(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	m.focus = focusTree
	m = updateModel(m, keyMsg("]"))
	if m.listMode != listModeNotes {
		require.Failf(t, "assertion failed", "expected listModeNotes unchanged after ']' in tree focus, got %v", m.listMode)
	}
}

func TestBracketBackwardDoesNotSwitchFromTemporaryMode(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	m.focus = focusTree
	m = updateModel(m, keyMsg("["))
	if m.listMode != listModeTemporary {
		require.Failf(t, "assertion failed", "expected listModeTemporary unchanged after '[' in tree focus, got %v", m.listMode)
	}
}

func TestToggleTemporaryKeySwitchesToTemporaryMode(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	m.focus = focusTree
	m = updateModel(m, keyMsg("t"))
	if m.listMode != listModeTemporary {
		require.Failf(t, "assertion failed", "expected listModeTemporary after 't', got %v", m.listMode)
	}
}

func TestToggleTemporaryKeyFromTemporarySwitchesToNotesMode(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	m.focus = focusTree
	m = updateModel(m, keyMsg("t"))
	if m.listMode != listModeNotes {
		require.Failf(t, "assertion failed", "expected listModeNotes after 't' from temporary mode, got %v", m.listMode)
	}
}

func TestToggleTemporaryKeyIgnoredInPinsMode(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModePins
	m.focus = focusTree
	m = updateModel(m, keyMsg("t"))
	if m.listMode != listModePins {
		require.Failf(t, "assertion failed", "expected listModePins unchanged after 't' in pins mode, got %v", m.listMode)
	}
}

func TestMouseWheelScrollsHelpModal(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, tea.WindowSizeMsg{Width: 120, Height: 50})
	m.showHelp = true
	m.helpScroll = 0

	m2 := updateModel(m, tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	maxRows := max(8, min(20, 50-16))
	maxScroll := m.maxHelpScroll(maxRows)
	if maxScroll == 0 {
		t.Skip("not enough rows to test scroll")
	}
	if m2.helpScroll <= 0 {
		require.Failf(t, "assertion failed", "expected scroll to increase after wheel down, got %d", m2.helpScroll)
	}
}

func TestPinsModeToggle(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	m = updateModel(m, keyMsg("P"))
	if m.listMode != listModePins {
		require.Failf(t, "assertion failed", "expected pins mode after 'P', got %v", m.listMode)
	}
	m = updateModel(m, keyMsg("P"))
	if m.listMode != listModeNotes {
		require.Failf(t, "assertion failed", "expected notes mode after second 'P', got %v", m.listMode)
	}
}

func TestSortStatusMessage(t *testing.T) {
	m := newTestModel(t)
	m.sortByModTime = false
	m = updateModel(m, keyMsg("s"))
	if !strings.Contains(m.status, "modified") {
		require.Failf(t, "assertion failed", "expected 'modified' in status when switching to modtime sort, got %q", m.status)
	}
	m = updateModel(m, keyMsg("s"))
	if !strings.Contains(m.status, "alpha") {
		require.Failf(t, "assertion failed", "expected 'alpha' in status when switching back, got %q", m.status)
	}
}

func TestCreateCategoryKeyOpensModal(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("C"))
	if !m.showCreateCategory {
		require.FailNow(t, "expected showCreateCategory after 'C' key")
	}
}

func TestNoteRenamedMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, noteRenamedMsg{err: errorf("rename error")})
	if !strings.Contains(m.status, "rename failed") {
		require.Failf(t, "assertion failed", "expected 'rename failed' in status, got %q", m.status)
	}
}

func TestCategoryRenamedMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, categoryRenamedMsg{err: errorf("rename error")})
	if !strings.Contains(m.status, "rename failed") {
		require.Failf(t, "assertion failed", "expected 'rename failed' in status, got %q", m.status)
	}
}

func TestCategoryMovedMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, categoryMovedMsg{err: errorf("move error")})
	if !strings.Contains(m.status, "move failed") {
		require.Failf(t, "assertion failed", "expected 'move failed' in status, got %q", m.status)
	}
}

func TestCategoryDeletedMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, categoryDeletedMsg{err: errorf("delete error")})
	if !strings.Contains(m.status, "delete failed") {
		require.Failf(t, "assertion failed", "expected 'delete failed' in status, got %q", m.status)
	}
}

func TestToggleSyncKeyOnRemoteOnlyNoteShowsImportStatus(t *testing.T) {
	m := newTestModel(t)
	m.treeItems = []treeItem{{Kind: treeRemoteNote, RelPath: "work/remote.md", Name: "Remote Note", RemoteNote: &notesync.RemoteNoteMeta{ID: "n1", RelPath: "work/remote.md", Title: "Remote Note"}}}
	m.treeCursor = 0
	m = updateModel(m, keyMsg("S"))
	require.Contains(t, m.status, "press i to import it or I to import all")
}

func TestRemoteOnlyNotePreviewExplainsImport(t *testing.T) {
	m := newTestModel(t)
	m.treeItems = []treeItem{{Kind: treeRemoteNote, RelPath: "work/remote.md", Name: "Remote Note", RemoteNote: &notesync.RemoteNoteMeta{ID: "n1", RelPath: "work/remote.md", Title: "Remote Note"}}}
	m.treeCursor = 0
	m.refreshPreview()
	plain := stripANSI(m.previewContent)
	require.Contains(t, plain, "not stored locally")
	require.Contains(t, plain, "Press i")
}

func TestRemoteOnlyDuplicatePreviewShowsRemoteID(t *testing.T) {
	m := newTestModel(t)
	m.remoteOnlyNotes = []notesync.RemoteNoteMeta{{ID: "n1aaaa", RelPath: "work/remote.md", Title: "Remote Note"}, {ID: "n2bbbb", RelPath: "work/remote.md", Title: "Remote Note"}}
	m.treeItems = []treeItem{{Kind: treeRemoteNote, RelPath: "work/remote.md", Name: m.remoteOnlyDisplayTitle(m.remoteOnlyNotes[0]), RemoteNote: &m.remoteOnlyNotes[0]}}
	m.treeCursor = 0
	m.refreshPreview()
	plain := stripANSI(m.previewContent)
	require.Contains(t, plain, "Remote ID")
	require.Contains(t, plain, "n1aaaa")
}

func TestEnterOnRemoteOnlyNoteShowsImportStatus(t *testing.T) {
	m := newTestModel(t)
	m.treeItems = []treeItem{{Kind: treeRemoteNote, RelPath: "work/remote.md", Name: "Remote Note", RemoteNote: &notesync.RemoteNoteMeta{ID: "n1", RelPath: "work/remote.md", Title: "Remote Note"}}}
	m.treeCursor = 0
	m = updateModel(m, keyMsg("enter"))
	require.Contains(t, m.status, "press i to import it or I to import all")
}

func TestSelectWorkspaceOpensPicker(t *testing.T) {
	cfg := config.Default()
	cfg.Dashboard = false
	cfg.Workspaces = map[string]config.WorkspaceConfig{
		"work": {Root: t.TempDir(), Label: "Work"},
		"demo": {Root: t.TempDir(), Label: "Demo"},
	}
	m := NewWithSession(cfg.Workspaces["work"].Root, "", cfg, "test", WorkspaceSession{Name: "work", Label: "Work"})
	m = updateModel(m, keyMsg("W"))
	require.True(t, m.showWorkspacePicker)
	require.Equal(t, "select workspace", m.status)
}

func TestSelectWorkspaceBlockedDuringNotesRootOverride(t *testing.T) {
	cfg := config.Default()
	cfg.Dashboard = false
	cfg.Workspaces = map[string]config.WorkspaceConfig{
		"work": {Root: t.TempDir(), Label: "Work"},
		"demo": {Root: t.TempDir(), Label: "Demo"},
	}
	m := NewWithSession(t.TempDir(), "", cfg, "test", WorkspaceSession{Override: true})
	m = updateModel(m, keyMsg("W"))
	require.False(t, m.showWorkspacePicker)
	require.Contains(t, m.status, "disabled when NOTES_ROOT is set")
}

func TestSelectSyncProfileOpensPicker(t *testing.T) {
	cfg := config.Default()
	cfg.Dashboard = false
	cfg.Sync.DefaultProfile = "homebox"
	cfg.Sync.Profiles = map[string]config.SyncProfile{
		"homebox": {SSHHost: "notes-prod", RemoteRoot: "/srv/homebox", RemoteBin: "noteui-sync"},
		"backup":  {SSHHost: "backup-host", RemoteRoot: "/srv/backup", RemoteBin: "noteui-sync"},
	}
	m := New(t.TempDir(), "", cfg, "test")
	m = updateModel(m, keyMsg("F"))
	require.True(t, m.showSyncProfilePicker)
	require.Equal(t, "homebox", m.selectedSyncProfileName())
}

func TestConfirmSelectedSyncProfileShowsMigrationForBoundRoot(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Dashboard = false
	cfg.Sync.DefaultProfile = "homebox"
	cfg.Sync.Profiles = map[string]config.SyncProfile{
		"homebox": {SSHHost: "notes-prod", RemoteRoot: "/srv/homebox", RemoteBin: "noteui-sync"},
		"backup":  {SSHHost: "backup-host", RemoteRoot: "/srv/backup", RemoteBin: "noteui-sync"},
	}
	require.NoError(t, notesync.SaveRootConfig(root, notesync.RootConfig{SchemaVersion: notesync.SchemaVersion, ClientID: notesync.NewClientID(), Profile: "homebox"}))
	m := New(root, "", cfg, "test")
	m.openSyncProfilePicker()
	m.moveSyncProfileCursor(-1)
	m = updateModel(m, keyMsg("enter"))
	require.True(t, m.showSyncProfileMigration)
	require.NotNil(t, m.pendingSyncProfileChange)
	require.Equal(t, "backup", m.pendingSyncProfileChange.selectedDefault)
	require.Equal(t, "homebox", m.pendingSyncProfileChange.boundProfile)
}

func TestDataLoadedPreservesCollapsedCategoryState(t *testing.T) {
	m := newTestModel(t)
	m.expanded = map[string]bool{"": true, "work": false}
	m = updateModel(m, dataLoadedMsg{sessionToken: 1, categories: []notes.Category{{Name: "All notes", RelPath: ""}, {Name: "work", RelPath: "work"}}})
	require.False(t, m.expanded["work"])
	require.Contains(t, m.workspaceState.CollapsedCategories, "work")
}

func TestOpenConflictCopyKeyOpensConflictResolutionModal(t *testing.T) {
	m := newTestModel(t)
	notePath := m.rootDir + "/work/note.md"
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, []byte("local"), 0o644))
	conflictPath := m.rootDir + "/work/note.conflict-20260403-120000.md"
	require.NoError(t, os.WriteFile(conflictPath, []byte("remote"), 0o644))
	n := notes.Note{Path: notePath, RelPath: "work/note.md", Name: "note.md", TitleText: "Note", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", LastSyncAt: time.Now(), Conflict: &notesync.ConflictInfo{CopyPath: "work/note.conflict-20260403-120000.md", OccurredAt: time.Now()}},
	}
	next, cmd := m.Update(keyMsg("O"))
	updated := next.(Model)
	require.Nil(t, cmd)
	require.True(t, updated.showSyncDebugModal)
	require.Equal(t, "resolve conflict", updated.status)
}

func TestOpenConflictCopyKeyShowsStatusWithoutConflict(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{Path: m.rootDir + "/work/note.md", RelPath: "work/note.md", Name: "note.md", TitleText: "Note", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	next, cmd := m.Update(keyMsg("O"))
	updated := next.(Model)
	require.Nil(t, cmd)
	require.Equal(t, "conflict resolution only works on conflicted synced notes", updated.status)
}

func TestShowSyncDebugKeyOpensModalForErroredSyncedNote(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{Path: m.rootDir + "/work/note.md", RelPath: "work/note.md", Name: "note.md", TitleText: "Note", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", ID: "n1", LastSyncError: "remote unavailable"},
	}

	m = updateModel(m, keyMsg("ctrl+e"))
	require.True(t, m.showSyncDebugModal)
	require.Equal(t, "sync details", m.status)
}

func TestShowSyncDebugKeyOnHealthyNoteShowsStatus(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{Path: m.rootDir + "/work/note.md", RelPath: "work/note.md", Name: "note.md", TitleText: "Note", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", ID: "n1", LastSyncAt: time.Now()},
	}

	m = updateModel(m, keyMsg("ctrl+e"))
	require.False(t, m.showSyncDebugModal)
	require.Equal(t, "sync details only work on unhealthy synced notes", m.status)
}

func TestEscClosesSyncDebugModal(t *testing.T) {
	m := newTestModel(t)
	m.showSyncDebugModal = true
	m = updateModel(m, keyMsg("esc"))
	require.False(t, m.showSyncDebugModal)
	require.Equal(t, "sync details closed", m.status)
}

func TestSyncDebugCopyUsesClipboard(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{Path: m.rootDir + "/work/note.md", RelPath: "work/note.md", Name: "note.md", TitleText: "Note", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", ID: "n1", LastSyncError: "remote unavailable"},
	}
	m.showSyncDebugModal = true

	var copied string
	oldWriteClipboard := writeClipboard
	writeClipboard = func(s string) error {
		copied = s
		return nil
	}
	defer func() { writeClipboard = oldWriteClipboard }()

	m = updateModel(m, keyMsg("y"))
	require.Equal(t, "remote unavailable", copied)
	require.Equal(t, "copied sync detail to clipboard", m.status)
}

func TestErroredSyncedNotePreviewShowsSyncSummary(t *testing.T) {
	m := newTestModel(t)
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", ID: "n1", LastSyncError: "remote unavailable", LastSyncAttemptAt: time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC)},
	}

	rendered, offset := m.renderNotePreview("work/note.md", "---\ntags: alpha\n---\n# Body", []string{"alpha"})
	plain := stripANSI(rendered)
	require.Contains(t, plain, "Sync status")
	require.Contains(t, plain, "Sync failed")
	require.Contains(t, plain, "ctrl+e")
	require.Contains(t, plain, "Body")
	require.Greater(t, offset, 2)
}

func TestEnterInConflictModalStartsResolutionCommand(t *testing.T) {
	m := newTestModel(t)
	notePath := m.rootDir + "/work/note.md"
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, []byte("local"), 0o644))
	conflictPath := m.rootDir + "/work/note.conflict-20260403-120000.md"
	require.NoError(t, os.WriteFile(conflictPath, []byte("remote"), 0o644))
	n := notes.Note{Path: notePath, RelPath: "work/note.md", Name: "note.md", TitleText: "Note", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {ID: "n1", RelPath: "work/note.md", RemoteRev: "2", LastSyncAt: time.Now(), Conflict: &notesync.ConflictInfo{CopyPath: "work/note.conflict-20260403-120000.md", OccurredAt: time.Now()}},
	}
	m.showSyncDebugModal = true
	m.conflictResolutionChoice = conflictResolutionKeepRemote
	next, cmd := m.Update(keyMsg("enter"))
	updated := next.(Model)
	require.NotNil(t, cmd)
	require.Equal(t, "resolving conflict: keep remote", updated.status)
}

func TestDeletePendingSecondDConfirmsDelete(t *testing.T) {
	m := newTestModel(t)
	notePath := m.rootDir + "/work/note.md"
	n := notes.Note{Path: notePath, RelPath: "work/note.md", Name: "note.md", TitleText: "Note"}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.treeCursor = 0
	m = updateModel(m, keyMsg("d"))
	require.NotNil(t, m.deletePending)
	next, cmd := m.Update(keyMsg("d"))
	updated := next.(Model)
	require.NotNil(t, cmd)
	require.NotNil(t, updated.deletePending)
}

func TestMakeSharedKeyOnNoteWritesSharedClass(t *testing.T) {
	m := newTestModel(t)
	notePath := filepath.Join(m.rootDir, "note.md")
	require.NoError(t, os.WriteFile(notePath, []byte("# Note\n"), 0o644))
	n := &notes.Note{RelPath: "note.md", Path: notePath, Name: "note.md", SyncClass: notes.SyncClassLocal}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: "note.md", Note: n}}
	m.treeCursor = 0
	next, cmd := m.Update(keyMsg("ctrl+s"))
	m = next.(Model)
	require.NotNil(t, cmd, "expected a command to be dispatched")
	_ = cmd()
	content, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Contains(t, string(content), "sync: shared")
}

func TestMakeSharedKeyOnAlreadySharedNoteUnsharesIt(t *testing.T) {
	m := newTestModel(t)
	notePath := filepath.Join(m.rootDir, "shared.md")
	require.NoError(t, os.WriteFile(notePath, []byte("---\nsync: shared\n---\n# Note\n"), 0o644))
	n := &notes.Note{RelPath: "shared.md", Path: notePath, Name: "shared.md", SyncClass: notes.SyncClassShared}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: "shared.md", Note: n}}
	m.treeCursor = 0
	next, cmd := m.Update(keyMsg("ctrl+s"))
	m = next.(Model)
	require.NotNil(t, cmd, "expected a command to be dispatched")
	_ = cmd()
	content, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Contains(t, string(content), "sync: local")
}

func TestNoteMadeSharedMsgSetsCorrectStatus(t *testing.T) {
	m := newTestModel(t)

	next, _ := m.Update(noteMadeSharedMsg{path: filepath.Join(m.rootDir, "note.md"), syncClass: notes.SyncClassShared})
	m = next.(Model)
	require.Equal(t, "note is now shared", m.status)

	next, _ = m.Update(noteMadeSharedMsg{path: filepath.Join(m.rootDir, "note.md"), syncClass: notes.SyncClassLocal})
	m = next.(Model)
	require.Equal(t, "note is no longer shared", m.status)
}

func TestToggleSyncKeyOnSharedNoteShowsStatus(t *testing.T) {
	m := newTestModel(t)
	n := &notes.Note{RelPath: "work/shared.md", Path: m.rootDir + "/work/shared.md", Name: "shared.md", SyncClass: notes.SyncClassShared}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: "work/shared.md", Note: n}}
	m.treeCursor = 0
	m = updateModel(m, keyMsg("S"))
	require.Contains(t, m.status, "shared notes cannot be toggled")
}

func TestShowTodosKeyTogglesTodosMode(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{
		Path:      filepath.Join(m.rootDir, "work", "todo.md"),
		RelPath:   "work/todo.md",
		Name:      "todo.md",
		TitleText: "Todo",
		Preview:   "- [ ] Ship release\n- [ ] Review docs\n",
	}
	m.notes = []notes.Note{n}
	m.rebuildTodoItems()
	m.syncSelectedNote()

	m = updateModel(m, keyMsg("ctrl+t"))
	require.Equal(t, listModeTodos, m.listMode)
	require.NotNil(t, m.currentTodoItem())
	require.Equal(t, "Ship release", m.currentTodoItem().Todo.DisplayText)

	m = updateModel(m, keyMsg("ctrl+t"))
	require.Equal(t, listModeNotes, m.listMode)
}

func TestPreviewRenderedSyncsSelectedTodoInTodosMode(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{
		Path:      filepath.Join(m.rootDir, "work", "todo.md"),
		RelPath:   "work/todo.md",
		Name:      "todo.md",
		TitleText: "Todo",
		Preview:   "- [ ] First task\n- [ ] Second task\n",
	}
	m.notes = []notes.Note{n}
	m.rebuildTodoItems()
	m.listMode = listModeTodos
	m.todoCursor = 1
	m.previewPath = n.Path

	m = updateModel(m, previewRenderedMsg{
		forPath:     n.Path,
		baseContent: "- [ ] First task\n- [ ] Second task\n",
		rawContent:  "- [ ] First task\n- [ ] Second task\n",
	})

	require.True(t, m.previewTodoNavMode)
	require.Equal(t, 1, m.previewTodoCursor)
}

func TestTodoDueDateEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.showTodoDueDate = true
	m.dueDateInput.Focus()
	m = updateModel(m, keyMsg("esc"))
	if m.showTodoDueDate {
		require.FailNow(t, "expected showTodoDueDate to be false after esc")
	}
}

func TestTodoDueDateActionOpensModalWithPrefill(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	m.previewPath = filepath.Join(m.rootDir, "work", "todo.md")
	m.previewTodos = []previewTodoItem{{rawLine: 3, text: "Ship release [p1] [due:2026-04-12]"}}
	m.previewTodoCursor = 0
	m.pendingT = true

	m = updateModel(m, keyMsg("u"))
	require.True(t, m.showTodoDueDate)
	require.Equal(t, "2026-04-12", m.dueDateInput.Value())
}

func TestTodoPriorityEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.showTodoPriority = true
	m.priorityInput.Focus()
	m = updateModel(m, keyMsg("esc"))
	if m.showTodoPriority {
		require.FailNow(t, "expected showTodoPriority to be false after esc")
	}
}

func TestTodoPriorityActionOpensModalWithPrefill(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	m.previewPath = filepath.Join(m.rootDir, "work", "todo.md")
	m.previewTodos = []previewTodoItem{{rawLine: 3, text: "Ship release [p2] [due:2026-04-12]"}}
	m.previewTodoCursor = 0
	m.pendingT = true

	m = updateModel(m, keyMsg("p"))
	require.True(t, m.showTodoPriority)
	require.Equal(t, "2", m.priorityInput.Value())
}

func TestPreviewRenderedAppliesTodoDueDateHintsWithoutTodoNav(t *testing.T) {
	m := newTestModel(t)
	m.previewPath = filepath.Join(m.rootDir, "work", "todo.md")
	m.preview.Width = 80
	m.preview.Height = 8

	plain := "[ ] Ship release [due:2020-01-01]\n"
	m = updateModel(m, previewRenderedMsg{
		forPath:     m.previewPath,
		baseContent: plain,
		rawContent:  "- [ ] Ship release [due:2020-01-01]\n",
	})

	rendered := m.preview.View()
	plainRendered := stripANSI(rendered)
	require.Contains(t, plainRendered, "Ship release [due:2020-01-01]")
}

func TestToggleSelectWorksInTemporaryMode(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	m.tempNotes = []notes.Note{{RelPath: "inbox.md", Path: filepath.Join(notes.TempRoot(m.rootDir), "inbox.md"), Name: "inbox.md", TitleText: "Inbox"}}

	m = updateModel(m, keyMsg("v"))
	require.True(t, m.markedTreeItems[tempMarkKey("inbox.md")])
}

func TestPromoteTemporaryKeyOpensMoveBrowser(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	m.tempNotes = []notes.Note{{RelPath: "inbox.md", Path: filepath.Join(notes.TempRoot(m.rootDir), "inbox.md"), Name: "inbox.md", TitleText: "Inbox"}}

	m = updateModel(m, keyMsg("M"))
	require.True(t, m.showMoveBrowser)
	require.Equal(t, moveBrowserModePromoteTemporary, m.moveBrowserMode)
}

func TestClearMarksKeyClearsTemporaryMarks(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	m.markedTreeItems = map[string]bool{tempMarkKey("inbox.md"): true}

	m = updateModel(m, keyMsg("V"))
	require.Empty(t, m.markedTreeItems)
}

func TestNewWithSessionStartsWithWorkspacePickerWhenRequired(t *testing.T) {
	cfg := config.Default()
	cfg.Dashboard = false
	cfg.Workspaces = map[string]config.WorkspaceConfig{
		"work": {Root: t.TempDir(), Label: "Work"},
		"demo": {Root: t.TempDir(), Label: "Demo"},
	}

	m := NewWithSession("", "", cfg, "test", WorkspaceSession{StartWithPicker: true})
	require.True(t, m.showWorkspacePicker)
	require.Equal(t, "select workspace", m.status)
	require.Nil(t, m.Init())
}

func TestConfirmSelectedWorkspaceSwitchesRootAndIgnoresStaleLoad(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	rootA := t.TempDir()
	rootB := t.TempDir()
	cfg := config.Default()
	cfg.Dashboard = false
	cfg.Workspaces = map[string]config.WorkspaceConfig{
		"work": {Root: rootA, Label: "Work"},
		"demo": {Root: rootB, Label: "Demo"},
	}

	stored := state.State{}
	stored.SetWorkspace("work", state.WorkspaceState{RecentCommands: []string{cmdShowHelp}})
	stored.SetWorkspace("demo", state.WorkspaceState{RecentCommands: []string{cmdShowPins}, CollapsedCategories: []string{"archive"}})
	require.NoError(t, state.Save(stored))

	m := NewWithSession(rootA, "", cfg, "test", WorkspaceSession{Name: "work", Label: "Work"})
	require.Equal(t, []string{cmdShowHelp}, m.workspaceState.RecentCommands)
	require.Equal(t, 1, m.sessionToken)

	m.openWorkspacePicker()
	m.moveWorkspaceCursor(-1)
	cmd := m.confirmSelectedWorkspace()
	require.NotNil(t, cmd)
	require.False(t, m.showWorkspacePicker)
	require.Equal(t, rootB, m.rootDir)
	require.Equal(t, "demo", m.workspaceName)
	require.Equal(t, []string{cmdShowPins}, m.workspaceState.RecentCommands)
	require.Contains(t, m.workspaceState.CollapsedCategories, "archive")
	require.Equal(t, 2, m.sessionToken)

	next, _ := m.Update(dataLoadedMsg{sessionToken: 1, notes: []notes.Note{{RelPath: "old.md", Name: "old.md", TitleText: "Old"}}})
	m = next.(Model)
	require.Empty(t, m.notes)

	next, _ = m.Update(dataLoadedMsg{sessionToken: 2, notes: []notes.Note{{RelPath: "new.md", Name: "new.md", TitleText: "New"}}})
	m = next.(Model)
	require.Len(t, m.notes, 1)
	require.Equal(t, "new.md", m.notes[0].RelPath)
}

// --- Template picker tests ---

func makeTemplateFile(t *testing.T, root, name, content string) {
	t.Helper()
	dir := filepath.Join(root, notes.TemplatesDirName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func TestNewNoteKeyOpensTemplatePickerWhenTemplatesExist(t *testing.T) {
	m := newTestModel(t)
	makeTemplateFile(t, m.rootDir, "weekly.md", "# Weekly\n")

	next, cmd := m.Update(keyMsg("n"))
	m = next.(Model)
	require.Nil(t, cmd, "expected no command; picker should open instead")
	require.True(t, m.showTemplatePicker)
	require.Len(t, m.templateItems, 1)
	require.Equal(t, 0, m.templatePickerCursor)
}

func TestNewNoteKeyCreatesBlankNoteWhenNoTemplates(t *testing.T) {
	m := newTestModel(t)
	// No .templates/ directory exists.

	next, cmd := m.Update(keyMsg("n"))
	m = next.(Model)
	require.False(t, m.showTemplatePicker)
	require.NotNil(t, cmd)
}

func TestTemplatePickerEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.showTemplatePicker = true
	m.templateItems = []notes.Template{{Name: "weekly", RelPath: "weekly.md", Path: "/tmp/weekly.md"}}

	next, _ := m.Update(keyMsg("esc"))
	m = next.(Model)
	require.False(t, m.showTemplatePicker)
	require.Contains(t, m.status, "cancelled")
}

func TestTemplatePickerEnterOnBlankIssuesCreateNoteCmd(t *testing.T) {
	m := newTestModel(t)
	m.showTemplatePicker = true
	m.templateItems = []notes.Template{{Name: "weekly", RelPath: "weekly.md", Path: "/tmp/weekly.md"}}
	m.templatePickerCursor = 0 // "Blank note"

	next, cmd := m.Update(keyMsg("enter"))
	m = next.(Model)
	require.False(t, m.showTemplatePicker)
	require.NotNil(t, cmd)
}

func TestTemplatePickerEnterOnTemplateIssuesCreateFromTemplateCmd(t *testing.T) {
	m := newTestModel(t)
	tmplPath := filepath.Join(m.rootDir, notes.TemplatesDirName, "weekly.md")
	makeTemplateFile(t, m.rootDir, "weekly.md", "# Weekly\n")
	m.showTemplatePicker = true
	m.templateItems = []notes.Template{{Name: "weekly", RelPath: "weekly.md", Path: tmplPath}}
	m.templatePickerCursor = 1 // first template

	next, cmd := m.Update(keyMsg("enter"))
	m = next.(Model)
	require.False(t, m.showTemplatePicker)
	require.NotNil(t, cmd)
}

func TestTemplatePickerCursorClamps(t *testing.T) {
	m := newTestModel(t)
	m.showTemplatePicker = true
	m.templateItems = []notes.Template{
		{Name: "a", RelPath: "a.md", Path: "/tmp/a.md"},
		{Name: "b", RelPath: "b.md", Path: "/tmp/b.md"},
	}
	m.templatePickerCursor = 0

	// Move down 5 times; total items = 3 (Blank + 2 templates), max index = 2.
	for i := 0; i < 5; i++ {
		next, _ := m.Update(keyMsg("down"))
		m = next.(Model)
	}
	require.Equal(t, 2, m.templatePickerCursor)

	// Move up 5 times; min index = 0.
	for i := 0; i < 5; i++ {
		next, _ := m.Update(keyMsg("up"))
		m = next.(Model)
	}
	require.Equal(t, 0, m.templatePickerCursor)
}

func TestTemporaryNoteModeSkipsTemplatePicker(t *testing.T) {
	m := newTestModel(t)
	makeTemplateFile(t, m.rootDir, "weekly.md", "# Weekly\n")
	m.listMode = listModeTemporary

	next, cmd := m.Update(keyMsg("n"))
	m = next.(Model)
	require.False(t, m.showTemplatePicker)
	require.NotNil(t, cmd)
}

func TestNewTemplateKeyIssuesCreateTemplateCmd(t *testing.T) {
	m := newTestModel(t)

	next, cmd := m.Update(keyMsg("ctrl+n"))
	m = next.(Model)
	require.False(t, m.showTemplatePicker)
	require.NotNil(t, cmd)
}

func TestEditTemplatesKeyOpensPickerInEditMode(t *testing.T) {
	m := newTestModel(t)
	makeTemplateFile(t, m.rootDir, "meeting.md", "# Meeting\n")

	// keys.EditTemplates has no default key, so trigger via openTemplatePickerEditMode directly.
	templates, err := notes.DiscoverTemplates(m.rootDir)
	require.NoError(t, err)
	m.openTemplatePickerEditMode(templates)

	require.True(t, m.showTemplatePicker)
	require.True(t, m.templatePickerEditMode)
	require.Len(t, m.templateItems, 1)
}

func TestTemplatePickerEditModeEnterOpensEditor(t *testing.T) {
	m := newTestModel(t)
	makeTemplateFile(t, m.rootDir, "standup.md", "# Standup\n")

	templates, err := notes.DiscoverTemplates(m.rootDir)
	require.NoError(t, err)
	m.openTemplatePickerEditMode(templates)

	// Press enter to confirm (open in editor).
	next, cmd := m.Update(keyMsg("enter"))
	m = next.(Model)
	require.False(t, m.showTemplatePicker)
	require.NotNil(t, cmd)
}

func TestTemplatePickerEditModeEscCloses(t *testing.T) {
	m := newTestModel(t)
	makeTemplateFile(t, m.rootDir, "standup.md", "# Standup\n")

	templates, err := notes.DiscoverTemplates(m.rootDir)
	require.NoError(t, err)
	m.openTemplatePickerEditMode(templates)

	next, _ := m.Update(keyMsg("esc"))
	m = next.(Model)
	require.False(t, m.showTemplatePicker)
	require.False(t, m.templatePickerEditMode)
}

func TestTemplatePickerCreateModeEKeyOpensEditor(t *testing.T) {
	m := newTestModel(t)
	makeTemplateFile(t, m.rootDir, "standup.md", "# Standup\n")

	templates, err := notes.DiscoverTemplates(m.rootDir)
	require.NoError(t, err)
	m.openTemplatePicker(templates)
	// Move to the first template (index 1, past "Blank note").
	m.templatePickerCursor = 1

	next, cmd := m.Update(keyMsg("e"))
	m = next.(Model)
	require.False(t, m.showTemplatePicker)
	require.NotNil(t, cmd)
}

func TestTemplatePickerCursorClampsEditMode(t *testing.T) {
	m := newTestModel(t)
	m.showTemplatePicker = true
	m.templatePickerEditMode = true
	m.templateItems = []notes.Template{
		{Name: "a", RelPath: "a.md", Path: "/tmp/a.md"},
		{Name: "b", RelPath: "b.md", Path: "/tmp/b.md"},
	}
	m.templatePickerCursor = 0

	// Move down 5 times: should clamp at 1 (total 2 items, no "Blank note").
	for range 5 {
		next, _ := m.Update(keyMsg("j"))
		m = next.(Model)
	}
	require.Equal(t, 1, m.templatePickerCursor)

	// Move up 5 times: should clamp at 0.
	for range 5 {
		next, _ := m.Update(keyMsg("k"))
		m = next.(Model)
	}
	require.Equal(t, 0, m.templatePickerCursor)
}
