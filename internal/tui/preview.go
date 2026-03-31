package tui

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/notes"
)

func (m *Model) refreshPreview() {
	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil {
			m.previewPath = ""
			m.previewContent = "No pinned item selected"
			m.previewPrivacyForcedByNote = false
			m.preview.SetContent(m.previewContent)
			m.rebuildPreviewHeadingsFromRendered()
			m.previewMatches = nil
			m.preview.GotoTop()
			return
		}

		switch item.Kind {
		case pinItemCategory:
			pathText := filepath.Join("~/notes", item.RelPath)
			count := m.countNotesUnder(item.RelPath)
			children := m.countChildCategories(item.RelPath)

			content := strings.Join([]string{
				"# " + pathText,
				"",
				fmt.Sprintf("- Subcategories: %d", children),
				fmt.Sprintf("- Notes: %d", count),
				"",
				"Pinned category. Press enter to jump to it in the tree.",
			}, "\n")

			rendered := m.renderPreviewMarkdown(pathText, content)
			m.previewPath = "pinned-category:" + item.RelPath
			m.previewContent = rendered
			m.previewPrivacyForcedByNote = false
			m.preview.SetContent(rendered)
			m.rebuildPreviewHeadingsFromRendered()
			m.previewMatches = nil
			m.preview.GotoTop()
			return

		case pinItemNote, pinItemTemporaryNote:
			if m.previewPath == item.Path {
				return
			}
			rel := item.RelPath
			if item.Kind == pinItemTemporaryNote {
				rel = filepath.Join(".tmp", rel)
			}
			m.previewPath = item.Path
			m.previewMatches = nil
			m.pendingPreviewCmd = m.notePreviewCmd(item.Path, rel, item.Tags)
			return
		}
	}

	if m.listMode == listModeTemporary {
		n := m.currentTempNote()
		if n == nil {
			m.previewPath = ""
			m.previewContent = "No temporary note selected"
			m.preview.SetContent(m.previewContent)
			m.rebuildPreviewHeadingsFromRendered()
			m.previewMatches = nil
			m.preview.GotoTop()
			return
		}

		if m.previewPath == n.Path {
			return
		}
		m.previewPath = n.Path
		m.previewMatches = nil
		m.pendingPreviewCmd = m.notePreviewCmd(n.Path, filepath.Join(".tmp", n.RelPath), n.Tags)
		return
	}

	item := m.currentTreeItem()
	if item == nil {
		m.previewPath = ""
		m.previewContent = "Nothing selected"
		m.previewPrivacyForcedByNote = false
		m.preview.SetContent(m.previewContent)
		m.rebuildPreviewHeadingsFromRendered()
		m.previewMatches = nil
		m.preview.GotoTop()
		return
	}

	if item.Kind == treeCategory {
		pathText := item.Name
		if item.RelPath == "" {
			pathText = "~/notes"
		} else {
			pathText = filepath.Join("~/notes", item.RelPath)
		}

		count := m.countNotesUnder(item.RelPath)
		children := m.countChildCategories(item.RelPath)

		content := strings.Join([]string{
			"# " + pathText,
			"",
			fmt.Sprintf("- Subcategories: %d", children),
			fmt.Sprintf("- Notes: %d", count),
			"",
			"Category selected. Press enter or space to expand/collapse.",
		}, "\n")

		rendered := m.renderPreviewMarkdown(pathText, content)
		m.previewPath = "category:" + item.RelPath
		m.previewContent = rendered
		m.previewPrivacyForcedByNote = false
		m.preview.SetContent(rendered)
		m.rebuildPreviewHeadingsFromRendered()
		m.previewMatches = nil
		m.preview.GotoTop()
		return
	}

	if item.Note == nil {
		m.previewPath = ""
		m.previewContent = "No note selected"
		m.preview.SetContent(m.previewContent)
		m.rebuildPreviewHeadingsFromRendered()
		m.previewMatches = nil
		m.preview.GotoTop()
		return
	}

	if m.previewPath == item.Note.Path {
		return
	}
	m.previewPath = item.Note.Path
	m.previewMatches = nil
	m.pendingPreviewCmd = m.notePreviewCmd(item.Note.Path, item.Note.RelPath, item.Note.Tags)
}

