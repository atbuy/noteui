package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/config"
)

const themePickerVisible = 10

// openThemePicker initialises the theme picker modal and positions the cursor
// on the currently active theme so the live preview does not flicker on open.
func (m *Model) openThemePicker() {
	themes := BuiltinThemes()
	current := NormalizeThemeName(m.cfg.Theme.Name)
	cursor := 0
	for i, t := range themes {
		if t.Name == current {
			cursor = i
			break
		}
	}
	m.showThemePicker = true
	m.themePickerInput.SetValue("")
	m.themePickerInput.Blur()
	m.themePickerCursor = cursor
	m.themePickerOrigTheme = current
	m.refreshThemePickerSelection(false)
	m.status = "theme picker"
}

// themePickerCenteredOffset returns a scroll offset that centres cursor in the
// visible window as much as possible.
func themePickerCenteredOffset(cursor, total int) int {
	offset := cursor - themePickerVisible/2
	if offset < 0 {
		return 0
	}
	maxOffset := total - themePickerVisible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		return maxOffset
	}
	return offset
}

func themePickerMatchesQuery(t BuiltinThemeEntry, query string) bool {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		return true
	}
	blob := strings.ToLower(strings.Join([]string{
		t.Name,
		strings.Join(t.Aliases, " "),
		t.Description,
	}, " "))
	for _, term := range terms {
		if !strings.Contains(blob, term) {
			return false
		}
	}
	return true
}

func (m Model) filteredThemePickerIndices() []int {
	themes := BuiltinThemes()
	query := m.themePickerInput.Value()
	out := make([]int, 0, len(themes))
	for i, t := range themes {
		if themePickerMatchesQuery(t, query) {
			out = append(out, i)
		}
	}
	return out
}

func (m Model) themePickerCursorPosition(filtered []int) int {
	for i, idx := range filtered {
		if idx == m.themePickerCursor {
			return i
		}
	}
	return 0
}

func (m *Model) refreshThemePickerSelection(applyPreview bool) {
	filtered := m.filteredThemePickerIndices()
	if len(filtered) == 0 {
		m.themePickerScrollOffset = 0
		return
	}

	pos := -1
	for i, idx := range filtered {
		if idx == m.themePickerCursor {
			pos = i
			break
		}
	}
	if pos == -1 {
		pos = 0
		m.themePickerCursor = filtered[0]
		if applyPreview {
			m.applyThemePickerPreview(m.themePickerCursor)
		}
	}
	m.themePickerScrollOffset = themePickerCenteredOffset(pos, len(filtered))
}

func (m *Model) applyThemePickerPreview(themeIdx int) {
	themes := BuiltinThemes()
	if themeIdx < 0 || themeIdx >= len(themes) {
		return
	}
	hovCfg := m.cfg
	hovCfg.Theme.Name = themes[themeIdx].Name
	ApplyTheme(hovCfg)
	m.previewPath = ""
	m.refreshPreview()
}

func (m Model) themePickerResolvedEntry(themeIdx int) BuiltinThemeEntry {
	entry := BuiltinThemes()[themeIdx]
	cfg := m.cfg
	cfg.Theme.Name = entry.Name
	entry.Palette = resolveThemePalette(cfg)
	return entry
}

func summarizeThemeLabels(labels []string, limit int) string {
	if len(labels) <= limit {
		return strings.Join(labels, ", ")
	}
	return strings.Join(labels[:limit], ", ") + fmt.Sprintf(", +%d more", len(labels)-limit)
}

func (m Model) themePickerGuardrailLines(themeIdx int) []string {
	cfg := m.cfg
	cfg.Theme.Name = BuiltinThemes()[themeIdx].Name
	raw := resolveThemePaletteRaw(cfg)

	lines := make([]string, 0, 2)
	if themeHasColorOverrides(m.cfg.Theme) {
		lines = append(lines, "Preview keeps current theme color overrides; enter still saves only theme.name.")
	}
	if adjusted := themePaletteAccessibilityAdjustments(raw); len(adjusted) > 0 {
		lines = append(lines, "Low-contrast colors auto-adjusted for readability: "+summarizeThemeLabels(adjusted, 3))
	}
	return lines
}

