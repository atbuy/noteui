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

	modalBgColor     lipgloss.Color
	modalBorderColor lipgloss.Color
	modalTitleColor  lipgloss.Color
	modalTextColor   lipgloss.Color
	modalMutedColor  lipgloss.Color
	modalAccentColor lipgloss.Color

	appStyle         lipgloss.Style
	titleBarStyle    lipgloss.Style
	panelTitleStyle  lipgloss.Style
	headerStyle      lipgloss.Style
	metaStyle        lipgloss.Style
	mutedStyle       lipgloss.Style
	chipStyle        lipgloss.Style
	emptyStyle       lipgloss.Style
	footerStyle      lipgloss.Style
	statusOKStyle    lipgloss.Style
	statusErrStyle   lipgloss.Style
	modalTitleStyle  lipgloss.Style
	modalTextStyle   lipgloss.Style
	modalMutedStyle  lipgloss.Style
	modalFooterStyle lipgloss.Style
	modalKeyStyle    lipgloss.Style

	currentBorder lipgloss.Border
	modalBorder   lipgloss.Border

	panelPaddingX int
	panelPaddingY int
	modalPaddingX int
	modalPaddingY int

	boldTitleBar    bool
	boldPanelTitles bool
	boldHeaders     bool
	boldSelected    bool
	boldModalTitles bool

	iconCategoryExpanded  string
	iconCategoryCollapsed string
	iconCategoryLeaf      string
	iconNote              string
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

	modalBgColor = firstNonEmptyColor(cfg.Modal.BgColor, p.PanelBgColor)
	modalBorderColor = firstNonEmptyColor(cfg.Modal.BorderColor, p.AccentColor)
	modalTitleColor = firstNonEmptyColor(cfg.Modal.TitleColor, p.TextColor)
	modalTextColor = firstNonEmptyColor(cfg.Modal.TextColor, p.TextColor)
	modalMutedColor = firstNonEmptyColor(cfg.Modal.MutedColor, p.MutedColor)
	modalAccentColor = firstNonEmptyColor(cfg.Modal.AccentColor, p.AccentSoftColor)

	panelPaddingX = max(0, cfg.Theme.PanelPaddingX)
	panelPaddingY = max(0, cfg.Theme.PanelPaddingY)
	appPaddingX := max(0, cfg.Theme.AppPaddingX)
	appPaddingY := max(0, cfg.Theme.AppPaddingY)

	modalPaddingX = max(0, cfg.Modal.PaddingX)
	modalPaddingY = max(0, cfg.Modal.PaddingY)

	boldTitleBar = cfg.Typography.BoldTitleBar
	boldPanelTitles = cfg.Typography.BoldPanelTitles
	boldHeaders = cfg.Typography.BoldHeaders
	boldSelected = cfg.Typography.BoldSelected
	boldModalTitles = cfg.Typography.BoldModalTitles

	iconCategoryExpanded = firstNonEmpty(cfg.Icons.CategoryExpanded, "▾")
	iconCategoryCollapsed = firstNonEmpty(cfg.Icons.CategoryCollapsed, "▸")
	iconCategoryLeaf = firstNonEmpty(cfg.Icons.CategoryLeaf, "•")
	iconNote = firstNonEmpty(cfg.Icons.Note, "·")

	currentBorder = borderFromName(cfg.Theme.BorderStyle)
	modalBorder = borderFromName(cfg.Modal.BorderStyle)

	appStyle = lipgloss.NewStyle().
		Padding(appPaddingY, appPaddingX)

	titleBarStyle = lipgloss.NewStyle().
		Bold(boldTitleBar).
		Foreground(textColor).
		Background(accentColor).
		Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
		Bold(boldPanelTitles).
		Foreground(accentSoftColor).
		Padding(0, 0, 1, 0)

	headerStyle = lipgloss.NewStyle().
		Bold(boldHeaders).
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

	modalTitleStyle = lipgloss.NewStyle().
		Bold(boldModalTitles).
		Foreground(modalTitleColor)

	modalTextStyle = lipgloss.NewStyle().
		Foreground(modalTextColor)

	modalMutedStyle = lipgloss.NewStyle().
		Foreground(modalMutedColor)

	modalFooterStyle = lipgloss.NewStyle().
		Foreground(modalMutedColor)

	modalKeyStyle = lipgloss.NewStyle().
		Width(14).
		Bold(true).
		Foreground(modalAccentColor)
}

func builtinTheme(name string) themePalette {
	switch normalizeThemeName(name) {
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
	case "catppuccin", "catppuccin-mocha", "mocha":
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
	case "catppuccin-latte", "latte":
		return themePalette{
			BgColor:         "#EFF1F5",
			PanelBgColor:    "#E6E9EF",
			BorderColor:     "#BCC0CC",
			AccentColor:     "#1E66F5",
			AccentSoftColor: "#7287FD",
			TextColor:       "#4C4F69",
			MutedColor:      "#6C6F85",
			SubtleColor:     "#BCC0CC",
			ChipBgColor:     "#CCD0DA",
			ErrorColor:      "#D20F39",
			SuccessColor:    "#40A02B",
			SelectedBgColor: "#1E66F5",
			SelectedFgColor: "#EFF1F5",
		}
	case "solarized-light":
		return themePalette{
			BgColor:         "#FDF6E3",
			PanelBgColor:    "#EEE8D5",
			BorderColor:     "#93A1A1",
			AccentColor:     "#268BD2",
			AccentSoftColor: "#2AA198",
			TextColor:       "#586E75",
			MutedColor:      "#657B83",
			SubtleColor:     "#93A1A1",
			ChipBgColor:     "#E4DDC8",
			ErrorColor:      "#DC322F",
			SuccessColor:    "#859900",
			SelectedBgColor: "#268BD2",
			SelectedFgColor: "#FDF6E3",
		}
	case "paper":
		return themePalette{
			BgColor:         "#FAFAF7",
			PanelBgColor:    "#F0EFEA",
			BorderColor:     "#C8C6BE",
			AccentColor:     "#4A7BD0",
			AccentSoftColor: "#6B93E5",
			TextColor:       "#2F3440",
			MutedColor:      "#667085",
			SubtleColor:     "#C8C6BE",
			ChipBgColor:     "#E6E3DB",
			ErrorColor:      "#C0392B",
			SuccessColor:    "#2E7D32",
			SelectedBgColor: "#4A7BD0",
			SelectedFgColor: "#FFFFFF",
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

func normalizeThemeName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "catppuccin-mocha", "mocha":
		return "catppuccin"
	case "catppuccin-latte":
		return "latte"
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}

func firstNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func firstNonEmptyColor(v, fallback string) lipgloss.Color {
	return lipgloss.Color(firstNonEmpty(v, fallback))
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

func modalCardStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Border(modalBorder).
		BorderForeground(modalBorderColor).
		Background(modalBgColor).
		Padding(modalPaddingY, modalPaddingX)
}
