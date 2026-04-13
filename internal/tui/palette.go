package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/editor"
	"atbuy/noteui/internal/notes"
	notesync "atbuy/noteui/internal/sync"
)

const (
	paletteMaxVisible        = 12
	paletteMaxRecentCommands = 8
)

// paletteCommand holds the definition for a single app command.
type paletteCommand struct {
	name     string
	desc     string
	category string
	action   string
}

const (
	cmdNewNote               = "new_note"
	cmdNewTemporaryNote      = "new_temporary_note"
	cmdNewTodoList           = "new_todo_list"
	cmdNewNoteFromTemplate   = "new_note_from_template"
	cmdNewTemplate           = "new_template"
	cmdEditTemplates         = "edit_templates"
	cmdNewCategory           = "new_category"
	cmdMoveCurrent           = "move_current"
	cmdMarkCurrent           = "mark_current"
	cmdClearMarks            = "clear_marks"
	cmdRenameCurrent         = "rename_current"
	cmdAddTags               = "add_tags"
	cmdTrashCurrent          = "trash_current"
	cmdTogglePin             = "toggle_pin"
	cmdPromoteTemporary      = "promote_temporary"
	cmdArchiveTemporary      = "archive_temporary"
	cmdMoveToTemporary       = "move_to_temporary"
	cmdToggleSort            = "toggle_sort"
	cmdRefresh               = "refresh"
	cmdTogglePrivacy         = "toggle_privacy"
	cmdToggleLineNumbers     = "toggle_line_numbers"
	cmdShowPins              = "show_pins"
	cmdShowTodos             = "show_todos"
	cmdShowHelp              = "show_help"
	cmdSwitchWorkspace       = "switch_workspace"
	cmdToggleTemporary       = "toggle_temporary"
	cmdToggleSync            = "toggle_sync"
	cmdMakeShared            = "make_shared"
	cmdSelectSyncProfile     = "select_sync_profile"
	cmdShowSyncDetails       = "show_sync_details"
	cmdResolveConflict       = "resolve_conflict"
	cmdDeleteRemoteKeepLocal = "delete_remote_keep_local"
	cmdImportCurrent         = "import_current"
	cmdImportAll             = "import_all"
	cmdSyncNow               = "sync_now"
	cmdShowSyncTimeline      = "show_sync_timeline"
	cmdToggleEncryption      = "toggle_encryption"
	cmdToggleTodo            = "toggle_todo"
	cmdAddTodo               = "add_todo"
	cmdDeleteTodo            = "delete_todo"
	cmdEditTodo              = "edit_todo"
	cmdSetTodoDueDate        = "set_todo_due_date"
	cmdSetTodoPriority       = "set_todo_priority"
)

// paletteKind identifies the type of an item in the command palette.
type paletteKind int

const (
	paletteKindNote     paletteKind = iota // note in the main tree
	paletteKindTempNote                    // note in .tmp/
	paletteKindCommand                     // named app command
)

type paletteSection int

const (
	paletteSectionSuggested paletteSection = iota
	paletteSectionCommand
	paletteSectionNote
	paletteSectionTempNote
)

// paletteItem is one entry shown in the command palette.
type paletteItem struct {
	kind    paletteKind
	section paletteSection
	score   int
	title   string     // primary display (note title)
	sub     string     // secondary display (relPath, or ".tmp/relPath")
	note    notes.Note // valid for paletteKindNote and paletteKindTempNote
	cmd     paletteCommand
}

type paletteSearchField struct {
	text   string
	weight int
}

func normalizePaletteRecentCommands(actions []string) []string {
	if len(actions) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(actions))
	out := make([]string, 0, min(len(actions), paletteMaxRecentCommands))
	for _, action := range actions {
		action = strings.TrimSpace(action)
		if action == "" || seen[action] {
			continue
		}
		seen[action] = true
		out = append(out, action)
		if len(out) >= paletteMaxRecentCommands {
			break
		}
	}
	return out
}

func paletteSectionTitle(section paletteSection) string {
	switch section {
	case paletteSectionSuggested:
		return "Suggested actions"
	case paletteSectionCommand:
		return "Commands"
	case paletteSectionNote:
		return "Notes"
	case paletteSectionTempNote:
		return "Temporary notes"
	default:
		return "Results"
	}
}

// isConflictCopy reports whether relPath matches the conflict copy naming pattern
// produced by createConflict(): "base.conflict-YYYYMMDD-HHMMSS.ext".
// Conflict copies are excluded from the palette since they are conflict artifacts,
// not regular notes; the user resolves them from the tree view.
func isConflictCopy(relPath string) bool {
	name := filepath.Base(relPath)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	idx := strings.LastIndex(base, ".conflict-")
	if idx < 0 {
		return false
	}
	suffix := base[idx+len(".conflict-"):]
	return len(suffix) == 15 // "YYYYMMDD-HHMMSS"
}

func (m *Model) openCommandPalette() {
	items := make([]paletteItem, 0, len(m.notes)+len(m.tempNotes))
	for _, n := range m.notes {
		if isConflictCopy(n.RelPath) {
			continue
		}
		items = append(items, paletteItem{
			kind:  paletteKindNote,
			title: n.Title(),
			sub:   n.RelPath,
			note:  n,
		})
	}
	for _, n := range m.tempNotes {
		items = append(items, paletteItem{
			kind:  paletteKindTempNote,
			title: n.Title(),
			sub:   ".tmp/" + n.RelPath,
			note:  n,
		})
	}
	m.commandPaletteItems = items
	m.commandPaletteInput.Reset()
	m.commandPaletteInput.Focus()
	m.commandPaletteCursor = 0
	m.rebuildPaletteFiltered()
	m.showCommandPalette = true
}

