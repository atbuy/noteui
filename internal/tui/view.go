package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"

	"atbuy/noteui/internal/notes"
)

// renderBaseView renders the full tree + preview layout without any modal overlays.
func (m Model) renderBaseView() string {
	usableWidth := max(40, m.width-6)
	leftWidth, rightWidth := m.panelWidths()

	leftInnerWidth := max(18, leftWidth-2-2*panelPaddingX)
	rightInnerWidth := max(18, rightWidth-2-2*panelPaddingX)

	panelBg := lipgloss.NewStyle().Background(bgSoftColor)

	leftBody := lipgloss.JoinVertical(
		lipgloss.Left,
		panelTitleStyle.Width(leftInnerWidth).Render(m.leftPanelTitle()),
		panelBg.Width(leftInnerWidth).Render(m.renderLeftPaneHint(leftInnerWidth)),
		panelBg.Width(leftInnerWidth).Render(m.renderSearchBar()),
		panelBg.Width(leftInnerWidth).Render(""),
		panelBg.Width(leftInnerWidth).Render(m.renderLeftPaneBody()),
	)

	rightBody := lipgloss.JoinVertical(
		lipgloss.Left,
		panelTitleStyle.Width(rightInnerWidth).Render(m.rightPanelTitle()),
		panelBg.Width(rightInnerWidth).Render(m.renderRightPaneHint(rightInnerWidth)),
		panelBg.Width(rightInnerWidth).Render(""),
		panelBg.Width(rightInnerWidth).Render(m.previewView()),
	)

	leftFocused := m.focus == focusTree
	rightFocused := m.focus == focusPreview

	left := panelStyle(leftWidth, m.height, leftFocused).Render(leftBody)
	right := panelStyle(rightWidth, m.height, rightFocused).Render(rightBody)

	titleText := " noteui "
	if strings.TrimSpace(m.version) != "" {
		titleText = fmt.Sprintf(" noteui %s ", m.version)
	}
	if workspace := strings.TrimSpace(m.activeWorkspaceDisplay()); workspace != "" {
		titleText = strings.TrimRight(titleText, " ") + " • " + workspace + " "
	}

	title := titleBarStyle.
		Width(usableWidth).
		Render(titleText)

	footer := footerStyle.
		Width(usableWidth).
		Render(m.renderStatus())

	spacer := lipgloss.NewStyle().Width(usableWidth).Background(bgColor).Render("")

	gapHeight := max(10, m.height-6)
	gap := lipgloss.NewStyle().
		Width(panelGapWidth).
		Height(gapHeight).
		Background(bgColor).
		Render("")

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right)

	base := appStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			body,
			spacer,
			footer,
		),
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Background(bgColor).
		Render(base)
}

