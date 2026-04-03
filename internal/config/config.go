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
	Dashboard  bool             `toml:"dashboard"`
	Theme      ThemeConfig      `toml:"theme"`
	Typography TypographyConfig `toml:"typography"`
	Icons      IconsConfig      `toml:"icons"`
	Modal      ModalConfig      `toml:"modal"`
	Preview    PreviewConfig    `toml:"preview"`
	Keys       KeysConfig       `toml:"keys"`
	Sync       SyncConfig       `toml:"sync"`
}

type SyncConfig struct {
	DefaultProfile string                 `toml:"default_profile"`
	Profiles       map[string]SyncProfile `toml:"profiles"`
}

type SyncProfile struct {
	SSHHost    string `toml:"ssh_host"`
	RemoteRoot string `toml:"remote_root"`
	RemoteBin  string `toml:"remote_bin"`
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
	CreateCategory           []string `toml:"create_category"`
	ToggleCategory           []string `toml:"toggle_category"`
	Delete                   []string `toml:"delete"`
	Move                     []string `toml:"move"`
	Rename                   []string `toml:"rename"`
	AddTag                   []string `toml:"add_tag"`
	ToggleSelect             []string `toml:"toggle_select"`
	Pin                      []string `toml:"pin"`
	ToggleSync               []string `toml:"toggle_sync"`
	SelectSyncProfile        []string `toml:"select_sync_profile"`
	OpenConflictCopy         []string `toml:"open_conflict_copy"`
	DeleteRemoteKeepLocal    []string `toml:"delete_remote_keep_local"`
	SyncImportCurrent        []string `toml:"sync_import_current"`
	SyncImport               []string `toml:"sync_import"`
	TogglePreviewPrivacy     []string `toml:"toggle_preview_privacy"`
	TogglePreviewLineNumbers []string `toml:"toggle_preview_line_numbers"`
	SortToggle               []string `toml:"sort_toggle"`
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
	PendingZ                 []string `toml:"pending_z"`
	DeleteConfirm            []string `toml:"delete_confirm"`
	ScrollPageDown           []string `toml:"scroll_page_down"`
	ScrollPageUp             []string `toml:"scroll_page_up"`
	ToggleEncryption         []string `toml:"toggle_encryption"`
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
			DeleteRemoteKeepLocal: []string{"U"},
			SyncImportCurrent:     []string{"i"},
			SyncImport:            []string{"I"},
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

	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return Default(), fmt.Errorf("config parse error: %w", err)
	}

	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, k := range undecoded {
			keys = append(keys, k.String())
		}
		sort.Strings(keys)
		return Default(), fmt.Errorf("unknown config key(s): %s", strings.Join(keys, ", "))
	}

	if err := Validate(cfg); err != nil {
		return Default(), err
	}

	return cfg, nil
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
		if strings.TrimSpace(profile.SSHHost) == "" {
			return fmt.Errorf("sync profile %q is missing ssh_host", name)
		}
		if strings.TrimSpace(profile.RemoteRoot) == "" {
			return fmt.Errorf("sync profile %q is missing remote_root", name)
		}
		if strings.TrimSpace(profile.RemoteBin) == "" {
			return fmt.Errorf("sync profile %q is missing remote_bin", name)
		}
	}

	if cfg.Sync.DefaultProfile != "" {
		if _, ok := cfg.Sync.Profiles[cfg.Sync.DefaultProfile]; !ok {
			return fmt.Errorf("unknown sync.default_profile %q", cfg.Sync.DefaultProfile)
		}
	}

	return nil
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
	}
}

func IsValidThemeName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return slices.Contains(ValidThemeNames(), name)
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
