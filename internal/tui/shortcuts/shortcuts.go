// Package shortcuts provides shared TUI keybinding defaults, overrides, and help metadata.
package shortcuts

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"

	"atbuy/noteui/internal/config"
)

type Map struct {
	Open                     key.Binding
	EditInApp                key.Binding
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
	RemoveTag                key.Binding
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
	MoveUp                   key.Binding
	MoveDown                 key.Binding
	CollapseCategory         key.Binding
	ExpandCategory           key.Binding
	JumpBottom               key.Binding
	PendingG                 key.Binding
	BracketForward           key.Binding
	BracketBackward          key.Binding
	HeadingJumpKey           key.Binding
	TodoKey                  key.Binding
	TodoAdd                  key.Binding
	TodoDelete               key.Binding
	TodoEdit                 key.Binding
	TodoDueDate              key.Binding
	TodoPriority             key.Binding
	PendingZ                 key.Binding
	DeleteConfirm            key.Binding
	ScrollPageDown           key.Binding
	ScrollPageUp             key.Binding
	ToggleEncryption         key.Binding
	NoteHistory              key.Binding
	TrashBrowser             key.Binding
	NewTemplate              key.Binding
	EditTemplates            key.Binding
	OpenDailyNote            key.Binding
	LinkKey                  key.Binding
	FollowLink               key.Binding
	ShowThemePicker          key.Binding
}

type HelpEntry struct {
	Section string
	Key     string
	Desc    string
}

type namedBinding struct {
	name    string
	binding *key.Binding
}

func DefaultMap() Map {
	return Map{
		Search:                   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "Search")),
		Open:                     key.NewBinding(key.WithKeys("enter", "o"), key.WithHelp("enter/o", "Open in editor")),
		EditInApp:                key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "Edit in app")),
		Refresh:                  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "Refresh")),
		Quit:                     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "Quit")),
		Focus:                    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "Switch focused pane")),
		NewNote:                  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "New note")),
		NewTemporaryNote:         key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "New temporary note")),
		NewTodoList:              key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "New todo list")),
		TogglePreviewPrivacy:     key.NewBinding(key.WithKeys("B"), key.WithHelp("B", "Toggle preview privacy")),
		TogglePreviewLineNumbers: key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "Toggle preview line numbers")),
		ShowHelp:                 key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "Help")),
		ShowPins:                 key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "Pins")),
		ShowTodos:                key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "Todos")),
		CreateCategory:           key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "New category")),
		ToggleCategory:           key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "Toggle category")),
		Delete:                   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "Delete")),
		Move:                     key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "Move")),
		Rename:                   key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "Rename note")),
		AddTag:                   key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "Add tag")),
		RemoveTag:                key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "Remove tag")),
		ToggleSelect:             key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "Mark item")),
		ClearMarks:               key.NewBinding(key.WithKeys("V"), key.WithHelp("V", "Clear marks")),
		Pin:                      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "Pin")),
		PromoteTemporary:         key.NewBinding(key.WithKeys("M"), key.WithHelp("M", "Promote temp")),
		ArchiveTemporary:         key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "Archive temp")),
		MoveToTemporary:          key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "Move to temp")),
		ToggleSync:               key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "Toggle sync")),
		MakeShared:               key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "Toggle note shared")),
		ToggleTemporary:          key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "Toggle temporary notes")),
		CommandPalette:           key.NewBinding(key.WithKeys("ctrl+p", ":"), key.WithHelp("ctrl+p/:", "Command palette")),
		SelectWorkspace:          key.NewBinding(key.WithKeys("W"), key.WithHelp("W", "Switch workspace")),
		SelectSyncProfile:        key.NewBinding(key.WithKeys("F"), key.WithHelp("F", "Select sync profile")),
		OpenConflictCopy:         key.NewBinding(key.WithKeys("O"), key.WithHelp("O", "Resolve conflict")),
		ShowSyncDebug:            key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "Sync details")),
		ShowSyncTimeline:         key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "Sync timeline")),
		DeleteRemoteKeepLocal:    key.NewBinding(key.WithKeys("U"), key.WithHelp("U", "Delete remote copy")),
		SyncImportCurrent:        key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "Import current remote note")),
		SyncImport:               key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "Import all missing synced notes")),
		UndoDelete:               key.NewBinding(key.WithKeys("Z"), key.WithHelp("Z", "Undo last trash operation")),
		SortKey:                  key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "Sort")),
		SortByName:               key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "Sort by name")),
		SortByModified:           key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "Sort by modified")),
		SortByCreated:            key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "Sort by created")),
		SortBySize:               key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "Sort by size")),
		SortReverse:              key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "Reverse sort order")),
		ScrollHalfPageUp:         key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "Scroll half page up")),
		ScrollHalfPageDown:       key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "Scroll half page down")),
		NextMatch:                key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "Next match")),
		PrevMatch:                key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "Previous match")),
		MoveUp:                   key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/up", "Move up")),
		MoveDown:                 key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/down", "Move down")),
		CollapseCategory:         key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/left", "Collapse category")),
		ExpandCategory:           key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/right", "Expand category")),
		JumpBottom:               key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "Jump to bottom")),
		PendingG:                 key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "Jump to top (gg)")),
		BracketForward:           key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "Next heading/todo in preview")),
		BracketBackward:          key.NewBinding(key.WithKeys("["), key.WithHelp("[", "Prev heading/todo in preview")),
		HeadingJumpKey:           key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "Heading jump key")),
		TodoKey:                  key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "Todo action key")),
		TodoAdd:                  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "Todo add key")),
		TodoDelete:               key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "Todo delete key")),
		TodoEdit:                 key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "Todo edit key")),
		TodoDueDate:              key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "Todo due date key")),
		TodoPriority:             key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "Todo priority key")),
		PendingZ:                 key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "Center (zz)")),
		DeleteConfirm:            key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "Confirm delete")),
		ScrollPageDown:           key.NewBinding(key.WithKeys("ctrl+f", "pgdown"), key.WithHelp("ctrl+f/pgdn", "Page down")),
		ScrollPageUp:             key.NewBinding(key.WithKeys("ctrl+b", "pgup"), key.WithHelp("ctrl+b/pgup", "Page up")),
		ToggleEncryption:         key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "Toggle encryption")),
		NoteHistory:              key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "Note history")),
		TrashBrowser:             key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "Trash browser")),
		NewTemplate:              key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "New template")),
		EditTemplates:            key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", "Edit templates")),
		OpenDailyNote:            key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "Open today's note")),
		LinkKey:                  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "Link jump key")),
		FollowLink:               key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "Follow selected link")),
		ShowThemePicker:          key.NewBinding(key.WithKeys("ctrl+y"), key.WithHelp("ctrl+y", "Theme picker")),
	}
}

