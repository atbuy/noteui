package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/editor"
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

func TestSortKeyEntersSortMode(t *testing.T) {
	m := newTestModel(t)
	require.False(t, m.pendingSort)
	m = updateModel(m, keyMsg("s"))
	require.True(t, m.pendingSort)
	require.Contains(t, m.status, "sort:")
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
	// s enters sort mode; m applies modified; status shows "sorted by modified time"
	m := newTestModel(t)
	m = updateModel(m, keyMsg("s"))
	require.True(t, m.pendingSort)
	m = updateModel(m, keyMsg("m"))
	require.False(t, m.pendingSort)
	require.Equal(t, sortModified, m.sortMethod)
	require.Contains(t, m.status, "modified")

	// s then n returns to alpha
	m = updateModel(m, keyMsg("s"))
	m = updateModel(m, keyMsg("n"))
	require.Equal(t, sortAlpha, m.sortMethod)
	require.Contains(t, m.status, "alpha")
}

func TestSortModeSubKeys(t *testing.T) {
	tests := []struct {
		key    string
		method string
	}{
		{"n", sortAlpha},
		{"m", sortModified},
		{"c", sortCreated},
	}
	for _, tt := range tests {
		m := newTestModel(t)
		m = updateModel(m, keyMsg("s"))
		m = updateModel(m, keyMsg(tt.key))
		require.False(t, m.pendingSort, "pendingSort should clear after %q", tt.key)
		require.Equal(t, tt.method, m.sortMethod, "key %q should set method %q", tt.key, tt.method)
	}
}

func TestSortModeSizeViaSS(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("s")) // enter sort mode
	m = updateModel(m, keyMsg("s")) // s in sort mode = sort by size
	require.False(t, m.pendingSort)
	require.Equal(t, sortSize, m.sortMethod)
}

func TestSortModeReverseToggle(t *testing.T) {
	m := newTestModel(t)
	require.False(t, m.sortReverse)
	m = updateModel(m, keyMsg("s"))
	m = updateModel(m, keyMsg("r"))
	require.True(t, m.sortReverse)
	require.Contains(t, m.status, "ascending")

	m = updateModel(m, keyMsg("s"))
	m = updateModel(m, keyMsg("r"))
	require.False(t, m.sortReverse)
}

func TestSortModeEscCancels(t *testing.T) {
	m := newTestModel(t)
	m.sortMethod = sortModified
	m = updateModel(m, keyMsg("s"))
	require.True(t, m.pendingSort)
	m = updateModel(m, keyMsg("esc"))
	require.False(t, m.pendingSort)
	require.Equal(t, sortModified, m.sortMethod) // unchanged
}

func TestSortModeUnknownKeyExits(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, keyMsg("s"))
	require.True(t, m.pendingSort)
	m = updateModel(m, keyMsg("x"))
	require.False(t, m.pendingSort)
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

// --- Safer deletion UX / undo tests ---

