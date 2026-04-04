package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/notes"
	notesync "atbuy/noteui/internal/sync"
)

const paletteMaxVisible = 12

// paletteCommand holds the definition for a single app command.
type paletteCommand struct {
	name     string
	desc     string
	category string
	action   string
}

const (
	cmdNewNote               = "new_note"
	cmdNewTemporaryNote      = "new_temporary_note"
	cmdNewTodoList           = "new_todo_list"
	cmdNewCategory           = "new_category"
	cmdMoveCurrent           = "move_current"
	cmdMarkCurrent           = "mark_current"
	cmdRenameCurrent         = "rename_current"
	cmdAddTags               = "add_tags"
	cmdTrashCurrent          = "trash_current"
	cmdTogglePin             = "toggle_pin"
	cmdToggleSort            = "toggle_sort"
	cmdRefresh               = "refresh"
	cmdTogglePrivacy         = "toggle_privacy"
	cmdToggleLineNumbers     = "toggle_line_numbers"
	cmdShowPins              = "show_pins"
	cmdShowHelp              = "show_help"
	cmdToggleTemporary       = "toggle_temporary"
	cmdToggleSync            = "toggle_sync"
	cmdMakeShared            = "make_shared"
	cmdSelectSyncProfile     = "select_sync_profile"
	cmdShowSyncDetails       = "show_sync_details"
	cmdResolveConflict       = "resolve_conflict"
	cmdDeleteRemoteKeepLocal = "delete_remote_keep_local"
	cmdImportCurrent         = "import_current"
	cmdImportAll             = "import_all"
	cmdSyncNow               = "sync_now"
	cmdToggleEncryption      = "toggle_encryption"
	cmdToggleTodo            = "toggle_todo"
	cmdAddTodo               = "add_todo"
	cmdDeleteTodo            = "delete_todo"
	cmdEditTodo              = "edit_todo"
)

// paletteKind identifies the type of an item in the command palette.
type paletteKind int

const (
	paletteKindNote     paletteKind = iota // note in the main tree
	paletteKindTempNote                    // note in .tmp/
	paletteKindCommand                     // named app command
)

// paletteItem is one entry shown in the command palette.
type paletteItem struct {
	kind  paletteKind
	title string     // primary display (note title)
	sub   string     // secondary display (relPath, or ".tmp/relPath")
	note  notes.Note // valid for paletteKindNote and paletteKindTempNote
	cmd   paletteCommand
}

// isConflictCopy reports whether relPath matches the conflict copy naming pattern
// produced by createConflict(): "base.conflict-YYYYMMDD-HHMMSS.ext".
// Conflict copies are excluded from the palette since they are conflict artifacts,
// not regular notes; the user resolves them from the tree view.
func isConflictCopy(relPath string) bool {
	name := filepath.Base(relPath)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	idx := strings.LastIndex(base, ".conflict-")
	if idx < 0 {
		return false
	}
	suffix := base[idx+len(".conflict-"):]
	return len(suffix) == 15 // "YYYYMMDD-HHMMSS"
}

func (m *Model) openCommandPalette() {
	items := make([]paletteItem, 0, len(m.notes)+len(m.tempNotes))
	for _, n := range m.notes {
		if isConflictCopy(n.RelPath) {
			continue
		}
		items = append(items, paletteItem{
			kind:  paletteKindNote,
			title: n.Title(),
			sub:   n.RelPath,
			note:  n,
		})
	}
	for _, n := range m.tempNotes {
		items = append(items, paletteItem{
			kind:  paletteKindTempNote,
			title: n.Title(),
			sub:   ".tmp/" + n.RelPath,
			note:  n,
		})
	}
	m.commandPaletteItems = items
	m.commandPaletteInput.Reset()
	m.commandPaletteInput.Focus()
	m.commandPaletteCursor = 0
	m.rebuildPaletteFiltered()
	m.showCommandPalette = true
}