func ApplyConfig(m *Map, cfg config.KeysConfig) {
	apply := func(b *key.Binding, override []string) {
		if len(override) == 0 {
			return
		}
		*b = key.NewBinding(key.WithKeys(override...), key.WithHelp(strings.Join(override, "/"), b.Help().Desc))
	}
	apply(&m.Open, cfg.Open)
	apply(&m.EditInApp, cfg.EditInApp)
	apply(&m.Refresh, cfg.Refresh)
	apply(&m.Quit, cfg.Quit)
	apply(&m.Focus, cfg.Focus)
	apply(&m.NewNote, cfg.NewNote)
	apply(&m.NewTemporaryNote, cfg.NewTemporaryNote)
	apply(&m.NewTodoList, cfg.NewTodoList)
	apply(&m.Search, cfg.Search)
	apply(&m.ShowHelp, cfg.ShowHelp)
	apply(&m.ShowPins, cfg.ShowPins)
	apply(&m.ShowTodos, cfg.ShowTodos)
	apply(&m.CreateCategory, cfg.CreateCategory)
	apply(&m.ToggleCategory, cfg.ToggleCategory)
	apply(&m.Delete, cfg.Delete)
	apply(&m.Move, cfg.Move)
	apply(&m.Rename, cfg.Rename)
	apply(&m.AddTag, cfg.AddTag)
	apply(&m.RemoveTag, cfg.RemoveTag)
	apply(&m.ToggleSelect, cfg.ToggleSelect)
	apply(&m.ClearMarks, cfg.ClearMarks)
	apply(&m.Pin, cfg.Pin)
	apply(&m.PromoteTemporary, cfg.PromoteTemporary)
	apply(&m.ArchiveTemporary, cfg.ArchiveTemporary)
	apply(&m.MoveToTemporary, cfg.MoveToTemporary)
	apply(&m.ToggleSync, cfg.ToggleSync)
	apply(&m.MakeShared, cfg.MakeShared)
	apply(&m.ToggleTemporary, cfg.ToggleTemporary)
	apply(&m.CommandPalette, cfg.CommandPalette)
	apply(&m.SelectWorkspace, cfg.SelectWorkspace)
	apply(&m.SelectSyncProfile, cfg.SelectSyncProfile)
	apply(&m.OpenConflictCopy, cfg.OpenConflictCopy)
	apply(&m.ShowSyncDebug, cfg.ShowSyncDebug)
	apply(&m.ShowSyncTimeline, cfg.ShowSyncTimeline)
	apply(&m.DeleteRemoteKeepLocal, cfg.DeleteRemoteKeepLocal)
	apply(&m.SyncImportCurrent, cfg.SyncImportCurrent)
	apply(&m.SyncImport, cfg.SyncImport)
	apply(&m.UndoDelete, cfg.UndoDelete)
	apply(&m.TogglePreviewPrivacy, cfg.TogglePreviewPrivacy)
	apply(&m.TogglePreviewLineNumbers, cfg.TogglePreviewLineNumbers)
	apply(&m.SortKey, cfg.SortKey)
	apply(&m.SortByName, cfg.SortByName)
	apply(&m.SortByModified, cfg.SortByModified)
	apply(&m.SortByCreated, cfg.SortByCreated)
	apply(&m.SortBySize, cfg.SortBySize)
	apply(&m.SortReverse, cfg.SortReverse)
	apply(&m.ScrollHalfPageUp, cfg.ScrollHalfPageUp)
	apply(&m.ScrollHalfPageDown, cfg.ScrollHalfPageDown)
	apply(&m.NextMatch, cfg.NextMatch)
	apply(&m.PrevMatch, cfg.PrevMatch)
	apply(&m.MoveUp, cfg.MoveUp)
	apply(&m.MoveDown, cfg.MoveDown)
	apply(&m.CollapseCategory, cfg.CollapseCategory)
	apply(&m.ExpandCategory, cfg.ExpandCategory)
	apply(&m.JumpBottom, cfg.JumpBottom)
	apply(&m.PendingG, cfg.PendingG)
	apply(&m.BracketForward, cfg.BracketForward)
	apply(&m.BracketBackward, cfg.BracketBackward)
	apply(&m.HeadingJumpKey, cfg.HeadingJumpKey)
	apply(&m.TodoKey, cfg.TodoKey)
	apply(&m.TodoAdd, cfg.TodoAdd)
	apply(&m.TodoDelete, cfg.TodoDelete)
	apply(&m.TodoEdit, cfg.TodoEdit)
	apply(&m.TodoDueDate, cfg.TodoDueDate)
	apply(&m.TodoPriority, cfg.TodoPriority)
	apply(&m.PendingZ, cfg.PendingZ)
	apply(&m.DeleteConfirm, cfg.DeleteConfirm)
	apply(&m.ScrollPageDown, cfg.ScrollPageDown)
	apply(&m.ScrollPageUp, cfg.ScrollPageUp)
	apply(&m.ToggleEncryption, cfg.ToggleEncryption)
	apply(&m.NoteHistory, cfg.NoteHistory)
	apply(&m.TrashBrowser, cfg.TrashBrowser)
	apply(&m.NewTemplate, cfg.NewTemplate)
	apply(&m.EditTemplates, cfg.EditTemplates)
	apply(&m.OpenDailyNote, cfg.OpenDailyNote)
	apply(&m.LinkKey, cfg.LinkKey)
	apply(&m.FollowLink, cfg.FollowLink)
	apply(&m.ShowThemePicker, cfg.ShowThemePicker)
}

