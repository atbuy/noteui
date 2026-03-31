package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/editor"
	"atbuy/noteui/internal/notes"
	"atbuy/noteui/internal/state"
)

type (
	paneFocus        int
	treeItemKind     int
	deleteTargetKind int
	moveTargetKind   int
	renameTargetKind int
	listMode         int
	pinItemKind      int
)

const (
	listModeNotes listMode = iota
	listModeTemporary
	listModePins
)

const (
	pinItemCategory pinItemKind = iota
	pinItemNote
	pinItemTemporaryNote
)

const (
	focusTree paneFocus = iota
	focusPreview
)

const (
	treePaneRatio   = 0.32
	minTreeWidth    = 30
	minPreviewWidth = 40
	panelGapWidth   = 2
)

const (
	treeCategory treeItemKind = iota
	treeNote
)

const (
	deleteNone deleteTargetKind = iota
	deleteTargetCategory
	deleteTargetNote
)

const (
	moveTargetNone moveTargetKind = iota
	moveTargetCategory
	moveTargetNote
)

const (
	renameTargetNone renameTargetKind = iota
	renameTargetNote
	renameTargetCategory
)

type dashboardRecentNote struct {
	Note    notes.Note
	IsTemp  bool
	Display string
}

type movePending struct {
	kind       moveTargetKind
	oldRelPath string
	name       string
}

type noteMovedMsg struct {
	oldRelPath string
	newRelPath string
	err        error
}

type noteRenamedMsg struct {
	oldPath string
	newPath string
	err     error
}

type categoryMovedMsg struct {
	oldRelPath string
	newRelPath string
	err        error
}

type renamePending struct {
	kind     renameTargetKind
	path     string
	relPath  string
	oldTitle string
	oldName  string
}

type categoryRenamedMsg struct {
	oldRelPath string
	newRelPath string
	err        error
}

type deletePending struct {
	kind    deleteTargetKind
	relPath string
	name    string
}

type noteDeletedMsg struct {
	path string
	err  error
}

type categoryDeletedMsg struct {
	relPath string
	err     error
}

type treeItem struct {
	Kind      treeItemKind
	Name      string
	RelPath   string
	Depth     int
	Expanded  bool
	Note      *notes.Note
	Category  *notes.Category
	MatchHint string
}

type pinItem struct {
	Kind    pinItemKind
	Name    string
	RelPath string
	Path    string
}

func (t treeItem) key() string {
	switch t.Kind {
	case treeCategory:
		return "c:" + t.RelPath
	case treeNote:
		return "n:" + t.RelPath
	default:
		return t.RelPath
	}
}

type Model struct {
	rootDir string
	version string

	notes           []notes.Note
	categories      []notes.Category
	expanded        map[string]bool
	tempNotes       []notes.Note
	tempCursor      int
	pinsCursor      int
	listMode        listMode
	lastNonPinsMode listMode

	treeItems    []treeItem
	treeCursor   int
	selected     *notes.Note
	width        int
	height       int
	previewWidth int
	status       string

	cfg            config.Config
	preview        viewport.Model
	previewPath    string
	previewContent string

	previewPrivacyEnabled      bool
	previewPrivacyForcedByNote bool

	watcher     interface{ Close() error }
	watchEvents <-chan teaMsg

	focus             paneFocus
	pendingG          bool
	pendingBracketDir string
	previewHover      bool
	previewPaneX      int
	previewPaneY      int
	previewPaneW      int
	previewPaneH      int
	previewHeadings   []int

	state       state.State
	pinnedNotes map[string]bool
	pinnedCats  map[string]bool

	showHelp      bool
	showDashboard bool

	showCreateCategory bool
	categoryInput      textinput.Model

	showMove    bool
	moveInput   textinput.Model
	movePending *movePending

	showRename    bool
	renameInput   textinput.Model
	renamePending *renamePending

	searchInput textinput.Model
	searchMode  bool

	deletePending  *deletePending
	preserveCursor int

	startupError string
}

type dataLoadedMsg struct {
	notes      []notes.Note
	tempNotes  []notes.Note
	categories []notes.Category
	err        error
}

type noteCreatedMsg struct {
	path string
	err  error
}

type categoryCreatedMsg struct {
	relPath string
	err     error
}

