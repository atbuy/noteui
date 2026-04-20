package tui

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/notes"
)

func (m Model) currentInAppEditableNote() (*notes.Note, bool, string) {
	if m.listMode == listModeTodos {
		item := m.currentTodoItem()
		if item == nil {
			return nil, false, "no note selected"
		}
		noteCopy := item.Note
		return &noteCopy, item.IsTemp, ""
	}

	if m.listMode == listModeTemporary {
		n := m.currentTempNote()
		if n == nil {
			return nil, false, "no temporary note selected"
		}
		return n, true, ""
	}

	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil {
			return nil, false, "no note selected"
		}
		if item.Kind != pinItemNote && item.Kind != pinItemTemporaryNote {
			return nil, false, "press enter to jump to item first"
		}
		for _, note := range m.notes {
			if note.Path == item.Path {
				noteCopy := note
				return &noteCopy, false, ""
			}
		}
		for _, note := range m.tempNotes {
			if note.Path == item.Path {
				noteCopy := note
				return &noteCopy, true, ""
			}
		}
		return nil, false, "note not found"
	}

	item := m.currentTreeItem()
	if item == nil {
		return nil, false, "no note selected"
	}
	if item.Kind == treeCategory {
		return nil, false, "select a note first"
	}
	if item.Kind == treeRemoteNote {
		return nil, false, "note is only on the server; press i to import it or I to import all"
	}
	if item.Note == nil {
		return nil, false, "no note selected"
	}
	noteCopy := *item.Note
	return &noteCopy, false, ""
}

func (m Model) inAppEditorUsesPreview() bool {
	return !m.editorFullscreen
}

func (m Model) inAppEditorSize() (int, int) {
	if m.inAppEditorUsesPreview() {
		width := m.preview.Width
		height := m.preview.Height
		if width > 0 && height > 0 {
			return width, height
		}
		_, rightWidth := m.panelWidths()
		return max(20, rightWidth-8), max(6, m.height-14)
	}
	return max(20, m.width), max(3, m.height)
}

func (m *Model) openInAppEditorCurrent() tea.Cmd {
	note, isTemp, status := m.currentInAppEditableNote()
	if note == nil {
		m.status = status
		return nil
	}

	m.editorRestoreFocus = m.focus
	m.pendingInAppEditPath = note.Path
	m.pendingInAppEditRel = note.RelPath
	m.pendingInAppEditTemp = isTemp

	if note.Encrypted {
		if strings.TrimSpace(m.sessionPassphrase) == "" {
			m.pendingEncryptPath = note.Path
			m.showPassphraseModal = true
			m.passphraseModalCtx = "unlock_in_app"
			m.passphraseInput.SetValue("")
			m.passphraseInput.Focus()
			m.status = "enter passphrase"
			return nil
		}
	}

	m.status = "opening in-app editor"
	if note.Encrypted {
		return saveNoteVersionAndEditorLoadCmd(
			m.rootDir,
			note.Path,
			note.RelPath,
			m.sessionPassphrase,
			isTemp,
		)
	}
	return editorLoadCmd(m.rootDir, note.Path, note.RelPath, note.Encrypted, m.sessionPassphrase, isTemp)
}

func (m *Model) closeInAppEditor(status string) {
	m.editorActive = false
	m.editorModel = nil
	m.focus = m.editorRestoreFocus
	m.editorRestoreFocus = focusTree
	m.editorLinkPickerMode = false
	m.showEditorURLPrompt = false
	m.editorURLInput.Blur()
	m.editorURLInput.SetValue("")
	m.pendingInAppEditPath = ""
	m.pendingInAppEditRel = ""
	m.pendingInAppEditTemp = false
	if strings.TrimSpace(status) != "" {
		m.status = status
	}
}

func (m *Model) openEditorLinkPicker() {
	items := make([]paletteItem, 0, len(m.notes)+len(m.tempNotes))
	for _, n := range m.notes {
		if isConflictCopy(n.RelPath) {
			continue
		}
		items = append(items, paletteItem{
			kind:  paletteKindNote,
			title: n.Title(),
			sub:   n.RelPath,
			note:  n,
		})
	}
	for _, n := range m.tempNotes {
		items = append(items, paletteItem{
			kind:  paletteKindTempNote,
			title: n.Title(),
			sub:   ".tmp/" + n.RelPath,
			note:  n,
		})
	}
	m.commandPaletteItems = items
	m.commandPaletteInput.Reset()
	m.commandPaletteInput.Focus()
	m.commandPaletteCursor = 0
	m.editorLinkPickerMode = true
	m.rebuildPaletteFiltered()
	m.showCommandPalette = true
}

func (m *Model) openEditorURLPrompt() {
	m.showEditorURLPrompt = true
	m.editorURLInput.SetValue("")
	m.editorURLInput.Focus()
}

func (m Model) renderEditorURLPromptModal() string {
	modalWidth, innerWidth := m.modalDimensions(56, 84)

	inputView := m.editorURLInput.View()
	inputField := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(inputView)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderModalTitle("Insert URL", innerWidth),
		m.renderModalHint("Wrap selection or insert [label](url) at cursor.", innerWidth),
		"",
		inputField,
		"",
		m.renderModalFooter("enter insert • esc cancel", innerWidth),
	)

	return modalCardStyle(modalWidth).Render(content)
}

func editorSavedLabel(path string) string {
	if strings.TrimSpace(path) == "" {
		return "note saved"
	}
	return "saved: " + filepath.Base(path)
}