func paletteCommands(m Model) []paletteCommand {
	cmds := []paletteCommand{
		{name: "New Note", desc: "Create a note in the current location", category: "notes", action: cmdNewNote},
		{name: "New Temporary Note", desc: "Create a temporary note", category: "notes", action: cmdNewTemporaryNote},
		{name: "Toggle Sort", desc: "Switch between alphabetical and modified sorting", category: "view", action: cmdToggleSort},
		{name: "Refresh", desc: "Refresh notes and sync state", category: "app", action: cmdRefresh},
		{name: "Toggle Preview Privacy", desc: "Toggle preview privacy", category: "preview", action: cmdTogglePrivacy},
		{name: "Toggle Preview Line Numbers", desc: "Toggle preview line numbers", category: "preview", action: cmdToggleLineNumbers},
		{name: "Show Help", desc: "Open the help modal", category: "app", action: cmdShowHelp},
		{name: "Show Pins", desc: "Toggle the pins view", category: "view", action: cmdShowPins},
		{name: "Show Todos", desc: "Toggle the global open todos view", category: "view", action: cmdShowTodos},
	}

	cmds = appendPaletteCommand(cmds, m.listMode != listModePins,
		paletteCommand{name: "Toggle Temporary Notes", desc: "Switch between notes and temporary notes", category: "view", action: cmdToggleTemporary})
	cmds = appendPaletteCommand(cmds, m.canSwitchWorkspace(),
		paletteCommand{name: "Switch Workspace", desc: "Switch to another configured workspace", category: "app", action: cmdSwitchWorkspace})
	cmds = appendPaletteCommand(cmds, m.listMode != listModeTemporary && m.listMode != listModePins,
		paletteCommand{name: "New Todo List", desc: "Create a new todo list in the current location", category: "notes", action: cmdNewTodoList})
	cmds = appendPaletteCommand(cmds, m.hasTemplates(),
		paletteCommand{name: "New Note from Template", desc: "Create a note using a user-defined template", category: "notes", action: cmdNewNoteFromTemplate})
	cmds = appendPaletteCommand(cmds, true,
		paletteCommand{name: "New Template", desc: "Create a new blank template in .templates/", category: "notes", action: cmdNewTemplate})
	cmds = appendPaletteCommand(cmds, m.hasTemplates(),
		paletteCommand{name: "Edit Templates", desc: "Open a template for editing", category: "notes", action: cmdEditTemplates})
	cmds = appendPaletteCommand(cmds, m.listMode == listModeNotes,
		paletteCommand{name: "New Category", desc: "Create a category in the notes tree", category: "tree", action: cmdNewCategory})
	cmds = appendPaletteCommand(cmds, m.canMoveCurrent(),
		paletteCommand{name: "Move Current Item", desc: "Move the selected item", category: "selection", action: cmdMoveCurrent})
	cmds = appendPaletteCommand(cmds, m.canMarkCurrent(),
		paletteCommand{name: "Mark Current Item", desc: "Mark or unmark the selected item", category: "selection", action: cmdMarkCurrent})
	cmds = appendPaletteCommand(cmds, m.canClearMarks(),
		paletteCommand{name: "Clear Marks", desc: "Clear all marked items", category: "selection", action: cmdClearMarks})
	cmds = appendPaletteCommand(cmds, m.canRenameCurrent(),
		paletteCommand{name: "Rename Current Item", desc: "Rename the selected item", category: "selection", action: cmdRenameCurrent})
	cmds = appendPaletteCommand(cmds, m.canAddTagsCurrent(),
		paletteCommand{name: "Add Tags", desc: "Add tags to the selected note", category: "selection", action: cmdAddTags})
	cmds = appendPaletteCommand(cmds, m.canTrashCurrent(),
		paletteCommand{name: "Trash Current Item", desc: "Trash the selected note or category", category: "selection", action: cmdTrashCurrent})
	cmds = appendPaletteCommand(cmds, m.canTogglePinCurrent(),
		paletteCommand{name: "Toggle Pin", desc: "Pin or unpin the current item or marked notes", category: "selection", action: cmdTogglePin})
	cmds = appendPaletteCommand(cmds, m.canPromoteTemporary(),
		paletteCommand{name: "Promote to Notes", desc: "Promote the selected temporary note or marked batch into main notes", category: "selection", action: cmdPromoteTemporary})
	cmds = appendPaletteCommand(cmds, m.canArchiveTemporary(),
		paletteCommand{name: "Archive Temporary Notes", desc: "Archive the selected temporary note or marked batch", category: "selection", action: cmdArchiveTemporary})
	cmds = appendPaletteCommand(cmds, m.canMoveSelectionToTemporary(),
		paletteCommand{name: "Move Notes to Temporary", desc: "Move the selected note or marked batch into temporary notes", category: "selection", action: cmdMoveToTemporary})
	cmds = appendPaletteCommand(cmds, m.canToggleSyncCurrent(),
		paletteCommand{name: "Toggle Note Sync", desc: "Toggle sync for the selected note", category: "sync", action: cmdToggleSync})
	cmds = appendPaletteCommand(cmds, m.canToggleSharedCurrent(),
		paletteCommand{name: "Toggle Note Shared", desc: "Toggle shared status of the selected note", category: "sync", action: cmdMakeShared})
	cmds = appendPaletteCommand(cmds, m.canSelectSyncProfile(),
		paletteCommand{name: "Select Sync Profile", desc: "Choose the default sync profile", category: "sync", action: cmdSelectSyncProfile})
	cmds = appendPaletteCommand(cmds, m.canShowSyncDetailsCurrent(),
		paletteCommand{name: "Show Sync Details", desc: "Open sync details for the selected note", category: "sync", action: cmdShowSyncDetails})
	cmds = appendPaletteCommand(cmds, m.canResolveConflictCurrent(),
		paletteCommand{name: "Resolve Conflict", desc: "Open conflict resolution for the selected note", category: "sync", action: cmdResolveConflict})
	cmds = appendPaletteCommand(cmds, m.canDeleteRemoteCopyCurrent(),
		paletteCommand{name: "Delete Remote Copy", desc: "Delete the remote copy and keep the local note", category: "sync", action: cmdDeleteRemoteKeepLocal})
	cmds = appendPaletteCommand(cmds, m.canImportCurrentRemoteNote(),
		paletteCommand{name: "Import Current Remote Note", desc: "Import the selected remote-only note", category: "sync", action: cmdImportCurrent})
	cmds = appendPaletteCommand(cmds, notesync.HasSyncProfile(m.cfg.Sync),
		paletteCommand{name: "Import All Remote Notes", desc: "Import all remote-only synced notes", category: "sync", action: cmdImportAll})
	cmds = appendPaletteCommand(cmds, notesync.HasSyncProfile(m.cfg.Sync),
		paletteCommand{name: "Sync Now", desc: "Run sync immediately", category: "sync", action: cmdSyncNow})
	cmds = appendPaletteCommand(cmds, notesync.HasSyncProfile(m.cfg.Sync),
		paletteCommand{name: "View Sync Timeline", desc: "Show history of sync runs for this workspace", category: "sync", action: cmdShowSyncTimeline})
	cmds = appendPaletteCommand(cmds, m.canToggleEncryptionCurrent(),
		paletteCommand{name: "Toggle Note Encryption", desc: "Encrypt or decrypt the selected note", category: "notes", action: cmdToggleEncryption})
	cmds = appendPaletteCommand(cmds, m.canAddTodoItem(),
		paletteCommand{name: "Add Todo Item", desc: "Add a todo item to the current note", category: "todo", action: cmdAddTodo})
	cmds = appendPaletteCommand(cmds, m.canToggleCurrentTodo(),
		paletteCommand{name: "Toggle Current Todo", desc: "Toggle the selected todo item", category: "todo", action: cmdToggleTodo})
	cmds = appendPaletteCommand(cmds, m.canToggleCurrentTodo(),
		paletteCommand{name: "Delete Current Todo", desc: "Delete the selected todo item", category: "todo", action: cmdDeleteTodo})
	cmds = appendPaletteCommand(cmds, m.canToggleCurrentTodo(),
		paletteCommand{name: "Edit Current Todo", desc: "Edit the selected todo item", category: "todo", action: cmdEditTodo})
	cmds = appendPaletteCommand(cmds, m.canToggleCurrentTodo(),
		paletteCommand{name: "Set Todo Due Date", desc: "Set or clear the selected todo due date", category: "todo", action: cmdSetTodoDueDate})
	cmds = appendPaletteCommand(cmds, m.canToggleCurrentTodo(),
		paletteCommand{name: "Set Todo Priority", desc: "Set or clear the selected todo priority", category: "todo", action: cmdSetTodoPriority})

	return cmds
}

