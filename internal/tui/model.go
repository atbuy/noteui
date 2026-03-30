package tui

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/editor"
	"atbuy/noteui/internal/notes"
	"atbuy/noteui/internal/state"
)

type (
	treeItemKind     int
	deleteTargetKind int
	moveTargetKind   int
	renameTargetKind int
)

const (
	treePaneRatio   = 0.32
	minTreeWidth    = 30
	minPreviewWidth = 40
	panelGapWidth   = 2
)

const (
	treeCategory treeItemKind = iota
	treeNote
)

const (
	deleteNone deleteTargetKind = iota
	deleteTargetCategory
	deleteTargetNote
)

const (
	moveTargetNone moveTargetKind = iota
	moveTargetCategory
	moveTargetNote
)

const (
	renameTargetNone renameTargetKind = iota
	renameTargetNote
	renameTargetCategory
)

type movePending struct {
	kind       moveTargetKind
	oldRelPath string
	name       string
}

type noteMovedMsg struct {
	oldRelPath string
	newRelPath string
	err        error
}

type noteRenamedMsg struct {
	oldPath string
	newPath string
	err     error
}

type categoryMovedMsg struct {
	oldRelPath string
	newRelPath string
	err        error
}

type renamePending struct {
	kind     renameTargetKind
	path     string
	relPath  string
	oldTitle string
	oldName  string
}

type categoryRenamedMsg struct {
	oldRelPath string
	newRelPath string
	err        error
}

type deletePending struct {
	kind    deleteTargetKind
	relPath string
	name    string
}

type noteDeletedMsg struct {
	path string
	err  error
}

type categoryDeletedMsg struct {
	relPath string
	err     error
}

type treeItem struct {
	Kind      treeItemKind
	Name      string
	RelPath   string
	Depth     int
	Expanded  bool
	Note      *notes.Note
	Category  *notes.Category
	MatchHint string
}

func (t treeItem) key() string {
	switch t.Kind {
	case treeCategory:
		return "c:" + t.RelPath
	case treeNote:
		return "n:" + t.RelPath
	default:
		return t.RelPath
	}
}

type Model struct {
	rootDir string
	version string

	notes      []notes.Note
	categories []notes.Category
	expanded   map[string]bool

	treeItems    []treeItem
	treeCursor   int
	selected     *notes.Note
	width        int
	height       int
	previewWidth int
	status       string

	cfg            config.Config
	preview        viewport.Model
	previewPath    string
	previewContent string

	state       state.State
	pinnedNotes map[string]bool
	pinnedCats  map[string]bool

	showHelp bool

	showCreateCategory bool
	categoryInput      textinput.Model

	showMove    bool
	moveInput   textinput.Model
	movePending *movePending

	showRename    bool
	renameInput   textinput.Model
	renamePending *renamePending

	searchInput textinput.Model
	searchMode  bool

	deletePending  *deletePending
	preserveCursor int

	startupError string
}
type dataLoadedMsg struct {
	notes      []notes.Note
	categories []notes.Category
	err        error
}

type noteCreatedMsg struct {
	path string
	err  error
}

type categoryCreatedMsg struct {
	relPath string
	err     error
}

func New(root, startupError string, cfg config.Config, version string) Model {
	categoryInput := textinput.New()
	categoryInput.Placeholder = "work/project-a"
	categoryInput.Prompt = "Category: "
	categoryInput.CharLimit = 200
	categoryInput.Width = 40

	searchInput := textinput.New()
	searchInput.Placeholder = "Search notes..."
	searchInput.Prompt = "/ "
	searchInput.CharLimit = 200
	searchInput.Width = 32

	moveInput := textinput.New()
	moveInput.Placeholder = "work/project-a/note.md"
	moveInput.Prompt = "Move to: "
	moveInput.CharLimit = 300
	moveInput.Width = 48

	renameInput := textinput.New()
	renameInput.Placeholder = "New title"
	renameInput.Prompt = "Title: "
	renameInput.CharLimit = 300
	renameInput.Width = 48

	vp := viewport.New(0, 0)

	st, _ := state.Load()

	pinnedNotes := make(map[string]bool, len(st.PinnedNotes))
	for _, p := range st.PinnedNotes {
		pinnedNotes[p] = true
	}

	pinnedCats := make(map[string]bool, len(st.PinnedCategories))
	for _, p := range st.PinnedCategories {
		pinnedCats[p] = true
	}

	expanded := map[string]bool{
		"": true,
	}
	for _, p := range st.CollapsedCategories {
		expanded[p] = false
	}

	return Model{
		rootDir:        root,
		version:        version,
		status:         "loading notes...",
		expanded:       expanded,
		categoryInput:  categoryInput,
		searchInput:    searchInput,
		moveInput:      moveInput,
		renameInput:    renameInput,
		preserveCursor: -1,
		startupError:   startupError,
		cfg:            cfg,
		preview:        vp,
		state:          st,
		pinnedNotes:    pinnedNotes,
		pinnedCats:     pinnedCats,
	}
}

