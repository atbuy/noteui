package config

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"atbuy/noteui/internal/fsutil"
)

func ResolvePath() (string, error) {
	path := os.Getenv("NOTEUI_CONFIG")
	if strings.TrimSpace(path) != "" {
		return path, nil
	}

	userCfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCfgDir, "noteui", "config.toml"), nil
}

func Save(cfg Config) error {
	if err := Validate(cfg); err != nil {
		return err
	}

	path, err := ResolvePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := marshalSparseConfig(cfg)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, data, 0o644)
}

// SaveTheme updates theme.name in the config file and returns the previous
// theme name and the path of the file that was written.
func SaveTheme(name string) (oldName, configPath string, err error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if !IsValidThemeName(name) {
		return "", "", Validate(Config{Theme: ThemeConfig{Name: name}})
	}

	path, err := ResolvePath()
	if err != nil {
		return "", "", err
	}

	cfg := loadForMutation(path)
	oldName = strings.TrimSpace(cfg.Theme.Name)
	if oldName == "" {
		oldName = Default().Theme.Name
	}
	if err := updateConfigString(path, "theme", "name", name, Default().Theme.Name); err != nil {
		return "", "", err
	}
	return oldName, path, nil
}

func SaveDefaultSyncProfile(profile string) (Config, string, error) {
	profile = strings.TrimSpace(profile)

	path, err := ResolvePath()
	if err != nil {
		return Config{}, "", err
	}

	cfg := loadForMutation(path)
	cfg.Sync.DefaultProfile = profile
	if err := updateConfigString(path, "sync", "default_profile", profile, Default().Sync.DefaultProfile); err != nil {
		return Config{}, "", err
	}
	return cfg, path, nil
}

