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
	"github.com/charmbracelet/lipgloss"

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

type previewRenderedMsg struct {
	forPath             string
	baseContent         string
	rawContent          string
	privacyForcedByNote bool
}

type todoModifiedMsg struct {
	path string
	err  error
}

type encryptedEdit struct {
	origPath string
	tempPath string
}

type (
	previewLockedMsg struct{ path string }
	encryptNoteMsg   struct {
		path string
		err  error
	}
	decryptNoteMsg struct {
		path string
		err  error
	}
	openEncryptedNoteReadyMsg struct {
		origPath, tempPath string
		err                error
	}
	reencryptFinishedMsg struct {
		newPath string
		err     error
	}
)

type previewTodoItem struct {
	rawLine  int
	rendLine int
	checked  bool
	text     string
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
	Tags    []string
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

	focus              paneFocus
	pendingG           bool
	pendingZ           bool
	pendingBracketDir  string
	previewHover       bool
	previewPaneX       int
	previewPaneY       int
	previewPaneW       int
	previewPaneH       int
	previewHeadings    []int
	previewMatches     []previewMatch
	previewMatchIndex  int
	previewBaseContent string
	pendingPreviewCmd  tea.Cmd

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

	previewTodos      []previewTodoItem
	previewTodoCursor int
	pendingTodoCursor int
	pendingT          bool
	showTodoAdd       bool
	showTodoEdit      bool
	todoInput         textinput.Model

	sessionPassphrase    string
	showPassphraseModal  bool
	passphraseInput      textinput.Model
	passphraseModalCtx   string
	showEncryptConfirm   bool
	encryptConfirmYes    bool
	pendingEncryptPath   string
	pendingEncryptedEdit *encryptedEdit

	startupError string

	sortByModTime bool
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
	searchInput.TextStyle = lipgloss.NewStyle().Foreground(textColor).Background(bgSoftColor)
	searchInput.PlaceholderStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(bgSoftColor)

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

	todoInput := textinput.New()
	todoInput.Placeholder = "Todo item text"
	todoInput.Prompt = ""
	todoInput.CharLimit = 300
	todoInput.Width = 48

	passphraseInput := textinput.New()
	passphraseInput.Placeholder = "Passphrase"
	passphraseInput.Prompt = ""
	passphraseInput.CharLimit = 256
	passphraseInput.Width = 48
	passphraseInput.EchoMode = textinput.EchoPassword
	passphraseInput.EchoCharacter = '•'

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
		todoInput:             todoInput,
		passphraseInput:       passphraseInput,
		preserveCursor:        -1,
		pendingTodoCursor:     -1,
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
		sortByModTime:         st.SortByModTime,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		refreshAllCmd(m.rootDir),
		startWatchTeaCmd(m.rootDir),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := m.handleMsg(msg)
	if m.pendingPreviewCmd != nil {
		previewCmd := m.pendingPreviewCmd
		m.pendingPreviewCmd = nil
		if cmd != nil {
			cmd = tea.Batch(cmd, previewCmd)
		} else {
			cmd = previewCmd
		}
	}
	return m, cmd
}

