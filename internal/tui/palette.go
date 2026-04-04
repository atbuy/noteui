package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/notes"
)

const paletteMaxVisible = 12

// paletteKind identifies the type of an item in the command palette.
// New kinds (e.g. paletteKindCommand for app-level commands) can be added here
// without changing any other palette logic.
type paletteKind int

const (
	paletteKindNote     paletteKind = iota // note in the main tree
	paletteKindTempNote                    // note in .tmp/
	// paletteKindCommand — future: named app commands
)

// paletteItem is one entry shown in the command palette.
type paletteItem struct {
	kind  paletteKind
	title string     // primary display (note title)
	sub   string     // secondary display (relPath, or ".tmp/relPath")
	note  notes.Note // valid for paletteKindNote and paletteKindTempNote
}

// isConflictCopy reports whether relPath matches the conflict copy naming pattern
// produced by createConflict(): "base.conflict-YYYYMMDD-HHMMSS.ext".
// Conflict copies are excluded from the palette since they are conflict artifacts,
// not regular notes — the user resolves them from the tree view.
func isConflictCopy(relPath string) bool {
	name := filepath.Base(relPath)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	idx := strings.LastIndex(base, ".conflict-")
	if idx < 0 {
		return false
	}
	suffix := base[idx+len(".conflict-"):]
	return len(suffix) == 15 // "YYYYMMDD-HHMMSS"
}

func (m *Model) openCommandPalette() {
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
	m.rebuildPaletteFiltered()
	m.showCommandPalette = true
}

func (m *Model) rebuildPaletteFiltered() {
	query := strings.TrimSpace(m.commandPaletteInput.Value())
	if query == "" {
		m.commandPaletteFiltered = m.commandPaletteItems
	} else {
		// Always allocate a fresh slice. Reusing commandPaletteFiltered as a buffer
		// is unsafe when it aliases commandPaletteItems (which happens when query was
		// previously empty), because appending into the buffer overwrites commandPaletteItems.
		filtered := make([]paletteItem, 0, len(m.commandPaletteItems))
		for _, item := range m.commandPaletteItems {
			if m.noteMatches(item.note, query) {
				filtered = append(filtered, item)
			}
		}
		m.commandPaletteFiltered = filtered
	}
	if m.commandPaletteCursor >= len(m.commandPaletteFiltered) {
		m.commandPaletteCursor = max(0, len(m.commandPaletteFiltered)-1)
	}
}

func (m *Model) commitPaletteSelection() {
	if len(m.commandPaletteFiltered) == 0 {
		return
	}
	item := m.commandPaletteFiltered[m.commandPaletteCursor]
	m.showCommandPalette = false
	m.commandPaletteInput.Blur()

	switch item.kind {
	case paletteKindNote:
		m.switchToNotesMode()
		m.selectTreeNote(item.note.RelPath)
	case paletteKindTempNote:
		m.switchToTemporaryMode()
		m.selectTemporaryNote(item.note.RelPath)
	}
}

