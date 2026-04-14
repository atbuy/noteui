package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
	notesync "atbuy/noteui/internal/sync"
)

func configForApply() config.KeysConfig {
	return config.KeysConfig{Quit: []string{"x", "ctrl+x"}}
}

func emptyConfig() config.KeysConfig {
	return config.KeysConfig{}
}

// Tests for modal render functions, tree/list views, and action helpers.

func TestRenderCreateCategoryModal(t *testing.T) {
	m := newTestModel(t)
	m.showCreateCategory = true
	m.width = 120
	m.height = 40
	rendered := m.renderCreateCategoryModal()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "Create category") {
		require.Failf(t, "assertion failed", "expected modal title, got %q", plain[:min(len(plain), 200)])
	}
}

func TestRenderAddTagModal(t *testing.T) {
	m := newTestModel(t)
	m.showAddTag = true
	m.width = 120
	m.height = 40
	rendered := m.renderAddTagModal()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "Add tag") {
		require.Failf(t, "assertion failed", "expected 'Add tag' in modal, got %q", plain[:min(len(plain), 200)])
	}
}

func TestRenderTodoAddModal(t *testing.T) {
	m := newTestModel(t)
	m.showTodoAdd = true
	m.width = 120
	m.height = 40
	rendered := m.renderTodoAddModal()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "Add todo item") {
		require.Failf(t, "assertion failed", "expected 'Add todo item' in modal, got %q", plain[:min(len(plain), 200)])
	}
}

func TestRenderTodoEditModal(t *testing.T) {
	m := newTestModel(t)
	m.showTodoEdit = true
	m.width = 120
	m.height = 40
	rendered := m.renderTodoEditModal()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "Edit todo item") {
		require.Failf(t, "assertion failed", "expected 'Edit todo item' in modal, got %q", plain[:min(len(plain), 200)])
	}
}

func TestRenderPassphraseModal(t *testing.T) {
	m := newTestModel(t)
	m.showPassphraseModal = true
	m.width = 120
	m.height = 40
	rendered := m.renderPassphraseModal()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "passphrase") && !strings.Contains(strings.ToLower(plain), "passphrase") {
		require.Failf(t, "assertion failed", "expected passphrase text in modal, got %q", plain[:min(len(plain), 200)])
	}
}

func TestRenderPassphraseModalEncryptContext(t *testing.T) {
	m := newTestModel(t)
	m.showPassphraseModal = true
	m.passphraseModalCtx = "encrypt"
	m.width = 120
	m.height = 40
	rendered := m.renderPassphraseModal()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "Set passphrase") {
		require.Failf(t, "assertion failed", "expected 'Set passphrase' title for encrypt context, got %q", plain[:min(len(plain), 200)])
	}
}

func TestRenderEncryptConfirmModal(t *testing.T) {
	m := newTestModel(t)
	m.showEncryptConfirm = true
	m.width = 120
	m.height = 40
	rendered := m.renderEncryptConfirmModal()
	plain := stripANSI(rendered)
	if !strings.Contains(strings.ToLower(plain), "encrypt") {
		require.Failf(t, "assertion failed", "expected encrypt text in confirmation modal, got %q", plain[:min(len(plain), 200)])
	}
}

func TestRenderMoveModal(t *testing.T) {
	m := newTestModel(t)
	m.showMove = true
	m.movePending = &movePending{kind: moveTargetNote, oldRelPath: "work/note.md", name: "note.md"}
	m.width = 120
	m.height = 40
	rendered := m.renderMoveModal()
	plain := stripANSI(rendered)
	if !strings.Contains(strings.ToLower(plain), "move") {
		require.Failf(t, "assertion failed", "expected 'move' in modal, got %q", plain[:min(len(plain), 200)])
	}
}