func appendPaletteCommand(cmds []paletteCommand, ok bool, cmd paletteCommand) []paletteCommand {
	if !ok {
		return cmds
	}
	return append(cmds, cmd)
}

func (m Model) canMoveCurrent() bool {
	if m.listMode == listModePins {
		return m.currentPinItem() != nil
	}
	if m.listMode == listModeTemporary {
		return m.currentTempNote() != nil
	}
	item := m.currentTreeItem()
	return item != nil && item.Kind != treeRemoteNote
}

func (m Model) canMarkCurrent() bool {
	switch m.listMode {
	case listModeTemporary:
		return m.currentTempNote() != nil
	case listModeNotes:
		item := m.currentTreeItem()
		if item == nil {
			return false
		}
		return item.Kind != treeCategory || item.RelPath != ""
	default:
		return false
	}
}

func (m Model) canRenameCurrent() bool {
	if m.listMode == listModePins {
		return m.currentPinItem() != nil
	}
	if m.listMode == listModeTemporary {
		return m.currentTempNote() != nil
	}
	item := m.currentTreeItem()
	if item == nil {
		return false
	}
	return item.Kind != treeRemoteNote && (item.Kind != treeCategory || item.RelPath != "")
}

func (m Model) canAddTagsCurrent() bool {
	if m.currentRemoteOnlyNote() != nil {
		return false
	}
	_, err := m.selectedTaggableNotePaths()
	return err == nil
}

func (m Model) canTrashCurrent() bool {
	if m.listMode == listModePins {
		return m.currentPinItem() != nil
	}
	if m.listMode == listModeTemporary {
		_, err := m.selectedTempNotesForAction()
		return err == nil
	}
	if m.listMode == listModeNotes && m.hasMarksInCurrentMode() {
		_, err := m.selectedMainNotesForAction()
		return err == nil
	}
	item := m.currentTreeItem()
	if item == nil {
		return false
	}
	if item.Kind == treeRemoteNote {
		return false
	}
	return item.Kind != treeCategory || item.RelPath != ""
}

func (m Model) canTogglePinCurrent() bool {
	if m.listMode == listModePins {
		return m.currentPinItem() != nil
	}
	if m.listMode == listModeTemporary {
		_, err := m.selectedTempNotesForAction()
		return err == nil
	}
	if m.listMode == listModeNotes && m.hasMarksInCurrentMode() {
		_, err := m.selectedMainNotesForAction()
		return err == nil
	}
	item := m.currentTreeItem()
	return item != nil && item.Kind != treeRemoteNote && (item.Kind != treeCategory || item.RelPath != "")
}

func (m Model) canToggleSyncCurrent() bool {
	if m.listMode != listModeNotes {
		return false
	}
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		return false
	}
	return item.Note.SyncClass != notes.SyncClassShared
}

func (m Model) canToggleSharedCurrent() bool {
	if m.listMode != listModeNotes {
		return false
	}
	item := m.currentTreeItem()
	return item != nil && item.Kind == treeNote && item.Note != nil
}

func (m Model) canSelectSyncProfile() bool {
	return len(sortedSyncProfileNames(m.cfg.Sync)) > 0
}

func (m Model) canShowSyncDetailsCurrent() bool {
	_, ok := m.currentSyncDebugDetails()
	return ok
}

func (m Model) canResolveConflictCurrent() bool {
	return strings.TrimSpace(m.currentConflictCopyPath()) != ""
}

func (m Model) canDeleteRemoteCopyCurrent() bool {
	if m.listMode != listModeNotes {
		return false
	}
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		return false
	}
	if item.Note.SyncClass != notes.SyncClassSynced {
		return false
	}
	_, ok := m.syncRecords[filepath.ToSlash(item.Note.RelPath)]
	return ok
}

func (m Model) canImportCurrentRemoteNote() bool {
	return m.currentRemoteOnlyNote() != nil
}

func (m Model) canToggleEncryptionCurrent() bool {
	return m.currentRemoteOnlyNote() == nil && strings.TrimSpace(m.currentNotePath()) != ""
}

func (m Model) canAddTodoItem() bool {
	return strings.TrimSpace(m.previewPath) != ""
}

func (m Model) canToggleCurrentTodo() bool {
	return len(m.previewTodos) > 0 && m.previewTodoCursor >= 0 && m.previewTodoCursor < len(m.previewTodos)
}

