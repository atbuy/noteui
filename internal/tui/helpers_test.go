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
	// Foreground colours (priority, due-date) must survive the background swap.
	selectedBg, ok := ansiBackgroundParam(selectedBgColor)
	require.True(t, ok)
	require.Contains(t, selected, selectedBg)
	// Verify that the original foreground colour params are still present
	// (they were only in the rendered content because markdown rendering
	// produced ANSI in the test env).
	require.Contains(t, selected, stripANSI(prioritySpan)) // "[p1]" plain text still there
	require.Contains(t, selected, stripANSI(dueSpan))      // "[due:2020-01-01]" still there

	plainRendered := renderTodoPreviewLine("[ ] Plain task [p2] [due:2020-01-01]")
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

func TestApplyTodoPreviewHighlightHighlightsWrappedRenderedLines(t *testing.T) {
	ApplyTheme(config.Default())
	m := Model{cfg: config.Default()}
	m.preview.Width = 24
	raw := "- [ ] This is a long todo item that wraps across rendered lines\n"
	rendered, lineOffset := m.renderNotePreview("test.md", raw, nil)
	m.rebuildPreviewTodos(raw, rendered, lineOffset)
	require.Greater(t, len(m.previewTodos), 0)
	todo := m.previewTodos[0]
	require.Greater(t, todo.rendEndLine, todo.rendLine, "expected todo to wrap to multiple lines")
	require.NotEmpty(t, todo.raw)

	highlighted, ok := m.replaceTodoPreviewBlock(rendered, todo)
	require.True(t, ok, "selected block replacement should match rendered todo lines")
	selectedBg, ok := ansiBackgroundParam(selectedBgColor)
	require.True(t, ok)
	lines := strings.Split(highlighted, "\n")
	for li := todo.rendLine; li <= todo.rendEndLine; li++ {
		require.Contains(t, lines[li], selectedBg, "line %d missing selected background", li)
		require.Equal(t, m.preview.Width, lipgloss.Width(lines[li]), "line %d not padded to full width", li)
	}
	require.Contains(t, stripANSI(lines[todo.rendLine]), "[ ]")
}

// TestRebuildPreviewTodosWrappedEndLineMatchesActualContent verifies that
// rebuildPreviewTodos computes rendEndLine by scanning the actual rendered
// content rather than re-rendering in isolation (which may use a different
// effective width, e.g. when line numbers are enabled).
func TestRebuildPreviewTodosWrappedEndLineMatchesActualContent(t *testing.T) {
	ApplyTheme(config.Default())

	checkAllLinesHighlighted := func(t *testing.T, m Model, raw string) {
		t.Helper()
		rendered, lineOffset := m.renderNotePreview("test.md", raw, nil)
		m.rebuildPreviewTodos(raw, rendered, lineOffset)
		require.Greater(t, len(m.previewTodos), 0, "expected todos to be parsed")

		selectedBg, ok := ansiBackgroundParam(selectedBgColor)
		require.True(t, ok)

		for ti, todo := range m.previewTodos {
			highlighted := applyTodoPreviewHighlight(rendered, todo, m.preview.Width)
			highlightedLines := strings.Split(highlighted, "\n")
			for li := todo.rendLine; li <= todo.rendEndLine; li++ {
				require.Less(t, li, len(highlightedLines), "todo[%d] rendEndLine=%d exceeds content", ti, todo.rendEndLine)
				line := highlightedLines[li]
				require.Contains(t, line, selectedBg, "todo[%d] line %d missing selected background", ti, li)
				require.Equal(t, m.preview.Width, lipgloss.Width(line),
					"todo[%d] line %d not padded to full width", ti, li)
			}
		}
	}

	// Without line numbers: widths match, single todo wrapping.
	t.Run("no_line_numbers", func(t *testing.T) {
		m := Model{cfg: config.Default()}
		m.preview.Width = 28
		checkAllLinesHighlighted(t, m, "- [ ] This is a really long todo item that definitely wraps at this narrow width\n")
	})

	// With line numbers: effective render width is narrower than m.preview.Width,
	// so the old span-via-rerender approach produced a short rendEndLine.
	t.Run("with_line_numbers", func(t *testing.T) {
		m := Model{cfg: config.Default()}
		m.preview.Width = 30
		m.previewLineNumbersEnabled = true
		checkAllLinesHighlighted(t, m, "- [ ] This is a really long todo item that definitely wraps\n")
	})

	// Multiple todos, middle one wraps.
	t.Run("multiple_todos_middle_wraps", func(t *testing.T) {
		m := Model{cfg: config.Default()}
		m.preview.Width = 28
		raw := "- [ ] Short todo\n- [ ] A longer todo item that definitely wraps at this width\n- [x] Done task\n"
		checkAllLinesHighlighted(t, m, raw)
	})
}