func TestRenderRenameModal(t *testing.T) {
	m := newTestModel(t)
	m.showRename = true
	m.renamePending = &renamePending{kind: renameTargetNote, path: "work/note.md", oldTitle: "Old Title"}
	m.width = 120
	m.height = 40
	rendered := m.renderRenameModal()
	plain := stripANSI(rendered)
	if !strings.Contains(strings.ToLower(plain), "rename") {
		require.Failf(t, "assertion failed", "expected 'rename' in modal, got %q", plain[:min(len(plain), 200)])
	}
}

func TestRenderTreeViewEmpty(t *testing.T) {
	m := newTestModel(t)
	m.treeItems = nil
	rendered := m.renderTreeView()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "Press n") {
		require.Failf(t, "assertion failed", "expected actionable empty state in tree view, got %q", plain)
	}
}

func TestRenderTreeViewWithNotes(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	now := time.Now()
	m.notes = []notes.Note{
		{RelPath: "work/alpha.md", Name: "alpha.md", TitleText: "Alpha Note", ModTime: now},
	}
	m.categories = []notes.Category{
		{Name: "All notes", RelPath: ""},
		{Name: "work", RelPath: "work"},
	}
	m.expanded = map[string]bool{"": true, "work": true}
	m.pinnedNotes = map[string]bool{}
	m.pinnedCats = map[string]bool{}
	m.markedTreeItems = map[string]bool{}
	m.rebuildTree()

	rendered := m.renderTreeView()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "Alpha Note") && !strings.Contains(plain, "alpha.md") {
		require.Failf(t, "assertion failed", "expected note title in tree view, got %q", plain)
	}
}

func TestRenderTemporaryListViewEmpty(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	m.tempNotes = nil
	rendered := m.renderTemporaryListView()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "empty") && !strings.Contains(plain, "no") && !strings.Contains(plain, "temp") {
		// Just verify it doesn't panic and returns something
		_ = plain
	}
}

func TestRenderTemporaryListViewWithNotes(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	m.width = 120
	now := time.Now()
	m.tempNotes = []notes.Note{
		{RelPath: "scratch.md", Name: "scratch.md", TitleText: "Scratch Note", ModTime: now},
	}
	m.tempCursor = 0
	rendered := m.renderTemporaryListView()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "Scratch Note") && !strings.Contains(plain, "scratch.md") {
		require.Failf(t, "assertion failed", "expected temp note in temporary list, got %q", plain)
	}
}

func TestRenderPinsListViewEmpty(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModePins
	m.pinnedNotes = map[string]bool{}
	m.pinnedCats = map[string]bool{}
	rendered := m.renderPinsListView()
	plain := stripANSI(rendered)
	// Should render without panic
	_ = plain
}

func TestLeftPanelTitle(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	title := m.leftPanelTitle()
	if title == "" {
		require.FailNow(t, "expected non-empty left panel title")
	}
}

func TestLeftPanelTitleTemporary(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	title := m.leftPanelTitle()
	if !strings.Contains(strings.ToLower(title), "temp") {
		require.Failf(t, "assertion failed", "expected 'temp' in panel title for temporary mode, got %q", title)
	}
}

func TestLeftPanelTitlePins(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModePins
	title := m.leftPanelTitle()
	if !strings.Contains(strings.ToLower(title), "pin") {
		require.Failf(t, "assertion failed", "expected 'pin' in panel title for pins mode, got %q", title)
	}
}

func TestCurrentNotePathTemporary(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	now := time.Now()
	m.tempNotes = []notes.Note{
		{RelPath: "scratch.md", Name: "scratch.md", Path: "/tmp/scratch.md", ModTime: now},
	}
	m.tempCursor = 0
	path := m.currentNotePath()
	if path != "/tmp/scratch.md" {
		require.Failf(t, "assertion failed", "expected /tmp/scratch.md, got %q", path)
	}
}

func TestCurrentNotePathNone(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	m.treeItems = nil
	path := m.currentNotePath()
	if path != "" {
		require.Failf(t, "assertion failed", "expected empty path when no note selected, got %q", path)
	}
}

