# Configuration reference

This page is the detailed reference for noteui’s configuration file.

## Config file lookup

If `NOTEUI_CONFIG` is set, noteui loads that file.

Otherwise it looks in your user config directory for:

```text
noteui/config.toml
```

If the file does not exist, noteui uses defaults.

!!! note

    Unknown config keys are rejected. Invalid values are also rejected and noteui falls back to defaults for that failed load.

## Top-level sections

The config supports these top-level keys:

- `dashboard`
- `theme`
- `typography`
- `icons`
- `modal`
- `preview`
- `keys`
- `sync`

## `dashboard`

`dashboard` controls whether the dashboard is enabled.

Example:

```toml
dashboard = true
```

## `theme`

Theme controls colors, borders, and spacing.

Supported fields include:

- `name`
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
- `border_style`
- `app_padding_x`
- `app_padding_y`
- `panel_padding_x`
- `panel_padding_y`

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

Valid `border_style` values:

- `rounded`
- `normal`
- `double`
- `thick`
- `hidden`

## `typography`

Typography fields:

- `bold_title_bar`
- `bold_panel_titles`
- `bold_headers`
- `bold_selected`
- `bold_modal_titles`

## `icons`

Icon fields:

- `category_expanded`
- `category_collapsed`
- `category_leaf`
- `note`

## `modal`

Modal fields:

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

Valid `modal.border_style` values follow the same set as `theme.border_style`.

## `preview`

Preview fields:

- `render_markdown`
- `disable_paths`
- `style`
- `syntax_highlight`
- `code_style`
- `privacy`
- `line_numbers`

Valid `preview.style` values:

- `dark`
- `light`
- `auto`
- `notty`

Recognized `preview.code_style` values include:

- `monokai`
- `github`
- `dracula`
- `swapoff`
- `onesenterprise`
- `native`
- `paraiso-dark`
- `paraiso-light`

## `sync`

Sync is optional. If `sync.default_profile` is unset, noteui does not attempt network sync.

Supported fields:

- `default_profile`
- `profiles.<name>.ssh_host`
- `profiles.<name>.remote_root`
- `profiles.<name>.remote_bin`

Example:

```toml
[sync]
default_profile = "homebox"

[sync.profiles.homebox]
ssh_host = "notes-prod"
remote_root = "/srv/noteui"
remote_bin = "/usr/local/bin/noteui-sync"
```

Notes are selected for sync per file with frontmatter:

```yaml
---
sync: synced
---
```

If the field is missing, or set to `local`, the note stays local-only and uses a hollow marker in `theme.unsynced_note_color`.

For synced notes, the tree marker semantics are:

- `theme.synced_note_color`: the note has a confirmed healthy remote state
- `theme.syncing_note_color`: a sync, import, or remote-delete action is currently in flight for that note
- `theme.unsynced_note_color`: the note is synced in intent, but its current remote state is not confirmed healthy

On startup, noteui treats synced notes as unconfirmed until the first remote check completes. This prevents stale local metadata from showing a green marker before the current remote state has been verified.

On a second machine, configure the same sync profile and use `sync_import_current` to import the selected remote-only note, or `sync_import` to pull all missing synced notes from the remote and initialize `.noteui-sync/` as needed. noteui refreshes only remote note metadata automatically; remote-only notes stay as muted placeholders in the tree until imported. The same action also restores deleted synced notes inside an existing root, but it skips any remote note whose target path already exists locally. To stop syncing one local note while keeping the file, use `delete_remote_keep_local` on a synced local note; this deletes the remote copy and switches the local file back to `sync: local`.

## `keys`

The `keys` section allows overriding default keybindings. Each field takes a list of key strings.

Useful sync-related defaults:

- `toggle_sync = ["S"]`
- `delete_remote_keep_local = ["U"]`
- `sync_import_current = ["i"]`
- `sync_import = ["I"]`

Supported fields include:

- `open`
- `refresh`
- `quit`
- `focus`
- `new_note`
- `new_temporary_note`
- `new_todo_list`
- `search`
- `show_help`
- `show_pins`
- `create_category`
- `toggle_category`
- `delete`
- `move`
- `rename`
- `add_tag`
- `toggle_select`
- `pin`
- `toggle_sync`
- `delete_remote_keep_local`
- `sync_import_current`
- `sync_import`
- `toggle_preview_privacy`
- `toggle_preview_line_numbers`
- `sort_toggle`
- `scroll_half_page_up`
- `scroll_half_page_down`
- `next_match`
- `prev_match`
- `move_up`
- `move_down`
- `collapse_category`
- `expand_category`
- `jump_bottom`
- `pending_g`
- `bracket_forward`
- `bracket_backward`
- `heading_jump_key`
- `todo_key`
- `todo_add`
- `todo_delete`
- `todo_edit`
- `pending_z`
- `delete_confirm`
- `scroll_page_down`
- `scroll_page_up`
- `toggle_encryption`
