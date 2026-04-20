package tui

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	ansi "github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	gmtext "github.com/yuin/goldmark/text"

	"atbuy/noteui/internal/notes"
)

const wikilinkURLPrefix = notes.WikilinkURLPrefix

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

type renderedMarkdownLink struct {
	target     string
	isWikilink bool
	showTarget bool
	label      string
}

type renderedBlockKind int

const (
	renderedKindUnknown renderedBlockKind = iota
	renderedKindHeading
	renderedKindParagraph
	renderedKindCode
	renderedKindList
	renderedKindBlockquote
	renderedKindThematic
)

// RenderedBlock describes a top-level markdown block: its kind, its line range
// in the raw source, and its line range in the rendered output.
type RenderedBlock struct {
	Kind        renderedBlockKind
	SourceLine  int // 0-indexed start line in the raw (body-only) source
	SourceCount int // number of source lines this block spans
	VisualStart int // 0-indexed start line in RenderedDoc.Lines
	VisualEnd   int // exclusive end line in RenderedDoc.Lines
	Level       int // heading level 1-6; 0 for non-headings
	IsTask      bool
	TaskChecked bool
}

// RenderedDoc is the output of renderMarkdownDoc: styled terminal lines plus
// structural metadata that maps back to source positions.
type RenderedDoc struct {
	Lines            []string
	Blocks           []RenderedBlock
	VisualLineSource []int // maps each visual line index to a source line (-1 for blank separators)
}

// renderMarkdownDoc renders raw markdown (body only, no frontmatter) and also
// returns block-level metadata so an editor can map rendered lines back to
// source line ranges.
func renderMarkdownDoc(raw string, opts markdownRenderOptions) RenderedDoc {
	if strings.TrimSpace(raw) == "" {
		return RenderedDoc{}
	}

	rewritten := notes.RewriteWikilinks(raw)
	source := []byte(rewritten)

	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	doc := md.Parser().Parse(gmtext.NewReader(source))

	r := markdownPreviewRenderer{
		source: source,
		width:  max(20, opts.Width),
		opts:   opts,
	}

	type blockItem struct {
		rendered      string
		kind          renderedBlockKind
		level         int
		srcLine       int
		srcCount      int
		tightWithPrev bool // no blank separator before this item (same list)
	}

	var items []blockItem
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if listNode, ok := child.(*ast.List); ok {
			// Expand each list item into its own block so j/k navigates per item.
			ordered := listNode.IsOrdered()
			idx := 0
			first := true
			for item := listNode.FirstChild(); item != nil; item = item.NextSibling() {
				listItem, ok := item.(*ast.ListItem)
				if !ok {
					continue
				}
				rendered := strings.TrimRight(r.renderListItem(listItem, idx, ordered, 0), "\n")
				if strings.TrimSpace(rendered) == "" {
					idx++
					continue
				}
				srcLine, srcCount := renderedBlockSourceRange(listItem, source)
				items = append(items, blockItem{
					rendered:      rendered,
					kind:          renderedKindList,
					srcLine:       srcLine,
					srcCount:      srcCount,
					tightWithPrev: !first,
				})
				first = false
				idx++
			}
			continue
		}
		rendered := strings.TrimRight(r.renderBlock(child, 0), "\n")
		if strings.TrimSpace(rendered) == "" {
			continue
		}
		kind, level := classifyRenderedBlock(child)
		srcLine, srcCount := renderedBlockSourceRange(child, source)
		items = append(items, blockItem{
			rendered: rendered,
			kind:     kind,
			level:    level,
			srcLine:  srcLine,
			srcCount: srcCount,
		})
	}

	if len(items) == 0 {
		return RenderedDoc{}
	}

	blankLine := lipgloss.NewStyle().Background(bgSoftColor).Width(opts.Width).Render("")
	var parts []string
	blocks := make([]RenderedBlock, len(items))
	lineOffset := 0

	for i, it := range items {
		if i > 0 && !it.tightWithPrev {
			parts = append(parts, blankLine)
			lineOffset++ // blank separator before this block
		}
		lineCount := strings.Count(it.rendered, "\n") + 1
		blocks[i] = RenderedBlock{
			Kind:        it.kind,
			Level:       it.level,
			SourceLine:  it.srcLine,
			SourceCount: it.srcCount,
			VisualStart: lineOffset,
			VisualEnd:   lineOffset + lineCount,
		}
		parts = append(parts, it.rendered)
		lineOffset += lineCount
	}

	text := strings.Join(parts, "\n")
	lines := strings.Split(text, "\n")

	// Build VisualLineSource: maps each visual line to its source line (-1 for blank separators).
	vls := make([]int, len(lines))
	for i := range vls {
		vls[i] = -1
	}
	for _, b := range blocks {
		for vi := b.VisualStart; vi < b.VisualEnd && vi < len(vls); vi++ {
			vls[vi] = b.SourceLine
		}
	}

	return RenderedDoc{
		Lines:            lines,
		Blocks:           blocks,
		VisualLineSource: vls,
	}
}

