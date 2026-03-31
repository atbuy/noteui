package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/notes"
)

func (m Model) View() string {
	if m.showDashboard {
		return m.renderDashboardView()
	}

	usableWidth := max(40, m.width-6)
	leftWidth, rightWidth := m.panelWidths()
	gap := strings.Repeat(" ", panelGapWidth)

	leftBody := lipgloss.JoinVertical(
		lipgloss.Left,
		panelTitleStyle.Render(m.leftPanelTitle()),
		m.renderSearchBar(),
		"",
		m.renderLeftPaneBody(),
	)

	rightBody := lipgloss.JoinVertical(
		lipgloss.Left,
		panelTitleStyle.Render("Preview"),
		m.previewView(),
	)

	leftFocused := m.focus == focusTree
	rightFocused := m.focus == focusPreview

	left := panelStyle(leftWidth, m.height, leftFocused).Render(leftBody)
	right := panelStyle(rightWidth, m.height, rightFocused).Render(rightBody)

	titleText := " noteui "
	if strings.TrimSpace(m.version) != "" {
		titleText = fmt.Sprintf(" noteui %s ", m.version)
	}

	title := titleBarStyle.
		Width(usableWidth).
		Render(titleText)

	footer := footerStyle.
		Width(usableWidth).
		Render(m.renderStatus())

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right)

	base := appStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			body,
			footer,
		),
	)

	if m.showCreateCategory {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderCreateCategoryModal(),
		)
	}

	if m.showMove {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderMoveModal(),
		)
	}

	if m.showRename {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderRenameModal(),
		)
	}

	if m.showHelp {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderHelpModal(),
		)
	}

	return base
}

