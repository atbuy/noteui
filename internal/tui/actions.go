package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"atbuy/noteui/internal/notes"
)

func (m *Model) armMoveCurrent() {
	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil {
			return
		}

		switch item.Kind {
		case pinItemCategory:
			m.showMove = true
			m.moveInput.Focus()
			m.movePending = &movePending{
				kind:       moveTargetCategory,
				oldRelPath: item.RelPath,
				name:       item.Name,
			}
			m.moveInput.SetValue(item.RelPath)
			m.moveInput.CursorEnd()
			m.status = "move pinned category"
			return

		case pinItemNote, pinItemTemporaryNote:
			m.showMove = true
			m.moveInput.Focus()
			m.movePending = &movePending{
				kind:       moveTargetNote,
				oldRelPath: item.RelPath,
				name:       item.Name,
			}
			m.moveInput.SetValue(item.RelPath)
			m.moveInput.CursorEnd()
			m.status = "move pinned note"
			return
		}
	}

	if m.listMode == listModeTemporary {
		n := m.currentTempNote()
		if n == nil {
			return
		}
		m.showMove = true
		m.moveInput.Focus()
		m.movePending = &movePending{
			kind:       moveTargetNote,
			oldRelPath: n.RelPath,
			name:       n.Title(),
		}
		m.moveInput.SetValue(n.RelPath)
		m.moveInput.CursorEnd()
		m.status = "move temporary note"
		return
	}

	item := m.currentTreeItem()
	if item == nil {
		return
	}
	if item.Kind == treeRemoteNote {
		m.status = "note is only on the server; press i to import it or I to import all"
		return
	}

	m.openMoveBrowser()
}

func (m Model) confirmMove(newRelPath string) tea.Cmd {
	if m.movePending == nil {
		return nil
	}

	switch m.movePending.kind {
	case moveTargetNote:
		root := m.rootDir

		switch m.listMode {
		case listModeTemporary:
			root = notes.TempRoot(m.rootDir)
		case listModePins:
			item := m.currentPinItem()
			if item != nil && item.Kind == pinItemTemporaryNote {
				root = notes.TempRoot(m.rootDir)
			}
		}
		return moveNoteCmd(root, m.movePending.oldRelPath, newRelPath)
	case moveTargetCategory:
		return moveCategoryCmd(m.rootDir, m.movePending.oldRelPath, newRelPath)
	default:
		return nil
	}
}

func (m *Model) armDeleteCurrent() {
	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil {
			return
		}

		switch item.Kind {
		case pinItemCategory:
			m.deletePending = &deletePending{
				kind:    deleteTargetCategory,
				relPath: item.RelPath,
				name:    item.Name,
			}
			m.status = "press d again to trash pinned category: " + item.Name
			return

		case pinItemNote, pinItemTemporaryNote:
			m.deletePending = &deletePending{
				kind:    deleteTargetNote,
				relPath: item.Path,
				name:    item.Name,
			}
			m.status = "press d again to trash pinned note: " + item.Name
			return
		}
	}

	if m.listMode == listModeTemporary {
		refs, err := m.selectedTempNotesForAction()
		if err != nil {
			m.status = err.Error()
			return
		}
		paths := make([]string, 0, len(refs))
		for _, ref := range refs {
			paths = append(paths, ref.path)
		}
		m.deletePending = &deletePending{kind: deleteTargetNote, notePaths: paths}
		m.status = countStatus(len(paths), "press d again to trash temporary note", "press d again to trash %d temporary notes")
		return
	}

	if m.listMode == listModeNotes && m.hasMarksInCurrentMode() {
		refs, err := m.selectedMainNotesForAction()
		if err != nil {
			m.status = err.Error()
			return
		}
		paths := make([]string, 0, len(refs))
		for _, ref := range refs {
			paths = append(paths, ref.path)
		}
		m.deletePending = &deletePending{kind: deleteTargetNote, notePaths: paths}
		m.status = countStatus(len(paths), "press d again to trash note", "press d again to trash %d notes")
		return
	}

	item := m.currentTreeItem()
	if item == nil {
		return
	}

	switch item.Kind {
	case treeCategory:
		if item.RelPath == "" {
			m.status = "cannot delete root category"
			return
		}
		m.deletePending = &deletePending{
			kind:    deleteTargetCategory,
			relPath: item.RelPath,
			name:    item.Name,
		}
		m.status = "press d again to trash category: " + item.Name

	case treeNote:
		if item.Note == nil {
			return
		}
		m.deletePending = &deletePending{
			kind:    deleteTargetNote,
			relPath: item.Note.Path,
			name:    item.Note.Name,
		}
		m.status = "press d again to trash note: " + item.Note.Name
	case treeRemoteNote:
		m.status = "note is only on the server; press i to import it or I to import all"
	}
}