func paletteCommands(m Model) []paletteCommand {
	cmds := []paletteCommand{
		{name: "New Note", desc: "Create a note in the current location", category: "notes", action: cmdNewNote},
		{name: "New Temporary Note", desc: "Create a temporary note", category: "notes", action: cmdNewTemporaryNote},
		{name: "Toggle Sort", desc: "Switch between alphabetical and modified sorting", category: "view", action: cmdToggleSort},
		{name: "Refresh", desc: "Refresh notes and sync state", category: "app", action: cmdRefresh},
		{name: "Toggle Preview Privacy", desc: "Toggle preview privacy", category: "preview", action: cmdTogglePrivacy},
		{name: "Toggle Preview Line Numbers", desc: "Toggle preview line numbers", category: "preview", action: cmdToggleLineNumbers},
		{name: "Show Help", desc: "Open the help modal", category: "app", action: cmdShowHelp},
		{name: "Show Pins", desc: "Toggle the pins view", category: "view", action: cmdShowPins},
	}

	cmds = appendPaletteCommand(cmds, m.listMode != listModePins,
		paletteCommand{name: "Toggle Temporary Notes", desc: "Switch between notes and temporary notes", category: "view", action: cmdToggleTemporary})
	cmds = appendPaletteCommand(cmds, m.listMode != listModeTemporary && m.listMode != listModePins,
		paletteCommand{name: "New Todo List", desc: "Create a new todo list in the current location", category: "notes", action: cmdNewTodoList})
	cmds = appendPaletteCommand(cmds, m.listMode == listModeNotes,
		paletteCommand{name: "New Category", desc: "Create a category in the notes tree", category: "tree", action: cmdNewCategory})
	cmds = appendPaletteCommand(cmds, m.canMoveCurrent(),
		paletteCommand{name: "Move Current Item", desc: "Move the selected item", category: "selection", action: cmdMoveCurrent})
	cmds = appendPaletteCommand(cmds, m.canMarkCurrent(),
		paletteCommand{name: "Mark Current Item", desc: "Mark or unmark the selected tree item", category: "selection", action: cmdMarkCurrent})
	cmds = appendPaletteCommand(cmds, m.canRenameCurrent(),
		paletteCommand{name: "Rename Current Item", desc: "Rename the selected item", category: "selection", action: cmdRenameCurrent})
	cmds = appendPaletteCommand(cmds, m.canAddTagsCurrent(),
		paletteCommand{name: "Add Tags", desc: "Add tags to the selected note", category: "selection", action: cmdAddTags})
	cmds = appendPaletteCommand(cmds, m.canTrashCurrent(),
		paletteCommand{name: "Trash Current Item", desc: "Trash the selected note or category", category: "selection", action: cmdTrashCurrent})
	cmds = appendPaletteCommand(cmds, m.canTogglePinCurrent(),
		paletteCommand{name: "Toggle Pin", desc: "Pin or unpin the current item", category: "selection", action: cmdTogglePin})
	cmds = appendPaletteCommand(cmds, m.canToggleSyncCurrent(),
		paletteCommand{name: "Toggle Note Sync", desc: "Toggle sync for the selected note", category: "sync", action: cmdToggleSync})
	cmds = appendPaletteCommand(cmds, m.canMakeSharedCurrent(),
		paletteCommand{name: "Make Note Shared", desc: "Make the selected note shared", category: "sync", action: cmdMakeShared})
	cmds = appendPaletteCommand(cmds, m.canSelectSyncProfile(),
		paletteCommand{name: "Select Sync Profile", desc: "Choose the default sync profile", category: "sync", action: cmdSelectSyncProfile})
	cmds = appendPaletteCommand(cmds, m.canShowSyncDetailsCurrent(),
		paletteCommand{name: "Show Sync Details", desc: "Open sync details for the selected note", category: "sync", action: cmdShowSyncDetails})
	cmds = appendPaletteCommand(cmds, m.canResolveConflictCurrent(),
		paletteCommand{name: "Resolve Conflict", desc: "Open conflict resolution for the selected note", category: "sync", action: cmdResolveConflict})
	cmds = appendPaletteCommand(cmds, m.canDeleteRemoteCopyCurrent(),
		paletteCommand{name: "Delete Remote Copy", desc: "Delete the remote copy and keep the local note", category: "sync", action: cmdDeleteRemoteKeepLocal})
	cmds = appendPaletteCommand(cmds, m.canImportCurrentRemoteNote(),
		paletteCommand{name: "Import Current Remote Note", desc: "Import the selected remote-only note", category: "sync", action: cmdImportCurrent})
	cmds = appendPaletteCommand(cmds, notesync.HasSyncProfile(m.cfg.Sync),
		paletteCommand{name: "Import All Remote Notes", desc: "Import all remote-only synced notes", category: "sync", action: cmdImportAll})
	cmds = appendPaletteCommand(cmds, notesync.HasSyncProfile(m.cfg.Sync),
		paletteCommand{name: "Sync Now", desc: "Run sync immediately", category: "sync", action: cmdSyncNow})
	cmds = appendPaletteCommand(cmds, m.canToggleEncryptionCurrent(),
		paletteCommand{name: "Toggle Note Encryption", desc: "Encrypt or decrypt the selected note", category: "notes", action: cmdToggleEncryption})
	cmds = appendPaletteCommand(cmds, m.canAddTodoItem(),
		paletteCommand{name: "Add Todo Item", desc: "Add a todo item to the current note", category: "todo", action: cmdAddTodo})
	cmds = appendPaletteCommand(cmds, m.canToggleCurrentTodo(),
		paletteCommand{name: "Toggle Current Todo", desc: "Toggle the selected todo item", category: "todo", action: cmdToggleTodo})
	cmds = appendPaletteCommand(cmds, m.canToggleCurrentTodo(),
		paletteCommand{name: "Delete Current Todo", desc: "Delete the selected todo item", category: "todo", action: cmdDeleteTodo})
	cmds = appendPaletteCommand(cmds, m.canToggleCurrentTodo(),
		paletteCommand{name: "Edit Current Todo", desc: "Edit the selected todo item", category: "todo", action: cmdEditTodo})

	return cmds
}