func (m *Model) rebuildPaletteFiltered() {
	query := strings.TrimSpace(m.commandPaletteInput.Value())
	filtered := make([]paletteItem, 0, len(m.commandPaletteItems)+32)

	for _, item := range m.commandPaletteItems {
		score, ok := m.paletteItemScore(item, query)
		if !ok {
			continue
		}
		item.score = score
		item.section = m.paletteSectionForItem(item, query)
		filtered = append(filtered, item)
	}
	for _, cmd := range paletteCommands(*m) {
		item := paletteItem{
			kind:  paletteKindCommand,
			title: cmd.name,
			sub:   cmd.category,
			cmd:   cmd,
		}
		score, ok := m.paletteItemScore(item, query)
		if !ok {
			continue
		}
		item.score = score + m.paletteCommandBoost(cmd, query)
		item.section = m.paletteSectionForItem(item, query)
		filtered = append(filtered, item)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return m.paletteItemLess(filtered[i], filtered[j])
	})

	m.commandPaletteFiltered = filtered
	if m.commandPaletteCursor >= len(m.commandPaletteFiltered) {
		m.commandPaletteCursor = max(0, len(m.commandPaletteFiltered)-1)
	}
}

func (m Model) paletteItemScore(item paletteItem, query string) (int, bool) {
	switch item.kind {
	case paletteKindCommand:
		return m.paletteCommandScore(item.cmd, query)
	case paletteKindNote, paletteKindTempNote:
		return m.paletteNoteScore(item.note, query)
	default:
		return 0, false
	}
}

func (m Model) paletteNoteScore(n notes.Note, query string) (int, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return 0, true
	}

	if after, ok := strings.CutPrefix(q, "#"); ok {
		score := paletteTagScore(n.Tags, after)
		if score == 0 {
			return 0, false
		}
		return 320 + score, true
	}

	previewText := n.Preview
	if n.Encrypted {
		previewText = "<encrypted>"
	}
	tagText := strings.Join(n.Tags, " ")
	blob := strings.Join([]string{n.Title(), n.Name, n.RelPath, tagText, previewText}, " ")
	matched := m.noteMatches(n, q)
	score := paletteCompositeScore(q,
		paletteSearchField{text: n.Title(), weight: 100},
		paletteSearchField{text: n.RelPath, weight: 88},
		paletteSearchField{text: n.Name, weight: 75},
		paletteSearchField{text: tagText, weight: 72},
		paletteSearchField{text: previewText, weight: 55},
		paletteSearchField{text: blob, weight: 62},
	)
	if matched {
		if score == 0 {
			score = 1
		}
		return 120 + score, true
	}
	if score == 0 {
		return 0, false
	}
	return score, true
}

func (m Model) paletteCommandScore(cmd paletteCommand, query string) (int, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return 0, true
	}
	blob := strings.Join([]string{cmd.name, cmd.desc, cmd.category}, " ")
	score := paletteCompositeScore(q,
		paletteSearchField{text: cmd.name, weight: 100},
		paletteSearchField{text: cmd.desc, weight: 72},
		paletteSearchField{text: cmd.category, weight: 58},
		paletteSearchField{text: blob, weight: 70},
	)
	if score == 0 && !paletteHasAllTerms(strings.ToLower(blob), q) {
		return 0, false
	}
	if score == 0 {
		score = 1
	}
	if paletteHasAllTerms(strings.ToLower(blob), q) {
		score += 90
	}
	return 100 + score, true
}

func (m Model) paletteCommandBoost(cmd paletteCommand, query string) int {
	boost := 0
	if m.paletteCommandIsSuggested(cmd.action) {
		boost += 180
	}
	if idx := m.paletteRecentCommandIndex(cmd.action); idx >= 0 {
		if strings.TrimSpace(query) == "" {
			boost += max(0, 220-idx*28)
		} else {
			boost += max(0, 110-idx*16)
		}
	}
	return boost
}

func (m Model) paletteSectionForItem(item paletteItem, query string) paletteSection {
	switch item.kind {
	case paletteKindCommand:
		if m.paletteCommandIsSuggested(item.cmd.action) {
			return paletteSectionSuggested
		}
		if strings.TrimSpace(query) == "" && m.paletteRecentCommandIndex(item.cmd.action) >= 0 {
			return paletteSectionSuggested
		}
		return paletteSectionCommand
	case paletteKindTempNote:
		return paletteSectionTempNote
	default:
		return paletteSectionNote
	}
}

func (m Model) paletteCommandIsSuggested(action string) bool {
	switch action {
	case cmdMoveCurrent,
		cmdMarkCurrent,
		cmdRenameCurrent,
		cmdAddTags,
		cmdTrashCurrent,
		cmdTogglePin,
		cmdToggleSync,
		cmdMakeShared,
		cmdShowSyncDetails,
		cmdResolveConflict,
		cmdDeleteRemoteKeepLocal,
		cmdImportCurrent,
		cmdToggleEncryption,
		cmdAddTodo,
		cmdToggleTodo,
		cmdDeleteTodo,
		cmdEditTodo:
		return true
	default:
		return false
	}
}

func (m Model) paletteRecentCommandIndex(action string) int {
	for idx, recent := range m.workspaceState.RecentCommands {
		if recent == action {
			return idx
		}
	}
	return -1
}

func paletteCompositeScore(query string, fields ...paletteSearchField) int {
	best := 0
	for _, field := range fields {
		if strings.TrimSpace(field.text) == "" || field.weight <= 0 {
			continue
		}
		score := paletteFieldScore(field.text, query)
		if score == 0 {
			continue
		}
		weighted := score * field.weight / 100
		if weighted > best {
			best = weighted
		}
	}
	return best
}

func paletteFieldScore(text, query string) int {
	field := strings.ToLower(strings.TrimSpace(text))
	q := strings.ToLower(strings.TrimSpace(query))
	if field == "" || q == "" {
		return 0
	}

	best := 0
	if field == q {
		best = max(best, 1000)
	}
	if strings.HasPrefix(field, q) {
		best = max(best, 900-min(160, len(field)-len(q))*4)
	}
	if idx := strings.Index(field, q); idx >= 0 {
		best = max(best, 760-min(240, idx*16))
	}
	tokenCount := len(strings.FieldsFunc(field, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}))
	if tokenCount <= 3 {
		if initials := paletteInitialism(field); initials != "" && strings.HasPrefix(initials, paletteCompact(q)) {
			best = max(best, 690)
		}
		if fuzzy, ok := paletteFuzzyScore(field, q); ok {
			best = max(best, 520+fuzzy)
		}
	}
	terms := strings.Fields(q)
	if len(terms) > 1 && paletteHasAllTerms(field, q) {
		termScore := 0
		for _, term := range terms {
			switch {
			case field == term:
				termScore += 120
			case strings.HasPrefix(field, term):
				termScore += 96
			case strings.Contains(field, term):
				termScore += 72
			}
		}
		best = max(best, 420+termScore)
	}
	return max(0, best)
}

