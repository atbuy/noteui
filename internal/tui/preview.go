package tui

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/notes"
)

func (m *Model) setPreviewPlaceholder(pathKey, content string) {
	m.previewPath = pathKey
	m.previewBaseContent = ""
	m.previewContent = content
	m.previewPrivacyForcedByNote = false
	m.previewLineNumberStart = 0
	m.previewTodos = nil
	m.previewTodoCursor = -1
	m.pendingTodoCursor = -1
	m.pendingT = false
	m.previewTodoNavMode = false
	m.previewLinks = nil
	m.previewLinkCursor = -1
	m.previewLinkNavMode = false
	m.setPreviewViewportContent(content)
	m.rebuildPreviewHeadingsFromRendered()
	m.previewMatches = nil
	m.previewMatchIndex = 0
	m.preview.GotoTop()
}

func (m *Model) setStaticPreview(pathKey, rendered string) {
	m.previewPath = pathKey
	m.previewBaseContent = rendered
	m.previewContent = rendered
	m.previewPrivacyForcedByNote = false
	m.previewLineNumberStart = 0
	m.previewTodos = nil
	m.previewTodoCursor = -1
	m.pendingTodoCursor = -1
	m.pendingT = false
	m.previewTodoNavMode = false
	m.previewLinks = nil
	m.previewLinkCursor = -1
	m.previewLinkNavMode = false
	m.setPreviewViewportContent(rendered)
	m.rebuildPreviewHeadingsFromRendered()
	m.previewMatches = nil
	m.previewMatchIndex = 0
	m.preview.GotoTop()
}

func (m Model) notesPreviewEmptyMessage() string {
	if query := strings.TrimSpace(m.searchInput.Value()); query != "" {
		return fmt.Sprintf("No notes match %q. Press esc to clear search.", query)
	}
	return "No notes yet. Press n for a note, T for a todo, C for a category, or N for temp."
}

func (m Model) tempPreviewEmptyMessage() string {
	if query := strings.TrimSpace(m.searchInput.Value()); query != "" {
		return fmt.Sprintf("No temporary notes match %q. Press esc to clear search.", query)
	}
	return "No temporary note selected. Press N to create one or t to return to notes."
}

func (m Model) pinsPreviewEmptyMessage() string {
	if query := strings.TrimSpace(m.searchInput.Value()); query != "" {
		return fmt.Sprintf("No pinned items match %q. Press esc to clear search.", query)
	}
	return "No pinned item selected. Press p on a note or category to pin it here."
}

func (m *Model) refreshPreview() {
	if m.listMode == listModeTodos {
		item := m.currentTodoItem()
		if item == nil {
			m.setPreviewPlaceholder("", m.todosPreviewEmptyMessage())
			return
		}

		if m.previewPath == item.Note.Path {
			if m.syncSelectedTodoInPreview() {
				m.reapplyTodoHighlight()
				if m.previewTodoCursor >= 0 && m.previewTodoCursor < len(m.previewTodos) {
					m.ensurePreviewLineVisible(m.previewTodos[m.previewTodoCursor].rendLine)
				}
			} else {
				m.setPreviewViewportContent(applyTodoDueDateHints(m.previewContent))
			}
			return
		}

		rel := item.Note.RelPath
		if item.IsTemp {
			rel = filepath.Join(".tmp", rel)
		}
		m.previewPath = item.Note.Path
		m.previewMatches = nil
		m.pendingPreviewCmd = m.notePreviewCmd(item.Note.Path, rel, item.Note.Tags)
		return
	}

	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil {
			m.setPreviewPlaceholder("", m.pinsPreviewEmptyMessage())
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
			m.setStaticPreview("pinned-category:"+item.RelPath, rendered)
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
			m.setPreviewPlaceholder("", m.tempPreviewEmptyMessage())
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

	if m.visibleTreeResultCount() == 0 {
		m.setPreviewPlaceholder("", m.notesPreviewEmptyMessage())
		return
	}

	item := m.currentTreeItem()
	if item == nil {
		m.setPreviewPlaceholder("", "Nothing selected. Move with j/k or press n to create a note.")
		return
	}

	if item.Kind == treeCategory {
		var pathText string
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
		m.setStaticPreview("category:"+item.RelPath, rendered)
		return
	}

	if item.Kind == treeRemoteNote && item.RemoteNote != nil {
		pathText := filepath.Join("~/notes", filepath.FromSlash(item.RemoteNote.RelPath))
		lines := []string{
			"# " + m.remoteOnlyDisplayTitle(*item.RemoteNote),
			"",
			"This note exists on the server but is not stored locally.",
			"",
			"- Path: `" + filepath.ToSlash(item.RemoteNote.RelPath) + "`",
			"- Status: remote only",
		}
		if m.hasRemoteOnlyPathDuplicate(item.RemoteNote.RelPath) {
			lines = append(lines, "- Remote ID: `"+item.RemoteNote.ID+"`")
		}
		lines = append(lines,
			"",
			"Press `i` to import this note or `I` to import all missing synced notes.",
		)
		content := strings.Join(lines, "\n")
		rendered := m.renderPreviewMarkdown(pathText, content)
		m.setStaticPreview("remote:"+item.RemoteNote.ID, rendered)
		return
	}

	if item.Note == nil {
		m.setPreviewPlaceholder("", "No note selected. Move with j/k or press n to create a note.")
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
					baseContent: "[decryption failed: wrong passphrase?]",
				}
			}
			raw = decrypted
		}

		private := notes.NoteIsPrivate(raw)
		rendered, lineNumberStart := m.renderNotePreview(relPath, raw, tags)
		body := notes.StripFrontMatter(raw)
		if m.effectivePreviewPrivacy(private) {
			rendered = blurRenderedText(rendered)
		}
		return previewRenderedMsg{
			forPath:             notePath,
			baseContent:         rendered,
			rawContent:          body,
			privacyForcedByNote: private,
			lineNumberStart:     lineNumberStart,
			todoLineOffset:      previewBodyLineOffset(raw, body),
		}
	}
}

