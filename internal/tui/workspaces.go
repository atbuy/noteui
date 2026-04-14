package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/state"
	notesync "atbuy/noteui/internal/sync"
)

type WorkspaceSession struct {
	Name            string
	Label           string
	Override        bool
	StartWithPicker bool
}

type workspaceOption struct {
	Name           string
	Label          string
	Root           string
	SyncRemoteRoot string
}

func sortedWorkspaceOptions(cfg config.Config) []workspaceOption {
	names := config.SortedWorkspaceNames(cfg)
	if len(names) == 0 {
		return nil
	}
	out := make([]workspaceOption, 0, len(names))
	for _, name := range names {
		workspace := cfg.Workspaces[name]
		out = append(out, workspaceOption{
			Name:           name,
			Label:          strings.TrimSpace(workspace.Label),
			Root:           filepath.Clean(strings.TrimSpace(workspace.Root)),
			SyncRemoteRoot: strings.TrimSpace(workspace.SyncRemoteRoot),
		})
	}
	return out
}

func workspaceDisplayName(name, label string) string {
	label = strings.TrimSpace(label)
	if label != "" {
		return label
	}
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return "Default"
}

func (m Model) activeWorkspaceDisplay() string {
	if m.workspaceOverride {
		return "NOTES_ROOT override"
	}
	if strings.TrimSpace(m.workspaceName) != "" || strings.TrimSpace(m.workspaceLabel) != "" {
		return workspaceDisplayName(m.workspaceName, m.workspaceLabel)
	}
	if len(m.workspaceOptions) > 0 {
		return "Default"
	}
	return ""
}

func (m Model) workspaceStateKey() string {
	if m.workspaceOverride {
		root := filepath.Clean(strings.TrimSpace(m.rootDir))
		if root == "." || root == "" {
			return "override"
		}
		return "override:" + root
	}
	if strings.TrimSpace(m.workspaceName) != "" {
		return strings.TrimSpace(m.workspaceName)
	}
	return "default"
}

func (m Model) canSwitchWorkspace() bool {
	return !m.workspaceOverride && len(m.workspaceOptions) > 1
}

func (m *Model) loadCurrentWorkspaceState() {
	m.workspaceState = state.WorkspaceState{}
	m.pinnedNotes = map[string]bool{}
	m.pinnedCats = map[string]bool{}
	m.expanded = map[string]bool{"": true}
	m.sortMethod = sortAlpha
	m.sortReverse = false
	m.pendingSort = false

	if strings.TrimSpace(m.rootDir) == "" {
		m.syncRecords = map[string]notesync.NoteRecord{}
		m.syncInFlight = map[string]bool{}
		return
	}

	ws := m.state.Workspace(m.workspaceStateKey())
	ws.RecentCommands = normalizePaletteRecentCommands(ws.RecentCommands)
	m.workspaceState = ws
	m.sortMethod = ws.SortMethod
	if m.sortMethod == "" {
		m.sortMethod = sortAlpha
	}
	m.sortReverse = ws.SortReverse

	for _, p := range ws.PinnedNotes {
		m.pinnedNotes[filepath.ToSlash(strings.TrimSpace(p))] = true
	}
	for _, p := range ws.PinnedCategories {
		m.pinnedCats[normalizeCategoryRelPath(p)] = true
	}
	for _, p := range ws.CollapsedCategories {
		if normalized := normalizeCategoryRelPath(p); normalized != "" {
			m.expanded[normalized] = false
		}
	}

	if notesync.HasSyncProfile(m.cfg.Sync) {
		if syncedNotes, syncedCats, err := notesync.LoadPinnedRelPaths(m.rootDir); err == nil {
			for _, p := range syncedNotes {
				m.pinnedNotes[p] = true
			}
			if syncedCats != nil {
				m.pinnedCats = make(map[string]bool, len(syncedCats))
				for _, p := range syncedCats {
					m.pinnedCats[p] = true
				}
			}
		}
	}
}

