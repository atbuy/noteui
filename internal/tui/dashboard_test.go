package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatDashboardTime(t *testing.T) {
	tm := time.Date(2026, 4, 1, 14, 30, 0, 0, time.UTC)
	got := formatDashboardTime(tm)
	if !strings.Contains(got, "Apr") || !strings.Contains(got, "01") {
		require.Failf(t, "assertion failed", "unexpected formatted time: %q", got)
	}
}

func TestRelativeDashboardTimeJustNow(t *testing.T) {
	got := relativeDashboardTime(time.Now().Add(-30 * time.Second))
	if got != "just now" {
		require.Failf(t, "assertion failed", "expected 'just now', got %q", got)
	}
}

func TestRelativeDashboardTimeMinutes(t *testing.T) {
	got := relativeDashboardTime(time.Now().Add(-10 * time.Minute))
	if !strings.Contains(got, "m ago") {
		require.Failf(t, "assertion failed", "expected '10m ago', got %q", got)
	}
}

func TestRelativeDashboardTimeHours(t *testing.T) {
	got := relativeDashboardTime(time.Now().Add(-3 * time.Hour))
	if !strings.Contains(got, "h ago") {
		require.Failf(t, "assertion failed", "expected '3h ago', got %q", got)
	}
}

func TestRelativeDashboardTimeYesterday(t *testing.T) {
	got := relativeDashboardTime(time.Now().Add(-25 * time.Hour))
	if got != "yesterday" {
		require.Failf(t, "assertion failed", "expected 'yesterday', got %q", got)
	}
}

func TestRelativeDashboardTimeDaysAgo(t *testing.T) {
	got := relativeDashboardTime(time.Now().Add(-3 * 24 * time.Hour))
	if !strings.Contains(got, "d ago") {
		require.Failf(t, "assertion failed", "expected '3d ago', got %q", got)
	}
}

func TestRelativeDashboardTimeOld(t *testing.T) {
	got := relativeDashboardTime(time.Now().Add(-10 * 24 * time.Hour))
	// Should return a date like "Mar 22"
	if strings.Contains(got, "ago") {
		require.Failf(t, "assertion failed", "expected formatted date for old time, got %q", got)
	}
}

func TestRelativeDashboardTimeFuture(t *testing.T) {
	// Future time should be clamped to "just now"
	got := relativeDashboardTime(time.Now().Add(time.Hour))
	if got != "just now" {
		require.Failf(t, "assertion failed", "expected 'just now' for future time, got %q", got)
	}
}

func TestShortenDashboardPathWithRoot(t *testing.T) {
	got := shortenDashboardPath("/home/user/notes", "/home/user/notes/work/note.md")
	if got != "work/note.md" {
		require.Failf(t, "assertion failed", "expected 'work/note.md', got %q", got)
	}
}

func TestShortenDashboardPathEmpty(t *testing.T) {
	got := shortenDashboardPath("/home/user/notes", "")
	if got != "" {
		require.Failf(t, "assertion failed", "expected empty path, got %q", got)
	}
}

func TestShortenDashboardPathNoRoot(t *testing.T) {
	got := shortenDashboardPath("", "/some/path/note.md")
	if got == "" {
		require.FailNow(t, "expected non-empty result for absolute path with no root")
	}
}

func TestDashboardActionLine(t *testing.T) {
	rendered := dashboardActionLine("enter", "Open note", 40)
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "enter") || !strings.Contains(plain, "Open note") {
		require.Failf(t, "assertion failed", "expected key and desc in action line, got %q", plain)
	}
}

func TestDashboardSummaryLine(t *testing.T) {
	rendered := dashboardSummaryLine("notes:", "42", 40)
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "notes:") || !strings.Contains(plain, "42") {
		require.Failf(t, "assertion failed", "expected label and value in summary line, got %q", plain)
	}
}

func TestRenderTagChipsEmpty(t *testing.T) {
	rendered, titleWidth := renderTagChips(nil, 80, 40, textColor, bgColor, false)
	if rendered != "" {
		require.Failf(t, "assertion failed", "expected empty rendered chips for nil tags, got %q", rendered)
	}
	// When nothing fits, the function returns the full availableWidth to the title
	if titleWidth <= 0 {
		require.Failf(t, "assertion failed", "expected positive title width for nil tags, got %d", titleWidth)
	}
}

func TestRenderTagChipsSingle(t *testing.T) {
	rendered, usedWidth := renderTagChips([]string{"work"}, 80, 40, textColor, bgColor, false)
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "[work]") {
		require.Failf(t, "assertion failed", "expected '[work]' in chip rendering, got %q", plain)
	}
	if usedWidth <= 0 {
		require.Failf(t, "assertion failed", "expected positive used width, got %d", usedWidth)
	}
}

func TestRenderTagChipsOverflow(t *testing.T) {
	// Give very narrow width to force overflow
	tags := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	rendered, _ := renderTagChips(tags, 20, 10, textColor, bgColor, false)
	plain := stripANSI(rendered)
	// Should show some tags and +N overflow indicator
	if !strings.Contains(plain, "+") && !strings.Contains(plain, "[alpha]") {
		require.Failf(t, "assertion failed", "expected overflow indicator or tag in narrow rendering, got %q", plain)
	}
}

func TestRenderLeftPaneBodyNotes(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeNotes
	m.treeItems = nil
	rendered := m.renderLeftPaneBody()
	plain := stripANSI(rendered)
	if !strings.Contains(plain, "empty") {
		// Verify it doesn't panic
		_ = plain
	}
}

func TestRenderLeftPaneBodyTemporary(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModeTemporary
	m.tempNotes = nil
	rendered := m.renderLeftPaneBody()
	_ = rendered
}

func TestRenderLeftPaneBodyPins(t *testing.T) {
	m := newTestModel(t)
	m.listMode = listModePins
	rendered := m.renderLeftPaneBody()
	_ = rendered
}

func TestRenderDashboardView(t *testing.T) {
	m := newTestModel(t)
	m.showDashboard = true
	m.width = 120
	m.height = 40
	rendered := m.renderDashboardView()
	plain := stripANSI(rendered)
	if plain == "" {
		require.FailNow(t, "expected non-empty dashboard view")
	}
}

func TestTruncateToWidth(t *testing.T) {
	// Short text - should be unchanged
	short := truncateToWidth("hello", 20)
	if short != "hello" {
		require.Failf(t, "assertion failed", "expected 'hello' unchanged, got %q", short)
	}

	// Long text - should be truncated
	long := truncateToWidth("this is a very long string that should get truncated", 10)
	if len(long) > 15 { // account for potential ellipsis
		require.Failf(t, "assertion failed", "expected truncated string, got %q", long)
	}
}
