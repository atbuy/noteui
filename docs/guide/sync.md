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
- `ctrl+s` toggles the selected note between `sync: shared` and `sync: local`
- synced notes are checked against the remote state after startup
- `F` opens the in-app default sync profile picker

## Shared notes

A shared note uses `sync: shared` in its frontmatter and is permanently synced to the remote. Unlike `sync: synced`, a shared note cannot be toggled with `S`; use `ctrl+s` instead, which toggles between `sync: shared` and `sync: local`.

```yaml
---
sync: shared
---
```

Shared notes appear in the tree with a `◆` marker instead of `●`. They participate in all sync operations identically to `sync: synced` notes.

## Understanding sync state in the tree

- hollow red `○`: local-only note
- green `●`: synced note with a previously successful sync record
- blue `◆`: shared note with a previously successful sync record
- orange blinking marker: sync, import, or remote-delete action in progress
- filled red `●`: note is intended to be synced, but noteui has a conflict or the latest sync check failed
- filled red `◆`: shared note with a conflict or failed sync check
- muted placeholder row: note exists on the server but not in the local notes tree yet

At startup, noteui uses the last healthy local sync record immediately. If the background sync later finds a conflict or remote problem, that note falls back to the red state.

## Resolving conflicts

A sync conflict means noteui kept your local note unchanged and wrote the remote body to a separate conflict copy beside it.

Use this workflow:

1. Select the conflicted synced note.
2. Press `ctrl+e` to open the sync details modal, which shows both copies side-by-side and displays when the conflict occurred.
3. Use `h`/`l` or left/right to choose which version to keep, then press `Enter` to apply.
4. Alternatively, press `O` to open the conflict copy in your editor for manual merging.

Important details:

- the original local note stays canonical for future sync
- editing only the conflict copy does not resolve the conflict
- the conflict state clears only after a later successful sync of the original note
- the conflict copy is left on disk intentionally as a safety file

If you prefer to inspect the file directly, the conflict copy is written beside the original note with a timestamped name such as `note.conflict-YYYYMMDD-HHMMSS.md`.

## Diagnosing unhealthy sync states

When a synced note turns red, press `ctrl+e` to open the sync details modal. It shows:

- a plain-English description of the issue
- how long ago the note was last successfully synced
- for conflicts: how long ago the conflict occurred
- a suggested next step

From the sync details modal you can also take recovery actions directly:

- press `r` to retry the sync without closing the modal first
- press `u` (only for "Remote copy missing") to unlink the note locally; this removes the sync record and resets the note to `sync: local` without making a network call

## Viewing sync history

Press `Y` to open the sync timeline, which shows a scrollable history of recent sync runs for the current workspace. Each entry displays:

- a status icon: `✓` for success, `⚡` for a run that completed with conflicts, `✗` for a run that failed
- the timestamp and sync profile used
- a summary of what changed (notes registered, updated, conflicts) or the error message

The timeline is also available from the command palette as **View Sync Timeline**. Sync history is persisted in `.noteui-sync/sync-events.jsonl` and kept up to the last 200 runs.

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

## Per-workspace sync isolation

If you run multiple workspaces and sync is configured, every workspace will sync to the same `remote_root` by default. This means notes from workspace A appear as remote-only placeholders in workspace B and vice versa, and pressing `I` in workspace B imports workspace A's notes into the wrong directory.

The fix is to add a `sync_remote_root` field to each workspace that needs its own remote path:

```toml
[sync]
default_profile = "homebox"

[sync.profiles.homebox]
ssh_host = "notes-prod"
remote_root = "/srv/noteui"      # fallback for any workspace without an override
remote_bin = "/usr/local/bin/noteui-sync"

[workspaces.work]
root = "/home/alice/notes/work"
label = "Work"
sync_remote_root = "/srv/noteui/work"

[workspaces.personal]
root = "/home/alice/notes/personal"
label = "Personal"
sync_remote_root = "/srv/noteui/personal"
```

`sync_remote_root` overrides the profile's `remote_root` for all sync operations originating from that workspace: push, pull, import, conflict resolution, and remote delete. The remote directories do not need to exist in advance; they are created on first sync.

The workspace picker displays the effective remote path under each workspace entry so you can confirm the isolation before switching.

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
- If a note shows "Remote copy missing", press `ctrl+e` and then `u` to unlink it locally, or sync again to recreate the remote copy.

For more debugging steps, see [Troubleshooting](../reference/troubleshooting.md).
