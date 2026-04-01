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

## `keys`

The `keys` section allows overriding default keybindings. Each field takes a list of key strings.

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

## Example config

!!! tip

    Start small. You do not need to define every section. Add only the values you want to override and let noteui keep the rest at defaults.

```toml
dashboard = true

[theme]
name = "nord"
border_style = "rounded"

[preview]
render_markdown = true
style = "dark"
syntax_highlight = true
code_style = "monokai"
privacy = false
line_numbers = true

[keys]
open = ["enter", "o"]
search = ["/"]
show_help = ["?"]
```
