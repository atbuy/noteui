package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (e *EditorModel) SetLineNumbers(enabled bool) {
	e.lineNumbersEnabled = enabled
	if e.renderMode {
		e.recomputeRenderedDoc()
		e.syncRenderViewTop()
	}
}

func (e *EditorModel) SetRelativeLineNumbers(enabled bool) {
	e.relativeLineNumbers = enabled
	if enabled {
		e.lineNumbersEnabled = true
	}
}

// gutterLineNumber returns the number to display in the gutter for lineIdx.
// In relative mode all lines show their distance from the cursor; the cursor
// line shows its absolute 1-based number. In absolute mode every line shows
// its 1-based number.
func (e EditorModel) gutterLineNumber(lineIdx int) int {
	if e.relativeLineNumbers && lineIdx != e.row {
		d := lineIdx - e.row
		if d < 0 {
			d = -d
		}
		return d
	}
	return lineIdx + 1
}

// gutterWidth returns the column width reserved for the line-number gutter.
// Uses the current renderedDoc line count so the width is stable across renders.
func (e EditorModel) gutterWidth() int {
	if !e.lineNumbersEnabled {
		return 0
	}
	n := 1
	if e.renderedDoc != nil {
		n = max(1, len(e.renderedDoc.Lines))
	}
	return len(fmt.Sprintf("%d", n)) + 1
}

// recomputeRenderedDoc re-renders e.lines as a RenderedDoc. Safe to call when
// renderMode is false (no-op).
func (e *EditorModel) recomputeRenderedDoc() {
	if !e.renderMode {
		return
	}
	raw := editorJoinLines(e.lines)
	gw := e.gutterWidth()
	opts := markdownRenderOptions{
		Width:           max(20, e.width-gw),
		SyntaxHighlight: true,
	}
	doc := renderMarkdownDoc(raw, opts)
	// Re-render if gutter width changed due to different visual line count.
	if e.lineNumbersEnabled {
		newGw := len(fmt.Sprintf("%d", max(1, len(doc.Lines)))) + 1
		if newGw != gw {
			opts.Width = max(20, e.width-newGw)
			doc = renderMarkdownDoc(raw, opts)
		}
	}
	e.renderedDoc = &doc
}

// rowToVisualRange returns the visual line range [start, end) for source line row.
// Returns (-1,-1) if not found in any block.
func (e *EditorModel) rowToVisualRange(row int) (int, int) {
	if e.renderedDoc == nil {
		return -1, -1
	}
	for _, b := range e.renderedDoc.Blocks {
		if row >= b.SourceLine && row < b.SourceLine+b.SourceCount {
			return b.VisualStart, b.VisualEnd
		}
	}
	return -1, -1
}

// rowToVisualStart returns the visual line index for source line row. For empty
// or separator source lines not covered by any block, it estimates the position
// as the visual end of the last block that precedes the row.
func (e *EditorModel) rowToVisualStart(row int) int {
	start, _ := e.rowToVisualRange(row)
	if start >= 0 {
		return start
	}
	if e.renderedDoc == nil {
		return 0
	}
	vStart := 0
	for _, b := range e.renderedDoc.Blocks {
		if b.SourceLine+b.SourceCount <= row {
			vStart = b.VisualEnd
		}
	}
	return vStart
}

// maybeRestoreRenderMode re-enables render mode if the document has renderable content.
func (e *EditorModel) maybeRestoreRenderMode() {
	if e.renderedDoc != nil && len(e.renderedDoc.Blocks) > 0 {
		e.renderMode = true
		e.recomputeRenderedDoc()
		e.syncRenderViewTop()
	}
}

// syncRenderViewTop adjusts renderViewTop so the cursor source line is visible.
func (e *EditorModel) syncRenderViewTop() {
	vStart := e.rowToVisualStart(e.row)
	if vStart < e.renderViewTop {
		e.renderViewTop = vStart
	}
	if vStart >= e.renderViewTop+e.contentHeight {
		e.renderViewTop = vStart - e.contentHeight + 1
	}
	if e.renderViewTop < 0 {
		e.renderViewTop = 0
	}
}

