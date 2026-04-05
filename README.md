[![CI](https://github.com/atbuy/noteui/actions/workflows/ci.yml/badge.svg)](https://github.com/atbuy/noteui/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/atbuy/noteui/branch/main/graph/badge.svg)](https://codecov.io/gh/atbuy/noteui)
[![Documentation](https://img.shields.io/badge/docs-online-blue)](https://atbuy.github.io/noteui/)

# noteui

`noteui` is a terminal note-taking application for browsing, searching, previewing, and organizing plain-text notes stored as regular files.

It is built for people who want a keyboard-driven notes workflow without giving up normal files, directories, and external editors.

## Documentation

Full documentation is published at:

<https://atbuy.github.io/noteui/>

Recommended entry points:

- [Getting started](https://atbuy.github.io/noteui/tutorial/getting-started/)
- [Installation](https://atbuy.github.io/noteui/tutorial/installation/)
- [Usage guide](https://atbuy.github.io/noteui/guide/usage/)
- [Configuration reference](https://atbuy.github.io/noteui/reference/configuration/)
- [Environment variables](https://atbuy.github.io/noteui/reference/environment/)
- [FAQ](https://atbuy.github.io/noteui/faq/)

## Highlights

- browse notes and categories in a tree view
- preview notes directly in the terminal
- search by title, path, content preview, and tags
- create, rename, move, and delete notes or categories
- keep temporary notes separate from your main notes
- create and manage todo notes, with a global open-tasks view
- promote, archive, and batch-process temporary notes
- pin important notes and categories
- optional SSH-based sync for `sync: synced` notes with tree sync markers
- customize theme, preview behavior, icons, and keybindings
- keep your notes as regular files on disk
- switch between named workspaces with isolated local UI state

## Install

Quick install examples:

Linux / macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/atbuy/noteui/main/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/atbuy/noteui/main/install.ps1 | iex
```

The easiest manual install path is still the pre-built release archives:

<https://github.com/atbuy/noteui/releases>

Linux and macOS releases are published as `.tar.gz` archives. Windows releases are published as `.zip` archives. Each release archive includes both `noteui` and `noteui-sync`.

## Quick start

1. Download the right release archive for your platform from the releases page.
2. Extract it.
3. Run `noteui`.
4. Start writing notes in your notes directory, which defaults to `$HOME/notes`.

By default, `noteui`:

- uses `$HOME/notes` as the notes root unless a workspace profile or `NOTES_ROOT` override is active
- stores temporary notes under `.tmp` inside the notes root
- opens notes with `NOTEUI_EDITOR`, then `EDITOR`, then `nvim`
- stores local UI state under `$HOME/.local/state/noteui/state.json`

## Sync setup

Sync is optional and SSH-based.

1. Build or install both binaries:
   - `noteui`
   - `noteui-sync`
2. Put `noteui-sync` on the remote machine in a path you can call over SSH.
3. Pick a remote storage directory on that machine, for example `/srv/noteui`.
4. Add a sync profile to your `config.toml`:

```toml
[sync]
default_profile = "homebox"

[sync.profiles.homebox]
ssh_host = "notes-prod"
remote_root = "/srv/noteui"
remote_bin = "/usr/local/bin/noteui-sync"
```

5. Mark any note you want synced with frontmatter:

```yaml
---
sync: synced
---
```

Notes without that field, or with `sync: local`, stay local-only. Sync status in the tree works like this:

- hollow red `○`: local-only note
- green `●`: synced note with a confirmed healthy remote state
- orange blinking dot: a sync, import, or remote-delete action is currently in flight for that note
- filled red `●`: synced note that is not currently confirmed healthy

When noteui starts, synced notes are treated as unconfirmed until the first remote check completes. That avoids showing stale green markers from old local metadata before the current remote state has been verified.

Press `S` on a selected local note to toggle `sync: local` and `sync: synced`. Press `U` on a synced local note to delete only its remote copy and keep the local file, switching it back to `sync: local`.

On another machine, noteui refreshes remote note metadata automatically but does not auto-download missing note bodies. Synced notes that exist on the server but not locally appear in the tree as muted `x` placeholder rows, show an import message in the preview, and cannot be edited until imported. Press `i` to import the selected remote-only note, or `I` to import all missing synced notes. This also works as recovery inside an existing notes root: if you delete a synced note locally, `I` will restore it from the server as long as the target path is free. noteui skips collisions instead of overwriting existing local files.

## Build from source

```bash
make build
./bin/noteui
```

Run the test suite with:

```bash
make test
```