func (m Model) renderPreviewMarkdown(relPath, raw string) string {
	if !m.cfg.Preview.RenderMarkdown || m.previewMarkdownDisabledFor(relPath) {
		return raw
	}

	raw = notes.RewriteWikilinks(raw)

	width := m.previewContentWidth()
	opts := markdownRenderOptions{
		Width:           width,
		SyntaxHighlight: m.cfg.Preview.SyntaxHighlight,
		CodeStyle:       m.cfg.Preview.CodeStyle,
	}

	rendered := renderMarkdownTerminal(raw, opts)
	if !m.previewLineNumbersEnabled {
		return rendered
	}

	for range 2 {
		adjustedWidth := max(12, m.previewContentWidth()-previewLineNumberGutterWidth(rendered))
		if adjustedWidth == opts.Width {
			break
		}
		opts.Width = adjustedWidth
		rendered = renderMarkdownTerminal(raw, opts)
	}

	return rendered
}

func (m Model) previewContentWidth() int {
	width := m.preview.Width
	if width <= 0 {
		width = max(20, m.previewWidth-8)
	}
	if !m.previewLineNumbersEnabled {
		return width
	}
	return max(12, width-4)
}

func previewLineNumberGutterWidth(content string) int {
	lines := strings.Split(content, "\n")
	count := len(lines)
	if count == 0 {
		count = 1
	}
	return len(fmt.Sprintf("%d", count)) + 1
}

func (m Model) formatPreviewForDisplay(content string) string {
	if !m.previewLineNumbersEnabled {
		return content
	}

	lines := strings.Split(content, "\n")
	count := len(lines)
	if count == 0 {
		count = 1
	}
	start := min(max(0, m.previewLineNumberStart), count)
	numberedCount := max(1, count-start)
	digits := len(fmt.Sprintf("%d", numberedCount))
	gutter := lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(bgSoftColor)
	row := lipgloss.NewStyle().Background(bgSoftColor)
	emptyGutter := gutter.Render(strings.Repeat(" ", digits+1))

	formatted := make([]string, 0, len(lines))
	for i, line := range lines {
		label := emptyGutter
		if i >= start {
			label = gutter.Render(fmt.Sprintf("%*d ", digits, i-start+1))
		}
		formatted = append(formatted, row.Render(lipgloss.JoinHorizontal(
			lipgloss.Top,
			label,
			line,
		)))
	}

	return strings.Join(formatted, "\n")
}

func (m *Model) setPreviewViewportContent(content string) {
	m.preview.SetContent(m.formatPreviewForDisplay(content))
}