func (m Model) themePickerSummary(filtered []int) string {
	total := len(BuiltinThemes())
	query := strings.TrimSpace(m.themePickerInput.Value())
	if query == "" {
		return fmt.Sprintf("%d built-in themes", total)
	}
	if len(filtered) == 0 {
		return fmt.Sprintf("0 of %d themes match name, alias, or description", total)
	}
	return fmt.Sprintf("%d of %d themes match name, alias, or description • %d/%d selected", len(filtered), total, m.themePickerCursorPosition(filtered)+1, len(filtered))
}

func (m *Model) updateThemePicker(msg tea.KeyMsg) tea.Cmd {
	switch {
	case msg.String() == "esc":
		m.cancelThemePicker()
		return nil
	case msg.Type == tea.KeyEnter:
		if len(m.filteredThemePickerIndices()) == 0 {
			m.status = "no matching themes"
			return nil
		}
		m.confirmThemePicker()
		return nil
	case msg.Type == tea.KeyTab:
		if m.themePickerInput.Focused() {
			m.themePickerInput.Blur()
		} else {
			m.themePickerInput.Focus()
		}
		return nil
	}

	if m.themePickerInput.Focused() {
		switch msg.Type {
		case tea.KeyUp:
			m.moveThemePickerCursor(-1)
			return nil
		case tea.KeyDown:
			m.moveThemePickerCursor(1)
			return nil
		}
		if !shouldUpdateTextInput(msg, m.themePickerInput) {
			return nil
		}
		before := m.themePickerInput.Value()
		var cmd tea.Cmd
		m.themePickerInput, cmd = m.themePickerInput.Update(msg)
		if m.themePickerInput.Value() != before {
			m.refreshThemePickerSelection(true)
		}
		return cmd
	}

	switch {
	case key.Matches(msg, keys.Search):
		m.themePickerInput.Focus()
	case key.Matches(msg, keys.MoveUp):
		m.moveThemePickerCursor(-1)
	case key.Matches(msg, keys.MoveDown):
		m.moveThemePickerCursor(1)
	}
	return nil
}

// moveThemePickerCursor shifts the cursor by delta (wrapping) and immediately
// applies the hovered theme for a live preview, including the preview pane.
func (m *Model) moveThemePickerCursor(delta int) {
	filtered := m.filteredThemePickerIndices()
	if len(filtered) == 0 {
		return
	}

	pos := m.themePickerCursorPosition(filtered)
	pos = (pos + delta + len(filtered)) % len(filtered)
	m.themePickerCursor = filtered[pos]

	// Keep cursor visible in the filtered scroll window.
	if pos >= m.themePickerScrollOffset+themePickerVisible {
		m.themePickerScrollOffset = pos - themePickerVisible + 1
	}
	if pos < m.themePickerScrollOffset {
		m.themePickerScrollOffset = pos
	}
	maxOffset := len(filtered) - themePickerVisible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.themePickerScrollOffset > maxOffset {
		m.themePickerScrollOffset = maxOffset
	}
	m.applyThemePickerPreview(m.themePickerCursor)
}

// confirmThemePicker saves the selected theme to config and closes the modal.
func (m *Model) confirmThemePicker() {
	themes := BuiltinThemes()
	name := themes[m.themePickerCursor].Name
	m.cfg.Theme.Name = name
	m.showThemePicker = false
	m.themePickerInput.Blur()
	if _, _, err := config.SaveTheme(name); err != nil {
		m.status = "could not save theme: " + err.Error()
	} else {
		m.status = "theme: " + name
	}
}

// cancelThemePicker closes the modal without saving and restores the original theme.
func (m *Model) cancelThemePicker() {
	m.showThemePicker = false
	m.themePickerInput.Blur()
	restoreCfg := m.cfg
	restoreCfg.Theme.Name = m.themePickerOrigTheme
	ApplyTheme(restoreCfg)
	// Re-render the preview with the restored theme's colours.
	m.previewPath = ""
	m.refreshPreview()
	m.status = "theme unchanged"
}

