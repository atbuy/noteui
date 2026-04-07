# Keybindings

This page documents the default keybindings that most users care about first.

!!! note

    The in-app help view is the live source of truth if you override keybindings in your config.

## Everyday keys

- `j` / `k` or arrow keys: move selection
- `h` / `l` or left/right: collapse or expand categories
- `enter` or `o`: open note in editor
- `ctrl+p` or `:`: open the command palette
- `/`: search
- `tab`: switch focused pane
- `t`: toggle between Notes and Temporary mode (tree focus)
- `ctrl+t`: toggle the global Todos mode
- `q`: quit
- `?`: show help
- `W`: open the workspace picker
- `F`: select the default sync profile

## Create and organize

- `n`: new note (shows a template picker if `.templates/` contains any template files; press `e` on a template entry to edit it)
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
- `s`: toggle sort order

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

## Advanced note actions

- `E`: toggle encryption
- `H`: open the version history modal for the selected local note
- `S`: toggle sync for the selected local note (`sync: local` ↔ `sync: synced`)
- `ctrl+s`: toggle shared status of the selected note (`sync: shared` ↔ `sync: local`)
- `O`: open the generated conflict copy for the selected conflicted synced note
- `ctrl+e`: open full sync debug details for the selected unhealthy synced note
- `U`: delete only the remote copy of a synced local note
- `i`: import the selected remote-only note
- `I`: import all missing synced notes

## Customizing keybindings

Keybindings can be overridden in the `[keys]` section of the config file.

See [Configuration reference](../reference/configuration.md) for details.

For end-to-end sync behavior, see the [Sync guide](sync.md).
