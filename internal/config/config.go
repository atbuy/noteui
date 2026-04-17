// Package config loads and validates the noteui configuration file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Dashboard        bool                       `toml:"dashboard"`
	DefaultWorkspace string                     `toml:"default_workspace"`
	Workspaces       map[string]WorkspaceConfig `toml:"workspaces"`
	Theme            ThemeConfig                `toml:"theme"`
	Typography       TypographyConfig           `toml:"typography"`
	Icons            IconsConfig                `toml:"icons"`
	Modal            ModalConfig                `toml:"modal"`
	Preview          PreviewConfig              `toml:"preview"`
	Keys             KeysConfig                 `toml:"keys"`
	Sync             SyncConfig                 `toml:"sync"`
	DailyNotes       DailyNotesConfig           `toml:"daily_notes"`
}

type DailyNotesConfig struct {
	Dir      string `toml:"dir"`
	Template string `toml:"template"`
}

type WorkspaceConfig struct {
	Root           string `toml:"root"`
	Label          string `toml:"label"`
	SyncRemoteRoot string `toml:"sync_remote_root"`
}

type StartupWorkspace struct {
	Name           string
	Label          string
	Root           string
	Override       bool
	NeedsSelection bool
}

type SyncConfig struct {
	DefaultProfile string                 `toml:"default_profile"`
	Profiles       map[string]SyncProfile `toml:"profiles"`
}

type SyncProfile struct {
	Kind        string `toml:"kind"`
	SSHHost     string `toml:"ssh_host"`
	RemoteRoot  string `toml:"remote_root"`
	RemoteBin   string `toml:"remote_bin"`
	WebDAVURL   string `toml:"webdav_url"`
	Auth        string `toml:"auth"`
	UsernameEnv string `toml:"username_env"`
	PasswordEnv string `toml:"password_env"`
	TokenEnv    string `toml:"token_env"`
}

type ThemeConfig struct {
	Name string `toml:"name"`

	BgColor           string `toml:"bg_color"`
	PanelBgColor      string `toml:"panel_bg_color"`
	BorderColor       string `toml:"border_color"`
	FocusBorderColor  string `toml:"focus_border_color"`
	AccentColor       string `toml:"accent_color"`
	AccentSoftColor   string `toml:"accent_soft_color"`
	TextColor         string `toml:"text_color"`
	MutedColor        string `toml:"muted_color"`
	SubtleColor       string `toml:"subtle_color"`
	ChipBgColor       string `toml:"chip_bg_color"`
	InlineCodeBgColor string `toml:"inline_code_bg_color"`
	PinnedNoteColor   string `toml:"pinned_note_color"`
	SyncedNoteColor   string `toml:"synced_note_color"`
	UnsyncedNoteColor string `toml:"unsynced_note_color"`
	SyncingNoteColor  string `toml:"syncing_note_color"`
	SharedNoteColor   string `toml:"shared_note_color"`
	MarkedItemColor   string `toml:"marked_item_color"`
	ErrorColor        string `toml:"error_color"`
	SuccessColor      string `toml:"success_color"`
	SelectedBgColor   string `toml:"selected_bg_color"`
	SelectedFgColor   string `toml:"selected_fg_color"`
	HighlightBgColor  string `toml:"highlight_bg_color"`

	BorderStyle   string `toml:"border_style"`
	AppPaddingX   int    `toml:"app_padding_x"`
	AppPaddingY   int    `toml:"app_padding_y"`
	PanelPaddingX int    `toml:"panel_padding_x"`
	PanelPaddingY int    `toml:"panel_padding_y"`
}

type TypographyConfig struct {
	BoldTitleBar    bool `toml:"bold_title_bar"`
	BoldPanelTitles bool `toml:"bold_panel_titles"`
	BoldHeaders     bool `toml:"bold_headers"`
	BoldSelected    bool `toml:"bold_selected"`
	BoldModalTitles bool `toml:"bold_modal_titles"`
}

