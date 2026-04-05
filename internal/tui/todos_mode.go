package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/notes"
)

type todoListItem struct {
	Note     notes.Note
	RelPath  string
	IsTemp   bool
	Todo     notes.TodoItem
	Source   string
	MetaText string
}

func (m *Model) rebuildTodoItems() {
	items := make([]todoListItem, 0, len(m.notes)+len(m.tempNotes))
	for _, note := range m.notes {
		items = append(items, buildTodoListItems(note, false)...)
	}
	for _, note := range m.tempNotes {
		items = append(items, buildTodoListItems(note, true)...)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return todoListItemLess(items[i], items[j])
	})
	m.todoItems = items
	m.clampTodoCursor()
}

func buildTodoListItems(note notes.Note, isTemp bool) []todoListItem {
	todos := notes.ExtractTodoItems(note.Preview, true)
	items := make([]todoListItem, 0, len(todos))
	relPath := filepath.ToSlash(note.RelPath)
	source := relPath
	if isTemp {
		source = filepath.ToSlash(filepath.Join(".tmp", note.RelPath))
	}
	for _, todo := range todos {
		metaParts := make([]string, 0, 2)
		if todo.Metadata.Priority > 0 {
			metaParts = append(metaParts, fmt.Sprintf("p%d", todo.Metadata.Priority))
		}
		if todo.Metadata.DueDate != "" {
			metaParts = append(metaParts, "due:"+todo.Metadata.DueDate)
		}
		items = append(items, todoListItem{
			Note:     note,
			RelPath:  relPath,
			IsTemp:   isTemp,
			Todo:     todo,
			Source:   source,
			MetaText: strings.Join(metaParts, " "),
		})
	}
	return items
}

func todoListItemLess(a, b todoListItem) bool {
	aHasDue := !a.Todo.Metadata.DueTime.IsZero()
	bHasDue := !b.Todo.Metadata.DueTime.IsZero()
	if aHasDue != bHasDue {
		return aHasDue
	}
	if aHasDue && !a.Todo.Metadata.DueTime.Equal(b.Todo.Metadata.DueTime) {
		return a.Todo.Metadata.DueTime.Before(b.Todo.Metadata.DueTime)
	}
	aPriority := todoPrioritySortValue(a.Todo.Metadata.Priority)
	bPriority := todoPrioritySortValue(b.Todo.Metadata.Priority)
	if aPriority != bPriority {
		return aPriority < bPriority
	}
	if a.Source != b.Source {
		return a.Source < b.Source
	}
	return a.Todo.Line < b.Todo.Line
}

func todoPrioritySortValue(priority int) int {
	if priority <= 0 {
		return 99
	}
	return priority
}

func (m Model) filteredTodoItems() []todoListItem {
	query := strings.ToLower(strings.TrimSpace(m.activeSearchQuery()))
	if query == "" {
		out := make([]todoListItem, len(m.todoItems))
		copy(out, m.todoItems)
		return out
	}
	out := make([]todoListItem, 0, len(m.todoItems))
	for _, item := range m.todoItems {
		if m.todoItemMatches(item, query) {
			out = append(out, item)
		}
	}
	return out
}

func (m Model) todoItemMatches(item todoListItem, query string) bool {
	if after, ok := strings.CutPrefix(query, "#"); ok {
		tag := strings.TrimSpace(after)
		if tag == "" {
			return true
		}
		for _, candidate := range item.Note.Tags {
			if strings.Contains(strings.ToLower(candidate), tag) {
				return true
			}
		}
		return false
	}
	blob := strings.ToLower(strings.Join([]string{
		item.Todo.Text,
		item.Todo.DisplayText,
		item.Note.Title(),
		item.Source,
		strings.Join(item.Note.Tags, " "),
		item.MetaText,
	}, " "))
	for _, term := range strings.Fields(query) {
		if !strings.Contains(blob, term) {
			return false
		}
	}
	return true
}

func (m *Model) clampTodoCursor() {
	items := m.filteredTodoItems()
	if len(items) == 0 {
		m.todoCursor = 0
		return
	}
	if m.todoCursor < 0 {
		m.todoCursor = 0
		return
	}
	if m.todoCursor >= len(items) {
		m.todoCursor = len(items) - 1
	}
}

func (m Model) currentTodoItem() *todoListItem {
	items := m.filteredTodoItems()
	if len(items) == 0 || m.todoCursor < 0 || m.todoCursor >= len(items) {
		return nil
	}
	item := items[m.todoCursor]
	return &item
}

func (m *Model) moveTodoCursor(delta int) {
	items := m.filteredTodoItems()
	if len(items) == 0 {
		m.todoCursor = 0
		m.syncSelectedNote()
		return
	}
	next := max(0, min(len(items)-1, m.todoCursor+delta))
	m.todoCursor = next
	m.syncSelectedNote()
}

func (m *Model) toggleTodosMode() {
	if m.listMode == listModeTodos {
		if m.lastNonPinsMode == listModeTemporary {
			m.switchToTemporaryMode()
		} else {
			m.switchToNotesMode()
		}
		return
	}
	m.listMode = listModeTodos
	m.status = "todos"
	m.syncSelectedNote()
}