func previewBodyLineOffset(raw, body string) int {
	normalizedRaw := strings.ReplaceAll(raw, "\r\n", "\n")
	if normalizedRaw == body {
		return 0
	}
	prefixLen := len(normalizedRaw) - len(body)
	if prefixLen <= 0 || prefixLen > len(normalizedRaw) {
		return 0
	}
	return strings.Count(normalizedRaw[:prefixLen], "\n")
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

func (m *Model) rebuildPreviewTodos(raw, rendered string, rawLineOffset int) {
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
			rawLine:  rawLineOffset + rawTodos[i].lineIdx,
			rendLine: rendTodoLines[i],
			checked:  rawTodos[i].checked,
			text:     rawTodos[i].text,
		})
	}

	for i := limit; i < len(rawTodos); i++ {
		m.previewTodos = append(m.previewTodos, previewTodoItem{
			rawLine:  rawLineOffset + rawTodos[i].lineIdx,
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
	m.ensurePreviewLineVisible(todo.rendLine)
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
	m.ensurePreviewLineVisible(todo.rendLine)
	m.status = fmt.Sprintf("todo %d/%d", m.previewTodoCursor+1, len(m.previewTodos))
	m.reapplyTodoHighlight()
}

func (m *Model) jumpToFirstTodo() {
	if len(m.previewTodos) == 0 {
		m.status = "no todos"
		return
	}
	m.previewTodoCursor = 0
	todo := m.previewTodos[m.previewTodoCursor]
	m.ensurePreviewLineVisible(todo.rendLine)
	m.status = fmt.Sprintf("todo %d/%d", m.previewTodoCursor+1, len(m.previewTodos))
	m.reapplyTodoHighlight()
}

func (m *Model) jumpToLastTodo() {
	if len(m.previewTodos) == 0 {
		m.status = "no todos"
		return
	}
	m.previewTodoCursor = len(m.previewTodos) - 1
	todo := m.previewTodos[m.previewTodoCursor]
	m.ensurePreviewLineVisible(todo.rendLine)
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
	if hasANSI(lines[rendLine]) {
		lines[rendLine] = renderSelectedRenderedTodoLine(lines[rendLine])
	} else {
		lines[rendLine] = renderSelectedTodoLine(lines[rendLine])
	}
	return strings.Join(lines, "\n")
}

func renderTodoPreviewBody(text string, selected bool, bg lipgloss.Color, textFg lipgloss.Color) string {
	display, metadata := notes.ParseTodoMetadata(text)
	display = strings.TrimSpace(display)
	base := lipgloss.NewStyle().Background(bg).Foreground(textFg).Bold(selected)
	if metadata.Priority == 0 && strings.TrimSpace(metadata.DueDate) == "" {
		return base.Render(display)
	}

	parts := make([]string, 0, 3)
	if display != "" {
		parts = append(parts, base.Render(display))
	}
	if metadata.Priority > 0 {
		priority := fmt.Sprintf("[p%d]", metadata.Priority)
		parts = append(parts, lipgloss.NewStyle().Background(bg).Foreground(todoPriorityColor(metadata.Priority)).Bold(selected).Render(priority))
	}
	if metadata.DueDate != "" {
		due := "[due:" + metadata.DueDate + "]"
		parts = append(parts, lipgloss.NewStyle().Background(bg).Foreground(todoDueDateColor(metadata.DueDate, time.Now().Format("2006-01-02"))).Bold(selected).Render(due))
	}
	return strings.Join(parts, lipgloss.NewStyle().Background(bg).Render(" "))
}

func renderTodoPreviewLine(plain string, selected bool) string {
	bg := bgSoftColor
	textFg := textColor
	checkedFg := successColor
	uncheckedFg := errorColor
	if selected {
		bg = selectedBgColor
		textFg = selectedFgColor
	}

	indentWidth := len(plain) - len(strings.TrimLeft(plain, " "))
	indent := strings.Repeat(" ", indentWidth)
	body := plain[indentWidth:]
	indentPart := lipgloss.NewStyle().Background(bg).Render(indent)
	spacePart := lipgloss.NewStyle().Background(bg).Render(" ")

	switch {
	case strings.HasPrefix(body, "[X] "), strings.HasPrefix(body, "[x] "):
		rest := body[4:]
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			indentPart,
			lipgloss.NewStyle().Background(bg).Foreground(checkedFg).Bold(true).Render("[X]"),
			spacePart,
			renderTodoPreviewBody(rest, selected, bg, textFg),
		)
	case strings.HasPrefix(body, "[ ] "):
		rest := body[4:]
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			indentPart,
			lipgloss.NewStyle().Background(bg).Foreground(uncheckedFg).Bold(true).Render("[ ]"),
			spacePart,
			renderTodoPreviewBody(rest, selected, bg, textFg),
		)
	default:
		return lipgloss.NewStyle().Background(bg).Foreground(textFg).Bold(selected).Render(plain)
	}
}