type IconsConfig struct {
	CategoryExpanded  string `toml:"category_expanded"`
	CategoryCollapsed string `toml:"category_collapsed"`
	CategoryLeaf      string `toml:"category_leaf"`
	Note              string `toml:"note"`
}

type ModalConfig struct {
	BgColor     string `toml:"bg_color"`
	BorderColor string `toml:"border_color"`
	TitleColor  string `toml:"title_color"`
	TextColor   string `toml:"text_color"`
	MutedColor  string `toml:"muted_color"`
	AccentColor string `toml:"accent_color"`
	ErrorColor  string `toml:"error_color"`
	BorderStyle string `toml:"border_style"`
	PaddingX    int    `toml:"padding_x"`
	PaddingY    int    `toml:"padding_y"`
}

type PreviewConfig struct {
	RenderMarkdown  bool     `toml:"render_markdown"`
	DisablePaths    []string `toml:"disable_paths"`
	Style           string   `toml:"style"`
	SyntaxHighlight bool     `toml:"syntax_highlight"`
	CodeStyle       string   `toml:"code_style"`
	Privacy         bool     `toml:"privacy"`
	LineNumbers     bool     `toml:"line_numbers"`
}

type KeysConfig struct {
	Open                     []string `toml:"open"`
	Refresh                  []string `toml:"refresh"`
	Quit                     []string `toml:"quit"`
	Focus                    []string `toml:"focus"`
	NewNote                  []string `toml:"new_note"`
	NewTemporaryNote         []string `toml:"new_temporary_note"`
	NewTodoList              []string `toml:"new_todo_list"`
	Search                   []string `toml:"search"`
	ShowHelp                 []string `toml:"show_help"`
	ShowPins                 []string `toml:"show_pins"`
	ShowTodos                []string `toml:"show_todos"`
	CreateCategory           []string `toml:"create_category"`
	ToggleCategory           []string `toml:"toggle_category"`
	Delete                   []string `toml:"delete"`
	Move                     []string `toml:"move"`
	Rename                   []string `toml:"rename"`
	AddTag                   []string `toml:"add_tag"`
	ToggleSelect             []string `toml:"toggle_select"`
	ClearMarks               []string `toml:"clear_marks"`
	Pin                      []string `toml:"pin"`
	PromoteTemporary         []string `toml:"promote_temporary"`
	ArchiveTemporary         []string `toml:"archive_temporary"`
	MoveToTemporary          []string `toml:"move_to_temporary"`
	ToggleSync               []string `toml:"toggle_sync"`
	MakeShared               []string `toml:"make_shared"`
	ToggleTemporary          []string `toml:"toggle_temporary"`
	CommandPalette           []string `toml:"command_palette"`
	SelectWorkspace          []string `toml:"select_workspace"`
	SelectSyncProfile        []string `toml:"select_sync_profile"`
	OpenConflictCopy         []string `toml:"open_conflict_copy"`
	ShowSyncDebug            []string `toml:"show_sync_debug"`
	ShowSyncTimeline         []string `toml:"show_sync_timeline"`
	DeleteRemoteKeepLocal    []string `toml:"delete_remote_keep_local"`
	SyncImportCurrent        []string `toml:"sync_import_current"`
	SyncImport               []string `toml:"sync_import"`
	UndoDelete               []string `toml:"undo_delete"`
	TogglePreviewPrivacy     []string `toml:"toggle_preview_privacy"`
	TogglePreviewLineNumbers []string `toml:"toggle_preview_line_numbers"`
	SortKey                  []string `toml:"sort_key"`
	SortByName               []string `toml:"sort_by_name"`
	SortByModified           []string `toml:"sort_by_modified"`
	SortByCreated            []string `toml:"sort_by_created"`
	SortBySize               []string `toml:"sort_by_size"`
	SortReverse              []string `toml:"sort_reverse"`
	ScrollHalfPageUp         []string `toml:"scroll_half_page_up"`
	ScrollHalfPageDown       []string `toml:"scroll_half_page_down"`
	NextMatch                []string `toml:"next_match"`
	PrevMatch                []string `toml:"prev_match"`
	MoveUp                   []string `toml:"move_up"`
	MoveDown                 []string `toml:"move_down"`
	CollapseCategory         []string `toml:"collapse_category"`
	ExpandCategory           []string `toml:"expand_category"`
	JumpBottom               []string `toml:"jump_bottom"`
	PendingG                 []string `toml:"pending_g"`
	BracketForward           []string `toml:"bracket_forward"`
	BracketBackward          []string `toml:"bracket_backward"`
	HeadingJumpKey           []string `toml:"heading_jump_key"`
	TodoKey                  []string `toml:"todo_key"`
	TodoAdd                  []string `toml:"todo_add"`
	TodoDelete               []string `toml:"todo_delete"`
	TodoEdit                 []string `toml:"todo_edit"`
	TodoDueDate              []string `toml:"todo_due_date"`
	TodoPriority             []string `toml:"todo_priority"`
	PendingZ                 []string `toml:"pending_z"`
	DeleteConfirm            []string `toml:"delete_confirm"`
	ScrollPageDown           []string `toml:"scroll_page_down"`
	ScrollPageUp             []string `toml:"scroll_page_up"`
	ToggleEncryption         []string `toml:"toggle_encryption"`
	NoteHistory              []string `toml:"note_history"`
	TrashBrowser             []string `toml:"trash_browser"`
	NewTemplate              []string `toml:"new_template"`
	EditTemplates            []string `toml:"edit_templates"`
	OpenDailyNote            []string `toml:"open_daily_note"`
	LinkKey                  []string `toml:"link_key"`
	FollowLink               []string `toml:"follow_link"`
	ShowThemePicker          []string `toml:"show_theme_picker"`
}