func (m Model) Init() tea.Cmd {
	return refreshAllCmd(m.rootDir)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		leftWidth, rightWidth := m.panelWidths()

		m.previewWidth = rightWidth
		m.searchInput.Width = max(16, leftWidth-8)
		m.categoryInput.Width = max(24, min(50, m.width-16))
		m.moveInput.Width = max(24, min(60, m.width-16))
		m.renameInput.Width = max(24, min(60, m.width-16))

		previewInnerWidth := max(20, rightWidth-8)
		previewInnerHeight := max(6, msg.Height-14)
		m.preview.Width = previewInnerWidth
		m.preview.Height = previewInnerHeight
		m.refreshPreview()
		return m, nil

	case noteMovedMsg:
		if msg.err != nil {
			m.status = "move failed: " + msg.err.Error()
			return m, nil
		}
		m.showMove = false
		m.movePending = nil
		m.moveInput.Blur()
		m.moveInput.SetValue("")
		m.preserveCursor = m.treeCursor
		m.status = "moved note: " + msg.newRelPath
		return m, refreshAllCmd(m.rootDir)

	case noteRenamedMsg:
		if msg.err != nil {
			m.status = "rename failed: " + msg.err.Error()
			return m, nil
		}
		m.showRename = false
		m.renamePending = nil
		m.renameInput.Blur()
		m.renameInput.SetValue("")
		m.preserveCursor = m.treeCursor
		m.status = "renamed note: " + filepath.Base(msg.newPath)
		return m, refreshAllCmd(m.rootDir)

	case categoryRenamedMsg:
		if msg.err != nil {
			m.status = "rename failed: " + msg.err.Error()
			return m, nil
		}
		m.showRename = false
		m.renamePending = nil
		m.renameInput.Blur()
		m.renameInput.SetValue("")
		m.preserveCursor = m.treeCursor
		m.rewriteCategoryStateSubtree(msg.oldRelPath, msg.newRelPath)
		_ = m.saveTreeState()
		m.status = "renamed category: " + msg.newRelPath
		return m, refreshAllCmd(m.rootDir)

	case categoryMovedMsg:
		if msg.err != nil {
			m.status = "move failed: " + msg.err.Error()
			return m, nil
		}
		m.showMove = false
		m.movePending = nil
		m.moveInput.Blur()
		m.moveInput.SetValue("")
		m.preserveCursor = m.treeCursor
		m.rewriteCategoryStateSubtree(msg.oldRelPath, msg.newRelPath)
		_ = m.saveTreeState()
		m.status = "moved category: " + msg.newRelPath
		return m, refreshAllCmd(m.rootDir)

	case noteDeletedMsg:
		if msg.err != nil {
			m.status = "delete failed: " + msg.err.Error()
			return m, nil
		}
		m.deletePending = nil
		m.preserveCursor = m.treeCursor
		m.status = "deleted note: " + msg.path
		return m, refreshAllCmd(m.rootDir)

	case categoryDeletedMsg:
		if msg.err != nil {
			m.status = "delete failed: " + msg.err.Error()
			return m, nil
		}
		m.deletePending = nil
		m.preserveCursor = m.treeCursor
		m.removeCategoryStateSubtree(msg.relPath)
		_ = m.saveTreeState()
		m.status = "deleted category: " + msg.relPath
		return m, refreshAllCmd(m.rootDir)

	case dataLoadedMsg:
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
			return m, nil
		}

		m.notes = msg.notes
		m.categories = msg.categories

		m.pruneCategoryStateToExisting()
		_ = m.saveTreeState()

		for _, c := range m.categories {
			if c.RelPath == "" {
				continue
			}
			if _, ok := m.expanded[c.RelPath]; !ok {
				m.expanded[c.RelPath] = true
			}
		}

		m.rebuildTree()

		if len(msg.notes) > 0 {
			m.status = fmt.Sprintf("loaded %d notes", len(msg.notes))
		} else {
			m.status = "no notes found"
		}
		return m, nil

	case noteCreatedMsg:
		if msg.err != nil {
			m.status = "create failed: " + msg.err.Error()
			return m, nil
		}
		m.status = "created: " + msg.path
		return m, tea.Batch(refreshAllCmd(m.rootDir), editor.Open(msg.path))

	case categoryCreatedMsg:
		if msg.err != nil {
			m.status = "category create failed: " + msg.err.Error()
			return m, nil
		}
		m.showCreateCategory = false
		m.categoryInput.Blur()
		m.categoryInput.SetValue("")
		m.status = "created category: " + msg.relPath
		return m, refreshAllCmd(m.rootDir)

	case editor.FinishedMsg:
		if msg.Err != nil {
			m.status = "editor error: " + msg.Err.Error()
			return m, nil
		}

		newPath, renamed, err := notes.RenameFromTitle(msg.Path)
		if err != nil {
			m.status = "rename failed: " + err.Error()
			return m, refreshAllCmd(m.rootDir)
		}

		if renamed {
			m.status = "renamed: " + filepath.Base(newPath)
		} else {
			m.status = "editor closed"
		}

		return m, refreshAllCmd(m.rootDir)

	case tea.KeyMsg:
		if m.showMove {
			switch msg.String() {
			case "pgdown", "ctrl+f":
				m.preview.PageDown()
				return m, nil
			case "pgup", "ctrl+b":
				m.preview.PageUp()
				return m, nil
			case "esc":
				m.showMove = false
				m.movePending = nil
				m.moveInput.Blur()
				m.moveInput.SetValue("")
				m.status = "move cancelled"
				return m, nil
			case "enter":
				value := strings.TrimSpace(m.moveInput.Value())
				if value == "" {
					m.showMove = false
					m.movePending = nil
					m.moveInput.Blur()
					m.status = "move cancelled"
					return m, nil
				}
				return m, m.confirmMove(value)
			}

			var cmd tea.Cmd
			m.moveInput, cmd = m.moveInput.Update(msg)
			return m, cmd
		}

		if m.showRename {
			switch msg.String() {
			case "esc":
				m.showRename = false
				m.renamePending = nil
				m.renameInput.Blur()
				m.renameInput.SetValue("")
				m.status = "rename cancelled"
				return m, nil
			case "enter":
				value := strings.TrimSpace(m.renameInput.Value())
				if value == "" {
					m.showRename = false
					m.renamePending = nil
					m.renameInput.Blur()
					m.status = "rename cancelled"
					return m, nil
				}

				switch m.renamePending.kind {
				case renameTargetNote:
					return m, renameNoteCmd(m.renamePending.path, value)
				case renameTargetCategory:
					return m, renameCategoryCmd(m.rootDir, m.renamePending.relPath, value)
				default:
					return m, nil
				}
			}

			var cmd tea.Cmd
			m.renameInput, cmd = m.renameInput.Update(msg)
			return m, cmd
		}

		if m.deletePending != nil {
			switch msg.String() {
			case "esc":
				m.deletePending = nil
				m.status = "delete cancelled"
				return m, nil
			case "d":
				return m, m.confirmDeleteCurrent()
			default:
				m.deletePending = nil
			}
		}

		if m.showHelp {
			switch msg.String() {
			case "esc", "q", "?":
				m.showHelp = false
				m.status = "help closed"
				return m, nil
			default:
				return m, nil
			}
		}

		if m.showCreateCategory {
			switch msg.String() {
			case "esc":
				m.showCreateCategory = false
				m.categoryInput.Blur()
				m.categoryInput.SetValue("")
				m.status = "category creation cancelled"
				return m, nil
			case "enter":
				value := strings.TrimSpace(m.categoryInput.Value())
				if value == "" {
					m.showCreateCategory = false
					m.categoryInput.Blur()
					m.status = "category creation cancelled"
					return m, nil
				}
				return m, createCategoryCmd(m.rootDir, value)
			}

			var cmd tea.Cmd
			m.categoryInput, cmd = m.categoryInput.Update(msg)
			return m, cmd
		}

		if key.Matches(msg, keys.Quit) {
			return m, tea.Quit
		}

		if key.Matches(msg, keys.ShowHelp) {
			m.showHelp = true
			m.status = "help"
			return m, nil
		}

		if key.Matches(msg, keys.Move) {
			m.armMoveCurrent()
			return m, nil
		}

		if key.Matches(msg, keys.Rename) {
			m.armRenameCurrent()
			return m, nil
		}

		if key.Matches(msg, keys.Delete) {
			m.armDeleteCurrent()
			return m, nil
		}

		if key.Matches(msg, keys.Pin) {
			if err := m.togglePinCurrent(); err != nil {
				m.status = "pin failed: " + err.Error()
			}
			return m, nil
		}

		// Search mode
		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.searchInput.Blur()
				m.status = "search applied"
				return m, nil
			case "enter":
				m.searchMode = false
				m.searchInput.Blur()
				m.status = "search applied"
				return m, nil
			}

			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.rebuildTree()
			return m, cmd
		}

		// Start search
		if key.Matches(msg, keys.Search) || key.Matches(msg, keys.Focus) {
			m.searchMode = true
			m.searchInput.Focus()
			m.status = "search"
			return m, nil
		}

		// Clear applied search on second esc
		if msg.String() == "esc" && strings.TrimSpace(m.searchInput.Value()) != "" {
			m.searchInput.SetValue("")
			m.searchInput.Blur()
			m.searchMode = false
			m.rebuildTree()
			m.status = "search cleared"
			return m, nil
		}

		if key.Matches(msg, keys.CreateCategory) {
			m.showCreateCategory = true
			m.categoryInput.SetValue(m.currentCategoryPrefix())
			m.categoryInput.Focus()
			m.categoryInput.CursorEnd()
			m.status = "new category"
			return m, nil
		}

		if key.Matches(msg, keys.Refresh) {
			m.status = "refreshing..."
			return m, refreshAllCmd(m.rootDir)
		}

		if key.Matches(msg, keys.NewNote) {
			return m, createNoteCmd(m.rootDir, m.currentTargetDir())
		}

		switch msg.String() {
		case "up", "k":
			m.moveTreeCursor(-1)
			return m, nil
		case "down", "j":
			m.moveTreeCursor(1)
			return m, nil
		case "right", "l":
			m.expandCurrentCategory()
			return m, nil
		case "left", "h":
			m.collapseCurrentCategory()
			return m, nil
		}

		if key.Matches(msg, keys.Open) || key.Matches(msg, keys.ToggleCategory) {
			return m, m.activateCurrentItem()
		}
	}

	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	_ = cmd

	return m, nil
}