func (m Model) notePreviewCmd(notePath, relPath string, tags []string) tea.Cmd {
	return func() tea.Msg {
		raw, err := notes.ReadAll(notePath)
		if err != nil {
			return previewRenderedMsg{
				forPath:     notePath,
				baseContent: "Failed to read note: " + err.Error(),
			}
		}

		if notes.NoteIsEncrypted(raw) {
			if m.sessionPassphrase == "" {
				return previewLockedMsg{path: notePath}
			}
			decrypted, err := notes.DecryptForPreview(raw, m.sessionPassphrase)
			if err != nil {
				return previewRenderedMsg{
					forPath:     notePath,
					baseContent: "[decryption failed — wrong passphrase?]",
				}
			}
			raw = decrypted
		}

		private := notes.NoteIsPrivate(raw)
		rendered := m.renderNotePreview(relPath, raw, tags)
		if m.effectivePreviewPrivacy(private) {
			rendered = blurRenderedText(rendered)
		}
		return previewRenderedMsg{
			forPath:             notePath,
			baseContent:         rendered,
			rawContent:          notes.StripFrontMatter(raw),
			privacyForcedByNote: private,
		}
	}
}

func (m Model) renderPreviewMarkdown(relPath, raw string) string {
	if !m.cfg.Preview.RenderMarkdown || m.previewMarkdownDisabledFor(relPath) {
		return raw
	}

	width := m.preview.Width
	if width <= 0 {
		width = max(20, m.previewWidth-8)
	}

	opts := markdownRenderOptions{
		Width:           width,
		SyntaxHighlight: m.cfg.Preview.SyntaxHighlight,
		CodeStyle:       m.cfg.Preview.CodeStyle,
	}

	return renderMarkdownTerminal(raw, opts)
}

func (m Model) previewMarkdownDisabledFor(relPath string) bool {
	relPath = filepath.ToSlash(relPath)
	for _, pattern := range m.cfg.Preview.DisablePaths {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		pattern = filepath.ToSlash(pattern)

		if ok, err := path.Match(pattern, relPath); err == nil && ok {
			return true
		}
		if relPath == pattern {
			return true
		}
		if strings.HasPrefix(relPath, strings.TrimSuffix(pattern, "/")+"/") {
			return true
		}
	}
	return false
}

func (m *Model) rebuildPreviewHeadingsFromRendered() {
	m.previewHeadings = m.previewHeadings[:0]

	lines := strings.Split(stripANSI(m.previewContent), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Heuristic based on your renderer:
		// H1/H2/H3/H4 render as plain heading text, and H2 is followed by underline.
		if strings.HasPrefix(trimmed, "▸ ") || strings.HasPrefix(trimmed, "• ") {
			m.previewHeadings = append(m.previewHeadings, i)
			continue
		}

		if i+1 < len(lines) {
			next := strings.TrimSpace(lines[i+1])
			if next != "" && isUnderlineHeadingLine(next) {
				m.previewHeadings = append(m.previewHeadings, i)
				continue
			}
		}
	}
}

func (m *Model) jumpToNextHeading() {
	if len(m.previewHeadings) == 0 {
		m.status = "no headings"
		return
	}

	cur := m.preview.YOffset
	for _, line := range m.previewHeadings {
		if line > cur {
			m.preview.SetYOffset(line)
			m.status = "next heading"
			return
		}
	}

	m.preview.SetYOffset(m.previewHeadings[len(m.previewHeadings)-1])
	m.status = "last heading"
}

func (m *Model) jumpToPrevHeading() {
	if len(m.previewHeadings) == 0 {
		m.status = "no headings"
		return
	}

	cur := m.preview.YOffset
	prev := m.previewHeadings[0]

	for _, line := range m.previewHeadings {
		if line >= cur {
			break
		}
		prev = line
	}

	m.preview.SetYOffset(prev)
	m.status = "previous heading"
}

