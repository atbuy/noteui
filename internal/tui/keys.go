package tui

import (
	"fmt"
	"sort"
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
	ShowSyncTimeline         key.Binding
	DeleteRemoteKeepLocal    key.Binding
	SyncImportCurrent        key.Binding
	SyncImport               key.Binding
	UndoDelete               key.Binding
	TogglePreviewPrivacy     key.Binding
	TogglePreviewLineNumbers key.Binding
	SortKey                  key.Binding
	SortByName               key.Binding
	SortByModified           key.Binding
	SortByCreated            key.Binding
	SortBySize               key.Binding
	SortReverse              key.Binding
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
	NoteHistory      key.Binding
	TrashBrowser     key.Binding
	NewTemplate      key.Binding
	EditTemplates    key.Binding
	OpenDailyNote    key.Binding
	LinkKey          key.Binding
	FollowLink       key.Binding
	ShowThemePicker  key.Binding
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
		key.WithHelp("ctrl+s", "Toggle note shared"),
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
	ShowSyncTimeline: key.NewBinding(
		key.WithKeys("Y"),
		key.WithHelp("Y", "Sync timeline"),
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
	UndoDelete: key.NewBinding(
		key.WithKeys("Z"),
		key.WithHelp("Z", "Undo last trash operation"),
	),
	SortKey: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "Sort"),
	),
	SortByName: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "Sort by name"),
	),
	SortByModified: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "Sort by modified"),
	),
	SortByCreated: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "Sort by created"),
	),
	SortBySize: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "Sort by size"),
	),
	SortReverse: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "Reverse sort order"),
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
	NoteHistory: key.NewBinding(
		key.WithKeys("H"),
		key.WithHelp("H", "Note history"),
	),
	TrashBrowser: key.NewBinding(
		key.WithKeys("X"),
		key.WithHelp("X", "Trash browser"),
	),
	NewTemplate: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "New template"),
	),
	EditTemplates: key.NewBinding(
		key.WithKeys("ctrl+k"),
		key.WithHelp("ctrl+k", "Edit templates"),
	),
	OpenDailyNote: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "Open today's note"),
	),
	LinkKey: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "Link jump key"),
	),
	FollowLink: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "Follow selected link"),
	),
	ShowThemePicker: key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("ctrl+y", "Theme picker"),
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
	apply(&keys.ShowSyncTimeline, cfg.ShowSyncTimeline)
	apply(&keys.DeleteRemoteKeepLocal, cfg.DeleteRemoteKeepLocal)
	apply(&keys.SyncImportCurrent, cfg.SyncImportCurrent)
	apply(&keys.SyncImport, cfg.SyncImport)
	apply(&keys.UndoDelete, cfg.UndoDelete)
	apply(&keys.TogglePreviewPrivacy, cfg.TogglePreviewPrivacy)
	apply(&keys.TogglePreviewLineNumbers, cfg.TogglePreviewLineNumbers)
	apply(&keys.SortKey, cfg.SortKey)
	apply(&keys.SortByName, cfg.SortByName)
	apply(&keys.SortByModified, cfg.SortByModified)
	apply(&keys.SortByCreated, cfg.SortByCreated)
	apply(&keys.SortBySize, cfg.SortBySize)
	apply(&keys.SortReverse, cfg.SortReverse)
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
	apply(&keys.NoteHistory, cfg.NoteHistory)
	apply(&keys.TrashBrowser, cfg.TrashBrowser)
	apply(&keys.NewTemplate, cfg.NewTemplate)
	apply(&keys.EditTemplates, cfg.EditTemplates)
	apply(&keys.OpenDailyNote, cfg.OpenDailyNote)
	apply(&keys.LinkKey, cfg.LinkKey)
	apply(&keys.FollowLink, cfg.FollowLink)
	apply(&keys.ShowThemePicker, cfg.ShowThemePicker)
}