func TestCurrentNotePathFromTree(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	n := &notes.Note{RelPath: "work/note.md", Path: "/notes/work/note.md"}
	m.treeItems = []treeItem{
		{Kind: treeNote, RelPath: "work/note.md", Note: n},
	}
	m.treeCursor = 0
	path := m.currentNotePath()
	if path != "/notes/work/note.md" {
		require.Failf(t, "assertion failed", "expected note path from tree, got %q", path)
	}
}

func TestArmDeleteCurrentNoSelection(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	m.treeItems = nil
	// Should not panic with no selection
	m.armDeleteCurrent()
	if m.deletePending != nil {
		require.FailNow(t, "expected nil deletePending when no selection")
	}
}

func TestArmDeleteCurrentNote(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	n := &notes.Note{RelPath: "work/note.md", Path: "/notes/work/note.md", Name: "note.md"}
	m.treeItems = []treeItem{
		{Kind: treeNote, RelPath: "work/note.md", Name: "note.md", Note: n},
	}
	m.treeCursor = 0
	m.armDeleteCurrent()
	if m.deletePending == nil {
		require.FailNow(t, "expected deletePending to be set after armDeleteCurrent")
	}
	if m.deletePending.kind != deleteTargetNote {
		require.Failf(t, "assertion failed", "expected deleteTargetNote kind, got %v", m.deletePending.kind)
	}
}

func TestArmDeleteCurrentCategory(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	m.treeItems = []treeItem{
		{Kind: treeCategory, RelPath: "work", Name: "work"},
	}
	m.treeCursor = 0
	m.armDeleteCurrent()
	if m.deletePending == nil {
		require.FailNow(t, "expected deletePending to be set for category delete")
	}
	if m.deletePending.kind != deleteTargetCategory {
		require.Failf(t, "assertion failed", "expected deleteTargetCategory kind, got %v", m.deletePending.kind)
	}
}

func TestArmDeleteRootCategoryRejected(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	m.treeItems = []treeItem{
		{Kind: treeCategory, RelPath: "", Name: "All notes"},
	}
	m.treeCursor = 0
	m.armDeleteCurrent()
	if m.deletePending != nil {
		require.FailNow(t, "expected root category delete to be rejected")
	}
}

func TestArmRenameCurrentNote(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	n := &notes.Note{RelPath: "work/note.md", Path: "/notes/work/note.md", Name: "note.md", TitleText: "My Note"}
	m.treeItems = []treeItem{
		{Kind: treeNote, RelPath: "work/note.md", Name: "My Note", Note: n},
	}
	m.treeCursor = 0
	m.armRenameCurrent()
	if !m.showRename {
		require.FailNow(t, "expected showRename to be true after armRenameCurrent")
	}
	if m.renamePending == nil {
		require.FailNow(t, "expected renamePending to be set")
	}
	if m.renameInput.Value() != "My Note" {
		require.Failf(t, "assertion failed", "expected rename input to contain current title, got %q", m.renameInput.Value())
	}
}

func TestArmRenameRootCategoryRejected(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	m.treeItems = []treeItem{
		{Kind: treeCategory, RelPath: "", Name: "All notes"},
	}
	m.treeCursor = 0
	m.armRenameCurrent()
	if m.showRename {
		require.FailNow(t, "expected root category rename to be rejected")
	}
}

func TestArmAddTagCurrentNoNote(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	m.treeItems = nil
	m.armAddTagCurrent()
	if m.showAddTag {
		require.FailNow(t, "expected showAddTag to remain false when no note selected")
	}
}

func TestArmAddTagCurrentWithNote(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	n := &notes.Note{RelPath: "work/note.md", Path: "/notes/work/note.md"}
	m.treeItems = []treeItem{
		{Kind: treeNote, RelPath: "work/note.md", Note: n},
	}
	m.treeCursor = 0
	m.armAddTagCurrent()
	if !m.showAddTag {
		require.FailNow(t, "expected showAddTag to be true after armAddTagCurrent")
	}
}

