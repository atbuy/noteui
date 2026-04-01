# noteui

`noteui` is a terminal notes application for people who want the speed of a TUI while keeping their notes as normal files and directories.

It helps you browse, search, preview, organize, and edit notes without moving them into a database or a proprietary format.

!!! tip "New here?"

    Start with the tutorial pages. They are written for first-time users and walk through installation, first launch, and the most common workflows.

## Why use noteui?

- your notes stay as files on disk
- categories are just folders
- you can still use your own editor and sync tools
- the terminal UI adds search, preview, organization, and keyboard-first workflows

## Choose your path

### New users

- [Getting started](tutorial/getting-started.md)
- [Installation](tutorial/installation.md)
- [Your first notes workflow](tutorial/first-notes.md)

### Regular usage

- [Usage guide](guide/usage.md)
- [Keybindings](guide/keybindings.md)

### Advanced usage

- [Encrypted notes](advanced/encryption.md)
- [Power-user workflows](advanced/power-user-workflows.md)

### Exact reference

- [Configuration reference](reference/configuration.md)
- [Environment variables](reference/environment.md)
- [Storage and state](reference/storage-and-state.md)
- [FAQ](faq.md)

## Core concepts

### Notes are files

noteui works with real files on disk. Supported note formats are:

- `.md`
- `.txt`
- `.org`
- `.norg`

### Categories are folders

Subdirectories under the notes root become categories in the UI.

### Temporary notes live under `.tmp`

Temporary notes are stored separately inside the notes root so they do not clutter your main note tree.

### noteui stores UI state separately

Pins, collapsed categories, and sort mode are stored outside your notes so your content stays simple and portable.
