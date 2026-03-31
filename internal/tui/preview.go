package tui

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

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
			m.preview.GotoTop()
			return

		case pinItemNote, pinItemTemporaryNote:
			raw, err := notes.ReadAll(item.Path)
			if err != nil {
				m.previewPath = item.Path
				m.previewContent = "Failed to read note: " + err.Error()
				m.previewPrivacyForcedByNote = false
				m.preview.SetContent(m.previewContent)
				m.rebuildPreviewHeadingsFromRendered()
				m.preview.GotoTop()
				return
			}

			private := notes.NoteIsPrivate(raw)

			rel := item.RelPath
			if item.Kind == pinItemTemporaryNote {
				rel = filepath.Join(".tmp", rel)
			}

			rendered := m.renderNotePreview(rel, raw, item.Tags)
			if m.effectivePreviewPrivacy(private) {
				rendered = blurRenderedText(rendered)
			}

			m.previewPrivacyForcedByNote = private
			m.previewPath = item.Path
			m.previewContent = rendered
			m.preview.SetContent(rendered)
			m.rebuildPreviewHeadingsFromRendered()
			m.preview.GotoTop()
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
			m.preview.GotoTop()
			return
		}

		if m.previewPath == n.Path && m.previewContent != "" {
			return
		}

		raw, err := notes.ReadAll(n.Path)
		if err != nil {
			m.previewPath = n.Path
			m.previewContent = "Failed to read note: " + err.Error()
			m.preview.SetContent(m.previewContent)
			m.rebuildPreviewHeadingsFromRendered()
			m.preview.GotoTop()
			return
		}

		private := notes.NoteIsPrivate(raw)

		rendered := m.renderNotePreview(filepath.Join(".tmp", n.RelPath), raw, n.Tags)
		if m.effectivePreviewPrivacy(private) {
			rendered = blurRenderedText(rendered)
		}

		m.previewPrivacyForcedByNote = private
		m.previewPath = n.Path
		m.previewContent = rendered
		m.preview.SetContent(rendered)
		m.rebuildPreviewHeadingsFromRendered()
		m.preview.GotoTop()
		return
	}

	item := m.currentTreeItem()
	if item == nil {
		m.previewPath = ""
		m.previewContent = "Nothing selected"
		m.previewPrivacyForcedByNote = false
		m.preview.SetContent(m.previewContent)
		m.rebuildPreviewHeadingsFromRendered()
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
		m.preview.GotoTop()
		return
	}

	if item.Note == nil {
		m.previewPath = ""
		m.previewContent = "No note selected"
		m.preview.SetContent(m.previewContent)
		m.rebuildPreviewHeadingsFromRendered()
		m.preview.GotoTop()
		return
	}

	if m.previewPath == item.Note.Path && m.previewContent != "" {
		return
	}

	raw, err := notes.ReadAll(item.Note.Path)
	if err != nil {
		m.previewPath = item.Note.Path
		m.previewContent = "Failed to read note: " + err.Error()
		m.preview.SetContent(m.previewContent)
		m.rebuildPreviewHeadingsFromRendered()
		m.preview.GotoTop()
		return
	}

	private := notes.NoteIsPrivate(raw)

	rendered := m.renderNotePreview(item.Note.RelPath, raw, item.Note.Tags)
	if m.effectivePreviewPrivacy(private) {
		rendered = blurRenderedText(rendered)
	}

	m.previewPrivacyForcedByNote = private
	m.previewPath = item.Note.Path
	m.previewContent = rendered
	m.preview.SetContent(rendered)
	m.rebuildPreviewHeadingsFromRendered()
	m.preview.GotoTop()
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
	if tagsHeader == "" {
		return rendered
	}

	return tagsHeader + "\n\n" + rendered
}