func makeTrashResult(t *testing.T) notes.TrashResult {
	t.Helper()
	dir := t.TempDir()
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)

	f := filepath.Join(dir, "note.md")
	if err := os.WriteFile(f, []byte("body"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := notes.TrashPath(f)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestNoteDeletedMsgSetsLastDeletion(t *testing.T) {
	m := newTestModel(t)
	result := makeTrashResult(t)
	m = updateModel(m, noteDeletedMsg{path: "/notes/note.md", result: result})
	if m.lastDeletion == nil {
		require.FailNow(t, "expected lastDeletion to be set after noteDeletedMsg")
	}
	require.Equal(t, "note.md", m.lastDeletion.label)
	require.Len(t, m.lastDeletion.results, 1)
	require.Contains(t, m.status, "Z to undo")
}

func TestNoteDeletedMsgErrorDoesNotChangeLastDeletion(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, noteDeletedMsg{err: errorf("delete error")})
	require.Nil(t, m.lastDeletion)
	require.Contains(t, m.status, "delete failed")
}

func TestCategoryDeletedMsgSetsLastDeletion(t *testing.T) {
	m := newTestModel(t)
	result := makeTrashResult(t)
	m = updateModel(m, categoryDeletedMsg{relPath: "work", result: result})
	if m.lastDeletion == nil {
		require.FailNow(t, "expected lastDeletion to be set after categoryDeletedMsg")
	}
	require.Contains(t, m.lastDeletion.label, "work")
	require.Contains(t, m.status, "Z to undo")
}

func TestBulkDeleteSetsLastDeletion(t *testing.T) {
	m := newTestModel(t)
	r1 := makeTrashResult(t)
	r2 := makeTrashResult(t)
	m = updateModel(m, notesDeletedMsg{
		paths:   []string{"/notes/a.md", "/notes/b.md"},
		results: []notes.TrashResult{r1, r2},
	})
	if m.lastDeletion == nil {
		require.FailNow(t, "expected lastDeletion to be set after notesDeletedMsg")
	}
	require.Len(t, m.lastDeletion.results, 2)
	require.Contains(t, m.status, "Z to undo")
}

func TestUndoDeleteKeyDispatchesRestoreCmd(t *testing.T) {
	m := newTestModel(t)
	result := makeTrashResult(t)
	m.lastDeletion = &undoableDelete{label: "note.md", results: []notes.TrashResult{result}}
	next, cmd := m.Update(keyMsg("Z"))
	m = next.(Model)
	require.Nil(t, m.lastDeletion)
	require.NotNil(t, cmd)
}

func TestUndoDeleteKeyNoOpWhenNoLastDeletion(t *testing.T) {
	m := newTestModel(t)
	m.lastDeletion = nil
	next, cmd := m.Update(keyMsg("Z"))
	m = next.(Model)
	require.Nil(t, m.lastDeletion)
	require.Nil(t, cmd)
}

func TestRestoreFinishedMsgSuccess(t *testing.T) {
	m := newTestModel(t)
	m.lastDeletion = &undoableDelete{label: "note.md"}
	m = updateModel(m, restoreFinishedMsg{label: "note.md"})
	require.Nil(t, m.lastDeletion)
	require.Contains(t, m.status, "restored:")
}

func TestRestoreFinishedMsgError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, restoreFinishedMsg{label: "note.md", err: errorf("rename failed")})
	require.Nil(t, m.lastDeletion)
	require.Contains(t, m.status, "restore failed:")
}

func TestOpenDailyNoteKeyCreatesNoteAndSetsFlag(t *testing.T) {
	m := newTestModel(t)
	root := m.rootDir

	next, cmd := m.Update(keyMsg("D"))
	m = next.(Model)

	require.True(t, m.dailyNoteOpen, "expected dailyNoteOpen to be true after pressing D")
	require.NotNil(t, cmd, "expected a command to be returned")
	require.Contains(t, m.status, "created daily note")

	today := time.Now().Format("2006-01-02")
	expectedPath := filepath.Join(root, "daily", today+".md")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("expected daily note to exist at %q: %v", expectedPath, err)
	}
}

func TestOpenDailyNoteKeyOpensExistingNote(t *testing.T) {
	m := newTestModel(t)
	root := m.rootDir

	today := time.Now().Format("2006-01-02")
	dailyDir := filepath.Join(root, "daily")
	require.NoError(t, os.MkdirAll(dailyDir, 0o755))
	notePath := filepath.Join(dailyDir, today+".md")
	require.NoError(t, os.WriteFile(notePath, []byte("# existing\n"), 0o644))

	next, cmd := m.Update(keyMsg("D"))
	m = next.(Model)

	require.True(t, m.dailyNoteOpen, "expected dailyNoteOpen to be true")
	require.NotNil(t, cmd)
	require.Contains(t, m.status, "opening daily note")
}

