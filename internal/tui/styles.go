package tui

import (
	"fmt"
	"math"
	"strconv"
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
	highlightBgColor lipgloss.Color

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
	HighlightBgColor string
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
	override(&p.HighlightBgColor, cfg.Theme.HighlightBgColor)

	p = normalizePaletteAccessibility(p)

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
	highlightBgColor = lipgloss.Color(p.HighlightBgColor)

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
		Background(bgColor).
		Padding(max(0, appPaddingY), max(0, appPaddingX))

	titleBarStyle = lipgloss.NewStyle().
		Bold(boldTitleBar).
		Foreground(textColor).
		Background(accentColor).
		Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
		Bold(boldPanelTitles).
		Foreground(accentSoftColor).
		Background(bgSoftColor).
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
		Background(bgSoftColor).
		Italic(true)

	footerStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(bgColor).
		BorderTop(true).
		BorderForeground(subtleColor).
		BorderBackground(bgColor).
		Padding(0, 1)

	statusOKStyle = lipgloss.NewStyle().
		Foreground(successColor).
		Background(bgColor)

	statusErrStyle = lipgloss.NewStyle().
		Foreground(errorColor).
		Background(bgColor).
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
		Foreground(accentSoftColor).
		Background(bgSoftColor)

	treeNoteStyle = lipgloss.NewStyle().
		Foreground(textColor).
		Background(bgSoftColor)

	treePinnedStyle = lipgloss.NewStyle().
		Foreground(accentColor).
		Background(bgSoftColor)

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
			BgColor:          "#2E3440",
			PanelBgColor:     "#3B4252",
			BorderColor:      "#434C5E",
			FocusBorderColor: "#88C0D0",
			AccentColor:      "#88C0D0",
			AccentSoftColor:  "#97B1CC",
			TextColor:        "#ECEFF4",
			MutedColor:       "#D8DEE9",
			SubtleColor:      "#4C566A",
			ChipBgColor:      "#434C5E",
			ErrorColor:       "#D9A2A7",
			SuccessColor:     "#A3BE8C",
			SelectedBgColor:  "#4C566A",
			SelectedFgColor:  "#ECEFF4",
			HighlightBgColor: "#3B4D6A",
		}

	case "gruvbox":
		return themePalette{
			BgColor:          "#282828",
			PanelBgColor:     "#32302F",
			BorderColor:      "#504945",
			FocusBorderColor: "#FABD2F",
			AccentColor:      "#FABD2F",
			AccentSoftColor:  "#83A598",
			TextColor:        "#EBDBB2",
			MutedColor:       "#A89984",
			SubtleColor:      "#665C54",
			ChipBgColor:      "#3C3836",
			ErrorColor:       "#FC6755",
			SuccessColor:     "#B8BB26",
			SelectedBgColor:  "#504945",
			SelectedFgColor:  "#FBF1C7",
			HighlightBgColor: "#504528",
		}

	case "catppuccin", "catppuccin-mocha", "mocha":
		return themePalette{
			BgColor:          "#1E1E2E",
			PanelBgColor:     "#181825",
			BorderColor:      "#313244",
			FocusBorderColor: "#CBA6F7",
			AccentColor:      "#CBA6F7",
			AccentSoftColor:  "#89B4FA",
			TextColor:        "#CDD6F4",
			MutedColor:       "#BAC2DE",
			SubtleColor:      "#6C7086",
			ChipBgColor:      "#313244",
			ErrorColor:       "#F38BA8",
			SuccessColor:     "#A6E3A1",
			SelectedBgColor:  "#45475A",
			SelectedFgColor:  "#CDD6F4",
			HighlightBgColor: "#3D3560",
		}

	case "latte":
		return themePalette{
			BgColor:          "#EFF1F5",
			PanelBgColor:     "#E6E9EF",
			BorderColor:      "#CCD0DA",
			FocusBorderColor: "#8839EF",
			AccentColor:      "#8839EF",
			AccentSoftColor:  "#1C5FE5",
			TextColor:        "#484B64",
			MutedColor:       "#65687D",
			SubtleColor:      "#BCC0CC",
			ChipBgColor:      "#CCD0DA",
			ErrorColor:       "#CC1038",
			SuccessColor:     "#307820",
			SelectedBgColor:  "#BCC0CC",
			SelectedFgColor:  "#4B4D67",
			HighlightBgColor: "#C5BDFF",
		}

	case "solarized-light":
		return themePalette{
			BgColor:          "#FDF6E3",
			PanelBgColor:     "#EEE8D5",
			BorderColor:      "#93A1A1",
			FocusBorderColor: "#B58900",
			AccentColor:      "#8F6C00",
			AccentSoftColor:  "#1E6DA5",
			TextColor:        "#3E4E53",
			MutedColor:       "#586B72",
			SubtleColor:      "#93A1A1",
			ChipBgColor:      "#E4DDC8",
			ErrorColor:       "#C22C29",
			SuccessColor:     "#5F6F00",
			SelectedBgColor:  "#D3CBB7",
			SelectedFgColor:  "#47595E",
			HighlightBgColor: "#C9D8E8",
		}

	case "paper":
		return themePalette{
			BgColor:          "#FAFAF7",
			PanelBgColor:     "#F2F1EC",
			BorderColor:      "#D8D5CC",
			FocusBorderColor: "#4A7BD0",
			AccentColor:      "#4472C0",
			AccentSoftColor:  "#AA5526",
			TextColor:        "#2F3440",
			MutedColor:       "#656E83",
			SubtleColor:      "#C8C6BE",
			ChipBgColor:      "#E8E5DC",
			ErrorColor:       "#C0392B",
			SuccessColor:     "#2E7D32",
			SelectedBgColor:  "#DDD9CF",
			SelectedFgColor:  "#2F3440",
			HighlightBgColor: "#D6E4F7",
		}

	case "onedark":
		return themePalette{
			BgColor:          "#1E2127",
			PanelBgColor:     "#282C34",
			BorderColor:      "#3A404C",
			FocusBorderColor: "#61AFEF",
			AccentColor:      "#61AFEF",
			AccentSoftColor:  "#C678DD",
			TextColor:        "#ABB2BF",
			MutedColor:       "#828997",
			SubtleColor:      "#3A404C",
			ChipBgColor:      "#2C313A",
			ErrorColor:       "#E06C75",
			SuccessColor:     "#98C379",
			SelectedBgColor:  "#353B45",
			SelectedFgColor:  "#E6EAF2",
			HighlightBgColor: "#2B4A63",
		}

	case "kanagawa":
		return themePalette{
			BgColor:          "#1F1F28",
			PanelBgColor:     "#2A2A37",
			BorderColor:      "#363646",
			FocusBorderColor: "#7E9CD8",
			AccentColor:      "#7E9CD8",
			AccentSoftColor:  "#DCA561",
			TextColor:        "#DCD7BA",
			MutedColor:       "#C8C093",
			SubtleColor:      "#54546D",
			ChipBgColor:      "#2D2D3A",
			ErrorColor:       "#E56D7B",
			SuccessColor:     "#98BB6C",
			SelectedBgColor:  "#2D4F67",
			SelectedFgColor:  "#DCD7BA",
			HighlightBgColor: "#2D4F67",
		}

	case "dracula":
		return themePalette{
			BgColor:          "#282A36",
			PanelBgColor:     "#21222C",
			BorderColor:      "#44475A",
			FocusBorderColor: "#FF79C6",
			AccentColor:      "#FF79C6",
			AccentSoftColor:  "#BD93F9",
			TextColor:        "#F8F8F2",
			MutedColor:       "#7A88B2",
			SubtleColor:      "#44475A",
			ChipBgColor:      "#343746",
			ErrorColor:       "#FF5555",
			SuccessColor:     "#50FA7B",
			SelectedBgColor:  "#44475A",
			SelectedFgColor:  "#F8F8F2",
			HighlightBgColor: "#44366A",
		}

	case "everforest", "everforest-dark":
		return themePalette{
			BgColor:          "#2D353B",
			PanelBgColor:     "#343F44",
			BorderColor:      "#475258",
			FocusBorderColor: "#A7C080",
			AccentColor:      "#A7C080",
			AccentSoftColor:  "#DBBC7F",
			TextColor:        "#DACFB7",
			MutedColor:       "#9FAAA2",
			SubtleColor:      "#475258",
			ChipBgColor:      "#374145",
			ErrorColor:       "#E99092",
			SuccessColor:     "#A7C080",
			SelectedBgColor:  "#425047",
			SelectedFgColor:  "#D3C6AA",
			HighlightBgColor: "#3A5247",
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
			SubtleColor:      "#565F89",
			ChipBgColor:      "#292E42",
			ErrorColor:       "#F7768E",
			SuccessColor:     "#9ECE6A",
			SelectedBgColor:  "#364A82",
			SelectedFgColor:  "#C0CAF5",
			HighlightBgColor: "#2A3F6A",
		}

	case "github-light":
		return themePalette{
			BgColor:          "#FFFFFF",
			PanelBgColor:     "#F6F8FA",
			BorderColor:      "#D0D7DE",
			FocusBorderColor: "#0550AE",
			AccentColor:      "#0550AE",
			AccentSoftColor:  "#0349A5",
			TextColor:        "#24292F",
			MutedColor:       "#57606A",
			SubtleColor:      "#AFB8C1",
			ChipBgColor:      "#EAEEF2",
			ErrorColor:       "#A40E26",
			SuccessColor:     "#116329",
			SelectedBgColor:  "#DDF4FF",
			SelectedFgColor:  "#0E1116",
			HighlightBgColor: "#B6E3FF",
		}

	case "github-dark":
		return themePalette{
			BgColor:          "#0D1117",
			PanelBgColor:     "#161B22",
			BorderColor:      "#30363D",
			FocusBorderColor: "#58A6FF",
			AccentColor:      "#58A6FF",
			AccentSoftColor:  "#79C0FF",
			TextColor:        "#C9D1D9",
			MutedColor:       "#8B949E",
			SubtleColor:      "#30363D",
			ChipBgColor:      "#21262D",
			ErrorColor:       "#F85149",
			SuccessColor:     "#3FB950",
			SelectedBgColor:  "#1C2B45",
			SelectedFgColor:  "#F0F6FC",
			HighlightBgColor: "#1C3A5C",
		}

	case "carbonfox":
		return themePalette{
			BgColor:          "#161616",
			PanelBgColor:     "#202020",
			BorderColor:      "#3A3A3A",
			FocusBorderColor: "#78A9FF",
			AccentColor:      "#78A9FF",
			AccentSoftColor:  "#08BDBA",
			TextColor:        "#F2F4F8",
			MutedColor:       "#B0B0B0",
			SubtleColor:      "#3A3A3A",
			ChipBgColor:      "#262626",
			ErrorColor:       "#FF8389",
			SuccessColor:     "#42BE65",
			SelectedBgColor:  "#2B2B2B",
			SelectedFgColor:  "#F2F4F8",
			HighlightBgColor: "#1E3A5C",
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
			SelectedBgColor:  "#3F5F9F",
			SelectedFgColor:  "#FFFFFF",
			HighlightBgColor: "#3A3A6A",
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

func normalizePaletteAccessibility(p themePalette) themePalette {
	p.TextColor = ensureContrast(p.TextColor, p.PanelBgColor, 7.0)
	p.MutedColor = ensureContrast(p.MutedColor, p.PanelBgColor, 4.5)
	p.AccentSoftColor = ensureContrast(p.AccentSoftColor, p.PanelBgColor, 4.5)
	p.AccentColor = ensureContrast(p.AccentColor, p.BgColor, 4.5)
	p.SelectedFgColor = ensureContrast(p.SelectedFgColor, p.SelectedBgColor, 4.5)
	return p
}

func ensureContrast(fgHex, bgHex string, minRatio float64) string {
	fg, ok := parseHexColor(fgHex)
	if !ok {
		return fgHex
	}
	bg, ok := parseHexColor(bgHex)
	if !ok {
		return fgHex
	}
	if contrastRatio(fg, bg) >= minRatio {
		return formatHexColor(fg)
	}

	black := rgbColor{0, 0, 0}
	white := rgbColor{255, 255, 255}

	best := fg
	bestDist := math.MaxFloat64
	for _, target := range []rgbColor{black, white} {
		adjusted, ok := blendForContrast(fg, bg, target, minRatio)
		if !ok {
			continue
		}
		dist := colorDistanceSq(fg, adjusted)
		if dist < bestDist {
			best = adjusted
			bestDist = dist
		}
	}

	if bestDist == math.MaxFloat64 {
		if contrastRatio(black, bg) >= contrastRatio(white, bg) {
			best = black
		} else {
			best = white
		}
	}

	return formatHexColor(best)
}

type rgbColor struct {
	r float64
	g float64
	b float64
}

func parseHexColor(hex string) (rgbColor, bool) {
	hex = strings.TrimSpace(strings.TrimPrefix(hex, "#"))
	if len(hex) != 6 {
		return rgbColor{}, false
	}
	value, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return rgbColor{}, false
	}
	return rgbColor{
		r: float64((value >> 16) & 0xFF),
		g: float64((value >> 8) & 0xFF),
		b: float64(value & 0xFF),
	}, true
}

