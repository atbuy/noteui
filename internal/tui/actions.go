package tui

import (
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

	m.showMove = true
	m.moveInput.Focus()

	switch item.Kind {
	case treeCategory:
		if item.RelPath == "" {
			m.showMove = false
			m.status = "cannot move root category"
			return
		}
		m.movePending = &movePending{
			kind:       moveTargetCategory,
			oldRelPath: item.RelPath,
			name:       item.Name,
		}
		m.moveInput.SetValue(item.RelPath)
		m.moveInput.CursorEnd()
		m.status = "move category"

	case treeNote:
		if item.Note == nil {
			m.showMove = false
			return
		}
		m.movePending = &movePending{
			kind:       moveTargetNote,
			oldRelPath: item.Note.RelPath,
			name:       item.Note.Title(),
		}
		m.moveInput.SetValue(item.Note.RelPath)
		m.moveInput.CursorEnd()
		m.status = "move note"
	}
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
		n := m.currentTempNote()
		if n == nil {
			return
		}
		m.deletePending = &deletePending{
			kind:    deleteTargetNote,
			relPath: n.Path,
			name:    n.Name,
		}
		m.status = "press d again to trash temporary note: " + n.Name
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
	}
}

func (m Model) confirmDeleteCurrent() tea.Cmd {
	if m.deletePending == nil {
		return nil
	}

	switch m.deletePending.kind {
	case deleteTargetNote:
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
	}
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
	} else {
		out = make([]notes.Note, 0, len(m.tempNotes))
		for _, n := range m.tempNotes {
			if m.noteMatches(n, query) {
				out = append(out, n)
			}
		}
	}

	if m.sortByModTime {
		sort.SliceStable(out, func(i, j int) bool {
			return out[i].ModTime.After(out[j].ModTime)
		})
	}

	return out
}

func (m Model) currentCategoryPrefix() string {
	item := m.currentTreeItem()
	if item == nil {
		return ""
	}

	if item.Kind == treeCategory {
		if item.RelPath == "" {
			return ""
		}
		return item.RelPath + string(filepath.Separator)
	}

	if item.Note != nil {
		dir := filepath.Dir(item.Note.RelPath)
		if dir == "." || dir == "" {
			return ""
		}
		return dir + string(filepath.Separator)
	}

	return ""
}