func (m Model) View() string {
	if m.showWorkspacePicker {
		background := m.renderBaseView()
		if m.showDashboard {
			background = m.renderDashboardView()
		}
		return placeOverlay(background, m.renderWorkspacePickerModal(), m.width, m.height)
	}

	if m.showDashboard {
		return m.renderDashboardView()
	}

	if m.showCommandPalette {
		return placeOverlay(m.renderBaseView(), m.renderCommandPaletteModal(), m.width, m.height)
	}

	// For all other modals, compute the base view once and use it as the background canvas.
	fullScreen := m.renderBaseView()

	if m.showCreateCategory {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderCreateCategoryModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showMoveBrowser {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderMoveBrowserModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showMove {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderMoveModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showRename {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderRenameModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showAddTag {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderAddTagModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showSyncDebugModal {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderSyncDebugModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showSyncTimeline {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderSyncTimelineModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showSyncProfilePicker {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderSyncProfilePickerModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showSyncProfileMigration {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderSyncProfileMigrationModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showThemePicker {
		return placeOverlay(m.renderBaseView(), m.renderThemePickerModal(), m.width, m.height)
	}

	if m.showHelp {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderHelpModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showTodoAdd {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderTodoAddModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showTodoEdit {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderTodoEditModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showTodoDueDate {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderTodoDueDateModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showTodoPriority {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderTodoPriorityModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showPassphraseModal {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderPassphraseModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showEncryptConfirm {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderEncryptConfirmModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showNoteHistory {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderNoteHistoryModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showTrashBrowser {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderTrashBrowserModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	if m.showTemplatePicker {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.renderTemplatePickerModal(),
			lipgloss.WithWhitespaceBackground(bgColor),
		)
	}

	return fullScreen
}

func (m Model) renderDashboardView() string {
	cardWidth := min(92, max(60, m.width-10))
	innerWidth := max(24, cardWidth-6)
	surface := lipgloss.NewStyle().
		Width(innerWidth).
		Background(bgSoftColor)

	titleText := "noteui"
	if strings.TrimSpace(m.version) != "" {
		titleText = fmt.Sprintf("noteui %s", m.version)
	}

	rootText := "No workspace selected"
	if strings.TrimSpace(m.rootDir) != "" {
		rootText = m.rootDir
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		Width(innerWidth).
		Background(bgSoftColor).
		Align(lipgloss.Center).
		Render(titleText)

	subtitle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(innerWidth).
		Background(bgSoftColor).
		Align(lipgloss.Center).
		Render("Fast local notes with previews, temporary notes, pins, and privacy controls")

	divider := lipgloss.NewStyle().
		Foreground(subtleColor).
		Width(innerWidth).
		Background(bgSoftColor).
		Render(strings.Repeat("─", innerWidth))

	rootLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentSoftColor).
		Width(innerWidth).
		Background(bgSoftColor).
		Render("Root")

	rootValue := lipgloss.NewStyle().
		Foreground(textColor).
		Width(innerWidth).
		Background(bgSoftColor).
		Render(rootText)

	workspaceLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentSoftColor).
		Width(innerWidth).
		Background(bgSoftColor).
		Render("Workspace")

	summaryLines := []string{}
	if workspace := strings.TrimSpace(m.activeWorkspaceDisplay()); workspace != "" {
		summaryLines = append(summaryLines, dashboardSummaryLine("Active:", workspace, innerWidth))
	}
	summaryLines = append(summaryLines,
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
	)
	workspaceBlock := lipgloss.JoinVertical(lipgloss.Left, summaryLines...)

	recentLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentSoftColor).
		Width(innerWidth).
		Background(bgSoftColor).
		Render("Recent")

	recentItems := m.dashboardRecentNotes(5)
	recentLines := make([]string, 0, len(recentItems)*2)
	if len(recentItems) == 0 {
		recentLines = append(recentLines, lipgloss.NewStyle().
			Width(innerWidth).
			Foreground(mutedColor).
			Background(bgSoftColor).
			Render(trimOrPad("No recent notes", innerWidth)))
	} else {
		timestampWidth := 24
		gapWidth := 2
		leftWidth := max(16, innerWidth-timestampWidth-gapWidth)

		for i, item := range recentItems {
			tag := "[note]"
			if item.IsTemp {
				tag = "[temp]"
			}

			numText := fmt.Sprintf("%d", i+1)
			num := lipgloss.NewStyle().
				Width(lipgloss.Width(numText)).
				Bold(true).
				Foreground(accentColor).
				Background(bgSoftColor).
				Render(numText)

			tagStyled := lipgloss.NewStyle().
				Width(lipgloss.Width(tag)).
				Foreground(mutedColor).
				Background(bgSoftColor).
				Render(tag)

			prefix := lipgloss.JoinHorizontal(
				lipgloss.Left,
				num,
				lipgloss.NewStyle().Width(2).Background(bgSoftColor).Render("  "),
				tagStyled,
				lipgloss.NewStyle().Width(1).Background(bgSoftColor).Render(" "),
			)

			prefixWidth := lipgloss.Width(prefix)
			titleWidth := max(8, leftWidth-prefixWidth)

			titleCol := lipgloss.NewStyle().
				Width(titleWidth).
				MaxWidth(titleWidth).
				Foreground(textColor).
				Background(bgSoftColor).
				Render(trimOrPad(item.Display, titleWidth))

			leftCol := lipgloss.NewStyle().
				Width(leftWidth).
				Background(bgSoftColor).
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
				Background(bgSoftColor).
				Render(trimOrPad(timeText, timestampWidth))

			topLine := lipgloss.JoinHorizontal(
				lipgloss.Top,
				leftCol,
				lipgloss.NewStyle().
					Width(gapWidth).
					Background(bgSoftColor).
					Render(strings.Repeat(" ", gapWidth)),
				timeCol,
			)

			pathText := trimOrPad(
				"    "+shortenDashboardPath(m.rootDir, item.Note.Path),
				innerWidth,
			)
			pathLine := lipgloss.NewStyle().
				Width(innerWidth).
				Foreground(mutedColor).
				Background(bgSoftColor).
				Render(pathText)

			recentLines = append(recentLines, topLine, pathLine)
		}
	}
	recentBlock := lipgloss.JoinVertical(lipgloss.Left, recentLines...)

	actionsLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentSoftColor).
		Width(innerWidth).
		Background(bgSoftColor).
		Render("Quick actions")

	actionLines := []string{
		dashboardActionLine("enter", "Open workspace", innerWidth),
		dashboardActionLine(keys.BracketForward.Help().Key, "Open Temporary", innerWidth),
		dashboardActionLine(keys.ShowPins.Help().Key, "Open Pins", innerWidth),
		dashboardActionLine(keys.NewTemporaryNote.Help().Key, "Create temporary note", innerWidth),
		dashboardActionLine("1-5", "Open recent note", innerWidth),
		dashboardActionLine(keys.Quit.Help().Key, "Quit", innerWidth),
	}
	actionsBlock := lipgloss.JoinVertical(lipgloss.Left, actionLines...)

	tipsLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentSoftColor).
		Width(innerWidth).
		Background(bgSoftColor).
		Render("Tip")

	tipsBlock := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(innerWidth).
		Background(bgSoftColor).
		Render("This dashboard is optional. Set dashboard = false in your TOML config to start directly in the main workspace.")

	warning := ""
	if m.startupError != "" {
		warning = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Width(innerWidth).
			Background(bgSoftColor).
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

	cardBody := surface.Render(
		fillWidthBackground(
			lipgloss.JoinVertical(lipgloss.Left, contentParts...),
			innerWidth,
			bgSoftColor,
		),
	)

	card := lipgloss.NewStyle().
		Width(cardWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		BorderBackground(bgSoftColor).
		Background(bgSoftColor).
		Render(cardBody)

	placed := lipgloss.Place(
		max(1, m.width),
		max(1, m.height),
		lipgloss.Center,
		lipgloss.Center,
		card,
		lipgloss.WithWhitespaceBackground(bgColor),
	)

	return lipgloss.NewStyle().
		Width(max(1, m.width)).
		Height(max(1, m.height)).
		Background(bgColor).
		Render(placed)
}

func (m Model) renderTreeView() string {
	if m.visibleTreeResultCount() == 0 {
		return m.renderPaneEmptyState(m.treeEmptyStateMessage())
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
	case treeNote, treeRemoteNote:
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

	markPrefix := "  "
	if m.isMarkedTreeItem(item) {
		markPrefix = "+ "
	}

	pinMark := "  "
	if pinned {
		pinMark = "★ "
	}

	encMark := ""
	if item.Kind == treeNote && item.Note != nil && item.Note.Encrypted {
		encMark = "[enc] "
	}

	syncMark := ""
	syncColor := textColor
	if item.Kind == treeNote && item.Note != nil {
		syncMark, syncColor = m.noteSyncMarker(item.Note)
	} else if item.Kind == treeRemoteNote {
		if item.RemoteNote != nil && m.syncInFlight[remoteOnlySyncVisualKey(item.RemoteNote.ID)] {
			syncMark, syncColor = m.blinkingSyncMarker()
		} else {
			syncMark = "x "
			syncColor = mutedColor
		}
	}

	indent := strings.Repeat("  ", item.Depth)
	leftPrefix := indent + markPrefix + pinMark
	rightPrefix := encMark + icon + " "
	prefixWidth := lipgloss.Width(leftPrefix) + lipgloss.Width(syncMark) + lipgloss.Width(rightPrefix)

	var tags []string
	if item.Kind == treeNote && item.Note != nil {
		tags = item.Note.Tags
	}

	availableWidth := rowWidth - 2 - prefixWidth

	titleWidth := availableWidth

	rowBg := bgSoftColor
	rowFg := textColor
	tagFg := mutedColor
	rowBold := false
	marked := m.isMarkedTreeItem(item)
	if selected {
		rowBg = selectedBgColor
		rowFg = selectedFgColor
		tagFg = selectedFgColor
		rowBold = boldSelected
		if item.Kind == treeCategory {
			rowFg = accentColor
		}
		if pinned {
			if item.Kind == treeCategory {
				rowFg = accentColor
			} else {
				rowFg = pinnedNoteColor
			}
		}
	} else {
		if item.Kind == treeCategory {
			rowFg = accentColor
		}
		if pinned {
			if item.Kind == treeCategory {
				rowFg = accentColor
			} else {
				rowFg = pinnedNoteColor
			}
		}
	}
	if item.Kind == treeRemoteNote {
		rowFg = mutedColor
		tagFg = mutedColor
	}
	if marked {
		rowFg = markedItemColor
		tagFg = markedItemColor
	}

	tagsPart := ""
	if len(tags) > 0 {
		maxTagsWidth := max(10, availableWidth*40/100)
		tagsPart, titleWidth = renderTagChips(
			tags,
			availableWidth,
			maxTagsWidth,
			tagFg,
			rowBg,
			rowBold,
		)
	}

	title := item.Name
	truncatedTitle := truncateToWidth(title, titleWidth)
	titlePadded := truncatedTitle + strings.Repeat(
		" ",
		max(0, titleWidth-lipgloss.Width(truncatedTitle)),
	)

	searchQuery := m.activeSearchQuery()
	matchBg := highlightBgColor
	matchFg := selectedFgColor
	if selected {
		matchBg = accentColor
		matchFg = bgColor
	}

	prefixStyle := lipgloss.NewStyle().
		Foreground(rowFg).
		Background(rowBg).
		Bold(rowBold)
	prefixPart := prefixStyle.Render(leftPrefix)
	if syncMark != "" {
		prefixPart += lipgloss.NewStyle().
			Foreground(syncColor).
			Background(rowBg).
			Bold(rowBold).
			Render(syncMark)
	}
	prefixPart += prefixStyle.Render(rightPrefix)
	titlePart := highlightSearchText(titlePadded, searchQuery, rowFg, rowBg, matchBg, matchFg)

	mainLine := lipgloss.NewStyle().
		Width(rowWidth).
		Padding(0, 1).
		Background(rowBg).
		Render(prefixPart + titlePart + tagsPart)

	// Build the hint line if present.
	hintLine := ""
	if item.MatchHint != "" {
		hintIndent := strings.Repeat(" ", prefixWidth+2)
		hintAvailable := max(8, rowWidth-2-lipgloss.Width(hintIndent))
		hintBg := bgSoftColor
		hintFg := mutedColor
		if selected {
			hintBg = rowBg
			hintFg = rowFg
		}

		if after, ok := strings.CutPrefix(item.MatchHint, "tag:"); ok {
			tagName := after
			hintPrefix := lipgloss.NewStyle().
				Background(hintBg).
				Render(hintIndent)
			hintTag := lipgloss.NewStyle().
				Foreground(accentSoftColor).
				Background(chipBgColor).
				Padding(0, 1).
				Render("tag: " + tagName)
			hintLine = fillWidthBackground(hintPrefix+hintTag, rowWidth, hintBg)
		} else {
			excerpt := truncateToWidth(item.MatchHint, hintAvailable-4)
			highlighted := highlightSearchText(excerpt, searchQuery, hintFg, hintBg, matchBg, matchFg)
			hintPrefix := lipgloss.NewStyle().
				Foreground(hintFg).
				Background(hintBg).
				Render(hintIndent + "… ")
			hintLine = fillWidthBackground(hintPrefix+highlighted, rowWidth, hintBg)
		}
	}

	if hintLine == "" {
		return mainLine
	}

	return lipgloss.JoinVertical(lipgloss.Left, mainLine, hintLine)
}

func renderTagChips(
	tags []string,
	availableWidth, maxTagsWidth int,
	fg, bg lipgloss.Color,
	bold bool,
) (string, int) {
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

	tagsStr := lipgloss.NewStyle().
		Foreground(fg).
		Background(bg).
		Bold(bold).
		Render(b.String())
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
		return m.renderPaneEmptyState(m.temporaryEmptyStateMessage())
	}

	lines := make([]string, 0, len(tempNotes))
	rowWidth := m.treeInnerWidth()

	for i, n := range tempNotes {
		pinMark := "  "
		if m.isPinnedTemporaryNote(n.RelPath) {
			pinMark = "★ "
		}

		label := trimOrPad(pinMark+iconNote+" "+n.Title(), rowWidth-2)
		marked := m.isMarkedTempNote(n.RelPath)

		if i == m.tempCursor {
			fg := selectedFgColor
			if m.isPinnedTemporaryNote(n.RelPath) {
				fg = pinnedNoteColor
			}
			if marked {
				fg = markedItemColor
			}
			lines = append(lines, lipgloss.NewStyle().
				Width(rowWidth).
				Padding(0, 1).
				Foreground(fg).
				Background(selectedBgColor).
				Bold(boldSelected).
				Render(label))
			continue
		}

		rowStyle := treeNoteStyle
		switch {
		case marked:
			rowStyle = rowStyle.Foreground(markedItemColor)
		case m.isPinnedTemporaryNote(n.RelPath):
			rowStyle = rowStyle.Foreground(pinnedNoteColor)
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
		return m.renderPaneEmptyState(m.pinsEmptyStateMessage())
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
			fg := pinnedNoteColor
			if item.Kind == pinItemCategory {
				fg = accentColor
			}
			lines = append(lines, lipgloss.NewStyle().
				Width(rowWidth).
				Padding(0, 1).
				Foreground(fg).
				Background(selectedBgColor).
				Bold(boldSelected).
				Render(plain))
			continue
		}

		fg := pinnedNoteColor
		if item.Kind == pinItemCategory {
			fg = accentColor
		}
		style := treeNoteStyle.Foreground(fg).Width(rowWidth).Padding(0, 1)
		lines = append(lines, style.Render(plain))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderSearchBar() string {
	width := m.treeInnerWidth()
	if m.searchMode || m.hasActiveSearch() {
		input := fillWidthBackground(strings.TrimRight(m.searchInput.View(), " "), width, bgSoftColor)
		metaText := truncateToWidth("Search active • "+m.renderSearchMeta(), width)
		meta := lipgloss.NewStyle().
			Foreground(accentSoftColor).
			Background(bgSoftColor).
			Render(metaText)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			input,
			fillWidthBackground(meta, width, bgSoftColor),
		)
	}
	hint := truncateToWidth("Press / to search titles, paths, tags, and preview text", width)
	return fillWidthBackground(mutedStyle.Render(hint), width, bgSoftColor)
}

func (m Model) renderStatus() string {
	if m.deletePending != nil {
		return statusErrStyle.Render("TRASH PENDING • press d to confirm • esc to cancel")
	}

	if m.startupError != "" {
		return statusErrStyle.Render("CONFIG ERROR • " + m.startupError)
	}

	parts := []string{m.renderModeSegment()}
	if workspace := m.renderWorkspaceSegment(); workspace != "" {
		parts = append(parts, workspace)
	}
	parts = append(parts,
		m.renderFocusSegment(),
		m.renderSelectionSegment(),
		m.renderPrivacySegment(),
		m.renderSortSegment(),
	)

	if filter := m.renderFilterSegment(); filter != "" {
		parts = append(parts, filter)
	}

	if preview := m.renderPreviewSegment(); preview != "" {
		parts = append(parts, preview)
	}

	if conflict := m.renderConflictSegment(); conflict != "" {
		parts = append(parts, conflict)
	}

	if syncHint := m.renderSelectedSyncIssueHint(); syncHint != "" {
		parts = append(parts, syncHint)
	}

	if m.status != "" {
		parts = append(parts, m.status)
	}

	line := strings.Join(parts, "  •  ")

	switch {
	case strings.HasPrefix(m.status, "error:"),
		strings.HasPrefix(m.status, "sync profile check failed:"),
		strings.HasPrefix(m.status, "editor error:"),
		strings.HasPrefix(m.status, "create failed:"),
		strings.HasPrefix(m.status, "category create failed:"),
		strings.HasPrefix(m.status, "delete failed:"),
		strings.HasPrefix(m.status, "rename failed:"),
		strings.HasPrefix(m.status, "move failed:"),
		strings.HasPrefix(m.status, "pin failed:"),
		strings.HasPrefix(m.status, "sync failed:"),
		strings.HasPrefix(m.status, "toggle sync failed:"),
		strings.HasPrefix(m.status, "sync import failed:"),
		strings.HasPrefix(m.status, "remote delete failed:"),
		strings.HasPrefix(m.status, "todo error:"),
		strings.HasPrefix(m.status, "encryption failed:"),
		strings.HasPrefix(m.status, "decryption failed:"),
		strings.HasPrefix(m.status, "re-encryption failed:"),
		strings.HasPrefix(m.status, "wrong passphrase"),
		strings.HasPrefix(m.status, "error reading note:"),
		strings.HasPrefix(m.status, "error opening note:"),
		strings.HasPrefix(m.status, "sync profile save failed:"),
		strings.HasPrefix(m.status, "sync root rebind failed:"),
		strings.HasPrefix(m.status, "sync debug copy failed:"),
		strings.HasPrefix(m.status, "restore failed:"):
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
	case m.showWorkspacePicker:
		return "WORKSPACE"
	case m.showCreateCategory:
		return "NEW CATEGORY"
	case m.showMoveBrowser:
		return "MOVE"
	case m.showMove:
		return "MOVE"
	case m.showRename:
		return "RENAME"
	case m.showSyncDebugModal:
		return "SYNC DEBUG"
	case m.showSyncProfilePicker:
		return "SYNC PROFILE"
	case m.showSyncProfileMigration:
		return "SYNC ROOT"
	case m.showTodoAdd:
		return "ADD TODO"
	case m.showTodoEdit:
		return "EDIT TODO"
	case m.showTodoDueDate:
		return "TODO DUE DATE"
	case m.showTodoPriority:
		return "TODO PRIORITY"
	case m.showPassphraseModal:
		return "PASSPHRASE"
	case m.showEncryptConfirm:
		return "CONFIRM"
	case m.showNoteHistory:
		return "HISTORY"
	case m.showTemplatePicker:
		return "TEMPLATE"
	case m.searchMode:
		switch m.listMode {
		case listModeTemporary:
			return "SEARCH TEMP"
		case listModePins:
			return "SEARCH PINS"
		case listModeTodos:
			return "SEARCH TODOS"
		default:
			return "SEARCH"
		}
	case m.focus == focusPreview:
		return "PREVIEW"
	case m.listMode == listModeTemporary:
		return "TEMP"
	case m.listMode == listModePins:
		return "PINS"
	case m.listMode == listModeTodos:
		return "TODOS"
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

	if m.listMode == listModeTodos {
		item := m.currentTodoItem()
		if item == nil {
			return "todo: none"
		}
		text := strings.TrimSpace(item.Todo.DisplayText)
		if text == "" {
			text = strings.TrimSpace(item.Todo.Text)
		}
		if text == "" {
			text = item.Source
		}
		return "todo: " + truncateToWidth(text, 48)
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
	case treeRemoteNote:
		if item.RemoteNote == nil {
			return "remote note"
		}
		return "remote note: " + remoteOnlyNoteTitle(*item.RemoteNote)
	}

	return "selection"
}

func (m Model) renderConflictSegment() string {
	if !m.hasConflictCopyForCurrentSelection() {
		return ""
	}
	return "conflict: press " + keys.OpenConflictCopy.Help().Key + " to resolve"
}

func (m Model) renderFilterSegment() string {
	filter := m.activeSearchQuery()
	if filter == "" {
		return ""
	}

	parts := []string{
		"filter: " + filter,
		formatCountLabel(m.currentSearchResultCount(), "result"),
	}
	if m.previewSupportsSearchMatches() {
		parts = append(parts, formatCountLabel(len(m.previewMatches), "preview match"))
	}
	return strings.Join(parts, " · ")
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
	modalWidth, innerWidth := m.modalDimensions(60, 96)
	maxRows := max(8, min(20, m.height-16))
	query := m.helpInput.Value()
	if m.helpModalCache != "" && m.helpModalCacheQuery == query && m.helpModalCacheWidth == m.width && m.helpModalCacheHeight == m.height && m.helpModalCacheRows == maxRows && m.helpModalCacheScroll == m.helpScroll {
		return m.helpModalCache
	}

	var rows []string
	if m.helpRowsCache != nil && m.helpRowsCacheQuery == query && m.helpRowsCacheWidth == innerWidth {
		rows = m.helpRowsCache
	} else {
		rows = m.renderedHelpRows(innerWidth, false)
	}
	scroll := m.helpScroll
	maxScroll := max(0, len(rows)-maxRows)
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	body := m.helpBodyCache
	if body == "" || m.helpBodyCacheQuery != query || m.helpBodyCacheWidth != innerWidth || m.helpBodyCacheRows != maxRows || m.helpBodyCacheScroll != scroll {
		end := min(len(rows), scroll+maxRows)
		visibleRows := append([]string{}, rows[scroll:end]...)
		for len(visibleRows) < maxRows {
			visibleRows = append(visibleRows, m.renderModalBlank(innerWidth))
		}
		body = lipgloss.NewStyle().Width(innerWidth).Height(maxRows).Background(modalBgColor).Render(lipgloss.JoinVertical(lipgloss.Left, visibleRows...))
	}

	scrollText := "all"
	if len(rows) > maxRows {
		scrollText = fmt.Sprintf("%d-%d of %d", scroll+1, min(len(rows), scroll+maxRows), len(rows))
	}
	filterHint := "Type to filter by section, key, or description"
	if strings.TrimSpace(query) != "" {
		filterHint = "Filtered help"
	}

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.renderModalTitle("Help", innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalHint(filterHint+" • "+scrollText, innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalInputRow("Filter", m.helpInput, innerWidth),
				m.renderModalBlank(innerWidth),
				body,
				m.renderModalBlank(innerWidth),
				m.renderModalFooter("Type to filter • up/down scroll • home/end top/bottom • ctrl+d/u page • mouse wheel • esc to close", innerWidth),
			),
		)

	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderMoveBrowserModal() string {
	modalWidth, innerWidth := m.modalDimensions(60, 88)
	summary := m.moveSelectionSummary()
	destination := "~/notes"
	if relPath := m.currentMoveDestinationPath(); relPath != "" {
		destination = filepath.Join("~/notes", relPath)
	}

	summaryLines := []string{
		fmt.Sprintf("Notes: %d", summary.notes),
		fmt.Sprintf("Categories: %d", summary.categories),
		"Destination: " + destination,
	}
	lines := make([]string, 0, len(summaryLines))
	for _, line := range summaryLines {
		lines = append(lines, lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(line))
	}

	list := m.renderMoveDestinationList(innerWidth, min(14, max(6, m.height-20)))
	errorLine := ""
	if strings.TrimSpace(m.moveBrowserError) != "" {
		errorLine = lipgloss.NewStyle().
			Width(innerWidth).
			Background(modalBgColor).
			Render(modalErrorStyle.Render(m.moveBrowserError))
	}

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.renderModalTitle(m.moveBrowserTitle(), innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalHint(m.moveBrowserHint(), innerWidth),
				m.renderModalBlank(innerWidth),
				lipgloss.JoinVertical(lipgloss.Left, lines...),
				m.renderModalBlank(innerWidth),
				list,
				func() string {
					if errorLine == "" {
						return ""
					}
					return lipgloss.JoinVertical(lipgloss.Left, m.renderModalBlank(innerWidth), errorLine)
				}(),
				m.renderModalBlank(innerWidth),
				m.renderModalFooter(m.moveBrowserFooter(), innerWidth),
			),
		)

	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderMoveDestinationList(innerWidth, maxRows int) string {
	items := m.moveDestinationItems()
	if len(items) == 0 {
		return lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render("(no categories)")
	}

	cursor := m.moveDestCursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(items) {
		cursor = len(items) - 1
	}

	start := 0
	if len(items) > maxRows {
		start = max(0, cursor-maxRows/2)
		if start+maxRows > len(items) {
			start = len(items) - maxRows
		}
	}
	end := min(len(items), start+maxRows)

	lines := make([]string, 0, end-start+2)
	if start > 0 {
		lines = append(lines, lipgloss.NewStyle().Width(innerWidth).Foreground(modalMutedColor).Background(modalBgColor).Render("..."))
	}
	for i := start; i < end; i++ {
		lines = append(lines, m.renderMoveDestinationLine(items[i], i == cursor, innerWidth))
	}
	if end < len(items) {
		lines = append(lines, lipgloss.NewStyle().Width(innerWidth).Foreground(modalMutedColor).Background(modalBgColor).Render("..."))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderMoveDestinationLine(item treeItem, selected bool, width int) string {
	icon := iconCategoryLeaf
	if m.categoryHasChildren(item.RelPath) {
		if item.Expanded || item.RelPath == "" {
			icon = iconCategoryExpanded
		} else {
			icon = iconCategoryCollapsed
		}
	}

	indent := strings.Repeat("  ", item.Depth)
	pinMark := "  "
	if item.RelPath != "" && m.isPinnedCategory(item.RelPath) {
		pinMark = "★ "
	}
	prefix := indent + pinMark + icon + " "
	prefixWidth := lipgloss.Width(prefix)
	rowBg := modalBgColor
	rowFg := accentColor
	if selected {
		rowBg = selectedBgColor
		rowFg = selectedFgColor
	}
	nameWidth := max(4, width-2-prefixWidth)
	name := truncateToWidth(item.Name, nameWidth)
	name += strings.Repeat(" ", max(0, nameWidth-lipgloss.Width(name)))
	prefixPart := lipgloss.NewStyle().Foreground(rowFg).Background(rowBg).Render(prefix)
	namePart := lipgloss.NewStyle().Foreground(rowFg).Background(rowBg).Width(nameWidth).Render(name)
	return lipgloss.NewStyle().Width(width).Padding(0, 1).Background(rowBg).Render(prefixPart + namePart)
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

func (m Model) renderAddTagModal() string {
	return m.renderStandardModal(
		"Add tag",
		"Add one or more tags to the selected note. Separate multiple tags with commas.",
		"Tags",
		m.tagInput,
		"Enter to add • Esc to cancel",
	)
}

func (m Model) renderTodoAddModal() string {
	return m.renderStandardModal(
		"Add todo item",
		"Type the text for your new todo item.",
		"Todo",
		m.todoInput,
		"Enter to add • Esc to cancel",
	)
}

func (m Model) renderTodoEditModal() string {
	return m.renderStandardModal(
		"Edit todo item",
		"Edit the text of the selected todo item.",
		"Todo",
		m.todoInput,
		"Enter to save • Esc to cancel",
	)
}

func (m Model) renderTodoDueDateModal() string {
	return m.renderStandardModal(
		"Set todo due date",
		"Use YYYY-MM-DD. Leave empty to clear the current due date.",
		"Due date",
		m.dueDateInput,
		"Enter to save • Esc to cancel",
	)
}

func (m Model) renderTodoPriorityModal() string {
	return m.renderStandardModal(
		"Set todo priority",
		"Use a positive number. Leave empty to clear the current priority.",
		"Priority",
		m.priorityInput,
		"Enter to save • Esc to cancel",
	)
}

func (m Model) renderPassphraseModal() string {
	title := "Enter passphrase"
	hint := "This passphrase protects all encrypted notes in this session."
	if m.passphraseModalCtx == "encrypt" {
		title = "Set passphrase"
	}
	return m.renderStandardModal(
		title,
		hint,
		"Passphrase",
		m.passphraseInput,
		"Enter to confirm • Esc to cancel",
	)
}

func (m Model) renderEncryptConfirmModal() string {
	modalWidth, innerWidth := m.modalDimensions(40, 64)

	bodyText := "The note body will be encrypted on disk."
	confirmTitle := "Encrypt this note?"
	if m.passphraseModalCtx == "decrypt" {
		confirmTitle = "Remove encryption?"
		bodyText = "The note will be saved as plaintext on disk."
	}

	yesStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(modalBgColor).
		Background(modalBgColor)
	noStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(modalBgColor).
		Background(modalBgColor)

	if m.encryptConfirmYes {
		yesStyle = yesStyle.
			Background(selectedBgColor).
			Foreground(selectedFgColor).
			BorderForeground(selectedBgColor)
		noStyle = noStyle.
			Foreground(mutedColor).
			BorderForeground(borderColor)
	} else {
		yesStyle = yesStyle.
			Foreground(mutedColor).
			BorderForeground(borderColor)
		noStyle = noStyle.
			Background(selectedBgColor).
			Foreground(selectedFgColor).
			BorderForeground(selectedBgColor)
	}

	yesBtn := yesStyle.Render("Yes")
	noBtn := noStyle.Render("No")
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, yesBtn, m.renderModalBlockGap(2, max(lipgloss.Height(yesBtn), lipgloss.Height(noBtn))), noBtn)

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.renderModalTitle(confirmTitle, innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalHint(bodyText, innerWidth),
				m.renderModalBlank(innerWidth),
				lipgloss.NewStyle().
					Width(innerWidth).
					Background(modalBgColor).
					Render(buttons),
				m.renderModalBlank(innerWidth),
				m.renderModalFooter(
					"left/right to switch • Enter to confirm • Esc to cancel",
					innerWidth,
				),
			),
		)

	return modalCardStyle(modalWidth).Render(content)
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
	return m.renderModalGap(innerWidth)
}

func (m Model) renderModalGap(width int) string {
	return m.renderModalBlockGap(width, 1)
}

func (m Model) renderModalBlockGap(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	line := lipgloss.NewStyle().
		Width(width).
		Background(modalBgColor).
		Render(strings.Repeat(" ", width))
	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderModalInputRow(label string, input textinput.Model, innerWidth int) string {
	local := input
	local.Prompt = ""
	labelWidth := 12
	fieldInnerWidth := max(12, min(36, innerWidth-20))
	fieldOuterWidth := fieldInnerWidth + 4
	local.Width = fieldInnerWidth

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
		Width(labelWidth).
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

	rawInput := local.View()
	trimmedInput := strings.TrimRight(rawInput, " ")
	inputPad := max(0, fieldInnerWidth-lipgloss.Width(trimmedInput))
	inputView := trimmedInput + lipgloss.NewStyle().
		Width(inputPad).
		Background(modalBgColor).
		Render(strings.Repeat(" ", inputPad))

	inputField := lipgloss.NewStyle().
		Width(fieldOuterWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(modalAccentColor).
		BorderBackground(modalBgColor).
		Background(modalBgColor).
		Padding(0, 1).
		Render(inputView)

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
	keyWidth := 18
	descWidth := max(10, width-keyWidth)

	keyPart := lipgloss.NewStyle().
		Width(keyWidth).
		Background(modalBgColor).
		Render(
			modalKeyStyle.
				Width(keyWidth).
				Align(lipgloss.Left).
				Render(k),
		)

	descPart := lipgloss.NewStyle().
		Width(descWidth).
		Background(modalBgColor).
		Render(
			modalTextStyle.
				Width(descWidth).
				Align(lipgloss.Right).
				Render(desc),
		)

	return lipgloss.NewStyle().
		Width(width).
		Background(modalBgColor).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, keyPart, descPart))
}

func (m Model) previewView() string {
	return fillWidthBackground(m.preview.View(), m.preview.Width, bgSoftColor)
}

func fillWidthBackground(content string, width int, bg lipgloss.Color) string {
	if width <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	padStyle := lipgloss.NewStyle().Background(bg)

	for i, line := range lines {
		trimmed := strings.TrimRight(line, " ")
		w := lipgloss.Width(trimmed)
		if w < width {
			lines[i] = trimmed + padStyle.Render(strings.Repeat(" ", width-w))
		} else {
			lines[i] = trimmed
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) leftPanelTitle() string {
	var title string
	switch m.listMode {
	case listModeTemporary:
		title = fmt.Sprintf("Temporary (%d)", len(m.filteredTempNotes()))
		if marked := m.currentMarkedCount(); marked > 0 {
			title += fmt.Sprintf(" • marked %d", marked)
		}
	case listModePins:
		title = fmt.Sprintf("Pins (%d)", len(m.filteredPinnedItems()))
	case listModeTodos:
		title = fmt.Sprintf("Todos (%d)", len(m.filteredTodoItems()))
	default:
		title = fmt.Sprintf("Tree (%d)", m.visibleTreeResultCount())
		if marked := m.markedTreeCount(); marked > 0 {
			title += fmt.Sprintf(" • marked %d", marked)
		}
	}
	if m.focus == focusTree {
		title += " • focused"
	}
	return title
}

func (m Model) rightPanelTitle() string {
	title := "Preview"
	if m.previewSupportsSearchMatches() {
		title += fmt.Sprintf(" (%d matches)", len(m.previewMatches))
	}
	if m.focus == focusPreview {
		title += " • focused"
	}
	return title
}

func (m Model) renderLeftPaneHint(width int) string {
	hint := "tab to focus the list"
	if m.focus == focusTree {
		switch m.listMode {
		case listModeTemporary:
			hint = "focused • / search • M promote • ctrl+a archive • t back to notes"
		case listModePins:
			hint = "focused • / search • p pin current note or category"
		case listModeTodos:
			hint = "focused • / search • j/k tasks • tt toggle • enter jump to note"
		default:
			hint = "focused • / search • j/k move • tab to preview"
		}
	}
	style := mutedStyle.Background(bgSoftColor)
	if m.focus == focusTree {
		style = style.Foreground(accentSoftColor)
	}
	return fillWidthBackground(style.Render(truncateToWidth(hint, width)), width, bgSoftColor)
}

func (m Model) renderRightPaneHint(width int) string {
	hint := "tab to focus preview"
	if m.focus == focusPreview {
		parts := []string{"focused", "ctrl+d/u scroll", "B privacy", "L lines"}
		if m.previewSupportsSearchMatches() {
			parts = append(parts, "n/N matches")
		}
		if len(m.previewTodos) > 0 {
			parts = append(parts, "]t/[t todos")
		}
		hint = strings.Join(parts, " • ")
	}
	style := mutedStyle.Background(bgSoftColor)
	if m.focus == focusPreview {
		style = style.Foreground(accentSoftColor)
	}
	return fillWidthBackground(style.Render(truncateToWidth(hint, width)), width, bgSoftColor)
}

func (m Model) renderLeftPaneBody() string {
	switch m.listMode {
	case listModeTemporary:
		return m.renderTemporaryListView()
	case listModePins:
		return m.renderPinsListView()
	case listModeTodos:
		return m.renderTodoListView()
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
	keyWidth := lipgloss.Width(keyText)
	if keyWidth > width {
		keyWidth = width
	}

	sepWidth := 2
	if keyWidth+sepWidth > width {
		sepWidth = max(0, width-keyWidth)
	}
	descWidth := max(0, width-keyWidth-sepWidth)

	keyPart := lipgloss.NewStyle().
		Width(keyWidth).
		Bold(true).
		Foreground(accentColor).
		Background(bgSoftColor).
		Render(trimOrPad(keyText, keyWidth))

	sepPart := lipgloss.NewStyle().
		Width(sepWidth).
		Background(bgSoftColor).
		Render(strings.Repeat(" ", sepWidth))

	descPart := lipgloss.NewStyle().
		Width(descWidth).
		Foreground(textColor).
		Background(bgSoftColor).
		Render(trimOrPad(desc, descWidth))

	return lipgloss.JoinHorizontal(lipgloss.Left, keyPart, sepPart, descPart)
}

func dashboardSummaryLine(label, value string, width int) string {
	labelWidth := lipgloss.Width(label)
	if labelWidth > width {
		labelWidth = width
	}

	sepWidth := 1
	if labelWidth+sepWidth > width {
		sepWidth = max(0, width-labelWidth)
	}
	valueWidth := max(0, width-labelWidth-sepWidth)

	labelPart := lipgloss.NewStyle().
		Width(labelWidth).
		Foreground(mutedColor).
		Background(bgSoftColor).
		Render(trimOrPad(label, labelWidth))

	sepPart := lipgloss.NewStyle().
		Width(sepWidth).
		Background(bgSoftColor).
		Render(strings.Repeat(" ", sepWidth))

	valuePart := lipgloss.NewStyle().
		Width(valueWidth).
		Foreground(textColor).
		Bold(true).
		Background(bgSoftColor).
		Render(trimOrPad(value, valueWidth))

	return lipgloss.JoinHorizontal(lipgloss.Left, labelPart, sepPart, valuePart)
}

func (m Model) renderSortSegment() string {
	label := map[string]string{
		sortAlpha:    "alpha",
		sortModified: "modified",
		sortCreated:  "created",
		sortSize:     "size",
	}[m.sortMethod]
	if label == "" {
		label = "alpha"
	}
	if m.sortReverse {
		label += " ↑"
	}
	return "sort: " + label
}

// placeOverlay composites the modal string centered over the base string using
// ANSI-aware line splicing. Unlike lipgloss.Place with whitespace options, this
// preserves the base view content visible around the modal border.
func placeOverlay(base, modal string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	modalLines := strings.Split(modal, "\n")
	modalH := len(modalLines)
	modalW := 0
	for _, l := range modalLines {
		if w := lipgloss.Width(l); w > modalW {
			modalW = w
		}
	}
	startY := (height - modalH) / 2
	startX := (width - modalW) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}
	for i, overlayLine := range modalLines {
		baseIdx := startY + i
		if baseIdx < 0 || baseIdx >= len(baseLines) {
			continue
		}
		baseLine := baseLines[baseIdx]
		left := xansi.Truncate(baseLine, startX, "")
		right := xansi.TruncateLeft(baseLine, startX+modalW, "")
		baseLines[baseIdx] = left + overlayLine + right
	}
	return strings.Join(baseLines, "\n")
}

func (m Model) renderFocusSegment() string {
	if m.focus == focusPreview {
		return "focus: preview"
	}
	return "focus: list"
}

func (m Model) activeSearchQuery() string {
	return strings.TrimSpace(m.searchInput.Value())
}

func (m Model) hasActiveSearch() bool {
	return m.activeSearchQuery() != ""
}

func (m Model) visibleTreeResultCount() int {
	count := 0
	for _, item := range m.treeItems {
		if item.Kind == treeCategory && item.RelPath == "" {
			continue
		}
		count++
	}
	return count
}

func (m Model) currentSearchResultCount() int {
	switch m.listMode {
	case listModeTemporary:
		return len(m.filteredTempNotes())
	case listModePins:
		return len(m.filteredPinnedItems())
	case listModeTodos:
		return len(m.filteredTodoItems())
	default:
		return m.visibleTreeResultCount()
	}
}

func formatCountLabel(count int, label string) string {
	suffix := "s"
	if count == 1 {
		suffix = ""
	} else if strings.HasSuffix(label, "match") {
		suffix = "es"
	}
	return fmt.Sprintf("%d %s%s", count, label, suffix)
}

func (m Model) renderSearchMeta() string {
	parts := []string{formatCountLabel(m.currentSearchResultCount(), "result")}
	if m.previewSupportsSearchMatches() {
		parts = append(parts, formatCountLabel(len(m.previewMatches), "preview match"))
	}
	parts = append(parts, "esc clears")
	return strings.Join(parts, " • ")
}

func (m Model) previewSupportsSearchMatches() bool {
	query := m.activeSearchQuery()
	if query == "" || strings.HasPrefix(strings.ToLower(query), "#") || strings.TrimSpace(m.previewPath) == "" {
		return false
	}
	return !strings.HasPrefix(m.previewPath, "category:") &&
		!strings.HasPrefix(m.previewPath, "pinned-category:") &&
		!strings.HasPrefix(m.previewPath, "remote:")
}

func (m Model) renderPaneEmptyState(message string) string {
	return lipgloss.NewStyle().
		Width(m.treeInnerWidth()).
		Foreground(mutedColor).
		Background(bgSoftColor).
		Italic(true).
		Render(message)
}

func (m Model) treeEmptyStateMessage() string {
	if query := m.activeSearchQuery(); query != "" {
		return fmt.Sprintf("No notes match %q. Press esc to clear search.", query)
	}
	return "No notes yet. Press n for a note, T for a todo, C for a category, or N for temp."
}

func (m Model) temporaryEmptyStateMessage() string {
	if query := m.activeSearchQuery(); query != "" {
		return fmt.Sprintf("No temporary notes match %q. Press esc to clear search.", query)
	}
	return "No temporary notes. Press N to create one or t to return to notes."
}

func (m Model) renderNoteHistoryModal() string {
	modalWidth, innerWidth := m.modalDimensions(62, 96)
	maxVisible := max(4, min(20, m.height-14))

	title := m.noteHistoryRelPath
	if title == "" {
		title = "note"
	}
	subtitle := fmt.Sprintf("%d version", len(m.noteHistoryEntries))
	if len(m.noteHistoryEntries) != 1 {
		subtitle += "s"
	}

	start := 0
	if m.noteHistoryCursor >= maxVisible {
		start = m.noteHistoryCursor - maxVisible + 1
	}
	end := min(start+maxVisible, len(m.noteHistoryEntries))

	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		entry := m.noteHistoryEntries[i]
		prefix := "  "
		if i == m.noteHistoryCursor {
			prefix = "› "
		}
		ts := entry.Timestamp.Local().Format("2006-01-02 15:04:05")
		label := entry.FirstLine
		if label == "" {
			label = entry.ID
		}
		line := prefix + ts + "  " + label
		bg := modalBgColor
		fg := textColor
		if i == m.noteHistoryCursor {
			bg = selectedBgColor
			fg = selectedFgColor
		}
		rows = append(rows, lipgloss.NewStyle().Width(innerWidth).Padding(0, 1).Foreground(fg).Background(bg).Render(line))
	}
	if len(rows) == 0 {
		rows = append(rows, lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Foreground(mutedColor).Render("  No versions found"))
	}

	content := lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderModalTitle("Note Version History", innerWidth),
			m.renderModalHint(title+" • "+subtitle, innerWidth),
			m.renderModalBlank(innerWidth),
			lipgloss.JoinVertical(lipgloss.Left, rows...),
			m.renderModalBlank(innerWidth),
			m.renderModalFooter("up/down navigate • enter restore • esc close", innerWidth),
		),
	)
	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderTrashBrowserModal() string {
	modalWidth, innerWidth := m.modalDimensions(62, 96)
	maxVisible := max(4, min(20, m.height-14))

	subtitle := fmt.Sprintf("%d item", len(m.trashBrowserItems))
	if len(m.trashBrowserItems) != 1 {
		subtitle += "s"
	}

	start := 0
	if m.trashBrowserCursor >= maxVisible {
		start = m.trashBrowserCursor - maxVisible + 1
	}
	end := min(start+maxVisible, len(m.trashBrowserItems))

	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		item := m.trashBrowserItems[i]
		prefix := "  "
		if i == m.trashBrowserCursor {
			prefix = "› "
		}
		rel, _ := filepath.Rel(m.rootDir, filepath.Dir(item.OriginalPath))
		label := item.Name
		if rel != "" && rel != "." {
			label = rel + "/" + label
		}
		ts := item.DeletionDate.Local().Format("2006-01-02 15:04")
		line := prefix + ts + "  " + label
		bg := modalBgColor
		fg := textColor
		if i == m.trashBrowserCursor {
			bg = selectedBgColor
			fg = selectedFgColor
		}
		rows = append(rows, lipgloss.NewStyle().Width(innerWidth).Padding(0, 1).
			Foreground(fg).Background(bg).Render(line))
	}
	if len(rows) == 0 {
		rows = append(rows, lipgloss.NewStyle().Width(innerWidth).
			Background(modalBgColor).Foreground(mutedColor).Render("  Trash is empty"))
	}

	content := lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderModalTitle("Trash Browser", innerWidth),
			m.renderModalHint(subtitle, innerWidth),
			m.renderModalBlank(innerWidth),
			lipgloss.JoinVertical(lipgloss.Left, rows...),
			m.renderModalBlank(innerWidth),
			m.renderModalFooter("up/down navigate • enter restore • esc close", innerWidth),
		),
	)
	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderTemplatePickerModal() string {
	modalWidth, innerWidth := m.modalDimensions(52, 78)

	var rows []string
	if m.templatePickerEditMode {
		// Edit mode: list only templates, no "Blank note" entry.
		for i, tmpl := range m.templateItems {
			prefix := "  "
			if i == m.templatePickerCursor {
				prefix = "> "
			}
			bg := modalBgColor
			fg := textColor
			if i == m.templatePickerCursor {
				bg = selectedBgColor
				fg = selectedFgColor
			}
			rows = append(rows, lipgloss.NewStyle().
				Width(innerWidth).
				Padding(0, 1).
				Foreground(fg).
				Background(bg).
				Render(prefix+tmpl.Name))
		}
	} else {
		// Create mode: index 0 is "Blank note", indices 1..N are templates.
		total := len(m.templateItems) + 1
		for i := 0; i < total; i++ {
			label := "Blank note"
			if i > 0 {
				label = m.templateItems[i-1].Name
			}
			prefix := "  "
			if i == m.templatePickerCursor {
				prefix = "> "
			}
			bg := modalBgColor
			fg := textColor
			if i == m.templatePickerCursor {
				bg = selectedBgColor
				fg = selectedFgColor
			}
			rows = append(rows, lipgloss.NewStyle().
				Width(innerWidth).
				Padding(0, 1).
				Foreground(fg).
				Background(bg).
				Render(prefix+label))
		}
	}

	title := "New note"
	hint := "Choose a template or start with a blank note."
	footer := "up/down navigate  enter confirm  e edit  esc cancel"
	if m.templatePickerEditMode {
		title = "Edit template"
		hint = "Select a template to open it in your editor."
		footer = "up/down navigate  enter open  esc cancel"
	}

	content := lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderModalTitle(title, innerWidth),
			m.renderModalHint(hint, innerWidth),
			m.renderModalBlank(innerWidth),
			lipgloss.JoinVertical(lipgloss.Left, rows...),
			m.renderModalBlank(innerWidth),
			m.renderModalFooter(footer, innerWidth),
		),
	)
	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) pinsEmptyStateMessage() string {
	if query := m.activeSearchQuery(); query != "" {
		return fmt.Sprintf("No pinned items match %q. Press esc to clear search.", query)
	}
	return "No pinned items. Press p on a note or category to pin it here."
}

func highlightMatch(text, query string) string {
	return highlightSearchText(text, query, mutedColor, bgSoftColor, highlightBgColor, selectedFgColor)
}

func highlightSearchText(
	text, query string,
	baseFg, baseBg, matchBg, matchFg lipgloss.Color,
) string {
	base := lipgloss.NewStyle().
		Foreground(baseFg).
		Background(baseBg)

	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" || strings.HasPrefix(q, "#") {
		return base.Render(text)
	}

	terms := strings.Fields(q)
	if len(terms) == 0 {
		return base.Render(text)
	}

	type interval struct{ start, end int }
	var intervals []interval
	lower := strings.ToLower(text)
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
		return base.Render(text)
	}

	merged := []interval{intervals[0]}
	for _, iv := range intervals[1:] {
		last := &merged[len(merged)-1]
		if iv.start <= last.end {
			if iv.end > last.end {
				last.end = iv.end
			}
			continue
		}
		merged = append(merged, iv)
	}

	var b strings.Builder
	pos := 0
	for _, iv := range merged {
		if pos < iv.start {
			b.WriteString(base.Render(text[pos:iv.start]))
		}
		b.WriteString(lipgloss.NewStyle().
			Background(matchBg).
			Foreground(matchFg).
			Render(text[iv.start:iv.end]))
		pos = iv.end
	}
	if pos < len(text) {
		b.WriteString(base.Render(text[pos:]))
	}
	return b.String()
}
