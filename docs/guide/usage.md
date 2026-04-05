# Usage guide

This guide explains how noteui behaves in day-to-day use.

## Notes tree and preview

The main interface is split into two panes:

- the notes tree
- the preview pane

The tree shows categories and notes. The preview shows the selected note.

Use `tab` to switch focus between panes.

## Navigation

Move through the tree with:

- `j` / `k`
- arrow keys

Expand and collapse categories with:

- `l` / right
- `h` / left

## Search

Press `/` to search.

Search matches against:

- note title
- file name
- relative path
- preview text
- tags

Tag-style search also works through the tag-aware behavior in the notes layer.

## Creating notes and categories

- `n`: create a note
- `N`: create a temporary note
- `T`: create a todo note
- `C`: create a category

## Renaming and moving

- `R`: rename the selected note or category
- `m`: move the selected item

Move operations stay inside the notes root.

## Deleting

- `d`: delete the selected item

!!! warning

    noteui deletes into the user trash instead of immediately removing content permanently.

Marked notes let the existing note actions work on a batch:

- `p`: pin or unpin marked notes
- `A`: add tags to marked notes
- `d`: trash marked notes after the same confirmation step

## Pins

- `p`: pin the current note or category
- `P`: open the pinned items list

Pins are stored in local UI state, separate from your notes.

## Temporary notes

Temporary notes live under `.tmp` inside the notes root and are handled as a separate mode in the UI.

This is useful for:

- quick capture
- drafts
- short-lived task notes
- material you do not want mixed with your main note hierarchy yet

Switch between your normal notes and temporary notes using `t` when focused on the tree.

Temporary-note lifecycle actions now include:

- `M`: promote the current temporary note or marked temp-note batch into the main notes tree
- `ctrl+a`: archive the current temporary note or marked batch into `archive/tmp/`
- `v`: mark temporary notes for batch actions
- `V`: clear current marks

Normal notes can also be sent back into temporary storage with `ctrl+r`.

## Todos

Todo workflows now cover both per-note task editing and a global open-tasks view.

Use:

- `T` to create a todo note template
- `ctrl+t` to toggle the global open todos view
- `tt` to toggle the selected todo
- `ta` to add a todo item
- `te` to edit a todo item
- `td` to delete a todo item
- `tu` to set or clear a todo due date
- `enter` in Todos mode to jump to the source note

Todo metadata can be stored inline in normal markdown task lines:

- priority: `[p1]`, `[p2]`, `[p3]`
- due date: `[due:YYYY-MM-DD]`

The Todos view shows only open tasks and sorts them by due date first, then priority, then note order.

## Preview behavior

Preview behavior can include:

- markdown rendering
- syntax highlighting
- privacy blur mode
- line numbers

All of these can be influenced by configuration.

For the exact config keys and defaults, see the [Configuration reference](../reference/configuration.md).

## Sync workflows

If sync is configured, noteui can mark local notes for sync, refresh remote metadata, and import remote-only notes on demand.

Important behaviors:

- `S` toggles the selected local note between `sync: local` and `sync: synced`
- `F` changes the configured default sync profile from inside noteui
- `O` opens the generated conflict copy for the selected conflicted synced note
- `U` deletes only the remote copy and keeps the local file
- `i` imports the selected remote-only note
- `I` imports all missing synced notes

Remote-only notes appear as muted placeholder rows until imported. If a synced note has a conflict, merge the conflict copy back into the original local note and sync again.

See the [Sync guide](sync.md) for setup and recovery details.

## Encrypted notes

noteui supports encrypted note bodies for workflows that want encrypted content on disk with preview/edit support inside the app.

See [Encrypted notes](../advanced/encryption.md) for details.

## Where to go next

- Need the full key list? Read [Keybindings](keybindings.md).
- Need exact config keys and defaults? Read [Configuration reference](../reference/configuration.md).
- Need sync setup and recovery? Read [Sync guide](sync.md).