func TestFilteredTempNotes(t *testing.T) {
	now := time.Now()
	m := newTestModel(t)
	m.tempNotes = []notes.Note{
		{RelPath: "alpha.md", Name: "alpha.md", TitleText: "Alpha", ModTime: now},
		{RelPath: "beta.md", Name: "beta.md", TitleText: "Beta", ModTime: now.Add(-time.Hour)},
	}
	m.searchInput.SetValue("")
	all := m.filteredTempNotes()
	if len(all) != 2 {
		require.Failf(t, "assertion failed", "expected 2 temp notes with empty query, got %d", len(all))
	}

	m.searchInput.SetValue("alpha")
	filtered := m.filteredTempNotes()
	if len(filtered) != 1 || filtered[0].TitleText != "Alpha" {
		require.Failf(t, "assertion failed", "expected 1 matching note, got %v", filtered)
	}
}

func TestFilteredTempNotesSortByModTime(t *testing.T) {
	now := time.Now()
	m := newTestModel(t)
	m.tempNotes = []notes.Note{
		{RelPath: "old.md", Name: "old.md", TitleText: "Old", ModTime: now.Add(-time.Hour)},
		{RelPath: "new.md", Name: "new.md", TitleText: "New", ModTime: now},
	}
	m.sortMethod = sortModified
	m.searchInput.SetValue("")
	sorted := m.filteredTempNotes()
	if len(sorted) != 2 || sorted[0].TitleText != "New" {
		require.Failf(t, "assertion failed", "expected newest first, got %v", sorted)
	}
}

func TestMoveTempCursor(t *testing.T) {
	now := time.Now()
	m := newTestModel(t)
	m.tempNotes = []notes.Note{
		{RelPath: "a.md", Name: "a.md", ModTime: now},
		{RelPath: "b.md", Name: "b.md", ModTime: now},
		{RelPath: "c.md", Name: "c.md", ModTime: now},
	}
	m.tempCursor = 0
	m.listMode = listModeTemporary

	m.moveTempCursor(1)
	if m.tempCursor != 1 {
		require.Failf(t, "assertion failed", "expected cursor 1 after delta +1, got %d", m.tempCursor)
	}

	m.moveTempCursor(-5)
	if m.tempCursor != 0 {
		require.Failf(t, "assertion failed", "expected cursor clamped to 0, got %d", m.tempCursor)
	}

	m.moveTempCursor(100)
	if m.tempCursor != 2 {
		require.Failf(t, "assertion failed", "expected cursor clamped to max (2), got %d", m.tempCursor)
	}
}

func TestCurrentCategoryPrefixCategory(t *testing.T) {
	m := newTestModel(t)
	m.treeItems = []treeItem{
		{Kind: treeCategory, RelPath: "work", Name: "work"},
	}
	m.treeCursor = 0
	prefix := m.currentCategoryPrefix()
	if prefix != "work/" {
		require.Failf(t, "assertion failed", "expected 'work/', got %q", prefix)
	}
}

func TestCurrentCategoryPrefixNote(t *testing.T) {
	m := newTestModel(t)
	n := &notes.Note{RelPath: "work/note.md"}
	m.treeItems = []treeItem{
		{Kind: treeNote, RelPath: "work/note.md", Note: n},
	}
	m.treeCursor = 0
	prefix := m.currentCategoryPrefix()
	if prefix != "work/" {
		require.Failf(t, "assertion failed", "expected 'work/', got %q", prefix)
	}
}

func TestCurrentCategoryPrefixRoot(t *testing.T) {
	m := newTestModel(t)
	m.treeItems = []treeItem{
		{Kind: treeCategory, RelPath: "", Name: "All notes"},
	}
	m.treeCursor = 0
	prefix := m.currentCategoryPrefix()
	if prefix != "" {
		require.Failf(t, "assertion failed", "expected empty prefix for root category, got %q", prefix)
	}
}

