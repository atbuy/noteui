# Usage guide

This guide explains how noteui behaves in day-to-day use.

## Demo mode

Run `noteui --demo` to launch noteui with a set of built-in sample notes
copied into a temporary directory. The session is fully interactive: every
keybinding works, including create, rename, move, delete, and search. Nothing
is ever written to your real notes root, and the temporary directory is
removed automatically when you quit.

The demo workspace ships with:

- two root-level notes (`inbox.md`, `journal.md`)
- three categories (`personal/`, `reference/`, `work/`) with a total of five
  nested notes covering meeting notes, project status, reading lists, ideas,
  and a command reference
- one temporary note under `.tmp/` (reachable with `t`)
- a tag set that includes `project`, `meeting`, `books`, `reference`, `inbox`

Demo mode also:

- forces the dashboard off so the tree and preview are visible immediately
- clears any sync configuration for the session, so no SSH connection is
  attempted and nothing is uploaded anywhere
- labels the active workspace as `Demo` in the title bar

Use demo mode when you want to:

- try noteui before deciding to set up a personal notes directory
- record a screencast or GIF without exposing any real notes
- reproduce a layout or rendering bug against a known set of fixtures
- run the TUI in CI to verify that it starts up and renders a frame

## Quick capture

Run `noteui --capture "text"` (or `-w` for short) to append a timestamped
entry to `inbox.md` in the active notes root without opening the TUI:

```
noteui --capture "follow up with Alice about the proposal"
noteui -w "follow up with Alice about the proposal"
```

If you omit the text argument, noteui reads from stdin instead:

```
echo "remember to update the config" | noteui --capture
git log --oneline -5 | noteui --capture
```

The active notes root follows the same resolution order as normal startup:
`NOTES_ROOT` environment variable, then `default_workspace` in config, then
the first configured workspace, then `~/notes`.

The capture note (`inbox.md`) is created automatically if it does not exist.
Each entry is prefixed with a timestamp:

```
- [2026-04-11 19:30] follow up with Alice about the proposal
```

This is designed for shell aliases and scripts where opening the full TUI
would be disruptive.

## Validating the config

Run `noteui +check-config` to validate the config file and report any problems without opening the TUI:

```
noteui +check-config
```

The command:

- prints the resolved config file path
- loads and validates the file, reporting any parse or validation errors
- lists any warnings (deprecated fields, unknown sync profile references, etc.)
- checks for keybinding conflicts in the `[keys]` section, including ambiguous sort-menu and preview/todo chord overrides
- exits with code 0 if valid, code 1 if any errors were found

This is useful for CI, shell scripts, or any time you want to confirm that a config edit is syntactically and semantically correct before restarting noteui. Warnings do not cause a non-zero exit; only errors do.

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

## In-app editor

Use `e` on the selected note to open the in-app editor. This keeps the TUI active instead of suspending into `$NOTEUI_EDITOR` or `$EDITOR`.

If you prefer to keep the tree visible while editing, set:

```toml
[preview]
edit_in_preview = true
```

With that option enabled, `e` opens the same in-app editor inside the preview pane instead of taking over the full screen.

The external editor path is unchanged:

- `enter` or `o` still opens the selected note in your configured external editor
- `e` opens the in-app editor

The in-app editor supports a focused vim-style subset:

- motions: `h`, `j`, `k`, `l`, `w`, `b`, `e`, `0`, `^`, `$`, `gg`, `G`
- insert and open: `i`, `a`, `I`, `A`, `o`, `O`
- insert mode editing: `backspace` deletes the previous character, `ctrl+w` deletes backward to the previous word boundary
- edit operators: `d`, `c`, `y`, `x`, `dd`, `cc`, `yy`, `p`, `P`
- visual and search: `v`, `V`, `/`, `?`, `n`, `N`
- command line: `:w`, `:w!`, `:wq`, `:q`, `:q!`, `:e!`

Link insertion is built in:

- `gl` opens the note picker and inserts a wikilink
- `gu` prompts for a URL and inserts or wraps a markdown link

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

After a successful trash operation the status bar shows `Z to undo`. Press `Z` to restore the trashed item back to its original path. The undo affordance is available until the next deletion or workspace switch. Trashing a second note replaces it; only the most recent deletion can be undone.

To browse all notes trashed from the current workspace, press `X`. The trash browser shows each item's original location and deletion time. Navigate with `j`/`k`, press `enter` to restore an item to its original path, and `esc` to close the modal. The trash browser is also available from the command palette as "Trash Browser".

Marked notes let the existing note actions work on a batch:

- `p`: pin or unpin marked notes
- `A`: add tags to marked notes
- `d`: trash marked notes after the same confirmation step

## Sorting

Press `s` to open the sort menu. The status bar shows the available sub-keys:

- `n`: sort alphabetically by path (default)
- `m`: sort by modification time, newest first
- `c`: sort by creation date (reads `date:`, `created:`, or `created-at:` from frontmatter; falls back to modification time), newest first
- `s`: sort by file size, largest first
- `r`: toggle ascending/descending order
- `esc`: cancel without changing the sort

The current sort method is shown in the footer. When sync is configured, the footer also shows the active sync profile and effective remote root for the current workspace. When reverse order is active, a `^` indicator appears next to the method name. Sort preference is stored per workspace in local state and persists across restarts.

Sort methods can also be applied from the command palette ("Sort by Name", "Sort by Modified", "Sort by Created", "Sort by Size", "Reverse Sort Order").

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

## Wikilinks

Inside any note you can write `[[note title]]` to link to another note by its title or filename. Aliased wikilinks are also supported: `[[target|label]]`.

When the preview pane renders a note with wikilinks, each `[[target]]` or `[[target|label]]` appears as a styled link. To follow a wikilink:

1. Focus the preview pane with `tab`.
2. Scroll until a `[[target]]` line is visible.
3. Press `enter`. noteui finds the matching note, selects it in the tree, and opens it in your editor.

If the target does not match any note, the status bar shows `no note found for [[target]]`.

Matching is case-insensitive and checks in this order:

1. Exact title match
2. Filename stem match (for example, `[[my-note]]` matches `my-note.md`)
3. Title prefix match (for example, `[[Meeting]]` matches a note titled "Meeting 2026")

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
- `F` changes the configured default sync profile from inside noteui by updating only `sync.default_profile`
- `O` opens the generated conflict copy for the selected conflicted synced note
- `U` deletes only the remote copy and keeps the local file
- `i` imports the selected remote-only note
- `I` imports all missing synced notes

Remote-only notes appear as muted placeholder rows until imported. If a synced note has a conflict, merge the conflict copy back into the original local note and sync again.

See the [Sync guide](sync.md) for setup and recovery details.

## Encrypted notes

noteui supports encrypted note bodies for workflows that want encrypted content on disk with preview/edit support inside the app.

Encrypted notes can also be edited with `e`. noteui decrypts the body in memory for the in-app editor, and `:w` or `:wq` re-encrypts the body without writing plaintext to disk.

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
