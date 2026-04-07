# Usage guide

This guide explains how noteui behaves in day-to-day use.

## Workspace profiles

If you configure multiple workspaces in `config.toml`, noteui can switch between them from the command palette or directly with `W`.

Each workspace has its own:

- notes root
- pinned notes and categories
- collapsed category state
- recent command history
- sort preference

If `NOTES_ROOT` is set in the environment, that override wins for the current session and workspace switching is disabled.

### Per-workspace sync isolation

If you use sync with multiple workspaces, set `sync_remote_root` on each workspace to give it a dedicated remote directory. Without this, every workspace syncs to the same remote path, which causes notes from one workspace to appear as remote-only placeholders in another.

```toml
[workspaces.work]
root = "/home/alice/notes/work"
sync_remote_root = "/srv/noteui/work"

[workspaces.personal]
root = "/home/alice/notes/personal"
sync_remote_root = "/srv/noteui/personal"
```

The workspace picker shows the `sync_remote_root` value under each workspace entry so you can confirm the mapping before switching.

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
- full note body
- tags

Search supports both exact substring and fuzzy matching. For each term, noteui first tries an exact substring match across title, filename, path, and body. If no exact match is found, it falls back to a fuzzy subsequence match on the title and path, meaning the characters of your query must appear in order, but not necessarily consecutively. Results are sorted by relevance: title matches rank above body matches, which rank above fuzzy path matches.

Prefix a term with `#` to search by tag only: `#urgent` matches only notes whose tags contain "urgent".

Multi-term search requires all terms to match. For example, `config deploy` only shows notes that match both "config" and "deploy".

## Creating notes and categories

- `n`: create a note (shows a template picker if `.templates/` contains any template files; otherwise creates a blank note immediately)
- `N`: create a temporary note
- `T`: create a todo note
- `C`: create a category
- `ctrl+n`: create a new blank template file in `.templates/` and open it in your editor

## Note templates

Store reusable note skeletons as note files (`.md`, `.txt`, `.org`, or `.norg`) inside a `.templates/` directory at the root of your notes workspace.

When you press `n` and at least one template file exists, a picker appears. Select a template with `j`/`k` and press `enter`. Choose "Blank note" at the top of the list to skip templates and create a blank note as usual. Press `e` on a template entry to open that template directly in your editor for editing.

To create a new template, press `ctrl+n`. This creates a blank template file in `.templates/` and opens it in your editor. You can also use "New Template" in the command palette.

To edit an existing template, use "Edit Templates" in the command palette. This opens the same template picker but in edit mode: every item in the list opens in your editor when confirmed.

Templates support these variables, which are replaced at creation time:

| Variable    | Replaced with                          |
| ----------- | -------------------------------------- |
| `{{date}}`  | Current date in YYYY-MM-DD format      |
| `{{time}}`  | Current time in HH:MM (24-hour) format |
| `{{title}}` | Empty string; set by the note heading  |

Example template file `.templates/meeting.md`:

```markdown
# {{title}}

Date: {{date}}
Time: {{time}}

## Agenda

## Notes

## Action items
```

The `N` (temporary note) and `T` (todo note) keys always bypass the template picker.

Templates are also accessible from the command palette as "New Note from Template" when `.templates/` is non-empty.

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
- `tp` to set or clear a todo priority
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

## Note version history

noteui keeps an automatic version history for every local note.

Versions are saved at these moments: before opening an encrypted note, after any re-encryption, after encrypting or decrypting, after the editor closes for a non-encrypted note, and after any todo edit.

To browse and restore versions:

1. Select a local note in the tree.
2. Press `H`.
3. The history modal opens, listing versions newest-first with a timestamp and the first line of each version.
4. Use `j` / `k` to move through the list.
5. Press `enter` to restore the highlighted version.
6. Press `esc` to close without restoring.

Restoring a version saves the current file content as a new history entry first, so the restore itself is undoable: press `H` again and you will see the pre-restore state at the top of the list.

Encrypted notes show `encrypted` as the first-line preview in the history list rather than the raw blob.

noteui keeps at most 50 versions per note and prunes older ones automatically. Versions are stored in `.noteui-history/` inside the notes root, a hidden directory that the notes tree never shows. See [Storage and state](../reference/storage-and-state.md) for the exact layout.

## Where to go next

- Need the full key list? Read [Keybindings](keybindings.md).
- Need exact config keys and defaults? Read [Configuration reference](../reference/configuration.md).
- Need sync setup and recovery? Read [Sync guide](sync.md).