func New(root, startupError string, cfg config.Config, version string) Model {
	categoryInput := textinput.New()
	categoryInput.Placeholder = "work/project-a"
	categoryInput.Prompt = "Category: "
	categoryInput.CharLimit = 200
	categoryInput.Width = 40

	searchInput := textinput.New()
	searchInput.Placeholder = "Search notes..."
	searchInput.Prompt = "/ "
	searchInput.CharLimit = 200
	searchInput.Width = 32

	moveInput := textinput.New()
	moveInput.Placeholder = "work/project-a/note.md"
	moveInput.Prompt = "Move to: "
	moveInput.CharLimit = 300
	moveInput.Width = 48

	renameInput := textinput.New()
	renameInput.Placeholder = "New title"
	renameInput.Prompt = "Title: "
	renameInput.CharLimit = 300
	renameInput.Width = 48

	vp := viewport.New(0, 0)

	st, _ := state.Load()

	pinnedNotes := make(map[string]bool, len(st.PinnedNotes))
	for _, p := range st.PinnedNotes {
		pinnedNotes[p] = true
	}

	pinnedCats := make(map[string]bool, len(st.PinnedCategories))
	for _, p := range st.PinnedCategories {
		pinnedCats[p] = true
	}

	expanded := map[string]bool{
		"": true,
	}
	for _, p := range st.CollapsedCategories {
		expanded[p] = false
	}

	return Model{
		rootDir:               root,
		version:               version,
		status:                "loading notes...",
		expanded:              expanded,
		categoryInput:         categoryInput,
		searchInput:           searchInput,
		moveInput:             moveInput,
		renameInput:           renameInput,
		preserveCursor:        -1,
		startupError:          startupError,
		cfg:                   cfg,
		preview:               vp,
		focus:                 focusTree,
		state:                 st,
		pinnedNotes:           pinnedNotes,
		pinnedCats:            pinnedCats,
		listMode:              listModeNotes,
		lastNonPinsMode:       listModeNotes,
		tempCursor:            0,
		pinsCursor:            0,
		previewPrivacyEnabled: cfg.Preview.Privacy,
		showDashboard:         cfg.Dashboard,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		refreshAllCmd(m.rootDir),
		startWatchTeaCmd(m.rootDir),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		m.previewHover = m.mouseInPreview(msg.X, msg.Y)

		if m.previewHover || m.focus == focusPreview {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.preview.ScrollUp(3)
				return m, nil
			case tea.MouseButtonWheelDown:
				m.preview.ScrollDown(3)
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		leftWidth, rightWidth := m.panelWidths()

		m.previewWidth = rightWidth
		m.searchInput.Width = max(16, leftWidth-8)
		m.categoryInput.Width = max(24, min(50, m.width-16))
		m.moveInput.Width = max(24, min(60, m.width-16))
		m.renameInput.Width = max(24, min(60, m.width-16))

		previewInnerWidth := max(20, rightWidth-8)
		previewInnerHeight := max(6, msg.Height-14)
		m.preview.Width = previewInnerWidth
		m.preview.Height = previewInnerHeight
		m.refreshPreview()
		return m, nil

	case noteMovedMsg:
		if msg.err != nil {
			m.status = "move failed: " + msg.err.Error()
			return m, nil
		}
		m.showMove = false
		m.movePending = nil
		m.moveInput.Blur()
		m.moveInput.SetValue("")
		m.preserveCursor = m.treeCursor
		m.status = "moved note: " + msg.newRelPath
		return m, refreshAllCmd(m.rootDir)

	case noteRenamedMsg:
		if msg.err != nil {
			m.status = "rename failed: " + msg.err.Error()
			return m, nil
		}
		m.showRename = false
		m.renamePending = nil
		m.renameInput.Blur()
		m.renameInput.SetValue("")
		m.preserveCursor = m.treeCursor
		m.status = "renamed note: " + filepath.Base(msg.newPath)
		return m, refreshAllCmd(m.rootDir)

	case categoryRenamedMsg:
		if msg.err != nil {
			m.status = "rename failed: " + msg.err.Error()
			return m, nil
		}
		m.showRename = false
		m.renamePending = nil
		m.renameInput.Blur()
		m.renameInput.SetValue("")
		m.preserveCursor = m.treeCursor
		m.rewriteCategoryStateSubtree(msg.oldRelPath, msg.newRelPath)
		_ = m.saveTreeState()
		m.status = "renamed category: " + msg.newRelPath
		return m, refreshAllCmd(m.rootDir)

	case categoryMovedMsg:
		if msg.err != nil {
			m.status = "move failed: " + msg.err.Error()
			return m, nil
		}
		m.showMove = false
		m.movePending = nil
		m.moveInput.Blur()
		m.moveInput.SetValue("")
		m.preserveCursor = m.treeCursor
		m.rewriteCategoryStateSubtree(msg.oldRelPath, msg.newRelPath)
		_ = m.saveTreeState()
		m.status = "moved category: " + msg.newRelPath
		return m, refreshAllCmd(m.rootDir)

	case noteDeletedMsg:
		if msg.err != nil {
			m.status = "delete failed: " + msg.err.Error()
			return m, nil
		}
		m.deletePending = nil
		m.preserveCursor = m.treeCursor
		m.status = "deleted note: " + msg.path
		return m, refreshAllCmd(m.rootDir)

	case categoryDeletedMsg:
		if msg.err != nil {
			m.status = "delete failed: " + msg.err.Error()
			return m, nil
		}
		m.deletePending = nil
		m.preserveCursor = m.treeCursor
		m.removeCategoryStateSubtree(msg.relPath)
		_ = m.saveTreeState()
		m.status = "deleted category: " + msg.relPath
		return m, refreshAllCmd(m.rootDir)

	case dataLoadedMsg:
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
			return m, nil
		}

		m.notes = msg.notes
		m.tempNotes = msg.tempNotes
		m.categories = msg.categories

		m.pruneCategoryStateToExisting()
		_ = m.saveTreeState()

		for _, c := range m.categories {
			if c.RelPath == "" {
				continue
			}
			if _, ok := m.expanded[c.RelPath]; !ok {
				m.expanded[c.RelPath] = true
			}
		}

		m.rebuildTree()

		if len(m.tempNotes) == 0 {
			m.tempCursor = 0
		} else if m.tempCursor >= len(m.filteredTempNotes()) {
			m.tempCursor = max(0, len(m.filteredTempNotes())-1)
		}

		if len(m.filteredPinnedItems()) == 0 {
			m.pinsCursor = 0
		} else if m.pinsCursor >= len(m.filteredPinnedItems()) {
			m.pinsCursor = len(m.filteredPinnedItems()) - 1
		}

		m.previewPath = ""

		m.syncSelectedNote()

		if len(msg.notes) > 0 {
			m.status = fmt.Sprintf("loaded %d notes", len(msg.notes))
		} else {
			m.status = "no notes found"
		}
		return m, nil

	case noteCreatedMsg:
		if msg.err != nil {
			m.status = "create failed: " + msg.err.Error()
			return m, nil
		}
		m.status = "created: " + msg.path
		return m, tea.Batch(refreshAllCmd(m.rootDir), editor.Open(msg.path))

	case categoryCreatedMsg:
		if msg.err != nil {
			m.status = "category create failed: " + msg.err.Error()
			return m, nil
		}
		m.showCreateCategory = false
		m.categoryInput.Blur()
		m.categoryInput.SetValue("")
		m.status = "created category: " + msg.relPath
		return m, refreshAllCmd(m.rootDir)

	case editor.FinishedMsg:
		if msg.Err != nil {
			m.status = "editor error: " + msg.Err.Error()
			return m, nil
		}

		newPath, renamed, err := notes.RenameFromTitle(msg.Path)
		if err != nil {
			m.status = "rename failed: " + err.Error()
			return m, refreshAllCmd(m.rootDir)
		}

		// Empty file with temp name was deleted.
		if newPath == "" && !renamed {
			m.status = "note discarded"
			return m, refreshAllCmd(m.rootDir)
		}

		if renamed {
			m.status = "renamed: " + filepath.Base(newPath)
		} else {
			m.status = "editor closed"
		}

		return m, refreshAllCmd(m.rootDir)

	case watchStartedMsg:
		if msg.err != nil {
			m.status = "watch disabled: " + msg.err.Error()
			return m, nil
		}
		m.watcher = msg.watcher
		m.watchEvents = msg.events
		m.status = "ready"
		return m, waitForWatchTeaCmd(m.watchEvents)

	case watchRefreshMsg:
		m.status = "auto refresh"
		m.previewPath = ""
		if m.watchEvents != nil {
			return m, tea.Batch(
				refreshAllCmd(m.rootDir),
				waitForWatchTeaCmd(m.watchEvents),
			)
		}
		return m, refreshAllCmd(m.rootDir)

	case watchErrorMsg:
		if msg.err != nil {
			m.status = "watch error: " + msg.err.Error()
		}
		if m.watchEvents != nil {
			return m, waitForWatchTeaCmd(m.watchEvents)
		}
		return m, nil

	case tea.KeyMsg:
		if m.showDashboard {
			switch msg.String() {
			case "enter":
				m.showDashboard = false
				m.status = "workspace"
				return m, nil

			case "]":
				m.showDashboard = false
				m.switchToTemporaryMode()
				m.status = "temporary"
				return m, nil

			case "P":
				m.showDashboard = false
				m.listMode = listModePins
				m.status = "pins"
				m.syncSelectedNote()
				return m, nil

			case "N":
				m.showDashboard = false
				return m, createTemporaryNoteCmd(m.rootDir)

			case "1":
				cmd := m.openDashboardRecent(0)
				if cmd != nil {
					m.showDashboard = false
					m.status = "opening recent note"
				}
				return m, cmd
			case "2":
				return m, m.openDashboardRecent(1)
			case "3":
				return m, m.openDashboardRecent(2)
			case "4":
				return m, m.openDashboardRecent(3)
			case "5":
				return m, m.openDashboardRecent(4)

			case "q", "ctrl+c":
				if m.watcher != nil {
					_ = m.watcher.Close()
				}
				return m, tea.Quit
			}

			return m, nil
		}
		if m.showMove {
			switch msg.String() {
			case "esc":
				m.showMove = false
				m.movePending = nil
				m.moveInput.Blur()
				m.moveInput.SetValue("")
				m.status = "move cancelled"
				return m, nil
			case "enter":
				value := strings.TrimSpace(m.moveInput.Value())
				if value == "" {
					m.showMove = false
					m.movePending = nil
					m.moveInput.Blur()
					m.status = "move cancelled"
					return m, nil
				}
				return m, m.confirmMove(value)
			}

			var cmd tea.Cmd
			m.moveInput, cmd = m.moveInput.Update(msg)
			return m, cmd
		}

		if m.showRename {
			switch msg.String() {
			case "esc":
				m.showRename = false
				m.renamePending = nil
				m.renameInput.Blur()
				m.renameInput.SetValue("")
				m.status = "rename cancelled"
				return m, nil
			case "enter":
				value := strings.TrimSpace(m.renameInput.Value())
				if value == "" {
					m.showRename = false
					m.renamePending = nil
					m.renameInput.Blur()
					m.status = "rename cancelled"
					return m, nil
				}

				switch m.renamePending.kind {
				case renameTargetNote:
					return m, renameNoteCmd(m.renamePending.path, value)
				case renameTargetCategory:
					return m, renameCategoryCmd(m.rootDir, m.renamePending.relPath, value)
				default:
					return m, nil
				}
			}

			var cmd tea.Cmd
			m.renameInput, cmd = m.renameInput.Update(msg)
			return m, cmd
		}

		if m.deletePending != nil {
			switch msg.String() {
			case "esc":
				m.deletePending = nil
				m.status = "delete cancelled"
				return m, nil
			case "d":
				return m, m.confirmDeleteCurrent()
			default:
				m.deletePending = nil
			}
		}

		if m.showHelp {
			switch msg.String() {
			case "esc", "q", "?":
				m.showHelp = false
				m.status = "help closed"
				return m, nil
			default:
				return m, nil
			}
		}

		if m.showCreateCategory {
			switch msg.String() {
			case "esc":
				m.showCreateCategory = false
				m.categoryInput.Blur()
				m.categoryInput.SetValue("")
				m.status = "category creation cancelled"
				return m, nil
			case "enter":
				value := strings.TrimSpace(m.categoryInput.Value())
				if value == "" {
					m.showCreateCategory = false
					m.categoryInput.Blur()
					m.status = "category creation cancelled"
					return m, nil
				}
				return m, createCategoryCmd(m.rootDir, value)
			}

			var cmd tea.Cmd
			m.categoryInput, cmd = m.categoryInput.Update(msg)
			return m, cmd
		}

		if key.Matches(msg, keys.ShowPins) {
			m.togglePinsMode()
			return m, nil
		}

		if msg.String() == "esc" && m.listMode == listModePins {
			if m.focus == focusPreview {
				m.focus = focusTree
				m.status = "tree focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil
			}

			if m.searchMode {
				m.searchMode = false
				m.searchInput.Blur()
				m.status = "search applied"
				return m, nil
			}

			if m.lastNonPinsMode == listModeTemporary {
				m.switchToTemporaryMode()
			} else {
				m.switchToNotesMode()
			}
			m.status = "left pins"
			return m, nil
		}

		if msg.String() == "esc" && m.listMode == listModeTemporary {
			if m.focus == focusPreview {
				m.focus = focusTree
				m.status = "tree focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil
			}

			if m.searchMode {
				m.searchMode = false
				m.searchInput.Blur()
				m.status = "search applied"
				return m, nil
			}

			m.switchToNotesMode()
			m.status = "left temporary"
			return m, nil
		}

		if key.Matches(msg, keys.Quit) {
			if m.watcher != nil {
				_ = m.watcher.Close()
			}
			return m, tea.Quit
		}

		if key.Matches(msg, keys.ShowHelp) {
			m.showHelp = true
			m.status = "help"
			return m, nil
		}

		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.searchInput.Blur()
				m.status = "search applied"
				return m, nil

			case "enter":
				m.searchMode = false
				m.searchInput.Blur()
				m.status = "search applied"
				return m, nil

			case "up":
				switch m.listMode {
				case listModeTemporary:
					m.moveTempCursor(-1)
				case listModePins:
					m.movePinsCursor(-1)
				default:
					m.moveTreeCursor(-1)
				}
				return m, nil

			case "down":
				switch m.listMode {
				case listModeTemporary:
					m.moveTempCursor(1)
				case listModePins:
					m.movePinsCursor(1)
				default:
					m.moveTreeCursor(1)
				}
				return m, nil
			}

			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.rebuildTree()
			m.syncSelectedNote()
			return m, cmd
		}

		if key.Matches(msg, keys.Search) {
			m.searchMode = true
			m.searchInput.Focus()
			m.status = "search"
			return m, nil
		}

		if msg.String() == "esc" && strings.TrimSpace(m.searchInput.Value()) != "" {
			m.searchInput.SetValue("")
			m.searchInput.Blur()
			m.searchMode = false
			m.rebuildTree()
			m.syncSelectedNote()
			m.status = "search cleared"
			return m, nil
		}

		if key.Matches(msg, keys.Focus) {
			if m.focus == focusTree {
				m.focus = focusPreview
				m.status = "preview focused"
			} else {
				m.focus = focusTree
				m.status = "tree focused"
			}
			m.pendingG = false
			m.pendingBracketDir = ""
			return m, nil
		}

		if m.focus == focusPreview && !m.showHelp && !m.showCreateCategory && !m.showMove &&
			!m.showRename {
			switch msg.String() {
			case "esc":
				m.focus = focusTree
				m.status = "tree focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case "j", "down":
				m.preview.ScrollDown(1)
				m.status = "preview focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case "k", "up":
				m.preview.ScrollUp(1)
				m.status = "preview focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case "pgdown", "ctrl+f":
				m.preview.PageDown()
				m.status = "preview focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case "pgup", "ctrl+b":
				m.preview.PageUp()
				m.status = "preview focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case "G":
				m.preview.GotoBottom()
				m.status = "preview bottom"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case "g":
				if m.pendingG {
					m.preview.GotoTop()
					m.status = "preview top"
					m.pendingG = false
					return m, nil
				}
				m.pendingG = true
				m.pendingBracketDir = ""
				return m, nil

			case "]":
				m.pendingBracketDir = "]"
				m.pendingG = false
				return m, nil

			case "[":
				m.pendingBracketDir = "["
				m.pendingG = false
				return m, nil

			case "h":
				if m.pendingBracketDir == "]" {
					m.jumpToNextHeading()
					m.pendingBracketDir = ""
					return m, nil
				}
				if m.pendingBracketDir == "[" {
					m.jumpToPrevHeading()
					m.pendingBracketDir = ""
					return m, nil
				}
			}
		}

		m.pendingG = false
		m.pendingBracketDir = ""

		if key.Matches(msg, keys.Move) {
			m.armMoveCurrent()
			return m, nil
		}

		if key.Matches(msg, keys.Rename) {
			m.armRenameCurrent()
			return m, nil
		}

		if key.Matches(msg, keys.Delete) {
			m.armDeleteCurrent()
			return m, nil
		}

		if key.Matches(msg, keys.Pin) {
			if err := m.togglePinCurrent(); err != nil {
				m.status = "pin failed: " + err.Error()
			}
			return m, nil
		}

		if key.Matches(msg, keys.CreateCategory) {
			if m.listMode != listModeNotes {
				m.status = "categories only available in notes tree"
				return m, nil
			}
			m.showCreateCategory = true
			m.categoryInput.SetValue(m.currentCategoryPrefix())
			m.categoryInput.Focus()
			m.categoryInput.CursorEnd()
			m.status = "new category"
			return m, nil
		}

		if key.Matches(msg, keys.Refresh) {
			m.status = "refreshing..."
			return m, refreshAllCmd(m.rootDir)
		}

		if key.Matches(msg, keys.NewTemporaryNote) {
			return m, createTemporaryNoteCmd(m.rootDir)
		}

		if key.Matches(msg, keys.NewNote) {
			if m.listMode == listModeTemporary {
				return m, createTemporaryNoteCmd(m.rootDir)
			}
			if m.listMode == listModePins {
				m.status = "press enter to jump to item first"
				return m, nil
			}
			return m, createNoteCmd(m.rootDir, m.currentTargetDir())
		}

		if key.Matches(msg, keys.TogglePreviewPrivacy) {
			if m.cfg.Preview.Privacy {
				m.status = "preview privacy forced by config"
				return m, nil
			}

			m.previewPrivacyEnabled = !m.previewPrivacyEnabled
			m.previewPath = ""

			if m.previewPrivacyEnabled {
				m.status = "preview privacy enabled"
			} else {
				m.status = "preview privacy disabled"
			}

			m.refreshPreview()
			return m, nil
		}

		switch msg.String() {
		case "[":
			if m.listMode == listModePins {
				m.switchToNotesMode()
			} else {
				m.toggleNotesTemporaryMode()
			}
			return m, nil

		case "]":
			if m.listMode == listModePins {
				m.switchToTemporaryMode()
			} else {
				m.toggleNotesTemporaryMode()
			}
			return m, nil

		case "up", "k":
			switch m.listMode {
			case listModeTemporary:
				m.moveTempCursor(-1)
			case listModePins:
				m.movePinsCursor(-1)
			default:
				m.moveTreeCursor(-1)
			}
			return m, nil

		case "down", "j":
			switch m.listMode {
			case listModeTemporary:
				m.moveTempCursor(1)
			case listModePins:
				m.movePinsCursor(1)
			default:
				m.moveTreeCursor(1)
			}
			return m, nil

		case "right", "l":
			if m.listMode == listModeNotes {
				m.expandCurrentCategory()
			}
			return m, nil

		case "left", "h":
			if m.listMode == listModeNotes {
				m.collapseCurrentCategory()
			}
			return m, nil
		}

		if key.Matches(msg, keys.Open) || key.Matches(msg, keys.ToggleCategory) {
			return m, m.activateCurrentItem()
		}
	}

	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	_ = cmd

	return m, nil
}

