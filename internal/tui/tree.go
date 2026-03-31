package tui

import (
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/editor"
	"atbuy/noteui/internal/notes"
)

func (m *Model) rebuildTree() {
	var selectedKey string
	if current := m.currentTreeItem(); current != nil {
		selectedKey = current.key()
	}

	var out []treeItem
	m.buildTree("", 0, &out)
	m.treeItems = out

	if len(m.treeItems) == 0 {
		m.treeCursor = 0
		m.selected = nil
		m.preserveCursor = -1
		return
	}

	restore := -1

	if selectedKey != "" {
		for i, item := range m.treeItems {
			if item.key() == selectedKey {
				restore = i
				break
			}
		}
	}

	if restore == -1 && m.preserveCursor >= 0 {
		restore = m.preserveCursor
		if restore >= len(m.treeItems) {
			restore = len(m.treeItems) - 1
		}
	}

	if restore == -1 {
		restore = 0
	}

	m.treeCursor = restore
	m.preserveCursor = -1
	m.syncSelectedNote()
}

func (m *Model) buildTree(parent string, depth int, out *[]treeItem) {
	query := strings.TrimSpace(strings.ToLower(m.searchInput.Value()))

	effectiveExpanded := func(rel string) bool {
		if rel == "" {
			return true
		}
		if query != "" {
			return true
		}
		return m.expanded[rel]
	}

	// Add a synthetic root item at the top of the tree.
	if parent == "" && depth == 0 {
		*out = append(*out, treeItem{
			Kind:     treeCategory,
			Name:     "/",
			RelPath:  "",
			Depth:    0,
			Expanded: true,
			Category: nil,
		})

		// Root should show its direct children one level below it.
		depth = 1
	}

	for _, cat := range m.directChildCategories(parent) {
		include := query == "" || m.categoryMatches(cat, query) ||
			m.categorySubtreeMatches(cat.RelPath, query)
		if !include {
			continue
		}

		item := treeItem{
			Kind:     treeCategory,
			Name:     cat.Name,
			RelPath:  cat.RelPath,
			Depth:    depth,
			Expanded: effectiveExpanded(cat.RelPath),
			Category: &cat,
		}
		*out = append(*out, item)

		if item.Expanded {
			m.buildTree(cat.RelPath, depth+1, out)
		}
	}

	for _, n := range m.directChildNotes(parent) {
		if query != "" && !m.noteMatches(n, query) {
			continue
		}
		noteCopy := n
		*out = append(*out, treeItem{
			Kind:    treeNote,
			Name:    n.Title(),
			RelPath: n.RelPath,
			Depth:   depth,
			Note:    &noteCopy,
		})
	}
}

func (m Model) directChildCategories(parent string) []notes.Category {
	out := make([]notes.Category, 0)
	for _, c := range m.categories {
		if c.RelPath == "" {
			continue
		}
		dir := filepath.Dir(c.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == parent {
			out = append(out, c)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		pi := m.isPinnedCategory(out[i].RelPath)
		pj := m.isPinnedCategory(out[j].RelPath)
		if pi != pj {
			return pi
		}
		return out[i].RelPath < out[j].RelPath
	})

	return out
}

func (m Model) directChildNotes(parent string) []notes.Note {
	out := make([]notes.Note, 0)
	for _, n := range m.notes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == parent {
			out = append(out, n)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		pi := m.isPinnedNote(out[i].RelPath)
		pj := m.isPinnedNote(out[j].RelPath)
		if pi != pj {
			return pi
		}
		return out[i].RelPath < out[j].RelPath
	})

	return out
}

func (m Model) noteMatches(n notes.Note, query string) bool {
	q := strings.ToLower(query)
	return strings.Contains(strings.ToLower(n.Title()), q) ||
		strings.Contains(strings.ToLower(n.Name), q) ||
		strings.Contains(strings.ToLower(n.RelPath), q) ||
		strings.Contains(strings.ToLower(n.Preview), q)
}

func (m Model) categoryMatches(c notes.Category, query string) bool {
	return strings.Contains(strings.ToLower(c.Name), query) ||
		strings.Contains(strings.ToLower(c.RelPath), query)
}