func (m Model) confirmDeleteCurrent() tea.Cmd {
	if m.deletePending == nil {
		return nil
	}

	switch m.deletePending.kind {
	case deleteTargetNote:
		if len(m.deletePending.notePaths) > 0 {
			return deleteNotesCmd(m.deletePending.notePaths)
		}
		return deleteNoteCmd(m.deletePending.relPath)
	case deleteTargetCategory:
		return deleteCategoryCmd(m.rootDir, m.deletePending.relPath)
	default:
		return nil
	}
}

func (m *Model) armRenameCurrent() {
	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil {
			return
		}

		switch item.Kind {
		case pinItemCategory:
			m.showRename = true
			m.renamePending = &renamePending{
				kind:    renameTargetCategory,
				relPath: item.RelPath,
				oldName: item.Name,
			}
			m.renameInput.SetValue(item.RelPath)
			m.renameInput.Focus()
			m.renameInput.CursorEnd()
			m.status = "rename pinned category"
			return

		case pinItemNote, pinItemTemporaryNote:
			m.showRename = true
			m.renamePending = &renamePending{
				kind:     renameTargetNote,
				path:     item.Path,
				oldTitle: item.Name,
			}
			m.renameInput.SetValue(item.Name)
			m.renameInput.Focus()
			m.renameInput.CursorEnd()
			m.status = "rename pinned note"
			return
		}
	}

	if m.listMode == listModeTemporary {
		n := m.currentTempNote()
		if n == nil {
			return
		}
		m.showRename = true
		m.renamePending = &renamePending{
			kind:     renameTargetNote,
			path:     n.Path,
			oldTitle: n.Title(),
		}
		m.renameInput.SetValue(n.Title())
		m.renameInput.Focus()
		m.renameInput.CursorEnd()
		m.status = "rename temporary note"
		return
	}

	item := m.currentTreeItem()
	if item == nil {
		return
	}

	switch item.Kind {
	case treeNote:
		if item.Note == nil {
			return
		}
		m.showRename = true
		m.renamePending = &renamePending{
			kind:     renameTargetNote,
			path:     item.Note.Path,
			oldTitle: item.Note.Title(),
		}
		m.renameInput.SetValue(item.Note.Title())
		m.renameInput.Focus()
		m.renameInput.CursorEnd()
		m.status = "rename note"

	case treeCategory:
		if item.RelPath == "" {
			m.status = "cannot rename root category"
			return
		}
		m.showRename = true
		m.renamePending = &renamePending{
			kind:    renameTargetCategory,
			relPath: item.RelPath,
			oldName: item.Name,
		}
		m.renameInput.SetValue(item.RelPath)
		m.renameInput.Focus()
		m.renameInput.CursorEnd()
		m.status = "rename category"
	case treeRemoteNote:
		m.status = "note is only on the server; press i to import it or I to import all"
	}
}

func (m *Model) toggleCurrentPreviewTodo() tea.Cmd {
	if len(m.previewTodos) == 0 {
		m.status = "no todos"
		return nil
	}
	if m.previewTodoCursor < 0 || m.previewTodoCursor >= len(m.previewTodos) {
		return nil
	}
	todo := m.previewTodos[m.previewTodoCursor]
	return toggleTodoCmd(m.previewPath, todo.rawLine)
}

func (m *Model) deleteCurrentPreviewTodo() tea.Cmd {
	if len(m.previewTodos) == 0 {
		m.status = "no todos"
		return nil
	}
	if m.previewTodoCursor < 0 || m.previewTodoCursor >= len(m.previewTodos) {
		return nil
	}
	todo := m.previewTodos[m.previewTodoCursor]
	return deleteTodoCmd(m.previewPath, todo.rawLine)
}

func (m *Model) armEditCurrentPreviewTodo() tea.Cmd {
	if len(m.previewTodos) == 0 {
		m.status = "no todos"
		return nil
	}
	if m.previewTodoCursor < 0 || m.previewTodoCursor >= len(m.previewTodos) {
		return nil
	}
	todo := m.previewTodos[m.previewTodoCursor]
	m.showTodoEdit = true
	m.todoInput.SetValue(todo.text)
	m.todoInput.Focus()
	m.todoInput.CursorEnd()
	m.status = "edit todo"
	return nil
}