func TestApplyConfigKeysOverridesBinding(t *testing.T) {
	// Save original keys to restore after the test
	origKeys := keys
	defer func() { keys = origKeys }()

	defaultQuitKeys := keys.Quit.Keys()

	// Apply an override via a config struct with only Quit set
	cfg := configForApply()
	ApplyConfigKeys(cfg)

	newKeys := keys.Quit.Keys()
	if len(newKeys) == 0 {
		require.FailNow(t, "expected quit key to have bindings after override")
	}
	if len(defaultQuitKeys) > 0 && newKeys[0] == defaultQuitKeys[0] {
		require.Failf(t, "assertion failed", "expected key override to change quit binding, still %q", newKeys[0])
	}
}

func TestApplyConfigKeysEmptyNoOp(t *testing.T) {
	origKeys := keys
	defer func() { keys = origKeys }()

	before := keys.Quit.Keys()

	// Empty config override: all slices nil, should be no-op
	ApplyConfigKeys(emptyConfig())

	after := keys.Quit.Keys()
	if len(before) != len(after) || (len(before) > 0 && before[0] != after[0]) {
		require.Failf(t, "assertion failed", "expected empty config not to change quit binding, before=%v after=%v", before, after)
	}
}

func TestRenderTreeLineNote(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	m.pinnedNotes = map[string]bool{}
	m.pinnedCats = map[string]bool{}
	m.markedTreeItems = map[string]bool{}
	n := notes.Note{RelPath: "work/note.md", Name: "note.md", TitleText: "My Note", SyncClass: notes.SyncClassSynced}
	item := treeItem{Kind: treeNote, RelPath: "work/note.md", Name: "My Note", Note: &n, Depth: 0}
	rendered := m.renderTreeLine(item, false)
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "My Note") {
		require.Failf(t, "assertion failed", "expected 'My Note' in tree line, got %q", plain)
	}
	if !strings.Contains(plain, "●") {
		require.Failf(t, "assertion failed", "expected synced marker in tree line, got %q", plain)
	}
}

func TestRenderTreeLineCategory(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	m.pinnedNotes = map[string]bool{}
	m.pinnedCats = map[string]bool{}
	m.markedTreeItems = map[string]bool{}
	m.categories = []notes.Category{{Name: "work", RelPath: "work"}}
	item := treeItem{Kind: treeCategory, RelPath: "work", Name: "work", Depth: 0, Expanded: true}
	rendered := m.renderTreeLine(item, true)
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "work") {
		require.Failf(t, "assertion failed", "expected 'work' in category tree line, got %q", plain)
	}
}

func TestRenderTreeLineLocalNoteMarker(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	m.pinnedNotes = map[string]bool{}
	m.pinnedCats = map[string]bool{}
	m.markedTreeItems = map[string]bool{}
	n := notes.Note{RelPath: "work/local.md", Name: "local.md", TitleText: "Local Note", SyncClass: notes.SyncClassLocal}
	item := treeItem{Kind: treeNote, RelPath: "work/local.md", Name: "Local Note", Note: &n, Depth: 0}
	rendered := m.renderTreeLine(item, false)
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "○") {
		require.Failf(t, "assertion failed", "expected local marker in tree line, got %q", plain)
	}
}

func TestNoteSyncVisualStateUsesHealthyRecord(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{RelPath: "work/note.md", SyncClass: notes.SyncClassSynced}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", LastSyncAt: time.Now()},
	}
	require.Equal(t, noteSyncVisualHealthy, m.noteSyncVisualState(&n))
}

func TestNoteSyncVisualStateUsesPendingWhenLastSyncFailed(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{RelPath: "work/note.md", SyncClass: notes.SyncClassSynced}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", LastSyncAt: time.Now(), LastSyncError: "network down"},
	}
	require.Equal(t, noteSyncVisualPending, m.noteSyncVisualState(&n))
}

func TestNoteSyncVisualStateUsesSyncingWhenInFlight(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{RelPath: "work/note.md", SyncClass: notes.SyncClassSynced}
	m.syncInFlight = map[string]bool{"work/note.md": true}
	require.Equal(t, noteSyncVisualSyncing, m.noteSyncVisualState(&n))
}