func formatHexColor(c rgbColor) string {
	return fmt.Sprintf("#%02X%02X%02X", clampChannel(c.r), clampChannel(c.g), clampChannel(c.b))
}

func clampChannel(v float64) int {
	return max(0, min(255, int(math.Round(v))))
}

func blendForContrast(start, bg, target rgbColor, minRatio float64) (rgbColor, bool) {
	lo := 0.0
	hi := 1.0
	best := start
	found := false
	for range 24 {
		mid := (lo + hi) / 2
		candidate := mixColor(start, target, mid)
		if contrastRatio(candidate, bg) >= minRatio {
			best = candidate
			found = true
			hi = mid
		} else {
			lo = mid
		}
	}
	return best, found
}

func mixColor(a, b rgbColor, t float64) rgbColor {
	return rgbColor{
		r: a.r + (b.r-a.r)*t,
		g: a.g + (b.g-a.g)*t,
		b: a.b + (b.b-a.b)*t,
	}
}

func colorDistanceSq(a, b rgbColor) float64 {
	dr := a.r - b.r
	dg := a.g - b.g
	db := a.b - b.b
	return dr*dr + dg*dg + db*db
}

func contrastRatio(a, b rgbColor) float64 {
	la := relativeLuminance(a)
	lb := relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(c rgbColor) float64 {
	r := linearizeChannel(c.r / 255.0)
	g := linearizeChannel(c.g / 255.0)
	b := linearizeChannel(c.b / 255.0)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func linearizeChannel(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
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
		BorderBackground(bgColor).
		Background(bgSoftColor).
		Width(max(20, width-2)).
		Height(max(8, height-8)).
		Padding(panelPaddingY, panelPaddingX)
}

func modalCardStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Border(modalBorder).
		BorderForeground(modalBorderColor).
		BorderBackground(modalBgColor).
		Background(modalBgColor).
		Padding(modalPaddingY, modalPaddingX)
}
