package tui

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	gmtext "github.com/yuin/goldmark/text"
)

type markdownRenderOptions struct {
	Width           int
	SyntaxHighlight bool
	CodeStyle       string
}

type markdownPreviewRenderer struct {
	source []byte
	width  int
	opts   markdownRenderOptions
}

func renderMarkdownTerminal(raw string, opts markdownRenderOptions) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
		),
	)

	source := []byte(raw)
	doc := md.Parser().Parse(gmtext.NewReader(source))

	r := markdownPreviewRenderer{
		source: source,
		width:  max(20, opts.Width),
		opts:   opts,
	}

	out := r.renderBlocks(doc, 0)
	return strings.TrimSpace(out)
}

func (r markdownPreviewRenderer) renderBlocks(parent ast.Node, indent int) string {
	var parts []string
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		rendered := strings.TrimRight(r.renderBlock(child, indent), "\n")
		if strings.TrimSpace(rendered) == "" {
			continue
		}
		parts = append(parts, rendered)
	}
	blankLine := lipgloss.NewStyle().Background(bgSoftColor).Width(r.width).Render("")
	return strings.Join(parts, "\n"+blankLine+"\n")
}

func (r markdownPreviewRenderer) renderBlock(node ast.Node, indent int) string {
	switch n := node.(type) {
	case *ast.Heading:
		text := strings.TrimSpace(r.renderInlineChildren(n))
		if text == "" {
			return ""
		}
		return r.renderHeading(text, n.Level, indent)

	case *ast.Paragraph:
		text := strings.TrimSpace(r.renderInlineChildren(n))
		if text == "" {
			return ""
		}
		return r.wrap(text, indent)

	case *ast.TextBlock:
		text := strings.TrimSpace(r.renderInlineChildren(n))
		if text == "" {
			return ""
		}
		return r.wrap(text, indent)

	case *ast.FencedCodeBlock:
		return r.renderCodeBlock(n, indent)

	case *ast.CodeBlock:
		code := strings.TrimRight(r.linesText(n.Lines()), "\n")
		if code == "" {
			return ""
		}
		return r.renderPlainCode(code, "", indent)

	case *ast.Blockquote:
		body := r.renderBlocks(n, indent)
		if strings.TrimSpace(body) == "" {
			return ""
		}
		return prefixLines(body, strings.Repeat(" ", indent)+"│ ")

	case *ast.List:
		return r.renderList(n, indent)

	case *ast.ListItem:
		return r.renderListItem(n, 0, false, indent)

	case *ast.ThematicBreak:
		line := lipgloss.NewStyle().
			Foreground(borderColor).
			Background(bgSoftColor).
			Width(max(8, r.width-indent)).
			Render(strings.Repeat("─", max(8, r.width-indent)))
		return prefixLines(line, strings.Repeat(" ", indent))

	case *ast.HTMLBlock:
		text := strings.TrimSpace(string(n.Lines().Value(r.source)))
		if text == "" {
			return ""
		}
		return prefixLines(
			mutedStyle.Render("[html block omitted]"),
			strings.Repeat(" ", indent),
		)

	default:
		if node.FirstChild() != nil {
			return r.renderBlocks(node, indent)
		}
		text := strings.TrimSpace(string(nodeText(node, r.source)))
		if text == "" {
			return ""
		}
		return r.wrap(text, indent)
	}
}

func (r markdownPreviewRenderer) renderHeading(text string, level int, indent int) string {
	base := strings.Repeat(" ", indent)

	switch level {
	case 1:
		return prefixLines(
			lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor).
				Background(bgSoftColor).
				Width(max(10, r.width-indent)).
				Render(text),
			base,
		)
	case 2:
		underline := strings.Repeat("─", max(3, lipgloss.Width(text)))
		block := lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().
				Bold(true).
				Foreground(textColor).
				Background(bgSoftColor).
				Width(max(10, r.width-indent)).
				Render(text),
			lipgloss.NewStyle().
				Foreground(accentSoftColor).
				Background(bgSoftColor).
				Width(max(10, r.width-indent)).
				Render(underline),
		)
		return prefixLines(block, base)
	case 3:
		return prefixLines(
			lipgloss.NewStyle().
				Bold(true).
				Foreground(accentSoftColor).
				Background(bgSoftColor).
				Width(max(10, r.width-indent)).
				Render(text),
			base,
		)
	default:
		return prefixLines(
			lipgloss.NewStyle().
				Bold(true).
				Foreground(textColor).
				Background(bgSoftColor).
				Width(max(10, r.width-indent)).
				Render(text),
			base,
		)
	}
}

