package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
)

func TestRenderSortSegment(t *testing.T) {
	if got := (Model{}).renderSortSegment(); got != "sort: alpha" {
		t.Fatalf("expected alpha sort segment, got %q", got)
	}
	if got := (Model{sortByModTime: true}).renderSortSegment(); got != "sort: modified" {
		t.Fatalf("expected modified sort segment, got %q", got)
	}
}

func TestHighlightMatch(t *testing.T) {
	matched := highlightMatch("Project Notes", "notes")
	if plain := stripANSI(matched); plain != "Project Notes" {
		t.Fatalf("expected text to be preserved, got %q", plain)
	}

	unmatched := highlightMatch("Project Notes", "todo")
	if plain := stripANSI(unmatched); plain != "Project Notes" {
		t.Fatalf("expected unmatched text to remain unchanged, got %q", plain)
	}
}

func TestTrimOrPad(t *testing.T) {
	if got := trimOrPad("abc", 5); got != "abc  " {
		t.Fatalf("expected padded string, got %q", got)
	}
	if got := trimOrPad("abcdef", 4); got != "abcd" {
		t.Fatalf("expected trimmed string, got %q", got)
	}
	if got := trimOrPad("abcd", 4); got != "abcd" {
		t.Fatalf("expected exact-width string, got %q", got)
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
		t.Fatalf("expected tag match excerpt, got %q", got)
	}
	if got := findMatchExcerpt(n, "follow"); got != "Planning › Need follow-up soon" {
		t.Fatalf("unexpected excerpt, got %q", got)
	}

	n.Encrypted = true
	if got := findMatchExcerpt(n, "follow"); got != "<encrypted>" {
		t.Fatalf("expected encrypted marker, got %q", got)
	}
}

func TestPreviewPrivacyAndBlurHelpers(t *testing.T) {
	m := Model{cfg: config.Config{Preview: config.PreviewConfig{Privacy: true}}}
	if !m.effectivePreviewPrivacy(false) {
		t.Fatal("expected config privacy to force preview privacy")
	}
	m = Model{previewPrivacyEnabled: true}
	if !m.effectivePreviewPrivacy(false) {
		t.Fatal("expected runtime privacy toggle to force preview privacy")
	}
	m = Model{}
	if !m.effectivePreviewPrivacy(true) {
		t.Fatal("expected note-forced privacy to force preview privacy")
	}

	blurred := blurRenderedText("abc \t\n\x1b[31mred\x1b[0m")
	if !strings.Contains(blurred, "•") || !strings.Contains(blurred, "\x1b[31m") {
		t.Fatalf("expected blurred text to preserve escapes and mask text, got %q", blurred)
	}

	plain := stripANSI("A\x1b[31mred\x1b[0mB")
	if plain != "AredB" {
		t.Fatalf("expected ANSI to be removed, got %q", plain)
	}
}

func TestUnderlineTagsAndPreviewRenderingHelpers(t *testing.T) {
	if !isUnderlineHeadingLine("────") {
		t.Fatal("expected box-drawing underline to be recognized")
	}
	if isUnderlineHeadingLine("--") {
		t.Fatal("expected plain hyphen line not to be recognized")
	}
	if got := renderTagsHeader(nil); got != "" {
		t.Fatalf("expected empty tags header, got %q", got)
	}
	if plain := stripANSI(renderTagsHeader([]string{"alpha", "beta"})); !strings.Contains(plain, "alpha") || !strings.Contains(plain, "beta") {
		t.Fatalf("expected tags header to contain tags, got %q", plain)
	}

	m := Model{cfg: config.Default()}
	rendered, offset := m.renderNotePreview("work/note.md", "---\ntags: alpha\n---\n# Body", []string{"alpha"})
	if offset != 2 {
		t.Fatalf("expected line number offset 2 when tags header is present, got %d", offset)
	}
	if plain := stripANSI(rendered); !strings.Contains(plain, "alpha") || !strings.Contains(plain, "Body") {
		t.Fatalf("expected rendered preview to contain tag header and body, got %q", plain)
	}
}

func TestPreviewMatchBuilders(t *testing.T) {
	content := "alpha beta\nalphaalpha\n~/notes/demo"
	matches := buildPreviewMatches(content, "alpha")
	if len(matches) != 2 {
		t.Fatalf("expected 2 merged matches, got %d", len(matches))
	}
	if got := buildPreviewMatches(content, "#tag"); got != nil {
		t.Fatalf("expected tag query to skip content matches, got %#v", got)
	}

	highlighted := applyMatchHighlights(content, "alpha", matches, 1)
	if plain := stripANSI(highlighted); plain != content {
		t.Fatalf("expected highlight application to preserve text, got %q", plain)
	}
	if got := previewLineForeground("~/notes/demo"); got != accentColor {
		t.Fatalf("expected note path line to use accent color, got %q", got)
	}
	if got := previewLineForeground("other"); got != textColor {
		t.Fatalf("expected normal line to use text color, got %q", got)
	}
	if plain := stripANSI(highlightTermsInLine("alpha beta", []string{"alpha"}, 0, lipgloss.Color("#FFFFFF"))); plain != "alpha beta" {
		t.Fatalf("expected highlighted line to preserve content, got %q", plain)
	}
}

func TestThemeAndColorHelpers(t *testing.T) {
	if got := normalizeThemeName(" mocha "); got != "catppuccin" {
		t.Fatalf("expected mocha alias to normalize to catppuccin, got %q", got)
	}
	if got := normalizeThemeName("Catppuccin-Latte"); got != "latte" {
		t.Fatalf("expected latte alias to normalize to latte, got %q", got)
	}

	if _, ok := parseHexColor("#AABBCC"); !ok {
		t.Fatal("expected valid hex color to parse")
	}
	if _, ok := parseHexColor("nope"); ok {
		t.Fatal("expected invalid hex color parse to fail")
	}
	if got := formatHexColor(rgbColor{r: 10, g: 20, b: 30}); got != "#0A141E" {
		t.Fatalf("unexpected formatted color: %q", got)
	}
	if got := firstNonEmpty(" value ", "fallback"); got != " value " {
		t.Fatalf("expected non-empty string to win, got %q", got)
	}
	if got := string(firstNonEmptyColor("", "#112233")); got != "#112233" {
		t.Fatalf("unexpected fallback color: %q", got)
	}

	if got := ensureContrast("#777777", "#777777", 4.5); got == "#777777" {
		t.Fatalf("expected low-contrast color to be adjusted, got %q", got)
	}
	if got := deriveInlineCodeBgColor("#F0F0F0", "#CCCCCC"); got == "#CCCCCC" {
		t.Fatalf("expected inline code bg color to be derived, got %q", got)
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
		t.Fatal("expected accessibility normalization to adjust low-contrast text color")
	}
}
