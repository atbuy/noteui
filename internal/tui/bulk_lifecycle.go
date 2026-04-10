package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"atbuy/noteui/internal/notes"
)

const archiveTemporaryRelPath = "archive/tmp"

type moveBrowserMode int

const (
	moveBrowserModeMove moveBrowserMode = iota
	moveBrowserModePromoteTemporary
)

type selectedNoteRef struct {
	title   string
	relPath string
	path    string
	isTemp  bool
	pinKey  string
}

type noteRelocationItem struct {
	srcRoot    string
	oldRelPath string
	dstRoot    string
	newRelPath string
	oldPinPath string
	newPinPath string
}

type notesRelocatedMsg struct {
	items  []noteRelocationItem
	status string
	err    error
}

type notesDeletedMsg struct {
	paths   []string
	results []notes.TrashResult
	err     error
}

type notesTaggedMsg struct {
	paths []string
	tags  []string
	err   error
}

func tempMarkKey(relPath string) string {
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	return "t:" + strings.Trim(relPath, "/")
}

func (m Model) isMarkedTempNote(relPath string) bool {
	return m.markedTreeItems[tempMarkKey(relPath)]
}

func (m Model) markedTreeCategoryCount() int {
	count := 0
	for key := range m.markedTreeItems {
		if strings.HasPrefix(key, "c:") {
			count++
		}
	}
	return count
}

func (m Model) markedTreeNoteCount() int {
	count := 0
	for key := range m.markedTreeItems {
		if strings.HasPrefix(key, "n:") {
			count++
		}
	}
	return count
}

func (m Model) markedTempCount() int {
	count := 0
	for key := range m.markedTreeItems {
		if strings.HasPrefix(key, "t:") {
			count++
		}
	}
	return count
}

func (m Model) currentMarkedCount() int {
	switch m.listMode {
	case listModeTemporary:
		return m.markedTempCount()
	case listModeNotes:
		return m.markedTreeCategoryCount() + m.markedTreeNoteCount()
	default:
		return 0
	}
}

func (m Model) hasMarksInCurrentMode() bool {
	return m.currentMarkedCount() > 0
}

func (m *Model) clearAllMarks() {
	if len(m.markedTreeItems) == 0 {
		m.status = "no marks to clear"
		return
	}
	m.clearMarkedTreeItems()
	m.status = "marks cleared"
}

func (m Model) selectedMainNotesForAction() ([]selectedNoteRef, error) {
	if m.listMode != listModeNotes {
		return nil, fmt.Errorf("main-note action only works in notes view")
	}
	if m.markedTreeCategoryCount()+m.markedTreeNoteCount() > 0 {
		if m.markedTreeCategoryCount() > 0 {
			return nil, fmt.Errorf("marked categories only work with move")
		}
		out := make([]selectedNoteRef, 0, m.markedTreeNoteCount())
		for _, n := range m.notes {
			if !m.markedTreeItems["n:"+n.RelPath] {
				continue
			}
			out = append(out, selectedNoteRef{title: n.Title(), relPath: n.RelPath, path: n.Path, pinKey: n.RelPath})
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("no marked notes")
		}
		sort.Slice(out, func(i, j int) bool { return out[i].relPath < out[j].relPath })
		return out, nil
	}
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		return nil, fmt.Errorf("no note selected")
	}
	return []selectedNoteRef{{title: item.Note.Title(), relPath: item.Note.RelPath, path: item.Note.Path, pinKey: item.Note.RelPath}}, nil
}

func (m Model) selectedTempNotesForAction() ([]selectedNoteRef, error) {
	if m.listMode != listModeTemporary {
		return nil, fmt.Errorf("temporary-note action only works in temporary view")
	}
	if m.markedTempCount() > 0 {
		out := make([]selectedNoteRef, 0, m.markedTempCount())
		for _, n := range m.tempNotes {
			if !m.markedTreeItems[tempMarkKey(n.RelPath)] {
				continue
			}
			out = append(out, selectedNoteRef{title: n.Title(), relPath: n.RelPath, path: n.Path, isTemp: true, pinKey: tempPinnedKey(n.RelPath)})
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("no marked temporary notes")
		}
		sort.Slice(out, func(i, j int) bool { return out[i].relPath < out[j].relPath })
		return out, nil
	}
	n := m.currentTempNote()
	if n == nil {
		return nil, fmt.Errorf("no temporary note selected")
	}
	return []selectedNoteRef{{title: n.Title(), relPath: n.RelPath, path: n.Path, isTemp: true, pinKey: tempPinnedKey(n.RelPath)}}, nil
}

