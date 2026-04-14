package tui

import (
	"fmt"
	"strings"

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
	m.themePickerCursor = cursor
	m.themePickerOrigTheme = current
	m.themePickerScrollOffset = themePickerCenteredOffset(cursor, len(themes))
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

// moveThemePickerCursor shifts the cursor by delta (wrapping) and immediately
// applies the hovered theme for a live preview, including the preview pane.
func (m *Model) moveThemePickerCursor(delta int) {
	themes := BuiltinThemes()
	n := len(themes)
	m.themePickerCursor = (m.themePickerCursor + delta + n) % n

	// Keep cursor visible in the scroll window.
	if m.themePickerCursor >= m.themePickerScrollOffset+themePickerVisible {
		m.themePickerScrollOffset = m.themePickerCursor - themePickerVisible + 1
	}
	if m.themePickerCursor < m.themePickerScrollOffset {
		m.themePickerScrollOffset = m.themePickerCursor
	}

	// Apply the hovered theme globally so the base view re-renders with it.
	hovCfg := m.cfg
	hovCfg.Theme.Name = themes[m.themePickerCursor].Name
	ApplyTheme(hovCfg)

	// Clear the cached preview path so refreshPreview re-renders the preview
	// pane content with the new theme's colours.
	m.previewPath = ""
	m.refreshPreview()
}

// confirmThemePicker saves the selected theme to config and closes the modal.
func (m *Model) confirmThemePicker() {
	themes := BuiltinThemes()
	name := themes[m.themePickerCursor].Name
	m.cfg.Theme.Name = name
	m.showThemePicker = false
	if _, _, err := config.SaveTheme(name); err != nil {
		m.status = "could not save theme: " + err.Error()
	} else {
		m.status = "theme: " + name
	}
}

// cancelThemePicker closes the modal without saving and restores the original theme.
func (m *Model) cancelThemePicker() {
	m.showThemePicker = false
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
	themes := BuiltinThemes()
	hov := themes[m.themePickerCursor]

	modalW := max(60, min(70, m.width-10))
	innerW := modalW - 2*modalPaddingX

	// Header: hovered theme name + description.
	hovName := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(hov.Palette.AccentColor)).
		Background(modalBgColor).
		Render(hov.Name)
	desc := modalMutedStyle.Render(hov.Description)
	header := hovName + modalMutedStyle.Render(" - ") + desc

	// Scroll indicator in the title.
	scrollLabel := fmt.Sprintf(" (%d/%d)", m.themePickerCursor+1, len(themes))
	title := modalTitleStyle.Bold(true).Render("Theme Picker") +
		modalMutedStyle.Render(scrollLabel)

	// Theme list rows.
	end := min(m.themePickerScrollOffset+themePickerVisible, len(themes))
	rowLines := make([]string, 0, end-m.themePickerScrollOffset)
	for i := m.themePickerScrollOffset; i < end; i++ {
		t := themes[i]
		selected := i == m.themePickerCursor

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

	footer := modalFooterStyle.Render("j/k: navigate   enter: apply   esc: cancel")

	body := strings.Join([]string{
		title,
		"",
		header,
		"",
		strings.Join(rowLines, "\n"),
		"",
		footer,
	}, "\n")

	return modalCardStyle(modalW).Render(body)
}
