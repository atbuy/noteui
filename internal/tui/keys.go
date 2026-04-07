package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"

	"atbuy/noteui/internal/config"
)

type keyMap struct {
	Open                     key.Binding
	Refresh                  key.Binding
	Quit                     key.Binding
	Focus                    key.Binding
	NewNote                  key.Binding
	NewTemporaryNote         key.Binding
	NewTodoList              key.Binding
	Search                   key.Binding
	ShowHelp                 key.Binding
	ShowPins                 key.Binding
	ShowTodos                key.Binding
	CreateCategory           key.Binding
	ToggleCategory           key.Binding
	Delete                   key.Binding
	Move                     key.Binding
	Rename                   key.Binding
	AddTag                   key.Binding
	ToggleSelect             key.Binding
	ClearMarks               key.Binding
	Pin                      key.Binding
	PromoteTemporary         key.Binding
	ArchiveTemporary         key.Binding
	MoveToTemporary          key.Binding
	ToggleSync               key.Binding
	MakeShared               key.Binding
	ToggleTemporary          key.Binding
	CommandPalette           key.Binding
	SelectWorkspace          key.Binding
	SelectSyncProfile        key.Binding
	OpenConflictCopy         key.Binding
	ShowSyncDebug            key.Binding
	DeleteRemoteKeepLocal    key.Binding
	SyncImportCurrent        key.Binding
	SyncImport               key.Binding
	TogglePreviewPrivacy     key.Binding
	TogglePreviewLineNumbers key.Binding
	SortToggle               key.Binding
	ScrollHalfPageUp         key.Binding
	ScrollHalfPageDown       key.Binding
	NextMatch                key.Binding
	PrevMatch                key.Binding

	MoveUp           key.Binding
	MoveDown         key.Binding
	CollapseCategory key.Binding
	ExpandCategory   key.Binding
	JumpBottom       key.Binding
	PendingG         key.Binding
	BracketForward   key.Binding
	BracketBackward  key.Binding
	HeadingJumpKey   key.Binding
	TodoKey          key.Binding
	TodoAdd          key.Binding
	TodoDelete       key.Binding
	TodoEdit         key.Binding
	TodoDueDate      key.Binding
	TodoPriority     key.Binding
	PendingZ         key.Binding
	DeleteConfirm    key.Binding
	ScrollPageDown   key.Binding
	ScrollPageUp     key.Binding
	ToggleEncryption key.Binding
}