func (m Model) renderDashboardView() string {
	cardWidth := min(92, max(60, m.width-10))
	innerWidth := max(24, cardWidth-6)

	titleText := "noteui"
	if strings.TrimSpace(m.version) != "" {
		titleText = fmt.Sprintf("noteui %s", m.version)
	}

	rootText := filepath.Join("~", "notes")
	if strings.TrimSpace(m.rootDir) != "" {
		rootText = m.rootDir
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		Width(innerWidth).
		Align(lipgloss.Center).
		Render(titleText)

	subtitle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(innerWidth).
		Align(lipgloss.Center).
		Render("Fast local notes with previews, temporary notes, pins, and privacy controls")

	divider := lipgloss.NewStyle().
		Foreground(subtleColor).
		Width(innerWidth).
		Render(strings.Repeat("─", innerWidth))

	rootLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentSoftColor).
		Render("Root")

	rootValue := lipgloss.NewStyle().
		Foreground(textColor).
		Width(innerWidth).
		Render(rootText)

	workspaceLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentSoftColor).
		Render("Workspace")

	summaryLines := []string{
		dashboardSummaryLine("Notes:", fmt.Sprintf("%d", len(m.notes)), innerWidth),
		dashboardSummaryLine("Temporary:", fmt.Sprintf("%d", len(m.tempNotes)), innerWidth),
		dashboardSummaryLine(
			"Categories:",
			fmt.Sprintf("%d", m.dashboardCategoriesCount()),
			innerWidth,
		),
		dashboardSummaryLine(
			"Pinned notes:",
			fmt.Sprintf("%d", m.dashboardPinnedNotesCount()),
			innerWidth,
		),
		dashboardSummaryLine(
			"Pinned categories:",
			fmt.Sprintf("%d", m.dashboardPinnedCategoriesCount()),
			innerWidth,
		),
		dashboardSummaryLine("Theme:", m.dashboardThemeName(), innerWidth),
		dashboardSummaryLine("Privacy:", m.dashboardPrivacySummary(), innerWidth),
	}
	workspaceBlock := lipgloss.JoinVertical(lipgloss.Left, summaryLines...)

	recentLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentSoftColor).
		Render("Recent")

	recentItems := m.dashboardRecentNotes(5)
	recentLines := make([]string, 0, len(recentItems)*2)
	if len(recentItems) == 0 {
		recentLines = append(recentLines, mutedStyle.Render("No recent notes"))
	} else {
		timestampWidth := 24
		gapWidth := 2
		leftWidth := max(16, innerWidth-timestampWidth-gapWidth)

		for i, item := range recentItems {
			tag := "[note]"
			if item.IsTemp {
				tag = "[temp]"
			}

			num := lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor).
				Render(fmt.Sprintf("%d", i+1))

			tagStyled := lipgloss.NewStyle().
				Foreground(mutedColor).
				Render(tag)

			prefix := lipgloss.JoinHorizontal(
				lipgloss.Left,
				num,
				"  ",
				tagStyled,
				" ",
			)

			prefixWidth := lipgloss.Width(prefix)
			titleWidth := max(8, leftWidth-prefixWidth)

			titleCol := lipgloss.NewStyle().
				Width(titleWidth).
				MaxWidth(titleWidth).
				Foreground(textColor).
				Render(trimOrPad(item.Display, titleWidth))

			leftCol := lipgloss.NewStyle().
				Width(leftWidth).
				Render(lipgloss.JoinHorizontal(lipgloss.Left, prefix, titleCol))

			timeText := relativeDashboardTime(
				item.Note.ModTime,
			) + " · " + formatDashboardTime(
				item.Note.ModTime,
			)
			timeCol := lipgloss.NewStyle().
				Width(timestampWidth).
				Align(lipgloss.Right).
				Foreground(mutedColor).
				Render(timeText)

			topLine := lipgloss.JoinHorizontal(
				lipgloss.Top,
				leftCol,
				strings.Repeat(" ", gapWidth),
				timeCol,
			)

			pathLine := lipgloss.NewStyle().
				Width(innerWidth).
				PaddingLeft(4).
				Foreground(mutedColor).
				Render(shortenDashboardPath(m.rootDir, item.Note.Path))

			recentLines = append(recentLines, topLine, pathLine)
		}
	}
	recentBlock := lipgloss.JoinVertical(lipgloss.Left, recentLines...)

	actionsLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentSoftColor).
		Render("Quick actions")

	actionLines := []string{
		dashboardActionLine("enter", "Open workspace", innerWidth),
		dashboardActionLine("]", "Open Temporary", innerWidth),
		dashboardActionLine("P", "Open Pins", innerWidth),
		dashboardActionLine("N", "Create temporary note", innerWidth),
		dashboardActionLine("1-5", "Open recent note", innerWidth),
		dashboardActionLine("q", "Quit", innerWidth),
	}
	actionsBlock := lipgloss.JoinVertical(lipgloss.Left, actionLines...)

	tipsLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentSoftColor).
		Render("Tip")

	tipsBlock := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(innerWidth).
		Render("This dashboard is optional. Set dashboard = false in your TOML config to start directly in the main workspace.")

	warning := ""
	if m.startupError != "" {
		warning = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Width(innerWidth).
			Render("Config warning: " + m.startupError)
	}

	contentParts := []string{
		title,
		subtitle,
		"",
		divider,
		"",
		rootLabel,
		rootValue,
		"",
		workspaceLabel,
		workspaceBlock,
		"",
		recentLabel,
		recentBlock,
		"",
		actionsLabel,
		actionsBlock,
		"",
		tipsLabel,
		tipsBlock,
	}

	if warning != "" {
		contentParts = append(contentParts, "", warning)
	}

	card := lipgloss.NewStyle().
		Width(cardWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Render(lipgloss.JoinVertical(lipgloss.Left, contentParts...))

	return lipgloss.Place(
		max(1, m.width),
		max(1, m.height),
		lipgloss.Center,
		lipgloss.Center,
		card,
	)
}