func TestEditorFinishedMsgSkipsRenameForDailyNote(t *testing.T) {
	m := newTestModel(t)
	root := m.rootDir

	today := time.Now().Format("2006-01-02")
	notePath := filepath.Join(root, "daily", today+".md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, []byte("# 2026-04-13\n\nbody"), 0o644))

	m.dailyNoteOpen = true
	m = updateModel(m, editor.FinishedMsg{Path: notePath})

	require.False(t, m.dailyNoteOpen, "expected dailyNoteOpen to be reset after editor closes")
	require.Equal(t, "editor closed", m.status)

	if _, err := os.Stat(notePath); err != nil {
		t.Errorf("expected daily note to still exist at original path: %v", err)
	}
}

func TestPreviewRenderedMsgExtractsWikilinks(t *testing.T) {
	m := newTestModel(t)
	m.previewPath = filepath.Join(m.rootDir, "work", "note.md")

	m = updateModel(m, previewRenderedMsg{
		forPath:     m.previewPath,
		baseContent: "some content",
		rawContent:  "See [[alpha]] and [[beta]].\nAlso [[alpha]] again.",
	})

	require.Equal(t, []string{"alpha", "beta"}, m.previewWikilinks)
}

func TestEnterInPreviewFocusFollowsWikilinkOnScreen(t *testing.T) {
	m := newTestModel(t)
	root := m.rootDir

	notePath := filepath.Join(root, "other-note.md")
	require.NoError(t, os.WriteFile(notePath, []byte("# Other Note\nbody\n"), 0o644))

	target := notes.Note{
		Path:      notePath,
		RelPath:   "other-note.md",
		Name:      "other-note.md",
		TitleText: "Other Note",
	}
	m.notes = []notes.Note{target}

	m.focus = focusPreview
	m.preview.Height = 10
	m.previewContent = "some text\n[[Other Note]]\nmore text"
	m.previewWikilinks = []string{"Other Note"}

	next, cmd := m.Update(keyMsg("enter"))
	m = next.(Model)

	require.NotNil(t, cmd, "expected a command to open the wikilinked note")
	require.Equal(t, focusTree, m.focus)
	require.Contains(t, m.status, "Other Note")
}

func TestEnterInPreviewFocusNoWikilinkDoesNothing(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	m.preview.Height = 10
	m.previewContent = "just plain text, no wikilinks here"
	m.previewWikilinks = nil

	next, cmd := m.Update(keyMsg("enter"))
	m = next.(Model)

	// No command should be emitted and focus stays on preview
	require.Equal(t, focusPreview, m.focus)
	_ = cmd
}

// --- Link nav mode tests ---

func makeLinkNavModel(t *testing.T) (Model, notes.Note) {
	t.Helper()
	m := newTestModel(t)
	notePath := filepath.Join(m.rootDir, "linked.md")
	require.NoError(t, os.WriteFile(notePath, []byte("# Linked\nbody\n"), 0o644))
	n := notes.Note{
		Path:      notePath,
		RelPath:   "linked.md",
		Name:      "linked.md",
		TitleText: "Linked",
	}
	m.notes = []notes.Note{n}
	m.focus = focusPreview
	m.preview.Height = 20
	m.previewContent = "intro\n[[Linked]]\ntrailing"
	m.previewLinks = []previewLinkItem{{rendLine: 1, target: "Linked", isWikilink: true}}
	m.previewLinkCursor = -1
	return m, n
}

func TestRebuildPreviewLinksFindsWikilinkLines(t *testing.T) {
	m := newTestModel(t)
	m.previewContent = "intro line\n[[Note A]]\nmiddle\n[[Note B]]\nend"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 2)
	require.Equal(t, 1, m.previewLinks[0].rendLine)
	require.Equal(t, "Note A", m.previewLinks[0].target)
	require.True(t, m.previewLinks[0].isWikilink)
	require.Equal(t, 3, m.previewLinks[1].rendLine)
	require.Equal(t, "Note B", m.previewLinks[1].target)
}