func appendPaletteCommand(cmds []paletteCommand, ok bool, cmd paletteCommand) []paletteCommand {
	if !ok {
		return cmds
	}
	return append(cmds, cmd)
}

func (m Model) canMoveCurrent() bool {
	if m.listMode == listModePins {
		return m.currentPinItem() != nil
	}
	if m.listMode == listModeTemporary {
		return m.currentTempNote() != nil
	}
	item := m.currentTreeItem()
	return item != nil && item.Kind != treeRemoteNote
}

func (m Model) canMarkCurrent() bool {
	if m.listMode != listModeNotes {
		return false
	}
	item := m.currentTreeItem()
	if item == nil {
		return false
	}
	return !(item.Kind == treeCategory && item.RelPath == "")
}

func (m Model) canRenameCurrent() bool {
	if m.listMode == listModePins {
		return m.currentPinItem() != nil
	}
	if m.listMode == listModeTemporary {
		return m.currentTempNote() != nil
	}
	item := m.currentTreeItem()
	if item == nil {
		return false
	}
	return item.Kind != treeRemoteNote && !(item.Kind == treeCategory && item.RelPath == "")
}

func (m Model) canAddTagsCurrent() bool {
	return m.currentRemoteOnlyNote() == nil && strings.TrimSpace(m.currentNotePath()) != ""
}

func (m Model) canTrashCurrent() bool {
	if m.listMode == listModePins {
		return m.currentPinItem() != nil
	}
	if m.listMode == listModeTemporary {
		return m.currentTempNote() != nil
	}
	item := m.currentTreeItem()
	if item == nil {
		return false
	}
	if item.Kind == treeRemoteNote {
		return false
	}
	return !(item.Kind == treeCategory && item.RelPath == "")
}