func (m Model) renderTreeView() string {
	if len(m.treeItems) == 0 {
		return emptyStyle.Render("(empty)")
	}

	lines := make([]string, 0, len(m.treeItems))
	for i, item := range m.treeItems {
		lines = append(lines, m.renderTreeLine(item, i == m.treeCursor))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderTreeLine(item treeItem, selected bool) string {
	rowWidth := m.treeInnerWidth()

	var icon string
	switch item.Kind {
	case treeCategory:
		if m.categoryHasChildren(item.RelPath) {
			if item.Expanded {
				icon = iconCategoryExpanded
			} else {
				icon = iconCategoryCollapsed
			}
		} else {
			icon = iconCategoryLeaf
		}
	case treeNote:
		icon = iconNote
	}

	pinned := false
	switch item.Kind {
	case treeCategory:
		pinned = m.isPinnedCategory(item.RelPath)
	case treeNote:
		if item.Note != nil {
			pinned = m.isPinnedNote(item.Note.RelPath)
		}
	}

	pinMark := "  "
	if pinned {
		pinMark = "★ "
	}

	indent := strings.Repeat("  ", item.Depth)
	leftPrefix := indent + pinMark + icon + " "
	prefixWidth := lipgloss.Width(leftPrefix)

	// Tags are only shown for notes.
	var tags []string
	if item.Kind == treeNote && item.Note != nil {
		tags = item.Note.Tags
	}

	// Calculate available space.
	// Leave 2 for padding on each side.
	availableWidth := rowWidth - 2 - prefixWidth

	tagsPart := ""
	titleWidth := availableWidth

	if len(tags) > 0 {
		// Reserve up to 40% of available width for tags, minimum 10 chars.
		maxTagsWidth := max(10, availableWidth*40/100)
		tagsPart, titleWidth = renderTagChips(tags, availableWidth, maxTagsWidth)
	}

	title := item.Name
	truncatedTitle := truncateToWidth(title, titleWidth)

	// Pad the title to fill its allocated width so tags are right-aligned.
	titlePadded := truncatedTitle + strings.Repeat(
		" ",
		max(0, titleWidth-lipgloss.Width(truncatedTitle)),
	)

	plainLine := leftPrefix + titlePadded + tagsPart

	if selected {
		return lipgloss.NewStyle().
			Width(rowWidth).
			Padding(0, 1).
			Foreground(selectedFgColor).
			Background(selectedBgColor).
			Bold(boldSelected).
			Render(plainLine)
	}

	rowStyle := treeNoteStyle
	if item.Kind == treeCategory {
		rowStyle = treeCategoryStyle
	}
	if pinned {
		rowStyle = rowStyle.Foreground(accentColor)
	}

	return rowStyle.
		Width(rowWidth).
		Padding(0, 1).
		Render(plainLine)
}

func renderTagChips(tags []string, availableWidth, maxTagsWidth int) (string, int) {
	// Build chips and fit as many as possible within maxTagsWidth.
	// Each chip looks like: [tag]
	// If not all fit, append +N for the remainder.

	type chip struct {
		text  string
		width int
	}

	chips := make([]chip, 0, len(tags))
	for _, t := range tags {
		text := "[" + t + "]"
		chips = append(chips, chip{text: text, width: lipgloss.Width(text)})
	}

	fitted := make([]chip, 0, len(chips))
	usedWidth := 0
	remaining := 0

	for i, c := range chips {
		// Account for a space before each chip.
		needed := c.width
		if len(fitted) > 0 {
			needed++
		}

		// Check if we need to reserve space for an overflow indicator.
		overflowText := ""
		if i < len(chips)-1 {
			overflowText = fmt.Sprintf("+%d", len(chips)-i)
		}
		overflowWidth := 0
		if overflowText != "" {
			overflowWidth = lipgloss.Width(overflowText) + 1
		}

		if usedWidth+needed+overflowWidth <= maxTagsWidth {
			fitted = append(fitted, c)
			usedWidth += needed
		} else {
			remaining = len(chips) - i
			break
		}
	}

	if len(fitted) == 0 {
		// Nothing fits at all, return full width to title.
		return "", availableWidth
	}

	var b strings.Builder
	for i, c := range fitted {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(c.text)
	}

	if remaining > 0 {
		fmt.Fprintf(&b, " +%d", remaining)
	}

	tagsStr := mutedStyle.Render(b.String())
	tagsRenderedWidth := usedWidth
	if remaining > 0 {
		tagsRenderedWidth += lipgloss.Width(fmt.Sprintf(" +%d", remaining))
	}

	titleWidth := availableWidth - tagsRenderedWidth
	return tagsStr, max(8, titleWidth)
}

func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	cur := 0

	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if cur+rw > maxWidth-1 {
			break
		}
		out = append(out, r)
		cur += rw
	}

	return string(out) + "…"
}

