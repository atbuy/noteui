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
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Bold(true)

	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("62"))

	l := list.New([]list.Item{}, delegate, 30, 20)
	l.Title = fmt.Sprintf("Notes (%s)", filepath.Clean(root))
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	l.Styles.NoItems = emptyStyle
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(accentSoftColor)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(accentColor)

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

		usableWidth := max(40, msg.Width-6)
		leftWidth := max(28, usableWidth/3)
		gap := 2
		rightWidth := max(30, usableWidth-leftWidth-gap)

		m.previewWidth = rightWidth
		m.list.SetSize(
			max(16, leftWidth-4),
			max(8, msg.Height-10),
		)

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
		filtering := m.list.FilterState() == list.Filtering

		// Global quit always works.
		if key.Matches(msg, keys.Quit) {
			return m, tea.Quit
		}

		// While filtering, the list owns almost all keys.
		if filtering {
			switch msg.String() {
			case "esc":
				m.list.ResetFilter()
				m.list.FilterInput.Blur()
				m.searchFocus = false
				m.status = "list focused"
				return m, nil

			case "enter":
				m.list.FilterInput.Blur()
				m.searchFocus = false
				m.status = "filter applied"
				if n := m.currentNote(); n != nil {
					m.selected = n
				}
				return m, nil
			}

			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			if n := m.currentNote(); n != nil {
				m.selected = n
			}
			return m, cmd
		}

		// Not filtering.
		switch {
		case key.Matches(msg, keys.Search), key.Matches(msg, keys.Focus):
			m.searchFocus = true
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd

		case msg.String() == "esc":
			m.list.ResetFilter()
			m.list.FilterInput.Blur()
			m.searchFocus = false
			m.status = "list focused"
			return m, nil

		case key.Matches(msg, keys.Refresh):
			m.status = "refreshing..."
			return m, loadNotesCmd(m.rootDir)

		case key.Matches(msg, keys.NewNote):
			return m, createNoteCmd(m.rootDir)

		case key.Matches(msg, keys.Open):
			if n := m.currentNote(); n != nil {
				m.status = "opening in nvim: " + n.RelPath
				return m, editor.Open(n.Path)
			}
			return m, nil
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
	usableWidth := max(40, m.width-6)
	leftWidth := max(28, usableWidth/3)
	gap := "  "
	rightWidth := max(30, usableWidth-leftWidth-len(gap))

	leftBody := lipgloss.JoinVertical(
		lipgloss.Left,
		panelTitleStyle.Render("Notes"),
		m.list.View(),
	)

	rightBody := lipgloss.JoinVertical(
		lipgloss.Left,
		panelTitleStyle.Render("Preview"),
		m.previewView(),
	)

	leftFocused := !m.searchFocus
	rightFocused := m.searchFocus

	left := panelStyle(leftWidth, m.height, leftFocused).Render(leftBody)
	right := panelStyle(rightWidth, m.height, rightFocused).Render(rightBody)

	title := titleBarStyle.
		Width(usableWidth).
		Render(" notetui ")

	footer := footerStyle.
		Width(usableWidth).
		Render(m.renderStatus() + "   •   / search   tab focus   enter open   n new   r refresh   q quit")

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right)

	return appStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			body,
			footer,
		),
	)
}

func (m Model) previewView() string {
	if m.selected == nil {
		return emptyStyle.Render("No note selected")
	}

	content := m.selected.Preview
	if strings.TrimSpace(content) == "" {
		content = "(empty file)"
	}

	metaRow := lipgloss.JoinHorizontal(
		lipgloss.Left,
		chipStyle.Render("Category: "+m.selected.Category),
		chipStyle.Render("Modified: "+m.selected.ModTime.Format("2006-01-02 15:04")),
	)

	contentStyle := lipgloss.NewStyle().
		Width(max(20, m.previewWidth-8))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render(m.selected.RelPath),
		metaRow,
		"",
		contentStyle.Render(content),
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

func (m Model) renderStatus() string {
	switch {
	case strings.HasPrefix(m.status, "error:"),
		strings.HasPrefix(m.status, "editor error:"),
		strings.HasPrefix(m.status, "create failed:"):
		return statusErrStyle.Render(m.status)
	default:
		return statusOKStyle.Render(m.status)
	}
}
