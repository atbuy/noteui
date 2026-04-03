package tui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type helpEntry struct {
	section string
	key     string
	desc    string
}

type helpSection struct {
	title   string
	entries []helpEntry
}

func (m Model) helpEntries() []helpEntry {
	bf := keys.BracketForward.Help().Key
	bb := keys.BracketBackward.Help().Key
	return []helpEntry{
		{section: "Tree", key: keys.MoveDown.Help().Key + " / " + keys.MoveUp.Help().Key, desc: "Move up and down"},
		{section: "Tree", key: keys.ScrollHalfPageUp.Help().Key + " / " + keys.ScrollHalfPageDown.Help().Key, desc: "Scroll half page up / down"},
		{section: "Tree", key: keys.CollapseCategory.Help().Key + "/" + keys.ExpandCategory.Help().Key, desc: "Collapse/Expand category"},
		{section: "Tree", key: keys.PendingG.Help().Key + keys.PendingG.Help().Key + " / " + keys.JumpBottom.Help().Key, desc: "Jump to top / bottom of list"},
		{section: "Tree", key: keys.Open.Help().Key, desc: "Open note or jump from Pins"},
		{section: "Tree", key: keys.Move.Help().Key, desc: "Move current item or marked batch"},
		{section: "Tree", key: keys.ToggleSelect.Help().Key, desc: "Mark/unmark item for bulk move"},
		{section: "Tree", key: keys.Rename.Help().Key, desc: "Rename note/category"},
		{section: "Tree", key: keys.AddTag.Help().Key, desc: "Add tag to selected note"},
		{section: "Tree", key: keys.Pin.Help().Key, desc: "Pin or unpin current item"},
		{section: "Tree", key: keys.ToggleSync.Help().Key, desc: "Toggle selected note sync"},
		{section: "Tree", key: keys.SelectSyncProfile.Help().Key, desc: "Select default sync profile"},
		{section: "Tree", key: keys.OpenConflictCopy.Help().Key, desc: "Open generated conflict copy"},
		{section: "Tree", key: keys.DeleteRemoteKeepLocal.Help().Key, desc: "Delete remote copy, keep local note"},
		{section: "Tree", key: keys.SyncImportCurrent.Help().Key, desc: "Import current remote note"},
		{section: "Tree", key: keys.SyncImport.Help().Key, desc: "Import all missing synced notes"},
		{section: "Tree", key: keys.CreateCategory.Help().Key, desc: "Create category"},
		{section: "Tree", key: keys.NewTodoList.Help().Key, desc: "New todo list (tree focus)"},
		{section: "Notes", key: keys.NewNote.Help().Key, desc: "New note in current view"},
		{section: "Notes", key: keys.NewTemporaryNote.Help().Key, desc: "New temporary note"},
		{section: "Notes", key: bf + "/" + bb, desc: "Switch Notes / Temporary"},
		{section: "Notes", key: keys.ShowPins.Help().Key, desc: "Toggle Pins view"},
		{section: "Preview", key: keys.NextMatch.Help().Key + " / " + keys.PrevMatch.Help().Key, desc: "Next / previous match in preview"},
		{section: "Preview", key: keys.PendingZ.Help().Key + keys.PendingZ.Help().Key, desc: "Center current match in preview"},
		{section: "Preview", key: keys.TogglePreviewPrivacy.Help().Key, desc: "Toggle preview privacy"},
		{section: "Preview", key: keys.TogglePreviewLineNumbers.Help().Key, desc: "Toggle preview line numbers"},
		{section: "Preview", key: bf + keys.HeadingJumpKey.Help().Key + " / " + bb + keys.HeadingJumpKey.Help().Key, desc: "Next / prev heading in preview"},
		{section: "Preview", key: bf + keys.TodoKey.Help().Key + " / " + bb + keys.TodoKey.Help().Key, desc: "Next / prev todo in preview"},
		{section: "Preview", key: keys.PendingG.Help().Key + keys.PendingG.Help().Key + " / " + keys.JumpBottom.Help().Key, desc: "First / last todo in todo nav"},
		{section: "Preview", key: keys.TodoKey.Help().Key + keys.TodoKey.Help().Key, desc: "Toggle current todo checkbox"},
		{section: "Preview", key: keys.TodoKey.Help().Key + keys.TodoAdd.Help().Key, desc: "Add new todo item"},
		{section: "Preview", key: keys.TodoKey.Help().Key + keys.TodoDelete.Help().Key, desc: "Delete current todo item"},
		{section: "Preview", key: keys.TodoKey.Help().Key + keys.TodoEdit.Help().Key, desc: "Edit current todo item"},
		{section: "Preview", key: keys.ToggleEncryption.Help().Key, desc: "Toggle note encryption"},
		{section: "Filter", key: keys.Search.Help().Key, desc: "Search"},
		{section: "Filter", key: "#tag", desc: "Filter by tag in search"},
		{section: "Filter", key: "esc", desc: "Leave search, then clear on second press"},
		{section: "Global", key: keys.Focus.Help().Key, desc: "Switch focused pane"},
		{section: "Global", key: keys.SortToggle.Help().Key, desc: "Toggle sort (alpha / modified)"},
		{section: "Global", key: keys.Refresh.Help().Key, desc: "Refresh"},
		{section: "Global", key: keys.Delete.Help().Key + keys.DeleteConfirm.Help().Key, desc: "Trash note/category"},
		{section: "Global", key: "esc", desc: "Close help"},
		{section: "Global", key: keys.Quit.Help().Key, desc: "Quit"},
	}
}

