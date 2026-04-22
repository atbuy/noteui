# Getting started

This guide is for first-time users who want to get from “I just found noteui” to “I can use it daily” with as little friction as possible.

## What noteui is

noteui is a terminal application for working with notes stored as plain files.

If you already keep notes in Markdown or other text formats and want:

- fast keyboard navigation
- search and preview
- better organization from the terminal
- file-based notes you still fully own

then noteui is meant for that workflow.

## What noteui is not

noteui is not a hosted service, database-backed notes platform, or collaborative web app.

!!! note

    Your notes remain regular files in your notes directory. You can still edit them outside noteui with your own editor or sync them with other tools.

## Before you start

You need:

- a noteui binary from the [releases page](https://github.com/atbuy/noteui/releases)
- a terminal
- optionally, an editor set through `NOTEUI_EDITOR` or `EDITOR`; these can include command arguments such as `code -w`

## Default locations

By default, noteui uses:

- notes root: `$HOME/notes`
- temporary notes: `$HOME/notes/.tmp`
- state file: `$HOME/.local/state/noteui/state.json`
- config file: your user config directory under `noteui/config.toml`
- WebDAV credential fallback file: your user config directory under `noteui/secrets.toml`

If the notes directory does not exist yet, noteui creates it when needed.

## Try it first without a setup

If you just want to see what noteui looks and feels like before touching
anything on your machine, run:

```bash
noteui --demo
```

This launches noteui against a small bundle of sample notes copied into a
throwaway temporary directory. Your real `$HOME/notes` folder is not touched,
and the sample content is cleaned up automatically when you quit.

Use it to try navigation, search, previews, and the temporary notes view
without committing to a setup. When you are ready for the real thing,
continue with the steps below.

## First launch

Run:

```bash
noteui
```

If you are launching the extracted binary directly, run that binary instead.

On first launch, you will usually see an empty notes tree until you create your first note.

If you create or edit `config.toml` before that first launch, you can validate
it without starting the full UI:

```bash
noteui +check-config
```

## First useful actions

Start with these keys:

- `n`: create a note
- `/`: search
- `enter` or `o`: open the selected note in your editor
- `tab`: switch focus between the tree and preview
- `?`: open help
- `q`: quit

## Recommended first workflow

1. Launch noteui.
2. Press `n` to create a note.
3. Open it with `enter`.
4. Add a title and some text in your editor.
5. Return to noteui and browse the preview.
6. Press `/` and search for a word in the note.

At that point, you already have the basic noteui loop working.

## Where to go next

- Want install help for your OS? Read [Installation](installation.md).
- Want common noteui workflows? Read [First notes workflow](first-notes.md).
- Want full behavior details? Read the [Usage guide](../guide/usage.md).