func (m Model) canTogglePinCurrent() bool {
	if m.listMode == listModePins {
		return m.currentPinItem() != nil
	}
	if m.listMode == listModeTemporary {
		return m.currentTempNote() != nil
	}
	item := m.currentTreeItem()
	return item != nil && item.Kind != treeRemoteNote && !(item.Kind == treeCategory && item.RelPath == "")
}

func (m Model) canToggleSyncCurrent() bool {
	if m.listMode != listModeNotes {
		return false
	}
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		return false
	}
	return item.Note.SyncClass != notes.SyncClassShared
}

func (m Model) canMakeSharedCurrent() bool {
	if m.listMode != listModeNotes {
		return false
	}
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		return false
	}
	return item.Note.SyncClass != notes.SyncClassShared
}

func (m Model) canSelectSyncProfile() bool {
	return len(sortedSyncProfileNames(m.cfg.Sync)) > 0
}

func (m Model) canShowSyncDetailsCurrent() bool {
	_, ok := m.currentSyncDebugDetails()
	return ok
}

func (m Model) canResolveConflictCurrent() bool {
	return strings.TrimSpace(m.currentConflictCopyPath()) != ""
}

func (m Model) canDeleteRemoteCopyCurrent() bool {
	if m.listMode != listModeNotes {
		return false
	}
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		return false
	}
	if item.Note.SyncClass != notes.SyncClassSynced {
		return false
	}
	_, ok := m.syncRecords[filepath.ToSlash(item.Note.RelPath)]
	return ok
}

func (m Model) canImportCurrentRemoteNote() bool {
	return m.currentRemoteOnlyNote() != nil
}

func (m Model) canToggleEncryptionCurrent() bool {
	return m.currentRemoteOnlyNote() == nil && strings.TrimSpace(m.currentNotePath()) != ""
}

func (m Model) canAddTodoItem() bool {
	return strings.TrimSpace(m.previewPath) != ""
}

func (m Model) canToggleCurrentTodo() bool {
	return len(m.previewTodos) > 0 && m.previewTodoCursor >= 0 && m.previewTodoCursor < len(m.previewTodos)
}

func (m *Model) rebuildPaletteFiltered() {
	query := strings.TrimSpace(m.commandPaletteInput.Value())
	filtered := make([]paletteItem, 0, len(m.commandPaletteItems)+32)

	for _, item := range m.commandPaletteItems {
		if query == "" || m.noteMatches(item.note, query) {
			filtered = append(filtered, item)
		}
	}
	for _, cmd := range paletteCommands(*m) {
		if query == "" || paletteCommandMatches(cmd, query) {
			filtered = append(filtered, paletteItem{
				kind:  paletteKindCommand,
				title: cmd.name,
				sub:   cmd.category,
				cmd:   cmd,
			})
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return paletteItemLess(filtered[i], filtered[j], query)
	})

	m.commandPaletteFiltered = filtered
	if m.commandPaletteCursor >= len(m.commandPaletteFiltered) {
		m.commandPaletteCursor = max(0, len(m.commandPaletteFiltered)-1)
	}
}

func paletteCommandMatches(cmd paletteCommand, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	blob := strings.ToLower(strings.Join([]string{cmd.name, cmd.desc, cmd.category}, " "))
	for term := range strings.FieldsSeq(q) {
		if !strings.Contains(blob, term) {
			return false
		}
	}
	return true
}

func paletteItemLess(a, b paletteItem, query string) bool {
	ab, bb := paletteItemBucket(a, query), paletteItemBucket(b, query)
	if ab != bb {
		return ab < bb
	}
	ap, bp := paletteItemTypePriority(a), paletteItemTypePriority(b)
	if ap != bp {
		return ap < bp
	}
	at, bt := strings.ToLower(a.title), strings.ToLower(b.title)
	if at != bt {
		return at < bt
	}
	return strings.ToLower(a.sub) < strings.ToLower(b.sub)
}

