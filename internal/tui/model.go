// Package tui implements the Bubble Tea terminal UI for noteui.
package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/editor"
	"atbuy/noteui/internal/notes"
	"atbuy/noteui/internal/state"
	notesync "atbuy/noteui/internal/sync"
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
	listModeTodos
)

const (
	sortAlpha    = "alpha"
	sortModified = "modified"
	sortCreated  = "created"
	sortSize     = "size"
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
	treeRemoteNote
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

const (
	conflictResolutionKeepLocal = iota
	conflictResolutionKeepRemote
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
	kind      deleteTargetKind
	relPath   string
	name      string
	notePaths []string
}

type noteDeletedMsg struct {
	path   string
	result notes.TrashResult
	err    error
}

type categoryDeletedMsg struct {
	relPath string
	result  notes.TrashResult
	err     error
}

type restoreFinishedMsg struct {
	label string
	err   error
}

type undoableDelete struct {
	label   string
	results []notes.TrashResult
}

type previewRenderedMsg struct {
	forPath             string
	baseContent         string
	rawContent          string
	privacyForcedByNote bool
	lineNumberStart     int
	todoLineOffset      int
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
	rawLine     int
	rendLine    int
	rendEndLine int
	checked     bool
	text        string
	raw         string
}

type previewLinkItem struct {
	rendLine    int
	rendCol     int // byte offset in the ANSI-stripped line where the match starts
	rendLen     int // bytes to highlight on rendLine (capped at the first newline in the match)
	rendEndLine int // line where the full match ends
	rendEndCol  int // byte offset in rendEndLine where the full match ends (exclusive)
	target      string
	isWikilink  bool
	showTarget  bool
}

type treeItem struct {
	Kind       treeItemKind
	Name       string
	RelPath    string
	Depth      int
	Expanded   bool
	Note       *notes.Note
	RemoteNote *notesync.RemoteNoteMeta
	Category   *notes.Category
	MatchHint  string
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
	case treeRemoteNote:
		if t.RemoteNote != nil {
			return "r:" + t.RemoteNote.ID
		}
		return "r:" + t.RelPath
	default:
		return t.RelPath
	}
}

type Model struct {
	rootDir           string
	workspaceName     string
	workspaceLabel    string
	workspaceOverride bool
	workspaceOptions  []workspaceOption
	workspaceCursor   int
	version           string

	notes            []notes.Note
	categories       []notes.Category
	remoteOnlyNotes  []notesync.RemoteNoteMeta
	remoteCategories []notes.Category
	expanded         map[string]bool
	tempNotes        []notes.Note
	tempCursor       int
	pinsCursor       int
	todoCursor       int
	listMode         listMode
	lastNonPinsMode  listMode

	todoItems       []todoListItem
	treeItems       []treeItem
	treeCursor      int
	markedTreeItems map[string]bool
	selected        *notes.Note
	width           int
	height          int
	previewWidth    int
	status          string
	deferredStatus  string

	cfg                    config.Config
	preview                viewport.Model
	previewPath            string
	previewContent         string
	previewLineNumberStart int

	previewPrivacyEnabled      bool
	previewPrivacyForcedByNote bool
	previewLineNumbersEnabled  bool

	watcher     interface{ Close() error }
	watchEvents <-chan teaMsg

	focus                 paneFocus
	pendingG              bool
	pendingZ              bool
	pendingBracketDir     string
	previewHover          bool
	previewPaneX          int
	previewPaneY          int
	previewPaneW          int
	previewPaneH          int
	previewHeadings       []int
	previewMatches        []previewMatch
	previewMatchIndex     int
	previewBaseContent    string
	previewRawContent     string
	pendingPreviewCmd     tea.Cmd
	pendingPreviewYOffset int

	state          state.State
	workspaceState state.WorkspaceState
	pinnedNotes    map[string]bool
	pinnedCats     map[string]bool

	showCommandPalette     bool
	commandPaletteInput    textinput.Model
	commandPaletteItems    []paletteItem
	commandPaletteFiltered []paletteItem
	commandPaletteCursor   int
	editorLinkPickerMode   bool

	showHelp             bool
	helpScroll           int
	helpRowsCache        []string
	helpRowsCacheQuery   string
	helpRowsCacheWidth   int
	helpBodyCache        string
	helpBodyCacheQuery   string
	helpBodyCacheWidth   int
	helpBodyCacheRows    int
	helpBodyCacheScroll  int
	helpModalCache       string
	helpModalCacheQuery  string
	helpModalCacheWidth  int
	helpModalCacheHeight int
	helpModalCacheRows   int
	helpModalCacheScroll int
	helpMouseSuppressed  bool
	showDashboard        bool

	showCreateCategory bool
	categoryInput      textinput.Model

	showMoveBrowser  bool
	moveBrowserMode  moveBrowserMode
	moveDestCursor   int
	moveBrowserError string

	showMove    bool
	moveInput   textinput.Model
	movePending *movePending

	showRename    bool
	renameInput   textinput.Model
	renamePending *renamePending

	showAddTag bool
	tagInput   textinput.Model
	helpInput  textinput.Model

	showWorkspacePicker        bool
	showSyncProfilePicker      bool
	showSyncDebugModal         bool
	showSyncTimeline           bool
	syncTimelineOffset         int
	syncTimelineEvents         []notesync.SyncEvent
	conflictResolutionChoice   int
	syncProfileNames           []string
	syncProfileCursor          int
	showSyncProfileMigration   bool
	syncProfileMigrationChoice int
	pendingSyncProfileChange   *syncProfileChange

	searchInput textinput.Model
	searchMode  bool

	deletePending  *deletePending
	lastDeletion   *undoableDelete
	preserveCursor int

	previewTodos        []previewTodoItem
	previewTodoCursor   int
	pendingTodoCursor   int
	pendingT            bool
	previewTodoNavMode  bool
	previewLinks        []previewLinkItem
	previewLinkCursor   int
	previewLinkNavMode  bool
	showTodoAdd         bool
	showTodoEdit        bool
	showTodoDueDate     bool
	showTodoPriority    bool
	todoInput           textinput.Model
	dueDateInput        textinput.Model
	priorityInput       textinput.Model
	showEditorURLPrompt bool
	editorURLInput      textinput.Model

	sessionPassphrase    string
	showPassphraseModal  bool
	passphraseInput      textinput.Model
	passphraseModalCtx   string
	showEncryptConfirm   bool
	encryptConfirmYes    bool
	pendingEncryptPath   string
	pendingEncryptedEdit *encryptedEdit
	dailyNoteOpen        bool
	previewWikilinks     []string

	startupError string

	sessionToken             int
	sortMethod               string
	sortReverse              bool
	pendingSort              bool
	syncDebounceToken        int
	syncRunning              bool
	syncSpinnerFrame         int
	syncRecords              map[string]notesync.NoteRecord
	syncInFlight             map[string]bool
	startupSyncChecked       bool
	pendingSyncedPinRelPaths []string
	pendingSyncedCategories  []string
	applyPendingSyncedPins   bool

	showNoteHistory    bool
	noteHistoryEntries []notes.HistoryEntry
	noteHistoryCursor  int
	noteHistoryRelPath string
	noteHistoryAbsPath string

	showTrashBrowser   bool
	trashBrowserItems  []notes.TrashedItem
	trashBrowserCursor int

	showTemplatePicker     bool
	templatePickerEditMode bool
	templateItems          []notes.Template
	templatePickerCursor   int
	templatePickerRelDir   string

	showThemePicker         bool
	themePickerCursor       int
	themePickerScrollOffset int
	themePickerOrigTheme    string

	editorActive         bool
	editorFullscreen     bool
	editorModel          *EditorModel
	editorRestoreFocus   paneFocus
	pendingInAppEditPath string
	pendingInAppEditRel  string
	pendingInAppEditTemp bool
	pendingSelectRelPath string
	pendingSelectIsTemp  bool
}

type dataLoadedMsg struct {
	notes        []notes.Note
	tempNotes    []notes.Note
	categories   []notes.Category
	err          error
	sessionToken int
}

type noteCreatedMsg struct {
	path string
	err  error
}

type categoryCreatedMsg struct {
	relPath string
	err     error
}

type noteTaggedMsg struct {
	path string
	tags []string
	err  error
}

type (
	helpMouseResumeMsg struct{}
	syncSpinnerTickMsg struct{}
)

func disableMouseCmd() tea.Cmd {
	return func() tea.Msg { return tea.DisableMouse() }
}

func enableMouseCellMotionCmd() tea.Cmd {
	return func() tea.Msg { return tea.EnableMouseCellMotion() }
}

func helpMouseResumeCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return helpMouseResumeMsg{}
	})
}

func syncSpinnerCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
		return syncSpinnerTickMsg{}
	})
}

func New(root, startupError string, cfg config.Config, version string) Model {
	return NewWithSession(root, startupError, cfg, version, WorkspaceSession{})
}