func (m Model) renderTemporaryListView() string {
	tempNotes := m.filteredTempNotes()
	if len(tempNotes) == 0 {
		return emptyStyle.Render("(no temporary notes)")
	}

	lines := make([]string, 0, len(tempNotes))
	rowWidth := m.treeInnerWidth()

	for i, n := range tempNotes {
		pinMark := "  "
		if m.isPinnedTemporaryNote(n.RelPath) {
			pinMark = "★ "
		}

		label := trimOrPad(pinMark+iconNote+" "+n.Title(), rowWidth-2)

		if i == m.tempCursor {
			lines = append(lines, lipgloss.NewStyle().
				Width(rowWidth).
				Padding(0, 1).
				Foreground(selectedFgColor).
				Background(selectedBgColor).
				Bold(boldSelected).
				Render(label))
			continue
		}

		rowStyle := treeNoteStyle
		if m.isPinnedTemporaryNote(n.RelPath) {
			rowStyle = rowStyle.Copy().Foreground(accentColor)
		}

		lines = append(lines, rowStyle.
			Width(rowWidth).
			Padding(0, 1).
			Render(label))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderPinsListView() string {
	items := m.filteredPinnedItems()
	if len(items) == 0 {
		return emptyStyle.Render("(no pinned items)")
	}

	lines := make([]string, 0, len(items))
	rowWidth := m.treeInnerWidth()

	for i, item := range items {
		var prefix string
		switch item.Kind {
		case pinItemCategory:
			prefix = "★ " + iconCategoryLeaf + " [cat] "
		case pinItemNote:
			prefix = "★ " + iconNote + " [note] "
		case pinItemTemporaryNote:
			prefix = "★ " + iconNote + " [temp] "
		}

		label := item.Name
		if strings.TrimSpace(label) == "" {
			label = item.RelPath
		}

		plain := trimOrPad(prefix+label, rowWidth-2)

		if i == m.pinsCursor {
			lines = append(lines, lipgloss.NewStyle().
				Width(rowWidth).
				Padding(0, 1).
				Foreground(selectedFgColor).
				Background(selectedBgColor).
				Bold(boldSelected).
				Render(plain))
			continue
		}

		style := treeNoteStyle.Foreground(accentColor).Width(rowWidth).Padding(0, 1)
		lines = append(lines, style.Render(plain))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderSearchBar() string {
	if m.searchMode || strings.TrimSpace(m.searchInput.Value()) != "" {
		return m.searchInput.View()
	}
	return mutedStyle.Render("Press / to search")
}

func (m Model) renderStatus() string {
	if m.deletePending != nil {
		return statusErrStyle.Render("TRASH PENDING • press d to confirm • esc to cancel")
	}

	if m.startupError != "" {
		return statusErrStyle.Render("CONFIG ERROR • " + m.startupError)
	}

	parts := []string{
		m.renderModeSegment(),
		m.renderSelectionSegment(),
		m.renderPrivacySegment(),
		m.renderSortSegment(),
	}

	if filter := m.renderFilterSegment(); filter != "" {
		parts = append(parts, filter)
	}

	if preview := m.renderPreviewSegment(); preview != "" {
		parts = append(parts, preview)
	}

	if m.status != "" {
		parts = append(parts, m.status)
	}

	line := strings.Join(parts, "  •  ")

	switch {
	case strings.HasPrefix(m.status, "error:"),
		strings.HasPrefix(m.status, "editor error:"),
		strings.HasPrefix(m.status, "create failed:"),
		strings.HasPrefix(m.status, "category create failed:"),
		strings.HasPrefix(m.status, "delete failed:"),
		strings.HasPrefix(m.status, "rename failed:"),
		strings.HasPrefix(m.status, "move failed:"),
		strings.HasPrefix(m.status, "pin failed:"):
		return statusErrStyle.Render(line)
	default:
		return statusOKStyle.Render(line)
	}
}

func (m Model) renderModeSegment() string {
	switch {
	case m.showDashboard:
		return "DASHBOARD"
	case m.showHelp:
		return "HELP"
	case m.showCreateCategory:
		return "NEW CATEGORY"
	case m.showMove:
		return "MOVE"
	case m.showRename:
		return "RENAME"
	case m.searchMode:
		switch m.listMode {
		case listModeTemporary:
			return "SEARCH TEMP"
		case listModePins:
			return "SEARCH PINS"
		default:
			return "SEARCH"
		}
	case m.focus == focusPreview:
		return "PREVIEW"
	case m.listMode == listModeTemporary:
		return "TEMP"
	case m.listMode == listModePins:
		return "PINS"
	default:
		return "TREE"
	}
}

func (m Model) renderSelectionSegment() string {
	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil {
			return "pins: none"
		}

		switch item.Kind {
		case pinItemCategory:
			return "pinned category: ★ " + item.Name
		case pinItemNote:
			return "pinned note: ★ " + item.Name
		case pinItemTemporaryNote:
			return "pinned temp: ★ " + item.Name
		}
	}

	if m.listMode == listModeTemporary {
		n := m.currentTempNote()
		if n == nil {
			return "temporary: none"
		}
		if m.isPinnedTemporaryNote(n.RelPath) {
			return "temporary: ★ " + n.Title()
		}
		return "temporary: " + n.Title()
	}

	item := m.currentTreeItem()
	if item == nil {
		return "nothing selected"
	}

	switch item.Kind {
	case treeCategory:
		name := item.Name
		if item.RelPath == "" {
			name = "~/notes"
		}
		if m.isPinnedCategory(item.RelPath) {
			return "category: ★ " + name
		}
		return "category: " + name

	case treeNote:
		if item.Note == nil {
			return "note"
		}
		title := item.Note.Title()
		if m.isPinnedNote(item.Note.RelPath) {
			return "note: ★ " + title
		}
		return "note: " + title
	}

	return "selection"
}

func (m Model) renderFilterSegment() string {
	filter := strings.TrimSpace(m.searchInput.Value())
	if filter == "" {
		return ""
	}
	return "filter: " + filter
}

func (m Model) renderPreviewSegment() string {
	if m.preview.TotalLineCount() == 0 {
		return ""
	}

	// Word count and reading time.
	wordCount := 0
	if m.selected != nil {
		raw, err := notes.ReadAll(m.selected.Path)
		if err == nil {
			wordCount = notes.WordCount(raw)
		}
	}

	wordPart := ""
	if wordCount > 0 {
		mins := notes.ReadingTimeMinutes(wordCount)
		wordPart = fmt.Sprintf("%d words · ~%d min read", wordCount, mins)
	}

	// Scroll position.
	atTop := m.preview.AtTop()
	atBottom := m.preview.AtBottom()

	scrollPart := ""
	switch {
	case atTop && atBottom:
		scrollPart = "100%"
	case atTop:
		scrollPart = "top"
	case atBottom:
		scrollPart = "bottom"
	default:
		total := m.preview.TotalLineCount()
		offset := m.preview.YOffset
		height := m.preview.Height

		maxOffset := total - height
		if maxOffset > 0 {
			pct := int(float64(offset) / float64(maxOffset) * 100.0)
			pct = max(pct, 0)
			pct = min(pct, 100)
			scrollPart = fmt.Sprintf("%d%%", pct)
		}
	}

	if wordPart != "" && scrollPart != "" {
		return fmt.Sprintf("preview: %s · %s", wordPart, scrollPart)
	}
	if wordPart != "" {
		return "preview: " + wordPart
	}
	if scrollPart != "" {
		return "preview: " + scrollPart
	}
	return ""
}

func (m Model) renderPrivacySegment() string {
	switch {
	case m.cfg.Preview.Privacy:
		return "privacy: config"
	case m.previewPrivacyForcedByNote:
		return "privacy: note"
	case m.previewPrivacyEnabled:
		return "privacy: on"
	default:
		return "privacy: off"
	}
}

func (m Model) renderHelpModal() string {
	modalWidth, innerWidth := m.modalDimensions(50, 76)

	title := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(modalTitleStyle.Render("Help"))

	lines := []string{
		m.renderHelpLine("j / k", "Move up and down", innerWidth),
		m.renderHelpLine("ctrl+u / ctrl+d", "Scroll half page up / down", innerWidth),
		m.renderHelpLine("enter / o", "Open note or jump from Pins", innerWidth),
		m.renderHelpLine("h/l", "Collapse/Expand category", innerWidth),
		m.renderHelpLine("]/[", "Switch Notes / Temporary", innerWidth),
		m.renderHelpLine("P", "Toggle Pins view", innerWidth),
		m.renderHelpLine("/", "Search", innerWidth),
		m.renderHelpLine("esc", "Leave search, then clear on second press", innerWidth),
		m.renderHelpLine("n", "New note in current view", innerWidth),
		m.renderHelpLine("N", "New temporary note", innerWidth),
		m.renderHelpLine("B", "Toggle preview privacy", innerWidth),
		m.renderHelpLine("C", "Create category", innerWidth),
		m.renderHelpLine("dd", "Trash note/category", innerWidth),
		m.renderHelpLine("r", "Refresh", innerWidth),
		m.renderHelpLine("q", "Quit", innerWidth),
		m.renderHelpLine("esc / q / ?", "Close help", innerWidth),
		m.renderHelpLine("m", "Move note/category", innerWidth),
		m.renderHelpLine("R", "Rename note/category", innerWidth),
		m.renderHelpLine("p", "Pin or unpin current item", innerWidth),
		m.renderHelpLine("gg / G", "Jump to top / bottom of list", innerWidth),
		m.renderHelpLine("s", "Toggle sort (alpha / modified)", innerWidth),
		m.renderHelpLine("#tag", "Filter by tag in search", innerWidth),
	}

	body := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	footer := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(modalFooterStyle.Render("Press esc, q, or ? to close"))

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				title,
				lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(""),
				body,
				lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(""),
				footer,
			),
		)

	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderMoveModal() string {
	title := "Move"
	hint := "Enter the new relative path under ~/notes."
	label := "Path"

	if m.movePending != nil {
		switch m.movePending.kind {
		case moveTargetNote:
			title = "Move note"
			hint = "Move the note to a new relative path under ~/notes."
			label = "Note"
		case moveTargetCategory:
			title = "Move category"
			hint = "Move the category to a new relative path under ~/notes."
			label = "Category"
		}
	}

	return m.renderStandardModal(
		title,
		hint,
		label,
		m.moveInput,
		"Enter to move • Esc to cancel",
	)
}

func (m Model) renderRenameModal() string {
	title := "Rename note"
	hint := "Change the note title. The file name will update automatically."
	label := "Title"

	if m.renamePending != nil && m.renamePending.kind == renameTargetCategory {
		title = "Rename category"
		hint = "Change the category path under ~/notes."
		label = "Category"
	}

	return m.renderStandardModal(
		title,
		hint,
		label,
		m.renameInput,
		"Enter to rename • Esc to cancel",
	)
}

func (m Model) renderStandardModal(
	title, hint, label string,
	input textinput.Model,
	footer string,
) string {
	modalWidth, innerWidth := m.modalDimensions(48, 76)

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.renderModalTitle(title, innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalHint(hint, innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalInputRow(label, input, innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalFooter(footer, innerWidth),
			),
		)

	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderCreateCategoryModal() string {
	return m.renderStandardModal(
		"Create category",
		"Use / to create nested categories, e.g. work/project-a",
		"Category",
		m.categoryInput,
		"Enter to create • Esc to cancel",
	)
}

func (m Model) renderModalTitle(text string, innerWidth int) string {
	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(modalTitleStyle.Render(text))
}

func (m Model) renderModalHint(text string, innerWidth int) string {
	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(modalMutedStyle.Render(text))
}

func (m Model) renderModalFooter(text string, innerWidth int) string {
	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(modalFooterStyle.Render(text))
}

func (m Model) renderModalBlank(innerWidth int) string {
	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render("")
}

func (m Model) renderModalInputRow(label string, input textinput.Model, innerWidth int) string {
	local := input
	local.Prompt = ""
	local.Width = max(12, min(36, innerWidth-20))

	local.TextStyle = lipgloss.NewStyle().
		Foreground(modalTextColor).
		Background(modalBgColor)

	local.PlaceholderStyle = lipgloss.NewStyle().
		Foreground(modalMutedColor).
		Background(modalBgColor)

	local.Cursor.Style = lipgloss.NewStyle().
		Foreground(modalTextColor).
		Background(modalTextColor)

	labelText := lipgloss.NewStyle().
		Foreground(modalAccentColor).
		Background(modalBgColor).
		Bold(true).
		Render(label + ":")

	// Make the label a 3-line block so its text aligns with the input text line,
	// not with the top border of the input box.
	promptBlock := lipgloss.NewStyle().
		Background(modalBgColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				"",
				labelText,
				"",
			),
		)

	inputField := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(modalAccentColor).
		BorderBackground(modalBgColor).
		Background(modalBgColor).
		Padding(0, 1).
		Render(local.View())

	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				promptBlock,
				"",
				inputField,
			),
		)
}

func (m Model) renderHelpLine(k, desc string, width int) string {
	keyWidth := 14
	descWidth := max(10, width-keyWidth)

	keyPart := lipgloss.NewStyle().
		Width(keyWidth).
		Background(modalBgColor).
		Render(
			modalKeyStyle.
				Width(keyWidth).
				Render(k),
		)

	descPart := lipgloss.NewStyle().
		Width(descWidth).
		Background(modalBgColor).
		Render(
			modalTextStyle.
				Width(descWidth).
				Render(desc),
		)

	return lipgloss.NewStyle().
		Width(width).
		Background(modalBgColor).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, keyPart, descPart))
}