func (m Model) filteredHelpSections() []helpSection {
	entries := m.helpEntries()
	query := strings.ToLower(strings.TrimSpace(m.helpInput.Value()))
	sectionOrder := []string{"Filter", "Tree", "Notes", "Preview", "Global"}
	bySection := make(map[string][]helpEntry)
	for _, entry := range entries {
		if query != "" {
			blob := strings.ToLower(entry.section + " " + entry.key + " " + entry.desc)
			if !strings.Contains(blob, query) {
				continue
			}
		}
		bySection[entry.section] = append(bySection[entry.section], entry)
	}

	out := make([]helpSection, 0, len(sectionOrder))
	for _, title := range sectionOrder {
		entries := bySection[title]
		if len(entries) == 0 {
			continue
		}
		out = append(out, helpSection{title: title, entries: entries})
	}
	return out
}

// helpRowCount returns the number of rendered rows without doing any lipgloss
// rendering.  Used by clampHelpScroll so that scroll clamping is cheap even
// when mouse motion events arrive at high frequency.
func (m Model) helpRowCount() int {
	sections := m.filteredHelpSections()
	if len(sections) == 0 {
		return 1 // "no matching commands" placeholder
	}
	n := 0
	for i, s := range sections {
		if i > 0 {
			n++ // blank separator row between sections
		}
		n++                 // section title row
		n += len(s.entries) // one row per entry
	}
	return n
}

// rebuildHelpRowsCache renders all help rows and stores them in the model.
// It is a no-op when the filter query and modal width have not changed.
// Call this from Update whenever helpInput.Value() or m.width may change so
// that View() can do a cheap slice lookup instead of a full lipgloss render.
func (m *Model) rebuildHelpRowsCache() {
	_, innerWidth := m.modalDimensions(60, 96)
	query := m.helpInput.Value()
	if m.helpRowsCache != nil && m.helpRowsCacheQuery == query && m.helpRowsCacheWidth == innerWidth {
		return
	}
	m.helpRowsCache = m.renderedHelpRows(innerWidth, false)
	m.helpRowsCacheQuery = query
	m.helpRowsCacheWidth = innerWidth
	m.invalidateHelpBodyCache()
}

