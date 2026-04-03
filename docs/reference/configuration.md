# Configuration reference

This page documents the supported `config.toml` keys for noteui.

## Config file lookup and validation

noteui loads configuration in this order:

1. If `NOTEUI_CONFIG` is set, use that file path.
2. Otherwise use `noteui/config.toml` inside the user config directory.

If the file does not exist, noteui uses defaults.

If the file contains unknown keys or invalid values, noteui rejects that load and falls back to defaults.

## Minimal example

```toml
dashboard = true

[theme]
name = "nord"

[preview]
style = "auto"

[sync]
default_profile = "homebox"

[sync.profiles.homebox]
ssh_host = "notes-prod"
remote_root = "/srv/noteui"
remote_bin = "/usr/local/bin/noteui-sync"
```

## Full example

```toml
dashboard = true

[theme]
name = "default"
border_style = "rounded"
app_padding_x = 2
app_padding_y = 1
panel_padding_x = 1
panel_padding_y = 0

[typography]
bold_title_bar = true
bold_panel_titles = true
bold_headers = true
bold_selected = true
bold_modal_titles = true

[icons]
category_expanded = "▾"
category_collapsed = "▸"
category_leaf = "•"
note = "·"

[modal]
border_style = "rounded"
padding_x = 2
padding_y = 1

[preview]
render_markdown = true
style = "dark"
syntax_highlight = true
code_style = "monokai"
privacy = false
line_numbers = true

[sync]
default_profile = "homebox"

[sync.profiles.homebox]
ssh_host = "notes-prod"
remote_root = "/srv/noteui"
remote_bin = "/usr/local/bin/noteui-sync"

[keys]
toggle_sync = ["S"]
delete_remote_keep_local = ["U"]
sync_import_current = ["i"]
sync_import = ["I"]
```

## Top-level keys

- `dashboard`
- `theme`
- `typography`
- `icons`
- `modal`
- `preview`
- `keys`
- `sync`

## `dashboard`

Type: boolean

Default:

```toml
dashboard = true
```

Controls whether the dashboard is enabled.

## `theme`

Theme chooses the built-in theme and optional visual overrides.

### `theme.name`

Type: string

Default:

```toml
[theme]
name = "default"
```

Valid built-in theme names:

- `default`
- `nord`
- `gruvbox`
- `catppuccin`
- `catppuccin-mocha`
- `mocha`
- `catppuccin-latte`
- `latte`
- `solarized-light`
- `paper`
- `onedark`
- `kanagawa`
- `dracula`
- `everforest`
- `everforest-dark`
- `tokyo-night-storm`
- `tokyonight-storm`
- `github-light`
- `github-dark`
- `carbonfox`

### Theme color overrides

Type: string for each field

Supported override fields:

- `bg_color`
- `panel_bg_color`
- `border_color`
- `focus_border_color`
- `accent_color`
- `accent_soft_color`
- `text_color`
- `muted_color`
- `subtle_color`
- `chip_bg_color`
- `inline_code_bg_color`
- `pinned_note_color`
- `synced_note_color`
- `unsynced_note_color`
- `syncing_note_color`
- `marked_item_color`
- `error_color`
- `success_color`
- `selected_bg_color`
- `selected_fg_color`
- `highlight_bg_color`

These fields default to the selected built-in theme. Leave them unset unless you want to override a specific color.

### `theme.border_style`

Type: string

Default:

```toml
[theme]
border_style = "rounded"
```

Valid values:

- `rounded`
- `normal`
- `double`
- `thick`
- `hidden`

### Theme spacing

Type: integer

Defaults:

```toml
[theme]
app_padding_x = 2
app_padding_y = 1
panel_padding_x = 1
panel_padding_y = 0
```

These fields control outer app padding and per-panel padding.

## `typography`

Type: booleans

Defaults:

```toml
[typography]
bold_title_bar = true
bold_panel_titles = true
bold_headers = true
bold_selected = true
bold_modal_titles = true
```

Supported fields:

- `bold_title_bar`
- `bold_panel_titles`
- `bold_headers`
- `bold_selected`
- `bold_modal_titles`

## `icons`

Type: strings

Defaults:

```toml
[icons]
category_expanded = "▾"
category_collapsed = "▸"
category_leaf = "•"
note = "·"
```

Supported fields:

- `category_expanded`
- `category_collapsed`
- `category_leaf`
- `note`

## `modal`

Modal controls popup colors, border style, and padding.

Supported fields:

- `bg_color`
- `border_color`
- `title_color`
- `text_color`
- `muted_color`
- `accent_color`
- `error_color`
- `border_style`
- `padding_x`
- `padding_y`

Fields left unset inherit the app's effective visual styling.

Defaults:

```toml
[modal]
border_style = "rounded"
padding_x = 2
padding_y = 1
```