var keys = keyMap{
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "Search"),
	),
	Open: key.NewBinding(
		key.WithKeys("enter", "o"),
		key.WithHelp("enter/o", "Open in editor"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "Refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "Quit"),
	),
	Focus: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "Switch focused pane"),
	),
	NewNote: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "New note"),
	),
	NewTemporaryNote: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "New temporary note"),
	),
	NewTodoList: key.NewBinding(
		key.WithKeys("T"),
		key.WithHelp("T", "New todo list"),
	),
	TogglePreviewPrivacy: key.NewBinding(
		key.WithKeys("B"),
		key.WithHelp("B", "Toggle preview privacy"),
	),
	TogglePreviewLineNumbers: key.NewBinding(
		key.WithKeys("L"),
		key.WithHelp("L", "Toggle preview line numbers"),
	),
	ShowHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "Help"),
	),
	ShowPins: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "Pins"),
	),
	ShowTodos: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "Todos"),
	),
	CreateCategory: key.NewBinding(
		key.WithKeys("C"),
		key.WithHelp("C", "New category"),
	),
	ToggleCategory: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "Toggle category"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "Delete"),
	),
	Move: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "Move"),
	),
	Rename: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "Rename note"),
	),
	AddTag: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "Add tag"),
	),
	ToggleSelect: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "Mark item"),
	),
	ClearMarks: key.NewBinding(
		key.WithKeys("V"),
		key.WithHelp("V", "Clear marks"),
	),
	Pin: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "Pin"),
	),
	PromoteTemporary: key.NewBinding(
		key.WithKeys("M"),
		key.WithHelp("M", "Promote temp"),
	),
	ArchiveTemporary: key.NewBinding(
		key.WithKeys("ctrl+a"),
		key.WithHelp("ctrl+a", "Archive temp"),
	),
	MoveToTemporary: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "Move to temp"),
	),
	ToggleSync: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "Toggle sync"),
	),
	MakeShared: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "Make note shared"),
	),
	ToggleTemporary: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "Toggle temporary notes"),
	),
	CommandPalette: key.NewBinding(
		key.WithKeys("ctrl+p", ":"),
		key.WithHelp("ctrl+p/:", "Command palette"),
	),
	SelectWorkspace: key.NewBinding(
		key.WithKeys("W"),
		key.WithHelp("W", "Switch workspace"),
	),
	SelectSyncProfile: key.NewBinding(
		key.WithKeys("F"),
		key.WithHelp("F", "Select sync profile"),
	),
	OpenConflictCopy: key.NewBinding(
		key.WithKeys("O"),
		key.WithHelp("O", "Resolve conflict"),
	),
	ShowSyncDebug: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "Sync details"),
	),
	DeleteRemoteKeepLocal: key.NewBinding(
		key.WithKeys("U"),
		key.WithHelp("U", "Delete remote copy"),
	),
	SyncImportCurrent: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "Import current remote note"),
	),
	SyncImport: key.NewBinding(
		key.WithKeys("I"),
		key.WithHelp("I", "Import all missing synced notes"),
	),
	SortToggle: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "Toggle sort order"),
	),
	ScrollHalfPageUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "Scroll half page up"),
	),
	ScrollHalfPageDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "Scroll half page down"),
	),
	NextMatch: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "Next match"),
	),
	PrevMatch: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "Previous match"),
	),

	MoveUp: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/up", "Move up"),
	),
	MoveDown: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/down", "Move down"),
	),
	CollapseCategory: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("h/left", "Collapse category"),
	),
	ExpandCategory: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("l/right", "Expand category"),
	),
	JumpBottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "Jump to bottom"),
	),
	PendingG: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "Jump to top (gg)"),
	),
	BracketForward: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "Next heading/todo in preview"),
	),
	BracketBackward: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "Prev heading/todo in preview"),
	),
	HeadingJumpKey: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "Heading jump key"),
	),
	TodoKey: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "Todo action key"),
	),
	TodoAdd: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "Todo add key"),
	),
	TodoDelete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "Todo delete key"),
	),
	TodoEdit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "Todo edit key"),
	),
	TodoDueDate: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "Todo due date key"),
	),
	TodoPriority: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "Todo priority key"),
	),
	PendingZ: key.NewBinding(
		key.WithKeys("z"),
		key.WithHelp("z", "Center (zz)"),
	),
	DeleteConfirm: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "Confirm delete"),
	),
	ScrollPageDown: key.NewBinding(
		key.WithKeys("ctrl+f", "pgdown"),
		key.WithHelp("ctrl+f/pgdn", "Page down"),
	),
	ScrollPageUp: key.NewBinding(
		key.WithKeys("ctrl+b", "pgup"),
		key.WithHelp("ctrl+b/pgup", "Page up"),
	),
	ToggleEncryption: key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "Toggle encryption"),
	),
}