func (m *Model) invalidateHelpModalCache() {
	m.helpModalCache = ""
	m.helpModalCacheQuery = ""
	m.helpModalCacheWidth = 0
	m.helpModalCacheHeight = 0
	m.helpModalCacheRows = 0
	m.helpModalCacheScroll = 0
}

func (m *Model) invalidateHelpBodyCache() {
	m.helpBodyCache = ""
	m.helpBodyCacheQuery = ""
	m.helpBodyCacheWidth = 0
	m.helpBodyCacheRows = 0
	m.helpBodyCacheScroll = 0
	m.invalidateHelpModalCache()
}

func (m *Model) rebuildHelpBodyCache(maxRows int) {
	if maxRows <= 0 {
		m.invalidateHelpBodyCache()
		return
	}
	m.rebuildHelpRowsCache()
	_, innerWidth := m.modalDimensions(60, 96)
	query := m.helpInput.Value()
	rows := m.helpRowsCache
	if rows == nil {
		rows = m.renderedHelpRows(innerWidth, false)
	}
	if m.helpBodyCache != "" && m.helpBodyCacheQuery == query && m.helpBodyCacheWidth == innerWidth && m.helpBodyCacheRows == maxRows && m.helpBodyCacheScroll == m.helpScroll {
		return
	}
	end := min(len(rows), m.helpScroll+maxRows)
	visibleRows := append([]string{}, rows[m.helpScroll:end]...)
	for len(visibleRows) < maxRows {
		visibleRows = append(visibleRows, m.renderModalBlank(innerWidth))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, visibleRows...)
	m.helpBodyCache = lipgloss.NewStyle().Width(innerWidth).Height(maxRows).Background(modalBgColor).Render(body)
	m.helpBodyCacheQuery = query
	m.helpBodyCacheWidth = innerWidth
	m.helpBodyCacheRows = maxRows
	m.helpBodyCacheScroll = m.helpScroll
}

func (m *Model) rebuildHelpModalCache(maxRows int) {
	if maxRows <= 0 {
		m.invalidateHelpModalCache()
		return
	}
	m.rebuildHelpBodyCache(maxRows)
	modalWidth, innerWidth := m.modalDimensions(60, 96)
	query := m.helpInput.Value()
	if m.helpModalCache != "" && m.helpModalCacheQuery == query && m.helpModalCacheWidth == m.width && m.helpModalCacheHeight == m.height && m.helpModalCacheRows == maxRows && m.helpModalCacheScroll == m.helpScroll {
		return
	}

	rows := m.helpRowsCache
	if rows == nil || m.helpRowsCacheQuery != query || m.helpRowsCacheWidth != innerWidth {
		rows = m.renderedHelpRows(innerWidth, false)
	}
	scrollText := "all"
	if len(rows) > maxRows {
		scrollText = fmt.Sprintf("%d-%d of %d", m.helpScroll+1, min(len(rows), m.helpScroll+maxRows), len(rows))
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
				m.helpBodyCache,
				m.renderModalBlank(innerWidth),
				m.renderModalFooter("Type to filter • up/down scroll • home/end top/bottom • ctrl+d/u page • mouse wheel • esc to close", innerWidth),
			),
		)

	m.helpModalCache = modalCardStyle(modalWidth).Render(content)
	m.helpModalCacheQuery = query
	m.helpModalCacheWidth = m.width
	m.helpModalCacheHeight = m.height
	m.helpModalCacheRows = maxRows
	m.helpModalCacheScroll = m.helpScroll
}

func (m *Model) maxHelpScroll(maxRows int) int {
	if maxRows <= 0 {
		return 0
	}
	_, innerWidth := m.modalDimensions(60, 96)
	if m.helpRowsCache != nil && m.helpRowsCacheQuery == m.helpInput.Value() && m.helpRowsCacheWidth == innerWidth {
		return max(0, len(m.helpRowsCache)-maxRows)
	}
	return max(0, m.helpRowCount()-maxRows)
}