func paletteTagScore(tags []string, tag string) int {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return 240
	}
	best := 0
	for _, candidate := range tags {
		score := paletteFieldScore(candidate, tag)
		if score > best {
			best = score
		}
	}
	return best
}

func paletteHasAllTerms(text, query string) bool {
	for _, term := range strings.Fields(strings.ToLower(strings.TrimSpace(query))) {
		if !strings.Contains(text, term) {
			return false
		}
	}
	return true
}

func paletteFuzzyScore(text, query string) (int, bool) {
	fieldRunes := []rune(paletteCompact(text))
	queryRunes := []rune(paletteCompact(query))
	if len(fieldRunes) == 0 || len(queryRunes) == 0 {
		return 0, false
	}

	qIdx := 0
	start := -1
	prev := -1
	gaps := 0
	for i, r := range fieldRunes {
		if qIdx >= len(queryRunes) || r != queryRunes[qIdx] {
			continue
		}
		if start < 0 {
			start = i
		}
		if prev >= 0 {
			gaps += i - prev - 1
		}
		prev = i
		qIdx++
		if qIdx == len(queryRunes) {
			break
		}
	}
	if qIdx != len(queryRunes) {
		return 0, false
	}

	score := 180 - gaps*10 - start*4 - max(0, len(fieldRunes)-len(queryRunes))*2
	if score < 120 {
		return 0, false
	}
	return score, true
}

func paletteCompact(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(text)) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

func paletteInitialism(text string) string {
	parts := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	for _, part := range parts {
		r := []rune(part)
		if len(r) == 0 {
			continue
		}
		b.WriteRune(r[0])
	}
	return b.String()
}

func (m Model) paletteItemLess(a, b paletteItem) bool {
	if ap, bp := paletteSectionPriority(a.section), paletteSectionPriority(b.section); ap != bp {
		return ap < bp
	}
	if a.score != b.score {
		return a.score > b.score
	}
	at, bt := strings.ToLower(a.title), strings.ToLower(b.title)
	if at != bt {
		return at < bt
	}
	return strings.ToLower(a.sub) < strings.ToLower(b.sub)
}

func paletteSectionPriority(section paletteSection) int {
	switch section {
	case paletteSectionSuggested:
		return 0
	case paletteSectionCommand:
		return 1
	case paletteSectionNote:
		return 2
	case paletteSectionTempNote:
		return 3
	default:
		return 4
	}
}

func (m *Model) tabCompletePalette() {
	if len(m.commandPaletteFiltered) == 0 {
		return
	}
	item := m.commandPaletteFiltered[m.commandPaletteCursor]
	m.commandPaletteInput.SetValue(item.title)
	m.commandPaletteInput.CursorEnd()
	m.rebuildPaletteFiltered()
}

func (m *Model) commitPaletteSelection() tea.Cmd {
	if len(m.commandPaletteFiltered) == 0 {
		return nil
	}
	item := m.commandPaletteFiltered[m.commandPaletteCursor]
	m.showCommandPalette = false
	m.commandPaletteInput.Blur()

	switch item.kind {
	case paletteKindNote:
		m.switchToNotesMode()
		m.selectTreeNote(item.note.RelPath)
	case paletteKindTempNote:
		m.switchToTemporaryMode()
		m.selectTemporaryNote(item.note.RelPath)
	case paletteKindCommand:
		m.recordPaletteCommandUse(item.cmd.action)
		return m.executePaletteCommand(item.cmd.action)
	}
	return nil
}

func (m *Model) recordPaletteCommandUse(action string) {
	action = strings.TrimSpace(action)
	if action == "" {
		return
	}
	next := make([]string, 0, paletteMaxRecentCommands)
	next = append(next, action)
	for _, existing := range m.workspaceState.RecentCommands {
		if existing == action || strings.TrimSpace(existing) == "" {
			continue
		}
		next = append(next, existing)
		if len(next) >= paletteMaxRecentCommands {
			break
		}
	}
	m.workspaceState.RecentCommands = normalizePaletteRecentCommands(next)
	_ = m.saveLocalState()
}

func (m *Model) openHelpModal() {
	m.showHelp = true
	m.helpScroll = 0
	m.helpMouseSuppressed = false
	m.helpInput.SetValue("")
	m.helpInput.Focus()
	m.rebuildHelpRowsCache()
	m.rebuildHelpModalCache(max(8, min(20, m.height-16)))
	m.pendingG = false
	m.status = "help"
}

func (m *Model) toggleSortOrder() {
	m.sortByModTime = !m.sortByModTime
	_ = m.saveTreeState()
	m.rebuildTree()
	if m.sortByModTime {
		m.status = "sorting by modified time"
	} else {
		m.status = "sorting alphabetically"
	}
}

func (m *Model) startRefresh() tea.Cmd {
	m.status = "refreshing..."
	return batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), m.scheduleSync())
}

func (m *Model) openCreateCategory() {
	if m.listMode != listModeNotes {
		m.status = "categories only available in notes tree"
		return
	}
	m.showCreateCategory = true
	m.categoryInput.SetValue(m.currentCategoryPrefix())
	m.categoryInput.Focus()
	m.categoryInput.CursorEnd()
	m.status = "new category"
}

func (m *Model) startNewNote() tea.Cmd {
	if m.listMode == listModeTemporary {
		return createTemporaryNoteCmd(m.rootDir)
	}
	if m.listMode == listModePins {
		m.status = "press enter to jump to item first"
		return nil
	}
	templates, err := notes.DiscoverTemplates(m.rootDir)
	if err != nil || len(templates) == 0 {
		return createNoteCmd(m.rootDir, m.currentTargetDir())
	}
	m.openTemplatePicker(templates)
	return nil
}

func (m *Model) openDailyNote() tea.Cmd {
	templatePath := ""
	if m.cfg.DailyNotes.Template != "" {
		templatePath = filepath.Join(notes.TemplatesRoot(m.rootDir), m.cfg.DailyNotes.Template)
	}
	path, created, err := notes.OpenOrCreateDailyNote(m.rootDir, m.cfg.DailyNotes.Dir, templatePath, time.Now())
	if err != nil {
		m.status = "daily note error: " + err.Error()
		return nil
	}
	m.dailyNoteOpen = true
	if created {
		m.status = "created daily note"
		return batchCmds(refreshAllCmd(m.rootDir, m.sessionToken), editor.Open(path))
	}
	m.status = "opening daily note"
	return editor.Open(path)
}

