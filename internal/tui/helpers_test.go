package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
)

func TestRenderSortSegment(t *testing.T) {
	if got := (Model{}).renderSortSegment(); got != "sort: alpha" {
		require.Failf(t, "assertion failed", "expected alpha sort segment, got %q", got)
	}
	if got := (Model{sortMethod: sortModified}).renderSortSegment(); got != "sort: modified" {
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

func TestSyncTodoCursorToActiveMatchUsesMatchingTodoLine(t *testing.T) {
	m := newTestModel(t)
	m.previewTodoNavMode = true
	m.previewTodos = []previewTodoItem{
		{rendLine: 2, text: "first"},
		{rendLine: 5, text: "second"},
	}
	m.previewMatches = []previewMatch{{line: 5, occurrIdx: 0}}
	m.previewMatchIndex = 0

	m.syncTodoCursorToActiveMatch()

	require.Equal(t, 1, m.previewTodoCursor)
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

func TestTodoPreviewDueHintsPreserveRenderedInlineMarkdown(t *testing.T) {
	ApplyTheme(config.Default())
	rendered := renderMarkdownTerminal("- [ ] Fix **bold** and `code` [p1] [due:2020-01-01]\n", markdownRenderOptions{Width: 80})
	codeSpan := lipgloss.NewStyle().Foreground(accentSoftColor).Background(inlineCodeBgColor).Render("code")
	prioritySpan := lipgloss.NewStyle().Foreground(todoPriorityColor(1)).Background(bgSoftColor).Render("[p1]")
	dueSpan := lipgloss.NewStyle().Foreground(todoDueDateColor("2020-01-01", time.Now().Format("2006-01-02"))).Background(bgSoftColor).Render("[due:2020-01-01]")
	require.Contains(t, rendered, codeSpan)
	require.Contains(t, rendered, prioritySpan)
	require.Contains(t, rendered, dueSpan)

	hinted := applyTodoDueDateHints(rendered)
	require.Contains(t, stripANSI(hinted), "[ ] Fix bold and code [p1] [due:2020-01-01]")
	require.Contains(t, hinted, codeSpan)
	require.Contains(t, hinted, prioritySpan)
	require.Contains(t, hinted, dueSpan)

	selected := applyTodoLineHighlight(hinted, 0)
	require.Contains(t, stripANSI(selected), "[ ] Fix bold and code [p1] [due:2020-01-01]")
	require.Contains(t, selected, codeSpan)
	selectedPrioritySpan := lipgloss.NewStyle().Foreground(todoPriorityColor(1)).Background(selectedBgColor).Render("[p1]")
	selectedDueSpan := lipgloss.NewStyle().Foreground(todoDueDateColor("2020-01-01", time.Now().Format("2006-01-02"))).Background(selectedBgColor).Render("[due:2020-01-01]")
	require.Contains(t, selected, selectedPrioritySpan)
	require.Contains(t, selected, selectedDueSpan)

	plainRendered := renderTodoPreviewLine("[ ] Plain task [p2] [due:2020-01-01]", false)
	require.Contains(t, stripANSI(plainRendered), "[ ] Plain task [p2] [due:2020-01-01]")
	require.Contains(t, plainRendered, lipgloss.NewStyle().Foreground(todoPriorityColor(2)).Background(bgSoftColor).Render("[p2]"))
	require.Contains(t, plainRendered, dueSpan)

	row := (Model{}).renderTodoListRow(todoListItem{Todo: notes.TodoItem{DisplayText: "List task", Metadata: notes.TodoMetadata{Priority: 3}}}, 80, false)
	require.Contains(t, row, lipgloss.NewStyle().Foreground(todoPriorityColor(3)).Background(bgSoftColor).Render("[p3]"))

	m := Model{cfg: config.Default()}
	m.preview.Width = 80
	normalPreview, _ := m.renderNotePreview("work/todo.md", "- [ ] Normal task [p1] [due:2020-01-01]\n", nil)
	require.Contains(t, stripANSI(normalPreview), "[ ] Normal task [p1] [due:2020-01-01]")
	require.Contains(t, normalPreview, prioritySpan)
	require.Contains(t, normalPreview, dueSpan)

	gapRendered := applyTodoMetadataHighlights("[p1] [due:2020-01-01]")
	gapSpan := lipgloss.NewStyle().Foreground(textColor).Background(bgSoftColor).Render(" ")
	require.Contains(t, gapRendered, prioritySpan+gapSpan+dueSpan)
}

func TestCheckboxListItemHasNoExtraSpaceBetweenMarkerAndText(t *testing.T) {
	ApplyTheme(config.Default())
	rendered := renderMarkdownTerminal("- [ ] unchecked\n- [x] checked\n", markdownRenderOptions{Width: 80})
	plain := stripANSI(rendered)
	require.Contains(t, plain, "[ ] unchecked")
	require.Contains(t, plain, "[X] checked")
	// Marker and text must be separated by exactly one space: no gap like "[ ]    text".
	require.NotContains(t, plain, "[ ]  ")
	require.NotContains(t, plain, "[X]  ")
}

func TestBulletListItemHasNoExtraSpaceBetweenMarkerAndText(t *testing.T) {
	ApplyTheme(config.Default())
	rendered := renderMarkdownTerminal("- hello world\n", markdownRenderOptions{Width: 80})
	plain := stripANSI(rendered)
	require.Contains(t, plain, "• hello world")
	require.NotContains(t, plain, "•  ")
}

func TestNewThemesLoadWithoutPanic(t *testing.T) {
	for _, name := range []string{"crimson", "dusk"} {
		cfg := config.Default()
		cfg.Theme.Name = name
		ApplyTheme(cfg)
		if string(accentColor) == "" {
			require.Failf(t, "assertion failed", "theme %q produced empty accent color", name)
		}
	}
}

func TestThemeAndColorHelpers(t *testing.T) {
	if got := normalizeThemeName(" mocha "); got != "catppuccin" {
		require.Failf(t, "assertion failed", "expected mocha alias to normalize to catppuccin, got %q", got)
	}
	if got := normalizeThemeName("Catppuccin-Latte"); got != "latte" {
		require.Failf(t, "assertion failed", "expected latte alias to normalize to latte, got %q", got)
	}
	if got := normalizeThemeName("crimson"); got != "crimson" {
		require.Failf(t, "assertion failed", "expected 'crimson' to pass through unchanged, got %q", got)
	}
	if got := normalizeThemeName("dusk"); got != "dusk" {
		require.Failf(t, "assertion failed", "expected 'dusk' to pass through unchanged, got %q", got)
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

func TestTodoDueDateIsOverdue(t *testing.T) {
	overdue := todoListItem{Todo: notes.TodoItem{Metadata: notes.TodoMetadata{DueDate: time.Now().Add(-24 * time.Hour).Format("2006-01-02")}}}
	notDue := todoListItem{Todo: notes.TodoItem{Metadata: notes.TodoMetadata{DueDate: time.Now().Add(24 * time.Hour).Format("2006-01-02")}}}
	require.True(t, todoDueDateIsOverdue(overdue))
	require.False(t, todoDueDateIsOverdue(notDue))
	require.False(t, todoDueDateIsOverdue(todoListItem{}))
}