// ApplyConfigKeys overwrites key bindings with user-provided overrides from config.
// Fields with empty slices are left at their defaults.
func ApplyConfigKeys(cfg config.KeysConfig) {
	apply := func(b *key.Binding, override []string) {
		if len(override) == 0 {
			return
		}
		*b = key.NewBinding(
			key.WithKeys(override...),
			key.WithHelp(strings.Join(override, "/"), b.Help().Desc),
		)
	}
	apply(&keys.Open, cfg.Open)
	apply(&keys.Refresh, cfg.Refresh)
	apply(&keys.Quit, cfg.Quit)
	apply(&keys.Focus, cfg.Focus)
	apply(&keys.NewNote, cfg.NewNote)
	apply(&keys.NewTemporaryNote, cfg.NewTemporaryNote)
	apply(&keys.NewTodoList, cfg.NewTodoList)
	apply(&keys.Search, cfg.Search)
	apply(&keys.ShowHelp, cfg.ShowHelp)
	apply(&keys.ShowPins, cfg.ShowPins)
	apply(&keys.ShowTodos, cfg.ShowTodos)
	apply(&keys.CreateCategory, cfg.CreateCategory)
	apply(&keys.ToggleCategory, cfg.ToggleCategory)
	apply(&keys.Delete, cfg.Delete)
	apply(&keys.Move, cfg.Move)
	apply(&keys.Rename, cfg.Rename)
	apply(&keys.AddTag, cfg.AddTag)
	apply(&keys.ToggleSelect, cfg.ToggleSelect)
	apply(&keys.ClearMarks, cfg.ClearMarks)
	apply(&keys.Pin, cfg.Pin)
	apply(&keys.PromoteTemporary, cfg.PromoteTemporary)
	apply(&keys.ArchiveTemporary, cfg.ArchiveTemporary)
	apply(&keys.MoveToTemporary, cfg.MoveToTemporary)
	apply(&keys.ToggleSync, cfg.ToggleSync)
	apply(&keys.MakeShared, cfg.MakeShared)
	apply(&keys.ToggleTemporary, cfg.ToggleTemporary)
	apply(&keys.CommandPalette, cfg.CommandPalette)
	apply(&keys.SelectWorkspace, cfg.SelectWorkspace)
	apply(&keys.SelectSyncProfile, cfg.SelectSyncProfile)
	apply(&keys.OpenConflictCopy, cfg.OpenConflictCopy)
	apply(&keys.ShowSyncDebug, cfg.ShowSyncDebug)
	apply(&keys.DeleteRemoteKeepLocal, cfg.DeleteRemoteKeepLocal)
	apply(&keys.SyncImportCurrent, cfg.SyncImportCurrent)
	apply(&keys.SyncImport, cfg.SyncImport)
	apply(&keys.TogglePreviewPrivacy, cfg.TogglePreviewPrivacy)
	apply(&keys.TogglePreviewLineNumbers, cfg.TogglePreviewLineNumbers)
	apply(&keys.SortToggle, cfg.SortToggle)
	apply(&keys.ScrollHalfPageUp, cfg.ScrollHalfPageUp)
	apply(&keys.ScrollHalfPageDown, cfg.ScrollHalfPageDown)
	apply(&keys.NextMatch, cfg.NextMatch)
	apply(&keys.PrevMatch, cfg.PrevMatch)
	apply(&keys.MoveUp, cfg.MoveUp)
	apply(&keys.MoveDown, cfg.MoveDown)
	apply(&keys.CollapseCategory, cfg.CollapseCategory)
	apply(&keys.ExpandCategory, cfg.ExpandCategory)
	apply(&keys.JumpBottom, cfg.JumpBottom)
	apply(&keys.PendingG, cfg.PendingG)
	apply(&keys.BracketForward, cfg.BracketForward)
	apply(&keys.BracketBackward, cfg.BracketBackward)
	apply(&keys.HeadingJumpKey, cfg.HeadingJumpKey)
	apply(&keys.TodoKey, cfg.TodoKey)
	apply(&keys.TodoAdd, cfg.TodoAdd)
	apply(&keys.TodoDelete, cfg.TodoDelete)
	apply(&keys.TodoEdit, cfg.TodoEdit)
	apply(&keys.TodoDueDate, cfg.TodoDueDate)
	apply(&keys.TodoPriority, cfg.TodoPriority)
	apply(&keys.PendingZ, cfg.PendingZ)
	apply(&keys.DeleteConfirm, cfg.DeleteConfirm)
	apply(&keys.ScrollPageDown, cfg.ScrollPageDown)
	apply(&keys.ScrollPageUp, cfg.ScrollPageUp)
	apply(&keys.ToggleEncryption, cfg.ToggleEncryption)
}