func (m *Model) rebuildPreviewTodos(raw, rendered string) {
	m.previewTodos = m.previewTodos[:0]

	rawLines := strings.Split(raw, "\n")
	type rawTodo struct {
		lineIdx int
		checked bool
		text    string
	}
	var rawTodos []rawTodo
	for i, line := range rawLines {
		trimmed := strings.TrimLeft(line, " \t")
		switch {
		case strings.HasPrefix(trimmed, "- [ ] "):
			rawTodos = append(rawTodos, rawTodo{i, false, trimmed[6:]})
		case strings.HasPrefix(trimmed, "- [x] "), strings.HasPrefix(trimmed, "- [X] "):
			rawTodos = append(rawTodos, rawTodo{i, true, trimmed[6:]})
		}
	}
	if len(rawTodos) == 0 {
		m.previewTodoCursor = -1
		return
	}

	rendLines := strings.Split(stripANSI(rendered), "\n")
	var rendTodoLines []int
	for i, line := range rendLines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[ ]") || strings.HasPrefix(t, "[X]") ||
			strings.HasPrefix(t, "[x]") {
			rendTodoLines = append(rendTodoLines, i)
		}
	}

	limit := min(len(rawTodos), len(rendTodoLines))
	for i := range limit {
		m.previewTodos = append(m.previewTodos, previewTodoItem{
			rawLine:  rawTodos[i].lineIdx,
			rendLine: rendTodoLines[i],
			checked:  rawTodos[i].checked,
			text:     rawTodos[i].text,
		})
	}

	for i := limit; i < len(rawTodos); i++ {
		m.previewTodos = append(m.previewTodos, previewTodoItem{
			rawLine:  rawTodos[i].lineIdx,
			rendLine: -1,
			checked:  rawTodos[i].checked,
			text:     rawTodos[i].text,
		})
	}

	if m.previewTodoCursor >= len(m.previewTodos) {
		m.previewTodoCursor = max(0, len(m.previewTodos)-1)
	}
	if m.previewTodoCursor < 0 && !m.previewTodoNavMode {
		m.previewTodoCursor = -1
	}
}

func (m *Model) jumpToNextTodo() {
	if len(m.previewTodos) == 0 {
		m.status = "no todos"
		return
	}
	if m.previewTodoCursor < 0 {
		m.previewTodoCursor = 0
	} else {
		m.previewTodoCursor = (m.previewTodoCursor + 1) % len(m.previewTodos)
	}
	todo := m.previewTodos[m.previewTodoCursor]
	if todo.rendLine >= 0 {
		m.preview.SetYOffset(todo.rendLine)
	}
	m.status = fmt.Sprintf("todo %d/%d", m.previewTodoCursor+1, len(m.previewTodos))
	m.reapplyTodoHighlight()
}

func (m *Model) jumpToPrevTodo() {
	if len(m.previewTodos) == 0 {
		m.status = "no todos"
		return
	}
	if m.previewTodoCursor < 0 {
		m.previewTodoCursor = len(m.previewTodos) - 1
	} else {
		m.previewTodoCursor = (m.previewTodoCursor - 1 + len(m.previewTodos)) % len(m.previewTodos)
	}
	todo := m.previewTodos[m.previewTodoCursor]
	if todo.rendLine >= 0 {
		m.preview.SetYOffset(todo.rendLine)
	}
	m.status = fmt.Sprintf("todo %d/%d", m.previewTodoCursor+1, len(m.previewTodos))
	m.reapplyTodoHighlight()
}

func applyTodoLineHighlight(content string, rendLine int) string {
	if rendLine < 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	if rendLine >= len(lines) {
		return content
	}
	plain := stripANSI(lines[rendLine])
	lines[rendLine] = renderSelectedTodoLine(plain)
	return strings.Join(lines, "\n")
}

func renderSelectedTodoLine(plain string) string {
	base := lipgloss.NewStyle().
		Background(selectedBgColor).
		Foreground(selectedFgColor).
		Bold(true)

	indentWidth := len(plain) - len(strings.TrimLeft(plain, " "))
	indent := strings.Repeat(" ", indentWidth)
	body := plain[indentWidth:]

	switch {
	case strings.HasPrefix(body, "[X] "), strings.HasPrefix(body, "[x] "):
		rest := body[4:]
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			base.Render(indent),
			lipgloss.NewStyle().
				Background(selectedBgColor).
				Foreground(successColor).
				Bold(true).
				Render("[X]"),
			base.Render(" "),
			base.Render(rest),
		)
	case strings.HasPrefix(body, "[ ] "):
		rest := body[4:]
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			base.Render(indent),
			lipgloss.NewStyle().
				Background(selectedBgColor).
				Foreground(errorColor).
				Bold(true).
				Render("[ ]"),
			base.Render(" "),
			base.Render(rest),
		)
	default:
		return base.Render(plain)
	}
}

func (m *Model) reapplyTodoHighlight() {
	if !m.previewTodoNavMode {
		m.preview.SetContent(m.previewContent)
		return
	}
	if len(m.previewTodos) == 0 || m.previewTodoCursor < 0 ||
		m.previewTodoCursor >= len(m.previewTodos) {
		m.preview.SetContent(m.previewContent)
		return
	}
	todo := m.previewTodos[m.previewTodoCursor]
	m.preview.SetContent(applyTodoLineHighlight(m.previewContent, todo.rendLine))
}

