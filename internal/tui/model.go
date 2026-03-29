package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/editor"
	"atbuy/noteui/internal/notes"
)

type treeItemKind int

type deleteTargetKind int

const (
	treeCategory treeItemKind = iota
	treeNote
)

const (
	deleteNone deleteTargetKind = iota
	deleteTargetCategory
	deleteTargetNote
)

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

	showHelp bool

	showCreateCategory bool
	categoryInput      textinput.Model

	searchInput textinput.Model
	searchMode  bool

	deletePending  *deletePending
	preserveCursor int
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

func New(root string) Model {
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

	return Model{
		rootDir:        root,
		status:         "loading notes...",
		expanded:       map[string]bool{},
		categoryInput:  categoryInput,
		searchInput:    searchInput,
		preserveCursor: -1,
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

		usableWidth := max(40, msg.Width-6)
		leftWidth := max(28, usableWidth/3)
		gap := 2
		rightWidth := max(30, usableWidth-leftWidth-gap)

		m.previewWidth = rightWidth
		m.searchInput.Width = max(16, leftWidth-8)
		m.categoryInput.Width = max(24, min(50, m.width-16))
		return m, nil

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
		m.status = "deleted category: " + msg.relPath
		return m, refreshAllCmd(m.rootDir)

	case dataLoadedMsg:
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
			return m, nil
		}

		m.notes = msg.notes
		m.categories = msg.categories

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

		if key.Matches(msg, keys.Delete) {
			m.armDeleteCurrent()
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

	return m, nil
}

func (m Model) View() string {
	usableWidth := max(40, m.width-6)
	leftWidth := max(28, usableWidth/3)
	gap := "  "
	rightWidth := max(30, usableWidth-leftWidth-len(gap))

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

	title := titleBarStyle.
		Width(usableWidth).
		Render(" noteui ")

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
	var icon string
	switch item.Kind {
	case treeCategory:
		if m.categoryHasChildren(item.RelPath) {
			if item.Expanded {
				icon = "▾"
			} else {
				icon = "▸"
			}
		} else {
			icon = "•"
		}
	case treeNote:
		icon = "·"
	}

	indent := strings.Repeat("  ", item.Depth)
	label := indent + icon + " " + item.Name

	style := lipgloss.NewStyle()
	if selected {
		style = style.
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Bold(true)
	} else {
		switch item.Kind {
		case treeCategory:
			style = style.Foreground(accentSoftColor)
		case treeNote:
			style = style.Foreground(textColor)
		}
	}
	return style.Render(label)
}

func (m Model) previewView() string {
	item := m.currentTreeItem()
	if item == nil {
		return emptyStyle.Render("Nothing selected")
	}

	if item.Kind == treeCategory {
		path := item.Name
		if item.RelPath == "" {
			path = "~/notes"
		} else {
			path = filepath.Join("~/notes", item.RelPath)
		}

		count := m.countNotesUnder(item.RelPath)
		children := m.countChildCategories(item.RelPath)

		meta := lipgloss.JoinHorizontal(
			lipgloss.Left,
			chipStyle.Render(fmt.Sprintf("Subcategories: %d", children)),
			chipStyle.Render(fmt.Sprintf("Notes: %d", count)),
		)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			headerStyle.Render(path),
			meta,
			"",
			mutedStyle.Render("Category selected. Press enter or space to expand/collapse."),
		)
	}

	if item.Note == nil {
		return emptyStyle.Render("No note selected")
	}

	content := item.Note.Preview
	if strings.TrimSpace(content) == "" {
		content = "(empty file)"
	}

	metaRow := lipgloss.JoinHorizontal(
		lipgloss.Left,
		chipStyle.Render("Category: "+item.Note.Category),
		chipStyle.Render("Modified: "+item.Note.ModTime.Format("2006-01-02 15:04")),
	)

	contentStyle := lipgloss.NewStyle().
		Width(max(20, m.previewWidth-8))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render(item.Note.Title()),
		metaStyle.Render(item.Note.RelPath),
		metaRow,
		"",
		contentStyle.Render(content),
	)
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
	next := m.treeCursor + delta
	if next < 0 {
		next = 0
	}
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
		return
	}
	m.selected = item.Note
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
		return "inbox"
	}

	if item.Kind == treeCategory {
		if item.RelPath == "" {
			return "inbox"
		}
		return item.RelPath
	}

	if item.Note != nil {
		dir := filepath.Dir(item.Note.RelPath)
		if dir == "." || dir == "" {
			return "inbox"
		}
		return dir
	}

	return "inbox"
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
		return statusErrStyle.Render("Delete pending: press d to confirm • esc to cancel")
	}

	search := strings.TrimSpace(m.searchInput.Value())
	if search != "" && !m.searchMode {
		return statusOKStyle.Render(m.status + " • filter: " + search)
	}

	switch {
	case strings.HasPrefix(m.status, "error:"),
		strings.HasPrefix(m.status, "editor error:"),
		strings.HasPrefix(m.status, "create failed:"),
		strings.HasPrefix(m.status, "category create failed:"),
		strings.HasPrefix(m.status, "delete failed:"),
		strings.HasPrefix(m.status, "rename failed:"):
		return statusErrStyle.Render(m.status)
	default:
		return statusOKStyle.Render(m.status)
	}
}

func (m Model) renderHelpModal() string {
	modalWidth := min(76, max(50, m.width-10))

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(textColor).
		Render("Help")

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHelpLine("j / k", "move up and down"),
		m.renderHelpLine("enter / o", "open note or toggle category"),
		m.renderHelpLine("h / l", "collapse / expand category"),
		m.renderHelpLine("/", "search tree"),
		m.renderHelpLine("esc", "leave search, then clear on second press"),
		m.renderHelpLine("n", "new note in current category"),
		m.renderHelpLine("c", "create category"),
		m.renderHelpLine("dd", "delete note/category"),
		m.renderHelpLine("r", "refresh"),
		m.renderHelpLine("q", "quit"),
		m.renderHelpLine("esc / q / ?", "close help"),
	)

	footer := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("Press esc, q, or ? to close")

	card := lipgloss.NewStyle().
		Width(modalWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", body, "", footer))

	return card
}

func (m Model) renderCreateCategoryModal() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(textColor).
		Render("Create category")

	hint := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("Use / to create nested categories, e.g. work/project-a")

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		hint,
		"",
		m.categoryInput.View(),
		"",
		lipgloss.NewStyle().Foreground(mutedColor).Render("Enter to create • Esc to cancel"),
	)

	return lipgloss.NewStyle().
		Width(min(76, max(48, m.width-10))).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Render(body)
}

func (m Model) renderHelpLine(k, desc string) string {
	keyStyle := lipgloss.NewStyle().
		Width(14).
		Bold(true).
		Foreground(accentSoftColor)

	descStyle := lipgloss.NewStyle().
		Foreground(textColor)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		keyStyle.Render(k),
		descStyle.Render(desc),
	)
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
		m.status = "press d again to delete category: " + item.Name

	case treeNote:
		if item.Note == nil {
			return
		}
		m.deletePending = &deletePending{
			kind:    deleteTargetNote,
			relPath: item.Note.Path,
			name:    item.Note.Name,
		}
		m.status = "press d again to delete note: " + item.Note.Name
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