func NewWithSession(root, startupError string, cfg config.Config, version string, session WorkspaceSession) Model {
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

	tagInput := textinput.New()
	tagInput.Placeholder = "project-x, urgent"
	tagInput.Prompt = "Tags: "
	tagInput.CharLimit = 300
	tagInput.Width = 48

	helpInput := textinput.New()
	helpInput.Placeholder = "Filter commands..."
	helpInput.Prompt = "Filter: "
	helpInput.CharLimit = 200
	helpInput.Width = 32
	helpInput.TextStyle = lipgloss.NewStyle().Foreground(modalTextColor).Background(modalBgColor)
	helpInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(modalMutedColor).Background(modalBgColor)

	todoInput := textinput.New()
	todoInput.Placeholder = "Todo item text"
	todoInput.Prompt = ""
	todoInput.CharLimit = 300
	todoInput.Width = 48

	dueDateInput := textinput.New()
	dueDateInput.Placeholder = "YYYY-MM-DD"
	dueDateInput.Prompt = ""
	dueDateInput.CharLimit = 32
	dueDateInput.Width = 24

	priorityInput := textinput.New()
	priorityInput.Placeholder = "1, 2, 3..."
	priorityInput.Prompt = ""
	priorityInput.CharLimit = 8
	priorityInput.Width = 16

	commandPaletteInput := textinput.New()
	commandPaletteInput.Placeholder = "Search notes and commands..."
	commandPaletteInput.Prompt = "> "
	commandPaletteInput.CharLimit = 200
	commandPaletteInput.Width = 60

	passphraseInput := textinput.New()
	passphraseInput.Placeholder = "Passphrase"
	passphraseInput.Prompt = ""
	passphraseInput.CharLimit = 256
	passphraseInput.Width = 48
	passphraseInput.EchoMode = textinput.EchoPassword
	passphraseInput.EchoCharacter = '•'

	editorURLInput := textinput.New()
	editorURLInput.Placeholder = "https://example.com"
	editorURLInput.Prompt = "URL: "
	editorURLInput.CharLimit = 1000
	editorURLInput.Width = 64

	vp := viewport.New(0, 0)

	st, _ := state.Load()
	options := sortedWorkspaceOptions(cfg)

	m := Model{
		rootDir:                   root,
		workspaceName:             strings.TrimSpace(session.Name),
		workspaceLabel:            strings.TrimSpace(session.Label),
		workspaceOverride:         session.Override,
		workspaceOptions:          options,
		workspaceCursor:           0,
		version:                   version,
		status:                    "loading notes...",
		commandPaletteInput:       commandPaletteInput,
		categoryInput:             categoryInput,
		searchInput:               searchInput,
		moveInput:                 moveInput,
		renameInput:               renameInput,
		tagInput:                  tagInput,
		helpInput:                 helpInput,
		todoInput:                 todoInput,
		dueDateInput:              dueDateInput,
		priorityInput:             priorityInput,
		editorURLInput:            editorURLInput,
		passphraseInput:           passphraseInput,
		preserveCursor:            -1,
		pendingTodoCursor:         -1,
		pendingPreviewYOffset:     -1,
		startupError:              startupError,
		cfg:                       cfg,
		preview:                   vp,
		focus:                     focusTree,
		state:                     st,
		markedTreeItems:           make(map[string]bool),
		listMode:                  listModeNotes,
		lastNonPinsMode:           listModeNotes,
		tempCursor:                0,
		pinsCursor:                0,
		todoCursor:                0,
		previewPrivacyEnabled:     cfg.Preview.Privacy,
		previewLineNumbersEnabled: cfg.Preview.LineNumbers,
		editorFullscreen:          cfg.Editor.Fullscreen,
		showDashboard:             cfg.Dashboard && !session.StartWithPicker,
		showWorkspacePicker:       session.StartWithPicker,
		syncRecords:               map[string]notesync.NoteRecord{},
		syncInFlight:              map[string]bool{},
		startupSyncChecked:        !notesync.HasSyncProfile(cfg.Sync),
		sessionToken:              1,
	}
	m.loadCurrentWorkspaceState()
	if session.StartWithPicker {
		m.status = "select workspace"
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return m.startWorkspaceSessionCmd()
}

func (m *Model) refreshSyncRecords() {
	m.syncRecords = map[string]notesync.NoteRecord{}
	if !notesync.HasSyncProfile(m.cfg.Sync) {
		return
	}
	records, err := notesync.LoadNoteRecords(m.rootDir)
	if err != nil {
		return
	}
	for _, rec := range records {
		relPath := filepath.ToSlash(strings.TrimSpace(rec.RelPath))
		if relPath == "" {
			continue
		}
		m.syncRecords[relPath] = rec
	}
}

func (m Model) pendingSyncRelPaths() map[string]bool {
	out := make(map[string]bool)
	for _, note := range m.notes {
		if note.SyncClass != notes.SyncClassSynced && note.SyncClass != notes.SyncClassShared {
			continue
		}
		relPath := filepath.ToSlash(note.RelPath)
		rec, ok := m.syncRecords[relPath]
		if !ok || rec.Conflict != nil || strings.TrimSpace(rec.LastSyncError) != "" || rec.LastSyncAt.IsZero() {
			out[relPath] = true
			continue
		}
		raw, err := notes.ReadAll(note.Path)
		if err != nil {
			out[relPath] = true
			continue
		}
		if rec.LastSyncedHash != notesync.HashContent(raw) || rec.Encrypted != note.Encrypted {
			out[relPath] = true
		}
	}
	return out
}

// activeWorkspaceSyncRemoteRoot returns the per-workspace sync_remote_root override
// for the currently active workspace, or an empty string if none is configured.
func (m Model) activeWorkspaceSyncRemoteRoot() string {
	if strings.TrimSpace(m.workspaceName) == "" {
		return ""
	}
	ws, ok := m.cfg.Workspaces[m.workspaceName]
	if !ok {
		return ""
	}
	return strings.TrimSpace(ws.SyncRemoteRoot)
}

func (m *Model) openTrashBrowser() tea.Cmd {
	return loadTrashBrowserCmd(m.rootDir)
}

func (m *Model) openNoteHistory() tea.Cmd {
	note := m.currentLocalNote()
	if note == nil {
		m.status = "select a local note to view its history"
		return nil
	}
	relPath := strings.TrimSpace(note.RelPath)
	if relPath == "" {
		m.status = "could not determine note path"
		return nil
	}
	return loadNoteHistoryCmd(m.rootDir, relPath)
}

func (m *Model) startSyncRun() tea.Cmd {
	if m.syncRunning || !notesync.HasSyncProfile(m.cfg.Sync) {
		return nil
	}
	m.syncRunning = true
	m.syncSpinnerFrame = 0
	m.syncInFlight = m.pendingSyncRelPaths()
	return batchCmds(
		syncNowCmd(m.rootDir, m.activeWorkspaceSyncRemoteRoot(), m.cfg.Sync, m.localPinnedNoteKeys(), m.localPinnedCategories(), m.sessionToken),
		syncSpinnerCmd(),
	)
}

func (m *Model) finishSyncRun() {
	m.syncRunning = false
	m.syncSpinnerFrame = 0
	m.syncInFlight = map[string]bool{}
	m.refreshSyncRecords()
}

func (m *Model) startSyncVisual(relPaths ...string) tea.Cmd {
	if len(relPaths) == 0 {
		return nil
	}
	if len(m.syncInFlight) == 0 {
		m.syncSpinnerFrame = 0
	}
	if m.syncInFlight == nil {
		m.syncInFlight = map[string]bool{}
	}
	for _, relPath := range relPaths {
		relPath = filepath.ToSlash(strings.TrimSpace(relPath))
		if relPath == "" {
			continue
		}
		m.syncInFlight[relPath] = true
	}
	if len(m.syncInFlight) == 0 {
		return nil
	}
	return syncSpinnerCmd()
}

type noteSyncVisualState int

const (
	noteSyncVisualLocal noteSyncVisualState = iota
	noteSyncVisualPending
	noteSyncVisualSyncing
	noteSyncVisualHealthy
	noteSyncVisualSharedHealthy
	noteSyncVisualSharedPending
	noteSyncVisualSharedSyncing
)

func (m Model) noteSyncVisualState(note *notes.Note) noteSyncVisualState {
	if note == nil {
		return noteSyncVisualLocal
	}
	relPath := filepath.ToSlash(strings.TrimSpace(note.RelPath))
	if note.SyncClass == notes.SyncClassShared {
		if m.syncInFlight[relPath] {
			return noteSyncVisualSharedSyncing
		}
		if !m.hasHealthySyncRecord(relPath) {
			return noteSyncVisualSharedPending
		}
		return noteSyncVisualSharedHealthy
	}
	if note.SyncClass != notes.SyncClassSynced {
		return noteSyncVisualLocal
	}
	if m.syncInFlight[relPath] {
		return noteSyncVisualSyncing
	}
	if !m.hasHealthySyncRecord(relPath) {
		return noteSyncVisualPending
	}
	return noteSyncVisualHealthy
}

func (m Model) hasHealthySyncRecord(relPath string) bool {
	rec, ok := m.syncRecords[filepath.ToSlash(strings.TrimSpace(relPath))]
	if !ok {
		return false
	}
	if rec.Conflict != nil || strings.TrimSpace(rec.LastSyncError) != "" || rec.LastSyncAt.IsZero() {
		return false
	}
	return true
}

func (m Model) blinkingSyncMarker() (string, lipgloss.Color) {
	if m.syncSpinnerFrame%2 == 0 {
		return "● ", syncingNoteColor
	}
	return "◌ ", syncingNoteColor
}

func (m Model) noteSyncMarker(note *notes.Note) (string, lipgloss.Color) {
	switch m.noteSyncVisualState(note) {
	case noteSyncVisualLocal:
		return "○ ", unsyncedNoteColor
	case noteSyncVisualHealthy:
		return "● ", syncedNoteColor
	case noteSyncVisualSyncing:
		return m.blinkingSyncMarker()
	case noteSyncVisualSharedHealthy:
		return "◆ ", sharedNoteColor
	case noteSyncVisualSharedSyncing:
		return m.blinkingSyncMarker()
	case noteSyncVisualSharedPending:
		return "◆ ", unsyncedNoteColor
	default:
		return "● ", unsyncedNoteColor
	}
}

func (m *Model) setRemoteOnlyNotes(items []notesync.RemoteNoteMeta) bool {
	normalized := normalizeRemoteOnlyNotes(items)
	if remoteNoteMetaSlicesEqual(m.remoteOnlyNotes, normalized) {
		return false
	}
	m.remoteOnlyNotes = normalized
	m.remoteCategories = remoteCategoriesFromNotes(normalized)
	return true
}

func normalizeRemoteOnlyNotes(items []notesync.RemoteNoteMeta) []notesync.RemoteNoteMeta {
	if len(items) == 0 {
		return nil
	}
	out := make([]notesync.RemoteNoteMeta, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.RelPath = filepath.ToSlash(strings.TrimSpace(item.RelPath))
		item.Title = strings.TrimSpace(item.Title)
		if item.ID == "" || item.RelPath == "" || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RelPath == out[j].RelPath {
			return out[i].ID < out[j].ID
		}
		return out[i].RelPath < out[j].RelPath
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func remoteNoteMetaSlicesEqual(a, b []notesync.RemoteNoteMeta) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func remoteCategoriesFromNotes(items []notesync.RemoteNoteMeta) []notes.Category {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(items))
	out := make([]notes.Category, 0, len(items))
	for _, item := range items {
		dir := filepath.Dir(item.RelPath)
		for dir != "." && dir != "" {
			dir = filepath.ToSlash(dir)
			if !seen[dir] {
				seen[dir] = true
				out = append(out, notes.Category{Name: filepath.Base(dir), RelPath: dir, Depth: strings.Count(dir, "/")})
			}
			next := filepath.Dir(dir)
			if next == dir {
				break
			}
			dir = next
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].RelPath < out[j].RelPath })
	return out
}

func (m Model) currentRemoteOnlyNote() *notesync.RemoteNoteMeta {
	if m.listMode != listModeNotes {
		return nil
	}
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeRemoteNote || item.RemoteNote == nil {
		return nil
	}
	return item.RemoteNote
}

func remoteOnlySyncVisualKey(noteID string) string {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return ""
	}
	return "remote:" + noteID
}

func shortRemoteNoteID(noteID string) string {
	noteID = strings.TrimSpace(noteID)
	if len(noteID) <= 6 {
		return noteID
	}
	return noteID[:6]
}

