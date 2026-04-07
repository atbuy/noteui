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

Archived temporary notes created from the in-app archive action are moved into:

```text
<notes-root>/archive/tmp
```

## Deleted notes and categories

Deleted content is moved to the user trash.

!!! warning

    Delete actions are not implemented as immediate hard-delete operations inside the notes root.

## Local UI state

noteui stores UI state separately from your note files.

That state currently includes:

- collapsed categories
- sort mode
- temporary-note pins
- local-only pins for roots without sync configured

Default location:

```text
$HOME/.local/state/noteui/state.json
```

## Sync metadata

When sync is configured for a notes root, noteui also stores hidden sync metadata inside that root:

```text
<notes-root>/.noteui-sync/
  config.json
  pins.json
  notes/
    <note-id>.json
  conflicts/
    <note-id>.json
```

That metadata stores sync-only state such as:

- the local client ID and selected profile
- stable note IDs for synced notes
- last known remote revisions and content hashes
- last sync attempt and last sync error state
- synced pin state for synced notes and categories
- conflict records and generated conflict-copy paths

`.noteui-sync/` does not store decrypted note bodies, passphrases, or general UI state.

A practical consequence is that noteui can remember prior sync state locally, but it still verifies remote state on startup before showing synced notes as healthy. Until that first remote check finishes, synced notes are treated as unconfirmed in the tree.

When importing missing synced notes, noteui creates or updates this directory locally as part of the bootstrap and recovery flow.

## Note version history

noteui automatically saves versions of a note to a hidden history directory inside the notes root:

```text
<notes-root>/.noteui-history/
  <escaped-note-path>/
    20260407-150405-a1b2c3d4.md
    20260407-091142-bb3e77ff.md
    ...
```

The directory name under `.noteui-history/` is derived from the note's relative path with `/` replaced by `__`, so `work/plan.md` maps to `.noteui-history/work__plan.md/`.

Each version file name encodes the UTC save time and the first 8 hex characters of the SHA-256 hash of the content.

Versions are saved automatically at these moments:

- before opening an encrypted note for editing
- after a successful re-encryption when an encrypted edit is finished
- after encrypting a note
- after decrypting a note
- after the editor closes for a non-encrypted note
- after any todo edit

noteui keeps at most 50 versions per note. Older versions are pruned automatically after each save.

`.noteui-history/` is hidden from the notes tree; noteui's file discovery skips directories that start with `.`.

## Why this matters

This separation keeps your notes portable and easy to manage with external tools while still allowing noteui to remember interface state and sync bookkeeping.