func (m *Model) openTemplatePicker(templates []notes.Template) {
	m.templateItems = templates
	m.templatePickerCursor = 0
	m.templatePickerEditMode = false
	m.templatePickerRelDir = m.currentTargetDir()
	m.showTemplatePicker = true
	m.status = "select template"
}

func (m *Model) openTemplatePickerEditMode(templates []notes.Template) {
	m.templateItems = templates
	m.templatePickerCursor = 0
	m.templatePickerEditMode = true
	m.templatePickerRelDir = ""
	m.showTemplatePicker = true
	m.status = "select template to edit"
}

func (m *Model) closeTemplatePicker(status string) {
	m.showTemplatePicker = false
	m.templatePickerEditMode = false
	m.templateItems = nil
	m.templatePickerCursor = 0
	m.templatePickerRelDir = ""
	if strings.TrimSpace(status) != "" {
		m.status = status
	}
}

func (m *Model) moveTemplateCursor(delta int) {
	total := len(m.templateItems)
	if !m.templatePickerEditMode {
		total++ // +1 for "Blank note" at index 0
	}
	next := m.templatePickerCursor + delta
	if next < 0 {
		next = 0
	}
	if next >= total {
		next = total - 1
	}
	m.templatePickerCursor = next
}

func (m *Model) confirmTemplatePicker() tea.Cmd {
	if m.templatePickerEditMode {
		return m.editTemplateAtCursor()
	}
	if m.templatePickerCursor == 0 {
		relDir := m.templatePickerRelDir
		m.closeTemplatePicker("")
		return createNoteCmd(m.rootDir, relDir)
	}
	idx := m.templatePickerCursor - 1
	if idx >= len(m.templateItems) {
		m.closeTemplatePicker("template selection out of range")
		return nil
	}
	tmpl := m.templateItems[idx]
	relDir := m.templatePickerRelDir
	m.closeTemplatePicker("")
	return createNoteFromTemplateCmd(m.rootDir, relDir, tmpl.Path)
}

func (m *Model) editTemplateAtCursor() tea.Cmd {
	idx := m.templatePickerCursor
	if m.templatePickerEditMode {
		// In edit mode, index 0 maps directly to templateItems[0] (no "Blank note").
	} else {
		// In create mode, cursor > 0 maps to templateItems[cursor-1].
		idx = m.templatePickerCursor - 1
	}
	if idx < 0 || idx >= len(m.templateItems) {
		return nil
	}
	tmpl := m.templateItems[idx]
	m.closeTemplatePicker("")
	return editor.Open(tmpl.Path)
}

func (m Model) hasTemplates() bool {
	templates, err := notes.DiscoverTemplates(m.rootDir)
	return err == nil && len(templates) > 0
}

func (m *Model) startNewTodoList() tea.Cmd {
	if m.listMode == listModeTemporary || m.listMode == listModePins {
		m.status = "todo lists only available in notes tree"
		return nil
	}
	if m.listMode == listModeTodos {
		return createTodoNoteCmd(m.rootDir, m.currentTargetDir())
	}
	return createTodoNoteCmd(m.rootDir, m.currentTargetDir())
}

func (m *Model) togglePreviewPrivacy() {
	if m.cfg.Preview.Privacy {
		m.status = "preview privacy forced by config"
		return
	}
	m.previewPrivacyEnabled = !m.previewPrivacyEnabled
	m.previewPath = ""
	if m.previewPrivacyEnabled {
		m.status = "preview privacy enabled"
	} else {
		m.status = "preview privacy disabled"
	}
	m.refreshPreview()
}

func (m *Model) togglePreviewLineNumbers() {
	m.previewLineNumbersEnabled = !m.previewLineNumbersEnabled
	m.previewPath = ""
	if m.previewLineNumbersEnabled {
		m.status = "preview line numbers enabled"
	} else {
		m.status = "preview line numbers disabled"
	}
	m.refreshPreview()
}

func (m *Model) toggleNoteSyncCurrent() tea.Cmd {
	item := m.currentTreeItem()
	if item != nil && item.Kind == treeRemoteNote {
		m.status = "note is only on the server; press i to import it or I to import all"
		return nil
	}
	if item == nil || item.Kind != treeNote || item.Note == nil {
		m.status = "sync toggle only works on notes"
		return nil
	}
	if item.Note.SyncClass == notes.SyncClassShared {
		m.status = "shared notes cannot be toggled"
		return nil
	}
	return toggleNoteSyncCmd(item.Note.Path)
}

func (m *Model) makeCurrentNoteShared() tea.Cmd {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		m.status = "toggle shared only works on notes"
		return nil
	}
	target := notes.SyncClassShared
	if item.Note.SyncClass == notes.SyncClassShared {
		target = notes.SyncClassLocal
	}
	return toggleNoteSharedCmd(item.Note.Path, target)
}

func (m *Model) deleteRemoteCopyCurrent() tea.Cmd {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeNote || item.Note == nil {
		m.status = "remote delete only works on synced local notes"
		return nil
	}
	if item.Note.SyncClass != notes.SyncClassSynced {
		m.status = "remote delete only works on synced local notes"
		return nil
	}
	if _, ok := m.syncRecords[filepath.ToSlash(item.Note.RelPath)]; !ok {
		m.status = "note is not linked to a remote copy"
		return nil
	}
	m.status = "deleting remote copy..."
	return batchCmds(
		deleteRemoteNoteKeepLocalCmd(m.rootDir, item.Note.Path, m.activeWorkspaceSyncRemoteRoot(), m.cfg.Sync, m.sessionToken),
		m.startSyncVisual(item.Note.RelPath),
	)
}

func (m *Model) importCurrentRemoteNote() tea.Cmd {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeRemoteNote || item.RemoteNote == nil {
		m.status = "single-note import only works on remote notes"
		return nil
	}
	m.status = "importing remote note..."
	return batchCmds(
		importCurrentSyncedNoteCmd(m.rootDir, m.activeWorkspaceSyncRemoteRoot(), m.cfg.Sync, item.RemoteNote.ID, m.sessionToken),
		m.startSyncVisual(remoteOnlySyncVisualKey(item.RemoteNote.ID)),
	)
}

func (m *Model) importAllRemoteNotes() tea.Cmd {
	m.status = "importing synced notes..."
	return importSyncedNotesCmd(m.rootDir, m.activeWorkspaceSyncRemoteRoot(), m.cfg.Sync, m.sessionToken)
}