// toggleCurrentTaskCheckbox flips - [ ] <-> - [x] on e.row.
func (e *EditorModel) toggleCurrentTaskCheckbox() {
	if e.row < 0 || e.row >= len(e.lines) {
		return
	}
	s := string(e.lines[e.row])
	var newS string
	switch {
	case strings.Contains(s, "- [ ]"):
		newS = strings.Replace(s, "- [ ]", "- [x]", 1)
	case strings.Contains(s, "- [x]"):
		newS = strings.Replace(s, "- [x]", "- [ ]", 1)
	case strings.Contains(s, "- [X]"):
		newS = strings.Replace(s, "- [X]", "- [ ]", 1)
	default:
		return
	}
	e.pushUndo()
	e.lines[e.row] = []rune(newS)
	e.dirty = true
	e.recomputeRenderedDoc()
	e.syncRenderViewTop()
}

// viewRendered is the single rendered-mode view for all editor modes.
// All visual lines render normally. The visual line at vStart for e.row shows
// the raw source with a single cursor cell at e.col.
func (e EditorModel) viewRendered() string {
	e.syncRenderViewTop()
	vStart := e.rowToVisualStart(e.row)

	gw := e.gutterWidth()
	contentWidth := e.width - gw

	gutterStyle := lipgloss.NewStyle().Foreground(mutedColor).Background(bgSoftColor)
	digits := gw - 1
	if digits < 1 {
		digits = 1
	}

	var b strings.Builder
	for i := 0; i < e.contentHeight; i++ {
		lineIdx := e.renderViewTop + i

		if gw > 0 {
			var lineNum int
			if e.relativeLineNumbers && lineIdx != vStart {
				lineNum = lineIdx - vStart
				if lineNum < 0 {
					lineNum = -lineNum
				}
			} else {
				lineNum = lineIdx + 1
			}
			b.WriteString(gutterStyle.Render(fmt.Sprintf("%*d ", digits, lineNum)))
		}

		if lineIdx == vStart {
			b.WriteString(e.renderRawSourceLine(e.row, e.col, contentWidth))
		} else {
			var rendered string
			if e.renderedDoc != nil && lineIdx < len(e.renderedDoc.Lines) {
				rendered = e.renderedDoc.Lines[lineIdx]
			}
			b.WriteString(editorFillLine(rendered, contentWidth, bgSoftColor, textColor))
		}

		if i < e.contentHeight-1 {
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')
	b.WriteString(e.renderCommandLine())
	b.WriteByte('\n')
	b.WriteString(e.renderStatusLine())
	return lipgloss.NewStyle().
		Width(e.width).
		Height(e.height).
		Background(bgSoftColor).
		Render(b.String())
}

// renderRawSourceLine renders e.lines[row] as plain text with a single cursor
// cell highlighted at col, padded to width.
func (e EditorModel) renderRawSourceLine(row, col, width int) string {
	if row < 0 || row >= len(e.lines) {
		return editorFillLine("", width, bgSoftColor, textColor)
	}
	line := e.lines[row]
	cursorStyle := lipgloss.NewStyle().Background(accentSoftColor).Foreground(bgSoftColor).Bold(true)
	baseStyle := lipgloss.NewStyle().Background(bgSoftColor).Foreground(textColor)

	var b strings.Builder
	visCol := 0
	for i := 0; i <= len(line) && visCol < width; i++ {
		hasCh := i < len(line)
		var ch rune
		if hasCh {
			ch = line[i]
		} else {
			ch = ' '
		}
		if i == col {
			b.WriteString(cursorStyle.Render(string(ch)))
		} else {
			b.WriteString(baseStyle.Render(string(ch)))
		}
		visCol++
		if !hasCh {
			break
		}
	}
	for visCol < width {
		b.WriteString(baseStyle.Render(" "))
		visCol++
	}
	return b.String()
}
