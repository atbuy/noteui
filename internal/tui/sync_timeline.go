package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	notesync "atbuy/noteui/internal/sync"
)

func (m Model) renderSyncTimelineModal() string {
	modalWidth, innerWidth := m.modalDimensions(64, 100)
	maxVisible := max(4, min(20, m.height-14))

	events := m.syncTimelineEvents
	start := m.syncTimelineOffset
	if start >= len(events) {
		start = max(0, len(events)-1)
	}
	end := min(start+maxVisible, len(events))

	rows := make([]string, 0, maxVisible)
	for i := start; i < end; i++ {
		rows = append(rows, m.renderSyncTimelineRow(events[i], innerWidth))
	}
	if len(rows) == 0 {
		rows = append(rows, lipgloss.NewStyle().
			Width(innerWidth).
			Padding(0, 1).
			Background(modalBgColor).
			Foreground(mutedColor).
			Render("No sync runs recorded yet."))
	}

	scrollHint := ""
	total := len(events)
	if total > maxVisible {
		scrollHint = fmt.Sprintf(" (%d/%d)", start+1, total)
	}

	content := lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderModalTitle("Sync Timeline", innerWidth),
			m.renderModalHint(fmt.Sprintf("Recent sync runs for this workspace%s", scrollHint), innerWidth),
			m.renderModalBlank(innerWidth),
			lipgloss.JoinVertical(lipgloss.Left, rows...),
			m.renderModalBlank(innerWidth),
			m.renderModalFooter("j/k scroll  Esc close", innerWidth),
		),
	)
	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderSyncTimelineRow(event notesync.SyncEvent, width int) string {
	icon, iconColor := syncTimelineIcon(event.Type)

	ts := ""
	if !event.Timestamp.IsZero() {
		ts = event.Timestamp.Local().Format("2006-01-02 15:04")
	}
	summary := syncTimelineSummary(event)

	// base style: background only; inner pieces use foreground on top of this
	base := lipgloss.NewStyle().Background(modalBgColor)
	// styled space/indent used as separators so no unstyled gaps appear between pieces
	spc := base.Render("  ")
	indent := base.Render("    ")

	iconStr := base.Foreground(iconColor).Render(icon)
	tsStr := base.Foreground(mutedColor).Render(ts)

	// Top line: icon  timestamp  [profile]
	top := iconStr + spc + tsStr
	if event.ProfileName != "" {
		top += spc + base.Foreground(mutedColor).Render("["+event.ProfileName+"]")
	}

	// Bottom line: indent summary  duration
	summaryStr := base.Foreground(textColor).Render(summary)
	bottom := indent + summaryStr
	if event.DurationMs > 0 {
		bottom += spc + base.Foreground(mutedColor).Render(fmt.Sprintf("%dms", event.DurationMs))
	}

	// innerWidth accounts for the 1-space padding we add manually on each side
	innerWidth := width - 2
	topLine := lipgloss.PlaceHorizontal(innerWidth, lipgloss.Left, top,
		lipgloss.WithWhitespaceBackground(modalBgColor))
	bottomLine := lipgloss.PlaceHorizontal(innerWidth, lipgloss.Left, bottom,
		lipgloss.WithWhitespaceBackground(modalBgColor))

	pad := base.Render(" ")
	return pad + topLine + pad + "\n" + pad + bottomLine + pad
}

func syncTimelineIcon(t notesync.SyncEventType) (string, lipgloss.Color) {
	switch t {
	case notesync.SyncEventSuccess:
		return "✓", successColor
	case notesync.SyncEventConflict:
		return "⚡", syncingNoteColor
	case notesync.SyncEventError:
		return "✗", errorColor
	default:
		return "·", mutedColor
	}
}

func syncTimelineSummary(event notesync.SyncEvent) string {
	switch event.Type {
	case notesync.SyncEventError:
		msg := "sync failed"
		if event.ErrorMsg != "" {
			msg += ": " + truncateLine(event.ErrorMsg, 60)
		}
		return msg
	case notesync.SyncEventConflict:
		parts := buildSyncCounts(event)
		parts = append(parts, fmt.Sprintf("%d conflict(s)", event.Conflicts))
		return strings.Join(parts, ", ")
	default:
		parts := buildSyncCounts(event)
		if len(parts) == 0 {
			return "up to date"
		}
		return strings.Join(parts, ", ")
	}
}

func buildSyncCounts(event notesync.SyncEvent) []string {
	var parts []string
	if event.RegisteredNotes > 0 {
		parts = append(parts, fmt.Sprintf("%d registered", event.RegisteredNotes))
	}
	if event.UpdatedNotes > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", event.UpdatedNotes))
	}
	return parts
}

func truncateLine(s string, max int) string {
	s = strings.SplitN(s, "\n", 2)[0]
	if len(s) > max {
		return s[:max-1] + "..."
	}
	return s
}