func (m *Model) startImmediateSync() tea.Cmd {
	if !notesync.HasSyncProfile(m.cfg.Sync) {
		return nil
	}
	m.status = "syncing..."
	return m.startSyncRun()
}

func (m *Model) armAddTodoItem() {
	path := m.previewPath
	if path == "" {
		m.status = "no note selected"
		return
	}
	m.showTodoAdd = true
	m.todoInput.SetValue("")
	m.todoInput.Focus()
	m.status = "add todo"
}

func (m *Model) trashCurrentItem() tea.Cmd {
	m.armDeleteCurrent()
	if m.deletePending == nil {
		return nil
	}
	cmd := m.confirmDeleteCurrent()
	m.deletePending = nil
	if cmd != nil {
		m.status = "deleting..."
	}
	return cmd
}

func (m *Model) executePaletteCommand(action string) tea.Cmd {
	switch action {
	case cmdNewNote:
		return m.startNewNote()
	case cmdNewTemporaryNote:
		return createTemporaryNoteCmd(m.rootDir)
	case cmdNewTodoList:
		return m.startNewTodoList()
	case cmdNewNoteFromTemplate:
		templates, err := notes.DiscoverTemplates(m.rootDir)
		if err != nil || len(templates) == 0 {
			m.status = "no templates found in .templates/"
			return nil
		}
		m.openTemplatePicker(templates)
		return nil
	case cmdNewTemplate:
		return createTemplateCmd(m.rootDir)
	case cmdEditTemplates:
		templates, err := notes.DiscoverTemplates(m.rootDir)
		if err != nil || len(templates) == 0 {
			m.status = "no templates found in .templates/"
			return nil
		}
		m.openTemplatePickerEditMode(templates)
		return nil
	case cmdNewCategory:
		m.openCreateCategory()
		return nil
	case cmdMoveCurrent:
		m.armMoveCurrent()
		return nil
	case cmdMarkCurrent:
		m.toggleMarkCurrent()
		return nil
	case cmdClearMarks:
		m.clearAllMarks()
		return nil
	case cmdRenameCurrent:
		m.armRenameCurrent()
		return nil
	case cmdAddTags:
		m.armAddTagCurrent()
		return nil
	case cmdTrashCurrent:
		return m.trashCurrentItem()
	case cmdTogglePin:
		if err := m.togglePinCurrent(); err != nil {
			m.status = "pin failed: " + err.Error()
			return nil
		}
		return m.scheduleSync()
	case cmdPromoteTemporary:
		m.openPromoteTemporaryBrowser()
		return nil
	case cmdArchiveTemporary:
		return m.archiveTemporarySelection()
	case cmdMoveToTemporary:
		return m.moveSelectionToTemporary()
	case cmdToggleSort:
		m.toggleSortOrder()
		return nil
	case cmdRefresh:
		return m.startRefresh()
	case cmdTogglePrivacy:
		m.togglePreviewPrivacy()
		return nil
	case cmdToggleLineNumbers:
		m.togglePreviewLineNumbers()
		return nil
	case cmdShowPins:
		m.togglePinsMode()
		return nil
	case cmdShowTodos:
		m.toggleTodosMode()
		return nil
	case cmdShowHelp:
		m.openHelpModal()
		return nil
	case cmdSwitchWorkspace:
		m.openWorkspacePicker()
		return nil
	case cmdToggleTemporary:
		if m.listMode != listModePins {
			m.toggleNotesTemporaryMode()
		}
		return nil
	case cmdToggleSync:
		return m.toggleNoteSyncCurrent()
	case cmdMakeShared:
		return m.makeCurrentNoteShared()
	case cmdSelectSyncProfile:
		m.openSyncProfilePicker()
		return nil
	case cmdShowSyncDetails:
		m.openCurrentSyncDebugModal()
		return nil
	case cmdResolveConflict:
		return m.openCurrentConflictCopy()
	case cmdDeleteRemoteKeepLocal:
		return m.deleteRemoteCopyCurrent()
	case cmdImportCurrent:
		return m.importCurrentRemoteNote()
	case cmdImportAll:
		return m.importAllRemoteNotes()
	case cmdSyncNow:
		return m.startImmediateSync()
	case cmdShowSyncTimeline:
		return m.openSyncTimeline()
	case cmdToggleEncryption:
		m.armToggleEncryption()
		return nil
	case cmdToggleTodo:
		return m.toggleCurrentPreviewTodo()
	case cmdAddTodo:
		m.armAddTodoItem()
		return nil
	case cmdDeleteTodo:
		return m.deleteCurrentPreviewTodo()
	case cmdEditTodo:
		return m.armEditCurrentPreviewTodo()
	case cmdSetTodoDueDate:
		m.armSetCurrentTodoDueDate()
		return nil
	case cmdSetTodoPriority:
		m.armSetCurrentTodoPriority()
		return nil
	}
	return nil
}