func (m Model) categorySubtreeMatches(relPath, query string) bool {
	prefix := relPath + string(filepath.Separator)

	for _, c := range m.categories {
		if c.RelPath != relPath && strings.HasPrefix(c.RelPath, prefix) &&
			m.categoryMatches(c, query) {
			return true
		}
	}
	for _, n := range m.notes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == relPath || strings.HasPrefix(dir, prefix) {
			if m.noteMatches(n, query) {
				return true
			}
		}
	}
	return false
}

func (m *Model) moveTreeCursor(delta int) {
	if len(m.treeItems) == 0 {
		return
	}
	next := max(m.treeCursor+delta, 0)
	if next >= len(m.treeItems) {
		next = len(m.treeItems) - 1
	}
	m.treeCursor = next
	m.syncSelectedNote()
}

func (m *Model) syncSelectedNote() {
	switch m.listMode {
	case listModeTemporary:
		n := m.currentTempNote()
		if n == nil {
			m.selected = nil
			m.refreshPreview()
			return
		}
		m.selected = n
		m.refreshPreview()
		return

	case listModePins:
		item := m.currentPinItem()
		if item == nil {
			m.selected = nil
			m.refreshPreview()
			return
		}

		switch item.Kind {
		case pinItemNote:
			for _, n := range m.notes {
				if n.RelPath == item.RelPath {
					noteCopy := n
					m.selected = &noteCopy
					m.refreshPreview()
					return
				}
			}
		case pinItemTemporaryNote:
			for _, n := range m.tempNotes {
				if n.RelPath == item.RelPath {
					noteCopy := n
					m.selected = &noteCopy
					m.refreshPreview()
					return
				}
			}
		}

		m.selected = nil
		m.refreshPreview()
		return

	default:
		item := m.currentTreeItem()
		if item == nil || item.Kind != treeNote || item.Note == nil {
			m.selected = nil
			m.refreshPreview()
			return
		}
		m.selected = item.Note
		m.refreshPreview()
	}
}

func (m Model) currentTreeItem() *treeItem {
	if len(m.treeItems) == 0 || m.treeCursor < 0 || m.treeCursor >= len(m.treeItems) {
		return nil
	}
	item := m.treeItems[m.treeCursor]
	return &item
}

func (m *Model) activateCurrentItem() tea.Cmd {
	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil {
			return nil
		}

		switch item.Kind {
		case pinItemCategory:
			m.switchToNotesMode()
			m.selectTreeCategory(item.RelPath)
			m.status = "jumped to pinned category"
			return nil

		case pinItemNote:
			m.switchToNotesMode()
			m.selectTreeNote(item.RelPath)
			m.status = "jumped to pinned note"
			return nil

		case pinItemTemporaryNote:
			m.switchToTemporaryMode()
			m.selectTemporaryNote(item.RelPath)
			m.status = "jumped to pinned temporary note"
			return nil
		}
	}

	if m.listMode == listModeTemporary {
		n := m.currentTempNote()
		if n == nil {
			return nil
		}
		m.status = "opening in nvim: " + n.RelPath
		return editor.Open(n.Path)
	}

	item := m.currentTreeItem()
	if item == nil {
		return nil
	}

	if item.Kind == treeCategory {
		m.toggleCurrentCategory()
		return nil
	}

	if item.Note != nil {
		m.status = "opening in nvim: " + item.Note.RelPath
		return editor.Open(item.Note.Path)
	}

	return nil
}

func (m *Model) selectTreeCategory(relPath string) {
	m.rebuildTree()
	for i, item := range m.treeItems {
		if item.Kind == treeCategory && item.RelPath == relPath {
			m.treeCursor = i
			m.syncSelectedNote()
			return
		}
	}
}

func (m *Model) selectTreeNote(relPath string) {
	m.rebuildTree()
	for i, item := range m.treeItems {
		if item.Kind == treeNote && item.RelPath == relPath {
			m.treeCursor = i
			m.syncSelectedNote()
			return
		}
	}
}