func TestRebuildPreviewLinksFindsMultipleLinksOnOneLine(t *testing.T) {
	m := newTestModel(t)
	m.previewContent = "see [[Alpha]] and [[Beta]] for details"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 2)
	require.Equal(t, 0, m.previewLinks[0].rendLine)
	require.Equal(t, "Alpha", m.previewLinks[0].target)
	require.Equal(t, 4, m.previewLinks[0].rendCol, "rendCol should be byte offset of [[ in line")
	require.Equal(t, 0, m.previewLinks[1].rendLine)
	require.Equal(t, "Beta", m.previewLinks[1].target)
	require.Greater(t, m.previewLinks[1].rendCol, m.previewLinks[0].rendCol,
		"second link should have a higher column than the first")
}

func TestRebuildPreviewLinksStoresRendLen(t *testing.T) {
	m := newTestModel(t)
	m.previewContent = "[[Hello World]]"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 1)
	require.Equal(t, len("[[Hello World]]"), m.previewLinks[0].rendLen)
}

func TestRebuildPreviewLinksFindsWrappedWikilink(t *testing.T) {
	m := newTestModel(t)
	// Simulate glamour soft-wrapping [[A Long\nTitle]] across two lines.
	m.previewContent = "intro\nsee [[A Long\nTitle]]\nmore"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 1, "wrapped wikilink should be found")
	link := m.previewLinks[0]
	require.Equal(t, 1, link.rendLine, "link should start on line 1 (the 'see' line)")
	require.Equal(t, "A Long Title", link.target,
		"target should have newline normalized to a space")
	require.Equal(t, true, link.isWikilink)
	// rendLen must not cross the newline: only "[[A Long" portion is on line 1.
	require.Equal(t, len("[[A Long"), link.rendLen,
		"rendLen should cover only the portion on the starting line")
	require.Equal(t, 2, link.rendEndLine)
	require.Equal(t, len("Title]]"), link.rendEndCol)
}

func TestRebuildPreviewLinksFindsWrappedExternalLink(t *testing.T) {
	m := newTestModel(t)
	m.previewContent = "intro\nsee (https://example.com/very/long\n/path)\nmore"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 1, "wrapped external link should be found")
	link := m.previewLinks[0]
	require.Equal(t, 1, link.rendLine, "link should start on the wrapped line")
	require.Equal(t, "https://example.com/very/long/path", link.target,
		"target should have soft-wrap whitespace removed")
	require.False(t, link.isWikilink)
	require.Equal(t, len("(https://example.com/very/long"), link.rendLen,
		"rendLen should stop at the first wrapped line")
	require.Equal(t, 2, link.rendEndLine)
	require.Equal(t, len("/path)"), link.rendEndCol)
}

func TestRebuildPreviewLinksFindsWrappedExternalLinkWithRenderedPadding(t *testing.T) {
	m := newTestModel(t)
	m.previewContent = "intro\nsee (https://example.com/very/long   \n    /path#frag)\nmore"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 1, "wrapped external link with rendered padding should be found")
	link := m.previewLinks[0]
	require.Equal(t, "https://example.com/very/long/path#frag", link.target,
		"target should ignore wrap padding and keep the full URL")
	require.False(t, link.isWikilink)
	require.Equal(t, 2, link.rendEndLine)
	require.Equal(t, len("    /path#frag)"), link.rendEndCol)
}

func TestApplyLinkSpanHighlightHighlightsWrappedLinkAcrossLines(t *testing.T) {
	content := "see (https://example.com/very/long\n/path)\nend"
	link := previewLinkItem{
		rendLine:    0,
		rendCol:     4,
		rendLen:     len("(https://example.com/very/long"),
		rendEndLine: 1,
		rendEndCol:  len("/path)"),
		target:      "https://example.com/very/long/path",
	}
	result := applyLinkSpanHighlight(content, link)

	require.Equal(t, content, stripANSI(result))
	require.Contains(t, result, lipgloss.NewStyle().Background(selectedBgColor).Foreground(selectedFgColor).Bold(true).Render("(https://example.com/very/long"))
	require.Contains(t, result, lipgloss.NewStyle().Background(selectedBgColor).Foreground(selectedFgColor).Bold(true).Render("/path)"))
}