func classifyRenderedBlock(node ast.Node) (renderedBlockKind, int) {
	switch n := node.(type) {
	case *ast.Heading:
		return renderedKindHeading, n.Level
	case *ast.Paragraph, *ast.TextBlock:
		return renderedKindParagraph, 0
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		return renderedKindCode, 0
	case *ast.List:
		return renderedKindList, 0
	case *ast.Blockquote:
		return renderedKindBlockquote, 0
	case *ast.ThematicBreak:
		return renderedKindThematic, 0
	default:
		_ = n
		return renderedKindUnknown, 0
	}
}

// renderedBlockSourceRange computes the 0-indexed start line and line count for
// a goldmark AST node relative to source. The byte offsets in source come from
// the wikilink-rewritten content, but because rewriting never adds or removes
// newlines the line numbers are identical in the original source.
func renderedBlockSourceRange(node ast.Node, source []byte) (startLine, lineCount int) {
	lines := node.Lines()
	if lines != nil && lines.Len() > 0 {
		firstSeg := lines.At(0)
		lastSeg := lines.At(lines.Len() - 1)

		start := bytes.Count(source[:firstSeg.Start], []byte("\n"))

		stopByte := lastSeg.Stop
		if stopByte > len(source) {
			stopByte = len(source)
		}
		var end int
		if stopByte > 0 {
			// Stop is exclusive; Stop-1 is the last byte of the segment.
			end = bytes.Count(source[:stopByte], []byte("\n"))
			if stopByte > 0 && source[stopByte-1] == '\n' {
				// The trailing newline is a line separator, not a new line of content.
				end--
			}
		} else {
			end = start
		}

		if _, ok := node.(*ast.FencedCodeBlock); ok {
			// Include the opening and closing ``` fence lines.
			if start > 0 {
				start--
			}
			end++
			maxLine := bytes.Count(source, []byte("\n"))
			if end > maxLine {
				end = maxLine
			}
		}

		if end < start {
			end = start
		}
		return start, end - start + 1
	}

	// Container node (List, Blockquote, …): derive range from children.
	first := -1
	last := 0
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		sl, sc := renderedBlockSourceRange(child, source)
		if sc <= 0 {
			continue
		}
		if first < 0 || sl < first {
			first = sl
		}
		if sl+sc > last {
			last = sl + sc
		}
	}
	if first < 0 {
		return 0, 0
	}
	return first, last - first
}

func extractRenderedMarkdownLinks(raw string) []renderedMarkdownLink {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	raw = notes.RewriteWikilinks(raw)
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
		),
	)
	doc := md.Parser().Parse(gmtext.NewReader([]byte(raw)))

	var links []renderedMarkdownLink
	var walk func(ast.Node)
	walk = func(node ast.Node) {
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			switch n := child.(type) {
			case *ast.Link:
				label := strings.TrimSpace(stripANSI(renderInlineLabelText(n, []byte(raw))))
				if label == "" {
					label = strings.TrimSpace(string(nodeText(n, []byte(raw))))
				}
				dest := strings.TrimSpace(string(n.Destination))
				item := renderedMarkdownLink{label: label}
				if strings.HasPrefix(dest, wikilinkURLPrefix) {
					item.target = notes.DecodeWikilinkTarget(strings.TrimPrefix(dest, wikilinkURLPrefix))
					item.isWikilink = true
					item.label = "[[" + label + "]]"
				} else {
					item.target = dest
					item.showTarget = dest != ""
				}
				if item.label != "" && item.target != "" {
					links = append(links, item)
				}
			case *ast.AutoLink:
				text := strings.TrimSpace(string(n.Label([]byte(raw))))
				if text != "" {
					links = append(links, renderedMarkdownLink{label: text, target: text})
				}
			}
			walk(child)
		}
	}
	walk(doc)
	return links
}