func renderSelectedTodoLine(plain string) string {
	return renderTodoPreviewLine(plain, true)
}

func renderSelectedRenderedTodoLine(line string) string {
	line = replaceRenderedBackground(line, bgSoftColor, selectedBgColor)
	if strings.TrimSpace(stripANSI(line)) == "" {
		return lipgloss.NewStyle().Background(selectedBgColor).Render(stripANSI(line))
	}
	return line
}

func replaceRenderedBackground(line string, from, to lipgloss.Color) string {
	fromParam, ok := ansiBackgroundParam(from)
	if !ok {
		return line
	}
	toParam, ok := ansiBackgroundParam(to)
	if !ok {
		return line
	}
	return strings.ReplaceAll(line, fromParam, toParam)
}

func ansiBackgroundParam(color lipgloss.Color) (string, bool) {
	rgb, ok := parseHexColor(string(color))
	if !ok {
		return "", false
	}
	return fmt.Sprintf("48;2;%d;%d;%d", clampChannel(rgb.r), clampChannel(rgb.g), clampChannel(rgb.b)), true
}

func hasANSI(s string) bool {
	return strings.Contains(s, "\x1b[")
}

func applyTodoMetadataHighlights(content string) string {
	if content == "" {
		return content
	}
	base := lipgloss.NewStyle().Foreground(textColor).Background(bgSoftColor)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = renderTodoMetadataHintsInRenderedLine(line, base)
	}
	return strings.Join(lines, "\n")
}

func renderTodoMetadataHintsInRenderedLine(line string, base lipgloss.Style) string {
	plain, chars := renderedVisibleByteRanges(line)
	if plain == "" {
		return line
	}

	today := time.Now().Format("2006-01-02")
	type interval struct {
		start int
		end   int
		style lipgloss.Style
	}
	var intervals []interval
	for offset := 0; offset < len(plain); {
		start, end, style, ok := nextTodoMetadataHint(plain[offset:], base, today)
		if !ok {
			break
		}
		intervals = append(intervals, interval{
			start: offset + start,
			end:   offset + end,
			style: style,
		})
		offset += end
	}
	if len(intervals) == 0 {
		return line
	}

	var out strings.Builder
	lastByte := 0
	for _, item := range intervals {
		if item.start < 0 || item.end > len(chars) || item.start >= item.end {
			continue
		}
		byteStart := chars[item.start].start
		byteEnd := chars[item.end-1].end
		writeRenderedGapWithBackground(&out, line[lastByte:byteStart], base)
		out.WriteString(item.style.Render(plain[item.start:item.end]))
		lastByte = byteEnd
	}
	writeRenderedGapWithBackground(&out, line[lastByte:], base)
	return out.String()
}

type renderedByteRange struct {
	start int
	end   int
}

func writeRenderedGapWithBackground(out *strings.Builder, gap string, base lipgloss.Style) {
	plain := stripANSI(gap)
	if plain != "" && strings.TrimSpace(plain) == "" {
		out.WriteString(base.Render(plain))
		return
	}
	out.WriteString(gap)
}

func renderedVisibleByteRanges(s string) (string, []renderedByteRange) {
	var plain strings.Builder
	var chars []renderedByteRange
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
		plain.WriteByte(ch)
		chars = append(chars, renderedByteRange{start: i, end: i + 1})
	}

	return plain.String(), chars
}

func applyTodoDueDateHints(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if hasANSI(line) {
			continue
		}
		plain := stripANSI(line)
		trimmed := strings.TrimSpace(plain)
		if strings.HasPrefix(trimmed, "[ ] ") || strings.HasPrefix(trimmed, "[X] ") || strings.HasPrefix(trimmed, "[x] ") {
			lines[i] = renderTodoPreviewLine(plain, false)
		}
	}
	return strings.Join(lines, "\n")
}

func (m *Model) reapplyTodoHighlight() {
	content := applyTodoDueDateHints(m.previewContent)
	if !m.previewTodoNavMode {
		m.setPreviewViewportContent(content)
		return
	}
	if len(m.previewTodos) == 0 || m.previewTodoCursor < 0 ||
		m.previewTodoCursor >= len(m.previewTodos) {
		m.setPreviewViewportContent(content)
		return
	}
	todo := m.previewTodos[m.previewTodoCursor]
	m.setPreviewViewportContent(applyTodoLineHighlight(content, todo.rendLine))
}

