# Power-user workflows

This page is for users who want to push noteui harder than the default “open and take notes” workflow.

## Keep your notes portable

Because noteui uses normal files, a strong workflow is:

- edit in noteui when you want the TUI experience
- edit directly in your editor when that is faster
- sync your notes with your own tools

## Use temporary notes deliberately

Temporary notes are excellent for:

- quick capture
- session notes
- rough drafts
- inbox-style processing later

## Tune preview behavior

If you use noteui heavily in terminals, the preview section of the config is worth tuning:

- markdown rendering
- syntax highlighting
- code style
- privacy mode
- line numbers

## Keybinding strategy

If you are already deeply used to Vim-style or other terminal workflows, remap noteui’s keys in the config so your most common actions fit your existing muscle memory.

## Use note history as a safety net

noteui automatically saves versions of every note as you edit them. If you ever need to recover an earlier draft or undo a destructive change:

1. Press `H` on the note to open the version history modal.
2. Navigate to the version you want with `j` / `k`.
3. Press `enter` to restore it.

The restore saves the current content first, so you can always undo the restore by pressing `H` again and picking the entry that was the top of the list before you restored.

This is especially useful with encrypted notes: if re-encryption produces a blob you cannot decrypt, the last healthy encrypted state is always one `H` away.

## Isolate workspaces with per-workspace sync roots

If you use multiple workspaces and sync, add `sync_remote_root` to each workspace in your config. Without it, every workspace writes to the same remote directory, and notes cross-contaminate when you switch workspaces and press `I`.

See the [Sync guide](../guide/sync.md#per-workspace-sync-isolation) for the full setup.

## Recommended advanced path

1. Choose a theme.
2. Tune preview defaults.
3. Set your preferred editor.
4. Remap only the keys you use constantly.
5. Keep the rest close to defaults so the help view stays familiar.
6. Add `sync_remote_root` to each workspace if you use sync with multiple workspaces.