func renderInlineLabelText(node ast.Node, source []byte) string {
	if node == nil {
		return ""
	}

	switch n := node.(type) {
	case *ast.Text:
		s := string(n.Segment.Value(source))
		if n.HardLineBreak() {
			return s + "\n"
		}
		if n.SoftLineBreak() {
			return s + " "
		}
		return s
	case *ast.String:
		return string(n.Value)
	case *ast.CodeSpan:
		return strings.TrimSpace(string(nodeText(n, source)))
	case *extast.TaskCheckBox:
		return ""
	default:
		var parts strings.Builder
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			parts.WriteString(renderInlineLabelText(child, source))
		}
		if parts.Len() > 0 {
			return parts.String()
		}
		return string(nodeText(node, source))
	}
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
		label = stripANSI(label)
		dest := strings.TrimSpace(string(n.Destination))

		if strings.HasPrefix(dest, wikilinkURLPrefix) {
			return lipgloss.NewStyle().
				Underline(true).
				Foreground(accentSoftColor).
				Background(bgSoftColor).
				Render("[[" + label + "]]")
		}

		return lipgloss.NewStyle().
			Underline(true).
			Foreground(accentColor).
			Background(bgSoftColor).
			Render(label)

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
		Foreground(textColor).
		Background(bgSoftColor).
		Render(text)
	rendered = wrapStyledTextOnWhitespace(rendered, width)
	rendered = reapplyWrappedLineBaseStyle(rendered, textColor, bgSoftColor)
	rendered = fillStyledLinesBackground(rendered, width, bgSoftColor)

	return prefixLines(rendered, strings.Repeat(" ", indent))
}

func reapplyWrappedLineBaseStyle(content string, fg, bg lipgloss.Color) string {
	styleStart, ok := ansiBaseStylePrefix(fg, bg)
	if !ok || content == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+styleStart)
		line = strings.ReplaceAll(line, ansi.ResetStyle, ansi.ResetStyle+styleStart)
		lines[i] = styleStart + line
	}
	return strings.Join(lines, "\n")
}

func ansiBaseStylePrefix(fg, bg lipgloss.Color) (string, bool) {
	fgRGB, ok := parseHexColor(string(fg))
	if !ok {
		return "", false
	}
	bgRGB, ok := parseHexColor(string(bg))
	if !ok {
		return "", false
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm",
		clampChannel(fgRGB.r), clampChannel(fgRGB.g), clampChannel(fgRGB.b),
		clampChannel(bgRGB.r), clampChannel(bgRGB.g), clampChannel(bgRGB.b)), true
}

func fillStyledLinesBackground(content string, width int, bg lipgloss.Color) string {
	if width <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	padStyle := lipgloss.NewStyle().Background(bg)

	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < width {
			lines[i] = line + padStyle.Render(strings.Repeat(" ", width-w))
		}
	}

	return strings.Join(lines, "\n")
}

func wrapStyledTextOnWhitespace(s string, limit int) string {
	if limit < 1 {
		return s
	}

	var (
		cluster    []byte
		buf        bytes.Buffer
		word       bytes.Buffer
		space      bytes.Buffer
		curWidth   int
		wordWidth  int
		spaceWidth int
		pstate     = parser.GroundState
		b          = []byte(s)
	)

	addSpace := func() {
		if curWidth == 0 || (spaceWidth == 0 && space.Len() == 0) {
			space.Reset()
			spaceWidth = 0
			return
		}
		curWidth += spaceWidth
		buf.Write(space.Bytes())
		space.Reset()
		spaceWidth = 0
	}

	addWord := func() {
		if word.Len() == 0 {
			return
		}
		addSpace()
		curWidth += wordWidth
		buf.Write(word.Bytes())
		word.Reset()
		wordWidth = 0
	}

	addNewline := func() {
		buf.WriteByte('\n')
		curWidth = 0
		space.Reset()
		spaceWidth = 0
	}

	i := 0
	for i < len(b) {
		state, action := parser.Table.Transition(pstate, b[i])
		if state == parser.Utf8State {
			var width int
			cluster, width = ansi.FirstGraphemeCluster(b[i:], ansi.GraphemeWidth)
			i += len(cluster)

			r, _ := utf8.DecodeRune(cluster)
			switch {
			case r == '\n':
				addWord()
				addNewline()
			case r != utf8.RuneError && unicode.IsSpace(r) && r != 0xA0:
				addWord()
				space.Write(cluster)
				spaceWidth += width
			default:
				word.Write(cluster)
				wordWidth += width
				if curWidth > 0 && curWidth+spaceWidth+wordWidth > limit {
					addNewline()
				}
			}

			pstate = parser.GroundState
			continue
		}

		switch action {
		case parser.PrintAction, parser.ExecuteAction:
			r := rune(b[i])
			switch {
			case r == '\n':
				addWord()
				addNewline()
			case unicode.IsSpace(r):
				addWord()
				space.WriteByte(b[i])
				if action == parser.PrintAction {
					spaceWidth++
				}
			default:
				word.WriteByte(b[i])
				if action == parser.PrintAction {
					wordWidth++
				}
				if curWidth > 0 && curWidth+spaceWidth+wordWidth > limit {
					addNewline()
				}
			}
		default:
			word.WriteByte(b[i])
		}

		if pstate != parser.Utf8State {
			pstate = state
		}
		i++
	}

	addWord()
	return buf.String()
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
