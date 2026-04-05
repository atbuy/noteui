package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type moveBatchItem struct {
	kind       moveTargetKind
	oldRelPath string
	newRelPath string
	name       string
}

type moveSelectionSummary struct {
	notes      int
	categories int
}

type batchMovedMsg struct {
	items []moveBatchItem
	err   error
}

func (m *Model) openMoveBrowser() {
	if m.listMode != listModeNotes {
		return
	}

	selection := m.currentMoveSelection()
	if len(selection) == 0 {
		m.status = "nothing selected"
		return
	}
	if len(selection) == 1 && selection[0].Kind == treeCategory && selection[0].RelPath == "" {
		m.status = "cannot move root category"
		return
	}

	m.showMoveBrowser = true
	m.moveBrowserMode = moveBrowserModeMove
	m.moveDestCursor = 0
	m.moveBrowserError = ""
	m.setMoveDestinationCursor(m.preferredMoveDestinationPath(selection))
	m.pendingG = false
	m.pendingBracketDir = ""
	m.status = "choose destination category"
}

func (m *Model) closeMoveBrowser(status string) {
	m.showMoveBrowser = false
	m.moveBrowserMode = moveBrowserModeMove
	m.moveDestCursor = 0
	m.moveBrowserError = ""
	m.pendingG = false
	m.pendingBracketDir = ""
	if status != "" {
		m.status = status
	}
}

func (m *Model) toggleMarkCurrent() {
	if m.markedTreeItems == nil {
		m.markedTreeItems = make(map[string]bool)
	}

	switch m.listMode {
	case listModeTemporary:
		n := m.currentTempNote()
		if n == nil {
			m.status = "nothing selected"
			return
		}
		key := tempMarkKey(n.RelPath)
		if m.markedTreeItems[key] {
			delete(m.markedTreeItems, key)
			m.status = "unmarked: " + n.Title()
			return
		}
		m.markedTreeItems[key] = true
		m.status = "marked: " + n.Title()
		return
	case listModeNotes:
		item := m.currentTreeItem()
		if item == nil {
			m.status = "nothing selected"
			return
		}
		if item.Kind == treeCategory && item.RelPath == "" {
			m.status = "cannot mark root category"
			return
		}
		key := item.key()
		if m.markedTreeItems[key] {
			delete(m.markedTreeItems, key)
			m.status = "unmarked: " + item.Name
			return
		}
		m.markedTreeItems[key] = true
		m.status = "marked: " + item.Name
		return
	default:
		m.status = "multi-select is only available in notes and temporary views"
		return
	}
}

func (m *Model) clearMarkedTreeItems() {
	m.markedTreeItems = make(map[string]bool)
}

func (m *Model) pruneMarkedTreeItems() {
	if len(m.markedTreeItems) == 0 {
		return
	}

	existing := make(map[string]bool, len(m.notes)+len(m.tempNotes)+len(m.categories))
	for _, n := range m.notes {
		existing["n:"+n.RelPath] = true
	}
	for _, n := range m.tempNotes {
		existing[tempMarkKey(n.RelPath)] = true
	}
	for _, c := range m.categories {
		if c.RelPath == "" {
			continue
		}
		existing["c:"+c.RelPath] = true
	}

	for key := range m.markedTreeItems {
		if !existing[key] {
			delete(m.markedTreeItems, key)
		}
	}
}

func (m Model) markedTreeCount() int {
	return m.currentMarkedCount()
}

func (m Model) isMarkedTreeItem(item treeItem) bool {
	if len(m.markedTreeItems) == 0 {
		return false
	}
	return m.markedTreeItems[item.key()]
}

func (m Model) currentMoveSelection() []treeItem {
	if len(m.markedTreeItems) == 0 {
		item := m.currentTreeItem()
		if item == nil {
			return nil
		}
		return []treeItem{*item}
	}

	out := make([]treeItem, 0, len(m.markedTreeItems))
	for _, c := range m.categories {
		if c.RelPath == "" {
			continue
		}
		key := "c:" + c.RelPath
		if !m.markedTreeItems[key] {
			continue
		}
		catCopy := c
		out = append(out, treeItem{
			Kind:     treeCategory,
			Name:     catCopy.Name,
			RelPath:  catCopy.RelPath,
			Depth:    catCopy.Depth + 1,
			Expanded: m.expanded[catCopy.RelPath],
			Category: &catCopy,
		})
	}
	for _, n := range m.notes {
		key := "n:" + n.RelPath
		if !m.markedTreeItems[key] {
			continue
		}
		noteCopy := n
		depth := 1
		if dir := filepath.Dir(noteCopy.RelPath); dir != "." && dir != "" {
			depth += strings.Count(dir, string(filepath.Separator)) + 1
		}
		out = append(out, treeItem{
			Kind:    treeNote,
			Name:    noteCopy.Title(),
			RelPath: noteCopy.RelPath,
			Depth:   depth,
			Note:    &noteCopy,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].RelPath < out[j].RelPath
	})
	return out
}