func (m *Model) resetWorkspaceTransientState() {
	m.notes = nil
	m.categories = nil
	m.tempNotes = nil
	m.remoteOnlyNotes = nil
	m.remoteCategories = nil
	m.todoItems = nil
	m.treeItems = nil
	m.treeCursor = 0
	m.tempCursor = 0
	m.pinsCursor = 0
	m.todoCursor = 0
	m.markedTreeItems = make(map[string]bool)
	m.selected = nil
	m.previewPath = ""
	m.previewContent = ""
	m.previewBaseContent = ""
	m.previewMatches = nil
	m.previewMatchIndex = 0
	m.previewHeadings = nil
	m.previewTodos = nil
	m.previewTodoCursor = -1
	m.pendingTodoCursor = -1
	m.pendingPreviewYOffset = -1
	m.previewTodoNavMode = false
	m.previewLinks = nil
	m.previewLinkCursor = -1
	m.previewLinkNavMode = false
	m.previewLineNumberStart = 0
	m.preview.GotoTop()
	m.searchMode = false
	m.searchInput.Blur()
	m.searchInput.SetValue("")
	m.deletePending = nil
	m.preserveCursor = -1
	m.pendingG = false
	m.pendingZ = false
	m.pendingT = false
	m.pendingBracketDir = ""
	m.previewHover = false
	m.showCommandPalette = false
	m.commandPaletteInput.Blur()
	m.commandPaletteInput.SetValue("")
	m.commandPaletteItems = nil
	m.commandPaletteFiltered = nil
	m.commandPaletteCursor = 0
	m.showHelp = false
	m.helpScroll = 0
	m.helpInput.Blur()
	m.helpInput.SetValue("")
	m.showCreateCategory = false
	m.categoryInput.Blur()
	m.categoryInput.SetValue("")
	m.closeMoveBrowser("")
	m.showMove = false
	m.moveInput.Blur()
	m.moveInput.SetValue("")
	m.movePending = nil
	m.showRename = false
	m.renameInput.Blur()
	m.renameInput.SetValue("")
	m.renamePending = nil
	m.showAddTag = false
	m.tagInput.Blur()
	m.tagInput.SetValue("")
	m.showSyncProfilePicker = false
	m.showSyncProfileMigration = false
	m.pendingSyncProfileChange = nil
	m.showSyncDebugModal = false
	m.lastDeletion = nil
	m.showTodoAdd = false
	m.showTodoEdit = false
	m.showTodoDueDate = false
	m.showTodoPriority = false
	m.todoInput.Blur()
	m.todoInput.SetValue("")
	m.dueDateInput.Blur()
	m.dueDateInput.SetValue("")
	m.priorityInput.Blur()
	m.priorityInput.SetValue("")
	m.showPassphraseModal = false
	m.passphraseInput.Blur()
	m.passphraseInput.SetValue("")
	m.showEncryptConfirm = false
	m.encryptConfirmYes = false
	m.pendingEncryptPath = ""
	m.pendingEncryptedEdit = nil
	m.sessionPassphrase = ""
	m.listMode = listModeNotes
	m.lastNonPinsMode = listModeNotes
	m.focus = focusTree
	m.startupSyncChecked = !notesync.HasSyncProfile(m.cfg.Sync)
	m.syncRunning = false
	m.syncSpinnerFrame = 0
	m.syncInFlight = map[string]bool{}
	m.syncRecords = map[string]notesync.NoteRecord{}
	m.pendingSyncedPinRelPaths = nil
	m.pendingSyncedCategories = nil
	m.applyPendingSyncedPins = false
	m.showDashboard = false
	m.showNoteHistory = false
	m.noteHistoryEntries = nil
	m.noteHistoryCursor = 0
	m.noteHistoryRelPath = ""
	m.noteHistoryAbsPath = ""
	m.showTrashBrowser = false
	m.trashBrowserItems = nil
	m.trashBrowserCursor = 0
	m.status = "switching workspace..."
}

func (m *Model) stopWorkspaceWatch() {
	if m.watcher != nil {
		_ = m.watcher.Close()
	}
	m.watcher = nil
	m.watchEvents = nil
}

func (m Model) startWorkspaceSessionCmd() tea.Cmd {
	if strings.TrimSpace(m.rootDir) == "" {
		return nil
	}
	return batchCmds(
		refreshAllCmd(m.rootDir, m.sessionToken),
		startWatchTeaCmd(m.rootDir, m.sessionToken),
		startSyncCmd(m.sessionToken),
	)
}

func (m *Model) openWorkspacePicker() {
	if m.workspaceOverride {
		m.status = "workspace switching is disabled when NOTES_ROOT is set"
		return
	}
	if len(m.workspaceOptions) == 0 {
		m.status = "workspaces are not configured"
		return
	}
	m.showWorkspacePicker = true
	m.workspaceCursor = m.currentWorkspaceIndex()
	m.status = "select workspace"
}

func (m *Model) closeWorkspacePicker(status string) {
	m.showWorkspacePicker = false
	if strings.TrimSpace(status) != "" {
		m.status = status
	}
}

func (m Model) currentWorkspaceIndex() int {
	for i, opt := range m.workspaceOptions {
		if opt.Name == m.workspaceName {
			return i
		}
	}
	return 0
}

