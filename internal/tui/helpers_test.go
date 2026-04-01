package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
)

func TestRenderSortSegment(t *testing.T) {
	if got := (Model{}).renderSortSegment(); got != "sort: alpha" {
		require.Failf(t, "assertion failed", "expected alpha sort segment, got %q", got)
	}
	if got := (Model{sortByModTime: true}).renderSortSegment(); got != "sort: modified" {
		require.Failf(t, "assertion failed", "expected modified sort segment, got %q", got)
	}
}

func TestHighlightMatch(t *testing.T) {
	matched := highlightMatch("Project Notes", "notes")
	if plain := stripANSI(matched); plain != "Project Notes" {
		require.Failf(t, "assertion failed", "expected text to be preserved, got %q", plain)
	}

	unmatched := highlightMatch("Project Notes", "todo")
	if plain := stripANSI(unmatched); plain != "Project Notes" {
		require.Failf(t, "assertion failed", "expected unmatched text to remain unchanged, got %q", plain)
	}
}

func TestTrimOrPad(t *testing.T) {
	if got := trimOrPad("abc", 5); got != "abc  " {
		require.Failf(t, "assertion failed", "expected padded string, got %q", got)
	}
	if got := trimOrPad("abcdef", 4); got != "abcd" {
		require.Failf(t, "assertion failed", "expected trimmed string, got %q", got)
	}
	if got := trimOrPad("abcd", 4); got != "abcd" {
		require.Failf(t, "assertion failed", "expected exact-width string, got %q", got)
	}
}

func TestFindMatchExcerpt(t *testing.T) {
	n := notes.Note{
		Preview: strings.Join([]string{
			"---",
			"tags: project, urgent",
			"---",
			"## Planning",
			"Need follow-up soon",
		}, "\n"),
		Tags: []string{"project", "urgent"},
	}

	if got := findMatchExcerpt(n, "#urg"); got != "tag:urgent" {
		require.Failf(t, "assertion failed", "expected tag match excerpt, got %q", got)
	}
	if got := findMatchExcerpt(n, "follow"); got != "Planning › Need follow-up soon" {
		require.Failf(t, "assertion failed", "unexpected excerpt, got %q", got)
	}

	n.Encrypted = true
	if got := findMatchExcerpt(n, "follow"); got != "<encrypted>" {
		require.Failf(t, "assertion failed", "expected encrypted marker, got %q", got)
	}
}

func TestPreviewPrivacyAndBlurHelpers(t *testing.T) {
	m := Model{cfg: config.Config{Preview: config.PreviewConfig{Privacy: true}}}
	if !m.effectivePreviewPrivacy(false) {
		require.FailNow(t, "expected config privacy to force preview privacy")
	}
	m = Model{previewPrivacyEnabled: true}
	if !m.effectivePreviewPrivacy(false) {
		require.FailNow(t, "expected runtime privacy toggle to force preview privacy")
	}
	m = Model{}
	if !m.effectivePreviewPrivacy(true) {
		require.FailNow(t, "expected note-forced privacy to force preview privacy")
	}

	blurred := blurRenderedText("abc \t\n\x1b[31mred\x1b[0m")
	if !strings.Contains(blurred, "•") || !strings.Contains(blurred, "\x1b[31m") {
		require.Failf(t, "assertion failed", "expected blurred text to preserve escapes and mask text, got %q", blurred)
	}

	plain := stripANSI("A\x1b[31mred\x1b[0mB")
	if plain != "AredB" {
		require.Failf(t, "assertion failed", "expected ANSI to be removed, got %q", plain)
	}
}

