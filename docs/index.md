
<div style="width: 100%; display: flex; justify-content: center;">
    <img src="assets/logo_full.svg" alt="noteui logo" width="460" />
</div>

# noteui

`noteui` is a terminal notes application for people who want the speed of a TUI while keeping their notes as normal files and directories.

It helps you browse, search, preview, organize, and edit notes without moving them into a database or a proprietary format.

!!! tip "New here?"

    Start with the tutorial pages. They are written for first-time users and walk through installation, first launch, and the most common workflows.

!!! example "Try it in one command"

    Already installed noteui? Run `noteui --demo` to launch the UI against a
    throwaway set of sample notes. Nothing touches your real notes directory.
    See [Demo mode](guide/usage.md#demo-mode) for details.

## Common tasks

- Install noteui and the sync helper: [Installation](tutorial/installation.md)
- Learn the core workflow: [Getting started](tutorial/getting-started.md)
- Customize appearance, preview, and keys: [Configuration reference](reference/configuration.md)
- Set up SSH-based sync: [Sync guide](guide/sync.md)
- Isolate multiple workspaces from each other over sync: [Per-workspace sync isolation](guide/sync.md#per-workspace-sync-isolation)
- Browse and restore note versions: [Note version history](guide/usage.md#note-version-history)
- Understand encrypted notes: [Encrypted notes](advanced/encryption.md)
- Recover remote-only or missing synced notes: [Troubleshooting](reference/troubleshooting.md)

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
- [Sync guide](guide/sync.md)

### Advanced usage

- [Encrypted notes](advanced/encryption.md)
- [Power-user workflows](advanced/power-user-workflows.md)

### Exact reference

- [Configuration reference](reference/configuration.md)
- [Environment variables](reference/environment.md)
- [Storage and state](reference/storage-and-state.md)
- [Troubleshooting](reference/troubleshooting.md)
- [Docs maintenance](reference/docs-maintenance.md)
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