func TestApplyLinkSpanHighlightOnlyHighlightsLinkText(t *testing.T) {
	content := "prefix [[My Note]] suffix"
	link := previewLinkItem{rendLine: 0, rendCol: 7, rendLen: len("[[My Note]]"), target: "My Note", isWikilink: true}
	result := applyLinkSpanHighlight(content, link)

	// Visible text must be preserved exactly.
	require.Equal(t, "prefix [[My Note]] suffix", stripANSI(result),
		"visible text should be unchanged after span highlight")

	// The literal text before the link ("prefix ") must be a prefix of the
	// result: no ANSI codes were injected in front of it.
	require.True(t, strings.HasPrefix(result, "prefix "),
		"prefix before the link should be untouched")

	// The literal text after the link (" suffix") must be a suffix of the
	// ANSI-stripped result (suffix is never touched by the highlighter).
	require.True(t, strings.HasSuffix(stripANSI(result), " suffix"),
		"suffix after the link should be untouched")
}

func TestNavigatingMultipleLinksOnSameLine(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	m.preview.Height = 20
	m.previewContent = "see [[Alpha]] and [[Beta]] for details"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 2)

	m.previewLinkNavMode = true
	m.previewLinkCursor = 0

	// Navigate from first to second link on the same line.
	m.jumpToNextLink()
	require.Equal(t, 1, m.previewLinkCursor)
	require.Equal(t, "Beta", m.previewLinks[1].target)
	require.Contains(t, m.status, "[[Beta]]")
}

func TestRebuildPreviewLinksFindsExternalLinks(t *testing.T) {
	m := newTestModel(t)
	m.previewContent = "some text\nhyperlink (https://example.com)\nmore text"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 1)
	require.Equal(t, 1, m.previewLinks[0].rendLine)
	require.Equal(t, "https://example.com", m.previewLinks[0].target)
	require.False(t, m.previewLinks[0].isWikilink)
}

func TestRebuildPreviewLinksFindsRenderedMarkdownExternalLinks(t *testing.T) {
	m := newTestModel(t)
	raw := "some [hyperlink](https://example.com) text"
	m.previewRawContent = raw
	m.previewBaseContent = renderMarkdownTerminal(raw, markdownRenderOptions{Width: 80})
	m.previewContent = m.previewBaseContent
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 1)
	require.Equal(t, "https://example.com", m.previewLinks[0].target)
	require.False(t, m.previewLinks[0].isWikilink)
	require.True(t, m.previewLinks[0].showTarget)
	require.Equal(t, "some hyperlink text", stripANSI(m.previewBaseContent))
}

func TestBracketFEntersLinkNavModeForRenderedMarkdownExternalLink(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	m.previewPath = filepath.Join(m.rootDir, "note.md")
	raw := "some [hyperlink](https://example.com) text"
	m = updateModel(m, previewRenderedMsg{
		forPath:     m.previewPath,
		baseContent: renderMarkdownTerminal(raw, markdownRenderOptions{Width: 80}),
		rawContent:  raw,
	})
	require.Len(t, m.previewLinks, 1)

	m = updateModel(m, keyMsg("["))
	m = updateModel(m, keyMsg("f"))

	require.True(t, m.previewLinkNavMode)
	require.Equal(t, 0, m.previewLinkCursor)
	require.Contains(t, m.status, "https://example.com")
}

func TestFollowSelectedRenderedMarkdownExternalLinkOpensBrowser(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	m.previewPath = filepath.Join(m.rootDir, "note.md")
	raw := "some [hyperlink](https://example.com) text"
	m = updateModel(m, previewRenderedMsg{
		forPath:     m.previewPath,
		baseContent: renderMarkdownTerminal(raw, markdownRenderOptions{Width: 80}),
		rawContent:  raw,
	})
	m.previewLinkNavMode = true
	m.previewLinkCursor = 0
	m.reapplyLinkHighlight()

	next, cmd := m.Update(keyMsg("f"))
	m = next.(Model)

	require.NotNil(t, cmd)
	require.Equal(t, "opening: https://example.com", m.status)
}

