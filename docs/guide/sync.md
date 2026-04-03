# Sync guide

This guide explains how noteui's SSH-based sync works in practice.

## What sync does

noteui sync is opt-in and note-based.

- notes without `sync: synced` stay local-only
- notes with `sync: synced` are tracked through the configured remote profile
- noteui keeps local sync bookkeeping in `.noteui-sync/` inside the notes root

Sync is not a full bidirectional live filesystem mirror. noteui refreshes remote metadata automatically, but remote-only notes are imported on demand.

## Requirements

You need:

- `noteui` on the local machine
- `noteui-sync` available on the remote machine
- SSH access from the local machine to the remote machine
- a writable remote storage directory for noteui sync data

## Basic setup

1. Install or build both binaries.
2. Put `noteui-sync` on the remote machine somewhere callable over SSH.
3. Choose a remote sync root such as `/srv/noteui`.
4. Add a sync profile to your config:

```toml
[sync]
default_profile = "homebox"

[sync.profiles.homebox]
ssh_host = "notes-prod"
remote_root = "/srv/noteui"
remote_bin = "/usr/local/bin/noteui-sync"
```

5. Start noteui with that config.

If `sync.default_profile` is empty, noteui does not attempt network sync.

## Marking notes for sync

Sync selection lives in note frontmatter:

```yaml
---
sync: synced
---
```

For local-only notes, either omit the field or set:

```yaml
---
sync: local
---
```

Inside noteui:

- `S` toggles the selected local note between `sync: local` and `sync: synced`
- synced notes are checked against the remote state after startup

## Understanding sync state in the tree

- hollow red `○`: local-only note
- green `●`: synced note with confirmed healthy remote state
- orange blinking marker: sync, import, or remote-delete action in progress
- filled red `●`: note is intended to be synced, but the current remote state is not yet confirmed healthy
- muted placeholder row: note exists on the server but not in the local notes tree yet

On startup, synced notes are treated as unconfirmed until the first remote check completes. This avoids showing stale healthy markers from old local metadata.

## Remote-only notes and import flows

When a note exists on the server but not locally, noteui shows a remote-only placeholder row.

Use:

- `i` to import the selected remote-only note
- `I` to import all missing synced notes

This also works as recovery. If you delete a synced note locally and the target path is still free, `I` can restore it from the server.

If a local file already exists at the target path, noteui skips that import instead of overwriting the local file.

## Removing the remote copy but keeping the local file

Use `U` on a synced local note to:

- delete only the remote copy
- keep the local file
- switch the note back to `sync: local`

Use this when you no longer want that note synced but do not want to delete the local content.

## How sync interacts with encrypted notes

Encrypted notes can still be synced, but sync should be thought of as transport for the note file and sync metadata.

- sync does not store your passphrase
- `.noteui-sync/` does not store decrypted note bodies
- another machine still needs the passphrase in the current session before the note can be edited or previewed as decrypted text

For the encryption workflow itself, see [Encrypted notes](../advanced/encryption.md).

## Common problems

- If sync never starts, check that `sync.default_profile` matches an existing profile name.
- If the remote command fails, verify `remote_bin` points to a real `noteui-sync` path on the remote host.
- If SSH works manually but sync still fails, confirm the remote user can write to `remote_root`.
- If notes appear as remote-only placeholders, import them with `i` or `I`.

For more debugging steps, see [Troubleshooting](../reference/troubleshooting.md).