func (m Model) todosEmptyStateMessage() string {
	if query := m.activeSearchQuery(); query != "" {
		return fmt.Sprintf("No open todos match %q. Press esc to clear search.", query)
	}
	return "No open todos. Press T to create a todo list or add unchecked tasks to notes."
}

func (m Model) todosPreviewEmptyMessage() string {
	if query := m.activeSearchQuery(); query != "" {
		return fmt.Sprintf("No open todos match %q. Press esc to clear search.", query)
	}
	return "No open todo selected. Use j/k to choose a task or ctrl+t to leave Todos."
}

func (m Model) renderTodoListView() string {
	items := m.filteredTodoItems()
	if len(items) == 0 {
		return m.renderPaneEmptyState(m.todosEmptyStateMessage())
	}
	rowWidth := m.treeInnerWidth()
	lines := make([]string, 0, len(items))
	for i, item := range items {
		lines = append(lines, m.renderTodoListRow(item, rowWidth, i == m.todoCursor))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func todoDueDateIsOverdue(item todoListItem) bool {
	if strings.TrimSpace(item.Todo.Metadata.DueDate) == "" {
		return false
	}
	return item.Todo.Metadata.DueDate < time.Now().Format("2006-01-02")
}

func (m Model) renderTodoListRow(item todoListItem, rowWidth int, selected bool) string {
	innerWidth := max(0, rowWidth-2)
	text := strings.TrimSpace(item.Todo.DisplayText)
	if text == "" {
		text = strings.TrimSpace(item.Todo.Text)
	}
	if text == "" {
		text = item.Source
	}

	rowBg := bgSoftColor
	textFg := textColor
	priorityFg := accentSoftColor
	dueFg := mutedColor
	sourceFg := mutedColor
	if item.Todo.Metadata.Priority == 1 {
		textFg = accentColor
		priorityFg = accentColor
	}
	if todoDueDateIsOverdue(item) {
		dueFg = errorColor
	}
	if selected {
		rowBg = selectedBgColor
		textFg = selectedFgColor
		priorityFg = selectedFgColor
		sourceFg = selectedFgColor
		if !todoDueDateIsOverdue(item) {
			dueFg = mutedColor
		}
	}

	priority := ""
	if item.Todo.Metadata.Priority > 0 {
		priority = fmt.Sprintf("[p%d]", item.Todo.Metadata.Priority)
	}
	due := ""
	if item.Todo.Metadata.DueDate != "" {
		due = "[due:" + item.Todo.Metadata.DueDate + "]"
	}
	source := ""
	if strings.TrimSpace(item.Source) != "" {
		source = "· " + item.Source
	}

	fixedWidth := 0
	if priority != "" {
		fixedWidth += lipgloss.Width(priority) + 1
	}
	if due != "" {
		fixedWidth += lipgloss.Width(due) + 1
	}
	textWidth := max(1, innerWidth-fixedWidth)
	textRendered := lipgloss.NewStyle().Foreground(textFg).Background(rowBg).Bold(selected && boldSelected).Render(truncateToWidth(text, textWidth))
	plainWidth := lipgloss.Width(truncateToWidth(text, textWidth))

	sep := lipgloss.NewStyle().Background(rowBg).Render(" ")
	parts := make([]string, 0, 4)
	usedWidth := 0
	appendPart := func(part string, width int) {
		if len(parts) > 0 {
			parts = append(parts, sep)
			usedWidth++
		}
		parts = append(parts, part)
		usedWidth += width
	}
	if priority != "" {
		appendPart(lipgloss.NewStyle().Foreground(priorityFg).Background(rowBg).Bold(selected && boldSelected).Render(priority), lipgloss.Width(priority))
	}
	appendPart(textRendered, plainWidth)
	if due != "" && usedWidth+1+lipgloss.Width(due) <= innerWidth {
		appendPart(lipgloss.NewStyle().Foreground(dueFg).Background(rowBg).Bold(selected && boldSelected).Render(due), lipgloss.Width(due))
	}
	if source != "" && usedWidth < innerWidth {
		sourceWidth := innerWidth - usedWidth
		if len(parts) > 0 {
			sourceWidth--
		}
		if sourceWidth > 4 {
			truncated := truncateToWidth(source, sourceWidth)
			appendPart(lipgloss.NewStyle().Foreground(sourceFg).Background(rowBg).Bold(selected && boldSelected).Render(truncated), lipgloss.Width(truncated))
		}
	}

	content := strings.Join(parts, "")
	if usedWidth < innerWidth {
		content += lipgloss.NewStyle().Background(rowBg).Render(strings.Repeat(" ", innerWidth-usedWidth))
	}
	return lipgloss.NewStyle().Width(rowWidth).Padding(0, 1).Background(rowBg).Render(content)
}

func (m *Model) syncSelectedTodoInPreview() bool {
	selected := m.currentTodoItem()
	if selected == nil || strings.TrimSpace(m.previewPath) == "" || selected.Note.Path != m.previewPath {
		m.previewTodoCursor = -1
		m.previewTodoNavMode = false
		return false
	}
	m.previewTodoNavMode = true
	for i, todo := range m.previewTodos {
		if todo.rawLine == selected.Todo.Line {
			m.previewTodoCursor = i
			return true
		}
	}
	m.previewTodoCursor = -1
	return false
}