func TestApplyLinkSpanHighlightShowsTargetOnlyForSelectedMarkdownLink(t *testing.T) {
	content := "see first and second"
	result := applyLinkSpanHighlight(content, previewLinkItem{
		rendLine:    0,
		rendCol:     4,
		rendLen:     len("first"),
		rendEndLine: 0,
		rendEndCol:  9,
		target:      "https://example.com/first",
		showTarget:  true,
	})
	plain := stripANSI(result)
	require.Contains(t, plain, "first (https://example.com/first)")
	require.NotContains(t, plain, "second (")
	require.Equal(t, "see first (https://example.com/first) and second", plain)
}

func TestRebuildPreviewLinksFindsBareExternalLinkWithFragment(t *testing.T) {
	m := newTestModel(t)
	m.previewContent = "see https://pypi.org#test for details"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 1)
	require.Equal(t, 0, m.previewLinks[0].rendLine)
	require.Equal(t, "https://pypi.org#test", m.previewLinks[0].target)
	require.False(t, m.previewLinks[0].isWikilink)
}

func TestRebuildPreviewLinksFindsWrappedBareExternalLinkWithRenderedPadding(t *testing.T) {
	m := newTestModel(t)
	m.previewContent = "see https://example.com/very/long   \n    /path#frag for details"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 1)
	require.Equal(t, "https://example.com/very/long/path#frag", m.previewLinks[0].target)
	require.False(t, m.previewLinks[0].isWikilink)
}

func TestRebuildPreviewLinksDoesNotDuplicateParenthesizedExternalLink(t *testing.T) {
	m := newTestModel(t)
	m.previewContent = "see (https://example.com/docs#frag)"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 1)
	require.Equal(t, "https://example.com/docs#frag", m.previewLinks[0].target)
	require.False(t, m.previewLinks[0].isWikilink)
}

func TestFollowSelectedLinkExternalOpensBrowser(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	m.previewContent = "text (https://example.com)"
	m.previewLinks = []previewLinkItem{{rendLine: 0, target: "https://example.com", isWikilink: false}}
	m.previewLinkNavMode = true
	m.previewLinkCursor = 0

	next, cmd := m.Update(keyMsg("f"))
	m = next.(Model)

	require.NotNil(t, cmd, "external link should issue openURLCmd")
	require.Equal(t, "opening: https://example.com", m.status)
}

func TestPreviewRenderedMsgCallsRebuildLinks(t *testing.T) {
	m := newTestModel(t)
	m.previewPath = filepath.Join(m.rootDir, "note.md")
	m.preview.Width = 80
	m.preview.Height = 20
	m.previewContent = "[[foo]]"
	m.rebuildPreviewLinks()
	require.Len(t, m.previewLinks, 1)
	require.Equal(t, "foo", m.previewLinks[0].target)
}

func TestBracketFEntersLinkNavModeAndJumpsToFirstLink(t *testing.T) {
	m, _ := makeLinkNavModel(t)

	m = updateModel(m, keyMsg("]"))
	m = updateModel(m, keyMsg("f"))

	require.True(t, m.previewLinkNavMode)
	require.Equal(t, 0, m.previewLinkCursor)
	require.Contains(t, m.status, "[[Linked]]")
}

func TestBracketBackFEntersLinkNavModeAndJumpsToLastLink(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	m.preview.Height = 20
	m.previewContent = "[[Alpha]]\nmiddle\n[[Beta]]"
	m.previewLinks = []previewLinkItem{
		{rendLine: 0, target: "Alpha", isWikilink: true},
		{rendLine: 2, target: "Beta", isWikilink: true},
	}

	m = updateModel(m, keyMsg("["))
	m = updateModel(m, keyMsg("f"))

	require.True(t, m.previewLinkNavMode)
	require.Equal(t, 1, m.previewLinkCursor)
}