func (m *Model) selectTemporaryNote(relPath string) {
	items := m.filteredTempNotes()
	for i, n := range items {
		if n.RelPath == relPath {
			m.tempCursor = i
			m.syncSelectedNote()
			return
		}
	}
}

func (m *Model) toggleCurrentCategory() {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeCategory {
		return
	}
	if !m.categoryHasChildren(item.RelPath) {
		m.status = "category: " + item.Name
		return
	}

	m.expanded[item.RelPath] = !m.expanded[item.RelPath]
	if m.expanded[item.RelPath] {
		m.status = "expanded: " + item.Name
	} else {
		m.status = "collapsed: " + item.Name
	}

	_ = m.saveTreeState()
	m.rebuildTree()
}

func (m *Model) expandCurrentCategory() {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeCategory {
		return
	}
	if m.categoryHasChildren(item.RelPath) {
		m.expanded[item.RelPath] = true
		m.status = "expanded: " + item.Name
		_ = m.saveTreeState()
		m.rebuildTree()
	}
}

func (m *Model) collapseCurrentCategory() {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeCategory {
		return
	}

	if m.categoryHasChildren(item.RelPath) && m.expanded[item.RelPath] {
		m.expanded[item.RelPath] = false
		m.status = "collapsed: " + item.Name
		_ = m.saveTreeState()
		m.rebuildTree()
		return
	}

	parent := filepath.Dir(item.RelPath)
	if parent == "." {
		parent = ""
	}
	for i, t := range m.treeItems {
		if t.Kind == treeCategory && t.RelPath == parent {
			m.treeCursor = i
			m.syncSelectedNote()
			return
		}
	}
}

func (m Model) categoryHasChildren(relPath string) bool {
	if relPath == "" {
		return len(m.directChildCategories("")) > 0 || len(m.directChildNotes("")) > 0
	}
	prefix := relPath + string(filepath.Separator)
	for _, c := range m.categories {
		if c.RelPath != relPath && strings.HasPrefix(c.RelPath, prefix) {
			return true
		}
	}
	for _, n := range m.notes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == relPath {
			return true
		}
	}
	return false
}

func (m Model) currentTargetDir() string {
	item := m.currentTreeItem()
	if item == nil {
		return ""
	}

	switch item.Kind {
	case treeCategory:
		return item.RelPath
	case treeNote:
		if item.Note == nil {
			return ""
		}
		dir := filepath.Dir(item.Note.RelPath)
		if dir == "." {
			return ""
		}
		return dir
	default:
		return ""
	}
}

func (m Model) countNotesUnder(relPath string) int {
	if relPath == "" {
		return len(m.notes)
	}
	prefix := relPath + string(filepath.Separator)
	count := 0
	for _, n := range m.notes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == relPath || strings.HasPrefix(dir, prefix) {
			count++
		}
	}
	return count
}

func (m Model) countChildCategories(relPath string) int {
	count := 0
	for _, c := range m.categories {
		if c.RelPath == "" {
			continue
		}
		dir := filepath.Dir(c.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == relPath {
			count++
		}
	}
	return count
}

func (m Model) treeInnerWidth() int {
	leftWidth, _ := m.panelWidths()
	return max(16, leftWidth-6)
}

func (m Model) panelWidths() (int, int) {
	usableWidth := max(40, m.width-6)

	leftWidth := int(float64(usableWidth) * treePaneRatio)
	leftWidth = max(minTreeWidth, leftWidth)

	rightWidth := usableWidth - leftWidth - panelGapWidth
	rightWidth = max(minPreviewWidth, rightWidth)

	// Rebalance if minimums pushed things too far.
	if leftWidth+rightWidth+panelGapWidth > usableWidth {
		leftWidth = max(minTreeWidth, usableWidth-rightWidth-panelGapWidth)
	}

	return leftWidth, rightWidth
}

func trimOrPad(s string, width int) string {
	w := lipgloss.Width(s)
	if w == width {
		return s
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}

	// Simple trim for now.
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	cur := 0
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if cur+rw > width {
			break
		}
		out = append(out, r)
		cur += rw
	}
	if cur < width {
		out = append(out, []rune(strings.Repeat(" ", width-cur))...)
	}
	return string(out)
}
