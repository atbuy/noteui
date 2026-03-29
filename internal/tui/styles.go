package tui

import "github.com/charmbracelet/lipgloss"

var (
	borderColor     = lipgloss.Color("240")
	accentColor     = lipgloss.Color("69")
	accentSoftColor = lipgloss.Color("111")
	mutedColor      = lipgloss.Color("245")
	bgSoftColor     = lipgloss.Color("236")
	textColor       = lipgloss.Color("230")
	errorColor      = lipgloss.Color("204")

	appStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleBarStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor).
			Background(accentColor).
			Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentSoftColor).
			Padding(0, 0, 1, 0)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor)

	metaStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	chipStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(bgSoftColor).
			Padding(0, 1).
			MarginRight(1)

	emptyStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			BorderTop(true).
			BorderForeground(bgSoftColor).
			Padding(0, 1)

	statusOKStyle = lipgloss.NewStyle().
			Foreground(accentSoftColor)

	statusErrStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)
)

func panelStyle(width, height int, focused bool) lipgloss.Style {
	bc := borderColor
	if focused {
		bc = accentColor
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bc).
		Width(max(20, width-2)).
		Height(max(8, height-8)).
		Padding(0, 1)
}