func Default() Config {
	return Config{
		Dashboard: true,
		Theme: ThemeConfig{
			Name:          "default",
			BorderStyle:   "rounded",
			AppPaddingX:   2,
			AppPaddingY:   1,
			PanelPaddingX: 1,
			PanelPaddingY: 0,
		},
		Typography: TypographyConfig{
			BoldTitleBar:    true,
			BoldPanelTitles: true,
			BoldHeaders:     true,
			BoldSelected:    true,
			BoldModalTitles: true,
		},
		Icons: IconsConfig{
			CategoryExpanded:  "▾",
			CategoryCollapsed: "▸",
			CategoryLeaf:      "•",
			Note:              "·",
		},
		Modal: ModalConfig{
			BorderStyle: "rounded",
			PaddingX:    2,
			PaddingY:    1,
		},
		Preview: PreviewConfig{
			RenderMarkdown:  true,
			DisablePaths:    nil,
			Style:           "dark",
			SyntaxHighlight: true,
			CodeStyle:       "monokai",
			Privacy:         false,
			LineNumbers:     true,
		},
		Keys: KeysConfig{
			ToggleSync:            []string{"S"},
			ShowSyncDebug:         []string{"ctrl+e"},
			DeleteRemoteKeepLocal: []string{"U"},
			SyncImportCurrent:     []string{"i"},
			SyncImport:            []string{"I"},
			NoteHistory:           []string{"H"},
		},
		DailyNotes: DailyNotesConfig{
			Dir: "daily",
		},
	}
}

