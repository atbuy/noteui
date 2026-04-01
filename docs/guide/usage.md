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

## Todos

Todo note helpers support:

- creating a todo note template
- toggling todo items
- adding items
- editing items
- deleting items

## Preview behavior

Preview behavior can include:

- markdown rendering
- syntax highlighting
- privacy blur mode
- line numbers

All of these can be influenced by configuration.

## Encrypted notes

noteui supports encrypted note bodies for workflows that want encrypted content on disk with preview/edit support inside the app.

See [Encrypted notes](../advanced/encryption.md) for details.