func (m Model) hasRemoteOnlyPathDuplicate(relPath string) bool {
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	if relPath == "" {
		return false
	}
	count := 0
	for _, item := range m.remoteOnlyNotes {
		if filepath.ToSlash(strings.TrimSpace(item.RelPath)) != relPath {
			continue
		}
		count++
		if count > 1 {
			return true
		}
	}
	return false
}

func (m Model) remoteOnlyDisplayTitle(meta notesync.RemoteNoteMeta) string {
	title := remoteOnlyNoteTitle(meta)
	if !m.hasRemoteOnlyPathDuplicate(meta.RelPath) {
		return title
	}
	return title + " [" + shortRemoteNoteID(meta.ID) + "]"
}

func (m *Model) blockRemoteOnlyAction() bool {
	remote := m.currentRemoteOnlyNote()
	if remote == nil {
		return false
	}
	m.status = "note is only on the server; press i to import it or I to import all"
	return true
}

func remoteOnlyNoteTitle(meta notesync.RemoteNoteMeta) string {
	if strings.TrimSpace(meta.Title) != "" {
		return meta.Title
	}
	return filepath.Base(meta.RelPath)
}

func batchCmds(cmds ...tea.Cmd) tea.Cmd {
	var out []tea.Cmd
	for _, cmd := range cmds {
		if cmd != nil {
			out = append(out, cmd)
		}
	}
	if len(out) == 0 {
		return nil
	}
	if len(out) == 1 {
		return out[0]
	}
	return tea.Batch(out...)
}

func (m *Model) scheduleSync() tea.Cmd {
	if !notesync.HasSyncProfile(m.cfg.Sync) {
		return nil
	}
	m.syncDebounceToken++
	return syncDebounceCmd(m.syncDebounceToken, m.sessionToken)
}