func (m Model) currentPreviewTodoSelection() (path string, rawLine int, text string, ok bool) {
	if strings.TrimSpace(m.previewPath) == "" {
		return "", 0, "", false
	}
	if m.previewTodoCursor < 0 || m.previewTodoCursor >= len(m.previewTodos) {
		return "", 0, "", false
	}
	todo := m.previewTodos[m.previewTodoCursor]
	return m.previewPath, todo.rawLine, todo.text, true
}

func (m *Model) armSetCurrentTodoDueDate() {
	_, _, text, ok := m.currentPreviewTodoSelection()
	if !ok {
		m.status = "no todo selected"
		return
	}
	_, metadata := notes.ParseTodoMetadata(text)
	m.showTodoDueDate = true
	m.dueDateInput.SetValue(metadata.DueDate)
	m.dueDateInput.Focus()
	m.dueDateInput.CursorEnd()
	m.status = "set todo due date"
}

func (m *Model) armSetCurrentTodoPriority() {
	_, _, text, ok := m.currentPreviewTodoSelection()
	if !ok {
		m.status = "no todo selected"
		return
	}
	_, metadata := notes.ParseTodoMetadata(text)
	m.showTodoPriority = true
	if metadata.Priority > 0 {
		m.priorityInput.SetValue(fmt.Sprintf("%d", metadata.Priority))
	} else {
		m.priorityInput.SetValue("")
	}
	m.priorityInput.Focus()
	m.priorityInput.CursorEnd()
	m.status = "set todo priority"
}

func (m *Model) toggleNotesTemporaryMode() {
	if m.listMode == listModeTemporary {
		m.switchToNotesMode()
	} else {
		m.switchToTemporaryMode()
	}
}

func (m *Model) switchToNotesMode() {
	m.listMode = listModeNotes
	m.lastNonPinsMode = listModeNotes
	m.status = "notes"
	m.syncSelectedNote()
}

func (m *Model) switchToTemporaryMode() {
	m.listMode = listModeTemporary
	m.lastNonPinsMode = listModeTemporary
	m.status = "temporary"
	m.syncSelectedNote()
}

func (m *Model) togglePinsMode() {
	if m.listMode == listModePins {
		if m.lastNonPinsMode == listModeTemporary {
			m.switchToTemporaryMode()
		} else {
			m.switchToNotesMode()
		}
		return
	}

	if m.listMode == listModeTemporary {
		m.lastNonPinsMode = listModeTemporary
	} else {
		m.lastNonPinsMode = listModeNotes
	}

	m.listMode = listModePins
	m.status = "pins"
	m.syncSelectedNote()
}

func (m Model) currentTempNote() *notes.Note {
	items := m.filteredTempNotes()
	if len(items) == 0 || m.tempCursor < 0 || m.tempCursor >= len(items) {
		return nil
	}
	n := items[m.tempCursor]
	return &n
}

func (m *Model) moveTempCursor(delta int) {
	items := m.filteredTempNotes()
	if len(items) == 0 {
		return
	}
	next := max(m.tempCursor+delta, 0)
	if next >= len(items) {
		next = len(items) - 1
	}
	m.tempCursor = next
	m.syncSelectedNote()
}

func (m Model) filteredTempNotes() []notes.Note {
	query := strings.TrimSpace(strings.ToLower(m.searchInput.Value()))

	var out []notes.Note
	if query == "" {
		out = make([]notes.Note, len(m.tempNotes))
		copy(out, m.tempNotes)
		if m.sortByModTime {
			sort.SliceStable(out, func(i, j int) bool {
				return out[i].ModTime.After(out[j].ModTime)
			})
		}
	} else {
		out = filterAndScoreNotes(m.tempNotes, query)
	}

	return out
}

func (m Model) currentCategoryPrefix() string {
	item := m.currentTreeItem()
	if item == nil {
		return ""
	}

	switch item.Kind {
	case treeCategory:
		if item.RelPath == "" {
			return ""
		}
		return item.RelPath + string(filepath.Separator)
	case treeNote:
		if item.Note == nil {
			return ""
		}
		dir := filepath.Dir(item.Note.RelPath)
		if dir == "." || dir == "" {
			return ""
		}
		return dir + string(filepath.Separator)
	case treeRemoteNote:
		dir := filepath.Dir(item.RelPath)
		if dir == "." || dir == "" {
			return ""
		}
		return dir + string(filepath.Separator)
	}

	return ""
}

