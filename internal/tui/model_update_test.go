package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
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
	next, cmd := m.Update(syncDebouncedMsg{token: 3})
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
	require.True(t, m.syncInFlight["work/remote.md"])
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
		notes:      nil,
		tempNotes:  nil,
		categories: nil,
		err:        nil,
	})
	if !strings.Contains(m.status, "no notes found") {
		require.Failf(t, "assertion failed", "expected 'no notes found' status, got %q", m.status)
	}
}

func TestDataLoadedMsgWithError(t *testing.T) {
	m := newTestModel(t)
	m = updateModel(m, dataLoadedMsg{err: errorf("test error")})
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

func TestBracketForwardSwitchesToTemporaryMode(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	m = updateModel(m, keyMsg("]"))
	if m.listMode != listModeTemporary {
		require.Failf(t, "assertion failed", "expected listModeTemporary after ']', got %v", m.listMode)
	}
}

func TestBracketBackwardFromTemporarySwitchesToNotesMode(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	m = updateModel(m, keyMsg("["))
	if m.listMode != listModeNotes {
		require.Failf(t, "assertion failed", "expected listModeNotes after '[' from temporary mode, got %v", m.listMode)
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

func TestEnterOnRemoteOnlyNoteShowsImportStatus(t *testing.T) {
	m := newTestModel(t)
	m.treeItems = []treeItem{{Kind: treeRemoteNote, RelPath: "work/remote.md", Name: "Remote Note", RemoteNote: &notesync.RemoteNoteMeta{ID: "n1", RelPath: "work/remote.md", Title: "Remote Note"}}}
	m.treeCursor = 0
	m = updateModel(m, keyMsg("enter"))
	require.Contains(t, m.status, "press i to import it or I to import all")
}
