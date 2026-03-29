package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gmtext "github.com/yuin/goldmark/text"
)

type markdownPreviewRenderer struct {
	source []byte
	width  int
}

func renderMarkdownTerminal(raw string, width int) string {
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
		width:  max(20, width),
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
	return strings.Join(parts, "\n\n")
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
		text := strings.TrimSpace(r.linesText(n.Lines()))
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
		// Usually handled by renderList, but keep a fallback.
		return r.renderListItem(n, 0, false, indent)

	case *ast.ThematicBreak:
		line := strings.Repeat("─", max(8, r.width-indent))
		return prefixLines(line, strings.Repeat(" ", indent))

	case *ast.HTMLBlock:
		text := strings.TrimSpace(string(n.Text(r.source)))
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
		text := strings.TrimSpace(string(node.Text(r.source)))
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
				Width(max(10, r.width-indent)).
				Render(text),
			lipgloss.NewStyle().
				Foreground(accentSoftColor).
				Width(max(10, r.width-indent)).
				Render(underline),
		)
		return prefixLines(block, base)
	case 3:
		return prefixLines(
			lipgloss.NewStyle().
				Bold(true).
				Foreground(accentSoftColor).
				Width(max(10, r.width-indent)).
				Render(text),
			base,
		)
	default:
		return prefixLines(
			lipgloss.NewStyle().
				Bold(true).
				Foreground(textColor).
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
	marker := "• "
	if ordered {
		marker = fmt.Sprintf("%d. ", index+1)
	}

	itemIndent := indent + len(marker)
	var blocks []string

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

	firstLines := strings.Split(first, "\n")
	firstLines[0] = strings.Repeat(" ", indent) + marker + strings.TrimLeft(firstLines[0], " ")
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
		lang = strings.TrimSpace(string(n.Info.Text(r.source)))
		if fields := strings.Fields(lang); len(fields) > 0 {
			lang = fields[0]
		}
	}

	return r.renderPlainCode(code, lang, indent)
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
		switch {
		case n.HardLineBreak():
			return s + "\n"
		case n.SoftLineBreak():
			return s + " "
		default:
			return s
		}

	case *ast.String:
		return string(n.Value)

	case *ast.Emphasis:
		content := r.renderInlineChildren(n)
		if n.Level == 2 {
			return lipgloss.NewStyle().Bold(true).Render(content)
		}
		return lipgloss.NewStyle().Italic(true).Render(content)

	case *ast.CodeSpan:
		content := strings.TrimSpace(string(n.Text(r.source)))
		return lipgloss.NewStyle().
			Foreground(accentSoftColor).
			Render("`" + content + "`")

	case *ast.Link:
		label := strings.TrimSpace(r.renderInlineChildren(n))
		if label == "" {
			label = strings.TrimSpace(string(n.Text(r.source)))
		}
		dest := strings.TrimSpace(string(n.Destination))

		styled := lipgloss.NewStyle().
			Underline(true).
			Foreground(accentColor).
			Render(label)

		if dest != "" && dest != label {
			return styled + mutedStyle.Render(" ("+dest+")")
		}
		return styled

	case *ast.AutoLink:
		text := strings.TrimSpace(string(n.Text(r.source)))
		return lipgloss.NewStyle().
			Underline(true).
			Foreground(accentColor).
			Render(text)

	case *ast.Image:
		alt := strings.TrimSpace(r.renderInlineChildren(n))
		if alt == "" {
			alt = "image"
		}
		return mutedStyle.Render("[image: " + alt + "]")

	default:
		if node.FirstChild() != nil {
			return r.renderInlineChildren(node)
		}
		return string(node.Text(r.source))
	}
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
		Render(text)

	return prefixLines(rendered, strings.Repeat(" ", indent))
}

func prefixLines(text, prefix string) string {
	if prefix == "" {
		return text
	}

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = prefix
		} else {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
