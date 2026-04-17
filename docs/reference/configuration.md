# Configuration reference

This page documents the supported `config.toml` keys for noteui.

## Config file lookup and validation

noteui loads configuration in this order:

1. If `NOTEUI_CONFIG` is set, use that file path.
2. Otherwise use `noteui/config.toml` inside the user config directory.

If the file does not exist, noteui uses defaults.

If the file contains unknown keys or invalid values, noteui warns at startup instead of regenerating the file. It keeps the decoded portion of the config where possible and continues to use code defaults for anything missing or invalid.

## What noteui writes back

Defaults live in code. noteui does not rewrite your `config.toml` with every default value.

Today, in-app writes are intentionally narrow:

- the theme picker and `noteui +set-theme` update only `theme.name`
- the in-app sync profile picker updates only `sync.default_profile`

When noteui writes one of those values, it patches that key in place and preserves the rest of the file where possible instead of reformatting the whole config.

## Minimal example

```toml
dashboard = true
default_workspace = "work"

[workspaces.work]
root = "/home/alice/notes"
label = "Work"

[workspaces.demo]
root = "/home/alice/demo-notes"
label = "Demo"

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
default_workspace = "work"

[workspaces.work]
root = "/home/alice/notes"
label = "Work"
sync_remote_root = "/srv/noteui/work"

[workspaces.demo]
root = "/home/alice/demo-notes"
label = "Demo"
sync_remote_root = "/srv/noteui/demo"

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
kind = "ssh"
ssh_host = "notes-prod"
remote_root = "/srv/noteui"
remote_bin = "/usr/local/bin/noteui-sync"

[sync.profiles.cloud]
kind = "webdav"
webdav_url = "https://cloud.example.com/remote.php/dav/files/alice"
remote_root = "/Notes"
auth = "basic"
username_env = "NOTEUI_WEBDAV_USER"
password_env = "NOTEUI_WEBDAV_PASSWORD"

[keys]
toggle_sync = ["S"]
select_workspace = ["W"]
select_sync_profile = ["F"]
open_conflict_copy = ["O"]
show_sync_debug = ["ctrl+e"]
show_sync_timeline = ["Y"]
delete_remote_keep_local = ["U"]
sync_import_current = ["i"]
sync_import = ["I"]
```

## Top-level keys

- `dashboard`
- `default_workspace`
- `workspaces`
- `theme`
- `typography`
- `icons`
- `modal`
- `preview`
- `keys`
- `sync`
- `daily_notes`

## `dashboard`

Type: boolean

Default:

```toml
dashboard = true
```

Controls whether the dashboard is enabled.

## `default_workspace`

Type: string

Optional. When set, noteui starts in that configured workspace. The name must match a key under `workspaces`.

## `workspaces`

Type: table of named workspace entries

Each workspace defines a notes root and an optional display label.

Example:

```toml
[workspaces.work]
root = "/home/alice/notes"
label = "Work"

[workspaces.demo]
root = "/home/alice/demo-notes"
label = "Demo"
```

Supported keys per workspace:

- `root`: required notes root for that workspace
- `label`: optional UI label used in the footer, title bar, and picker
- `sync_remote_root`: optional path override that directs sync traffic for this workspace to a specific remote directory, instead of using the active sync profile's `remote_root`

`sync_remote_root` is the key to preventing cross-workspace note contamination when multiple workspaces share the same sync profile. Without it, all workspaces upload to and download from the same remote directory, which means workspace A's notes appear as remote-only placeholders in workspace B.

For SSH profiles, `sync_remote_root` is a normal remote filesystem path such as `/srv/noteui/work`.

For WebDAV profiles, `sync_remote_root` is still a remote path, but it is relative to the configured `webdav_url`. In other words, use values such as `/Notes/work` or `/Notes/personal`, not local filesystem paths like `/home/alice/notes/work`.

Example with per-workspace remote roots:

```toml
[workspaces.work]
root = "/home/alice/notes/work"
label = "Work"
sync_remote_root = "/srv/noteui/work"

[workspaces.personal]
root = "/home/alice/notes/personal"
label = "Personal"
sync_remote_root = "/srv/noteui/personal"
```

When `sync_remote_root` is set, the workspace picker shows it as a third line under the root path so you can confirm the mapping before switching.