func TestUnderlineTagsAndPreviewRenderingHelpers(t *testing.T) {
	if !isUnderlineHeadingLine("────") {
		require.FailNow(t, "expected box-drawing underline to be recognized")
	}
	if isUnderlineHeadingLine("--") {
		require.FailNow(t, "expected plain hyphen line not to be recognized")
	}
	if got := renderTagsHeader(nil); got != "" {
		require.Failf(t, "assertion failed", "expected empty tags header, got %q", got)
	}
	if plain := stripANSI(renderTagsHeader([]string{"alpha", "beta"})); !strings.Contains(plain, "alpha") || !strings.Contains(plain, "beta") {
		require.Failf(t, "assertion failed", "expected tags header to contain tags, got %q", plain)
	}

	m := Model{cfg: config.Default()}
	rendered, offset := m.renderNotePreview("work/note.md", "---\ntags: alpha\n---\n# Body", []string{"alpha"})
	if offset != 2 {
		require.Failf(t, "assertion failed", "expected line number offset 2 when tags header is present, got %d", offset)
	}
	if plain := stripANSI(rendered); !strings.Contains(plain, "alpha") || !strings.Contains(plain, "Body") {
		require.Failf(t, "assertion failed", "expected rendered preview to contain tag header and body, got %q", plain)
	}
}

func TestPreviewMatchBuilders(t *testing.T) {
	content := "alpha beta\nalphaalpha\n~/notes/demo"
	matches := buildPreviewMatches(content, "alpha")
	if len(matches) != 2 {
		require.Failf(t, "assertion failed", "expected 2 merged matches, got %d", len(matches))
	}
	if got := buildPreviewMatches(content, "#tag"); got != nil {
		require.Failf(t, "assertion failed", "expected tag query to skip content matches, got %#v", got)
	}

	highlighted := applyMatchHighlights(content, "alpha", matches, 1)
	if plain := stripANSI(highlighted); plain != content {
		require.Failf(t, "assertion failed", "expected highlight application to preserve text, got %q", plain)
	}
	if got := previewLineForeground("~/notes/demo"); got != accentColor {
		require.Failf(t, "assertion failed", "expected note path line to use accent color, got %q", got)
	}
	if got := previewLineForeground("other"); got != textColor {
		require.Failf(t, "assertion failed", "expected normal line to use text color, got %q", got)
	}
	if plain := stripANSI(highlightTermsInLine("alpha beta", []string{"alpha"}, 0, lipgloss.Color("#FFFFFF"))); plain != "alpha beta" {
		require.Failf(t, "assertion failed", "expected highlighted line to preserve content, got %q", plain)
	}
}

func TestThemeAndColorHelpers(t *testing.T) {
	if got := normalizeThemeName(" mocha "); got != "catppuccin" {
		require.Failf(t, "assertion failed", "expected mocha alias to normalize to catppuccin, got %q", got)
	}
	if got := normalizeThemeName("Catppuccin-Latte"); got != "latte" {
		require.Failf(t, "assertion failed", "expected latte alias to normalize to latte, got %q", got)
	}

	if _, ok := parseHexColor("#AABBCC"); !ok {
		require.FailNow(t, "expected valid hex color to parse")
	}
	if _, ok := parseHexColor("nope"); ok {
		require.FailNow(t, "expected invalid hex color parse to fail")
	}
	if got := formatHexColor(rgbColor{r: 10, g: 20, b: 30}); got != "#0A141E" {
		require.Failf(t, "assertion failed", "unexpected formatted color: %q", got)
	}
	if got := firstNonEmpty(" value ", "fallback"); got != " value " {
		require.Failf(t, "assertion failed", "expected non-empty string to win, got %q", got)
	}
	if got := string(firstNonEmptyColor("", "#112233")); got != "#112233" {
		require.Failf(t, "assertion failed", "unexpected fallback color: %q", got)
	}

	if got := ensureContrast("#777777", "#777777", 4.5); got == "#777777" {
		require.Failf(t, "assertion failed", "expected low-contrast color to be adjusted, got %q", got)
	}
	if got := deriveInlineCodeBgColor("#F0F0F0", "#CCCCCC"); got == "#CCCCCC" {
		require.Failf(t, "assertion failed", "expected inline code bg color to be derived, got %q", got)
	}

	p := themePalette{
		TextColor:       "#777777",
		PanelBgColor:    "#777777",
		MutedColor:      "#777777",
		AccentSoftColor: "#777777",
		AccentColor:     "#777777",
		BgColor:         "#777777",
		MarkedItemColor: "#777777",
		SelectedFgColor: "#777777",
		SelectedBgColor: "#777777",
	}
	normalized := normalizePaletteAccessibility(p)
	if normalized.TextColor == p.TextColor {
		require.FailNow(t, "expected accessibility normalization to adjust low-contrast text color")
	}
}
