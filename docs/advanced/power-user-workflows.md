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

## Split config for dotfiles

If you track your config in a dotfiles repo, you probably do not want your SSH hosts, remote paths, and private workspace roots in it. Split the config with `[meta] includes`: commit the shared part, keep the private part in a local file that never leaves the machine.

Committed `~/.config/noteui/config.toml`:

```toml
dashboard = true

[meta]
includes = ["local.toml"]

[theme]
name = "nord"

[preview]
style = "auto"
```

Gitignored `~/.config/noteui/local.toml`:

```toml
default_workspace = "personal"

[workspaces.personal]
root = "/home/alice/notes"

[sync]
default_profile = "homebox"

[sync.profiles.homebox]
ssh_host = "notes-prod"
remote_root = "/srv/noteui"
remote_bin = "/usr/local/bin/noteui-sync"
```

The include is merged over the main file, and named tables accumulate, so both files can define workspaces and sync profiles. On a fresh machine where `local.toml` does not exist yet, noteui starts with a warning instead of failing. Keep `sync.default_profile` next to the profiles it names so a machine without the include still validates.

Two related points:

- actual credential values (WebDAV passwords, tokens) do not belong in either file; use environment variables or `secrets.toml`, see [Environment variables](../reference/environment.md#webdav-credential-fallback-file)
- in-app writes such as the theme picker and sync profile picker follow the key to the file that defines it, so if you keep `theme.name` or `sync.default_profile` in the include, noteui updates the include; if that include is committed, the change shows up there in git

Full resolution and precedence rules are in the [configuration reference](../reference/configuration.md#splitting-the-config-across-files). Run `noteui +check-config` to verify the merged result.

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