func paletteItemBucket(item paletteItem, query string) int {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return 4
	}
	title := strings.ToLower(item.title)
	sub := strings.ToLower(item.sub)
	if title == q {
		return 0
	}
	if strings.HasPrefix(title, q) {
		return 1
	}
	if strings.Contains(title, q) {
		return 2
	}
	if item.kind == paletteKindCommand && strings.Contains(strings.ToLower(item.cmd.desc), q) {
		return 3
	}
	if strings.Contains(sub, q) {
		return 4
	}
	return 5
}

func paletteItemTypePriority(item paletteItem) int {
	if item.kind == paletteKindCommand {
		return 0
	}
	if item.kind == paletteKindNote {
		return 1
	}
	return 2
}

func (m *Model) tabCompletePalette() {
	if len(m.commandPaletteFiltered) == 0 {
		return
	}
	item := m.commandPaletteFiltered[m.commandPaletteCursor]
	m.commandPaletteInput.SetValue(item.title)
	m.commandPaletteInput.CursorEnd()
	m.rebuildPaletteFiltered()
}

func (m *Model) commitPaletteSelection() tea.Cmd {
	if len(m.commandPaletteFiltered) == 0 {
		return nil
	}
	item := m.commandPaletteFiltered[m.commandPaletteCursor]
	m.showCommandPalette = false
	m.commandPaletteInput.Blur()

	switch item.kind {
	case paletteKindNote:
		m.switchToNotesMode()
		m.selectTreeNote(item.note.RelPath)
	case paletteKindTempNote:
		m.switchToTemporaryMode()
		m.selectTemporaryNote(item.note.RelPath)
	case paletteKindCommand:
		return m.executePaletteCommand(item.cmd.action)
	}
	return nil
}

func (m *Model) openHelpModal() {
	m.showHelp = true
	m.helpScroll = 0
	m.helpMouseSuppressed = false
	m.helpInput.SetValue("")
	m.helpInput.Focus()
	m.rebuildHelpRowsCache()
	m.rebuildHelpModalCache(max(8, min(20, m.height-16)))
	m.pendingG = false
	m.status = "help"
}

func (m *Model) toggleSortOrder() {
	m.sortByModTime = !m.sortByModTime
	_ = m.saveTreeState()
	m.rebuildTree()
	if m.sortByModTime {
		m.status = "sorting by modified time"
	} else {
		m.status = "sorting alphabetically"
	}
}

func (m *Model) startRefresh() tea.Cmd {
	m.status = "refreshing..."
	return batchCmds(refreshAllCmd(m.rootDir), m.scheduleSync())
}

func (m *Model) openCreateCategory() {
	if m.listMode != listModeNotes {
		m.status = "categories only available in notes tree"
		return
	}
	m.showCreateCategory = true
	m.categoryInput.SetValue(m.currentCategoryPrefix())
	m.categoryInput.Focus()
	m.categoryInput.CursorEnd()
	m.status = "new category"
}

func (m *Model) startNewNote() tea.Cmd {
	if m.listMode == listModeTemporary {
		return createTemporaryNoteCmd(m.rootDir)
	}
	if m.listMode == listModePins {
		m.status = "press enter to jump to item first"
		return nil
	}
	return createNoteCmd(m.rootDir, m.currentTargetDir())
}

func (m *Model) startNewTodoList() tea.Cmd {
	if m.listMode == listModeTemporary || m.listMode == listModePins {
		m.status = "todo lists only available in notes tree"
		return nil
	}
	return createTodoNoteCmd(m.rootDir, m.currentTargetDir())
}

func (m *Model) togglePreviewPrivacy() {
	if m.cfg.Preview.Privacy {
		m.status = "preview privacy forced by config"
		return
	}
	m.previewPrivacyEnabled = !m.previewPrivacyEnabled
	m.previewPath = ""
	if m.previewPrivacyEnabled {
		m.status = "preview privacy enabled"
	} else {
		m.status = "preview privacy disabled"
	}
	m.refreshPreview()
}