func Load() (Config, error) {
	cfg := Default()

	path := os.Getenv("NOTEUI_CONFIG")
	if strings.TrimSpace(path) == "" {
		userCfgDir, err := os.UserConfigDir()
		if err != nil {
			return cfg, err
		}
		path = filepath.Join(userCfgDir, "noteui", "config.toml")
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	} else if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	md, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return decodeValidPrefixConfig(data), fmt.Errorf("config parse error: %w", err)
	}

	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, k := range undecoded {
			keys = append(keys, k.String())
		}
		sort.Strings(keys)
		return cfg, fmt.Errorf("unknown config key(s): %s", strings.Join(keys, ", "))
	}

	if err := Validate(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func decodeValidPrefixConfig(raw []byte) Config {
	best := Default()
	lines := strings.SplitAfter(string(raw), "\n")
	if len(lines) == 0 {
		return best
	}

	var prefix strings.Builder
	for _, line := range lines {
		prefix.WriteString(line)

		cfg := Default()
		if _, err := toml.Decode(prefix.String(), &cfg); err == nil {
			best = cfg
		}
	}

	return best
}

func Validate(cfg Config) error {
	if !IsValidThemeName(cfg.Theme.Name) {
		return fmt.Errorf(
			"unknown theme %q (valid: %s)",
			cfg.Theme.Name,
			strings.Join(ValidThemeNames(), ", "),
		)
	}

	if !isValidBorderStyle(cfg.Theme.BorderStyle) {
		return fmt.Errorf("invalid theme.border_style %q", cfg.Theme.BorderStyle)
	}

	if cfg.Modal.BorderStyle != "" && !isValidBorderStyle(cfg.Modal.BorderStyle) {
		return fmt.Errorf("invalid modal.border_style %q", cfg.Modal.BorderStyle)
	}

	if cfg.Preview.Style != "" && !isValidPreviewStyle(cfg.Preview.Style) {
		return fmt.Errorf(
			"invalid preview.style %q (valid: dark, light, auto, notty)",
			cfg.Preview.Style,
		)
	}

	if cfg.Preview.CodeStyle != "" && !isValidCodeStyle(cfg.Preview.CodeStyle) {
		return fmt.Errorf(
			"invalid preview.code_style %q (valid examples: monokai, github, dracula, swapoff, onesenterprise)",
			cfg.Preview.CodeStyle,
		)
	}

	for name, profile := range cfg.Sync.Profiles {
		if strings.TrimSpace(name) == "" {
			return errors.New("sync profile name cannot be empty")
		}
		if err := validateSyncProfile(name, profile); err != nil {
			return err
		}
	}

	if cfg.Sync.DefaultProfile != "" {
		if _, ok := cfg.Sync.Profiles[cfg.Sync.DefaultProfile]; !ok {
			return fmt.Errorf("unknown sync.default_profile %q", cfg.Sync.DefaultProfile)
		}
	}

	for name, workspace := range cfg.Workspaces {
		name = strings.TrimSpace(name)
		if name == "" {
			return errors.New("workspace name cannot be empty")
		}
		if strings.TrimSpace(workspace.Root) == "" {
			return fmt.Errorf("workspace %q is missing root", name)
		}
		if workspace.SyncRemoteRoot != "" && strings.TrimSpace(workspace.SyncRemoteRoot) == "" {
			return fmt.Errorf("workspace %q has a blank sync_remote_root; remove the field or provide a non-empty path", name)
		}
	}

	if cfg.DefaultWorkspace != "" {
		if _, ok := cfg.Workspaces[cfg.DefaultWorkspace]; !ok {
			return fmt.Errorf("unknown default_workspace %q", cfg.DefaultWorkspace)
		}
	}

	return nil
}

func SortedWorkspaceNames(cfg Config) []string {
	if len(cfg.Workspaces) == 0 {
		return nil
	}
	out := make([]string, 0, len(cfg.Workspaces))
	for name := range cfg.Workspaces {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func ResolveStartupWorkspace(cfg Config, notesRootOverride, fallbackRoot string) StartupWorkspace {
	notesRootOverride = strings.TrimSpace(notesRootOverride)
	if notesRootOverride != "" {
		return StartupWorkspace{
			Root:     filepath.Clean(notesRootOverride),
			Override: true,
		}
	}

	names := SortedWorkspaceNames(cfg)
	if len(names) == 0 {
		return StartupWorkspace{Root: filepath.Clean(strings.TrimSpace(fallbackRoot))}
	}

	if cfg.DefaultWorkspace != "" {
		workspace := cfg.Workspaces[cfg.DefaultWorkspace]
		return StartupWorkspace{
			Name:  cfg.DefaultWorkspace,
			Label: strings.TrimSpace(workspace.Label),
			Root:  filepath.Clean(strings.TrimSpace(workspace.Root)),
		}
	}

	if len(names) == 1 {
		name := names[0]
		workspace := cfg.Workspaces[name]
		return StartupWorkspace{
			Name:  name,
			Label: strings.TrimSpace(workspace.Label),
			Root:  filepath.Clean(strings.TrimSpace(workspace.Root)),
		}
	}

	return StartupWorkspace{NeedsSelection: true}
}

func ValidThemeNames() []string {
	return []string{
		"default",
		"nord",
		"gruvbox",
		"catppuccin",
		"catppuccin-mocha",
		"mocha",
		"catppuccin-latte",
		"latte",
		"solarized-light",
		"paper",
		"onedark",
		"kanagawa",
		"dracula",
		"everforest",
		"everforest-dark",
		"tokyo-night-storm",
		"tokyonight-storm",
		"github-light",
		"github-dark",
		"carbonfox",
		"crimson",
		"dusk",
		"rose-pine",
		"rosepine",
		"rose_pine",
		"monokai",
		"solarized-dark",
		"solarized",
		"ayu-dark",
		"ayu",
		"material",
		"material-dark",
		"nightfox",
	}
}

func IsValidThemeName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return slices.Contains(ValidThemeNames(), name)
}

const (
	SyncKindSSH    = "ssh"
	SyncKindWebDAV = "webdav"
	SyncAuthBasic  = "basic"
	SyncAuthBearer = "bearer"
	SyncAuthNone   = "none"
)

func ResolvedKind(p SyncProfile) string {
	kind := strings.ToLower(strings.TrimSpace(p.Kind))
	if kind == "" {
		return SyncKindSSH
	}
	return kind
}

func validateSyncProfile(name string, p SyncProfile) error {
	kind := ResolvedKind(p)
	switch kind {
	case SyncKindSSH:
		if strings.TrimSpace(p.SSHHost) == "" {
			return fmt.Errorf("sync profile %q is missing ssh_host", name)
		}
		if strings.TrimSpace(p.RemoteRoot) == "" {
			return fmt.Errorf("sync profile %q is missing remote_root", name)
		}
		if strings.TrimSpace(p.RemoteBin) == "" {
			return fmt.Errorf("sync profile %q is missing remote_bin", name)
		}
	case SyncKindWebDAV:
		if strings.TrimSpace(p.WebDAVURL) == "" {
			return fmt.Errorf("sync profile %q is missing webdav_url", name)
		}
		u := strings.TrimSpace(p.WebDAVURL)
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			return fmt.Errorf("sync profile %q webdav_url must start with http:// or https://", name)
		}
		auth := strings.ToLower(strings.TrimSpace(p.Auth))
		if auth == "" {
			auth = SyncAuthBasic
		}
		switch auth {
		case SyncAuthBasic:
			if strings.TrimSpace(p.UsernameEnv) == "" {
				return fmt.Errorf("sync profile %q is missing username_env (required for basic auth)", name)
			}
			if strings.TrimSpace(p.PasswordEnv) == "" {
				return fmt.Errorf("sync profile %q is missing password_env (required for basic auth)", name)
			}
		case SyncAuthBearer:
			if strings.TrimSpace(p.TokenEnv) == "" {
				return fmt.Errorf("sync profile %q is missing token_env (required for bearer auth)", name)
			}
		case SyncAuthNone:
		default:
			return fmt.Errorf("sync profile %q has unknown auth mode %q (valid: basic, bearer, none)", name, auth)
		}
	default:
		return fmt.Errorf("sync profile %q has unknown kind %q (valid: ssh, webdav)", name, kind)
	}
	return nil
}

func isValidBorderStyle(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "rounded", "normal", "double", "thick", "hidden":
		return true
	default:
		return false
	}
}

func isValidPreviewStyle(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "dark", "light", "auto", "notty":
		return true
	default:
		return false
	}
}

func isValidCodeStyle(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "",
		"monokai",
		"github",
		"dracula",
		"swapoff",
		"onesenterprise",
		"native",
		"paraiso-dark",
		"paraiso-light":
		return true
	default:
		return false
	}
}