func (m Model) View() string {
	usableWidth := max(40, m.width-6)
	leftWidth, rightWidth := m.panelWidths()
	gap := strings.Repeat(" ", panelGapWidth)

	leftBody := lipgloss.JoinVertical(
		lipgloss.Left,
		panelTitleStyle.Render("Tree"),
		m.renderSearchBar(),
		"",
		m.renderTreeView(),
	)

	rightBody := lipgloss.JoinVertical(
		lipgloss.Left,
		panelTitleStyle.Render("Preview"),
		m.previewView(),
	)

	left := panelStyle(leftWidth, m.height, true).Render(leftBody)
	right := panelStyle(rightWidth, m.height, false).Render(rightBody)

	titleText := " noteui "
	if strings.TrimSpace(m.version) != "" {
		titleText = fmt.Sprintf(" noteui %s ", m.version)
	}

	title := titleBarStyle.
		Width(usableWidth).
		Render(titleText)

	footer := footerStyle.
		Width(usableWidth).
		Render(m.renderStatus())

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right)

	base := appStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			body,
			footer,
		),
	)

	if m.showCreateCategory {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderCreateCategoryModal(),
		)
	}

	if m.showMove {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderMoveModal(),
		)
	}

	if m.showRename {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderRenameModal(),
		)
	}

	if m.showHelp {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderHelpModal(),
		)
	}

	return base
}

