package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/config"
)

var (
	borderColor      lipgloss.Color
	focusBorderColor lipgloss.Color
	accentColor      lipgloss.Color
	accentSoftColor  lipgloss.Color
	mutedColor       lipgloss.Color
	bgSoftColor      lipgloss.Color
	textColor        lipgloss.Color
	errorColor       lipgloss.Color
	successColor     lipgloss.Color
	bgColor          lipgloss.Color
	subtleColor      lipgloss.Color
	chipBgColor      lipgloss.Color
	selectedBgColor  lipgloss.Color
	selectedFgColor  lipgloss.Color

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

	treeCategoryStyle         lipgloss.Style
	treeNoteStyle             lipgloss.Style
	treePinnedStyle           lipgloss.Style
	treeSelectedCategoryStyle lipgloss.Style
	treeSelectedNoteStyle     lipgloss.Style
)

type themePalette struct {
	BgColor          string
	PanelBgColor     string
	BorderColor      string
	FocusBorderColor string
	AccentColor      string
	AccentSoftColor  string
	TextColor        string
	MutedColor       string
	SubtleColor      string
	ChipBgColor      string
	ErrorColor       string
	SuccessColor     string
	SelectedBgColor  string
	SelectedFgColor  string
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
	override(&p.FocusBorderColor, cfg.Theme.FocusBorderColor)
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
	focusBorderColor = lipgloss.Color(p.FocusBorderColor)
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
		Padding(max(0, appPaddingY), max(0, appPaddingX))

	titleBarStyle = lipgloss.NewStyle().
		Bold(boldTitleBar).
		Foreground(textColor).
		Background(accentColor).
		Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
		Bold(boldPanelTitles).
		Foreground(accentSoftColor).
		Padding(0, 0, 0, 0).
		MarginBottom(1)

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
		Padding(0, 1).
		MarginTop(1)

	statusOKStyle = lipgloss.NewStyle().
		Foreground(successColor)

	statusErrStyle = lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true)

	modalTitleStyle = lipgloss.NewStyle().
		Bold(boldModalTitles).
		Foreground(modalTitleColor).
		Background(modalBgColor)

	modalTextStyle = lipgloss.NewStyle().
		Foreground(modalTextColor).
		Background(modalBgColor)

	modalMutedStyle = lipgloss.NewStyle().
		Foreground(modalMutedColor).
		Background(modalBgColor)

	modalFooterStyle = lipgloss.NewStyle().
		Foreground(modalMutedColor).
		Background(modalBgColor)

	modalKeyStyle = lipgloss.NewStyle().
		Width(14).
		Bold(true).
		Foreground(modalAccentColor).
		Background(modalBgColor)

	treeCategoryStyle = lipgloss.NewStyle().
		Foreground(accentSoftColor)

	treeNoteStyle = lipgloss.NewStyle().
		Foreground(textColor)

	treePinnedStyle = lipgloss.NewStyle().
		Foreground(accentColor)

	treeSelectedCategoryStyle = lipgloss.NewStyle().
		Foreground(selectedFgColor).
		Background(selectedBgColor).
		Bold(boldSelected)

	treeSelectedNoteStyle = lipgloss.NewStyle().
		Foreground(selectedFgColor).
		Background(selectedBgColor).
		Bold(boldSelected)
}