func ValidateCollisions(m Map) []string {
	primary := []namedBinding{{"open", &m.Open}, {"edit_in_app", &m.EditInApp}, {"refresh", &m.Refresh}, {"quit", &m.Quit}, {"focus", &m.Focus}, {"new_note", &m.NewNote}, {"new_temporary_note", &m.NewTemporaryNote}, {"new_todo_list", &m.NewTodoList}, {"search", &m.Search}, {"show_help", &m.ShowHelp}, {"show_pins", &m.ShowPins}, {"show_todos", &m.ShowTodos}, {"create_category", &m.CreateCategory}, {"toggle_category", &m.ToggleCategory}, {"delete", &m.Delete}, {"move", &m.Move}, {"rename", &m.Rename}, {"add_tag", &m.AddTag}, {"remove_tag", &m.RemoveTag}, {"toggle_select", &m.ToggleSelect}, {"clear_marks", &m.ClearMarks}, {"pin", &m.Pin}, {"promote_temporary", &m.PromoteTemporary}, {"archive_temporary", &m.ArchiveTemporary}, {"move_to_temporary", &m.MoveToTemporary}, {"toggle_sync", &m.ToggleSync}, {"make_shared", &m.MakeShared}, {"toggle_temporary", &m.ToggleTemporary}, {"command_palette", &m.CommandPalette}, {"select_workspace", &m.SelectWorkspace}, {"select_sync_profile", &m.SelectSyncProfile}, {"open_conflict_copy", &m.OpenConflictCopy}, {"show_sync_debug", &m.ShowSyncDebug}, {"show_sync_timeline", &m.ShowSyncTimeline}, {"delete_remote_keep_local", &m.DeleteRemoteKeepLocal}, {"sync_import_current", &m.SyncImportCurrent}, {"sync_import", &m.SyncImport}, {"undo_delete", &m.UndoDelete}, {"toggle_preview_privacy", &m.TogglePreviewPrivacy}, {"toggle_preview_line_numbers", &m.TogglePreviewLineNumbers}, {"sort_key", &m.SortKey}, {"scroll_half_page_up", &m.ScrollHalfPageUp}, {"scroll_half_page_down", &m.ScrollHalfPageDown}, {"move_up", &m.MoveUp}, {"move_down", &m.MoveDown}, {"collapse_category", &m.CollapseCategory}, {"expand_category", &m.ExpandCategory}, {"jump_bottom", &m.JumpBottom}, {"pending_g", &m.PendingG}, {"bracket_forward", &m.BracketForward}, {"bracket_backward", &m.BracketBackward}, {"pending_z", &m.PendingZ}, {"scroll_page_down", &m.ScrollPageDown}, {"scroll_page_up", &m.ScrollPageUp}, {"toggle_encryption", &m.ToggleEncryption}, {"note_history", &m.NoteHistory}, {"trash_browser", &m.TrashBrowser}, {"new_template", &m.NewTemplate}, {"edit_templates", &m.EditTemplates}, {"open_daily_note", &m.OpenDailyNote}, {"show_theme_picker", &m.ShowThemePicker}}
	seen := make(map[string][]string)
	for _, nb := range primary {
		for _, k := range nb.binding.Keys() {
			seen[k] = append(seen[k], nb.name)
		}
	}
	var collisions []string
	for k, names := range seen {
		if len(names) > 1 {
			collisions = append(collisions, fmt.Sprintf("key %q is bound to multiple actions: %s", k, strings.Join(names, ", ")))
		}
	}
	collisions = append(collisions,
		validateSequenceFamily(
			"sort menu key",
			[]namedBinding{
				{"sort_by_name", &m.SortByName},
				{"sort_by_modified", &m.SortByModified},
				{"sort_by_created", &m.SortByCreated},
				{"sort_by_size", &m.SortBySize},
				{"sort_reverse", &m.SortReverse},
			},
		)...,
	)
	collisions = append(collisions,
		validateSequenceFamily(
			"preview bracket chord second key",
			[]namedBinding{
				{"heading_jump_key", &m.HeadingJumpKey},
				{"todo_key", &m.TodoKey},
				{"link_key", &m.LinkKey},
			},
		)...,
	)
	collisions = append(collisions,
		validateSequenceFamily(
			"todo chord second key",
			[]namedBinding{
				{"todo_key", &m.TodoKey},
				{"todo_add", &m.TodoAdd},
				{"todo_delete", &m.TodoDelete},
				{"todo_edit", &m.TodoEdit},
				{"todo_due_date", &m.TodoDueDate},
				{"todo_priority", &m.TodoPriority},
			},
		)...,
	)
	sort.Strings(collisions)
	return collisions
}

