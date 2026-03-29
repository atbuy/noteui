package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	borderColor = lipgloss.Color("240")

	headerStyle = lipgloss.NewStyle().Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
)

func panelStyle(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(max(8, height-2)).
		Padding(0, 1)
}