func builtinTheme(name string) themePalette {
	switch normalizeThemeName(name) {
	case "nord":
		return themePalette{
			BgColor:         "#2E3440",
			PanelBgColor:    "#3B4252",
			BorderColor:     "#4C566A",
			AccentColor:     "#88C0D0",
			AccentSoftColor: "#8FBCBB",
			TextColor:       "#ECEFF4",
			MutedColor:      "#D8DEE9",
			SubtleColor:     "#4C566A",
			ChipBgColor:     "#434C5E",
			ErrorColor:      "#BF616A",
			SuccessColor:    "#A3BE8C",
			SelectedBgColor: "#4C566A",
			SelectedFgColor: "#ECEFF4",
		}

	case "gruvbox":
		return themePalette{
			BgColor:         "#282828",
			PanelBgColor:    "#32302F",
			BorderColor:     "#504945",
			AccentColor:     "#D79921",
			AccentSoftColor: "#FABD2F",
			TextColor:       "#EBDBB2",
			MutedColor:      "#A89984",
			SubtleColor:     "#504945",
			ChipBgColor:     "#3C3836",
			ErrorColor:      "#FB4934",
			SuccessColor:    "#B8BB26",
			SelectedBgColor: "#504945",
			SelectedFgColor: "#FBF1C7",
		}

	case "catppuccin", "catppuccin-mocha", "mocha":
		return themePalette{
			BgColor:         "#1E1E2E",
			PanelBgColor:    "#313244",
			BorderColor:     "#45475A",
			AccentColor:     "#CBA6F7",
			AccentSoftColor: "#89B4FA",
			TextColor:       "#CDD6F4",
			MutedColor:      "#A6ADC8",
			SubtleColor:     "#45475A",
			ChipBgColor:     "#45475A",
			ErrorColor:      "#F38BA8",
			SuccessColor:    "#A6E3A1",
			SelectedBgColor: "#585B70",
			SelectedFgColor: "#CDD6F4",
		}

	case "latte":
		return themePalette{
			BgColor:         "#EFF1F5",
			PanelBgColor:    "#E6E9EF",
			BorderColor:     "#BCC0CC",
			AccentColor:     "#8839EF",
			AccentSoftColor: "#1E66F5",
			TextColor:       "#4C4F69",
			MutedColor:      "#6C6F85",
			SubtleColor:     "#BCC0CC",
			ChipBgColor:     "#CCD0DA",
			ErrorColor:      "#D20F39",
			SuccessColor:    "#40A02B",
			SelectedBgColor: "#BCC0CC",
			SelectedFgColor: "#4C4F69",
		}

	case "solarized-light":
		return themePalette{
			BgColor:         "#FDF6E3",
			PanelBgColor:    "#EEE8D5",
			BorderColor:     "#93A1A1",
			AccentColor:     "#B58900",
			AccentSoftColor: "#268BD2",
			TextColor:       "#586E75",
			MutedColor:      "#657B83",
			SubtleColor:     "#93A1A1",
			ChipBgColor:     "#E4DDC8",
			ErrorColor:      "#DC322F",
			SuccessColor:    "#859900",
			SelectedBgColor: "#D3CBB7",
			SelectedFgColor: "#586E75",
		}

	case "paper":
		return themePalette{
			BgColor:         "#FAFAF7",
			PanelBgColor:    "#F0EFEA",
			BorderColor:     "#C8C6BE",
			AccentColor:     "#B65C2A",
			AccentSoftColor: "#4A7BD0",
			TextColor:       "#2F3440",
			MutedColor:      "#667085",
			SubtleColor:     "#C8C6BE",
			ChipBgColor:     "#E6E3DB",
			ErrorColor:      "#C0392B",
			SuccessColor:    "#2E7D32",
			SelectedBgColor: "#DDD9CF",
			SelectedFgColor: "#2F3440",
		}

	case "onedark":
		return themePalette{
			BgColor:          "#1E222A",
			PanelBgColor:     "#252B34",
			BorderColor:      "#353B45",
			FocusBorderColor: "#C4B28A",
			AccentColor:      "#98C379",
			AccentSoftColor:  "#E5C07B",
			TextColor:        "#ABB2BF",
			MutedColor:       "#7F848E",
			SubtleColor:      "#353B45",
			ChipBgColor:      "#2C313A",
			ErrorColor:       "#E06C75",
			SuccessColor:     "#98C379",
			SelectedBgColor:  "#353B45",
			SelectedFgColor:  "#E6EAF2",
		}

	case "kanagawa":
		return themePalette{
			BgColor:          "#1F1F28",
			PanelBgColor:     "#2A2A37",
			BorderColor:      "#54546D",
			FocusBorderColor: "#C4B28A",
			AccentColor:      "#C4B28A",
			AccentSoftColor:  "#7E9CD8",
			TextColor:        "#DCD7BA",
			MutedColor:       "#C8C093",
			SubtleColor:      "#54546D",
			ChipBgColor:      "#363646",
			ErrorColor:       "#E46876",
			SuccessColor:     "#98BB6C",
			SelectedBgColor:  "#2D4F67",
			SelectedFgColor:  "#DCD7BA",
		}

	case "dracula":
		return themePalette{
			BgColor:          "#282A36",
			PanelBgColor:     "#21222C",
			BorderColor:      "#44475A",
			FocusBorderColor: "#A7C080",
			AccentColor:      "#FF79C6",
			AccentSoftColor:  "#BD93F9",
			TextColor:        "#F8F8F2",
			MutedColor:       "#B6B6B2",
			SubtleColor:      "#44475A",
			ChipBgColor:      "#343746",
			ErrorColor:       "#FF5555",
			SuccessColor:     "#50FA7B",
			SelectedBgColor:  "#44475A",
			SelectedFgColor:  "#F8F8F2",
		}

	case "everforest", "everforest-dark":
		return themePalette{
			BgColor:          "#2B3339",
			PanelBgColor:     "#323C41",
			BorderColor:      "#4F5B58",
			FocusBorderColor: "#A7C080",
			AccentColor:      "#A7C080",
			AccentSoftColor:  "#DBBC7F",
			TextColor:        "#D3C6AA",
			MutedColor:       "#9DA9A0",
			SubtleColor:      "#4F5B58",
			ChipBgColor:      "#374247",
			ErrorColor:       "#E67E80",
			SuccessColor:     "#A7C080",
			SelectedBgColor:  "#425047",
			SelectedFgColor:  "#D3C6AA",
		}

	case "tokyo-night-storm", "tokyonight-storm", "tokyo night storm":
		return themePalette{
			BgColor:          "#24283B",
			PanelBgColor:     "#1F2335",
			BorderColor:      "#414868",
			FocusBorderColor: "#7AA2F7",
			AccentColor:      "#7AA2F7",
			AccentSoftColor:  "#7DCFFF",
			TextColor:        "#C0CAF5",
			MutedColor:       "#A9B1D6",
			SubtleColor:      "#414868",
			ChipBgColor:      "#2A3050",
			ErrorColor:       "#F7768E",
			SuccessColor:     "#9ECE6A",
			SelectedBgColor:  "#364A82",
			SelectedFgColor:  "#C0CAF5",
		}

	case "github-light":
		return themePalette{
			BgColor:          "#FFFFFF",
			PanelBgColor:     "#F6F8FA",
			BorderColor:      "#D0D7DE",
			FocusBorderColor: "#0969DA",
			AccentColor:      "#0969DA",
			AccentSoftColor:  "#1F6FEB",
			TextColor:        "#24292F",
			MutedColor:       "#57606A",
			SubtleColor:      "#D0D7DE",
			ChipBgColor:      "#EAEEF2",
			ErrorColor:       "#CF222E",
			SuccessColor:     "#1A7F37",
			SelectedBgColor:  "#DDF4FF",
			SelectedFgColor:  "#24292F",
		}

	case "github-dark":
		return themePalette{
			BgColor:          "#0D1117",
			PanelBgColor:     "#161B22",
			BorderColor:      "#30363D",
			FocusBorderColor: "#58A6FF",
			AccentColor:      "#8B949E",
			AccentSoftColor:  "#58A6FF",
			TextColor:        "#C9D1D9",
			MutedColor:       "#8B949E",
			SubtleColor:      "#30363D",
			ChipBgColor:      "#21262D",
			ErrorColor:       "#F85149",
			SuccessColor:     "#3FB950",
			SelectedBgColor:  "#21262D",
			SelectedFgColor:  "#F0F6FC",
		}

	case "carbonfox":
		return themePalette{
			BgColor:          "#161616",
			PanelBgColor:     "#202020",
			BorderColor:      "#3A3A3A",
			FocusBorderColor: "#78A9FF",
			AccentColor:      "#A0A0A0",
			AccentSoftColor:  "#78A9FF",
			TextColor:        "#F2F4F8",
			MutedColor:       "#B0B0B0",
			SubtleColor:      "#3A3A3A",
			ChipBgColor:      "#262626",
			ErrorColor:       "#FF8389",
			SuccessColor:     "#42BE65",
			SelectedBgColor:  "#2B2B2B",
			SelectedFgColor:  "#F2F4F8",
		}

	default:
		return themePalette{
			BgColor:          "#1E1E1E",
			PanelBgColor:     "#2A2A2A",
			BorderColor:      "#5F5F5F",
			FocusBorderColor: "#5F87D7",
			AccentColor:      "#5F87D7",
			AccentSoftColor:  "#87AFDF",
			TextColor:        "#E5E5E5",
			MutedColor:       "#A8A8A8",
			SubtleColor:      "#444444",
			ChipBgColor:      "#3A3A3A",
			ErrorColor:       "#D75F5F",
			SuccessColor:     "#87AF87",
			SelectedBgColor:  "#5F87D7",
			SelectedFgColor:  "#FFFFFF",
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
		bc = focusBorderColor
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