func (m *Model) togglePreviewLineNumbers() {
	m.previewLineNumbersEnabled = !m.previewLineNumbersEnabled
	m.previewPath = ""
	if m.previewLineNumbersEnabled {
		m.status = "preview line numbers enabled"
	} else {
		m.status = "preview line numbers disabled"
	}
	m.refreshPreview()
}

func (m *Model) toggleNoteSyncCurrent() tea.Cmd {
	item := m.currentTreeItem()
	if item != nil && item.Kind == treeRemoteNote {
		m.status = "note is only on the server; press i to import it or I to import all"
		return nil
	}
	if item == nil || item.Kind != treeNote || item.Note == nil {
		m.status = "sync toggle only works on notes"
		return nil
	}
	if item.Note.SyncClass == notes.SyncClassShared {
		m.status = "shared notes cannot be toggled"
		return nil
	}
	return toggleNoteSyncCmd(item.Note.Path)
}

func (m *Model) makeCurrentNoteShared() tea.Cmd {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		m.status = "make shared only works on notes"
		return nil
	}
	if item.Note.SyncClass == notes.SyncClassShared {
		m.status = "note is already shared"
		return nil
	}
	return makeNoteSharedCmd(item.Note.Path)
}

func (m *Model) deleteRemoteCopyCurrent() tea.Cmd {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		m.status = "remote delete only works on synced local notes"
		return nil
	}
	if item.Note.SyncClass != notes.SyncClassSynced {
		m.status = "remote delete only works on synced local notes"
		return nil
	}
	if _, ok := m.syncRecords[filepath.ToSlash(item.Note.RelPath)]; !ok {
		m.status = "note is not linked to a remote copy"
		return nil
	}
	m.status = "deleting remote copy..."
	return batchCmds(
		deleteRemoteNoteKeepLocalCmd(m.rootDir, item.Note.Path, m.cfg.Sync),
		m.startSyncVisual(item.Note.RelPath),
	)
}

func (m *Model) importCurrentRemoteNote() tea.Cmd {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeRemoteNote || item.RemoteNote == nil {
		m.status = "single-note import only works on remote notes"
		return nil
	}
	m.status = "importing remote note..."
	return batchCmds(
		importCurrentSyncedNoteCmd(m.rootDir, m.cfg.Sync, item.RemoteNote.ID),
		m.startSyncVisual(remoteOnlySyncVisualKey(item.RemoteNote.ID)),
	)
}

func (m *Model) importAllRemoteNotes() tea.Cmd {
	m.status = "importing synced notes..."
	return importSyncedNotesCmd(m.rootDir, m.cfg.Sync)
}

func (m *Model) startImmediateSync() tea.Cmd {
	if !notesync.HasSyncProfile(m.cfg.Sync) {
		return nil
	}
	m.status = "syncing..."
	return m.startSyncRun()
}

func (m *Model) armAddTodoItem() {
	path := m.previewPath
	if path == "" {
		m.status = "no note selected"
		return
	}
	m.showTodoAdd = true
	m.todoInput.SetValue("")
	m.todoInput.Focus()
	m.status = "add todo"
}

func (m *Model) trashCurrentItem() tea.Cmd {
	m.armDeleteCurrent()
	if m.deletePending == nil {
		return nil
	}
	cmd := m.confirmDeleteCurrent()
	m.deletePending = nil
	if cmd != nil {
		m.status = "deleting..."
	}
	return cmd
}