func (m Model) selectedTaggableNotePaths() ([]string, error) {
	switch m.listMode {
	case listModeNotes:
		refs, err := m.selectedMainNotesForAction()
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(refs))
		for _, ref := range refs {
			out = append(out, ref.path)
		}
		return out, nil
	case listModeTemporary:
		refs, err := m.selectedTempNotesForAction()
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(refs))
		for _, ref := range refs {
			out = append(out, ref.path)
		}
		return out, nil
	default:
		path := m.currentNotePath()
		if path == "" {
			return nil, fmt.Errorf("no note selected")
		}
		return []string{path}, nil
	}
}

func (m *Model) buildPromoteTemporaryBatch(destRelPath string) ([]noteRelocationItem, error) {
	refs, err := m.selectedTempNotesForAction()
	if err != nil {
		return nil, err
	}
	destRelPath = normalizeCategoryRelPath(destRelPath)
	items := make([]noteRelocationItem, 0, len(refs))
	tempRoot := notes.TempRoot(m.rootDir)
	for _, ref := range refs {
		newRelPath := ref.relPath
		if destRelPath != "" {
			newRelPath = filepath.ToSlash(filepath.Join(destRelPath, ref.relPath))
		}
		items = append(items, noteRelocationItem{
			srcRoot:    tempRoot,
			oldRelPath: ref.relPath,
			dstRoot:    m.rootDir,
			newRelPath: newRelPath,
			oldPinPath: ref.pinKey,
			newPinPath: newRelPath,
		})
	}
	return items, nil
}

func countStatus(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return fmt.Sprintf(plural, n)
}

func batchRelocateNotesCmd(items []noteRelocationItem, status string) tea.Cmd {
	return func() tea.Msg {
		for _, item := range items {
			if err := notes.MoveNoteBetweenRoots(item.srcRoot, item.oldRelPath, item.dstRoot, item.newRelPath); err != nil {
				return notesRelocatedMsg{items: items, status: status, err: err}
			}
		}
		return notesRelocatedMsg{items: items, status: status}
	}
}

func deleteNotesCmd(paths []string) tea.Cmd {
	copyPaths := append([]string(nil), paths...)
	return func() tea.Msg {
		var results []notes.TrashResult
		for _, path := range copyPaths {
			result, err := notes.DeleteNote(path)
			if err != nil {
				return notesDeletedMsg{paths: copyPaths, results: results, err: err}
			}
			results = append(results, result)
		}
		return notesDeletedMsg{paths: copyPaths, results: results}
	}
}

func addNoteTagsBatchCmd(paths []string, tags []string) tea.Cmd {
	copyPaths := append([]string(nil), paths...)
	copyTags := append([]string(nil), tags...)
	return func() tea.Msg {
		for _, path := range copyPaths {
			if err := notes.AddTagsToNote(path, copyTags); err != nil {
				return notesTaggedMsg{paths: copyPaths, tags: copyTags, err: err}
			}
		}
		return notesTaggedMsg{paths: copyPaths, tags: copyTags}
	}
}

func (m *Model) archiveTemporarySelection() tea.Cmd {
	refs, err := m.selectedTempNotesForAction()
	if err != nil {
		m.status = err.Error()
		return nil
	}
	items := make([]noteRelocationItem, 0, len(refs))
	tempRoot := notes.TempRoot(m.rootDir)
	for _, ref := range refs {
		newRelPath := filepath.ToSlash(filepath.Join(archiveTemporaryRelPath, ref.relPath))
		items = append(items, noteRelocationItem{
			srcRoot:    tempRoot,
			oldRelPath: ref.relPath,
			dstRoot:    m.rootDir,
			newRelPath: newRelPath,
			oldPinPath: ref.pinKey,
			newPinPath: newRelPath,
		})
	}
	m.status = "archiving..."
	return batchRelocateNotesCmd(items, countStatus(len(items), "archived 1 temporary note", "archived %d temporary notes"))
}