func (m Model) moveSelectionSummary() moveSelectionSummary {
	summary := moveSelectionSummary{}
	for _, item := range m.currentMoveSelection() {
		switch item.Kind {
		case treeCategory:
			summary.categories++
		case treeNote:
			summary.notes++
		}
	}
	return summary
}

func (m Model) moveDestinationItems() []treeItem {
	out := []treeItem{{
		Kind:     treeCategory,
		Name:     "/",
		RelPath:  "",
		Depth:    0,
		Expanded: true,
	}}
	m.buildMoveDestinationItems("", 1, &out)
	return out
}

func (m Model) buildMoveDestinationItems(parent string, depth int, out *[]treeItem) {
	for _, cat := range m.directChildCategories(parent) {
		item := treeItem{
			Kind:     treeCategory,
			Name:     cat.Name,
			RelPath:  cat.RelPath,
			Depth:    depth,
			Expanded: m.expanded[cat.RelPath],
			Category: &cat,
		}
		*out = append(*out, item)
		if item.Expanded {
			m.buildMoveDestinationItems(cat.RelPath, depth+1, out)
		}
	}
}

func (m *Model) setMoveDestinationCursor(relPath string) {
	items := m.moveDestinationItems()
	for i, item := range items {
		if item.RelPath == relPath {
			m.moveDestCursor = i
			return
		}
	}
	m.moveDestCursor = 0
}

func (m *Model) clampMoveDestinationCursor() {
	items := m.moveDestinationItems()
	if len(items) == 0 {
		m.moveDestCursor = 0
		return
	}
	if m.moveDestCursor < 0 {
		m.moveDestCursor = 0
		return
	}
	if m.moveDestCursor >= len(items) {
		m.moveDestCursor = len(items) - 1
	}
}

func (m Model) currentMoveDestination() *treeItem {
	items := m.moveDestinationItems()
	if len(items) == 0 {
		return nil
	}
	cursor := m.moveDestCursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(items) {
		cursor = len(items) - 1
	}
	item := items[cursor]
	return &item
}

func (m Model) currentMoveDestinationPath() string {
	item := m.currentMoveDestination()
	if item == nil {
		return ""
	}
	return item.RelPath
}

func (m *Model) moveDestinationCursor(delta int) {
	items := m.moveDestinationItems()
	if len(items) == 0 {
		return
	}
	next := max(m.moveDestCursor+delta, 0)
	if next >= len(items) {
		next = len(items) - 1
	}
	m.moveDestCursor = next
}

func (m *Model) jumpMoveDestinationTop() {
	m.moveDestCursor = 0
}

func (m *Model) jumpMoveDestinationBottom() {
	items := m.moveDestinationItems()
	if len(items) == 0 {
		m.moveDestCursor = 0
		return
	}
	m.moveDestCursor = len(items) - 1
}

func (m *Model) expandMoveDestination() {
	item := m.currentMoveDestination()
	if item == nil || item.Kind != treeCategory {
		return
	}
	if item.RelPath != "" && m.categoryHasChildren(item.RelPath) {
		m.expanded[item.RelPath] = true
		m.setMoveDestinationCursor(item.RelPath)
		_ = m.saveTreeState()
	}
}

func (m *Model) collapseMoveDestination() {
	item := m.currentMoveDestination()
	if item == nil || item.Kind != treeCategory {
		return
	}

	if item.RelPath != "" && m.categoryHasChildren(item.RelPath) && m.expanded[item.RelPath] {
		m.expanded[item.RelPath] = false
		m.setMoveDestinationCursor(item.RelPath)
		_ = m.saveTreeState()
		return
	}

	parent := filepath.Dir(item.RelPath)
	if parent == "." {
		parent = ""
	}
	m.setMoveDestinationCursor(parent)
}

func (m Model) preferredMoveDestinationPath(selection []treeItem) string {
	if len(selection) != 1 {
		return ""
	}
	item := selection[0]
	if item.Kind == treeCategory {
		parent := filepath.Dir(item.RelPath)
		if parent == "." {
			return ""
		}
		return parent
	}
	if item.Note != nil {
		parent := filepath.Dir(item.Note.RelPath)
		if parent == "." {
			return ""
		}
		return parent
	}
	return ""
}

func (m *Model) confirmMoveBrowser() tea.Cmd {
	if m.moveBrowserMode == moveBrowserModePromoteTemporary {
		items, err := m.buildPromoteTemporaryBatch(m.currentMoveDestinationPath())
		if err != nil {
			m.moveBrowserError = err.Error()
			m.status = "promote failed: " + err.Error()
			return nil
		}
		m.moveBrowserError = ""
		if len(items) == 0 {
			m.status = "nothing to promote"
			return nil
		}
		return batchRelocateNotesCmd(items, countStatus(len(items), "promoted 1 temporary note", "promoted %d temporary notes"))
	}

	items, err := m.buildMoveBatch(m.currentMoveDestinationPath())
	if err != nil {
		m.moveBrowserError = err.Error()
		m.status = "move failed: " + err.Error()
		return nil
	}
	m.moveBrowserError = ""
	if len(items) == 0 {
		m.status = "nothing to move"
		return nil
	}
	return batchMoveCmd(m.rootDir, items)
}