func (m *Model) executePaletteCommand(action string) tea.Cmd {
	switch action {
	case cmdNewNote:
		return m.startNewNote()
	case cmdNewTemporaryNote:
		return createTemporaryNoteCmd(m.rootDir)
	case cmdNewTodoList:
		return m.startNewTodoList()
	case cmdNewCategory:
		m.openCreateCategory()
		return nil
	case cmdMoveCurrent:
		m.armMoveCurrent()
		return nil
	case cmdMarkCurrent:
		m.toggleMarkCurrent()
		return nil
	case cmdRenameCurrent:
		m.armRenameCurrent()
		return nil
	case cmdAddTags:
		m.armAddTagCurrent()
		return nil
	case cmdTrashCurrent:
		return m.trashCurrentItem()
	case cmdTogglePin:
		if err := m.togglePinCurrent(); err != nil {
			m.status = "pin failed: " + err.Error()
			return nil
		}
		return m.scheduleSync()
	case cmdToggleSort:
		m.toggleSortOrder()
		return nil
	case cmdRefresh:
		return m.startRefresh()
	case cmdTogglePrivacy:
		m.togglePreviewPrivacy()
		return nil
	case cmdToggleLineNumbers:
		m.togglePreviewLineNumbers()
		return nil
	case cmdShowPins:
		m.togglePinsMode()
		return nil
	case cmdShowHelp:
		m.openHelpModal()
		return nil
	case cmdToggleTemporary:
		if m.listMode != listModePins {
			m.toggleNotesTemporaryMode()
		}
		return nil
	case cmdToggleSync:
		return m.toggleNoteSyncCurrent()
	case cmdMakeShared:
		return m.makeCurrentNoteShared()
	case cmdSelectSyncProfile:
		m.openSyncProfilePicker()
		return nil
	case cmdShowSyncDetails:
		m.openCurrentSyncDebugModal()
		return nil
	case cmdResolveConflict:
		return m.openCurrentConflictCopy()
	case cmdDeleteRemoteKeepLocal:
		return m.deleteRemoteCopyCurrent()
	case cmdImportCurrent:
		return m.importCurrentRemoteNote()
	case cmdImportAll:
		return m.importAllRemoteNotes()
	case cmdSyncNow:
		return m.startImmediateSync()
	case cmdToggleEncryption:
		m.armToggleEncryption()
		return nil
	case cmdToggleTodo:
		return m.toggleCurrentPreviewTodo()
	case cmdAddTodo:
		m.armAddTodoItem()
		return nil
	case cmdDeleteTodo:
		return m.deleteCurrentPreviewTodo()
	case cmdEditTodo:
		return m.armEditCurrentPreviewTodo()
	}
	return nil
}