func (m Model) renderCommandPaletteModal() string {
	modalWidth, innerWidth := m.modalDimensions(68, 104)
	total := len(m.commandPaletteFiltered)
	groups := m.paletteVisibleSectionCount()

	// Compute scroll so the cursor row is always visible.
	scroll := max(0, m.commandPaletteCursor-paletteMaxVisible+1)
	scroll = max(0, min(scroll, max(0, total-paletteMaxVisible)))

	titleLeft := lipgloss.NewStyle().
		Foreground(modalAccentColor).
		Background(modalBgColor).
		Bold(true).
		Render("Command palette")
	countTextValue := fmt.Sprintf("%d results", total)
	if groups > 0 {
		countTextValue = fmt.Sprintf("%d results · %d groups", total, groups)
	}
	countText := lipgloss.NewStyle().
		Foreground(modalMutedColor).
		Background(modalBgColor).
		Render(countTextValue)
	gapSize := max(0, innerWidth-lipgloss.Width(titleLeft)-lipgloss.Width(countText))
	gapStr := lipgloss.NewStyle().Background(modalBgColor).Render(strings.Repeat(" ", gapSize))
	titleRow := fillWidthBackground(titleLeft+gapStr+countText, innerWidth, modalBgColor)

	inputRow := m.renderCommandPaletteInputRow(innerWidth)

	divider := lipgloss.NewStyle().
		Width(innerWidth).
		Foreground(modalMutedColor).
		Background(modalBgColor).
		Render(strings.Repeat("─", innerWidth))

	var resultLines []string
	if total == 0 {
		emptyText := "No matches"
		if query := strings.TrimSpace(m.commandPaletteInput.Value()); query != "" {
			emptyText = fmt.Sprintf("No matches for %q", query)
		}
		empty := lipgloss.NewStyle().
			Width(innerWidth).
			Foreground(modalMutedColor).
			Background(modalBgColor).
			Render(emptyText)
		resultLines = append(resultLines, empty)
	} else {
		end := min(scroll+paletteMaxVisible, total)
		currentSection := paletteSection(-1)
		for i := scroll; i < end; i++ {
			item := m.commandPaletteFiltered[i]
			if i == scroll || item.section != currentSection {
				resultLines = append(resultLines, m.renderPaletteSectionHeader(item.section, innerWidth))
				currentSection = item.section
			}
			resultLines = append(resultLines, m.renderPaletteRow(item, i, innerWidth))
		}
		if scroll > 0 {
			indicator := lipgloss.NewStyle().
				Width(innerWidth).Align(lipgloss.Center).
				Foreground(modalMutedColor).Background(modalBgColor).
				Render(fmt.Sprintf("↑ %d more", scroll))
			resultLines = append([]string{indicator}, resultLines...)
		}
		if end < total {
			indicator := lipgloss.NewStyle().
				Width(innerWidth).Align(lipgloss.Center).
				Foreground(modalMutedColor).Background(modalBgColor).
				Render(fmt.Sprintf("↓ %d more", total-end))
			resultLines = append(resultLines, indicator)
		}
	}

	resultsBlock := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(strings.Join(resultLines, "\n"))

	footer := m.renderModalFooter("tab complete · ↑↓ navigate · enter open/run · esc cancel", innerWidth)

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			titleRow,
			m.renderModalBlank(innerWidth),
			inputRow,
			m.renderModalBlank(innerWidth),
			divider,
			resultsBlock,
			divider,
			m.renderModalBlank(innerWidth),
			footer,
		))

	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) paletteVisibleSectionCount() int {
	if len(m.commandPaletteFiltered) == 0 {
		return 0
	}
	count := 1
	prev := m.commandPaletteFiltered[0].section
	for _, item := range m.commandPaletteFiltered[1:] {
		if item.section != prev {
			count++
			prev = item.section
		}
	}
	return count
}

func (m Model) renderPaletteSectionHeader(section paletteSection, innerWidth int) string {
	fg := modalMutedColor
	if section == paletteSectionSuggested {
		fg = modalAccentColor
	}
	return lipgloss.NewStyle().
		Width(innerWidth).
		Foreground(fg).
		Background(modalBgColor).
		Bold(section == paletteSectionSuggested).
		Render(paletteSectionTitle(section))
}

func (m Model) renderCommandPaletteInputRow(innerWidth int) string {
	inputCopy := m.commandPaletteInput
	inputCopy.Prompt = ""
	fieldOuterWidth := max(12, innerWidth)
	fieldContentWidth := max(8, fieldOuterWidth-4)
	prefix := lipgloss.NewStyle().
		Foreground(modalAccentColor).
		Background(modalBgColor).
		Bold(true).
		Render("> ")
	inputCopy.Width = max(1, fieldContentWidth-lipgloss.Width(prefix))
	inputCopy.TextStyle = lipgloss.NewStyle().Foreground(modalTextColor).Background(modalBgColor)
	inputCopy.PlaceholderStyle = lipgloss.NewStyle().Foreground(modalMutedColor).Background(modalBgColor)
	inputCopy.Cursor.Style = lipgloss.NewStyle().Foreground(modalTextColor).Background(modalTextColor)

	rawInput := strings.TrimRight(inputCopy.View(), " ")
	inputPad := max(0, fieldContentWidth-lipgloss.Width(prefix)-lipgloss.Width(rawInput))
	inputView := prefix + rawInput + lipgloss.NewStyle().
		Width(inputPad).
		Background(modalBgColor).
		Render(strings.Repeat(" ", inputPad))

	inputField := lipgloss.NewStyle().
		Width(fieldContentWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(modalAccentColor).
		BorderBackground(modalBgColor).
		Background(modalBgColor).
		Padding(0, 1).
		Render(inputView)

	return lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(inputField)
}

func (m Model) renderPaletteRow(item paletteItem, idx, innerWidth int) string {
	selected := idx == m.commandPaletteCursor
	bg := modalBgColor
	titleFg := modalTextColor
	subFg := modalMutedColor
	cursorFg := modalMutedColor
	badgeFg := modalMutedColor
	if selected {
		bg = selectedBgColor
		titleFg = selectedFgColor
		subFg = selectedFgColor
		cursorFg = selectedFgColor
		badgeFg = selectedFgColor
	} else if item.kind == paletteKindCommand {
		badgeFg = modalAccentColor
	}

	cursor := "  "
	if selected {
		cursor = "› "
	}
	badgeText := paletteBadgeLabel(item)
	cursorW := lipgloss.Width(cursor)
	badgeW := lipgloss.Width(badgeText)
	subWidth := max(12, min(30, innerWidth/3))
	titleWidth := max(8, innerWidth-cursorW-badgeW-subWidth-4)

	titleStr := paletteTruncate(item.title, titleWidth)
	subStr := paletteTruncateLeft(item.sub, subWidth)

	return lipgloss.NewStyle().Foreground(cursorFg).Background(bg).Render(cursor) +
		lipgloss.NewStyle().Foreground(badgeFg).Background(bg).Render(badgeText) +
		lipgloss.NewStyle().Background(bg).Render(" ") +
		lipgloss.NewStyle().Width(titleWidth).Foreground(titleFg).Background(bg).Render(titleStr) +
		lipgloss.NewStyle().Width(3).Background(bg).Render("   ") +
		lipgloss.NewStyle().Width(subWidth).Align(lipgloss.Right).Foreground(subFg).Background(bg).Render(subStr)
}

func paletteBadgeLabel(item paletteItem) string {
	switch item.kind {
	case paletteKindCommand:
		return "CMD "
	case paletteKindTempNote:
		return "TEMP"
	default:
		return "NOTE"
	}
}

// paletteTruncate truncates s to at most maxWidth visual characters.
func paletteTruncate(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// paletteTruncateLeft truncates s from the left to at most maxWidth visual characters.
func paletteTruncateLeft(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width("…"+string(runes)) > maxWidth {
		runes = runes[1:]
	}
	return "…" + string(runes)
}