func (m *Model) moveSelectionToTemporary() tea.Cmd {
	refs, err := m.selectedMainNotesForAction()
	if err != nil {
		m.status = err.Error()
		return nil
	}
	items := make([]noteRelocationItem, 0, len(refs))
	tempRoot := notes.TempRoot(m.rootDir)
	for _, ref := range refs {
		items = append(items, noteRelocationItem{
			srcRoot:    m.rootDir,
			oldRelPath: ref.relPath,
			dstRoot:    tempRoot,
			newRelPath: ref.relPath,
			oldPinPath: ref.pinKey,
			newPinPath: tempPinnedKey(ref.relPath),
		})
	}
	m.status = "moving to temporary..."
	return batchRelocateNotesCmd(items, countStatus(len(items), "moved 1 note to temporary", "moved %d notes to temporary"))
}

func (m *Model) openPromoteTemporaryBrowser() {
	if _, err := m.selectedTempNotesForAction(); err != nil {
		m.status = err.Error()
		return
	}
	m.showMoveBrowser = true
	m.moveBrowserMode = moveBrowserModePromoteTemporary
	m.moveDestCursor = 0
	m.moveBrowserError = ""
	m.setMoveDestinationCursor("")
	m.pendingG = false
	m.pendingBracketDir = ""
	m.status = "choose destination category"
}

func (m Model) moveBrowserTitle() string {
	if m.moveBrowserMode == moveBrowserModePromoteTemporary {
		return "Promote Temporary Notes"
	}
	return "Move"
}

func (m Model) moveBrowserHint() string {
	if m.moveBrowserMode == moveBrowserModePromoteTemporary {
		return "Choose an existing destination category in the main notes tree for the current temporary note or marked batch."
	}
	return "Choose an existing destination category for the current item or marked batch."
}

func (m Model) moveBrowserFooter() string {
	if m.moveBrowserMode == moveBrowserModePromoteTemporary {
		return "Enter to promote • h/l collapse/expand • Esc to cancel"
	}
	return "Enter to move • h/l collapse/expand • Esc to cancel"
}

func (m Model) moveBrowserCancelStatus() string {
	if m.moveBrowserMode == moveBrowserModePromoteTemporary {
		return "promote cancelled"
	}
	return "move cancelled"
}

func (m Model) canPromoteTemporary() bool {
	if m.listMode != listModeTemporary {
		return false
	}
	_, err := m.selectedTempNotesForAction()
	return err == nil
}

func (m Model) canArchiveTemporary() bool {
	if m.listMode != listModeTemporary {
		return false
	}
	_, err := m.selectedTempNotesForAction()
	return err == nil
}

func (m Model) canMoveSelectionToTemporary() bool {
	if m.listMode != listModeNotes {
		return false
	}
	_, err := m.selectedMainNotesForAction()
	return err == nil
}

func (m Model) canClearMarks() bool {
	return len(m.markedTreeItems) > 0
}

func (m *Model) removePinnedForAbsolutePath(path string) {
	if rel, err := filepath.Rel(m.rootDir, path); err == nil {
		rel = filepath.ToSlash(rel)
		if rel != "." && !strings.HasPrefix(rel, "../") {
			delete(m.pinnedNotes, rel)
		}
	}
	tempRoot := notes.TempRoot(m.rootDir)
	if rel, err := filepath.Rel(tempRoot, path); err == nil {
		rel = filepath.ToSlash(rel)
		if rel != "." && !strings.HasPrefix(rel, "../") {
			delete(m.pinnedNotes, tempPinnedKey(rel))
		}
	}
}

func (m *Model) removePinnedForPaths(paths []string) {
	for _, path := range paths {
		m.removePinnedForAbsolutePath(path)
	}
}