// A todo followed by a blank line and then a paragraph (and then another
// todo) must not have its highlight stretched across the gap. Previously
// the renderer set rendEndLine to the line before the next todo, so the
// blank line and paragraph lit up with the selection background. The fix
// scans indented continuation lines only, so the highlight stops at the
// todo's own wrap or at the first blank line.
func TestRebuildPreviewTodosHighlightDoesNotBleedAcrossBlankLines(t *testing.T) {
	ApplyTheme(config.Default())
	m := Model{cfg: config.Default()}
	m.preview.Width = 80

	raw := "" +
		"- [ ] first todo\n" +
		"\n" +
		"Some paragraph between todos.\n" +
		"\n" +
		"- [ ] second todo\n"

	rendered, lineOffset := m.renderNotePreview("test.md", raw, nil)
	m.rebuildPreviewTodos(raw, rendered, lineOffset)
	require.Len(t, m.previewTodos, 2)

	first := m.previewTodos[0]
	require.Equal(t, first.rendLine, first.rendEndLine,
		"first todo is a single rendered line; highlight must not extend beyond it")

	highlighted := applyTodoPreviewHighlight(rendered, first, m.preview.Width)
	lines := strings.Split(highlighted, "\n")
	selectedBg, ok := ansiBackgroundParam(selectedBgColor)
	require.True(t, ok)

	require.Contains(t, lines[first.rendLine], selectedBg,
		"the todo line itself must still be highlighted")
	for li := first.rendLine + 1; li < m.previewTodos[1].rendLine; li++ {
		require.NotContains(t, lines[li], selectedBg,
			"line %d (between todos) must not carry the selection background", li)
	}
}