func (m Model) localPinnedNoteKeys() []string {
	out := make([]string, 0, len(m.pinnedNotes))
	for p := range m.pinnedNotes {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func (m Model) localPinnedCategories() []string {
	out := make([]string, 0, len(m.pinnedCats))
	for p := range m.pinnedCats {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func (m *Model) applySyncedPins(noteRelPaths, cats []string) {
	syncedSet := make(map[string]bool, len(m.notes))
	for _, note := range m.notes {
		if note.SyncClass == notes.SyncClassSynced || note.SyncClass == notes.SyncClassShared {
			syncedSet[note.RelPath] = true
		}
	}
	newPinned := make(map[string]bool, len(m.pinnedNotes)+len(noteRelPaths))
	for k, v := range m.pinnedNotes {
		if strings.HasPrefix(k, ".tmp/") || !syncedSet[k] {
			newPinned[k] = v
		}
	}
	for _, relPath := range noteRelPaths {
		newPinned[relPath] = true
	}
	m.pinnedNotes = newPinned
	if cats != nil {
		m.pinnedCats = make(map[string]bool, len(cats))
		for _, relPath := range cats {
			m.pinnedCats[relPath] = true
		}
	}
	m.syncStateFromPins()
	m.rebuildTree()
	m.syncSelectedNote()
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
	if m.editorActive && m.editorModel != nil && !m.showCommandPalette && !m.showEditorURLPrompt {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			updated, cmd := m.editorModel.Update(msg)
			m.editorModel = &updated
			return m, cmd
		case tea.MouseMsg:
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case editorLoadedMsg:
		if msg.err != nil {
			m.status = "editor open failed: " + msg.err.Error()
			return m, nil
		}
		m.editorFullscreen = m.cfg.Editor.Fullscreen
		editorWidth, editorHeight := m.inAppEditorSize()
		editorModel := NewEditorModel(
			msg.path,
			msg.relPath,
			m.rootDir,
			msg.content,
			editorWidth,
			editorHeight,
			msg.encrypted,
			m.sessionPassphrase,
			msg.isTemp,
		)
		editorModel.markLoaded(msg.hash, msg.modTime)
		editorModel.SetLineNumbers(m.previewLineNumbersEnabled)
		m.editorModel = &editorModel
		m.editorActive = true
		m.showEditorURLPrompt = false
		m.editorLinkPickerMode = false
		if m.inAppEditorUsesPreview() {
			m.focus = focusPreview
		}
		m.status = "editing in app"
		return m, nil

	case editorSavedMsg:
		if m.editorModel == nil {
			return m, nil
		}
		if msg.discarded {
			m.pendingSelectRelPath = ""
			m.pendingSelectIsTemp = false
			m.closeInAppEditor("note discarded")
			return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())
		}
		m.editorModel.markSaved(msg.newPath, msg.hash, msg.modTime)
		m.editorModel.setStatus(editorSavedLabel(msg.newPath))
		m.pendingSelectRelPath = editorRelativePath(m.rootDir, msg.newPath, m.editorModel.isTemp)
		m.pendingSelectIsTemp = m.editorModel.isTemp
		cmd := batchCmds(saveNoteVersionCmd(m.rootDir, msg.newPath), refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())
		if msg.closeAfter {
			m.closeInAppEditor(editorSavedLabel(msg.newPath))
		}
		return m, cmd

	case editorSaveErrMsg:
		if m.editorModel != nil {
			m.editorModel.setStatus("save failed: " + msg.err.Error())
			return m, nil
		}
		m.status = "save failed: " + msg.err.Error()
		return m, nil

	case editorConflictMsg:
		if m.editorModel != nil {
			m.editorModel.setStatus("file changed on disk; use :w! or :e!")
		}
		return m, nil

	case editorReloadedMsg:
		if m.editorModel == nil {
			return m, nil
		}
		m.editorModel.applyReload(msg.content, msg.hash, msg.modTime)
		return m, nil

	case editorReloadErrMsg:
		if m.editorModel != nil {
			m.editorModel.setStatus("reload failed: " + msg.err.Error())
			return m, nil
		}
		m.status = "reload failed: " + msg.err.Error()
		return m, nil

	case editorClosedMsg:
		status := "editor closed"
		if msg.discarded {
			status = "discarded editor changes"
		}
		m.closeInAppEditor(status)
		return m, nil

	case editorLinkPickerMsg:
		m.openEditorLinkPicker()
		return m, nil

	case editorURLPromptMsg:
		m.openEditorURLPrompt()
		return m, nil

	case editorToggleFullscreenMsg:
		if m.editorModel != nil {
			m.editorFullscreen = !m.editorFullscreen
			w, h := m.inAppEditorSize()
			m.editorModel.Resize(w, h)
			if m.inAppEditorUsesPreview() {
				m.focus = focusPreview
			} else {
				m.focus = focusTree
			}
		}
		return m, nil

	case previewRenderedMsg:
		if msg.forPath != m.previewPath {
			return m, nil
		}
		m.previewBaseContent = msg.baseContent
		m.previewRawContent = msg.rawContent
		m.previewPrivacyForcedByNote = msg.privacyForcedByNote
		m.previewLineNumberStart = msg.lineNumberStart
		query := strings.TrimSpace(m.searchInput.Value())
		m.previewMatches = buildPreviewMatches(msg.baseContent, query)
		m.previewMatchIndex = 0
		highlighted := applyMatchHighlights(msg.baseContent, query, m.previewMatches, 0)
		m.previewContent = highlighted
		m.rebuildPreviewHeadingsFromRendered()
		m.previewWikilinks = notes.ExtractWikilinks(msg.rawContent)
		m.rebuildPreviewTodos(msg.rawContent, msg.baseContent, msg.todoLineOffset)
		m.rebuildPreviewLinks()
		if m.listMode == listModeTodos {
			m.syncSelectedTodoInPreview()
		} else if m.pendingTodoCursor >= 0 {
			m.previewTodoCursor = m.pendingTodoCursor
			m.pendingTodoCursor = -1
			m.previewTodoNavMode = m.previewTodoCursor >= 0
		} else if !m.previewTodoNavMode {
			m.previewTodoCursor = -1
		} else {
			m.previewTodoCursor = 0
		}
		if m.previewTodoNavMode {
			m.reapplyTodoHighlight()
		} else if m.previewLinkNavMode {
			m.reapplyLinkHighlight()
		} else {
			m.setPreviewViewportContent(applyTodoDueDateHints(m.previewContent))
		}
		if m.pendingPreviewYOffset >= 0 {
			m.preview.SetYOffset(m.pendingPreviewYOffset)
			m.pendingPreviewYOffset = -1
		} else if m.listMode == listModeTodos && m.previewTodoNavMode && m.previewTodoCursor >= 0 && m.previewTodoCursor < len(m.previewTodos) {
			m.ensurePreviewLineVisible(m.previewTodos[m.previewTodoCursor].rendLine)
		} else if len(m.previewMatches) > 0 && query != "" {
			m.scrollToMatchLine(m.previewMatches[0].line)
		} else {
			m.preview.GotoTop()
		}
		return m, nil

	case syncStartMsg:
		if msg.sessionToken != m.sessionToken {
			return m, nil
		}
		return m, m.startSyncRun()

	case syncDebouncedMsg:
		if msg.sessionToken != m.sessionToken || msg.token != m.syncDebounceToken || m.syncRunning {
			return m, nil
		}
		return m, m.startSyncRun()

	case syncSpinnerTickMsg:
		if !m.syncRunning && len(m.syncInFlight) == 0 {
			return m, nil
		}
		m.syncSpinnerFrame++
		if !m.syncRunning && len(m.syncInFlight) == 0 {
			return m, nil
		}
		return m, syncSpinnerCmd()

	case syncEventsLoadedMsg:
		m.syncTimelineEvents = msg.events
		return m, nil

	case openURLMsg:
		if msg.err != nil {
			m.status = "failed to open URL: " + msg.err.Error()
		} else {
			m.status = "opened: " + msg.url
		}
		return m, nil

	case syncUnlinkLocalMsg:
		if msg.err != nil {
			m.status = "unlink failed: " + msg.err.Error()
			return m, nil
		}
		m.status = "note unlinked (kept local)"
		return m, refreshAllCmd(m.rootDir, m.sessionToken)

	case syncFinishedMsg:
		if msg.sessionToken != m.sessionToken {
			return m, nil
		}
		m.startupSyncChecked = true
		m.finishSyncRun()
		if msg.err != nil {
			if notesync.HasSyncProfile(m.cfg.Sync) {
				m.status = "sync failed: " + msg.err.Error()
			}
			return m, loadSyncEventsCmd(m.rootDir)
		}
		placeholdersChanged := m.setRemoteOnlyNotes(msg.result.RemoteOnlyNotes)
		if msg.result.PinnedNoteRelPaths != nil || msg.result.PinnedCategories != nil {
			m.applySyncedPins(msg.result.PinnedNoteRelPaths, msg.result.PinnedCategories)
		}
		if msg.result.NotesChanged {
			m.deferredStatus = "sync updated local notes"
			return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), loadSyncEventsCmd(m.rootDir))
		}
		if msg.result.PinsChanged {
			m.status = "sync updated pins"
			return m, loadSyncEventsCmd(m.rootDir)
		}
		if msg.result.RegisteredNotes > 0 || msg.result.UpdatedNotes > 0 || msg.result.Conflicts > 0 {
			m.status = fmt.Sprintf("sync complete: %d registered, %d updated, %d conflicts", msg.result.RegisteredNotes, msg.result.UpdatedNotes, msg.result.Conflicts)
			if placeholdersChanged {
				m.rebuildTree()
			}
			return m, loadSyncEventsCmd(m.rootDir)
		}
		if placeholdersChanged {
			m.rebuildTree()
		}
		return m, loadSyncEventsCmd(m.rootDir)

	case syncProfileSavedMsg:
		if msg.err != nil {
			if msg.rebound {
				m.status = "sync root rebind failed: " + msg.err.Error()
			} else {
				m.status = "sync profile save failed: " + msg.err.Error()
			}
			return m, nil
		}
		m.cfg = msg.cfg
		ApplyConfigKeys(m.cfg.Keys)
		status := "default sync profile set to " + msg.profile
		if strings.TrimSpace(msg.showInfo) != "" {
			status = fmt.Sprintf("default sync profile set to %s; current root stays on %s", msg.profile, msg.showInfo)
		}
		if msg.rebound {
			status = "default sync profile set to " + msg.profile + "; current root rebound"
		}
		m.status = status
		return m, m.scheduleSync()

	case syncImportFinishedMsg:
		if msg.sessionToken != m.sessionToken {
			return m, nil
		}
		if m.syncRunning || len(m.syncInFlight) > 0 {
			m.finishSyncRun()
		}
		if msg.err != nil {
			m.status = "sync import failed: " + msg.err.Error()
			return m, nil
		}
		placeholdersChanged := m.setRemoteOnlyNotes(msg.result.RemoteOnlyNotes)
		status := "sync import complete: remote is empty"
		switch {
		case msg.result.ImportedNotes > 0 && msg.result.SkippedImports > 0:
			status = fmt.Sprintf("sync import complete: %d notes, %d skipped", msg.result.ImportedNotes, msg.result.SkippedImports)
		case msg.result.ImportedNotes > 0:
			status = fmt.Sprintf("sync import complete: %d notes", msg.result.ImportedNotes)
		case msg.result.SkippedImports > 0:
			status = fmt.Sprintf("sync import: %d skipped (note already exists locally, run sync to reconcile)", msg.result.SkippedImports)
		case msg.result.PinsChanged:
			status = "sync import updated pins"
		}
		if msg.result.ImportedNotes > 0 || msg.result.PinsChanged {
			m.pendingSyncedPinRelPaths = msg.result.PinnedNoteRelPaths
			m.pendingSyncedCategories = msg.result.PinnedCategories
			m.applyPendingSyncedPins = true
			m.deferredStatus = status
			return m, refreshAllCmd(m.rootDir, m.sessionToken)
		}
		m.status = status
		if placeholdersChanged {
			m.rebuildTree()
		}
		return m, nil

	case todoModifiedMsg:

		if msg.err != nil {
			m.status = "todo error: " + msg.err.Error()
			return m, nil
		}
		m.pendingTodoCursor = m.previewTodoCursor
		m.pendingPreviewYOffset = m.preview.YOffset
		m.previewPath = ""
		m.status = "todo updated"
		return m, batchCmds(saveNoteVersionCmd(m.rootDir, msg.path), refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case helpMouseResumeMsg:
		if !m.helpMouseSuppressed {
			return m, nil
		}
		m.helpMouseSuppressed = false
		return m, enableMouseCellMotionCmd()

	case tea.MouseMsg:
		m.updatePreviewMouseBounds()

		if m.showHelp {
			maxRows := max(8, min(20, m.height-16))
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if m.moveHelpScroll(-3, maxRows) {
					m.rebuildHelpModalCache(maxRows)
					return m, nil
				}
				if !m.helpMouseSuppressed {
					m.helpMouseSuppressed = true
					return m, tea.Batch(disableMouseCmd(), helpMouseResumeCmd())
				}
				return m, nil
			case tea.MouseButtonWheelDown:
				if m.moveHelpScroll(3, maxRows) {
					m.rebuildHelpModalCache(maxRows)
					return m, nil
				}
				if !m.helpMouseSuppressed {
					m.helpMouseSuppressed = true
					return m, tea.Batch(disableMouseCmd(), helpMouseResumeCmd())
				}
				return m, nil
			default:
				return m, nil
			}
		}

		m.previewHover = m.mouseInPreview(msg.X, msg.Y)

		if m.previewHover || m.focus == focusPreview {
			step := m.previewMouseScrollStep()
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.preview.ScrollUp(step)
				return m, nil
			case tea.MouseButtonWheelDown:
				m.preview.ScrollDown(step)
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.helpRowsCache = nil // invalidate; rebuildHelpRowsCache will refill on next open/type
		m.invalidateHelpBodyCache()
		m.rebuildHelpModalCache(max(8, min(20, m.height-16)))

		leftWidth, rightWidth := m.panelWidths()

		m.previewWidth = rightWidth
		m.searchInput.Width = max(16, leftWidth-8)
		m.commandPaletteInput.Width = max(40, min(86, m.width-8))
		m.categoryInput.Width = max(24, min(50, m.width-16))
		m.moveInput.Width = max(24, min(60, m.width-16))
		m.renameInput.Width = max(24, min(60, m.width-16))
		m.tagInput.Width = max(24, min(60, m.width-16))
		m.todoInput.Width = max(24, min(60, m.width-16))
		m.dueDateInput.Width = max(18, min(24, m.width-20))
		m.priorityInput.Width = max(12, min(20, m.width-20))

		previewInnerWidth := max(20, rightWidth-8)
		previewInnerHeight := max(6, msg.Height-14)
		m.preview.Width = previewInnerWidth
		m.preview.Height = previewInnerHeight
		if m.editorModel != nil {
			editorWidth, editorHeight := m.inAppEditorSize()
			m.editorModel.Resize(editorWidth, editorHeight)
		}
		m.updatePreviewMouseBounds()
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
		m.rewritePinnedNotePath(msg.oldRelPath, msg.newRelPath)
		_ = m.saveTreeState()
		m.preserveCursor = m.treeCursor
		m.status = "moved note: " + msg.newRelPath
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

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
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case remoteNoteDeletedMsg:
		if msg.sessionToken != m.sessionToken {
			return m, nil
		}
		if m.syncRunning || len(m.syncInFlight) > 0 {
			m.finishSyncRun()
		}
		if msg.err != nil {
			m.status = "remote delete failed: " + msg.err.Error()
			return m, nil
		}
		m.previewPath = ""
		m.deferredStatus = "deleted remote copy; note kept locally"
		return m, refreshAllCmd(m.rootDir, m.sessionToken)

	case conflictResolvedMsg:
		if msg.err != nil {
			m.status = "conflict resolution failed: " + msg.err.Error()
			return m, nil
		}
		m.closeSyncDebugModal("")
		m.previewPath = ""
		if msg.keepRemote {
			m.deferredStatus = "conflict resolved: kept remote version"
		} else {
			m.deferredStatus = "conflict resolved: kept local version"
		}
		return m, refreshAllCmd(m.rootDir, m.sessionToken)

	case noteVersionSavedMsg:
		// History saves are fire-and-forget; no status update needed.
		return m, nil

	case noteHistoryLoadedMsg:
		if msg.err != nil {
			m.status = "history load failed: " + msg.err.Error()
			return m, nil
		}
		if len(msg.entries) == 0 {
			m.status = "no history for this note yet"
			return m, nil
		}
		m.noteHistoryRelPath = msg.relPath
		m.noteHistoryAbsPath = filepath.Join(m.rootDir, filepath.FromSlash(msg.relPath))
		m.noteHistoryEntries = msg.entries
		m.noteHistoryCursor = 0
		m.showNoteHistory = true
		m.status = "note history"
		return m, nil

	case noteVersionRestoredMsg:
		if msg.err != nil {
			m.status = "restore failed: " + msg.err.Error()
			return m, nil
		}
		m.showNoteHistory = false
		m.noteHistoryEntries = nil
		m.noteHistoryRelPath = ""
		m.noteHistoryAbsPath = ""
		m.previewPath = ""
		m.deferredStatus = "version restored"
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case trashBrowserLoadedMsg:
		if msg.err != nil {
			m.status = "trash load failed: " + msg.err.Error()
			return m, nil
		}
		if len(msg.items) == 0 {
			m.status = "trash is empty (no items from this workspace)"
			return m, nil
		}
		m.trashBrowserItems = msg.items
		m.trashBrowserCursor = 0
		m.showTrashBrowser = true
		m.status = "trash browser"
		return m, nil

	case trashRestoreMsg:
		if msg.err != nil {
			m.status = "restore failed: " + msg.err.Error()
			return m, nil
		}
		m.showTrashBrowser = false
		m.trashBrowserItems = nil
		m.trashBrowserCursor = 0
		m.status = "restored: " + filepath.Base(msg.item.OriginalPath)
		return m, refreshAllCmd(m.rootDir, m.sessionToken)

	case noteSyncClassToggledMsg:
		if msg.err != nil {
			m.status = "toggle sync failed: " + msg.err.Error()
			return m, nil
		}
		m.previewPath = ""
		if msg.syncClass == notes.SyncClassLocal {
			m.status = "note unsynced locally; remote copy still exists. Press U to delete it from remote"
		} else {
			m.status = "note sync: " + msg.syncClass
		}
		if msg.syncClass == notes.SyncClassSynced {
			relPath, err := filepath.Rel(m.rootDir, msg.path)
			if err == nil {
				return m, batchCmds(
					refreshAllCmd(m.rootDir, m.sessionToken),
					m.startSyncVisual(filepath.ToSlash(relPath)),
					m.scheduleSync(),
				)
			}
		}
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case noteMadeSharedMsg:
		if msg.err != nil {
			m.status = "toggle shared failed: " + msg.err.Error()
			return m, nil
		}
		m.previewPath = ""
		if msg.syncClass == notes.SyncClassShared {
			m.status = "note is now shared"
		} else {
			m.status = "note is no longer shared"
		}
		relPath, err := filepath.Rel(m.rootDir, msg.path)
		if err == nil {
			return m, batchCmds(
				refreshAllCmd(m.rootDir, m.sessionToken),
				m.startSyncVisual(filepath.ToSlash(relPath)),
				m.scheduleSync(),
			)
		}
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case noteTaggedMsg:

		if msg.err != nil {
			m.status = "add tag failed: " + msg.err.Error()
			return m, nil
		}
		m.showAddTag = false
		m.tagInput.Blur()
		m.tagInput.SetValue("")
		m.previewPath = ""
		m.preserveCursor = m.treeCursor
		if len(msg.tags) == 1 {
			m.status = "added tag: " + msg.tags[0]
		} else {
			m.status = fmt.Sprintf("added %d tags", len(msg.tags))
		}
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case notesTaggedMsg:
		if msg.err != nil {
			m.status = "add tag failed: " + msg.err.Error()
			return m, nil
		}
		m.showAddTag = false
		m.tagInput.Blur()
		m.tagInput.SetValue("")
		m.previewPath = ""
		m.preserveCursor = m.treeCursor
		if len(msg.paths) == 1 && len(msg.tags) == 1 {
			m.status = "added tag: " + msg.tags[0]
		} else if len(msg.tags) == 1 {
			m.status = fmt.Sprintf("added %q to %d notes", msg.tags[0], len(msg.paths))
		} else {
			m.status = fmt.Sprintf("added %d tags to %d notes", len(msg.tags), len(msg.paths))
		}
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

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
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

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
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case batchMovedMsg:
		if msg.err != nil {
			m.status = "move failed: " + msg.err.Error()
			return m, nil
		}
		m.closeMoveBrowser("")
		m.preserveCursor = m.treeCursor
		for _, item := range msg.items {
			switch item.kind {
			case moveTargetCategory:
				m.rewriteCategoryStateSubtree(item.oldRelPath, item.newRelPath)
			case moveTargetNote:
				m.rewritePinnedNotePath(item.oldRelPath, item.newRelPath)
			}
		}
		m.clearMarkedTreeItems()
		_ = m.saveTreeState()
		if len(msg.items) == 1 {
			m.status = "moved 1 item"
		} else {
			m.status = fmt.Sprintf("moved %d items", len(msg.items))
		}
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case notesRelocatedMsg:
		if msg.err != nil {
			m.status = "move failed: " + msg.err.Error()
			return m, nil
		}
		m.closeMoveBrowser("")
		m.preserveCursor = m.treeCursor
		for _, item := range msg.items {
			m.rewritePinnedNotePath(item.oldPinPath, item.newPinPath)
		}
		m.clearMarkedTreeItems()
		_ = m.saveTreeState()
		m.status = msg.status
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case noteDeletedMsg:
		if msg.err != nil {
			m.status = "delete failed: " + msg.err.Error()
			return m, nil
		}
		m.deletePending = nil
		m.removePinnedForAbsolutePath(msg.path)
		m.preserveCursor = m.treeCursor
		label := filepath.Base(msg.path)
		m.lastDeletion = &undoableDelete{label: label, results: []notes.TrashResult{msg.result}}
		m.status = "trashed note: " + label + "  •  Z to undo"
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case notesDeletedMsg:
		if msg.err != nil {
			m.status = "delete failed: " + msg.err.Error()
			return m, nil
		}
		m.deletePending = nil
		m.removePinnedForPaths(msg.paths)
		m.clearMarkedTreeItems()
		m.preserveCursor = m.treeCursor
		label := countStatus(len(msg.paths), "1 note", "%d notes")
		m.lastDeletion = &undoableDelete{label: label, results: msg.results}
		m.status = "trashed " + label + "  •  Z to undo"
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case categoryDeletedMsg:
		if msg.err != nil {
			m.status = "delete failed: " + msg.err.Error()
			return m, nil
		}
		m.deletePending = nil
		m.preserveCursor = m.treeCursor
		m.removeCategoryStateSubtree(msg.relPath)
		_ = m.saveTreeState()
		m.lastDeletion = &undoableDelete{label: msg.relPath + "/", results: []notes.TrashResult{msg.result}}
		m.status = "trashed category: " + msg.relPath + "  •  Z to undo"
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case restoreFinishedMsg:
		if msg.err != nil {
			m.lastDeletion = nil
			m.status = "restore failed: " + msg.err.Error()
			return m, nil
		}
		m.lastDeletion = nil
		m.status = "restored: " + msg.label
		return m, refreshAllCmd(m.rootDir, m.sessionToken)

	case dataLoadedMsg:
		if msg.sessionToken != m.sessionToken {
			return m, nil
		}
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
			return m, nil
		}

		m.notes = msg.notes
		m.tempNotes = msg.tempNotes
		m.categories = msg.categories
		m.refreshSyncRecords()
		if m.syncRunning {
			m.syncInFlight = m.pendingSyncRelPaths()
		}

		m.pruneCategoryStateToExisting()
		m.pruneMarkedTreeItems()
		_ = m.saveTreeState()

		for _, c := range m.categories {
			relPath := normalizeCategoryRelPath(c.RelPath)
			if relPath == "" {
				continue
			}
			if _, ok := m.expanded[relPath]; !ok {
				m.expanded[relPath] = true
			}
		}

		m.rebuildTree()
		m.rebuildTodoItems()
		if m.applyPendingSyncedPins {
			m.applySyncedPins(m.pendingSyncedPinRelPaths, m.pendingSyncedCategories)
			m.pendingSyncedPinRelPaths = nil
			m.pendingSyncedCategories = nil
			m.applyPendingSyncedPins = false
		}

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

		m.clampTodoCursor()
		m.previewPath = ""
		if strings.TrimSpace(m.pendingSelectRelPath) != "" {
			if m.pendingSelectIsTemp {
				m.switchToTemporaryMode()
				m.selectTemporaryNote(m.pendingSelectRelPath)
			} else {
				m.switchToNotesMode()
				m.selectTreeNote(m.pendingSelectRelPath)
			}
			m.pendingSelectRelPath = ""
			m.pendingSelectIsTemp = false
		} else {
			m.syncSelectedNote()
		}

		if m.deferredStatus != "" {
			m.status = m.deferredStatus
			m.deferredStatus = ""
		} else if len(msg.notes) > 0 {
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
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), editor.Open(msg.path), m.scheduleSync())

	case categoryCreatedMsg:
		if msg.err != nil {
			m.status = "category create failed: " + msg.err.Error()
			return m, nil
		}
		m.showCreateCategory = false
		m.categoryInput.Blur()
		m.categoryInput.SetValue("")
		m.status = "created category: " + msg.relPath
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case previewLockedMsg:
		if msg.path != m.previewPath {
			return m, nil
		}
		locked := m.lockedPreviewText()
		m.previewBaseContent = ""
		m.previewContent = locked
		m.previewLineNumberStart = 0
		m.previewTodos = nil
		m.setPreviewViewportContent(locked)
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
		return m, batchCmds(saveNoteVersionCmd(m.rootDir, msg.path), refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case decryptNoteMsg:
		if msg.err != nil {
			m.status = "decryption failed: " + msg.err.Error()
			return m, nil
		}
		m.status = "note decrypted"
		m.previewPath = ""
		return m, batchCmds(saveNoteVersionCmd(m.rootDir, msg.path), refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

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
			return m, refreshAllCmd(m.rootDir, m.sessionToken)
		}
		m.status = "note saved and re-encrypted"
		return m, batchCmds(saveNoteVersionCmd(m.rootDir, msg.newPath), refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

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

		if m.dailyNoteOpen {
			m.dailyNoteOpen = false
			m.status = "editor closed"
			return m, batchCmds(saveNoteVersionCmd(m.rootDir, msg.Path), refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())
		}

		newPath, renamed, err := notes.RenameFromTitle(msg.Path)
		if err != nil {
			m.status = "rename failed: " + err.Error()
			return m, refreshAllCmd(m.rootDir, m.sessionToken)
		}

		// Empty file with temp name was deleted.
		if newPath == "" && !renamed {
			m.status = "note discarded"
			return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())
		}

		if renamed {
			m.status = "renamed: " + filepath.Base(newPath)
		} else {
			m.status = "editor closed"
		}

		return m, batchCmds(saveNoteVersionCmd(m.rootDir, newPath), refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case watchStartedMsg:
		if msg.sessionToken != m.sessionToken {
			if msg.watcher != nil {
				_ = msg.watcher.Close()
			}
			return m, nil
		}
		if msg.err != nil {
			m.status = "watch disabled: " + msg.err.Error()
			return m, nil
		}
		m.watcher = msg.watcher
		m.watchEvents = msg.events
		m.status = "ready"
		return m, waitForWatchTeaCmd(m.watchEvents)

	case watchRefreshMsg:
		if msg.sessionToken != m.sessionToken {
			return m, nil
		}
		m.status = "auto refresh"
		m.pendingTodoCursor = m.previewTodoCursor
		m.previewPath = ""
		if m.watchEvents != nil {
			return m, batchCmds(
				refreshAllCmd(m.rootDir, m.sessionToken),
				waitForWatchTeaCmd(m.watchEvents),
				m.scheduleSync(),
			)
		}
		return m, batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())

	case watchErrorMsg:
		if msg.sessionToken != 0 && msg.sessionToken != m.sessionToken {
			return m, nil
		}
		if msg.err != nil {
			m.status = "watch error: " + msg.err.Error()
		}
		if m.watchEvents != nil {
			return m, waitForWatchTeaCmd(m.watchEvents)
		}
		return m, nil

	case tea.KeyMsg:
		if m.showEditorURLPrompt {
			switch msg.String() {
			case "esc":
				m.showEditorURLPrompt = false
				m.editorURLInput.Blur()
				m.editorURLInput.SetValue("")
				if m.editorModel != nil {
					m.editorModel.setStatus("URL insert cancelled")
				} else {
					m.status = "URL insert cancelled"
				}
				return m, nil
			case "enter":
				url := strings.TrimSpace(m.editorURLInput.Value())
				m.showEditorURLPrompt = false
				m.editorURLInput.Blur()
				m.editorURLInput.SetValue("")
				if url == "" {
					if m.editorModel != nil {
						m.editorModel.setStatus("URL cannot be empty")
					} else {
						m.status = "URL cannot be empty"
					}
					return m, nil
				}
				if m.editorModel != nil {
					m.editorModel.InsertURLLink(url)
				}
				return m, nil
			}
			if isMouseEscapeFragment(msg) {
				return m, nil
			}
			var cmd tea.Cmd
			m.editorURLInput, cmd = m.editorURLInput.Update(msg)
			return m, cmd
		}

		if m.showTemplatePicker {
			switch {
			case msg.String() == "esc":
				m.closeTemplatePicker("template selection cancelled")
				return m, nil
			case msg.String() == "enter":
				return m, m.confirmTemplatePicker()
			case msg.String() == "e" && !m.templatePickerEditMode && m.templatePickerCursor > 0:
				return m, m.editTemplateAtCursor()
			case key.Matches(msg, keys.MoveUp):
				m.moveTemplateCursor(-1)
				return m, nil
			case key.Matches(msg, keys.MoveDown):
				m.moveTemplateCursor(1)
				return m, nil
			}
			return m, nil
		}

		if m.showNoteHistory {
			switch {
			case msg.String() == "esc":
				m.showNoteHistory = false
				m.noteHistoryEntries = nil
				m.noteHistoryRelPath = ""
				m.noteHistoryAbsPath = ""
				m.status = "history closed"
				return m, nil
			case msg.String() == "enter":
				if len(m.noteHistoryEntries) == 0 {
					return m, nil
				}
				entry := m.noteHistoryEntries[m.noteHistoryCursor]
				return m, restoreNoteVersionCmd(m.rootDir, m.noteHistoryAbsPath, m.noteHistoryRelPath, entry.ID)
			case key.Matches(msg, keys.MoveUp):
				if m.noteHistoryCursor > 0 {
					m.noteHistoryCursor--
				}
				return m, nil
			case key.Matches(msg, keys.MoveDown):
				if m.noteHistoryCursor < len(m.noteHistoryEntries)-1 {
					m.noteHistoryCursor++
				}
				return m, nil
			}
			return m, nil
		}

		if m.showTrashBrowser {
			switch {
			case msg.String() == "esc":
				m.showTrashBrowser = false
				m.trashBrowserItems = nil
				m.trashBrowserCursor = 0
				m.status = "trash browser closed"
				return m, nil
			case msg.String() == "enter":
				if len(m.trashBrowserItems) == 0 {
					return m, nil
				}
				return m, restoreTrashItemCmd(m.trashBrowserItems[m.trashBrowserCursor])
			case key.Matches(msg, keys.MoveUp):
				if m.trashBrowserCursor > 0 {
					m.trashBrowserCursor--
				}
				return m, nil
			case key.Matches(msg, keys.MoveDown):
				if m.trashBrowserCursor < len(m.trashBrowserItems)-1 {
					m.trashBrowserCursor++
				}
				return m, nil
			}
			return m, nil
		}

		if m.showWorkspacePicker {
			switch {
			case msg.String() == "enter":
				return m, m.confirmSelectedWorkspace()
			case msg.String() == "esc":
				if strings.TrimSpace(m.rootDir) == "" {
					m.status = "select a workspace or press q to quit"
					return m, nil
				}
				m.closeWorkspacePicker("workspace switch cancelled")
				return m, nil
			case key.Matches(msg, keys.MoveUp):
				m.moveWorkspaceCursor(-1)
				return m, nil
			case key.Matches(msg, keys.MoveDown):
				m.moveWorkspaceCursor(1)
				return m, nil
			case strings.TrimSpace(m.rootDir) == "" && key.Matches(msg, keys.Quit):
				m.stopWorkspaceWatch()
				return m, tea.Quit
			}
			return m, nil
		}

		if m.showDashboard {
			switch {
			case msg.String() == "enter":
				m.showDashboard = false
				m.status = "workspace"
				return m, nil

			case key.Matches(msg, keys.ShowPins):
				m.showDashboard = false
				m.listMode = listModePins
				m.status = "pins"
				m.syncSelectedNote()
				return m, nil

			case key.Matches(msg, keys.ShowTodos):
				m.showDashboard = false
				m.toggleTodosMode()
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

			case key.Matches(msg, keys.CommandPalette):
				m.showDashboard = false
				m.openCommandPalette()
				m.status = "command palette"
				return m, nil

			case key.Matches(msg, keys.Quit):
				m.stopWorkspaceWatch()
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
			if isMouseEscapeFragment(msg) {
				return m, nil
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
				case "unlock", "unlock_edit", "unlock_in_app", "decrypt":
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
						return m, saveNoteVersionAndOpenEncryptedCmd(m.rootDir, m.pendingEncryptPath, m.sessionPassphrase)
					}
					if m.passphraseModalCtx == "unlock_in_app" {
						return m, saveNoteVersionAndEditorLoadCmd(
							m.rootDir,
							m.pendingInAppEditPath,
							m.pendingInAppEditRel,
							m.sessionPassphrase,
							m.pendingInAppEditTemp,
						)
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
			if isMouseEscapeFragment(msg) {
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

		if m.showTodoPriority {
			switch msg.String() {
			case "esc":
				m.showTodoPriority = false
				m.priorityInput.Blur()
				m.priorityInput.SetValue("")
				m.status = "todo priority cancelled"
				return m, nil
			case "enter":
				priority := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(m.priorityInput.Value()), "p"))
				if priority != "" {
					if _, ok := parseTodoPriorityHintToken("[p" + priority + "]"); !ok {
						m.status = "priority must be a positive number"
						return m, nil
					}
				}
				path, rawLine, _, ok := m.currentPreviewTodoSelection()
				m.showTodoPriority = false
				m.priorityInput.Blur()
				m.priorityInput.SetValue("")
				if !ok {
					m.status = "no todo selected"
					return m, nil
				}
				return m, updateTodoPriorityCmd(path, rawLine, priority)
			}
			if isMouseEscapeFragment(msg) {
				return m, nil
			}
			var cmd tea.Cmd
			m.priorityInput, cmd = m.priorityInput.Update(msg)
			return m, cmd
		}

		if m.showTodoDueDate {
			switch msg.String() {
			case "esc":
				m.showTodoDueDate = false
				m.dueDateInput.Blur()
				m.dueDateInput.SetValue("")
				m.status = "todo due date cancelled"
				return m, nil
			case "enter":
				dueDate := strings.TrimSpace(m.dueDateInput.Value())
				if dueDate != "" {
					if _, err := time.Parse("2006-01-02", dueDate); err != nil {
						m.status = "due date must use YYYY-MM-DD"
						return m, nil
					}
				}
				path, rawLine, _, ok := m.currentPreviewTodoSelection()
				m.showTodoDueDate = false
				m.dueDateInput.Blur()
				m.dueDateInput.SetValue("")
				if !ok {
					m.status = "no todo selected"
					return m, nil
				}
				return m, updateTodoDueDateCmd(path, rawLine, dueDate)
			}
			if isMouseEscapeFragment(msg) {
				return m, nil
			}
			var cmd tea.Cmd
			m.dueDateInput, cmd = m.dueDateInput.Update(msg)
			return m, cmd
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
			if isMouseEscapeFragment(msg) {
				return m, nil
			}
			var cmd tea.Cmd
			m.todoInput, cmd = m.todoInput.Update(msg)
			return m, cmd
		}

		if m.showSyncTimeline {
			switch {
			case msg.String() == "esc", msg.String() == "q":
				m.closeSyncTimeline()
				return m, nil
			case key.Matches(msg, keys.MoveUp):
				if m.syncTimelineOffset > 0 {
					m.syncTimelineOffset--
				}
				return m, nil
			case key.Matches(msg, keys.MoveDown):
				maxVisible := max(4, min(20, m.height-14))
				maxOffset := max(0, len(m.syncTimelineEvents)-maxVisible)
				if m.syncTimelineOffset < maxOffset {
					m.syncTimelineOffset++
				}
				return m, nil
			}
			return m, nil
		}

		if m.showSyncDebugModal {
			switch {
			case msg.String() == "esc":
				m.closeSyncDebugModal("sync details closed")
				return m, nil
			case m.hasConflictCopyForCurrentSelection() && (key.Matches(msg, keys.CollapseCategory) || msg.String() == "left"):
				m.conflictResolutionChoice = conflictResolutionKeepLocal
				return m, nil
			case m.hasConflictCopyForCurrentSelection() && (key.Matches(msg, keys.ExpandCategory) || msg.String() == "right"):
				m.conflictResolutionChoice = conflictResolutionKeepRemote
				return m, nil
			case m.hasConflictCopyForCurrentSelection() && msg.String() == "enter":
				return m, m.confirmCurrentConflictResolution()
			case msg.String() == "y":
				m.copyCurrentSyncDebugRawError()
				return m, nil
			case !m.hasConflictCopyForCurrentSelection() && msg.String() == "r":
				m.closeSyncDebugModal("")
				return m, m.startSyncRun()
			case !m.hasConflictCopyForCurrentSelection() && msg.String() == "u":
				return m, m.unlinkCurrentNoteLocally()
			}
			return m, nil
		}

		if m.showSyncProfilePicker {
			switch {
			case msg.String() == "esc":
				m.closeSyncProfilePicker("sync profile change cancelled")
				return m, nil
			case msg.String() == "enter":
				return m, m.confirmSelectedSyncProfile()
			case key.Matches(msg, keys.MoveUp):
				m.moveSyncProfileCursor(-1)
				return m, nil
			case key.Matches(msg, keys.MoveDown):
				m.moveSyncProfileCursor(1)
				return m, nil
			}
			return m, nil
		}

		if m.showSyncProfileMigration {
			switch {
			case msg.String() == "esc":
				m.closeSyncProfileMigration("sync profile change cancelled")
				return m, nil
			case msg.String() == "enter":
				return m, m.confirmSyncProfileMigration()
			case msg.String() == "left", msg.String() == "up":
				m.moveSyncProfileMigrationChoice(-1)
				return m, nil
			case msg.String() == "right", msg.String() == "down", msg.String() == "tab":
				m.moveSyncProfileMigrationChoice(1)
				return m, nil
			case key.Matches(msg, keys.MoveUp):
				m.moveSyncProfileMigrationChoice(-1)
				return m, nil
			case key.Matches(msg, keys.MoveDown):
				m.moveSyncProfileMigrationChoice(1)
				return m, nil
			}
			return m, nil
		}

		if m.showMoveBrowser {
			switch {
			case msg.String() == "esc":
				m.closeMoveBrowser(m.moveBrowserCancelStatus())
				return m, nil
			case msg.String() == "enter":
				return m, m.confirmMoveBrowser()
			case key.Matches(msg, keys.JumpBottom):
				m.moveBrowserError = ""
				m.jumpMoveDestinationBottom()
				m.pendingG = false
				return m, nil
			case key.Matches(msg, keys.PendingG):
				m.moveBrowserError = ""
				if m.pendingG {
					m.jumpMoveDestinationTop()
					m.pendingG = false
				} else {
					m.pendingG = true
				}
				return m, nil
			case key.Matches(msg, keys.ScrollHalfPageUp):
				m.moveBrowserError = ""
				m.moveDestinationCursor(-max(1, m.preview.Height/2))
				m.pendingG = false
				return m, nil
			case key.Matches(msg, keys.ScrollHalfPageDown):
				m.moveBrowserError = ""
				m.moveDestinationCursor(max(1, m.preview.Height/2))
				m.pendingG = false
				return m, nil
			case key.Matches(msg, keys.MoveUp):
				m.moveBrowserError = ""
				m.moveDestinationCursor(-1)
				m.pendingG = false
				return m, nil
			case key.Matches(msg, keys.MoveDown):
				m.moveBrowserError = ""
				m.moveDestinationCursor(1)
				m.pendingG = false
				return m, nil
			case key.Matches(msg, keys.ExpandCategory):
				m.moveBrowserError = ""
				m.expandMoveDestination()
				m.pendingG = false
				return m, nil
			case key.Matches(msg, keys.CollapseCategory):
				m.moveBrowserError = ""
				m.collapseMoveDestination()
				m.pendingG = false
				return m, nil
			}
			m.pendingG = false
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

			if isMouseEscapeFragment(msg) {
				return m, nil
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

			if isMouseEscapeFragment(msg) {
				return m, nil
			}
			var cmd tea.Cmd
			m.renameInput, cmd = m.renameInput.Update(msg)
			return m, cmd
		}

		if m.showAddTag {
			switch msg.String() {
			case "esc":
				m.showAddTag = false
				m.tagInput.Blur()
				m.tagInput.SetValue("")
				m.status = "add tag cancelled"
				return m, nil
			case "enter":
				value := strings.TrimSpace(m.tagInput.Value())
				if value == "" {
					m.showAddTag = false
					m.tagInput.Blur()
					m.tagInput.SetValue("")
					m.status = "add tag cancelled"
					return m, nil
				}
				paths, err := m.selectedTaggableNotePaths()
				if err != nil {
					m.status = err.Error()
					return m, nil
				}
				tags := parseTagInput(value)
				if len(tags) == 0 {
					m.showAddTag = false
					m.tagInput.Blur()
					m.tagInput.SetValue("")
					m.status = "add tag cancelled"
					return m, nil
				}
				if len(paths) == 1 {
					return m, addNoteTagsCmd(paths[0], tags)
				}
				return m, addNoteTagsBatchCmd(paths, tags)
			}

			if isMouseEscapeFragment(msg) {
				return m, nil
			}
			var cmd tea.Cmd
			m.tagInput, cmd = m.tagInput.Update(msg)
			return m, cmd
		}

		if m.deletePending != nil {
			switch {
			case msg.String() == "esc":
				m.deletePending = nil
				m.status = "delete cancelled"
				return m, nil
			case key.Matches(msg, keys.DeleteConfirm) || msg.String() == "d":
				return m, m.confirmDeleteCurrent()
			default:
				m.deletePending = nil
			}
		}

		if m.showHelp {
			maxRows := max(8, min(20, m.height-16))
			switch msg.String() {
			case "esc":
				m.showHelp = false
				m.helpInput.Blur()
				m.status = "help closed"
				if m.helpMouseSuppressed {
					m.helpMouseSuppressed = false
					return m, enableMouseCellMotionCmd()
				}
				return m, nil
			case "up":
				if m.moveHelpScroll(-1, maxRows) {
					m.rebuildHelpModalCache(maxRows)
				}
				return m, nil
			case "down":
				if m.moveHelpScroll(1, maxRows) {
					m.rebuildHelpModalCache(maxRows)
				}
				return m, nil
			case "home":
				m.helpScroll = 0
				m.clampHelpScroll(maxRows)
				m.rebuildHelpModalCache(maxRows)
				return m, nil
			case "end":
				m.helpScroll = 1 << 30
				m.clampHelpScroll(maxRows)
				m.rebuildHelpModalCache(maxRows)
				return m, nil
			}
			switch {
			case key.Matches(msg, keys.ScrollHalfPageUp):
				if m.moveHelpScroll(-max(1, maxRows/2), maxRows) {
					m.rebuildHelpModalCache(maxRows)
				}
				return m, nil
			case key.Matches(msg, keys.ScrollHalfPageDown):
				if m.moveHelpScroll(max(1, maxRows/2), maxRows) {
					m.rebuildHelpModalCache(maxRows)
				}
				return m, nil
			case key.Matches(msg, keys.ScrollPageUp):
				if m.moveHelpScroll(-maxRows, maxRows) {
					m.rebuildHelpModalCache(maxRows)
				}
				return m, nil
			case key.Matches(msg, keys.ScrollPageDown):
				if m.moveHelpScroll(maxRows, maxRows) {
					m.rebuildHelpModalCache(maxRows)
				}
				return m, nil
			default:
				if !shouldUpdateHelpInput(msg, m.helpInput) {
					return m, nil
				}
				var cmd tea.Cmd
				before := m.helpInput.Value()
				m.helpInput, cmd = m.helpInput.Update(msg)
				if m.helpInput.Value() != before {
					m.helpScroll = 0
					m.rebuildHelpRowsCache()
				}
				m.clampHelpScroll(maxRows)
				m.rebuildHelpModalCache(maxRows)
				return m, cmd
			}
		}

		if m.showThemePicker {
			switch {
			case msg.String() == "esc":
				m.cancelThemePicker()
			case msg.Type == tea.KeyEnter:
				m.confirmThemePicker()
			case key.Matches(msg, keys.MoveUp):
				m.moveThemePickerCursor(-1)
			case key.Matches(msg, keys.MoveDown):
				m.moveThemePickerCursor(1)
			}
			return m, nil
		}

		if m.showCommandPalette {
			var paletteCmd tea.Cmd
			switch msg.Type {
			case tea.KeyEsc:
				m.showCommandPalette = false
				m.commandPaletteInput.Blur()
				m.editorLinkPickerMode = false
			case tea.KeyEnter:
				if len(m.commandPaletteFiltered) > 0 {
					paletteCmd = m.commitPaletteSelection()
				}
			case tea.KeyTab:
				m.tabCompletePalette()
			case tea.KeyUp:
				if m.commandPaletteCursor > 0 {
					m.commandPaletteCursor--
				}
			case tea.KeyDown:
				if m.commandPaletteCursor < len(m.commandPaletteFiltered)-1 {
					m.commandPaletteCursor++
				}
			default:
				var cmd tea.Cmd
				m.commandPaletteInput, cmd = m.commandPaletteInput.Update(msg)
				m.rebuildPaletteFiltered()
				return m, cmd
			}
			return m, paletteCmd
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

			if isMouseEscapeFragment(msg) {
				return m, nil
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
				case listModeTodos:
					m.moveTodoCursor(-1)
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
				case listModeTodos:
					m.moveTodoCursor(1)
				default:
					m.moveTreeCursor(1)
				}
				return m, nil
			}

			if isMouseEscapeFragment(msg) {
				return m, nil
			}
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.previewPath = ""
			m.rebuildTree()
			m.clampTodoCursor()
			m.syncSelectedNote()
			return m, cmd
		}

		if key.Matches(msg, keys.ShowPins) {
			m.togglePinsMode()
			return m, nil
		}

		if key.Matches(msg, keys.ShowTodos) {
			m.toggleTodosMode()
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

		if msg.String() == "esc" && m.listMode == listModeTodos {
			if m.focus == focusPreview {
				m.focus = focusTree
				m.status = "list focused"
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
			m.status = "left todos"
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
			m.stopWorkspaceWatch()
			return m, tea.Quit
		}

		if m.pendingSort {
			m.pendingSort = false
			switch {
			case key.Matches(msg, keys.SortByName):
				m.applySortMethod(sortAlpha)
			case key.Matches(msg, keys.SortByModified):
				m.applySortMethod(sortModified)
			case key.Matches(msg, keys.SortByCreated):
				m.applySortMethod(sortCreated)
			case key.Matches(msg, keys.SortBySize), key.Matches(msg, keys.SortKey):
				m.applySortMethod(sortSize)
			case key.Matches(msg, keys.SortReverse):
				m.sortReverse = !m.sortReverse
				_ = m.saveTreeState()
				m.rebuildTree()
				dir := "descending"
				if m.sortReverse {
					dir = "ascending"
				}
				m.status = "sort order: " + dir
			case msg.String() == "esc":
				m.status = "sort cancelled"
			}
			return m, nil
		}

		if key.Matches(msg, keys.SortKey) {
			m.pendingSort = true
			m.status = "sort: [n]ame  [m]odified  [c]reated  [s]ize  [r]everse  esc=cancel"
			return m, nil
		}

		if key.Matches(msg, keys.ShowThemePicker) {
			m.openThemePicker()
			return m, nil
		}

		if key.Matches(msg, keys.ShowHelp) {
			m.openHelpModal()
			return m, nil
		}

		if key.Matches(msg, keys.CommandPalette) {
			m.openCommandPalette()
			m.status = "command palette"
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
			m.clampTodoCursor()
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

		if m.focus == focusPreview && !m.showHelp && !m.showCreateCategory && !m.showMoveBrowser &&
			!m.showMove && !m.showRename && !m.showAddTag {
			if !key.Matches(msg, keys.PendingZ) {
				m.pendingZ = false
			}
			if m.pendingT && !key.Matches(msg, keys.TodoKey) && !key.Matches(msg, keys.TodoAdd) &&
				!key.Matches(msg, keys.TodoDelete) &&
				!key.Matches(msg, keys.TodoEdit) && !key.Matches(msg, keys.TodoDueDate) && !key.Matches(msg, keys.TodoPriority) {
				m.pendingT = false
			}
			switch {
			case msg.String() == "esc" && m.previewLinkNavMode:
				m.previewLinkNavMode = false
				m.previewLinkCursor = -1
				m.pendingBracketDir = ""
				m.setPreviewViewportContent(applyTodoDueDateHints(m.previewContent))
				m.status = "link nav off"
				return m, nil

			case msg.String() == "esc" && m.previewTodoNavMode:
				m.previewTodoNavMode = false
				m.previewTodoCursor = -1
				m.pendingBracketDir = ""
				m.pendingT = false
				m.setPreviewViewportContent(applyTodoDueDateHints(m.previewContent))
				m.status = "todo nav off"
				return m, nil

			case msg.String() == "esc":
				m.focus = focusTree
				m.status = "tree focused"
				m.pendingG = false
				m.pendingBracketDir = ""
				m.pendingT = false
				m.previewTodoNavMode = false
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
				if m.previewLinkNavMode {
					m.jumpToLastLink()
				} else if m.previewTodoNavMode {
					m.jumpToLastTodo()
				} else {
					m.preview.GotoBottom()
					m.status = "preview bottom"
				}
				m.pendingG = false
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.PendingG):
				m.moveBrowserError = ""
				if m.pendingG {
					if m.previewLinkNavMode {
						m.jumpToFirstLink()
					} else if m.previewTodoNavMode {
						m.jumpToFirstTodo()
					} else {
						m.preview.GotoTop()
						m.status = "preview top"
					}
					m.pendingG = false
					return m, nil
				}
				m.pendingG = true
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.BracketForward):
				if m.previewLinkNavMode {
					m.jumpToNextLink()
					m.pendingBracketDir = ""
					return m, nil
				}
				if m.previewTodoNavMode {
					m.jumpToNextTodo()
					m.pendingBracketDir = ""
					return m, nil
				}
				m.pendingBracketDir = "]"
				m.pendingG = false
				return m, nil

			case key.Matches(msg, keys.BracketBackward):
				if m.previewLinkNavMode {
					m.jumpToPrevLink()
					m.pendingBracketDir = ""
					return m, nil
				}
				if m.previewTodoNavMode {
					m.jumpToPrevTodo()
					m.pendingBracketDir = ""
					return m, nil
				}
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
					m.previewTodoNavMode = true
					m.previewLinkNavMode = false
					m.previewLinkCursor = -1
					m.jumpToNextTodo()
					m.pendingBracketDir = ""
					m.pendingT = false
					m.status = "todo nav on"
					return m, nil
				}
				if m.pendingBracketDir == "[" {
					m.previewTodoNavMode = true
					m.previewLinkNavMode = false
					m.previewLinkCursor = -1
					m.jumpToPrevTodo()
					m.pendingBracketDir = ""
					m.pendingT = false
					m.status = "todo nav on"
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

			case key.Matches(msg, keys.TodoDueDate) && !key.Matches(msg, keys.TodoPriority) && m.pendingT:
				m.pendingT = false
				m.armSetCurrentTodoDueDate()
				return m, nil

			case key.Matches(msg, keys.TodoPriority) && m.pendingT:
				m.pendingT = false
				m.armSetCurrentTodoPriority()
				return m, nil

			case key.Matches(msg, keys.LinkKey):
				if m.pendingBracketDir == "]" {
					m.previewLinkNavMode = true
					m.previewTodoNavMode = false
					m.previewTodoCursor = -1
					m.jumpToNextLink()
					m.pendingBracketDir = ""
					return m, nil
				}
				if m.pendingBracketDir == "[" {
					m.previewLinkNavMode = true
					m.previewTodoNavMode = false
					m.previewTodoCursor = -1
					m.jumpToPrevLink()
					m.pendingBracketDir = ""
					return m, nil
				}
				if m.previewLinkNavMode {
					return m, m.followSelectedLink()
				}
				m.pendingBracketDir = ""
				return m, nil

			case key.Matches(msg, keys.FollowLink) && m.previewLinkNavMode:
				return m, m.followSelectedLink()

			case key.Matches(msg, keys.Open):
				if m.previewLinkNavMode {
					return m, m.followSelectedLink()
				}
				if !m.previewTodoNavMode {
					if cmd := m.followWikilinkUnderCursor(); cmd != nil {
						return m, cmd
					}
				}

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

		if m.focus == focusTree && m.listMode == listModeTodos {
			if m.pendingT && !key.Matches(msg, keys.TodoKey) && !key.Matches(msg, keys.TodoAdd) &&
				!key.Matches(msg, keys.TodoDelete) && !key.Matches(msg, keys.TodoEdit) &&
				!key.Matches(msg, keys.TodoDueDate) && !key.Matches(msg, keys.TodoPriority) {
				m.pendingT = false
			}
			switch {
			case key.Matches(msg, keys.TodoKey):
				if m.pendingT {
					m.pendingT = false
					return m, m.toggleCurrentPreviewTodo()
				}
				m.pendingT = true
				return m, nil
			case key.Matches(msg, keys.TodoAdd) && m.pendingT:
				m.pendingT = false
				m.armAddTodoItem()
				return m, nil
			case key.Matches(msg, keys.TodoDelete) && m.pendingT:
				m.pendingT = false
				return m, m.deleteCurrentPreviewTodo()
			case key.Matches(msg, keys.TodoEdit) && m.pendingT:
				m.pendingT = false
				return m, m.armEditCurrentPreviewTodo()
			case key.Matches(msg, keys.TodoDueDate) && !key.Matches(msg, keys.TodoPriority) && m.pendingT:
				m.pendingT = false
				m.armSetCurrentTodoDueDate()
				return m, nil

			case key.Matches(msg, keys.TodoPriority) && m.pendingT:
				m.pendingT = false
				m.armSetCurrentTodoPriority()
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
				case listModeTodos:
					items := m.filteredTodoItems()
					if len(items) > 0 {
						m.todoCursor = len(items) - 1
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
				m.moveBrowserError = ""
				if m.pendingG {
					m.pendingG = false
					switch m.listMode {
					case listModeTemporary:
						m.tempCursor = 0
						m.syncSelectedNote()
					case listModePins:
						m.pinsCursor = 0
						m.syncSelectedNote()
					case listModeTodos:
						m.todoCursor = 0
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
				case listModeTodos:
					m.moveTodoCursor(-half)
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
				case listModeTodos:
					m.moveTodoCursor(half)
				default:
					m.moveTreeCursor(half)
				}
				return m, nil
			}
		}

		m.pendingG = false
		m.pendingBracketDir = ""
		m.pendingT = false
		if m.focus != focusPreview {
			m.previewTodoNavMode = false
			m.previewLinkNavMode = false
			m.previewLinkCursor = -1
		}

		if key.Matches(msg, keys.Move) {
			m.armMoveCurrent()
			return m, nil
		}

		if key.Matches(msg, keys.ToggleSelect) {
			m.toggleMarkCurrent()
			return m, nil
		}

		if key.Matches(msg, keys.ClearMarks) {
			m.clearAllMarks()
			return m, nil
		}

		if key.Matches(msg, keys.Rename) {
			m.armRenameCurrent()
			return m, nil
		}

		if key.Matches(msg, keys.AddTag) {
			m.armAddTagCurrent()
			return m, nil
		}

		if key.Matches(msg, keys.Delete) {
			m.armDeleteCurrent()
			return m, nil
		}

		if key.Matches(msg, keys.Pin) {
			if err := m.togglePinCurrent(); err != nil {
				m.status = "pin failed: " + err.Error()
				return m, nil
			}
			return m, m.scheduleSync()
		}

		if key.Matches(msg, keys.SelectWorkspace) {
			m.openWorkspacePicker()
			return m, nil
		}

		if key.Matches(msg, keys.SelectSyncProfile) {
			m.openSyncProfilePicker()
			return m, nil
		}

		if key.Matches(msg, keys.ToggleSync) {
			return m, m.toggleNoteSyncCurrent()
		}

		if key.Matches(msg, keys.MakeShared) {
			return m, m.makeCurrentNoteShared()
		}

		if key.Matches(msg, keys.OpenConflictCopy) {
			return m, m.openCurrentConflictCopy()
		}

		if key.Matches(msg, keys.ShowSyncDebug) {
			m.openCurrentSyncDebugModal()
			return m, nil
		}

		if key.Matches(msg, keys.ShowSyncTimeline) {
			return m, m.openSyncTimeline()
		}

		if key.Matches(msg, keys.DeleteRemoteKeepLocal) {
			return m, m.deleteRemoteCopyCurrent()
		}

		if key.Matches(msg, keys.SyncImportCurrent) {
			return m, m.importCurrentRemoteNote()
		}

		if key.Matches(msg, keys.SyncImport) {
			return m, m.importAllRemoteNotes()
		}

		if key.Matches(msg, keys.UndoDelete) {
			if m.lastDeletion != nil {
				d := m.lastDeletion
				m.lastDeletion = nil
				return m, restoreFromTrashCmd(d.label, d.results)
			}
			return m, nil
		}

		if key.Matches(msg, keys.CreateCategory) {
			m.openCreateCategory()
			return m, nil
		}

		if key.Matches(msg, keys.Refresh) {
			return m, m.startRefresh()
		}

		if key.Matches(msg, keys.NewTemporaryNote) {
			return m, createTemporaryNoteCmd(m.rootDir)
		}

		if key.Matches(msg, keys.NewTodoList) {
			return m, m.startNewTodoList()
		}

		if key.Matches(msg, keys.NewNote) {
			return m, m.startNewNote()
		}

		if key.Matches(msg, keys.OpenDailyNote) {
			return m, m.openDailyNote()
		}

		if key.Matches(msg, keys.EditInApp) {
			return m, m.openInAppEditorCurrent()
		}

		if key.Matches(msg, keys.TogglePreviewPrivacy) {
			m.togglePreviewPrivacy()
			return m, nil
		}

		if key.Matches(msg, keys.TogglePreviewLineNumbers) {
			m.togglePreviewLineNumbers()
			return m, nil
		}

		if key.Matches(msg, keys.PromoteTemporary) {
			m.openPromoteTemporaryBrowser()
			return m, nil
		}

		if key.Matches(msg, keys.ArchiveTemporary) {
			return m, m.archiveTemporarySelection()
		}

		if key.Matches(msg, keys.MoveToTemporary) {
			return m, m.moveSelectionToTemporary()
		}

		switch {
		case key.Matches(msg, keys.ToggleTemporary):
			if m.listMode != listModePins {
				m.toggleNotesTemporaryMode()
			}
			return m, nil

		case key.Matches(msg, keys.MoveUp):
			switch m.listMode {
			case listModeTemporary:
				m.moveTempCursor(-1)
			case listModePins:
				m.movePinsCursor(-1)
			case listModeTodos:
				m.moveTodoCursor(-1)
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
			case listModeTodos:
				m.moveTodoCursor(1)
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

		if key.Matches(msg, keys.NoteHistory) {
			return m, m.openNoteHistory()
		}

		if key.Matches(msg, keys.TrashBrowser) {
			return m, m.openTrashBrowser()
		}

		if key.Matches(msg, keys.NewTemplate) {
			return m, createTemplateCmd(m.rootDir)
		}

		if key.Matches(msg, keys.EditTemplates) {
			templates, err := notes.DiscoverTemplates(m.rootDir)
			if err != nil || len(templates) == 0 {
				m.status = "no templates found in .templates/"
				return m, nil
			}
			m.openTemplatePickerEditMode(templates)
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
	return m.renderPreviewMarkdown("<encrypted>", strings.Join([]string{
		"# <encrypted>",
		"",
		"This note is encrypted.",
		"Press E to enter your passphrase.",
	}, "\n"))
}

func (m Model) modalDimensions(minWidth, maxWidth int) (int, int) {
	modalWidth := min(maxWidth, max(minWidth, m.width-4))
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
