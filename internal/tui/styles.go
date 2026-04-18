package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/tui/theme"
)

var (
	borderColor       lipgloss.Color
	focusBorderColor  lipgloss.Color
	accentColor       lipgloss.Color
	accentSoftColor   lipgloss.Color
	mutedColor        lipgloss.Color
	bgSoftColor       lipgloss.Color
	textColor         lipgloss.Color
	errorColor        lipgloss.Color
	successColor      lipgloss.Color
	bgColor           lipgloss.Color
	subtleColor       lipgloss.Color
	chipBgColor       lipgloss.Color
	inlineCodeBgColor lipgloss.Color
	pinnedNoteColor   lipgloss.Color
	syncedNoteColor   lipgloss.Color
	unsyncedNoteColor lipgloss.Color
	syncingNoteColor  lipgloss.Color
	sharedNoteColor   lipgloss.Color
	markedItemColor   lipgloss.Color
	selectedBgColor   lipgloss.Color
	selectedFgColor   lipgloss.Color
	highlightBgColor  lipgloss.Color

	modalBgColor     lipgloss.Color
	modalBorderColor lipgloss.Color
	modalTitleColor  lipgloss.Color
	modalTextColor   lipgloss.Color
	modalMutedColor  lipgloss.Color
	modalAccentColor lipgloss.Color
	modalErrorColor  lipgloss.Color

	appStyle         lipgloss.Style
	titleBarStyle    lipgloss.Style
	panelTitleStyle  lipgloss.Style
	mutedStyle       lipgloss.Style
	footerStyle      lipgloss.Style
	statusOKStyle    lipgloss.Style
	statusErrStyle   lipgloss.Style
	modalTitleStyle  lipgloss.Style
	modalTextStyle   lipgloss.Style
	modalMutedStyle  lipgloss.Style
	modalFooterStyle lipgloss.Style
	modalKeyStyle    lipgloss.Style
	modalErrorStyle  lipgloss.Style

	currentBorder lipgloss.Border
	modalBorder   lipgloss.Border

	panelPaddingX int
	panelPaddingY int
	appPaddingX   int
	appPaddingY   int
	modalPaddingX int
	modalPaddingY int

	boldTitleBar    bool
	boldPanelTitles bool
	boldSelected    bool
	boldModalTitles bool

	iconCategoryExpanded  string
	iconCategoryCollapsed string
	iconCategoryLeaf      string
	iconNote              string

	treeNoteStyle lipgloss.Style
)

type (
	themePalette      = theme.Palette
	BuiltinThemeEntry = theme.BuiltinThemeEntry
)

func builtinTheme(name string) themePalette { return theme.Builtin(name) }
func BuiltinThemes() []BuiltinThemeEntry    { return theme.BuiltinThemes() }
func NormalizeThemeName(name string) string { return theme.NormalizeName(name) }

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
	override(&p.InlineCodeBgColor, cfg.Theme.InlineCodeBgColor)
	override(&p.PinnedNoteColor, cfg.Theme.PinnedNoteColor)
	override(&p.SyncedNoteColor, cfg.Theme.SyncedNoteColor)
	override(&p.UnsyncedNoteColor, cfg.Theme.UnsyncedNoteColor)
	override(&p.SyncingNoteColor, cfg.Theme.SyncingNoteColor)
	override(&p.SharedNoteColor, cfg.Theme.SharedNoteColor)
	override(&p.MarkedItemColor, cfg.Theme.MarkedItemColor)
	override(&p.ErrorColor, cfg.Theme.ErrorColor)
	override(&p.SuccessColor, cfg.Theme.SuccessColor)
	override(&p.SelectedBgColor, cfg.Theme.SelectedBgColor)
	override(&p.SelectedFgColor, cfg.Theme.SelectedFgColor)
	override(&p.HighlightBgColor, cfg.Theme.HighlightBgColor)

	p = normalizePaletteAccessibility(p)
	if strings.TrimSpace(p.InlineCodeBgColor) == "" {
		p.InlineCodeBgColor = deriveInlineCodeBgColor(p.PanelBgColor, p.ChipBgColor)
	}
	if strings.TrimSpace(p.PinnedNoteColor) == "" {
		p.PinnedNoteColor = p.AccentSoftColor
	}
	if strings.TrimSpace(p.SyncedNoteColor) == "" {
		p.SyncedNoteColor = p.SuccessColor
	}
	if strings.TrimSpace(p.UnsyncedNoteColor) == "" {
		p.UnsyncedNoteColor = p.ErrorColor
	}
	if strings.TrimSpace(p.SyncingNoteColor) == "" {
		p.SyncingNoteColor = "#F59E0B"
	}
	if strings.TrimSpace(p.SharedNoteColor) == "" {
		p.SharedNoteColor = "#7C9EFF"
	}
	if strings.TrimSpace(p.MarkedItemColor) == "" {
		p.MarkedItemColor = "#E5A524"
	}

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
	inlineCodeBgColor = lipgloss.Color(p.InlineCodeBgColor)
	pinnedNoteColor = lipgloss.Color(p.PinnedNoteColor)
	syncedNoteColor = lipgloss.Color(p.SyncedNoteColor)
	unsyncedNoteColor = lipgloss.Color(p.UnsyncedNoteColor)
	syncingNoteColor = lipgloss.Color(p.SyncingNoteColor)
	sharedNoteColor = lipgloss.Color(p.SharedNoteColor)
	markedItemColor = lipgloss.Color(p.MarkedItemColor)
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
	modalErrorColor = firstNonEmptyColor(cfg.Modal.ErrorColor, p.ErrorColor)

	panelPaddingX = max(0, cfg.Theme.PanelPaddingX)
	panelPaddingY = max(0, cfg.Theme.PanelPaddingY)
	appPaddingX = max(0, cfg.Theme.AppPaddingX)
	appPaddingY = max(0, cfg.Theme.AppPaddingY)

	modalPaddingX = max(0, cfg.Modal.PaddingX)
	modalPaddingY = max(0, cfg.Modal.PaddingY)

	boldTitleBar = cfg.Typography.BoldTitleBar
	boldPanelTitles = cfg.Typography.BoldPanelTitles
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

	mutedStyle = lipgloss.NewStyle().
		Foreground(mutedColor)

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

	modalErrorStyle = lipgloss.NewStyle().
		Foreground(modalErrorColor).
		Background(modalBgColor).
		Bold(true)

	treeNoteStyle = lipgloss.NewStyle().
		Foreground(textColor).
		Background(bgSoftColor)
}

func normalizePaletteAccessibility(p themePalette) themePalette {
	p.TextColor = ensureContrast(p.TextColor, p.PanelBgColor, 7.0)
	p.MutedColor = ensureContrast(p.MutedColor, p.PanelBgColor, 4.5)
	p.AccentSoftColor = ensureContrast(p.AccentSoftColor, p.PanelBgColor, 4.5)
	p.AccentColor = ensureContrast(p.AccentColor, p.BgColor, 4.5)
	p.MarkedItemColor = ensureContrast(p.MarkedItemColor, p.PanelBgColor, 4.5)
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

func deriveInlineCodeBgColor(panelHex, chipHex string) string {
	panel, ok := parseHexColor(panelHex)
	if !ok {
		return chipHex
	}
	chip, ok := parseHexColor(chipHex)
	if !ok {
		return chipHex
	}

	amount := 0.24
	if relativeLuminance(panel) > 0.45 {
		amount = 0.30
	}

	return formatHexColor(mixColor(chip, rgbColor{}, amount))
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