Valid `modal.border_style` values match `theme.border_style`.

## `preview`

Preview controls terminal preview rendering.

Defaults:

```toml
[preview]
render_markdown = true
style = "dark"
syntax_highlight = true
code_style = "monokai"
privacy = false
line_numbers = true
```

### `preview.render_markdown`

Type: boolean

When true, noteui renders Markdown-style previews for supported note content.

### `preview.disable_paths`

Type: list of strings

Default: unset

Use this to turn off rich preview rendering for specific paths or subtrees.

Example:

```toml
[preview]
disable_paths = [".tmp/", "archive/large-exports/"]
```

### `preview.style`

Type: string

Valid values:

- `dark`
- `light`
- `auto`
- `notty`

### `preview.syntax_highlight`

Type: boolean

Enables syntax highlighting inside rendered code blocks.

### `preview.code_style`

Type: string

Supported values:

- `monokai`
- `github`
- `dracula`
- `swapoff`
- `onesenterprise`
- `native`
- `paraiso-dark`
- `paraiso-light`

### `preview.privacy`

Type: boolean

Controls privacy blur mode in the preview pane.

### `preview.line_numbers`

Type: boolean

Controls line numbers in the preview pane.

## `sync`

Sync is optional. If `sync.default_profile` is empty, noteui does not attempt network sync.

### `sync.default_profile`

Type: string

Default: empty

If set, it must match one of the names under `sync.profiles`.

### `sync.profiles.<name>`

Each sync profile supports:

- `ssh_host`
- `remote_root`
- `remote_bin`

All three are required when the profile exists.

Example:

```toml
[sync]
default_profile = "homebox"

[sync.profiles.homebox]
ssh_host = "notes-prod"
remote_root = "/srv/noteui"
remote_bin = "/usr/local/bin/noteui-sync"
```

For end-to-end sync setup, import flows, and recovery behavior, see the [Sync guide](../guide/sync.md).

## `keys`

Each `[keys]` field takes a list of key strings.

If a field is omitted or given an empty list, noteui keeps the built-in default binding.

Example:

```toml
[keys]
open = ["enter", "o"]
toggle_sync = ["S"]
sync_import = ["I"]
```

### Everyday navigation and panes

- `open`
  Default: `["enter", "o"]`
- `refresh`
  Default: `["r"]`
- `quit`
  Default: `["q", "ctrl+c"]`
- `focus`
  Default: `["tab"]`
- `search`
  Default: `["/"]`
- `move_up`
  Default: `["k", "up"]`
- `move_down`
  Default: `["j", "down"]`
- `collapse_category`
  Default: `["h", "left"]`
- `expand_category`
  Default: `["l", "right"]`
- `jump_bottom`
  Default: `["G"]`
- `pending_g`
  Default: `["g"]`
- `scroll_half_page_up`
  Default: `["ctrl+u"]`
- `scroll_half_page_down`
  Default: `["ctrl+d"]`
- `scroll_page_down`
  Default: `["ctrl+f", "pgdown"]`
- `scroll_page_up`
  Default: `["ctrl+b", "pgup"]`
- `next_match`
  Default: `["n"]`
- `prev_match`
  Default: `["N"]`

### Notes, categories, and organization

- `new_note`
  Default: `["n"]`
- `new_temporary_note`
  Default: `["N"]`
- `new_todo_list`
  Default: `["T"]`
- `create_category`
  Default: `["C"]`
- `toggle_category`
  Default: `[" "]`
- `delete`
  Default: `["d"]`
- `move`
  Default: `["m"]`
- `rename`
  Default: `["R"]`
- `add_tag`
  Default: `["A"]`
- `toggle_select`
  Default: `["v"]`
- `pin`
  Default: `["p"]`
- `show_pins`
  Default: `["P"]`
- `sort_toggle`
  Default: `["s"]`

### Preview and help

- `show_help`
  Default: `["?"]`
- `toggle_preview_privacy`
  Default: `["B"]`
- `toggle_preview_line_numbers`
  Default: `["L"]`

### Sync

- `toggle_sync`
  Default: `["S"]`
- `delete_remote_keep_local`
  Default: `["U"]`
- `sync_import_current`
  Default: `["i"]`
- `sync_import`
  Default: `["I"]`

### Todo and extra motions

- `bracket_forward`
  Default: `["]"]`
- `bracket_backward`
  Default: `["["]`
- `heading_jump_key`
  Default: `["h"]`
- `todo_key`
  Default: `["t"]`
- `todo_add`
  Default: `["a"]`
- `todo_delete`
  Default: `["d"]`
- `todo_edit`
  Default: `["e"]`
- `pending_z`
  Default: `["z"]`
- `delete_confirm`
  Default: `["d"]`
- `toggle_encryption`
  Default: `["E"]`

The in-app help modal is still the live source of truth if you change keybindings heavily.