func marshalSparseConfig(cfg Config) ([]byte, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(sparseConfigMap(cfg)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sparseConfigMap(cfg Config) map[string]any {
	defaults := Default()
	out := make(map[string]any)

	if cfg.Dashboard != defaults.Dashboard {
		out["dashboard"] = cfg.Dashboard
	}
	if cfg.DefaultWorkspace != defaults.DefaultWorkspace && strings.TrimSpace(cfg.DefaultWorkspace) != "" {
		out["default_workspace"] = strings.TrimSpace(cfg.DefaultWorkspace)
	}

	if theme := sparseThemeConfig(cfg.Theme, defaults.Theme); len(theme) > 0 {
		out["theme"] = theme
	}
	if typography := sparseTypographyConfig(cfg.Typography, defaults.Typography); len(typography) > 0 {
		out["typography"] = typography
	}
	if icons := sparseIconsConfig(cfg.Icons, defaults.Icons); len(icons) > 0 {
		out["icons"] = icons
	}
	if modal := sparseModalConfig(cfg.Modal, defaults.Modal); len(modal) > 0 {
		out["modal"] = modal
	}
	if preview := sparsePreviewConfig(cfg.Preview, defaults.Preview); len(preview) > 0 {
		out["preview"] = preview
	}
	if keys := sparseKeysConfig(cfg.Keys, defaults.Keys); len(keys) > 0 {
		out["keys"] = keys
	}
	if syncCfg := sparseSyncConfig(cfg.Sync, defaults.Sync); len(syncCfg) > 0 {
		out["sync"] = syncCfg
	}
	if dailyNotes := sparseDailyNotesConfig(cfg.DailyNotes, defaults.DailyNotes); len(dailyNotes) > 0 {
		out["daily_notes"] = dailyNotes
	}
	if workspaces := sparseWorkspacesConfig(cfg.Workspaces); len(workspaces) > 0 {
		out["workspaces"] = workspaces
	}

	return out
}

func sparseThemeConfig(cfg, defaults ThemeConfig) map[string]any {
	out := make(map[string]any)
	addStringIfDifferent(out, "name", cfg.Name, defaults.Name)
	addStringIfDifferent(out, "bg_color", cfg.BgColor, defaults.BgColor)
	addStringIfDifferent(out, "panel_bg_color", cfg.PanelBgColor, defaults.PanelBgColor)
	addStringIfDifferent(out, "border_color", cfg.BorderColor, defaults.BorderColor)
	addStringIfDifferent(out, "focus_border_color", cfg.FocusBorderColor, defaults.FocusBorderColor)
	addStringIfDifferent(out, "accent_color", cfg.AccentColor, defaults.AccentColor)
	addStringIfDifferent(out, "accent_soft_color", cfg.AccentSoftColor, defaults.AccentSoftColor)
	addStringIfDifferent(out, "text_color", cfg.TextColor, defaults.TextColor)
	addStringIfDifferent(out, "muted_color", cfg.MutedColor, defaults.MutedColor)
	addStringIfDifferent(out, "subtle_color", cfg.SubtleColor, defaults.SubtleColor)
	addStringIfDifferent(out, "chip_bg_color", cfg.ChipBgColor, defaults.ChipBgColor)
	addStringIfDifferent(out, "inline_code_bg_color", cfg.InlineCodeBgColor, defaults.InlineCodeBgColor)
	addStringIfDifferent(out, "pinned_note_color", cfg.PinnedNoteColor, defaults.PinnedNoteColor)
	addStringIfDifferent(out, "synced_note_color", cfg.SyncedNoteColor, defaults.SyncedNoteColor)
	addStringIfDifferent(out, "unsynced_note_color", cfg.UnsyncedNoteColor, defaults.UnsyncedNoteColor)
	addStringIfDifferent(out, "syncing_note_color", cfg.SyncingNoteColor, defaults.SyncingNoteColor)
	addStringIfDifferent(out, "shared_note_color", cfg.SharedNoteColor, defaults.SharedNoteColor)
	addStringIfDifferent(out, "marked_item_color", cfg.MarkedItemColor, defaults.MarkedItemColor)
	addStringIfDifferent(out, "error_color", cfg.ErrorColor, defaults.ErrorColor)
	addStringIfDifferent(out, "success_color", cfg.SuccessColor, defaults.SuccessColor)
	addStringIfDifferent(out, "selected_bg_color", cfg.SelectedBgColor, defaults.SelectedBgColor)
	addStringIfDifferent(out, "selected_fg_color", cfg.SelectedFgColor, defaults.SelectedFgColor)
	addStringIfDifferent(out, "highlight_bg_color", cfg.HighlightBgColor, defaults.HighlightBgColor)
	addStringIfDifferent(out, "border_style", cfg.BorderStyle, defaults.BorderStyle)
	addIntIfDifferent(out, "app_padding_x", cfg.AppPaddingX, defaults.AppPaddingX)
	addIntIfDifferent(out, "app_padding_y", cfg.AppPaddingY, defaults.AppPaddingY)
	addIntIfDifferent(out, "panel_padding_x", cfg.PanelPaddingX, defaults.PanelPaddingX)
	addIntIfDifferent(out, "panel_padding_y", cfg.PanelPaddingY, defaults.PanelPaddingY)
	return out
}

func sparseTypographyConfig(cfg, defaults TypographyConfig) map[string]any {
	out := make(map[string]any)
	addBoolIfDifferent(out, "bold_title_bar", cfg.BoldTitleBar, defaults.BoldTitleBar)
	addBoolIfDifferent(out, "bold_panel_titles", cfg.BoldPanelTitles, defaults.BoldPanelTitles)
	addBoolIfDifferent(out, "bold_headers", cfg.BoldHeaders, defaults.BoldHeaders)
	addBoolIfDifferent(out, "bold_selected", cfg.BoldSelected, defaults.BoldSelected)
	addBoolIfDifferent(out, "bold_modal_titles", cfg.BoldModalTitles, defaults.BoldModalTitles)
	return out
}

func sparseIconsConfig(cfg, defaults IconsConfig) map[string]any {
	out := make(map[string]any)
	addStringIfDifferent(out, "category_expanded", cfg.CategoryExpanded, defaults.CategoryExpanded)
	addStringIfDifferent(out, "category_collapsed", cfg.CategoryCollapsed, defaults.CategoryCollapsed)
	addStringIfDifferent(out, "category_leaf", cfg.CategoryLeaf, defaults.CategoryLeaf)
	addStringIfDifferent(out, "note", cfg.Note, defaults.Note)
	return out
}

func sparseModalConfig(cfg, defaults ModalConfig) map[string]any {
	out := make(map[string]any)
	addStringIfDifferent(out, "bg_color", cfg.BgColor, defaults.BgColor)
	addStringIfDifferent(out, "border_color", cfg.BorderColor, defaults.BorderColor)
	addStringIfDifferent(out, "title_color", cfg.TitleColor, defaults.TitleColor)
	addStringIfDifferent(out, "text_color", cfg.TextColor, defaults.TextColor)
	addStringIfDifferent(out, "muted_color", cfg.MutedColor, defaults.MutedColor)
	addStringIfDifferent(out, "accent_color", cfg.AccentColor, defaults.AccentColor)
	addStringIfDifferent(out, "error_color", cfg.ErrorColor, defaults.ErrorColor)
	addStringIfDifferent(out, "border_style", cfg.BorderStyle, defaults.BorderStyle)
	addIntIfDifferent(out, "padding_x", cfg.PaddingX, defaults.PaddingX)
	addIntIfDifferent(out, "padding_y", cfg.PaddingY, defaults.PaddingY)
	return out
}

func sparsePreviewConfig(cfg, defaults PreviewConfig) map[string]any {
	out := make(map[string]any)
	addBoolIfDifferent(out, "render_markdown", cfg.RenderMarkdown, defaults.RenderMarkdown)
	addStringsIfDifferent(out, "disable_paths", cfg.DisablePaths, defaults.DisablePaths)
	addStringIfDifferent(out, "style", cfg.Style, defaults.Style)
	addBoolIfDifferent(out, "syntax_highlight", cfg.SyntaxHighlight, defaults.SyntaxHighlight)
	addStringIfDifferent(out, "code_style", cfg.CodeStyle, defaults.CodeStyle)
	addBoolIfDifferent(out, "privacy", cfg.Privacy, defaults.Privacy)
	addBoolIfDifferent(out, "line_numbers", cfg.LineNumbers, defaults.LineNumbers)
	return out
}

func sparseKeysConfig(cfg, defaults KeysConfig) map[string]any {
	out := make(map[string]any)
	addStringsIfDifferent(out, "open", cfg.Open, defaults.Open)
	addStringsIfDifferent(out, "refresh", cfg.Refresh, defaults.Refresh)
	addStringsIfDifferent(out, "quit", cfg.Quit, defaults.Quit)
	addStringsIfDifferent(out, "focus", cfg.Focus, defaults.Focus)
	addStringsIfDifferent(out, "new_note", cfg.NewNote, defaults.NewNote)
	addStringsIfDifferent(out, "new_temporary_note", cfg.NewTemporaryNote, defaults.NewTemporaryNote)
	addStringsIfDifferent(out, "new_todo_list", cfg.NewTodoList, defaults.NewTodoList)
	addStringsIfDifferent(out, "search", cfg.Search, defaults.Search)
	addStringsIfDifferent(out, "show_help", cfg.ShowHelp, defaults.ShowHelp)
	addStringsIfDifferent(out, "show_pins", cfg.ShowPins, defaults.ShowPins)
	addStringsIfDifferent(out, "show_todos", cfg.ShowTodos, defaults.ShowTodos)
	addStringsIfDifferent(out, "create_category", cfg.CreateCategory, defaults.CreateCategory)
	addStringsIfDifferent(out, "toggle_category", cfg.ToggleCategory, defaults.ToggleCategory)
	addStringsIfDifferent(out, "delete", cfg.Delete, defaults.Delete)
	addStringsIfDifferent(out, "move", cfg.Move, defaults.Move)
	addStringsIfDifferent(out, "rename", cfg.Rename, defaults.Rename)
	addStringsIfDifferent(out, "add_tag", cfg.AddTag, defaults.AddTag)
	addStringsIfDifferent(out, "toggle_select", cfg.ToggleSelect, defaults.ToggleSelect)
	addStringsIfDifferent(out, "clear_marks", cfg.ClearMarks, defaults.ClearMarks)
	addStringsIfDifferent(out, "pin", cfg.Pin, defaults.Pin)
	addStringsIfDifferent(out, "promote_temporary", cfg.PromoteTemporary, defaults.PromoteTemporary)
	addStringsIfDifferent(out, "archive_temporary", cfg.ArchiveTemporary, defaults.ArchiveTemporary)
	addStringsIfDifferent(out, "move_to_temporary", cfg.MoveToTemporary, defaults.MoveToTemporary)
	addStringsIfDifferent(out, "toggle_sync", cfg.ToggleSync, defaults.ToggleSync)
	addStringsIfDifferent(out, "make_shared", cfg.MakeShared, defaults.MakeShared)
	addStringsIfDifferent(out, "toggle_temporary", cfg.ToggleTemporary, defaults.ToggleTemporary)
	addStringsIfDifferent(out, "command_palette", cfg.CommandPalette, defaults.CommandPalette)
	addStringsIfDifferent(out, "select_workspace", cfg.SelectWorkspace, defaults.SelectWorkspace)
	addStringsIfDifferent(out, "select_sync_profile", cfg.SelectSyncProfile, defaults.SelectSyncProfile)
	addStringsIfDifferent(out, "open_conflict_copy", cfg.OpenConflictCopy, defaults.OpenConflictCopy)
	addStringsIfDifferent(out, "show_sync_debug", cfg.ShowSyncDebug, defaults.ShowSyncDebug)
	addStringsIfDifferent(out, "show_sync_timeline", cfg.ShowSyncTimeline, defaults.ShowSyncTimeline)
	addStringsIfDifferent(out, "delete_remote_keep_local", cfg.DeleteRemoteKeepLocal, defaults.DeleteRemoteKeepLocal)
	addStringsIfDifferent(out, "sync_import_current", cfg.SyncImportCurrent, defaults.SyncImportCurrent)
	addStringsIfDifferent(out, "sync_import", cfg.SyncImport, defaults.SyncImport)
	addStringsIfDifferent(out, "undo_delete", cfg.UndoDelete, defaults.UndoDelete)
	addStringsIfDifferent(out, "toggle_preview_privacy", cfg.TogglePreviewPrivacy, defaults.TogglePreviewPrivacy)
	addStringsIfDifferent(out, "toggle_preview_line_numbers", cfg.TogglePreviewLineNumbers, defaults.TogglePreviewLineNumbers)
	addStringsIfDifferent(out, "sort_key", cfg.SortKey, defaults.SortKey)
	addStringsIfDifferent(out, "sort_by_name", cfg.SortByName, defaults.SortByName)
	addStringsIfDifferent(out, "sort_by_modified", cfg.SortByModified, defaults.SortByModified)
	addStringsIfDifferent(out, "sort_by_created", cfg.SortByCreated, defaults.SortByCreated)
	addStringsIfDifferent(out, "sort_by_size", cfg.SortBySize, defaults.SortBySize)
	addStringsIfDifferent(out, "sort_reverse", cfg.SortReverse, defaults.SortReverse)
	addStringsIfDifferent(out, "scroll_half_page_up", cfg.ScrollHalfPageUp, defaults.ScrollHalfPageUp)
	addStringsIfDifferent(out, "scroll_half_page_down", cfg.ScrollHalfPageDown, defaults.ScrollHalfPageDown)
	addStringsIfDifferent(out, "next_match", cfg.NextMatch, defaults.NextMatch)
	addStringsIfDifferent(out, "prev_match", cfg.PrevMatch, defaults.PrevMatch)
	addStringsIfDifferent(out, "move_up", cfg.MoveUp, defaults.MoveUp)
	addStringsIfDifferent(out, "move_down", cfg.MoveDown, defaults.MoveDown)
	addStringsIfDifferent(out, "collapse_category", cfg.CollapseCategory, defaults.CollapseCategory)
	addStringsIfDifferent(out, "expand_category", cfg.ExpandCategory, defaults.ExpandCategory)
	addStringsIfDifferent(out, "jump_bottom", cfg.JumpBottom, defaults.JumpBottom)
	addStringsIfDifferent(out, "pending_g", cfg.PendingG, defaults.PendingG)
	addStringsIfDifferent(out, "bracket_forward", cfg.BracketForward, defaults.BracketForward)
	addStringsIfDifferent(out, "bracket_backward", cfg.BracketBackward, defaults.BracketBackward)
	addStringsIfDifferent(out, "heading_jump_key", cfg.HeadingJumpKey, defaults.HeadingJumpKey)
	addStringsIfDifferent(out, "todo_key", cfg.TodoKey, defaults.TodoKey)
	addStringsIfDifferent(out, "todo_add", cfg.TodoAdd, defaults.TodoAdd)
	addStringsIfDifferent(out, "todo_delete", cfg.TodoDelete, defaults.TodoDelete)
	addStringsIfDifferent(out, "todo_edit", cfg.TodoEdit, defaults.TodoEdit)
	addStringsIfDifferent(out, "todo_due_date", cfg.TodoDueDate, defaults.TodoDueDate)
	addStringsIfDifferent(out, "todo_priority", cfg.TodoPriority, defaults.TodoPriority)
	addStringsIfDifferent(out, "pending_z", cfg.PendingZ, defaults.PendingZ)
	addStringsIfDifferent(out, "delete_confirm", cfg.DeleteConfirm, defaults.DeleteConfirm)
	addStringsIfDifferent(out, "scroll_page_down", cfg.ScrollPageDown, defaults.ScrollPageDown)
	addStringsIfDifferent(out, "scroll_page_up", cfg.ScrollPageUp, defaults.ScrollPageUp)
	addStringsIfDifferent(out, "toggle_encryption", cfg.ToggleEncryption, defaults.ToggleEncryption)
	addStringsIfDifferent(out, "note_history", cfg.NoteHistory, defaults.NoteHistory)
	addStringsIfDifferent(out, "trash_browser", cfg.TrashBrowser, defaults.TrashBrowser)
	addStringsIfDifferent(out, "new_template", cfg.NewTemplate, defaults.NewTemplate)
	addStringsIfDifferent(out, "edit_templates", cfg.EditTemplates, defaults.EditTemplates)
	addStringsIfDifferent(out, "open_daily_note", cfg.OpenDailyNote, defaults.OpenDailyNote)
	addStringsIfDifferent(out, "link_key", cfg.LinkKey, defaults.LinkKey)
	addStringsIfDifferent(out, "follow_link", cfg.FollowLink, defaults.FollowLink)
	addStringsIfDifferent(out, "show_theme_picker", cfg.ShowThemePicker, defaults.ShowThemePicker)
	return out
}

func sparseSyncConfig(cfg, defaults SyncConfig) map[string]any {
	out := make(map[string]any)
	addStringIfDifferent(out, "default_profile", cfg.DefaultProfile, defaults.DefaultProfile)
	if profiles := sparseSyncProfiles(cfg.Profiles); len(profiles) > 0 {
		out["profiles"] = profiles
	}
	return out
}

func sparseSyncProfiles(items map[string]SyncProfile) map[string]any {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]any, len(items))
	for name, profile := range items {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		profileMap := make(map[string]any)
		addNonEmptyString(profileMap, "kind", profile.Kind)
		addNonEmptyString(profileMap, "ssh_host", profile.SSHHost)
		addNonEmptyString(profileMap, "remote_root", profile.RemoteRoot)
		addNonEmptyString(profileMap, "remote_bin", profile.RemoteBin)
		addNonEmptyString(profileMap, "webdav_url", profile.WebDAVURL)
		addNonEmptyString(profileMap, "auth", profile.Auth)
		addNonEmptyString(profileMap, "username_env", profile.UsernameEnv)
		addNonEmptyString(profileMap, "password_env", profile.PasswordEnv)
		if len(profileMap) > 0 {
			out[name] = profileMap
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sparseDailyNotesConfig(cfg, defaults DailyNotesConfig) map[string]any {
	out := make(map[string]any)
	addStringIfDifferent(out, "dir", cfg.Dir, defaults.Dir)
	addStringIfDifferent(out, "template", cfg.Template, defaults.Template)
	return out
}

func sparseWorkspacesConfig(items map[string]WorkspaceConfig) map[string]any {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]any, len(items))
	for name, workspace := range items {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		workspaceMap := make(map[string]any)
		addNonEmptyString(workspaceMap, "root", workspace.Root)
		addNonEmptyString(workspaceMap, "label", workspace.Label)
		addNonEmptyString(workspaceMap, "sync_remote_root", workspace.SyncRemoteRoot)
		if len(workspaceMap) > 0 {
			out[name] = workspaceMap
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func addBoolIfDifferent(out map[string]any, key string, value, defaults bool) {
	if value != defaults {
		out[key] = value
	}
}

func addIntIfDifferent(out map[string]any, key string, value, defaults int) {
	if value != defaults {
		out[key] = value
	}
}

func addStringIfDifferent(out map[string]any, key, value, defaults string) {
	if value != defaults && strings.TrimSpace(value) != "" {
		out[key] = value
	}
}

func addStringsIfDifferent(out map[string]any, key string, value, defaults []string) {
	if !slices.Equal(value, defaults) && len(value) > 0 {
		out[key] = value
	}
}

func addNonEmptyString(out map[string]any, key, value string) {
	if strings.TrimSpace(value) != "" {
		out[key] = value
	}
}

func loadForMutation(path string) Config {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	_, _ = toml.Decode(string(data), &cfg)
	return cfg
}

func updateConfigString(path, section, key, value, defaultValue string) error {
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		raw = nil
	default:
		return err
	}

	updated, changed := updateTOMLStringKey(raw, section, key, value, defaultValue)
	if !changed {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, updated, 0o644)
}

func updateTOMLStringKey(raw []byte, section, key, value, defaultValue string) ([]byte, bool) {
	newline := detectNewline(raw)
	lines := splitLinesPreserveEndings(string(raw))
	remove := strings.TrimSpace(value) == strings.TrimSpace(defaultValue)

	start, end, foundSection := findSection(lines, section)
	if !foundSection {
		if remove {
			return raw, false
		}
		lines = appendSection(lines, section, key, value, newline)
		return []byte(strings.Join(lines, "")), true
	}

	for i := start + 1; i < end; i++ {
		indent, comment, ok := matchKeyLine(lines[i], key)
		if !ok {
			continue
		}
		if remove {
			lines = append(lines[:i], lines[i+1:]...)
			lines = pruneEmptySection(lines, start, end-1)
		} else {
			lines[i] = buildKeyLine(indent, key, value, comment, lineEnding(lines[i], newline))
		}
		return []byte(strings.Join(lines, "")), true
	}

	if remove {
		return raw, false
	}

	insertAt := end
	if insertAt > 0 && !hasLineEnding(lines[insertAt-1]) {
		lines[insertAt-1] += newline
	}
	inserted := buildKeyLine("", key, value, "", newline)
	lines = append(lines[:insertAt], append([]string{inserted}, lines[insertAt:]...)...)
	return []byte(strings.Join(lines, "")), true
}

func detectNewline(raw []byte) string {
	if bytes.Contains(raw, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func splitLinesPreserveEndings(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.SplitAfter(raw, "\n")
}

func findSection(lines []string, section string) (start int, end int, found bool) {
	start = -1
	end = len(lines)
	for i, line := range lines {
		name, ok := parseTableHeader(line)
		if !ok {
			continue
		}
		if start >= 0 {
			end = i
			break
		}
		if name == section {
			start = i
			found = true
		}
	}
	if !found {
		return -1, len(lines), false
	}
	return start, end, true
}

func parseTableHeader(line string) (string, bool) {
	trimmed := strings.TrimSpace(strings.TrimRight(line, "\r\n"))
	if strings.HasPrefix(trimmed, "[[") || !strings.HasPrefix(trimmed, "[") {
		return "", false
	}
	end := strings.Index(trimmed, "]")
	if end <= 1 {
		return "", false
	}
	if strings.Contains(trimmed[1:end], "[") {
		return "", false
	}
	rest := strings.TrimSpace(trimmed[end+1:])
	if rest != "" && !strings.HasPrefix(rest, "#") {
		return "", false
	}
	return strings.TrimSpace(trimmed[1:end]), true
}

func matchKeyLine(line, key string) (indent string, comment string, ok bool) {
	base := strings.TrimRight(line, "\r\n")
	leftTrimmed := strings.TrimLeft(base, " \t")
	if leftTrimmed == "" || strings.HasPrefix(leftTrimmed, "#") {
		return "", "", false
	}
	if !strings.HasPrefix(leftTrimmed, key) {
		return "", "", false
	}
	rest := leftTrimmed[len(key):]
	rest = strings.TrimLeft(rest, " \t")
	if !strings.HasPrefix(rest, "=") {
		return "", "", false
	}
	indent = base[:len(base)-len(leftTrimmed)]
	_, comment = splitInlineComment(base)
	return indent, comment, true
}

func splitInlineComment(line string) (string, string) {
	inBasicString := false
	inLiteralString := false
	escaped := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\\':
			if inBasicString && !escaped {
				escaped = true
				continue
			}
		case '"':
			if !inLiteralString && !escaped {
				inBasicString = !inBasicString
			}
		case '\'':
			if !inBasicString && !escaped {
				inLiteralString = !inLiteralString
			}
		case '#':
			if !inBasicString && !inLiteralString {
				return line[:i], line[i:]
			}
		}
		escaped = false
	}
	return line, ""
}

func buildKeyLine(indent, key, value, comment, newline string) string {
	line := indent + key + " = " + tomlQuote(value)
	if comment != "" {
		if !strings.HasPrefix(comment, " ") && !strings.HasPrefix(comment, "\t") {
			line += " "
		}
		line += comment
	}
	return line + newline
}

func appendSection(lines []string, section, key, value, newline string) []string {
	if len(lines) > 0 {
		last := len(lines) - 1
		if !hasLineEnding(lines[last]) {
			lines[last] += newline
		}
		if strings.TrimSpace(strings.TrimRight(lines[last], "\r\n")) != "" {
			lines = append(lines, newline)
		}
	}
	lines = append(lines,
		"["+section+"]"+newline,
		buildKeyLine("", key, value, "", newline),
	)
	return lines
}

func pruneEmptySection(lines []string, start, end int) []string {
	if start < 0 || start >= len(lines) {
		return lines
	}
	if end > len(lines) {
		end = len(lines)
	}
	for i := start + 1; i < end; i++ {
		if strings.TrimSpace(strings.TrimRight(lines[i], "\r\n")) != "" {
			return lines
		}
	}
	removeEnd := end
	for removeEnd < len(lines) && strings.TrimSpace(strings.TrimRight(lines[removeEnd], "\r\n")) == "" {
		removeEnd++
	}
	return append(lines[:start], lines[removeEnd:]...)
}

func lineEnding(line, fallback string) string {
	switch {
	case strings.HasSuffix(line, "\r\n"):
		return "\r\n"
	case strings.HasSuffix(line, "\n"):
		return "\n"
	default:
		return fallback
	}
}

func hasLineEnding(line string) bool {
	return strings.HasSuffix(line, "\n")
}

func tomlQuote(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	)
	return `"` + replacer.Replace(value) + `"`
}