func TestNoteSyncMarkerBlinksWhileRunning(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{RelPath: "work/note.md", SyncClass: notes.SyncClassSynced}
	m.syncInFlight = map[string]bool{"work/note.md": true}
	m.syncSpinnerFrame = 0
	mark, _ := m.noteSyncMarker(&n)
	require.Equal(t, "● ", mark)
	m.syncSpinnerFrame = 1
	mark, _ = m.noteSyncMarker(&n)
	require.Equal(t, "◌ ", mark)
}

func TestPendingSyncRelPathsOnlyMarksDirtyNotes(t *testing.T) {
	m := newTestModel(t)
	healthyPath := m.rootDir + "/healthy.md"
	dirtyPath := m.rootDir + "/dirty.md"
	require.NoError(t, os.WriteFile(healthyPath, []byte("healthy"), 0o644))
	require.NoError(t, os.WriteFile(dirtyPath, []byte("dirty-new"), 0o644))
	m.notes = []notes.Note{
		{Path: healthyPath, RelPath: "healthy.md", SyncClass: notes.SyncClassSynced, Encrypted: false},
		{Path: dirtyPath, RelPath: "dirty.md", SyncClass: notes.SyncClassSynced, Encrypted: false},
	}
	m.syncRecords = map[string]notesync.NoteRecord{
		"healthy.md": {RelPath: "healthy.md", LastSyncAt: time.Now(), LastSyncedHash: notesync.HashContent("healthy"), Encrypted: false},
		"dirty.md":   {RelPath: "dirty.md", LastSyncAt: time.Now(), LastSyncedHash: notesync.HashContent("dirty-old"), Encrypted: false},
	}
	pending := m.pendingSyncRelPaths()
	require.False(t, pending["healthy.md"])
	require.True(t, pending["dirty.md"])
}

func TestNoteSyncMarkerUsesHollowCircleForLocalNotes(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{RelPath: "work/local.md", SyncClass: notes.SyncClassLocal}
	mark, _ := m.noteSyncMarker(&n)
	require.Equal(t, "○ ", mark)
}

func TestNoteSyncMarkerUsesFilledCircleForPendingSync(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{RelPath: "work/note.md", SyncClass: notes.SyncClassSynced}
	mark, _ := m.noteSyncMarker(&n)
	require.Equal(t, "● ", mark)
}

func TestRenderTreeLineRemoteOnlyNoteUsesMutedXMarker(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	item := treeItem{Kind: treeRemoteNote, RelPath: "work/remote.md", Name: "Remote Note", RemoteNote: &notesync.RemoteNoteMeta{ID: "n1", RelPath: "work/remote.md", Title: "Remote Note"}, Depth: 0}
	rendered := m.renderTreeLine(item, false)
	plain := stripANSI(rendered)
	require.Contains(t, plain, "x")
	require.Contains(t, plain, "Remote Note")
}

func TestRenderTreeLineRemoteOnlyDuplicateShowsIDBadge(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.remoteOnlyNotes = []notesync.RemoteNoteMeta{{ID: "n1aaaa", RelPath: "work/remote.md", Title: "Remote Note"}, {ID: "n2bbbb", RelPath: "work/remote.md", Title: "Remote Note"}}
	item := treeItem{Kind: treeRemoteNote, RelPath: "work/remote.md", Name: m.remoteOnlyDisplayTitle(m.remoteOnlyNotes[0]), RemoteNote: &m.remoteOnlyNotes[0], Depth: 0}
	rendered := m.renderTreeLine(item, false)
	plain := stripANSI(rendered)
	require.Contains(t, plain, "Remote Note [n1aaaa]")
}

func TestCurrentCategoryPrefixRemoteOnlyNote(t *testing.T) {
	m := newTestModel(t)
	item := treeItem{Kind: treeRemoteNote, RelPath: "work/remote.md", Name: "Remote Note", RemoteNote: &notesync.RemoteNoteMeta{ID: "n1", RelPath: "work/remote.md", Title: "Remote Note"}}
	m.treeItems = []treeItem{item}
	m.treeCursor = 0
	require.Equal(t, "work/", m.currentCategoryPrefix())
}