func (m Model) renderSearchBar() string {
	if m.searchMode || strings.TrimSpace(m.searchInput.Value()) != "" {
		return m.searchInput.View()
	}
	return mutedStyle.Render("Press / to search")
}

func (m Model) renderTreeView() string {
	if len(m.treeItems) == 0 {
		return emptyStyle.Render("(empty)")
	}

	lines := make([]string, 0, len(m.treeItems))
	for i, item := range m.treeItems {
		lines = append(lines, m.renderTreeLine(item, i == m.treeCursor))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderTreeLine(item treeItem, selected bool) string {
	rowWidth := m.treeInnerWidth()

	var icon string
	switch item.Kind {
	case treeCategory:
		if m.categoryHasChildren(item.RelPath) {
			if item.Expanded {
				icon = iconCategoryExpanded
			} else {
				icon = iconCategoryCollapsed
			}
		} else {
			icon = iconCategoryLeaf
		}
	case treeNote:
		icon = iconNote
	}

	pinned := false
	switch item.Kind {
	case treeCategory:
		pinned = m.isPinnedCategory(item.RelPath)
	case treeNote:
		if item.Note != nil {
			pinned = m.isPinnedNote(item.Note.RelPath)
		}
	}

	pinMark := "  "
	if pinned {
		pinMark = "★ "
	}

	indent := strings.Repeat("  ", item.Depth)
	leftPrefix := indent + pinMark + icon + " "
	label := item.Name

	plainLine := trimOrPad(leftPrefix+label, rowWidth-2)

	if selected {
		return lipgloss.NewStyle().
			Width(rowWidth).
			Padding(0, 1).
			Foreground(selectedFgColor).
			Background(selectedBgColor).
			Bold(boldSelected).
			Render(plainLine)
	}

	// Non-selected rows can still have different foreground colors.
	rowStyle := treeNoteStyle
	if item.Kind == treeCategory {
		rowStyle = treeCategoryStyle
	}
	if pinned {
		rowStyle = rowStyle.Copy().Foreground(accentColor)
	}

	return rowStyle.
		Width(rowWidth).
		Padding(0, 1).
		Render(plainLine)
}

func (m Model) previewView() string {
	return m.preview.View()
}

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
		if query != "" {
			return true
		}
		return m.expanded[rel]
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
	return strings.Contains(strings.ToLower(n.Name), q) ||
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
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		m.selected = nil
		m.refreshPreview()
		return
	}
	m.selected = item.Note
	m.refreshPreview()
}

func (m Model) currentTreeItem() *treeItem {
	if len(m.treeItems) == 0 || m.treeCursor < 0 || m.treeCursor >= len(m.treeItems) {
		return nil
	}
	item := m.treeItems[m.treeCursor]
	return &item
}

