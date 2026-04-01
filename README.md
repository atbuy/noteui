[![CI](https://github.com/atbuy/noteui/actions/workflows/ci.yml/badge.svg)](https://github.com/atbuy/noteui/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/atbuy/noteui/branch/main/graph/badge.svg)](https://codecov.io/gh/atbuy/noteui)
[![Documentation](https://img.shields.io/badge/docs-online-blue)](https://atbuy.github.io/noteui/)

# noteui

`noteui` is a terminal note-taking application for browsing, searching, previewing, and organizing plain-text notes stored as regular files.

It is built for people who want a keyboard-driven notes workflow without giving up normal files, directories, and external editors.

## Highlights

- browse notes and categories in a tree view
- preview notes directly in the terminal
- search by title, path, content preview, and tags
- create, rename, move, and delete notes or categories
- keep temporary notes separate from your main notes
- create and manage todo notes
- pin important notes and categories
- customize theme, preview behavior, icons, and keybindings
- keep your notes as regular files on disk

## Install

The easiest way to install `noteui` is from the pre-built release archives:

<https://github.com/atbuy/noteui/releases>

Linux and macOS releases are published as `.tar.gz` archives. Windows releases are published as `.zip` archives.

## Quick start

1. Download the right release archive for your platform from the releases page.
2. Extract it.
3. Run the binary.
4. Start writing notes in your notes directory, which defaults to `$HOME/notes`.

By default, `noteui`:

- uses `$HOME/notes` as the notes root
- stores temporary notes under `.tmp` inside the notes root
- opens notes with `NOTEUI_EDITOR`, then `EDITOR`, then `nvim`
- stores local UI state under `$HOME/.local/state/noteui/state.json`

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

## Build from source

```bash
make build
./bin/noteui
```

Run the test suite with:

```bash
make test
```