func (r markdownPreviewRenderer) renderList(n *ast.List, indent int) string {
	var parts []string
	ordered := n.IsOrdered()

	i := 0
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		item, ok := child.(*ast.ListItem)
		if !ok {
			continue
		}
		parts = append(parts, r.renderListItem(item, i, ordered, indent))
		i++
	}

	return strings.Join(parts, "\n")
}

func (r markdownPreviewRenderer) renderListItem(
	item *ast.ListItem,
	index int,
	ordered bool,
	indent int,
) string {
	marker := lipgloss.NewStyle().
		Background(bgSoftColor).
		Render("• ")
	markerWidth := lipgloss.Width(stripANSI(marker))
	if ordered {
		marker = lipgloss.NewStyle().
			Background(bgSoftColor).
			Render(fmt.Sprintf("%d. ", index+1))
		markerWidth = lipgloss.Width(stripANSI(marker))
	}

	if !ordered {
		if firstBlock := item.FirstChild(); firstBlock != nil {
			if firstInline := firstBlock.FirstChild(); firstInline != nil {
				if cb, ok := firstInline.(*extast.TaskCheckBox); ok {
					if cb.IsChecked {
						marker = lipgloss.NewStyle().
							Foreground(successColor).
							Background(bgSoftColor).
							Render("[X]") + lipgloss.NewStyle().Background(bgSoftColor).Render(" ")
					} else {
						marker = lipgloss.NewStyle().
							Foreground(errorColor).
							Background(bgSoftColor).
							Render("[ ]") + lipgloss.NewStyle().Background(bgSoftColor).Render(" ")
					}
					markerWidth = 4
				}
			}
		}
	}

	itemIndent := indent + markerWidth
	var blocks []string

	// Render all blocks at itemIndent so every line (including wrapped
	// continuation lines) gets the correct width and background colour from
	// prefixLines / wrap. We then replace the ANSI-encoded indent prefix on
	// the very first line with the actual marker.
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		rendered := strings.TrimRight(r.renderBlock(child, itemIndent), "\n")
		if strings.TrimSpace(rendered) == "" {
			continue
		}
		blocks = append(blocks, rendered)
	}

	if len(blocks) == 0 {
		return strings.Repeat(" ", indent) + marker
	}

	first := blocks[0]
	rest := blocks[1:]

	// prefixLines produces `lipgloss.NewStyle().Background(bgSoftColor).Render(prefix)`
	// at the start of every line. Reconstruct that exact styled prefix and
	// replace it once on the first line with indent+marker.
	styledItemPrefix := lipgloss.NewStyle().Background(bgSoftColor).Render(strings.Repeat(" ", itemIndent))
	styledIndentPrefix := lipgloss.NewStyle().Background(bgSoftColor).Render(strings.Repeat(" ", indent))
	firstLines := strings.Split(first, "\n")
	firstLines[0] = strings.Replace(firstLines[0], styledItemPrefix, styledIndentPrefix+marker, 1)
	first = strings.Join(firstLines, "\n")

	if len(rest) == 0 {
		return first
	}

	return first + "\n" + strings.Join(rest, "\n")
}

func (r markdownPreviewRenderer) renderCodeBlock(n *ast.FencedCodeBlock, indent int) string {
	code := strings.TrimRight(r.linesText(n.Lines()), "\n")
	if code == "" {
		return ""
	}

	lang := ""
	if n.Info != nil {
		lang = strings.TrimSpace(string(n.Info.Segment.Value(r.source)))
		if fields := strings.Fields(lang); len(fields) > 0 {
			lang = fields[0]
		}
	}

	if r.opts.SyntaxHighlight {
		if highlighted, ok := r.highlightCode(code, lang); ok {
			return r.renderHighlightedCode(highlighted, lang, indent)
		}
	}

	return r.renderPlainCode(code, lang, indent)
}

func (r markdownPreviewRenderer) highlightCode(code, lang string) (string, bool) {
	var lexer chroma.Lexer
	if lang != "" {
		lexer = lexers.Get(lang)
	}
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	styleName := strings.TrimSpace(r.opts.CodeStyle)
	if styleName == "" {
		styleName = "monokai"
	}
	style := chromastyles.Get(styleName)
	if style == nil {
		style = chromastyles.Get("monokai")
	}
	if style == nil {
		return "", false
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return "", false
	}

	formatter := formatters.TTY16m
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return "", false
	}

	return strings.TrimRight(buf.String(), "\n"), true
}

