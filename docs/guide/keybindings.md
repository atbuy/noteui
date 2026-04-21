# Keybindings

This page documents the default keybindings that most users care about first.

!!! note

    The in-app help view is the live source of truth if you override keybindings in your config.

## Everyday keys

- `j` / `k` or arrow keys: move selection
- `h` / `l` or left/right: collapse or expand categories
- `enter` or `o`: open note in editor
- `e`: edit the selected note in the in-app editor
- `ctrl+p` or `:`: open the command palette
- `/`: search
- `tab`: switch focused pane
- `t`: toggle between Notes and Temporary mode (tree focus)
- `ctrl+t`: toggle the global Todos mode
- `q`: quit
- `?`: show help
- `W`: open the workspace picker
- `F`: select the default sync profile; noteui updates only `sync.default_profile`

## Create and organize

- `n`: new note (shows a template picker if `.templates/` contains any template files; press `e` on a template entry to edit it)
- `D`: open or create today's daily note (defaults to `daily/YYYY-MM-DD.md`; see `[daily_notes]` in the configuration reference)
- `ctrl+n`: new template in `.templates/`
- `ctrl+k`: edit templates (opens template picker in edit mode)
- `N`: new temporary note
- `T`: new todo list
- `C`: create category
- `R`: rename
- `m`: move
- `d`: delete the current item, or trash marked notes after confirmation
- `A`: add tags to the current note or marked notes
- `v`: mark the current note/category/temp note
- `V`: clear current marks

## Pins and sorting

- `p`: pin current item or marked notes
- `P`: show pinned items
- `s`: open sort menu; then `n`=name, `m`=modified, `c`=created, `s`=size, `r`=reverse order, `esc`=cancel

## Temporary lifecycle

- `M`: promote the selected temporary note or marked temp-note batch into main notes
- `ctrl+a`: archive the selected temporary note or marked temp-note batch into `archive/tmp/`
- `ctrl+r`: move the selected note or marked note batch into temporary notes

## Todos mode

- `ctrl+t`: toggle the global open-tasks view
- `j` / `k`: move through open tasks
- `enter`: jump to the source note
- `tt`: toggle the selected task
- `ta`: add a task to the selected note
- `te`: edit the selected task
- `td`: delete the selected task
- `tu`: set or clear the selected task due date
- `tp`: set or clear the selected task priority

## Preview controls

- `B`: toggle preview privacy
- `L`: toggle preview line numbers
- `]f` / `[f`: enter link nav mode and jump to the next or previous link; navigates both `[[wikilinks]]` and regular `[text](url)` links; the selected link is highlighted, and markdown links reveal their URL only while selected
- In link nav: `f` or `enter` follows the selected link; for wikilinks, opens the referenced note; `[[target|label]]` resolves by `target`; for external URLs, opens the URL in the system browser; `esc` exits link nav
- `enter`: when not in link nav, follows the first visible `[[wikilink]]` on screen, including `[[target|label]]`
- `n` / `N`: next or previous search match in preview
- `]h` / `[h`: next or previous heading in preview
- `]t` / `[t`: next or previous todo in preview
- `tt`: toggle the current todo checkbox
- `ta`: add a todo item
- `te`: edit the current todo item
- `td`: delete the current todo item
- `tu`: set or clear the current todo due date
- `tp`: set or clear the current todo priority
- `ctrl+u` / `ctrl+d`: half-page scroll
- `ctrl+b` / `ctrl+f`: page up/down

## In-app editor

The in-app editor shows the same rendered preview as the preview pane, with a cursor line highlighted. Every source line is individually selectable and editable.

- `e`: open the selected note in the in-app editor
- `enter` / `o`: keep using the external editor path (unchanged)
- `ctrl+f`: toggle full-screen mode on and off (overrides the TOML default for the session)
- Normal mode supports `h`, `j`, `k`, `l`, `w`, `b`, `e`, `0`, `^`, `$`, `gg`, `G`; `ctrl+left`/`ctrl+right` jump to the previous/next word start; `home`/`end` jump to the start/end of the line
- `j`/`k` moves one source line at a time (list items, headings, code lines, and paragraph lines each get their own cursor stop)
- Insert and open commands: `i`, `a`, `I`, `A`, `o`, `O`; pressing `i` on a line shows that line as raw markdown with a text cursor; all other lines stay rendered
- Edit operators: `d`, `c`, `y`, `x`, `s`, `S`, `dd`, `cc`, `yy`, `p`, `P`; `s` deletes the character under the cursor and enters insert mode; `S` clears the entire line and enters insert mode
- Visual mode: `v` starts character selection, `V` starts line selection; extend with `h`/`j`/`k`/`l`, motion keys, `ctrl+left`/`ctrl+right`, or `home`/`end`; `y` yanks selection and copies it to the system clipboard; `d` deletes; `c` changes
- Search: `/`, `?`, `n`, `N`
- Command line: `:w`, `:w!`, `:wq`, `:q`, `:q!`, `:e!`; `:q` returns to preview without saving
- `tt`: toggle a checkbox on the current line (`- [ ]` to `- [x]` and back)
- `gl`: open the note picker and insert a wikilink
- `gu`: prompt for a URL and insert or wrap a markdown link

## Advanced note actions

- `E`: toggle encryption
- `H`: open the version history modal for the selected local note
- `X`: open the trash browser; lists notes trashed from the current workspace, sorted by deletion date; press `enter` to restore, `esc` to close
- `S`: toggle sync for the selected local note (`sync: local` ↔ `sync: synced`)
- `ctrl+s`: toggle shared status of the selected note (`sync: shared` ↔ `sync: local`)
- `O`: open the generated conflict copy for the selected conflicted synced note
- `ctrl+e`: open sync details for the selected unhealthy synced note; shows the issue, how long ago it occurred, and recovery options (`r` to retry sync, `u` to unlink a note whose remote copy is missing)
- `Y`: open the sync timeline, a scrollable log of recent sync runs with status, timestamps, and counts
- `U`: delete only the remote copy of a synced local note
- `i`: import the selected remote-only note
- `I`: import all missing synced notes
- `Z`: undo the last trash operation; restores the trashed note or category to its original path (available until the next deletion or workspace switch)

## Appearance

- `ctrl+y`: open the theme picker; hover over themes to preview them live across the whole UI; `esc` restores the original theme, `enter` saves only `theme.name`

## Customizing keybindings

Keybindings can be overridden in the `[keys]` section of the config file.

See [Configuration reference](../reference/configuration.md) for details.

For end-to-end sync behavior, see the [Sync guide](sync.md).