func parseTagInput(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(part), "#"))
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tag)
	}
	return out
}

func (m *Model) armAddTagCurrent() {
	if m.blockRemoteOnlyAction() {
		return
	}
	if _, err := m.selectedTaggableNotePaths(); err != nil {
		m.status = err.Error()
		return
	}
	m.showAddTag = true
	m.tagInput.SetValue("")
	m.tagInput.Focus()
	m.tagInput.CursorEnd()
	m.status = "add tag"
}

func (m Model) currentNotePath() string {
	if m.listMode == listModeTodos {
		item := m.currentTodoItem()
		if item == nil {
			return ""
		}
		return item.Note.Path
	}

	if m.listMode == listModeTemporary {
		n := m.currentTempNote()
		if n == nil {
			return ""
		}
		return n.Path
	}

	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil {
			return ""
		}
		if item.Kind == pinItemNote || item.Kind == pinItemTemporaryNote {
			return item.Path
		}
		return ""
	}

	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		return ""
	}
	return item.Note.Path
}

func (m Model) currentLocalNote() *notes.Note {
	if m.listMode == listModeTodos {
		item := m.currentTodoItem()
		if item == nil {
			return nil
		}
		noteCopy := item.Note
		return &noteCopy
	}

	if m.listMode == listModeTemporary {
		return m.currentTempNote()
	}

	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil || (item.Kind != pinItemNote && item.Kind != pinItemTemporaryNote) {
			return nil
		}
		for _, note := range m.notes {
			if note.Path == item.Path {
				noteCopy := note
				return &noteCopy
			}
		}
		for _, note := range m.tempNotes {
			if note.Path == item.Path {
				noteCopy := note
				return &noteCopy
			}
		}
		return nil
	}

	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		return nil
	}
	noteCopy := *item.Note
	return &noteCopy
}

func (m Model) currentConflictCopyPath() string {
	note := m.currentLocalNote()
	if note == nil || (note.SyncClass != notes.SyncClassSynced && note.SyncClass != notes.SyncClassShared) {
		return ""
	}
	rec, ok := m.syncRecords[filepath.ToSlash(strings.TrimSpace(note.RelPath))]
	if !ok || rec.Conflict == nil {
		return ""
	}
	copyPath := filepath.FromSlash(strings.TrimSpace(rec.Conflict.CopyPath))
	if copyPath == "" {
		return ""
	}
	return filepath.Join(m.rootDir, copyPath)
}

func (m Model) hasConflictCopyForCurrentSelection() bool {
	return m.currentConflictCopyPath() != ""
}

func (m *Model) openCurrentConflictCopy() tea.Cmd {
	copyPath := m.currentConflictCopyPath()
	if copyPath == "" {
		m.status = "conflict resolution only works on conflicted synced notes"
		return nil
	}
	_ = copyPath
	m.openCurrentSyncDebugModal()
	return nil
}

func (m *Model) armToggleEncryption() {
	if m.blockRemoteOnlyAction() {
		return
	}
	path := m.currentNotePath()
	if path == "" {
		m.status = "no note selected"
		return
	}

	raw, err := notes.ReadAll(path)
	if err != nil {
		m.status = "error reading note: " + err.Error()
		return
	}

	m.pendingEncryptPath = path

	if notes.NoteIsEncrypted(raw) {
		m.passphraseModalCtx = "decrypt"
	} else {
		m.passphraseModalCtx = "encrypt"
	}

	if m.sessionPassphrase == "" {
		m.showPassphraseModal = true
		m.passphraseInput.SetValue("")
		m.passphraseInput.Focus()
		if m.passphraseModalCtx == "encrypt" {
			m.status = "enter passphrase to encrypt"
		} else {
			m.status = "enter passphrase to decrypt"
		}
		return
	}

	m.showEncryptConfirm = true
	m.encryptConfirmYes = true
	if m.passphraseModalCtx == "encrypt" {
		m.status = "confirm: encrypt note?"
	} else {
		m.status = "confirm: remove encryption?"
	}
}

func (m *Model) armOpenEncrypted(path string) tea.Cmd {
	if m.sessionPassphrase == "" {
		m.pendingEncryptPath = path
		m.passphraseModalCtx = "unlock_edit"
		m.showPassphraseModal = true
		m.passphraseInput.SetValue("")
		m.passphraseInput.Focus()
		m.status = "enter passphrase to open"
		return nil
	}
	return saveNoteVersionAndOpenEncryptedCmd(m.rootDir, path, m.sessionPassphrase)
}