func (m Model) renderCommandPaletteModal() string {
	modalWidth, innerWidth := m.modalDimensions(60, 90)
	total := len(m.commandPaletteFiltered)

	// Compute scroll so the cursor row is always visible.
	scroll := max(0, m.commandPaletteCursor-paletteMaxVisible+1)
	scroll = max(0, min(scroll, max(0, total-paletteMaxVisible)))

	// Title row: "Quick open" left, note count right.
	titleLeft := lipgloss.NewStyle().
		Foreground(modalAccentColor).
		Background(modalBgColor).
		Bold(true).
		Render("Quick open")
	countStr := fmt.Sprintf("%d notes", total)
	countText := lipgloss.NewStyle().
		Foreground(modalMutedColor).
		Background(modalBgColor).
		Render(countStr)
	gapSize := max(0, innerWidth-lipgloss.Width(titleLeft)-lipgloss.Width(countText))
	gapStr := lipgloss.NewStyle().Background(modalBgColor).Render(strings.Repeat(" ", gapSize))
	titleRow := fillWidthBackground(titleLeft+gapStr+countText, innerWidth, modalBgColor)

	// Input row.
	inputCopy := m.commandPaletteInput
	inputCopy.Width = max(12, innerWidth-2)
	inputCopy.TextStyle = lipgloss.NewStyle().Foreground(modalTextColor).Background(modalBgColor)
	inputCopy.PlaceholderStyle = lipgloss.NewStyle().Foreground(modalMutedColor).Background(modalBgColor)
	inputCopy.Cursor.Style = lipgloss.NewStyle().Foreground(modalTextColor).Background(modalTextColor)
	inputRow := fillWidthBackground(inputCopy.View(), innerWidth, modalBgColor)

	// Divider.
	divider := lipgloss.NewStyle().
		Width(innerWidth).
		Foreground(modalMutedColor).
		Background(modalBgColor).
		Render(strings.Repeat("─", innerWidth))

	// Result list.
	var resultLines []string
	if total == 0 {
		empty := lipgloss.NewStyle().
			Width(innerWidth).
			Foreground(modalMutedColor).
			Background(modalBgColor).
			Render("No notes found")
		resultLines = append(resultLines, empty)
	} else {
		end := min(scroll+paletteMaxVisible, total)
		for i := scroll; i < end; i++ {
			resultLines = append(resultLines, m.renderPaletteRow(m.commandPaletteFiltered[i], i, innerWidth))
		}
		if scroll > 0 {
			indicator := lipgloss.NewStyle().
				Width(innerWidth).Align(lipgloss.Center).
				Foreground(modalMutedColor).Background(modalBgColor).
				Render(fmt.Sprintf("↑ %d more", scroll))
			resultLines = append([]string{indicator}, resultLines...)
		}
		if end < total {
			indicator := lipgloss.NewStyle().
				Width(innerWidth).Align(lipgloss.Center).
				Foreground(modalMutedColor).Background(modalBgColor).
				Render(fmt.Sprintf("↓ %d more", total-end))
			resultLines = append(resultLines, indicator)
		}
	}

	resultsBlock := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(strings.Join(resultLines, "\n"))

	footer := m.renderModalFooter("↑↓ navigate   enter open   esc cancel", innerWidth)

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			titleRow,
			m.renderModalBlank(innerWidth),
			inputRow,
			m.renderModalBlank(innerWidth),
			divider,
			resultsBlock,
			divider,
			m.renderModalBlank(innerWidth),
			footer,
		))

	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderPaletteRow(item paletteItem, idx, innerWidth int) string {
	selected := idx == m.commandPaletteCursor

	cursor := "  "
	if selected {
		cursor = "› "
	}
	cursorW := lipgloss.Width(cursor)
	subWidth := max(8, min(36, innerWidth/3))
	titleWidth := max(8, innerWidth-subWidth-cursorW-2)

	titleStr := paletteTruncate(item.title, titleWidth)
	subStr := paletteTruncateLeft(item.sub, subWidth)

	if selected {
		return lipgloss.NewStyle().Foreground(selectedFgColor).Background(selectedBgColor).Render(cursor) +
			lipgloss.NewStyle().Width(titleWidth).Foreground(selectedFgColor).Background(selectedBgColor).Render(titleStr) +
			lipgloss.NewStyle().Width(2).Background(selectedBgColor).Render("  ") +
			lipgloss.NewStyle().Width(subWidth).Align(lipgloss.Right).Foreground(selectedFgColor).Background(selectedBgColor).Render(subStr)
	}
	return lipgloss.NewStyle().Foreground(modalMutedColor).Background(modalBgColor).Render(cursor) +
		lipgloss.NewStyle().Width(titleWidth).Foreground(modalTextColor).Background(modalBgColor).Render(titleStr) +
		lipgloss.NewStyle().Width(2).Background(modalBgColor).Render("  ") +
		lipgloss.NewStyle().Width(subWidth).Align(lipgloss.Right).Foreground(modalMutedColor).Background(modalBgColor).Render(subStr)
}

// paletteTruncate truncates s to at most maxWidth visual characters.
func paletteTruncate(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// paletteTruncateLeft truncates s from the left to at most maxWidth visual characters.
func paletteTruncateLeft(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width("…"+string(runes)) > maxWidth {
		runes = runes[1:]
	}
	return "…" + string(runes)
}