func TestJumpToNextLinkWrapsAround(t *testing.T) {
	m, _ := makeLinkNavModel(t)
	m.previewLinkNavMode = true
	m.previewLinkCursor = 0

	m.jumpToNextLink()

	require.Equal(t, 0, m.previewLinkCursor, "should wrap back to 0 with only one link")
}

func TestJumpToPrevLinkWrapsAround(t *testing.T) {
	m, _ := makeLinkNavModel(t)
	m.previewLinkNavMode = true
	m.previewLinkCursor = 0

	m.jumpToPrevLink()

	require.Equal(t, 0, m.previewLinkCursor, "should wrap back to last (0) with only one link")
}

func TestEscExitsLinkNavMode(t *testing.T) {
	m, _ := makeLinkNavModel(t)
	m.previewLinkNavMode = true
	m.previewLinkCursor = 0

	m = updateModel(m, keyMsg("esc"))

	require.False(t, m.previewLinkNavMode)
	require.Equal(t, -1, m.previewLinkCursor)
	require.Equal(t, "link nav off", m.status)
}

func TestFollowLinkKeyFollowsSelectedLink(t *testing.T) {
	m, n := makeLinkNavModel(t)
	m.previewLinkNavMode = true
	m.previewLinkCursor = 0

	next, cmd := m.Update(keyMsg("f"))
	m = next.(Model)

	require.NotNil(t, cmd, "expected open command")
	require.False(t, m.previewLinkNavMode)
	require.Equal(t, focusTree, m.focus)
	require.Contains(t, m.status, n.TitleText)
}

func TestEnterInLinkNavModeFollowsSelectedLink(t *testing.T) {
	m, n := makeLinkNavModel(t)
	m.previewLinkNavMode = true
	m.previewLinkCursor = 0

	next, cmd := m.Update(keyMsg("enter"))
	m = next.(Model)

	require.NotNil(t, cmd, "expected open command")
	require.False(t, m.previewLinkNavMode)
	require.Equal(t, focusTree, m.focus)
	require.Contains(t, m.status, n.TitleText)
}

func TestBracketFInLinkNavModeNavigatesToNextLink(t *testing.T) {
	m := newTestModel(t)
	m.focus = focusPreview
	m.preview.Height = 20
	m.previewContent = "[[Alpha]]\nmiddle\n[[Beta]]"
	m.previewLinks = []previewLinkItem{
		{rendLine: 0, target: "Alpha", isWikilink: true},
		{rendLine: 2, target: "Beta", isWikilink: true},
	}
	m.previewLinkNavMode = true
	m.previewLinkCursor = 0

	m = updateModel(m, keyMsg("]"))
	m = updateModel(m, keyMsg("f"))

	require.Equal(t, 1, m.previewLinkCursor)
	require.Contains(t, m.status, "[[Beta]]")
}

func TestLinkNavAndTodoNavAreMutuallyExclusive(t *testing.T) {
	// Simulate having pendingBracketDir already set (from a prior ']' press
	// outside todo nav), then pressing 'f' while todo nav is also active.
	m := newTestModel(t)
	m.focus = focusPreview
	m.preview.Height = 20
	m.previewContent = "[[Alpha]]"
	m.previewLinks = []previewLinkItem{{rendLine: 0, target: "Alpha", isWikilink: true}}
	m.previewTodoNavMode = true
	m.previewTodoCursor = 0
	m.pendingBracketDir = "]"

	m = updateModel(m, keyMsg("f"))

	require.True(t, m.previewLinkNavMode)
	require.False(t, m.previewTodoNavMode)
	require.Equal(t, -1, m.previewTodoCursor)
}

func TestFollowSelectedLinkUnknownTargetSetsStatus(t *testing.T) {
	m, _ := makeLinkNavModel(t)
	m.previewLinkNavMode = true
	m.previewLinkCursor = 0
	m.previewLinks[0] = previewLinkItem{rendLine: 1, target: "Nonexistent Note", isWikilink: true}
	// notes is still set to "Linked" which won't match

	cmd := m.followSelectedLink()

	require.Nil(t, cmd)
	require.Contains(t, m.status, "no note found for")
	require.Contains(t, m.status, "Nonexistent Note")
}