var (
	renderedWikilinkRe     = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	renderedExternalLinkRe = regexp.MustCompile(`\((https?://[^)\s]+)\)`)
)

func (m *Model) rebuildPreviewLinks() {
	m.previewLinks = m.previewLinks[:0]
	lines := strings.Split(stripANSI(m.previewContent), "\n")
	for i, line := range lines {
		for _, idx := range renderedWikilinkRe.FindAllStringSubmatchIndex(line, -1) {
			m.previewLinks = append(m.previewLinks, previewLinkItem{
				rendLine:   i,
				rendCol:    idx[0],
				matchLen:   idx[1] - idx[0],
				target:     line[idx[2]:idx[3]],
				isWikilink: true,
			})
		}
		for _, idx := range renderedExternalLinkRe.FindAllStringSubmatchIndex(line, -1) {
			m.previewLinks = append(m.previewLinks, previewLinkItem{
				rendLine:   i,
				rendCol:    idx[0],
				matchLen:   idx[1] - idx[0],
				target:     line[idx[2]:idx[3]],
				isWikilink: false,
			})
		}
	}
	// Sort by line, then by column within the line, so navigation is in reading order.
	sort.Slice(m.previewLinks, func(a, b int) bool {
		la, lb := m.previewLinks[a], m.previewLinks[b]
		if la.rendLine != lb.rendLine {
			return la.rendLine < lb.rendLine
		}
		return la.rendCol < lb.rendCol
	})
	if m.previewLinkCursor >= len(m.previewLinks) {
		m.previewLinkCursor = max(0, len(m.previewLinks)-1)
	}
	if !m.previewLinkNavMode {
		m.previewLinkCursor = -1
	}
}

func linkStatusLabel(link previewLinkItem) string {
	if link.isWikilink {
		return "[[" + link.target + "]]"
	}
	return link.target
}

func (m *Model) reapplyLinkHighlight() {
	content := applyTodoDueDateHints(m.previewContent)
	if !m.previewLinkNavMode || len(m.previewLinks) == 0 ||
		m.previewLinkCursor < 0 || m.previewLinkCursor >= len(m.previewLinks) {
		m.setPreviewViewportContent(content)
		return
	}
	link := m.previewLinks[m.previewLinkCursor]
	m.setPreviewViewportContent(applyLinkSpanHighlight(content, link))
}

// applyLinkSpanHighlight highlights only the link's own text within its line,
// leaving the rest of the line (and all other lines) unchanged.
func applyLinkSpanHighlight(content string, link previewLinkItem) string {
	if link.rendLine < 0 || link.matchLen <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	if link.rendLine >= len(lines) {
		return content
	}
	line := lines[link.rendLine]

	// Map visible-character byte positions in the stripped text to byte
	// positions in the (potentially ANSI-decorated) line.
	plain, chars := renderedVisibleByteRanges(line)

	end := link.rendCol + link.matchLen
	if link.rendCol < 0 || end > len(plain) || end > len(chars) {
		// Out-of-range: nothing to highlight.
		return content
	}

	byteStart := chars[link.rendCol].start
	byteEnd := chars[end-1].end
	spanText := plain[link.rendCol:end]

	highlighted := lipgloss.NewStyle().
		Background(selectedBgColor).
		Foreground(selectedFgColor).
		Bold(true).
		Render(spanText)

	lines[link.rendLine] = line[:byteStart] + highlighted + line[byteEnd:]
	return strings.Join(lines, "\n")
}

func (m *Model) jumpToNextLink() {
	if len(m.previewLinks) == 0 {
		m.status = "no links"
		return
	}
	if m.previewLinkCursor < 0 {
		m.previewLinkCursor = 0
	} else {
		m.previewLinkCursor = (m.previewLinkCursor + 1) % len(m.previewLinks)
	}
	link := m.previewLinks[m.previewLinkCursor]
	m.ensurePreviewLineVisible(link.rendLine)
	m.status = fmt.Sprintf("link %d/%d: %s", m.previewLinkCursor+1, len(m.previewLinks), linkStatusLabel(link))
	m.reapplyLinkHighlight()
}

