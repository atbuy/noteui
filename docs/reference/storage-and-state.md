# Storage and state

This page explains what noteui stores and where it stores it.

## Notes

Your note content lives under the notes root.

Default:

```text
$HOME/notes
```

Supported note file extensions:

- `.md`
- `.txt`
- `.org`
- `.norg`

## Categories

Categories are directories below the notes root.

## Temporary notes

Temporary notes live under:

```text
<notes-root>/.tmp
```

## Deleted notes and categories

Deleted content is moved to the user trash.

!!! warning

    Delete actions are not implemented as immediate hard-delete operations inside the notes root.

## Local UI state

noteui stores UI state separately from your note files.

That state currently includes:

- pinned notes
- pinned categories
- collapsed categories
- sort mode

Default location:

```text
$HOME/.local/state/noteui/state.json
```

## Why this matters

This separation keeps your notes portable and easy to manage with external tools while still allowing noteui to remember interface state.