func (m Model) previewView() string {
	return m.preview.View()
}

func (m Model) leftPanelTitle() string {
	switch m.listMode {
	case listModeTemporary:
		return fmt.Sprintf("Temporary (%d)", len(m.filteredTempNotes()))
	case listModePins:
		return fmt.Sprintf("Pins (%d)", len(m.filteredPinnedItems()))
	default:
		count := max(0, len(m.treeItems)-1)
		return fmt.Sprintf("Tree (%d)", count)
	}
}

func (m Model) renderLeftPaneBody() string {
	switch m.listMode {
	case listModeTemporary:
		return m.renderTemporaryListView()
	case listModePins:
		return m.renderPinsListView()
	default:
		return m.renderTreeView()
	}
}

func formatDashboardTime(t time.Time) string {
	return t.Local().Format("Jan 02 15:04")
}

func relativeDashboardTime(t time.Time) string {
	now := time.Now()
	if t.After(now) {
		t = now
	}

	d := now.Sub(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "yesterday"
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Local().Format("Jan 02")
	}
}

func shortenDashboardPath(rootDir, fullPath string) string {
	if strings.TrimSpace(fullPath) == "" {
		return ""
	}

	if rootDir != "" {
		if rel, err := filepath.Rel(rootDir, fullPath); err == nil && rel != "." {
			return filepath.ToSlash(rel)
		}
	}

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		if after, ok := strings.CutPrefix(fullPath, home+string(filepath.Separator)); ok {
			return "~/" + filepath.ToSlash(
				after,
			)
		}
	}

	return filepath.ToSlash(fullPath)
}

func dashboardActionLine(keyText, desc string, width int) string {
	keyPart := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		Render(keyText)

	descPart := lipgloss.NewStyle().
		Foreground(textColor).
		Render(desc)

	line := lipgloss.JoinHorizontal(lipgloss.Left, keyPart, "  ", descPart)
	return lipgloss.NewStyle().Width(width).Render(line)
}

func dashboardSummaryLine(label, value string, width int) string {
	labelPart := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(label)

	valuePart := lipgloss.NewStyle().
		Foreground(textColor).
		Bold(true).
		Render(value)

	line := lipgloss.JoinHorizontal(lipgloss.Left, labelPart, " ", valuePart)
	return lipgloss.NewStyle().Width(width).Render(line)
}

func (m Model) renderSortSegment() string {
	if m.sortByModTime {
		return "sort: modified"
	}
	return "sort: alpha"
}