func (m *Model) activateCurrentItem() tea.Cmd {
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

	if item.Kind == treeCategory {
		return item.RelPath
	}

	if item.Note != nil {
		dir := filepath.Dir(item.Note.RelPath)
		if dir == "." {
			return ""
		}
		return dir
	}

	return ""
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

func refreshAllCmd(root string) tea.Cmd {
	return func() tea.Msg {
		n, err := notes.Discover(root)
		if err != nil {
			return dataLoadedMsg{err: err}
		}

		cats, err := notes.DiscoverCategories(root)
		if err != nil {
			return dataLoadedMsg{err: err}
		}

		return dataLoadedMsg{
			notes:      n,
			categories: cats,
		}
	}
}

func createNoteCmd(root, relDir string) tea.Cmd {
	return func() tea.Msg {
		path, err := notes.CreateNote(root, relDir)
		return noteCreatedMsg{path: path, err: err}
	}
}

func createCategoryCmd(root, relPath string) tea.Cmd {
	return func() tea.Msg {
		err := notes.CreateCategory(root, relPath)
		return categoryCreatedMsg{relPath: relPath, err: err}
	}
}

func (m Model) renderStatus() string {
	if m.deletePending != nil {
		return statusErrStyle.Render("TRASH PENDING • press d to confirm • esc to cancel")
	}

	if m.startupError != "" {
		return statusErrStyle.Render("CONFIG ERROR • " + m.startupError)
	}

	parts := []string{
		m.renderModeSegment(),
		m.renderSelectionSegment(),
	}

	if filter := m.renderFilterSegment(); filter != "" {
		parts = append(parts, filter)
	}

	if preview := m.renderPreviewSegment(); preview != "" {
		parts = append(parts, preview)
	}

	if m.status != "" {
		parts = append(parts, m.status)
	}

	line := strings.Join(parts, "  •  ")

	switch {
	case strings.HasPrefix(m.status, "error:"),
		strings.HasPrefix(m.status, "editor error:"),
		strings.HasPrefix(m.status, "create failed:"),
		strings.HasPrefix(m.status, "category create failed:"),
		strings.HasPrefix(m.status, "delete failed:"),
		strings.HasPrefix(m.status, "rename failed:"),
		strings.HasPrefix(m.status, "move failed:"),
		strings.HasPrefix(m.status, "pin failed:"):
		return statusErrStyle.Render(line)
	default:
		return statusOKStyle.Render(line)
	}
}

func (m Model) renderModeSegment() string {
	switch {
	case m.showHelp:
		return "HELP"
	case m.showCreateCategory:
		return "NEW CATEGORY"
	case m.showMove:
		return "MOVE"
	case m.showRename:
		return "RENAME"
	case m.searchMode:
		return "SEARCH"
	default:
		return "TREE"
	}
}

func (m Model) renderSelectionSegment() string {
	item := m.currentTreeItem()
	if item == nil {
		return "nothing selected"
	}

	switch item.Kind {
	case treeCategory:
		name := item.Name
		if item.RelPath == "" {
			name = "~/notes"
		}
		if m.isPinnedCategory(item.RelPath) {
			return "category: ★ " + name
		}
		return "category: " + name

	case treeNote:
		if item.Note == nil {
			return "note"
		}
		title := item.Note.Title()
		if m.isPinnedNote(item.Note.RelPath) {
			return "note: ★ " + title
		}
		return "note: " + title
	}

	return "selection"
}

func (m Model) renderFilterSegment() string {
	filter := strings.TrimSpace(m.searchInput.Value())
	if filter == "" {
		return ""
	}
	return "filter: " + filter
}

func (m Model) renderPreviewSegment() string {
	if m.preview.TotalLineCount() == 0 {
		return ""
	}

	atTop := m.preview.AtTop()
	atBottom := m.preview.AtBottom()

	switch {
	case atTop && atBottom:
		return "preview: 100%"
	case atTop:
		return "preview: top"
	case atBottom:
		return "preview: bottom"
	}

	total := m.preview.TotalLineCount()
	offset := m.preview.YOffset
	height := m.preview.Height

	if total <= 0 {
		return ""
	}

	maxOffset := total - height
	if maxOffset <= 0 {
		return "preview: 100%"
	}

	pct := int(float64(offset) / float64(maxOffset) * 100.0)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	return fmt.Sprintf("preview: %d%%", pct)
}

func (m Model) renderHelpModal() string {
	modalWidth, innerWidth := m.modalDimensions(50, 76)

	title := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(modalTitleStyle.Render("Help"))

	lines := []string{
		m.renderHelpLine("j/k", "Move up and down", innerWidth),
		m.renderHelpLine("enter/o", "Open note or toggle category", innerWidth),
		m.renderHelpLine("h/l", "Collapse/Expand category", innerWidth),
		m.renderHelpLine("/", "Search", innerWidth),
		m.renderHelpLine("esc", "Leave search, then clear on second press", innerWidth),
		m.renderHelpLine("n", "New note in current category", innerWidth),
		m.renderHelpLine("C", "Create category", innerWidth),
		m.renderHelpLine("dd", "Trash note/category", innerWidth),
		m.renderHelpLine("r", "Refresh", innerWidth),
		m.renderHelpLine("q", "Quit", innerWidth),
		m.renderHelpLine("esc/q/?", "Close help", innerWidth),
		m.renderHelpLine("m", "Move note/category", innerWidth),
		m.renderHelpLine("R", "Rename note/category", innerWidth),
		m.renderHelpLine("p", "Pin note/category", innerWidth),
	}

	body := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	footer := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(modalFooterStyle.Render("Press esc, q, or ? to close"))

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				title,
				lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(""),
				body,
				lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(""),
				footer,
			),
		)

	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderMoveModal() string {
	title := "Move"
	hint := "Enter the new relative path under ~/notes."
	label := "Path"

	if m.movePending != nil {
		switch m.movePending.kind {
		case moveTargetNote:
			title = "Move note"
			hint = "Move the note to a new relative path under ~/notes."
			label = "Note"
		case moveTargetCategory:
			title = "Move category"
			hint = "Move the category to a new relative path under ~/notes."
			label = "Category"
		}
	}

	return m.renderStandardModal(
		title,
		hint,
		label,
		m.moveInput,
		"Enter to move • Esc to cancel",
	)
}