func (m Model) buildMoveBatch(destRelPath string) ([]moveBatchItem, error) {
	selection := m.currentMoveSelection()
	if len(selection) == 0 {
		return nil, fmt.Errorf("nothing selected")
	}

	destRelPath = strings.TrimSpace(filepath.Clean(destRelPath))
	if destRelPath == "." {
		destRelPath = ""
	}
	if strings.HasPrefix(destRelPath, "..") {
		return nil, fmt.Errorf("destination must stay inside notes root")
	}

	selectedCats := make(map[string]bool)
	selectedNotes := make(map[string]bool)
	for _, item := range selection {
		switch item.Kind {
		case treeCategory:
			if item.RelPath == "" {
				return nil, fmt.Errorf("cannot move root category")
			}
			selectedCats[item.RelPath] = true
		case treeNote:
			if item.Note == nil {
				return nil, fmt.Errorf("invalid note selection")
			}
			selectedNotes[item.Note.RelPath] = true
		}
	}

	catPaths := make([]string, 0, len(selectedCats))
	for relPath := range selectedCats {
		catPaths = append(catPaths, relPath)
	}
	sort.Strings(catPaths)
	for i := 0; i < len(catPaths); i++ {
		for j := i + 1; j < len(catPaths); j++ {
			if hasCategoryPrefix(catPaths[j], catPaths[i]) {
				return nil, fmt.Errorf("cannot move a category and its descendant together: %s", catPaths[j])
			}
		}
	}
	for noteRelPath := range selectedNotes {
		noteDir := filepath.Dir(noteRelPath)
		if noteDir == "." {
			noteDir = ""
		}
		for catRelPath := range selectedCats {
			if noteDir == catRelPath || hasCategoryPrefix(noteDir, catRelPath) {
				return nil, fmt.Errorf("cannot move a category and a note inside it together: %s", noteRelPath)
			}
		}
	}

	batch := make([]moveBatchItem, 0, len(selection))
	seenTargets := make(map[string]string)
	selectedOldAbs := make(map[string]string)
	for _, item := range selection {
		selectedOldAbs[filepath.Join(m.rootDir, item.RelPath)] = item.Name
	}

	for _, item := range selection {
		batchItem := moveBatchItem{
			kind:       moveTargetNone,
			oldRelPath: item.RelPath,
			name:       item.Name,
		}
		baseName := filepath.Base(item.RelPath)
		targetRelPath := filepath.Join(destRelPath, baseName)
		if targetRelPath == "." {
			targetRelPath = baseName
		}
		if item.Kind == treeNote && item.Note != nil {
			batchItem.kind = moveTargetNote
			batchItem.oldRelPath = item.Note.RelPath
			batchItem.name = item.Note.Title()
			targetRelPath = filepath.Join(destRelPath, filepath.Base(item.Note.RelPath))
			if targetRelPath == "." {
				targetRelPath = filepath.Base(item.Note.RelPath)
			}
		} else if item.Kind == treeCategory {
			batchItem.kind = moveTargetCategory
		}

		targetRelPath = filepath.Clean(targetRelPath)
		if targetRelPath == "." {
			targetRelPath = ""
		}
		batchItem.newRelPath = targetRelPath

		if batchItem.newRelPath == batchItem.oldRelPath {
			return nil, fmt.Errorf("%s is already in that category", batchItem.name)
		}
		if batchItem.kind == moveTargetCategory && hasCategoryPrefix(batchItem.newRelPath, batchItem.oldRelPath) {
			return nil, fmt.Errorf("cannot move a category inside itself: %s", batchItem.name)
		}

		targetAbs := filepath.Join(m.rootDir, batchItem.newRelPath)
		if owner, exists := seenTargets[targetAbs]; exists {
			return nil, fmt.Errorf("destination conflict between %s and %s", owner, batchItem.name)
		}
		seenTargets[targetAbs] = batchItem.name

		if owner, exists := selectedOldAbs[targetAbs]; exists && targetAbs != filepath.Join(m.rootDir, batchItem.oldRelPath) {
			return nil, fmt.Errorf("destination overlaps selected item: %s", owner)
		}

		if _, err := os.Stat(targetAbs); err == nil {
			return nil, fmt.Errorf("target already exists: %s", batchItem.newRelPath)
		} else if !os.IsNotExist(err) {
			return nil, err
		}

		batch = append(batch, batchItem)
	}

	sort.SliceStable(batch, func(i, j int) bool {
		if batch[i].kind != batch[j].kind {
			return batch[i].kind < batch[j].kind
		}
		return batch[i].oldRelPath < batch[j].oldRelPath
	})
	return batch, nil
}

func (m *Model) rewritePinnedNotePath(oldRelPath, newRelPath string) {
	if oldRelPath == "" || newRelPath == "" || oldRelPath == newRelPath {
		return
	}
	if !m.pinnedNotes[oldRelPath] {
		return
	}
	delete(m.pinnedNotes, oldRelPath)
	m.pinnedNotes[newRelPath] = true
}