Workspace switching is available from the command palette when multiple workspaces are configured. Local UI state such as pins, collapsed folders, recent commands, and sort preference is stored separately per workspace.

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
- `crimson`
- `dusk`

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

Each sync profile has a `kind` field that selects the backend. When `kind` is omitted, it defaults to `"ssh"`.

#### SSH profiles (`kind = "ssh"`)

| Key | Required | Default | Description |
|-----|----------|---------|-------------|
| `kind` | no | `"ssh"` | Backend type |
| `ssh_host` | yes | | SSH host to connect to |
| `remote_root` | yes | | Remote directory for sync data |
| `remote_bin` | no | `"noteui-sync"` | Path to `noteui-sync` on remote |

Example:

```toml
[sync.profiles.homebox]
kind = "ssh"
ssh_host = "notes-prod"
remote_root = "/srv/noteui"
remote_bin = "/usr/local/bin/noteui-sync"
```

#### WebDAV profiles (`kind = "webdav"`)

| Key | Required | Default | Description |
|-----|----------|---------|-------------|
| `kind` | yes | | Must be `"webdav"` |
| `webdav_url` | yes | | Full URL to the authenticated WebDAV user endpoint (`http://` or `https://`) |
| `remote_root` | no | `"/noteui"` | Remote directory under `webdav_url` where noteui stores notes and metadata |
| `auth` | no | `"basic"` | Auth mode: `"basic"` or `"none"` |
| `username_env` | when auth=basic | | Env var name holding the username value |
| `password_env` | when auth=basic | | Env var name holding the password value |

Example:

```toml
[sync.profiles.cloud]
kind = "webdav"
webdav_url = "https://cloud.example.com/remote.php/dav/files/alice"
remote_root = "/Notes/personal"
auth = "basic"
username_env = "NOTEUI_WEBDAV_USER"
password_env = "NOTEUI_WEBDAV_PASSWORD"
```

Practical rules:

- `webdav_url` should usually stop at the user endpoint. For Nextcloud that is typically `https://<host>/remote.php/dav/files/<username>`.
- `remote_root` is joined under that endpoint. If you want noteui to sync into a Nextcloud folder named `Notes`, use `remote_root = "/Notes"`.
- Do not append the notes folder to `webdav_url` and repeat it again in `remote_root`.
- `username_env` and `password_env` are variable names, not the secrets themselves. The variables must be present in the same environment that launches `noteui`.
- noteui creates `remote_root` and its `.noteui-sync/` metadata directory automatically on first successful upload.
- if you use `sync_remote_root` with a WebDAV profile, it follows the same semantics as `remote_root`.

For end-to-end sync setup, import flows, conflict resolution, and recovery behavior, see the [Sync guide](../guide/sync.md).

## `keys`

Each `[keys]` field takes a list of key strings.

If a field is omitted or given an empty list, noteui keeps the built-in default binding.

Example:

```toml
[keys]
open = ["enter", "o"]
toggle_sync = ["S"]
open_conflict_copy = ["O"]
sync_import = ["I"]
note_history = ["H"]
trash_browser = ["X"]
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
- `command_palette`
  Default: `["ctrl+p", ":"]`
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
- `clear_marks`
  Default: `["V"]`
- `pin`
  Default: `["p"]`
- `promote_temporary`
  Default: `["M"]`
- `archive_temporary`
  Default: `["ctrl+a"]`
- `move_to_temporary`
  Default: `["ctrl+r"]`
- `show_pins`
  Default: `["P"]`
- `show_todos`
  Default: `["ctrl+t"]`
- `select_workspace`
  Default: `["W"]`
- `sort_key`
  Default: `["s"]`
  Opens the sort menu. Follow with a sub-key to pick a method.
- `sort_by_name`
  Default: `["n"]`
  Active only inside the sort menu. Sorts alphabetically by path.
- `sort_by_modified`
  Default: `["m"]`
  Active only inside the sort menu. Sorts by modification time, newest first.
- `sort_by_created`
  Default: `["c"]`
  Active only inside the sort menu. Sorts by creation date (frontmatter `date:`/`created:`/`created-at:`, falling back to modification time), newest first.
- `sort_by_size`
  Default: `["s"]`
  Active only inside the sort menu. Sorts by file size, largest first.
- `sort_reverse`
  Default: `["r"]`
  Active only inside the sort menu. Toggles ascending/descending order.

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
  Toggles the selected local note between `sync: local` and `sync: synced`.
- `make_shared`
  Default: `["ctrl+s"]`
  Toggles the selected note between `sync: shared` and `sync: local`.
- `select_sync_profile`
  Default: `["F"]`
- `open_conflict_copy`
  Default: `["O"]`
  Opens the generated conflict copy for the selected conflicted synced note.
- `show_sync_debug`
  Default: `["ctrl+e"]`
  Opens the sync details modal for the selected unhealthy synced note.
- `show_sync_timeline`
  Default: `["Y"]`
  Opens the sync timeline showing recent sync run history for the current workspace.
- `delete_remote_keep_local`
  Default: `["U"]`
- `sync_import_current`
  Default: `["i"]`
- `sync_import`
  Default: `["I"]`
- `undo_delete`
  Default: `["Z"]`
  Restores the most recently trashed note or category to its original path. Available until the next deletion or workspace switch.

### History and extra motions

- `note_history`
  Default: `["H"]`
  Opens the version history modal for the selected local note.
- `trash_browser`
  Default: `["X"]`
  Opens the trash browser modal, listing notes trashed from the current workspace. Navigate with `j`/`k`, press `enter` to restore, `esc` to close.

### Templates

- `new_template`
  Default: `["ctrl+n"]`
  Creates a blank template file in `.templates/` and opens it in your editor.
- `edit_templates`
  Default: `["ctrl+k"]`
  Opens the template picker in edit mode so you can select a template to edit.

### Daily notes

- `open_daily_note`
  Default: `["D"]`
  Opens today's daily note, creating it if it does not yet exist. See `[daily_notes]` for directory and template configuration.

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
- `todo_due_date`
  Default: `["u"]`
- `todo_priority`
  Default: `["p"]`
- `pending_z`
  Default: `["z"]`
- `delete_confirm`
  Default: `["d"]`
- `toggle_encryption`
  Default: `["E"]`
- `link_key`
  Default: `["f"]`
  Second key of the `]f` / `[f` chord to enter link nav mode in the preview pane. Also follows the selected link when already in link nav mode.
- `follow_link`
  Default: `["f"]`
  Follows (opens) the currently selected wikilink while in link nav mode. Shares the default with `link_key`.

The in-app help modal is still the live source of truth if you change keybindings heavily.


## `[keys]`

The `[keys]` section overrides default bindings.

Sync-related key defaults:

- `toggle_sync = ["S"]`
- `make_shared = ["ctrl+s"]`
- `select_workspace = ["W"]`
- `select_sync_profile = ["F"]`
- `open_conflict_copy = ["O"]`
- `show_sync_debug = ["ctrl+e"]`
- `show_sync_timeline = ["Y"]`
- `delete_remote_keep_local = ["U"]`
- `sync_import_current = ["i"]`
- `sync_import = ["I"]`

## `[daily_notes]`

Controls where daily notes are stored and which template is used when creating one.

### `daily_notes.dir`

Type: string

Default: `"daily"`

The subdirectory inside your notes root where daily notes are stored. The directory is created automatically on first use.

### `daily_notes.template`

Type: string

Default: `""` (no template; a minimal heading is used instead)

Path to a template file relative to `.templates/`. When set, pressing `D` creates the daily note by applying that template (with `{{date}}`, `{{time}}`, and `{{title}}` substitution). When empty, the note is created with a `# YYYY-MM-DD` heading.

Example:

```toml
[daily_notes]
dir = "journal"
template = "daily.md"
```

With this config, pressing `D` opens or creates `journal/YYYY-MM-DD.md`, using `.templates/daily.md` as the template for new files.

## Reserved directories

noteui uses the following hidden directories inside the notes root and skips them during normal note discovery:

| Directory | Purpose |
|---|---|
| `.tmp/` | Temporary notes |
| `.noteui-history/` | Automatic per-note version history |
| `.noteui-sync/` | Sync bookkeeping metadata and event log (`sync-events.jsonl`) |
| `.templates/` | User-defined note templates (see [Note templates](../guide/usage.md#note-templates)) |