func (m Model) modalDimensions(minWidth, maxWidth int) (int, int) {
	modalWidth := min(maxWidth, max(minWidth, m.width-10))
	innerWidth := max(20, modalWidth-(modalPaddingX*2)-2)
	return modalWidth, innerWidth
}

func (m Model) dashboardRecentNotes(limit int) []dashboardRecentNote {
	if limit <= 0 {
		return nil
	}

	out := make([]dashboardRecentNote, 0, len(m.notes)+len(m.tempNotes))

	for _, n := range m.notes {
		out = append(out, dashboardRecentNote{
			Note:    n,
			IsTemp:  false,
			Display: n.Title(),
		})
	}

	for _, n := range m.tempNotes {
		out = append(out, dashboardRecentNote{
			Note:    n,
			IsTemp:  true,
			Display: n.Title(),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Note.ModTime.After(out[j].Note.ModTime)
	})

	if len(out) > limit {
		out = out[:limit]
	}

	return out
}

func (m Model) openDashboardRecent(index int) tea.Cmd {
	items := m.dashboardRecentNotes(5)
	if index < 0 || index >= len(items) {
		return nil
	}
	return editor.Open(items[index].Note.Path)
}

func (m Model) dashboardCategoriesCount() int {
	count := 0
	for _, c := range m.categories {
		if c.RelPath != "" {
			count++
		}
	}
	return count
}

func (m Model) dashboardPinnedNotesCount() int {
	count := 0
	for _, p := range m.pinnedNotes {
		if p {
			count++
		}
	}
	return count
}

func (m Model) dashboardPinnedCategoriesCount() int {
	count := 0
	for _, p := range m.pinnedCats {
		if p {
			count++
		}
	}
	return count
}

func (m Model) dashboardPrivacySummary() string {
	if m.cfg.Preview.Privacy {
		return "config"
	}
	if m.previewPrivacyEnabled {
		return "on"
	}
	return "off"
}

func (m Model) dashboardThemeName() string {
	name := strings.TrimSpace(m.cfg.Theme.Name)
	if name == "" {
		return "default"
	}
	return name
}