func (r markdownPreviewRenderer) renderHighlightedCode(code, lang string, indent int) string {
	width := max(10, r.width-indent)

	var header string
	if lang != "" {
		header = lipgloss.NewStyle().
			Foreground(accentSoftColor).
			Render("[" + lang + "]")
	}

	blockParts := []string{}
	if header != "" {
		blockParts = append(blockParts, header)
	}
	blockParts = append(blockParts, code)

	block := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, blockParts...))

	return prefixLines(block, strings.Repeat(" ", indent))
}

func (r markdownPreviewRenderer) renderPlainCode(code, lang string, indent int) string {
	width := max(10, r.width-indent)

	var header string
	if lang != "" {
		header = lipgloss.NewStyle().
			Foreground(accentSoftColor).
			Render("[" + lang + "]")
	}

	body := lipgloss.NewStyle().
		Foreground(textColor).
		Width(width).
		Render(code)

	blockParts := []string{}
	if header != "" {
		blockParts = append(blockParts, header)
	}
	blockParts = append(blockParts, body)

	block := lipgloss.NewStyle().
		Foreground(textColor).
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, blockParts...))

	return prefixLines(block, strings.Repeat(" ", indent))
}

func renderTodoMetadataHintText(text string, base lipgloss.Style) string {
	if text == "" {
		return ""
	}

	var out strings.Builder
	today := time.Now().Format("2006-01-02")

	for len(text) > 0 {
		start, end, style, ok := nextTodoMetadataHint(text, base, today)
		if !ok {
			out.WriteString(base.Render(text))
			break
		}
		if start > 0 {
			out.WriteString(base.Render(text[:start]))
		}
		out.WriteString(style.Render(text[start:end]))
		text = text[end:]
	}

	return out.String()
}

func nextTodoMetadataHint(text string, base lipgloss.Style, today string) (int, int, lipgloss.Style, bool) {
	lower := strings.ToLower(text)
	bestStart := -1
	bestEnd := 0
	bestStyle := base
	setBest := func(start, end int, style lipgloss.Style) {
		if bestStart < 0 || start < bestStart {
			bestStart = start
			bestEnd = end
			bestStyle = style
		}
	}

	if dueStart := strings.Index(lower, "[due:"); dueStart >= 0 {
		if endRel := strings.IndexByte(lower[dueStart:], ']'); endRel >= 0 {
			end := dueStart + endRel + 1
			token := lower[dueStart:end]
			due := strings.TrimSuffix(strings.TrimPrefix(token, "[due:"), "]")
			setBest(dueStart, end, base.Foreground(todoDueDateColor(due, today)))
		}
	}

	for offset := 0; offset < len(lower); {
		rel := strings.Index(lower[offset:], "[p")
		if rel < 0 {
			break
		}
		start := offset + rel
		endRel := strings.IndexByte(lower[start:], ']')
		if endRel < 0 {
			break
		}
		end := start + endRel + 1
		if priority, ok := parseTodoPriorityHintToken(lower[start:end]); ok {
			setBest(start, end, base.Foreground(todoPriorityColor(priority)))
		}
		offset = start + 2
	}

	if bestStart < 0 {
		return 0, 0, lipgloss.Style{}, false
	}
	return bestStart, bestEnd, bestStyle, true
}

func todoDueDateColor(due, today string) lipgloss.Color {
	if len(due) == len("2006-01-02") {
		if _, err := time.Parse("2006-01-02", due); err == nil && due < today {
			return errorColor
		}
	}
	return mutedColor
}

func parseTodoPriorityHintToken(token string) (int, bool) {
	if !strings.HasPrefix(token, "[p") || !strings.HasSuffix(token, "]") {
		return 0, false
	}
	digits := token[2 : len(token)-1]
	if digits == "" {
		return 0, false
	}
	priority := 0
	for i := 0; i < len(digits); i++ {
		if digits[i] < 48 || digits[i] > 57 {
			return 0, false
		}
		priority = priority*10 + int(digits[i]-48)
	}
	if priority <= 0 {
		return 0, false
	}
	return priority, true
}

func todoPriorityColor(priority int) lipgloss.Color {
	switch {
	case priority <= 1:
		return errorColor
	case priority == 2:
		return accentColor
	case priority == 3:
		return accentSoftColor
	default:
		return mutedColor
	}
}

