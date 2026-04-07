# Encrypted notes

noteui supports encrypted note bodies for users who want notes stored in encrypted form on disk while still being able to preview and edit them through the app.

## How it works

- the note body is encrypted
- the encrypted state is reflected in frontmatter
- noteui can decrypt for preview or editing workflows when needed
- edited content can be re-encrypted when written back

## What this is good for

- personal notes that you want to keep encrypted on disk
- workflows where file portability still matters
- users who want encryption without adopting a separate note storage format

## Important constraints

!!! warning

    This is an application workflow for encrypted note bodies. It is not a general-purpose secret-management system.

- noteui stores the note body encrypted on disk, but it still treats the file as part of your normal notes tree
- the encrypted flag lives in frontmatter as `encrypted: true`
- encrypted notes still rely on your own filesystem, backup, and SSH hygiene

## Editing behavior

When you open an encrypted note through noteui:

- noteui decrypts the body for the editing workflow
- the encrypted marker is removed from the editable temporary content
- the file is written back encrypted when the edit completes successfully

Outside noteui, encrypted notes remain ordinary files that contain encrypted bodies. You can move, rename, or sync them with normal file tools, but editing the encrypted payload directly in another editor is usually not useful.

### Atomic writes

Noteui writes the encrypted file back using an atomic rename: the new content is written to a sibling temporary file first, then the original path is replaced in a single filesystem operation. This means the original file remains intact if the process is interrupted mid-write. You will never end up with a half-written, unreadable encrypted blob from a crash or a full disk.

### Encrypted note history

noteui saves a version of the encrypted blob to `.noteui-history/` before and after every encrypted edit cycle. If re-encryption ever produces content you cannot decrypt, press `H` on the note to open the version history modal and restore an earlier working version.

The history list shows `encrypted` as the preview text for each version, since the raw blob is not decrypted for display. Restoring a version saves the current (broken) blob as a new history entry first, so the restore is itself undoable.

## Frontmatter signal

Encrypted notes use the `encrypted` frontmatter field.

Example:

```yaml
---
encrypted: true
---
```

## Sync behavior

Encrypted notes can still participate in note sync because sync works on the note files and sync metadata rather than on decrypted in-memory text.

Important boundaries:

- `noteui-sync` does not store your passphrase
- `.noteui-sync/` does not store decrypted note bodies
- another machine still needs the passphrase in the current session before the note can be edited or previewed as decrypted text

If you use encryption and sync together, treat sync as transport for the encrypted note file and sync metadata, not as a secrets-management layer.

## Related workflows

- [Usage guide](../guide/usage.md)
- [Sync guide](../guide/sync.md)
- [Configuration reference](../reference/configuration.md)
- [Storage and state](../reference/storage-and-state.md)