func (m Model) renderCommandPaletteModal() string {
	modalWidth, innerWidth := m.modalDimensions(60, 90)
	total := len(m.commandPaletteFiltered)

	// Compute scroll so the cursor row is always visible.
	scroll := max(0, m.commandPaletteCursor-paletteMaxVisible+1)
	scroll = max(0, min(scroll, max(0, total-paletteMaxVisible)))

	titleLeft := lipgloss.NewStyle().
		Foreground(modalAccentColor).
		Background(modalBgColor).
		Bold(true).
		Render("Command palette")
	countText := lipgloss.NewStyle().
		Foreground(modalMutedColor).
		Background(modalBgColor).
		Render(fmt.Sprintf("%d results", total))
	gapSize := max(0, innerWidth-lipgloss.Width(titleLeft)-lipgloss.Width(countText))
	gapStr := lipgloss.NewStyle().Background(modalBgColor).Render(strings.Repeat(" ", gapSize))
	titleRow := fillWidthBackground(titleLeft+gapStr+countText, innerWidth, modalBgColor)

	inputCopy := m.commandPaletteInput
	inputCopy.Width = max(12, innerWidth-lipgloss.Width(inputCopy.Prompt))
	inputCopy.TextStyle = lipgloss.NewStyle().Foreground(modalTextColor).Background(modalBgColor)
	inputCopy.PlaceholderStyle = lipgloss.NewStyle().Foreground(modalMutedColor).Background(modalBgColor)
	inputCopy.Cursor.Style = lipgloss.NewStyle().Foreground(modalTextColor).Background(modalTextColor)
	inputRow := fillWidthBackground(inputCopy.View(), innerWidth, modalBgColor)

	divider := lipgloss.NewStyle().
		Width(innerWidth).
		Foreground(modalMutedColor).
		Background(modalBgColor).
		Render(strings.Repeat("─", innerWidth))

	var resultLines []string
	if total == 0 {
		empty := lipgloss.NewStyle().
			Width(innerWidth).
			Foreground(modalMutedColor).
			Background(modalBgColor).
			Render("No matches")
		resultLines = append(resultLines, empty)
	} else {
		end := min(scroll+paletteMaxVisible, total)
		for i := scroll; i < end; i++ {
			resultLines = append(resultLines, m.renderPaletteRow(m.commandPaletteFiltered[i], i, innerWidth))
		}
		if scroll > 0 {
			indicator := lipgloss.NewStyle().
				Width(innerWidth).Align(lipgloss.Center).
				Foreground(modalMutedColor).Background(modalBgColor).
				Render(fmt.Sprintf("↑ %d more", scroll))
			resultLines = append([]string{indicator}, resultLines...)
		}
		if end < total {
			indicator := lipgloss.NewStyle().
				Width(innerWidth).Align(lipgloss.Center).
				Foreground(modalMutedColor).Background(modalBgColor).
				Render(fmt.Sprintf("↓ %d more", total-end))
			resultLines = append(resultLines, indicator)
		}
	}

	resultsBlock := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(strings.Join(resultLines, "\n"))

	footer := m.renderModalFooter("tab complete · ↑↓ navigate · enter open/run · esc cancel", innerWidth)

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			titleRow,
			m.renderModalBlank(innerWidth),
			inputRow,
			m.renderModalBlank(innerWidth),
			divider,
			resultsBlock,
			divider,
			m.renderModalBlank(innerWidth),
			footer,
		))

	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderPaletteRow(item paletteItem, idx, innerWidth int) string {
	selected := idx == m.commandPaletteCursor

	cursor := "  "
	if selected {
		cursor = "› "
	}
	cursorW := lipgloss.Width(cursor)
	subWidth := max(8, min(36, innerWidth/3))
	titleWidth := max(8, innerWidth-subWidth-cursorW-2)

	titleStr := paletteTruncate(item.title, titleWidth)
	subStr := paletteTruncateLeft(item.sub, subWidth)

	if selected {
		subFg := selectedFgColor
		if item.kind == paletteKindCommand {
			subFg = modalMutedColor
		}
		return lipgloss.NewStyle().Foreground(selectedFgColor).Background(selectedBgColor).Render(cursor) +
			lipgloss.NewStyle().Width(titleWidth).Foreground(selectedFgColor).Background(selectedBgColor).Render(titleStr) +
			lipgloss.NewStyle().Width(2).Background(selectedBgColor).Render("  ") +
			lipgloss.NewStyle().Width(subWidth).Align(lipgloss.Right).Foreground(subFg).Background(selectedBgColor).Render(subStr)
	}
	return lipgloss.NewStyle().Foreground(modalMutedColor).Background(modalBgColor).Render(cursor) +
		lipgloss.NewStyle().Width(titleWidth).Foreground(modalTextColor).Background(modalBgColor).Render(titleStr) +
		lipgloss.NewStyle().Width(2).Background(modalBgColor).Render("  ") +
		lipgloss.NewStyle().Width(subWidth).Align(lipgloss.Right).Foreground(modalMutedColor).Background(modalBgColor).Render(subStr)
}

// paletteTruncate truncates s to at most maxWidth visual characters.
func paletteTruncate(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// paletteTruncateLeft truncates s from the left to at most maxWidth visual characters.
func paletteTruncateLeft(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width("…"+string(runes)) > maxWidth {
		runes = runes[1:]
	}
	return "…" + string(runes)
}