func (r markdownPreviewRenderer) renderInlineChildren(parent ast.Node) string {
	var b strings.Builder
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		b.WriteString(r.renderInlineNode(child))
	}
	return b.String()
}

func (r markdownPreviewRenderer) renderInlineNode(node ast.Node) string {
	switch n := node.(type) {
	case *ast.Text:
		s := string(n.Segment.Value(r.source))
		base := lipgloss.NewStyle().Foreground(textColor).Background(bgSoftColor)
		switch {
		case n.HardLineBreak():
			return renderTodoMetadataHintText(s, base) + "\n"
		case n.SoftLineBreak():
			return renderTodoMetadataHintText(s+" ", base)
		default:
			return renderTodoMetadataHintText(s, base)
		}

	case *ast.String:
		return renderTodoMetadataHintText(string(n.Value), lipgloss.NewStyle().
			Foreground(textColor).
			Background(bgSoftColor))

	case *ast.Emphasis:
		content := r.renderInlineChildren(n)
		if n.Level == 2 {
			return lipgloss.NewStyle().
				Bold(true).
				Foreground(textColor).
				Background(bgSoftColor).
				Render(content)
		}
		return lipgloss.NewStyle().
			Italic(true).
			Foreground(textColor).
			Background(bgSoftColor).
			Render(content)

	case *ast.CodeSpan:
		content := strings.TrimSpace(string(nodeText(n, r.source)))
		return lipgloss.NewStyle().
			Foreground(accentSoftColor).
			Background(inlineCodeBgColor).
			Render(content)

	case *ast.Link:
		label := strings.TrimSpace(r.renderInlineChildren(n))
		if label == "" {
			label = strings.TrimSpace(string(nodeText(n, r.source)))
		}
		dest := strings.TrimSpace(string(n.Destination))

		styled := lipgloss.NewStyle().
			Underline(true).
			Foreground(accentColor).
			Background(bgSoftColor).
			Render(label)

		if dest != "" && dest != label {
			return styled + lipgloss.NewStyle().
				Foreground(mutedColor).
				Background(bgSoftColor).
				Render(" ("+dest+")")
		}
		return styled

	case *ast.AutoLink:
		text := strings.TrimSpace(string(n.Label(r.source)))
		return lipgloss.NewStyle().
			Underline(true).
			Foreground(accentColor).
			Background(bgSoftColor).
			Render(text)

	case *ast.Image:
		alt := strings.TrimSpace(r.renderInlineChildren(n))
		if alt == "" {
			alt = "image"
		}
		return lipgloss.NewStyle().
			Foreground(mutedColor).
			Background(bgSoftColor).
			Render("[image: " + alt + "]")

	case *extast.TaskCheckBox:
		return ""

	default:
		if node.FirstChild() != nil {
			return r.renderInlineChildren(node)
		}
		return lipgloss.NewStyle().
			Foreground(textColor).
			Background(bgSoftColor).
			Render(string(nodeText(node, r.source)))
	}
}

// nodeText collects the text content of a goldmark AST node without using
// the deprecated Node.Text method. For ast.Text leaf nodes it reads the
// segment value directly; for block nodes with Lines it uses Lines.Value;
// for other container nodes it recurses into children.
func nodeText(node ast.Node, source []byte) []byte {
	if tn, ok := node.(*ast.Text); ok {
		return tn.Segment.Value(source)
	}
	if node.Type() == ast.TypeBlock {
		if lines := node.Lines(); lines != nil && lines.Len() > 0 {
			return lines.Value(source)
		}
	}
	var buf bytes.Buffer
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		buf.Write(nodeText(c, source))
	}
	return buf.Bytes()
}

func (r markdownPreviewRenderer) linesText(lines *gmtext.Segments) string {
	if lines == nil || lines.Len() == 0 {
		return ""
	}

	parts := make([]string, 0, lines.Len())
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		parts = append(parts, string(seg.Value(r.source)))
	}
	return strings.Join(parts, "")
}

func (r markdownPreviewRenderer) wrap(text string, indent int) string {
	width := max(10, r.width-indent)
	rendered := lipgloss.NewStyle().
		Width(width).
		Foreground(textColor).
		Background(bgSoftColor).
		Render(text)

	return prefixLines(rendered, strings.Repeat(" ", indent))
}

func prefixLines(text, prefix string) string {
	prefixStyled := lipgloss.NewStyle().Background(bgSoftColor).Render(prefix)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = prefixStyled
		} else {
			lines[i] = prefixStyled + line
		}
	}
	return strings.Join(lines, "\n")
}