func (m *Model) clampHelpScroll(maxRows int) {
	if maxRows <= 0 {
		m.helpScroll = 0
		return
	}
	maxScroll := m.maxHelpScroll(maxRows)
	if m.helpScroll < 0 {
		m.helpScroll = 0
	} else if m.helpScroll > maxScroll {
		m.helpScroll = maxScroll
	}
}

func (m *Model) moveHelpScroll(delta, maxRows int) bool {
	if delta == 0 || maxRows <= 0 {
		return false
	}
	maxScroll := m.maxHelpScroll(maxRows)
	if (delta < 0 && m.helpScroll == 0) || (delta > 0 && m.helpScroll == maxScroll) {
		return false
	}
	prev := m.helpScroll
	m.helpScroll += delta
	if m.helpScroll < 0 {
		m.helpScroll = 0
	} else if m.helpScroll > maxScroll {
		m.helpScroll = maxScroll
	}
	return m.helpScroll != prev
}

// isMouseEscapeFragment reports whether msg is a KeyRunes event that
// contains more than one rune and is not a bracketed paste.  This pattern
// arises when tmux (or another terminal multiplexer) splits a mouse escape
// sequence such as "\x1b[64;127;31M" across two reads: bubbletea sees the
// ESC alone as KeyEsc, and then the remainder "[64;127;31M" as a single
// multi-rune KeyRunes message.  Real keystrokes always produce exactly one
// rune (IME multi-character input arrives via msg.Paste instead), so any
// multi-rune non-paste KeyRunes event is safe to drop.
func isMouseEscapeFragment(msg tea.KeyMsg) bool {
	return !msg.Paste && msg.Type == tea.KeyRunes && len(msg.Runes) > 1
}

func shouldUpdateHelpInput(msg tea.KeyMsg, input textinput.Model) bool {
	if msg.Paste {
		return true
	}
	if isHelpInputEditKey(msg, input) {
		return true
	}
	if isMouseEscapeFragment(msg) {
		return false
	}
	if msg.Alt || msg.Type != tea.KeyRunes || len(msg.Runes) == 0 {
		return false
	}
	if len(msg.Runes) == 1 && msg.Runes[0] == '[' {
		return false
	}
	for _, r := range msg.Runes {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func isHelpInputEditKey(msg tea.KeyMsg, input textinput.Model) bool {
	km := input.KeyMap
	return key.Matches(msg, km.CharacterForward) ||
		key.Matches(msg, km.CharacterBackward) ||
		key.Matches(msg, km.WordForward) ||
		key.Matches(msg, km.WordBackward) ||
		key.Matches(msg, km.DeleteWordBackward) ||
		key.Matches(msg, km.DeleteWordForward) ||
		key.Matches(msg, km.DeleteAfterCursor) ||
		key.Matches(msg, km.DeleteBeforeCursor) ||
		key.Matches(msg, km.DeleteCharacterBackward) ||
		key.Matches(msg, km.DeleteCharacterForward) ||
		key.Matches(msg, km.LineStart) ||
		key.Matches(msg, km.LineEnd) ||
		key.Matches(msg, km.Paste)
}

func (m Model) renderedHelpRows(innerWidth int, includeWindow bool) []string {
	sections := m.filteredHelpSections()
	if len(sections) == 0 {
		return []string{lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(modalMutedStyle.Render("No matching commands"))}
	}

	rows := make([]string, 0, len(sections)*4)
	for si, section := range sections {
		if si > 0 {
			rows = append(rows, m.renderModalBlank(innerWidth))
		}
		rows = append(rows, m.renderHelpSectionTitle(section.title, innerWidth))
		for _, entry := range section.entries {
			rows = append(rows, m.renderHelpLine(entry.key, entry.desc, innerWidth))
		}
	}
	return rows
}

func (m Model) renderHelpSectionTitle(title string, innerWidth int) string {
	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(modalKeyStyle.Width(innerWidth).Render(title))
}