func (m *Model) jumpToPrevLink() {
	if len(m.previewLinks) == 0 {
		m.status = "no links"
		return
	}
	if m.previewLinkCursor < 0 {
		m.previewLinkCursor = len(m.previewLinks) - 1
	} else {
		m.previewLinkCursor = (m.previewLinkCursor - 1 + len(m.previewLinks)) % len(m.previewLinks)
	}
	link := m.previewLinks[m.previewLinkCursor]
	m.ensurePreviewLineVisible(link.rendLine)
	m.status = fmt.Sprintf("link %d/%d: %s", m.previewLinkCursor+1, len(m.previewLinks), linkStatusLabel(link))
	m.reapplyLinkHighlight()
}

func (m *Model) jumpToFirstLink() {
	if len(m.previewLinks) == 0 {
		m.status = "no links"
		return
	}
	m.previewLinkCursor = 0
	link := m.previewLinks[m.previewLinkCursor]
	m.ensurePreviewLineVisible(link.rendLine)
	m.status = fmt.Sprintf("link %d/%d: %s", m.previewLinkCursor+1, len(m.previewLinks), linkStatusLabel(link))
	m.reapplyLinkHighlight()
}

func (m *Model) jumpToLastLink() {
	if len(m.previewLinks) == 0 {
		m.status = "no links"
		return
	}
	m.previewLinkCursor = len(m.previewLinks) - 1
	link := m.previewLinks[m.previewLinkCursor]
	m.ensurePreviewLineVisible(link.rendLine)
	m.status = fmt.Sprintf("link %d/%d: %s", m.previewLinkCursor+1, len(m.previewLinks), linkStatusLabel(link))
	m.reapplyLinkHighlight()
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

	sep := lipgloss.NewStyle().Background(bgSoftColor).Render(" ")
	return strings.Join(chips, sep)
}

func (m Model) renderNotePreview(relPath string, raw string, tags []string) (string, int) {
	body := notes.StripFrontMatter(raw)
	rendered := m.renderPreviewMarkdown(relPath, body)
	lineNumberStart := 0

	if syncSummary := m.syncIssuePreviewMarkdown(relPath); syncSummary != "" {
		summaryRendered := m.renderPreviewMarkdown(relPath, syncSummary)
		rendered = summaryRendered + "\n\n" + rendered
		lineNumberStart += strings.Count(stripANSI(summaryRendered), "\n") + 2
	}

	tagsHeader := renderTagsHeader(tags)
	if tagsHeader != "" {
		rendered = tagsHeader + "\n\n" + rendered
		lineNumberStart += 2
	}

	rendered = applyTodoMetadataHighlights(rendered)

	return rendered, lineNumberStart
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
			highlightTermsInLine(plainLine, terms, activeOccurr, previewLineForeground(plainLine)),
			lipgloss.Width(plainLine),
			bgSoftColor,
		)
	}

	return strings.Join(lines, "\n")
}

func previewLineForeground(line string) lipgloss.Color {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "~/notes") {
		return accentColor
	}
	return textColor
}

func highlightTermsInLine(
	line string,
	terms []string,
	activeOccurrIdx int,
	fg lipgloss.Color,
) string {
	if line == "" {
		return line
	}

	base := lipgloss.NewStyle().
		Foreground(fg).
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
	m.syncTodoCursorToActiveMatch()
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
	m.syncTodoCursorToActiveMatch()
	m.reapplyPreviewHighlights()
	m.status = fmt.Sprintf("match %d/%d", m.previewMatchIndex+1, len(m.previewMatches))
}

func (m *Model) ensurePreviewLineVisible(line int) {
	if line < 0 {
		return
	}
	if line < m.preview.YOffset {
		m.preview.SetYOffset(line)
	} else if line >= m.preview.YOffset+m.preview.Height {
		m.preview.SetYOffset(line - m.preview.Height + 1)
	}
}

func (m *Model) scrollToMatchLine(line int) {
	m.ensurePreviewLineVisible(line)
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
	if m.previewLinkNavMode {
		m.reapplyLinkHighlight()
	} else {
		m.reapplyTodoHighlight()
	}
}

func (m *Model) syncTodoCursorToActiveMatch() {
	if !m.previewTodoNavMode || len(m.previewMatches) == 0 || m.previewMatchIndex < 0 ||
		m.previewMatchIndex >= len(m.previewMatches) {
		return
	}
	matchLine := m.previewMatches[m.previewMatchIndex].line
	for i, todo := range m.previewTodos {
		if todo.rendLine == matchLine {
			m.previewTodoCursor = i
			return
		}
	}
}

func (m Model) centeredOffset(line int) int {
	offset := line - m.preview.Height/2
	if offset < 0 {
		offset = 0
	}
	return offset
}