func (m Model) renderRenameModal() string {
	title := "Rename note"
	hint := "Change the note title. The file name will update automatically."
	label := "Title"

	if m.renamePending != nil && m.renamePending.kind == renameTargetCategory {
		title = "Rename category"
		hint = "Change the category path under ~/notes."
		label = "Category"
	}

	return m.renderStandardModal(
		title,
		hint,
		label,
		m.renameInput,
		"Enter to rename • Esc to cancel",
	)
}

func (m Model) renderCreateCategoryModal() string {
	return m.renderStandardModal(
		"Create category",
		"Use / to create nested categories, e.g. work/project-a",
		"Category",
		m.categoryInput,
		"Enter to create • Esc to cancel",
	)
}

func (m Model) modalDimensions(minWidth, maxWidth int) (int, int) {
	modalWidth := min(maxWidth, max(minWidth, m.width-10))
	innerWidth := max(20, modalWidth-(modalPaddingX*2)-2)
	return modalWidth, innerWidth
}

func (m Model) renderModalTitle(text string, innerWidth int) string {
	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(modalTitleStyle.Render(text))
}

func (m Model) renderModalHint(text string, innerWidth int) string {
	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(modalMutedStyle.Render(text))
}

func (m Model) renderModalFooter(text string, innerWidth int) string {
	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(modalFooterStyle.Render(text))
}

func (m Model) renderModalBlank(innerWidth int) string {
	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render("")
}

func (m Model) renderModalInputRow(label string, input textinput.Model, innerWidth int) string {
	local := input
	local.Prompt = ""
	local.Width = max(12, min(36, innerWidth-20))

	local.TextStyle = lipgloss.NewStyle().
		Foreground(modalTextColor).
		Background(modalBgColor)

	local.PlaceholderStyle = lipgloss.NewStyle().
		Foreground(modalMutedColor).
		Background(modalBgColor)

	local.Cursor.Style = lipgloss.NewStyle().
		Foreground(modalTextColor).
		Background(modalTextColor)

	labelText := lipgloss.NewStyle().
		Foreground(modalAccentColor).
		Background(modalBgColor).
		Bold(true).
		Render(label + ":")

	// Make the label a 3-line block so its text aligns with the input text line,
	// not with the top border of the input box.
	promptBlock := lipgloss.NewStyle().
		Background(modalBgColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				"",
				labelText,
				"",
			),
		)

	inputField := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(modalAccentColor).
		BorderBackground(modalBgColor).
		Background(modalBgColor).
		Padding(0, 1).
		Render(local.View())

	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				promptBlock,
				"",
				inputField,
			),
		)
}

func (m Model) renderStandardModal(
	title, hint, label string,
	input textinput.Model,
	footer string,
) string {
	modalWidth, innerWidth := m.modalDimensions(48, 76)

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.renderModalTitle(title, innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalHint(hint, innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalInputRow(label, input, innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalFooter(footer, innerWidth),
			),
		)

	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderHelpLine(k, desc string, width int) string {
	keyWidth := 14
	descWidth := max(10, width-keyWidth)

	keyPart := lipgloss.NewStyle().
		Width(keyWidth).
		Background(modalBgColor).
		Render(
			modalKeyStyle.
				Width(keyWidth).
				Render(k),
		)

	descPart := lipgloss.NewStyle().
		Width(descWidth).
		Background(modalBgColor).
		Render(
			modalTextStyle.
				Width(descWidth).
				Render(desc),
		)

	return lipgloss.NewStyle().
		Width(width).
		Background(modalBgColor).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, keyPart, descPart))
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

func (m *Model) armMoveCurrent() {
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
		return moveNoteCmd(m.rootDir, m.movePending.oldRelPath, newRelPath)
	case moveTargetCategory:
		return moveCategoryCmd(m.rootDir, m.movePending.oldRelPath, newRelPath)
	default:
		return nil
	}
}