func (m Model) mouseInPreview(x, y int) bool {
	return x >= m.previewPaneX &&
		x < m.previewPaneX+m.previewPaneW &&
		y >= m.previewPaneY &&
		y < m.previewPaneY+m.previewPaneH
}

func (m Model) effectivePreviewPrivacy(noteForced bool) bool {
	return m.cfg.Preview.Privacy || m.previewPrivacyEnabled || noteForced
}

func blurRenderedText(s string) string {
	var b strings.Builder
	inEsc := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if inEsc {
			b.WriteByte(ch)
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				inEsc = false
			}
			continue
		}

		if ch == 0x1b {
			inEsc = true
			b.WriteByte(ch)
			continue
		}

		if ch == '\n' || ch == '\t' || ch == ' ' {
			b.WriteByte(ch)
			continue
		}

		b.WriteRune('•')
	}

	return b.String()
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inEsc {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				inEsc = false
			}
			continue
		}
		if ch == 0x1b {
			inEsc = true
			continue
		}
		b.WriteByte(ch)
	}

	return b.String()
}

func isUnderlineHeadingLine(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r != '─' && r != '═' {
			return false
		}
	}
	return true
}

func renderTagsHeader(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	chips := make([]string, 0, len(tags))
	for _, t := range tags {
		chips = append(chips, lipgloss.NewStyle().
			Foreground(accentColor).
			Background(chipBgColor).
			Padding(0, 1).
			Render(t))
	}

	return strings.Join(chips, " ")
}

func (m Model) renderNotePreview(relPath string, raw string, tags []string) string {
	body := notes.StripFrontMatter(raw)
	rendered := m.renderPreviewMarkdown(relPath, body)

	tagsHeader := renderTagsHeader(tags)
	if tagsHeader != "" {
		rendered = tagsHeader + "\n\n" + rendered
	}

	return rendered
}

type previewMatch struct {
	line      int
	occurrIdx int
}

func buildPreviewMatches(content, query string) []previewMatch {
	if query == "" {
		return nil
	}

	q := strings.ToLower(strings.TrimSpace(query))
	if strings.HasPrefix(q, "#") {
		return nil
	}

	terms := strings.Fields(q)
	if len(terms) == 0 {
		return nil
	}

	type interval struct{ start, end int }

	var matches []previewMatch
	lines := strings.Split(stripANSI(content), "\n")
	for lineIdx, line := range lines {
		lower := strings.ToLower(line)

		var intervals []interval
		for _, term := range terms {
			if term == "" {
				continue
			}
			start := 0
			for {
				idx := strings.Index(lower[start:], term)
				if idx == -1 {
					break
				}
				abs := start + idx
				intervals = append(intervals, interval{abs, abs + len(term)})
				start = abs + len(term)
			}
		}

		if len(intervals) == 0 {
			continue
		}

		sort.Slice(intervals, func(i, j int) bool {
			return intervals[i].start < intervals[j].start
		})

		merged := []interval{intervals[0]}
		for _, iv := range intervals[1:] {
			last := &merged[len(merged)-1]
			if iv.start <= last.end {
				if iv.end > last.end {
					last.end = iv.end
				}
			} else {
				merged = append(merged, iv)
			}
		}

		for occurrIdx := range merged {
			matches = append(matches, previewMatch{line: lineIdx, occurrIdx: occurrIdx})
		}
	}

	return matches
}

func applyMatchHighlights(content, query string, matches []previewMatch, activeIdx int) string {
	if query == "" {
		return content
	}

	activeMatchLine := -1
	activeOccurrIdx := -1
	if len(matches) > 0 && activeIdx >= 0 && activeIdx < len(matches) {
		activeMatchLine = matches[activeIdx].line
		activeOccurrIdx = matches[activeIdx].occurrIdx
	}

	return highlightMatchesInRendered(content, query, activeMatchLine, activeOccurrIdx)
}