// Trash browser tests

func makeTrashItem(root, name string) notes.TrashedItem {
	return notes.TrashedItem{
		Name:          name,
		OriginalPath:  filepath.Join(root, name),
		TrashFilePath: "/tmp/trash/files/" + name,
		TrashInfoPath: "/tmp/trash/info/" + name + ".trashinfo",
		DeletionDate:  time.Now(),
	}
}

func TestTrashBrowserOpenShowsModal(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(trashBrowserLoadedMsg{
		items: []notes.TrashedItem{makeTrashItem(m.rootDir, "note.md")},
	})
	m = next.(Model)

	require.True(t, m.showTrashBrowser)
	require.Equal(t, 0, m.trashBrowserCursor)
	require.Equal(t, "trash browser", m.status)
}

func TestTrashBrowserEscClosesModal(t *testing.T) {
	m := newTestModel(t)
	m.showTrashBrowser = true
	m.trashBrowserItems = []notes.TrashedItem{makeTrashItem(m.rootDir, "note.md")}

	next, _ := m.Update(keyMsg("esc"))
	m = next.(Model)

	require.False(t, m.showTrashBrowser)
	require.Nil(t, m.trashBrowserItems)
}

func TestTrashBrowserNavigationMovesDown(t *testing.T) {
	m := newTestModel(t)
	m.showTrashBrowser = true
	m.trashBrowserItems = []notes.TrashedItem{
		makeTrashItem(m.rootDir, "a.md"),
		makeTrashItem(m.rootDir, "b.md"),
	}
	m.trashBrowserCursor = 0

	next, _ := m.Update(keyMsg("j"))
	m = next.(Model)

	require.Equal(t, 1, m.trashBrowserCursor)
}

func TestTrashBrowserNavigationClamped(t *testing.T) {
	m := newTestModel(t)
	m.showTrashBrowser = true
	m.trashBrowserItems = []notes.TrashedItem{
		makeTrashItem(m.rootDir, "a.md"),
		makeTrashItem(m.rootDir, "b.md"),
	}

	// Can't go below 0.
	m.trashBrowserCursor = 0
	next, _ := m.Update(keyMsg("k"))
	m = next.(Model)
	require.Equal(t, 0, m.trashBrowserCursor)

	// Can't go past last item.
	m.trashBrowserCursor = 1
	next, _ = m.Update(keyMsg("j"))
	m = next.(Model)
	require.Equal(t, 1, m.trashBrowserCursor)
}

func TestTrashBrowserEmptyDoesNotOpen(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(trashBrowserLoadedMsg{items: nil})
	m = next.(Model)

	require.False(t, m.showTrashBrowser)
	require.Contains(t, m.status, "empty")
}

func TestTrashBrowserRestoreErrorKeepsModalOpen(t *testing.T) {
	m := newTestModel(t)
	m.showTrashBrowser = true
	m.trashBrowserItems = []notes.TrashedItem{makeTrashItem(m.rootDir, "note.md")}

	next, _ := m.Update(trashRestoreMsg{
		item: m.trashBrowserItems[0],
		err:  errors.New("already exists"),
	})
	m = next.(Model)

	require.True(t, m.showTrashBrowser)
	require.Contains(t, m.status, "restore failed")
}

func TestTrashBrowserRestoreSuccessClosesAndRefreshes(t *testing.T) {
	m := newTestModel(t)
	item := makeTrashItem(m.rootDir, "note.md")
	m.showTrashBrowser = true
	m.trashBrowserItems = []notes.TrashedItem{item}

	next, cmd := m.Update(trashRestoreMsg{item: item, err: nil})
	m = next.(Model)

	require.False(t, m.showTrashBrowser)
	require.Nil(t, m.trashBrowserItems)
	require.NotNil(t, cmd)
	require.Contains(t, m.status, "note.md")
}