func (m Model) handleMsg(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case previewRenderedMsg:
		if msg.forPath != m.previewPath {
			return m, nil
		}
		m.previewBaseContent = msg.baseContent
		m.previewPrivacyForcedByNote = msg.privacyForcedByNote
		query := strings.TrimSpace(m.searchInput.Value())
		m.previewMatches = buildPreviewMatches(msg.baseContent, query)
		m.previewMatchIndex = 0
		highlighted := applyMatchHighlights(msg.baseContent, query, m.previewMatches, 0)
		m.previewContent = highlighted
		m.rebuildPreviewHeadingsFromRendered()
		if m.pendingTodoCursor >= 0 {
			m.previewTodoCursor = m.pendingTodoCursor
			m.pendingTodoCursor = -1
		} else {
			m.previewTodoCursor = 0
		}
		m.rebuildPreviewTodos(msg.rawContent, msg.baseContent)
		m.reapplyTodoHighlight()
		if len(m.previewMatches) > 0 && query != "" {
			m.scrollToMatchLine(m.previewMatches[0].line)
		} else {
			m.preview.GotoTop()
		}
		return m, nil

	case todoModifiedMsg:
		if msg.err != nil {
			m.status = "todo error: " + msg.err.Error()
			return m, nil
		}
		m.pendingTodoCursor = m.previewTodoCursor
		m.previewPath = ""
		m.status = "todo updated"
		return m, refreshAllCmd(m.rootDir)

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
		m.todoInput.Width = max(24, min(60, m.width-16))

		previewInnerWidth := max(20, rightWidth-8)
		previewInnerHeight := max(6, msg.Height-14)
		m.preview.Width = previewInnerWidth
		m.preview.Height = previewInnerHeight
		m.previewPath = ""
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

	case previewLockedMsg:
		if msg.path != m.previewPath {
			return m, nil
		}
		locked := m.lockedPreviewText()
		m.previewBaseContent = ""
		m.previewContent = locked
		m.previewTodos = nil
		m.preview.SetContent(locked)
		m.rebuildPreviewHeadingsFromRendered()
		m.previewMatches = nil
		m.preview.GotoTop()
		return m, nil

	case encryptNoteMsg:
		if msg.err != nil {
			m.status = "encryption failed: " + msg.err.Error()
			return m, nil
		}
		m.status = "note encrypted"
		m.previewPath = ""
		return m, refreshAllCmd(m.rootDir)

	case decryptNoteMsg:
		if msg.err != nil {
			m.status = "decryption failed: " + msg.err.Error()
			return m, nil
		}
		m.status = "note decrypted"
		m.previewPath = ""
		return m, refreshAllCmd(m.rootDir)

	case openEncryptedNoteReadyMsg:
		if msg.err != nil {
			m.status = "error opening note: " + msg.err.Error()
			return m, nil
		}
		m.pendingEncryptedEdit = &encryptedEdit{
			origPath: msg.origPath,
			tempPath: msg.tempPath,
		}
		return m, editor.Open(msg.tempPath)

	case reencryptFinishedMsg:
		if msg.err != nil {
			m.status = "re-encryption failed: " + msg.err.Error()
			return m, refreshAllCmd(m.rootDir)
		}
		m.status = "note saved and re-encrypted"
		return m, refreshAllCmd(m.rootDir)

	case editor.FinishedMsg:
		if msg.Err != nil {
			m.status = "editor error: " + msg.Err.Error()
			return m, nil
		}

		if m.pendingEncryptedEdit != nil {
			edit := m.pendingEncryptedEdit
			m.pendingEncryptedEdit = nil
			return m, reencryptFromTempCmd(edit.origPath, edit.tempPath, m.sessionPassphrase)
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
		m.pendingTodoCursor = m.previewTodoCursor
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
			switch {
			case msg.String() == "enter":
				m.showDashboard = false
				m.status = "workspace"
				return m, nil

			case key.Matches(msg, keys.BracketForward):
				m.showDashboard = false
				m.switchToTemporaryMode()
				m.status = "temporary"
				return m, nil

			case key.Matches(msg, keys.ShowPins):
				m.showDashboard = false
				m.listMode = listModePins
				m.status = "pins"
				m.syncSelectedNote()
				return m, nil

			case key.Matches(msg, keys.NewTemporaryNote):
				m.showDashboard = false
				return m, createTemporaryNoteCmd(m.rootDir)

			case msg.String() == "1":
				cmd := m.openDashboardRecent(0)
				if cmd != nil {
					m.showDashboard = false
					m.status = "opening recent note"
				}
				return m, cmd
			case msg.String() == "2":
				return m, m.openDashboardRecent(1)
			case msg.String() == "3":
				return m, m.openDashboardRecent(2)
			case msg.String() == "4":
				return m, m.openDashboardRecent(3)
			case msg.String() == "5":
				return m, m.openDashboardRecent(4)

			case key.Matches(msg, keys.Quit):
				if m.watcher != nil {
					_ = m.watcher.Close()
				}
				return m, tea.Quit
			}

			return m, nil
		}

		if m.showTodoAdd {
			switch msg.String() {
			case "esc":
				m.showTodoAdd = false
				m.todoInput.Blur()
				m.todoInput.SetValue("")
				m.status = "todo add cancelled"
				return m, nil
			case "enter":
				text := strings.TrimSpace(m.todoInput.Value())
				m.showTodoAdd = false
				m.todoInput.Blur()
				m.todoInput.SetValue("")
				if text == "" {
					m.status = "todo add cancelled"
					return m, nil
				}
				path := m.previewPath
				if path == "" {
					m.status = "no note selected"
					return m, nil
				}
				return m, addTodoCmd(path, text)
			}
			var cmd tea.Cmd
			m.todoInput, cmd = m.todoInput.Update(msg)
			return m, cmd
		}

		if m.showPassphraseModal {
			switch msg.String() {
			case "esc":
				m.showPassphraseModal = false
				m.passphraseInput.Blur()
				m.passphraseInput.SetValue("")
				m.status = "cancelled"
				return m, nil
			case "enter":
				passphrase := m.passphraseInput.Value()
				if strings.TrimSpace(passphrase) == "" {
					m.status = "passphrase cannot be empty"
					return m, nil
				}
				switch m.passphraseModalCtx {
				case "unlock", "unlock_edit", "decrypt":
					raw, err := notes.ReadAll(m.pendingEncryptPath)
					if err != nil {
						m.status = "error: " + err.Error()
						return m, nil
					}
					body := strings.TrimSpace(notes.StripFrontMatter(raw))
					if _, err := notes.DecryptBody(body, passphrase); err != nil {
						m.status = "wrong passphrase"
						m.passphraseInput.SetValue("")
						return m, nil
					}
					m.sessionPassphrase = passphrase
					m.showPassphraseModal = false
					m.passphraseInput.Blur()
					m.passphraseInput.SetValue("")
					if m.passphraseModalCtx == "unlock_edit" {
						return m, openEncryptedNoteCmd(m.pendingEncryptPath, m.sessionPassphrase)
					}
					if m.passphraseModalCtx == "decrypt" {
						m.showEncryptConfirm = true
						m.encryptConfirmYes = true
						m.status = "confirm: remove encryption?"
						return m, nil
					}
					m.previewPath = ""
					m.status = "passphrase accepted"
					m.refreshPreview()
					return m, nil
				case "encrypt":
					m.sessionPassphrase = passphrase
					m.showPassphraseModal = false
					m.passphraseInput.Blur()
					m.passphraseInput.SetValue("")
					m.showEncryptConfirm = true
					m.encryptConfirmYes = true
					m.status = "confirm: encrypt note?"
					return m, nil
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.passphraseInput, cmd = m.passphraseInput.Update(msg)
			return m, cmd
		}

		if m.showEncryptConfirm {
			switch msg.String() {
			case "esc":
				m.showEncryptConfirm = false
				m.status = "cancelled"
				return m, nil
			case "left", "right", "tab":
				m.encryptConfirmYes = !m.encryptConfirmYes
				return m, nil
			case "enter":
				m.showEncryptConfirm = false
				if !m.encryptConfirmYes {
					m.status = "cancelled"
					return m, nil
				}
				path := m.pendingEncryptPath
				passphrase := m.sessionPassphrase
				if m.passphraseModalCtx == "encrypt" {
					m.status = "encrypting..."
					return m, encryptNoteCmd(path, passphrase)
				}
				m.status = "decrypting..."
				return m, decryptNoteCmd(path, passphrase)
			}
			return m, nil
		}

		if m.showTodoEdit {
			switch msg.String() {
			case "esc":
				m.showTodoEdit = false
				m.todoInput.Blur()
				m.todoInput.SetValue("")
				m.status = "todo edit cancelled"
				return m, nil
			case "enter":
				text := strings.TrimSpace(m.todoInput.Value())
				m.showTodoEdit = false
				m.todoInput.Blur()
				m.todoInput.SetValue("")
				if text == "" {
					m.status = "todo edit cancelled"
					return m, nil
				}
				path := m.previewPath
				if path == "" || m.previewTodoCursor >= len(m.previewTodos) {
					m.status = "no todo selected"
					return m, nil
				}
				rawLine := m.previewTodos[m.previewTodoCursor].rawLine
				return m, editTodoCmd(path, rawLine, text)
			}
			var cmd tea.Cmd
			m.todoInput, cmd = m.todoInput.Update(msg)
			return m, cmd
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
			switch {
			case msg.String() == "esc":
				m.deletePending = nil
				m.status = "delete cancelled"
				return m, nil
			case key.Matches(msg, keys.DeleteConfirm):
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
			case "tab":
				m.searchMode = false
				m.searchInput.Blur()
				m.focus = focusPreview
				m.status = "preview focused"
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
			m.previewPath = ""
			m.rebuildTree()
			m.syncSelectedNote()
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

		if key.Matches(msg, keys.SortToggle) {
			m.sortByModTime = !m.sortByModTime
			_ = m.saveTreeState()
			m.rebuildTree()
			if m.sortByModTime {
				m.status = "sorting by modified time"
			} else {
				m.status = "sorting alphabetically"
			}
			return m, nil
		}

		if key.Matches(msg, keys.ShowHelp) {
			m.showHelp = true
			m.status = "help"
			return m, nil
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
			if !key.Matches(msg, keys.PendingZ) {
				m.pendingZ = false
			}
			if m.pendingT && !key.Matches(msg, keys.TodoKey) && !key.Matches(msg, keys.TodoAdd) &&
				!key.Matches(msg, keys.TodoDelete) &&
				!key.Matches(msg, keys.TodoEdit) {
				m.pendingT = false
			}
			switch {
			case msg.String() == "esc":
				m.focus = focusTree
				m.status = "tree focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				m.pendingT = false
				return m, nil

			case key.Matches(msg, keys.MoveDown):
				m.preview.ScrollDown(1)
				m.status = "preview focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.MoveUp):
				m.preview.ScrollUp(1)
				m.status = "preview focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.ScrollPageDown):
				m.preview.PageDown()
				m.status = "preview focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.ScrollPageUp):
				m.preview.PageUp()
				m.status = "preview focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.ScrollHalfPageUp):
				m.preview.ScrollUp(m.preview.Height / 2)
				m.status = "preview focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.ScrollHalfPageDown):
				m.preview.ScrollDown(m.preview.Height / 2)
				m.status = "preview focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.JumpBottom):
				m.preview.GotoBottom()
				m.status = "preview bottom"
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.PendingG):
				if m.pendingG {
					m.preview.GotoTop()
					m.status = "preview top"
					m.pendingG = false
					return m, nil
				}
				m.pendingG = true
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.BracketForward):
				m.pendingBracketDir = "]"
				m.pendingG = false
				return m, nil

			case key.Matches(msg, keys.BracketBackward):
				m.pendingBracketDir = "["
				m.pendingG = false
				return m, nil

			case key.Matches(msg, keys.HeadingJumpKey):
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

			case key.Matches(msg, keys.TodoKey):
				if m.pendingBracketDir == "]" {
					m.jumpToNextTodo()
					m.pendingBracketDir = ""
					m.pendingT = false
					return m, nil
				}
				if m.pendingBracketDir == "[" {
					m.jumpToPrevTodo()
					m.pendingBracketDir = ""
					m.pendingT = false
					return m, nil
				}
				if m.pendingT {
					m.pendingT = false
					return m, m.toggleCurrentPreviewTodo()
				}
				m.pendingT = true
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.TodoAdd) && m.pendingT:
				m.pendingT = false
				path := m.previewPath
				if path == "" {
					m.status = "no note selected"
					return m, nil
				}
				m.showTodoAdd = true
				m.todoInput.SetValue("")
				m.todoInput.Focus()
				m.status = "add todo"
				return m, nil

			case key.Matches(msg, keys.TodoDelete) && m.pendingT:
				m.pendingT = false
				return m, m.deleteCurrentPreviewTodo()

			case key.Matches(msg, keys.TodoEdit) && m.pendingT:
				m.pendingT = false
				return m, m.armEditCurrentPreviewTodo()

			case key.Matches(msg, keys.NextMatch):
				m.jumpToNextMatch()
				return m, nil

			case key.Matches(msg, keys.PrevMatch):
				m.jumpToPrevMatch()
				return m, nil

			case key.Matches(msg, keys.PendingZ):
				if m.pendingZ {
					m.centerCurrentMatch()
					m.pendingZ = false
					m.pendingG = false
					m.pendingBracketDir = ""
					return m, nil
				}
				m.pendingZ = true
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil
			}
		}

		// Tree navigation: gg and G.
		if m.focus == focusTree {
			switch {
			case key.Matches(msg, keys.JumpBottom):
				switch m.listMode {
				case listModeTemporary:
					items := m.filteredTempNotes()
					if len(items) > 0 {
						m.tempCursor = len(items) - 1
						m.syncSelectedNote()
					}
				case listModePins:
					items := m.filteredPinnedItems()
					if len(items) > 0 {
						m.pinsCursor = len(items) - 1
						m.syncSelectedNote()
					}
				default:
					if len(m.treeItems) > 0 {
						m.treeCursor = len(m.treeItems) - 1
						m.syncSelectedNote()
					}
				}
				m.pendingG = false
				return m, nil

			case key.Matches(msg, keys.PendingG):
				if m.pendingG {
					m.pendingG = false
					switch m.listMode {
					case listModeTemporary:
						m.tempCursor = 0
						m.syncSelectedNote()
					case listModePins:
						m.pinsCursor = 0
						m.syncSelectedNote()
					default:
						m.treeCursor = 0
						m.syncSelectedNote()
					}
				} else {
					m.pendingG = true
				}
				return m, nil

			case key.Matches(msg, keys.ScrollHalfPageUp):
				half := max(1, m.preview.Height/2)
				switch m.listMode {
				case listModeTemporary:
					m.moveTempCursor(-half)
				case listModePins:
					m.movePinsCursor(-half)
				default:
					m.moveTreeCursor(-half)
				}
				return m, nil

			case key.Matches(msg, keys.ScrollHalfPageDown):
				half := max(1, m.preview.Height/2)
				switch m.listMode {
				case listModeTemporary:
					m.moveTempCursor(half)
				case listModePins:
					m.movePinsCursor(half)
				default:
					m.moveTreeCursor(half)
				}
				return m, nil
			}
		}

		m.pendingG = false
		m.pendingBracketDir = ""
		m.pendingT = false

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

		if key.Matches(msg, keys.NewTodoList) {
			if m.listMode == listModeTemporary || m.listMode == listModePins {
				m.status = "todo lists only available in notes tree"
				return m, nil
			}
			return m, createTodoNoteCmd(m.rootDir, m.currentTargetDir())
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

		switch {
		case key.Matches(msg, keys.BracketBackward):
			if m.listMode == listModePins {
				m.switchToNotesMode()
			} else {
				m.toggleNotesTemporaryMode()
			}
			return m, nil

		case key.Matches(msg, keys.BracketForward):
			if m.listMode == listModePins {
				m.switchToTemporaryMode()
			} else {
				m.toggleNotesTemporaryMode()
			}
			return m, nil

		case key.Matches(msg, keys.MoveUp):
			switch m.listMode {
			case listModeTemporary:
				m.moveTempCursor(-1)
			case listModePins:
				m.movePinsCursor(-1)
			default:
				m.moveTreeCursor(-1)
			}
			return m, nil

		case key.Matches(msg, keys.MoveDown):
			switch m.listMode {
			case listModeTemporary:
				m.moveTempCursor(1)
			case listModePins:
				m.movePinsCursor(1)
			default:
				m.moveTreeCursor(1)
			}
			return m, nil

		case key.Matches(msg, keys.ExpandCategory):
			if m.listMode == listModeNotes {
				m.expandCurrentCategory()
			}
			return m, nil

		case key.Matches(msg, keys.CollapseCategory):
			if m.listMode == listModeNotes {
				m.collapseCurrentCategory()
			}
			return m, nil
		}

		if key.Matches(msg, keys.ToggleEncryption) {
			m.armToggleEncryption()
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

func (m Model) lockedPreviewText() string {
	return "[encrypted]\n\nThis note is encrypted.\nPress E to enter your passphrase."
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