func highlightMatchesInRendered(
	content, query string,
	activeMatchLine, activeOccurrIdx int,
) string {
	if query == "" {
		return content
	}

	q := strings.ToLower(strings.TrimSpace(query))
	if strings.HasPrefix(q, "#") {
		return content
	}

	terms := strings.Fields(q)
	if len(terms) == 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		plainLine := stripANSI(line)
		lowerPlain := strings.ToLower(plainLine)

		matched := false
		for _, term := range terms {
			if strings.Contains(lowerPlain, term) {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		activeOccurr := -1
		if i == activeMatchLine {
			activeOccurr = activeOccurrIdx
		}
		lines[i] = fillWidthBackground(
			highlightTermsInLine(plainLine, terms, activeOccurr),
			lipgloss.Width(plainLine),
			bgSoftColor,
		)
	}

	return strings.Join(lines, "\n")
}

func highlightTermsInLine(line string, terms []string, activeOccurrIdx int) string {
	if line == "" {
		return line
	}

	base := lipgloss.NewStyle().
		Foreground(textColor).
		Background(bgSoftColor)

	type interval struct{ start, end int }
	var intervals []interval

	lower := strings.ToLower(line)
	for _, term := range terms {
		if term == "" {
			continue
		}
		start := 0
		for {
			idx := strings.Index(lower[start:], term)
			if idx == -1 {
				break
			}
			abs := start + idx
			intervals = append(intervals, interval{abs, abs + len(term)})
			start = abs + len(term)
		}
	}

	if len(intervals) == 0 {
		return base.Render(line)
	}

	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].start < intervals[j].start
	})

	// Merge overlapping intervals.
	merged := []interval{intervals[0]}
	for _, iv := range intervals[1:] {
		last := &merged[len(merged)-1]
		if iv.start <= last.end {
			if iv.end > last.end {
				last.end = iv.end
			}
		} else {
			merged = append(merged, iv)
		}
	}

	var b strings.Builder
	pos := 0
	for occurrIdx, iv := range merged {
		if pos < iv.start {
			b.WriteString(base.Render(line[pos:iv.start]))
		}
		matchBg := highlightBgColor
		matchFg := selectedFgColor
		if occurrIdx == activeOccurrIdx {
			matchBg = accentColor
			matchFg = bgColor
		}
		b.WriteString(lipgloss.NewStyle().
			Background(matchBg).
			Foreground(matchFg).
			Render(line[iv.start:iv.end]))
		pos = iv.end
	}
	if pos < len(line) {
		b.WriteString(base.Render(line[pos:]))
	}

	return b.String()
}

func (m *Model) jumpToNextMatch() {
	if len(m.previewMatches) == 0 {
		m.status = "no matches"
		return
	}

	m.previewMatchIndex = (m.previewMatchIndex + 1) % len(m.previewMatches)
	m.scrollToMatchLine(m.previewMatches[m.previewMatchIndex].line)
	m.reapplyPreviewHighlights()
	m.status = fmt.Sprintf("match %d/%d", m.previewMatchIndex+1, len(m.previewMatches))
}

func (m *Model) jumpToPrevMatch() {
	if len(m.previewMatches) == 0 {
		m.status = "no matches"
		return
	}

	m.previewMatchIndex = (m.previewMatchIndex - 1 + len(m.previewMatches)) % len(m.previewMatches)
	m.scrollToMatchLine(m.previewMatches[m.previewMatchIndex].line)
	m.reapplyPreviewHighlights()
	m.status = fmt.Sprintf("match %d/%d", m.previewMatchIndex+1, len(m.previewMatches))
}

func (m *Model) scrollToMatchLine(line int) {
	if line < m.preview.YOffset {
		m.preview.SetYOffset(line)
	} else if line >= m.preview.YOffset+m.preview.Height {
		m.preview.SetYOffset(line - m.preview.Height + 1)
	}
}

func (m *Model) centerCurrentMatch() {
	if len(m.previewMatches) == 0 {
		m.status = "no matches"
		return
	}
	line := m.previewMatches[m.previewMatchIndex].line
	m.preview.SetYOffset(m.centeredOffset(line))
	m.status = fmt.Sprintf("match %d/%d", m.previewMatchIndex+1, len(m.previewMatches))
}

func (m *Model) reapplyPreviewHighlights() {
	query := strings.TrimSpace(m.searchInput.Value())
	highlighted := applyMatchHighlights(
		m.previewBaseContent,
		query,
		m.previewMatches,
		m.previewMatchIndex,
	)
	m.previewContent = highlighted
	m.reapplyTodoHighlight()
}

func (m Model) centeredOffset(line int) int {
	offset := line - m.preview.Height/2
	if offset < 0 {
		offset = 0
	}
	return offset
}