func validateSequenceFamily(label string, items []namedBinding) []string {
	seen := make(map[string][]string)
	for _, item := range items {
		for _, k := range item.binding.Keys() {
			seen[k] = append(seen[k], item.name)
		}
	}
	var collisions []string
	for k, names := range seen {
		if len(names) > 1 {
			collisions = append(collisions, fmt.Sprintf("%s %q is bound to multiple actions: %s", label, k, strings.Join(names, ", ")))
		}
	}
	sort.Strings(collisions)
	return collisions
}

func HelpEntries(m Map) []HelpEntry {
	bf := m.BracketForward.Help().Key
	bb := m.BracketBackward.Help().Key
	return []HelpEntry{
		{Section: "Tree", Key: m.CommandPalette.Help().Key, Desc: "Command palette: notes, actions, and workspace switch"},
		{Section: "Tree", Key: m.MoveDown.Help().Key + " / " + m.MoveUp.Help().Key, Desc: "Move up and down"},
		{Section: "Tree", Key: m.ScrollHalfPageUp.Help().Key + " / " + m.ScrollHalfPageDown.Help().Key, Desc: "Scroll half page up / down"},
		{Section: "Tree", Key: m.CollapseCategory.Help().Key + "/" + m.ExpandCategory.Help().Key, Desc: "Collapse/Expand category"},
		{Section: "Tree", Key: m.PendingG.Help().Key + m.PendingG.Help().Key + " / " + m.JumpBottom.Help().Key, Desc: "Jump to top / bottom of list"},
		{Section: "Tree", Key: m.Open.Help().Key, Desc: "Open note or jump from Pins"},
		{Section: "Tree", Key: m.EditInApp.Help().Key, Desc: "Edit current note in app"},
		{Section: "Tree", Key: m.Move.Help().Key, Desc: "Move current item or marked batch"},
		{Section: "Tree", Key: m.ToggleSelect.Help().Key, Desc: "Mark/unmark item for bulk actions"},
		{Section: "Tree", Key: m.Rename.Help().Key, Desc: "Rename note/category"},
		{Section: "Tree", Key: m.AddTag.Help().Key, Desc: "Add tags to selected note or marked notes"},
		{Section: "Tree", Key: m.RemoveTag.Help().Key, Desc: "Remove tags from selected note or marked notes"},
		{Section: "Tree", Key: m.Pin.Help().Key, Desc: "Pin or unpin current item or marked notes"},
		{Section: "Tree", Key: m.ToggleSync.Help().Key, Desc: "Toggle selected note sync"},
		{Section: "Tree", Key: m.MakeShared.Help().Key, Desc: "Toggle shared status of selected note"},
		{Section: "Tree", Key: m.SelectWorkspace.Help().Key, Desc: "Open the workspace picker"},
		{Section: "Tree", Key: m.SelectSyncProfile.Help().Key, Desc: "Select default sync profile (updates sync.default_profile only)"},
		{Section: "Tree", Key: m.OpenConflictCopy.Help().Key, Desc: "Resolve selected conflict"},
		{Section: "Tree", Key: m.ShowSyncDebug.Help().Key, Desc: "Show sync details"},
		{Section: "Tree", Key: m.ShowSyncTimeline.Help().Key, Desc: "View sync run history timeline"},
		{Section: "Tree", Key: m.DeleteRemoteKeepLocal.Help().Key, Desc: "Delete remote copy, keep local note"},
		{Section: "Tree", Key: m.SyncImportCurrent.Help().Key, Desc: "Import current remote note"},
		{Section: "Tree", Key: m.SyncImport.Help().Key, Desc: "Import all missing synced notes"},
		{Section: "Tree", Key: m.UndoDelete.Help().Key, Desc: "Undo last trash operation (restore from trash)"},
		{Section: "Tree", Key: m.CreateCategory.Help().Key, Desc: "Create category"},
		{Section: "Tree", Key: m.NewTodoList.Help().Key, Desc: "New todo list (tree focus)"},
		{Section: "Notes", Key: m.NewNote.Help().Key, Desc: "New note in current view"},
		{Section: "Notes", Key: m.OpenDailyNote.Help().Key, Desc: "Open or create today's daily note"},
		{Section: "Notes", Key: m.NewTemplate.Help().Key, Desc: "New template in .templates/"},
		{Section: "Notes", Key: m.EditTemplates.Help().Key, Desc: "Edit templates"},
		{Section: "Notes", Key: m.NewTemporaryNote.Help().Key, Desc: "New temporary note"},
		{Section: "Notes", Key: m.ToggleTemporary.Help().Key, Desc: "Toggle Notes / Temporary (tree focus)"},
		{Section: "Notes", Key: m.PromoteTemporary.Help().Key, Desc: "Promote selected temporary note or marked batch"},
		{Section: "Notes", Key: m.ArchiveTemporary.Help().Key, Desc: "Archive selected temporary note or marked batch"},
		{Section: "Notes", Key: m.MoveToTemporary.Help().Key, Desc: "Move selected note or marked batch to temporary"},
		{Section: "Notes", Key: m.ClearMarks.Help().Key, Desc: "Clear all current marks"},
		{Section: "Notes", Key: m.ShowPins.Help().Key, Desc: "Toggle Pins view"},
		{Section: "Notes", Key: m.ShowTodos.Help().Key, Desc: "Toggle global open todos view"},
		{Section: "Notes", Key: m.NoteHistory.Help().Key, Desc: "Open version history for the selected note"},
		{Section: "Notes", Key: m.TrashBrowser.Help().Key, Desc: "Open trash browser to restore trashed notes"},
		{Section: "Todos", Key: m.MoveDown.Help().Key + " / " + m.MoveUp.Help().Key, Desc: "Move through open tasks"},
		{Section: "Todos", Key: m.Open.Help().Key, Desc: "Jump to the source note"},
		{Section: "Todos", Key: m.TodoKey.Help().Key + m.TodoKey.Help().Key, Desc: "Toggle selected open task"},
		{Section: "Todos", Key: m.TodoKey.Help().Key + m.TodoAdd.Help().Key, Desc: "Add a task to the selected note"},
		{Section: "Todos", Key: m.TodoKey.Help().Key + m.TodoDelete.Help().Key, Desc: "Delete the selected open task"},
		{Section: "Todos", Key: m.TodoKey.Help().Key + m.TodoEdit.Help().Key, Desc: "Edit the selected open task"},
		{Section: "Todos", Key: m.TodoKey.Help().Key + m.TodoDueDate.Help().Key, Desc: "Set or clear the selected task due date"},
		{Section: "Todos", Key: m.TodoKey.Help().Key + m.TodoPriority.Help().Key, Desc: "Set or clear the selected task priority"},
		{Section: "Preview", Key: m.NextMatch.Help().Key + " / " + m.PrevMatch.Help().Key, Desc: "Next / previous match in preview"},
		{Section: "Preview", Key: m.PendingZ.Help().Key + m.PendingZ.Help().Key, Desc: "Center current match in preview"},
		{Section: "Preview", Key: m.TogglePreviewPrivacy.Help().Key, Desc: "Toggle preview privacy"},
		{Section: "Preview", Key: m.TogglePreviewLineNumbers.Help().Key, Desc: "Toggle preview line numbers"},
		{Section: "Preview", Key: bf + m.HeadingJumpKey.Help().Key + " / " + bb + m.HeadingJumpKey.Help().Key, Desc: "Next / prev heading in preview"},
		{Section: "Preview", Key: bf + m.TodoKey.Help().Key + " / " + bb + m.TodoKey.Help().Key, Desc: "Next / prev todo in preview"},
		{Section: "Preview", Key: m.PendingG.Help().Key + m.PendingG.Help().Key + " / " + m.JumpBottom.Help().Key, Desc: "First / last todo in todo nav"},
		{Section: "Preview", Key: m.TodoKey.Help().Key + m.TodoKey.Help().Key, Desc: "Toggle current todo checkbox"},
		{Section: "Preview", Key: m.TodoKey.Help().Key + m.TodoAdd.Help().Key, Desc: "Add new todo item"},
		{Section: "Preview", Key: m.TodoKey.Help().Key + m.TodoDelete.Help().Key, Desc: "Delete current todo item"},
		{Section: "Preview", Key: m.TodoKey.Help().Key + m.TodoEdit.Help().Key, Desc: "Edit current todo item"},
		{Section: "Preview", Key: m.TodoKey.Help().Key + m.TodoDueDate.Help().Key, Desc: "Set or clear current todo due date"},
		{Section: "Preview", Key: m.TodoKey.Help().Key + m.TodoPriority.Help().Key, Desc: "Set or clear current todo priority"},
		{Section: "Preview", Key: m.ToggleEncryption.Help().Key, Desc: "Toggle note encryption"},
		{Section: "Filter", Key: m.Search.Help().Key, Desc: "Search"},
		{Section: "Filter", Key: "#tag", Desc: "Filter by tag in search"},
		{Section: "Filter", Key: "esc", Desc: "Leave search, then clear on second press"},
		{Section: "Global", Key: m.ShowThemePicker.Help().Key, Desc: "Open theme picker (searchable live preview; / or tab filters; saves theme.name only)"},
		{Section: "Global", Key: m.Focus.Help().Key, Desc: "Switch focused pane"},
		{Section: "Global", Key: m.SortKey.Help().Key, Desc: "Sort menu (name / modified / created / size / reverse)"},
		{Section: "Global", Key: m.Refresh.Help().Key, Desc: "Refresh"},
		{Section: "Global", Key: m.Delete.Help().Key + m.DeleteConfirm.Help().Key, Desc: "Trash note/category"},
		{Section: "Global", Key: "esc", Desc: "Close help"},
		{Section: "Global", Key: m.Quit.Help().Key, Desc: "Quit"},
	}
}