func moveNoteCmd(root, oldRelPath, newRelPath string) tea.Cmd {
	return func() tea.Msg {
		err := notes.MoveNote(root, oldRelPath, newRelPath)
		return noteMovedMsg{
			oldRelPath: oldRelPath,
			newRelPath: newRelPath,
			err:        err,
		}
	}
}

func moveCategoryCmd(root, oldRelPath, newRelPath string) tea.Cmd {
	return func() tea.Msg {
		err := notes.MoveCategory(root, oldRelPath, newRelPath)
		return categoryMovedMsg{
			oldRelPath: oldRelPath,
			newRelPath: newRelPath,
			err:        err,
		}
	}
}

func (m *Model) armDeleteCurrent() {
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

func deleteNoteCmd(path string) tea.Cmd {
	return func() tea.Msg {
		err := notes.DeleteNote(path)
		return noteDeletedMsg{path: path, err: err}
	}
}

func deleteCategoryCmd(root, relPath string) tea.Cmd {
	return func() tea.Msg {
		err := notes.DeleteCategory(root, relPath)
		return categoryDeletedMsg{relPath: relPath, err: err}
	}
}

func (m *Model) armRenameCurrent() {
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

func renameNoteCmd(path, newTitle string) tea.Cmd {
	return func() tea.Msg {
		newPath, _, err := notes.RenameNoteTitle(path, newTitle)
		return noteRenamedMsg{
			oldPath: path,
			newPath: newPath,
			err:     err,
		}
	}
}

func renameCategoryCmd(root, oldRelPath, newRelPath string) tea.Cmd {
	return func() tea.Msg {
		err := notes.MoveCategory(root, oldRelPath, newRelPath)
		return categoryRenamedMsg{
			oldRelPath: oldRelPath,
			newRelPath: newRelPath,
			err:        err,
		}
	}
}

func (m *Model) refreshPreview() {
	item := m.currentTreeItem()
	if item == nil {
		m.previewPath = ""
		m.previewContent = "Nothing selected"
		m.preview.SetContent(m.previewContent)
		m.preview.GotoTop()
		return
	}

	if item.Kind == treeCategory {
		pathText := item.Name
		if item.RelPath == "" {
			pathText = "~/notes"
		} else {
			pathText = filepath.Join("~/notes", item.RelPath)
		}

		count := m.countNotesUnder(item.RelPath)
		children := m.countChildCategories(item.RelPath)

		content := strings.Join([]string{
			"# " + pathText,
			"",
			fmt.Sprintf("- Subcategories: %d", children),
			fmt.Sprintf("- Notes: %d", count),
			"",
			"Category selected. Press enter or space to expand/collapse.",
		}, "\n")

		rendered := m.renderPreviewMarkdown(pathText, content)
		m.previewPath = "category:" + item.RelPath
		m.previewContent = rendered
		m.preview.SetContent(rendered)
		m.preview.GotoTop()
		return
	}

	if item.Note == nil {
		m.previewPath = ""
		m.previewContent = "No note selected"
		m.preview.SetContent(m.previewContent)
		m.preview.GotoTop()
		return
	}

	if m.previewPath == item.Note.Path && m.previewContent != "" {
		return
	}

	raw, err := notes.ReadAll(item.Note.Path)
	if err != nil {
		m.previewPath = item.Note.Path
		m.previewContent = "Failed to read note: " + err.Error()
		m.preview.SetContent(m.previewContent)
		m.preview.GotoTop()
		return
	}

	rendered := m.renderPreviewMarkdown(item.Note.RelPath, raw)
	m.previewPath = item.Note.Path
	m.previewContent = rendered
	m.preview.SetContent(rendered)
	m.preview.GotoTop()
}

func (m Model) renderPreviewMarkdown(relPath, raw string) string {
	if !m.cfg.Preview.RenderMarkdown || m.previewMarkdownDisabledFor(relPath) {
		return raw
	}

	width := m.preview.Width
	if width <= 0 {
		width = max(20, m.previewWidth-8)
	}

	opts := markdownRenderOptions{
		Width:           width,
		SyntaxHighlight: m.cfg.Preview.SyntaxHighlight,
		CodeStyle:       m.cfg.Preview.CodeStyle,
	}

	return renderMarkdownTerminal(raw, opts)
}

func (m Model) previewMarkdownDisabledFor(relPath string) bool {
	relPath = filepath.ToSlash(relPath)
	for _, pattern := range m.cfg.Preview.DisablePaths {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		pattern = filepath.ToSlash(pattern)

		if ok, err := path.Match(pattern, relPath); err == nil && ok {
			return true
		}
		if relPath == pattern {
			return true
		}
		if strings.HasPrefix(relPath, strings.TrimSuffix(pattern, "/")+"/") {
			return true
		}
	}
	return false
}

func (m *Model) togglePinCurrent() error {
	item := m.currentTreeItem()
	if item == nil {
		return nil
	}

	switch item.Kind {
	case treeCategory:
		if item.RelPath == "" {
			m.status = "cannot pin root category"
			return nil
		}
		if m.pinnedCats[item.RelPath] {
			delete(m.pinnedCats, item.RelPath)
			m.status = "unpinned category: " + item.Name
		} else {
			m.pinnedCats[item.RelPath] = true
			m.status = "pinned category: " + item.Name
		}

	case treeNote:
		if item.Note == nil {
			return nil
		}
		if m.pinnedNotes[item.Note.RelPath] {
			delete(m.pinnedNotes, item.Note.RelPath)
			m.status = "unpinned note: " + item.Note.Title()
		} else {
			m.pinnedNotes[item.Note.RelPath] = true
			m.status = "pinned note: " + item.Note.Title()
		}
	}

	if err := m.saveTreeState(); err != nil {
		return err
	}

	m.rebuildTree()
	return nil
}

func (m *Model) syncStateFromPins() {
	m.state.PinnedNotes = m.state.PinnedNotes[:0]
	for p := range m.pinnedNotes {
		m.state.PinnedNotes = append(m.state.PinnedNotes, p)
	}
	sort.Strings(m.state.PinnedNotes)

	m.state.PinnedCategories = m.state.PinnedCategories[:0]
	for p := range m.pinnedCats {
		m.state.PinnedCategories = append(m.state.PinnedCategories, p)
	}
	sort.Strings(m.state.PinnedCategories)
}

func (m *Model) syncStateFromExpanded() {
	m.state.CollapsedCategories = m.state.CollapsedCategories[:0]

	for relPath, expanded := range m.expanded {
		if relPath == "" {
			continue
		}
		if !expanded {
			m.state.CollapsedCategories = append(m.state.CollapsedCategories, relPath)
		}
	}

	sort.Strings(m.state.CollapsedCategories)
}

func hasCategoryPrefix(path, prefix string) bool {
	if prefix == "" {
		return false
	}
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func rewriteCategoryPrefix(path, oldPrefix, newPrefix string) string {
	if path == oldPrefix {
		return newPrefix
	}
	if strings.HasPrefix(path, oldPrefix+"/") {
		return newPrefix + strings.TrimPrefix(path, oldPrefix)
	}
	return path
}

func (m *Model) removeCategoryStateSubtree(relPath string) {
	if relPath == "" {
		return
	}

	for k := range m.expanded {
		if hasCategoryPrefix(k, relPath) {
			delete(m.expanded, k)
		}
	}

	for k := range m.pinnedCats {
		if hasCategoryPrefix(k, relPath) {
			delete(m.pinnedCats, k)
		}
	}
}

func (m *Model) rewriteCategoryStateSubtree(oldRelPath, newRelPath string) {
	if oldRelPath == "" || newRelPath == "" || oldRelPath == newRelPath {
		return
	}

	newExpanded := make(map[string]bool, len(m.expanded))
	for k, v := range m.expanded {
		if hasCategoryPrefix(k, oldRelPath) {
			k = rewriteCategoryPrefix(k, oldRelPath, newRelPath)
		}
		newExpanded[k] = v
	}
	m.expanded = newExpanded

	newPinnedCats := make(map[string]bool, len(m.pinnedCats))
	for k, v := range m.pinnedCats {
		if hasCategoryPrefix(k, oldRelPath) {
			k = rewriteCategoryPrefix(k, oldRelPath, newRelPath)
		}
		newPinnedCats[k] = v
	}
	m.pinnedCats = newPinnedCats

	newPinnedNotes := make(map[string]bool, len(m.pinnedNotes))
	oldPrefix := oldRelPath + "/"
	newPrefix := newRelPath + "/"

	for k, v := range m.pinnedNotes {
		if after, ok := strings.CutPrefix(k, oldPrefix); ok {
			k = newPrefix + after
		}
		newPinnedNotes[k] = v
	}
	m.pinnedNotes = newPinnedNotes
}

func (m *Model) saveTreeState() error {
	m.syncStateFromPins()
	m.syncStateFromExpanded()
	return state.Save(m.state)
}

func (m Model) isPinnedCategory(relPath string) bool {
	return m.pinnedCats[relPath]
}

func (m Model) isPinnedNote(relPath string) bool {
	return m.pinnedNotes[relPath]
}

func (m Model) treeInnerWidth() int {
	leftWidth, _ := m.panelWidths()
	return max(16, leftWidth-6)
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

func (m *Model) pruneCategoryStateToExisting() {
	existing := make(map[string]bool, len(m.categories))
	for _, c := range m.categories {
		if c.RelPath != "" {
			existing[c.RelPath] = true
		}
	}

	for k := range m.expanded {
		if k == "" {
			continue
		}
		if !existing[k] {
			delete(m.expanded, k)
		}
	}

	for k := range m.pinnedCats {
		if !existing[k] {
			delete(m.pinnedCats, k)
		}
	}
}