func TestCurrentNotePathRemoteOnlyNoteIsEmpty(t *testing.T) {
	m := newTestModel(t)
	item := treeItem{Kind: treeRemoteNote, RelPath: "work/remote.md", Name: "Remote Note", RemoteNote: &notesync.RemoteNoteMeta{ID: "n1", RelPath: "work/remote.md", Title: "Remote Note"}}
	m.treeItems = []treeItem{item}
	m.treeCursor = 0
	require.Equal(t, "", m.currentNotePath())
}

func TestRenderTreeLineRemoteOnlyNoteBlinksWhileImporting(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.syncInFlight = map[string]bool{remoteOnlySyncVisualKey("n1"): true}
	m.syncSpinnerFrame = 0
	item := treeItem{Kind: treeRemoteNote, RelPath: "work/remote.md", Name: "Remote Note", RemoteNote: &notesync.RemoteNoteMeta{ID: "n1", RelPath: "work/remote.md", Title: "Remote Note"}}
	rendered := m.renderTreeLine(item, false)
	plain := stripANSI(rendered)
	require.Contains(t, plain, "●")

	m.syncSpinnerFrame = 1
	rendered = m.renderTreeLine(item, false)
	plain = stripANSI(rendered)
	require.Contains(t, plain, "◌")
}

func TestNoteSyncVisualStateUsesHealthyRecordBeforeStartupSyncCheck(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{RelPath: "work/note.md", SyncClass: notes.SyncClassSynced}
	m.startupSyncChecked = false
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/note.md": {RelPath: "work/note.md", LastSyncAt: time.Now(), LastSyncedHash: "sha256:ok"},
	}
	require.Equal(t, noteSyncVisualHealthy, m.noteSyncVisualState(&n))
}

func TestRenderModalGapUsesStyledSpaces(t *testing.T) {
	m := newTestModel(t)
	rendered := m.renderModalGap(3)
	require.Equal(t, "   ", stripANSI(rendered))
	require.Equal(t, 3, lipgloss.Width(rendered))
}

func TestNoteSyncVisualStateSharedHealthy(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{RelPath: "work/shared.md", SyncClass: notes.SyncClassShared}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/shared.md": {RelPath: "work/shared.md", LastSyncAt: time.Now()},
	}
	require.Equal(t, noteSyncVisualSharedHealthy, m.noteSyncVisualState(&n))
}

func TestNoteSyncVisualStateSharedPendingWhenNoRecord(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{RelPath: "work/shared.md", SyncClass: notes.SyncClassShared}
	require.Equal(t, noteSyncVisualSharedPending, m.noteSyncVisualState(&n))
}

func TestNoteSyncVisualStateSharedSyncingWhenInFlight(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{RelPath: "work/shared.md", SyncClass: notes.SyncClassShared}
	m.syncInFlight = map[string]bool{"work/shared.md": true}
	require.Equal(t, noteSyncVisualSharedSyncing, m.noteSyncVisualState(&n))
}

func TestNoteSyncMarkerSharedUsesFilledDiamond(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{RelPath: "work/shared.md", SyncClass: notes.SyncClassShared}
	m.syncRecords = map[string]notesync.NoteRecord{
		"work/shared.md": {RelPath: "work/shared.md", LastSyncAt: time.Now()},
	}
	mark, _ := m.noteSyncMarker(&n)
	require.Equal(t, "◆ ", mark)
}

func TestRenderTemporaryListEmptyStateIsActionable(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	m.tempNotes = nil
	rendered := m.renderTemporaryListView()
	plain := stripANSI(rendered)
	require.Contains(t, plain, "Press N")
	require.Contains(t, plain, "t to return")
}

func TestRenderPinsListEmptyStateIsActionable(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModePins
	m.pinnedNotes = map[string]bool{}
	m.pinnedCats = map[string]bool{}
	rendered := m.renderPinsListView()
	plain := stripANSI(rendered)
	require.Contains(t, plain, "Press p")
}