func TestHighlightTodoLinePreservesColorsAndFillsWidth(t *testing.T) {
	ApplyTheme(config.Default())

	renderLine := func(raw string) string {
		rendered := renderMarkdownTerminal(raw+"\n", markdownRenderOptions{Width: 80})
		return strings.Split(rendered, "\n")[0]
	}

	selectedBg, ok := ansiBackgroundParam(selectedBgColor)
	require.True(t, ok)

	// Unchecked todo: plain text preserved, selected bg injected, width filled.
	unchecked := highlightTodoLine(renderLine("- [ ] pending task"), 80)
	require.Contains(t, stripANSI(unchecked), "[ ] pending task")
	require.Contains(t, unchecked, selectedBg)
	require.Equal(t, 80, lipgloss.Width(unchecked))

	// Checked todo: same.
	checked := highlightTodoLine(renderLine("- [x] done task"), 80)
	require.Contains(t, stripANSI(checked), "[X] done task")
	require.Contains(t, checked, selectedBg)
	require.Equal(t, 80, lipgloss.Width(checked))

	// Continuation line (no checkbox) — still gets selected bg and full width.
	cont := highlightTodoLine(renderLine("- [ ] short"), 20) // use narrow width so it wraps in a real note
	require.Contains(t, cont, selectedBg)
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

func TestTodoListWrapsAtWhitespaceNotHyphenOrPunctuation(t *testing.T) {
	ApplyTheme(config.Default())
	rendered := renderMarkdownTerminal("- [ ] state-of-the-art roadmap.\n", markdownRenderOptions{Width: 14})
	plain := stripANSI(rendered)
	require.Contains(t, plain, "[ ] state-of-the-art")
	require.NotContains(t, plain, "state-\nof-the-art")
	require.NotContains(t, plain, "roadmap\n.")
	require.NotContains(t, plain, "\n.")
	bgParam, ok := ansiBackgroundParam(bgSoftColor)
	require.True(t, ok)
	for _, line := range strings.Split(rendered, "\n") {
		require.Contains(t, line, bgParam)
	}
}

func TestBulletListItemHasNoExtraSpaceBetweenMarkerAndText(t *testing.T) {
	ApplyTheme(config.Default())
	rendered := renderMarkdownTerminal("- hello world\n", markdownRenderOptions{Width: 80})
	plain := stripANSI(rendered)
	require.Contains(t, plain, "• hello world")
	require.NotContains(t, plain, "•  ")
}

func TestNewThemesLoadWithoutPanic(t *testing.T) {
	for _, name := range []string{
		"crimson", "dusk",
		"rose-pine", "monokai", "solarized-dark", "ayu-dark", "material", "nightfox",
	} {
		cfg := config.Default()
		cfg.Theme.Name = name
		ApplyTheme(cfg)
		if string(accentColor) == "" {
			require.Failf(t, "assertion failed", "theme %q produced empty accent color", name)
		}
	}
}

func TestThemeAndColorHelpers(t *testing.T) {
	if got := NormalizeThemeName(" mocha "); got != "catppuccin" {
		require.Failf(t, "assertion failed", "expected mocha alias to normalize to catppuccin, got %q", got)
	}
	if got := NormalizeThemeName("Catppuccin-Latte"); got != "latte" {
		require.Failf(t, "assertion failed", "expected latte alias to normalize to latte, got %q", got)
	}
	if got := NormalizeThemeName("crimson"); got != "crimson" {
		require.Failf(t, "assertion failed", "expected 'crimson' to pass through unchanged, got %q", got)
	}
	if got := NormalizeThemeName("dusk"); got != "dusk" {
		require.Failf(t, "assertion failed", "expected 'dusk' to pass through unchanged, got %q", got)
	}
	if got := NormalizeThemeName("rosepine"); got != "rose-pine" {
		require.Failf(t, "assertion failed", "expected rosepine alias to normalize to rose-pine, got %q", got)
	}
	if got := NormalizeThemeName("ayu"); got != "ayu-dark" {
		require.Failf(t, "assertion failed", "expected ayu alias to normalize to ayu-dark, got %q", got)
	}
	if got := NormalizeThemeName("solarized"); got != "solarized-dark" {
		require.Failf(t, "assertion failed", "expected solarized alias to normalize to solarized-dark, got %q", got)
	}
	if got := NormalizeThemeName("material-dark"); got != "material" {
		require.Failf(t, "assertion failed", "expected material-dark alias to normalize to material, got %q", got)
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

func TestBuiltinThemesListIsComplete(t *testing.T) {
	themes := BuiltinThemes()
	require.NotEmpty(t, themes)
	for _, entry := range themes {
		require.NotEmpty(t, entry.Name, "theme entry must have a name")
		require.NotEmpty(t, entry.Description, "theme %q must have a description", entry.Name)
		require.NotEmpty(t, entry.Palette.AccentColor, "theme %q must have an AccentColor", entry.Name)
		require.NotEmpty(t, entry.Palette.BgColor, "theme %q must have a BgColor", entry.Name)
		require.NotEmpty(t, entry.Palette.TextColor, "theme %q must have a TextColor", entry.Name)
	}

	names := make(map[string]bool, len(themes))
	for _, entry := range themes {
		require.False(t, names[entry.Name], "duplicate theme name %q in BuiltinThemes", entry.Name)
		names[entry.Name] = true
	}

	for _, name := range []string{
		"default", "nord", "gruvbox", "catppuccin", "latte", "solarized-light", "paper",
		"onedark", "kanagawa", "dracula", "everforest", "tokyo-night-storm", "github-light",
		"github-dark", "carbonfox", "crimson", "dusk",
		"rose-pine", "monokai", "solarized-dark", "ayu-dark", "material", "nightfox",
	} {
		require.True(t, names[name], "BuiltinThemes missing expected theme %q", name)
	}
}

func TestTodoDueDateIsOverdue(t *testing.T) {
	overdue := todoListItem{Todo: notes.TodoItem{Metadata: notes.TodoMetadata{DueDate: time.Now().Add(-24 * time.Hour).Format("2006-01-02")}}}
	notDue := todoListItem{Todo: notes.TodoItem{Metadata: notes.TodoMetadata{DueDate: time.Now().Add(24 * time.Hour).Format("2006-01-02")}}}
	require.True(t, todoDueDateIsOverdue(overdue))
	require.False(t, todoDueDateIsOverdue(notDue))
	require.False(t, todoDueDateIsOverdue(todoListItem{}))
}
