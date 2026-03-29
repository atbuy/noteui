package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/config"
)

var (
	borderColor     lipgloss.Color
	accentColor     lipgloss.Color
	accentSoftColor lipgloss.Color
	mutedColor      lipgloss.Color
	bgSoftColor     lipgloss.Color
	textColor       lipgloss.Color
	errorColor      lipgloss.Color
	successColor    lipgloss.Color
	bgColor         lipgloss.Color
	subtleColor     lipgloss.Color
	chipBgColor     lipgloss.Color
	selectedBgColor lipgloss.Color
	selectedFgColor lipgloss.Color

	appStyle        lipgloss.Style
	titleBarStyle   lipgloss.Style
	panelTitleStyle lipgloss.Style
	headerStyle     lipgloss.Style
	metaStyle       lipgloss.Style
	mutedStyle      lipgloss.Style
	chipStyle       lipgloss.Style
	emptyStyle      lipgloss.Style
	footerStyle     lipgloss.Style
	statusOKStyle   lipgloss.Style
	statusErrStyle  lipgloss.Style

	currentBorder lipgloss.Border
	panelPaddingX int
	panelPaddingY int
)

type themePalette struct {
	BgColor         string
	PanelBgColor    string
	BorderColor     string
	AccentColor     string
	AccentSoftColor string
	TextColor       string
	MutedColor      string
	SubtleColor     string
	ChipBgColor     string
	ErrorColor      string
	SuccessColor    string
	SelectedBgColor string
	SelectedFgColor string
}

func ApplyTheme(cfg config.Config) {
	p := builtinTheme(cfg.Theme.Name)

	override := func(current *string, value string) {
		if strings.TrimSpace(value) != "" {
			*current = value
		}
	}

	override(&p.BgColor, cfg.Theme.BgColor)
	override(&p.PanelBgColor, cfg.Theme.PanelBgColor)
	override(&p.BorderColor, cfg.Theme.BorderColor)
	override(&p.AccentColor, cfg.Theme.AccentColor)
	override(&p.AccentSoftColor, cfg.Theme.AccentSoftColor)
	override(&p.TextColor, cfg.Theme.TextColor)
	override(&p.MutedColor, cfg.Theme.MutedColor)
	override(&p.SubtleColor, cfg.Theme.SubtleColor)
	override(&p.ChipBgColor, cfg.Theme.ChipBgColor)
	override(&p.ErrorColor, cfg.Theme.ErrorColor)
	override(&p.SuccessColor, cfg.Theme.SuccessColor)
	override(&p.SelectedBgColor, cfg.Theme.SelectedBgColor)
	override(&p.SelectedFgColor, cfg.Theme.SelectedFgColor)

	bgColor = lipgloss.Color(p.BgColor)
	bgSoftColor = lipgloss.Color(p.PanelBgColor)
	borderColor = lipgloss.Color(p.BorderColor)
	accentColor = lipgloss.Color(p.AccentColor)
	accentSoftColor = lipgloss.Color(p.AccentSoftColor)
	textColor = lipgloss.Color(p.TextColor)
	mutedColor = lipgloss.Color(p.MutedColor)
	subtleColor = lipgloss.Color(p.SubtleColor)
	chipBgColor = lipgloss.Color(p.ChipBgColor)
	errorColor = lipgloss.Color(p.ErrorColor)
	successColor = lipgloss.Color(p.SuccessColor)
	selectedBgColor = lipgloss.Color(p.SelectedBgColor)
	selectedFgColor = lipgloss.Color(p.SelectedFgColor)

	panelPaddingX = max(0, cfg.Theme.PanelPaddingX)
	panelPaddingY = max(0, cfg.Theme.PanelPaddingY)
	appPaddingX := max(0, cfg.Theme.AppPaddingX)
	appPaddingY := max(0, cfg.Theme.AppPaddingY)

	currentBorder = borderFromName(cfg.Theme.BorderStyle)

	appStyle = lipgloss.NewStyle().
		Padding(appPaddingY, appPaddingX)

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
		Background(chipBgColor).
		Padding(0, 1).
		MarginRight(1)

	emptyStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true)

	footerStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		BorderTop(true).
		BorderForeground(subtleColor).
		Padding(0, 1)

	statusOKStyle = lipgloss.NewStyle().
		Foreground(successColor)

	statusErrStyle = lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true)
}

func builtinTheme(name string) themePalette {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "nord":
		return themePalette{
			BgColor:         "#2E3440",
			PanelBgColor:    "#3B4252",
			BorderColor:     "#4C566A",
			AccentColor:     "#5E81AC",
			AccentSoftColor: "#88C0D0",
			TextColor:       "#ECEFF4",
			MutedColor:      "#D8DEE9",
			SubtleColor:     "#4C566A",
			ChipBgColor:     "#434C5E",
			ErrorColor:      "#BF616A",
			SuccessColor:    "#A3BE8C",
			SelectedBgColor: "#5E81AC",
			SelectedFgColor: "#ECEFF4",
		}
	case "gruvbox":
		return themePalette{
			BgColor:         "#282828",
			PanelBgColor:    "#3C3836",
			BorderColor:     "#504945",
			AccentColor:     "#458588",
			AccentSoftColor: "#83A598",
			TextColor:       "#EBDBB2",
			MutedColor:      "#A89984",
			SubtleColor:     "#504945",
			ChipBgColor:     "#504945",
			ErrorColor:      "#FB4934",
			SuccessColor:    "#B8BB26",
			SelectedBgColor: "#458588",
			SelectedFgColor: "#FBF1C7",
		}
	case "catppuccin":
		return themePalette{
			BgColor:         "#1E1E2E",
			PanelBgColor:    "#313244",
			BorderColor:     "#45475A",
			AccentColor:     "#89B4FA",
			AccentSoftColor: "#B4BEFE",
			TextColor:       "#CDD6F4",
			MutedColor:      "#A6ADC8",
			SubtleColor:     "#45475A",
			ChipBgColor:     "#45475A",
			ErrorColor:      "#F38BA8",
			SuccessColor:    "#A6E3A1",
			SelectedBgColor: "#89B4FA",
			SelectedFgColor: "#11111B",
		}
	default:
		return themePalette{
			BgColor:         "#1E1E1E",
			PanelBgColor:    "#2A2A2A",
			BorderColor:     "#5F5F5F",
			AccentColor:     "#5F87D7",
			AccentSoftColor: "#87AFDF",
			TextColor:       "#E5E5E5",
			MutedColor:      "#A8A8A8",
			SubtleColor:     "#444444",
			ChipBgColor:     "#3A3A3A",
			ErrorColor:      "#D75F5F",
			SuccessColor:    "#87AF87",
			SelectedBgColor: "#5F87D7",
			SelectedFgColor: "#FFFFFF",
		}
	}
}

func borderFromName(name string) lipgloss.Border {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "normal":
		return lipgloss.NormalBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "thick":
		return lipgloss.ThickBorder()
	case "hidden":
		return lipgloss.HiddenBorder()
	default:
		return lipgloss.RoundedBorder()
	}
}

func init() {
	ApplyTheme(config.Default())
}

func panelStyle(width, height int, focused bool) lipgloss.Style {
	bc := borderColor
	if focused {
		bc = accentColor
	}

	return lipgloss.NewStyle().
		Border(currentBorder).
		BorderForeground(bc).
		Width(max(20, width-2)).
		Height(max(8, height-8)).
		Padding(panelPaddingY, panelPaddingX)
}
