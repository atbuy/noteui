package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"atbuy/noteui/internal/editor"
	"atbuy/noteui/internal/notes"
)

type Model struct {
	rootDir      string
	list         list.Model
	notes        []notes.Note
	selected     *notes.Note
	width        int
	height       int
	previewWidth int
	status       string
	searchFocus  bool
}

type notesLoadedMsg struct {
	notes []notes.Note
	err   error
}

type noteCreatedMsg struct {
	path string
	err  error
}

func New(root string) Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.SetHeight(2)

	l := list.New([]list.Item{}, delegate, 30, 20)
	l.Title = fmt.Sprintf("Notes (%s)", filepath.Clean(root))
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Padding(0, 1)

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Open, keys.NewNote, keys.Refresh}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Open, keys.NewNote, keys.Refresh, keys.Focus, keys.Quit}
	}

	return Model{
		rootDir: root,
		list:    l,
		status:  "loading notes...",
	}
}

func (m Model) Init() tea.Cmd {
	return loadNotesCmd(m.rootDir)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		listWidth := max(30, msg.Width/3)
		m.previewWidth = max(40, msg.Width-listWidth-4)
		m.list.SetSize(listWidth, msg.Height-4)

		return m, nil

	case notesLoadedMsg:
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
			return m, nil
		}

		m.notes = msg.notes
		items := make([]list.Item, 0, len(msg.notes))
		for _, n := range msg.notes {
			items = append(items, n)
		}
		m.list.SetItems(items)

		if len(msg.notes) > 0 {
			m.selected = &msg.notes[0]
			m.status = fmt.Sprintf("loaded %d notes", len(msg.notes))
		} else {
			m.selected = nil
			m.status = "no notes found"
		}

		return m, nil

	case noteCreatedMsg:
		if msg.err != nil {
			m.status = "create failed: " + msg.err.Error()
			return m, nil
		}
		m.status = "created: " + msg.path
		return m, tea.Batch(loadNotesCmd(m.rootDir), editor.Open(msg.path))

	case editor.FinishedMsg:
		if msg.Err != nil {
			m.status = "editor error: " + msg.Err.Error()
			return m, nil
		}
		m.status = "editor closed"
		return m, loadNotesCmd(m.rootDir)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Refresh):
			m.status = "refreshing..."
			return m, loadNotesCmd(m.rootDir)

		case key.Matches(msg, keys.Focus):
			m.searchFocus = !m.searchFocus
			if m.searchFocus {
				m.list.FilterInput.Focus()
				m.status = "search focused"
			} else {
				m.list.FilterInput.Blur()
				m.status = "list focused"
			}
			return m, nil

		case key.Matches(msg, keys.NewNote):
			return m, createNoteCmd(m.rootDir)

		case key.Matches(msg, keys.Open):
			if n := m.currentNote(); n != nil {
				m.status = "opening in nvim: " + n.RelPath
				return m, editor.Open(n.Path)
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	if n := m.currentNote(); n != nil {
		m.selected = n
	}

	return m, cmd
}

func (m Model) View() string {
	leftWidth := max(30, m.width/3)

	left := panelStyle(leftWidth, m.height).Render(m.list.View())
	right := panelStyle(m.previewWidth, m.height).Render(m.previewView())

	footer := footerStyle.Render(
		"/ search • tab focus • enter open • n new • r refresh • q quit",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, left, right),
		footer,
	)
}

func (m Model) previewView() string {
	if m.selected == nil {
		return "No note selected"
	}

	content := m.selected.Preview
	if strings.TrimSpace(content) == "" {
		content = "(empty file)"
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render(m.selected.RelPath),
		mutedStyle.Render("Category: "+m.selected.Category),
		mutedStyle.Render("Modified: "+m.selected.ModTime.Format("2006-01-02 15:04:05")),
		"",
		content,
	)
}

func (m Model) currentNote() *notes.Note {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}

	n, ok := item.(notes.Note)
	if !ok {
		return nil
	}
	return &n
}

func loadNotesCmd(root string) tea.Cmd {
	return func() tea.Msg {
		n, err := notes.Discover(root)
		return notesLoadedMsg{notes: n, err: err}
	}
}

func createNoteCmd(root string) tea.Cmd {
	return func() tea.Msg {
		path, err := notes.CreateInboxNote(root)
		return noteCreatedMsg{path: path, err: err}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