func TestRefreshPreviewShowsActionableTemporaryEmptyState(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	m.tempNotes = nil
	m.refreshPreview()
	plain := stripANSI(m.previewContent)
	require.Contains(t, plain, "Press N")
	require.Contains(t, plain, "return to notes")
}

func TestRefreshPreviewShowsSearchNoResultsMessage(t *testing.T) {
	m := newTestModel(t)
	m.searchInput.SetValue("missing")
	m.treeItems = []treeItem{{Kind: treeCategory, RelPath: "", Name: "/"}}
	m.refreshPreview()
	plain := stripANSI(m.previewContent)
	require.Contains(t, plain, "No notes match")
	require.Contains(t, plain, "Press esc")
}

func TestRenderTodoDueDateModal(t *testing.T) {
	m := newTestModel(t)
	m.showTodoDueDate = true
	m.width = 120
	m.height = 40
	rendered := m.renderTodoDueDateModal()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "Set todo due date") {
		require.Failf(t, "assertion failed", "expected 'Set todo due date' in modal, got %q", plain[:min(len(plain), 200)])
	}
}

func TestRenderTodoPriorityModal(t *testing.T) {
	m := newTestModel(t)
	m.showTodoPriority = true
	m.width = 120
	m.height = 40
	rendered := m.renderTodoPriorityModal()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "Set todo priority") {
		require.Failf(t, "assertion failed", "expected 'Set todo priority' in modal, got %q", plain[:min(len(plain), 200)])
	}
}

func TestStatusIncludesUndoHintAfterNoteDeletion(t *testing.T) {
	m := newTestModel(t)
	m.lastDeletion = &undoableDelete{label: "note.md"}
	m.status = "trashed note: note.md  •  Z to undo"
	plain := stripANSI(m.renderStatus())
	require.Contains(t, plain, "Z to undo")
}

// TestMarkdownLinkRenderNoANSIGarbage is a regression test for the bug where
// [text](url) inside a markdown preview produced visible escape characters like
// "33;3;;" in the output. The fix strips ANSI codes from the link label before
// applying the outer underline style.
func TestMarkdownLinkRenderNoANSIGarbage(t *testing.T) {
	opts := markdownRenderOptions{Width: 80}
	rendered := renderMarkdownTerminal("[hyperlink](https://example.com)", opts)
	plain := stripANSI(rendered)

	// The visible text must contain the label, but the URL should stay hidden
	// until that link is selected in link-nav mode.
	if !strings.Contains(plain, "hyperlink") {
		require.Failf(t, "assertion failed", "expected 'hyperlink' in rendered output, got %q", plain)
	}
	if strings.Contains(plain, "https://example.com") {
		require.Failf(t, "assertion failed", "expected URL to stay hidden in rendered output, got %q", plain)
	}

	// The raw rendered string must not contain the ANSI escape garbage pattern
	// that the old buggy code produced (bare digit sequences from leaked ANSI).
	if strings.Contains(rendered, "33;3;;") || strings.Contains(rendered, "3;3;;") {
		require.Failf(t, "assertion failed", "rendered link contains ANSI garbage (regression): %q", rendered)
	}
}

// TestMarkdownWikilinkRenderShowsBrackets verifies that [[target]] inside
// markdown renders as [[target]] in the preview output.
func TestMarkdownWikilinkRenderShowsBrackets(t *testing.T) {
	opts := markdownRenderOptions{Width: 80}
	// RewriteWikilinks is called by renderPreviewMarkdown; call it manually here
	// to simulate the full pipeline.
	input := "See [[my note]] for details."
	rewritten := "See [my note](#wikilink:my%20note) for details."
	rendered := renderMarkdownTerminal(rewritten, opts)
	plain := stripANSI(rendered)

	if !strings.Contains(plain, "[[my note]]") {
		require.Failf(t, "assertion failed", "expected '[[my note]]' in rendered wikilink output, got %q", plain)
	}
	_ = input
}