func (m *Model) moveWorkspaceCursor(delta int) {
	if len(m.workspaceOptions) == 0 {
		m.workspaceCursor = 0
		return
	}
	m.workspaceCursor += delta
	if m.workspaceCursor < 0 {
		m.workspaceCursor = 0
	}
	if m.workspaceCursor >= len(m.workspaceOptions) {
		m.workspaceCursor = len(m.workspaceOptions) - 1
	}
}

func (m Model) selectedWorkspaceOption() *workspaceOption {
	if len(m.workspaceOptions) == 0 {
		return nil
	}
	cursor := m.workspaceCursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(m.workspaceOptions) {
		cursor = len(m.workspaceOptions) - 1
	}
	opt := m.workspaceOptions[cursor]
	return &opt
}

func (m *Model) confirmSelectedWorkspace() tea.Cmd {
	opt := m.selectedWorkspaceOption()
	if opt == nil {
		m.closeWorkspacePicker("workspaces are not configured")
		return nil
	}
	return m.switchWorkspaceToOption(*opt)
}

func (m *Model) switchWorkspaceToOption(opt workspaceOption) tea.Cmd {
	if strings.TrimSpace(opt.Root) == "" {
		m.closeWorkspacePicker("workspace root is empty")
		return nil
	}
	if opt.Name == m.workspaceName && filepath.Clean(opt.Root) == filepath.Clean(m.rootDir) {
		m.closeWorkspacePicker("workspace already selected")
		return nil
	}
	if strings.TrimSpace(m.rootDir) != "" {
		if err := m.saveLocalState(); err != nil {
			m.closeWorkspacePicker("save failed: " + err.Error())
			return nil
		}
	}
	m.stopWorkspaceWatch()
	m.sessionToken++
	m.rootDir = filepath.Clean(opt.Root)
	m.workspaceName = opt.Name
	m.workspaceLabel = opt.Label
	m.workspaceOverride = false
	m.loadCurrentWorkspaceState()
	m.resetWorkspaceTransientState()
	m.closeWorkspacePicker(fmt.Sprintf("workspace: %s", workspaceDisplayName(opt.Name, opt.Label)))
	return m.startWorkspaceSessionCmd()
}

func (m Model) renderWorkspacePickerModal() string {
	modalWidth, innerWidth := m.modalDimensions(58, 88)
	lines := make([]string, 0, len(m.workspaceOptions)*2)
	for i, opt := range m.workspaceOptions {
		prefix := "  "
		if i == m.workspaceCursor {
			prefix = "› "
		}
		name := workspaceDisplayName(opt.Name, opt.Label)
		if opt.Name == m.workspaceName && !m.workspaceOverride {
			name += " (current)"
		}
		fg := textColor
		bg := modalBgColor
		if i == m.workspaceCursor {
			fg = selectedFgColor
			bg = selectedBgColor
		}
		row := []string{
			lipgloss.NewStyle().Width(innerWidth).Padding(0, 1).Foreground(fg).Background(bg).Render(prefix + name),
			lipgloss.NewStyle().Width(innerWidth).Padding(0, 3).Foreground(mutedColor).Background(bg).Render(trimOrPad(opt.Root, innerWidth-3)),
		}
		if opt.SyncRemoteRoot != "" {
			row = append(row, lipgloss.NewStyle().Width(innerWidth).Padding(0, 3).Foreground(mutedColor).Background(bg).Render("sync → "+trimOrPad(opt.SyncRemoteRoot, innerWidth-9)))
		}
		lines = append(lines, row...)
	}
	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Foreground(mutedColor).Render("No workspaces configured"))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, lines...)
	hint := "enter select • esc cancel"
	if strings.TrimSpace(m.rootDir) == "" {
		hint = "enter select • q quit"
	}
	footer := lipgloss.NewStyle().Width(innerWidth).Foreground(modalMutedColor).Background(modalBgColor).Render(hint)
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Width(innerWidth).Foreground(modalAccentColor).Background(modalBgColor).Bold(boldModalTitles).Render("Switch Workspace"),
		lipgloss.NewStyle().Width(innerWidth).Foreground(modalMutedColor).Background(modalBgColor).Render("Select the active notes root for this session."),
		"",
		body,
		"",
		footer,
	)
	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderWorkspaceSegment() string {
	label := strings.TrimSpace(m.activeWorkspaceDisplay())
	if label == "" {
		return ""
	}
	if m.workspaceOverride {
		return "WS override"
	}
	return "WS " + label
}