// ValidateKeyCollisions checks for duplicate key assignments among primary bindings
// (those active simultaneously in the main view). Context-specific bindings that
// intentionally reuse keys in sub-modes (search, todo-action, delete-confirm,
// bracket-pending) are excluded to avoid false positives.
// Returns one description string per collision; nil when there are no conflicts.
func ValidateKeyCollisions() []string {
	type named struct {
		name    string
		binding *key.Binding
	}
	primary := []named{
		{"open", &keys.Open},
		{"refresh", &keys.Refresh},
		{"quit", &keys.Quit},
		{"focus", &keys.Focus},
		{"new_note", &keys.NewNote},
		{"new_temporary_note", &keys.NewTemporaryNote},
		{"new_todo_list", &keys.NewTodoList},
		{"search", &keys.Search},
		{"show_help", &keys.ShowHelp},
		{"show_pins", &keys.ShowPins},
		{"show_todos", &keys.ShowTodos},
		{"create_category", &keys.CreateCategory},
		{"toggle_category", &keys.ToggleCategory},
		{"delete", &keys.Delete},
		{"move", &keys.Move},
		{"rename", &keys.Rename},
		{"add_tag", &keys.AddTag},
		{"toggle_select", &keys.ToggleSelect},
		{"clear_marks", &keys.ClearMarks},
		{"pin", &keys.Pin},
		{"promote_temporary", &keys.PromoteTemporary},
		{"archive_temporary", &keys.ArchiveTemporary},
		{"move_to_temporary", &keys.MoveToTemporary},
		{"toggle_sync", &keys.ToggleSync},
		{"make_shared", &keys.MakeShared},
		{"toggle_temporary", &keys.ToggleTemporary},
		{"command_palette", &keys.CommandPalette},
		{"select_workspace", &keys.SelectWorkspace},
		{"select_sync_profile", &keys.SelectSyncProfile},
		{"open_conflict_copy", &keys.OpenConflictCopy},
		{"show_sync_debug", &keys.ShowSyncDebug},
		{"show_sync_timeline", &keys.ShowSyncTimeline},
		{"delete_remote_keep_local", &keys.DeleteRemoteKeepLocal},
		{"sync_import_current", &keys.SyncImportCurrent},
		{"sync_import", &keys.SyncImport},
		{"undo_delete", &keys.UndoDelete},
		{"toggle_preview_privacy", &keys.TogglePreviewPrivacy},
		{"toggle_preview_line_numbers", &keys.TogglePreviewLineNumbers},
		{"sort_key", &keys.SortKey},
		{"scroll_half_page_up", &keys.ScrollHalfPageUp},
		{"scroll_half_page_down", &keys.ScrollHalfPageDown},
		{"move_up", &keys.MoveUp},
		{"move_down", &keys.MoveDown},
		{"collapse_category", &keys.CollapseCategory},
		{"expand_category", &keys.ExpandCategory},
		{"jump_bottom", &keys.JumpBottom},
		{"pending_g", &keys.PendingG},
		{"bracket_forward", &keys.BracketForward},
		{"bracket_backward", &keys.BracketBackward},
		{"pending_z", &keys.PendingZ},
		{"scroll_page_down", &keys.ScrollPageDown},
		{"scroll_page_up", &keys.ScrollPageUp},
		{"toggle_encryption", &keys.ToggleEncryption},
		{"note_history", &keys.NoteHistory},
		{"trash_browser", &keys.TrashBrowser},
		{"new_template", &keys.NewTemplate},
		{"edit_templates", &keys.EditTemplates},
		{"open_daily_note", &keys.OpenDailyNote},
		{"show_theme_picker", &keys.ShowThemePicker},
	}

	seen := make(map[string][]string) // key string -> action names
	for _, nb := range primary {
		for _, k := range nb.binding.Keys() {
			seen[k] = append(seen[k], nb.name)
		}
	}

	var collisions []string
	for k, names := range seen {
		if len(names) > 1 {
			collisions = append(collisions,
				fmt.Sprintf("key %q is bound to multiple actions: %s", k, strings.Join(names, ", ")))
		}
	}
	sort.Strings(collisions)
	return collisions
}