// renderThemePickerModal renders the theme picker as a centred modal card.
func (m Model) renderThemePickerModal() string {
	filtered := m.filteredThemePickerIndices()
	modalW, innerW := m.modalDimensions(64, 94)
	visibleRows := min(themePickerVisible, max(4, len(filtered)))
	if len(filtered) == 0 {
		visibleRows = 4
	}

	sections := []string{
		m.renderModalTitle("Theme Picker", innerW),
		m.renderModalBlank(innerW),
		m.renderModalHint(m.themePickerSummary(filtered), innerW),
		m.renderModalBlank(innerW),
		m.renderModalInputRow("Filter", m.themePickerInput, innerW),
		m.renderModalBlank(innerW),
	}

	if len(filtered) == 0 {
		sections = append(sections, lipgloss.NewStyle().
			Width(innerW).
			Height(visibleRows).
			Background(modalBgColor).
			Render(modalMutedStyle.Render("No themes match the current filter")))
	} else {
		hov := m.themePickerResolvedEntry(m.themePickerCursor)
		hovName := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(hov.Palette.AccentColor)).
			Background(modalBgColor).
			Render(hov.Name)
		header := hovName + modalMutedStyle.Render(" - ") + modalMutedStyle.Render(hov.Description)
		sections = append(sections, lipgloss.NewStyle().
			Width(innerW).
			Background(modalBgColor).
			Render(header))

		warningStyle := lipgloss.NewStyle().
			Foreground(modalAccentColor).
			Background(modalBgColor).
			Bold(true)
		for _, line := range m.themePickerGuardrailLines(m.themePickerCursor) {
			sections = append(sections, lipgloss.NewStyle().
				Width(innerW).
				Background(modalBgColor).
				Render(warningStyle.Render(line)))
		}
		sections = append(sections, m.renderModalBlank(innerW))

		end := min(m.themePickerScrollOffset+visibleRows, len(filtered))
		rowLines := make([]string, 0, visibleRows)
		for pos := m.themePickerScrollOffset; pos < end; pos++ {
			t := m.themePickerResolvedEntry(filtered[pos])
			selected := filtered[pos] == m.themePickerCursor

			swatch := func(hex string) string {
				return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("██")
			}
			swatches := swatch(t.Palette.BgColor) +
				swatch(t.Palette.PanelBgColor) +
				swatch(t.Palette.AccentColor) +
				swatch(t.Palette.AccentSoftColor) +
				swatch(t.Palette.TextColor) +
				swatch(t.Palette.SuccessColor) +
				swatch(t.Palette.ErrorColor)

			var cursorMark string
			var nameStyle lipgloss.Style
			var rowStyle lipgloss.Style
			if selected {
				cursorMark = "▸ "
				nameStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color(t.Palette.AccentColor)).
					Background(selectedBgColor)
				rowStyle = lipgloss.NewStyle().
					Width(innerW).
					Background(selectedBgColor)
			} else {
				cursorMark = "  "
				nameStyle = modalMutedStyle
				rowStyle = lipgloss.NewStyle().
					Width(innerW).
					Background(modalBgColor)
			}

			nameField := nameStyle.Render(fmt.Sprintf("%-20s", t.Name))
			rowLines = append(rowLines, rowStyle.Render(cursorMark+nameField+" "+swatches))
		}
		for len(rowLines) < visibleRows {
			rowLines = append(rowLines, m.renderModalBlank(innerW))
		}
		sections = append(sections, lipgloss.NewStyle().
			Width(innerW).
			Height(visibleRows).
			Background(modalBgColor).
			Render(strings.Join(rowLines, "\n")))
	}

	footer := "j/k or up/down navigate • / or tab filter • enter saves theme.name • esc cancels"
	if m.themePickerInput.Focused() {
		footer = "type to filter • up/down navigate results • tab returns to list • enter saves theme.name • esc cancels"
	}
	sections = append(
		sections,
		m.renderModalBlank(innerW),
		m.renderModalFooter(footer, innerW),
	)

	content := lipgloss.NewStyle().
		Width(innerW).
		Background(modalBgColor).
		Render(lipgloss.JoinVertical(lipgloss.Left, sections...))
	return modalCardStyle(modalW).Render(content)
}
