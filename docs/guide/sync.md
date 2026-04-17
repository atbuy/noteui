# Sync guide

This guide explains how noteui sync works in practice. noteui supports two sync backends: SSH (the original backend) and WebDAV (for syncing through Nextcloud or any WebDAV-capable server).

## What sync does

noteui sync is opt-in and note-based.

- notes without `sync: synced` stay local-only
- notes with `sync: synced` are tracked through the configured remote profile
- noteui keeps local sync bookkeeping in `.noteui-sync/` inside the notes root

Sync is not a full bidirectional live filesystem mirror. noteui refreshes remote metadata automatically, but remote-only notes are imported on demand.

## Requirements

### SSH backend

- `noteui` on the local machine
- `noteui-sync` available on the remote machine
- SSH access from the local machine to the remote machine
- a writable remote storage directory for noteui sync data

### WebDAV backend

- `noteui` on the local machine
- a WebDAV server (Nextcloud, ownCloud, Apache with mod_dav, etc.)
- HTTP(S) access from the local machine to the server
- a writable directory on the WebDAV server

## SSH setup

1. Install or build both binaries.
2. Put `noteui-sync` on the remote machine somewhere callable over SSH.
3. Choose a remote sync root such as `/srv/noteui`.
4. Add a sync profile to your config:

```toml
[sync]
default_profile = "homebox"

[sync.profiles.homebox]
kind = "ssh"
ssh_host = "notes-prod"
remote_root = "/srv/noteui"
remote_bin = "/usr/local/bin/noteui-sync"
```

5. Start noteui with that config.

The `kind` field is optional for SSH profiles; it defaults to `"ssh"` when omitted, so existing configs work without changes.

## WebDAV setup

For most users, there are only two path rules to remember:

- `webdav_url` points to your WebDAV user endpoint, not the final notes directory
- `remote_root` points to the directory under that endpoint where noteui should store synced notes

For Nextcloud, the endpoint format is usually:

```text
https://<host>/remote.php/dav/files/<username>
```

If you want synced notes to live in a Nextcloud folder named `Notes`, configure:

- `webdav_url = "https://<host>/remote.php/dav/files/<username>"`
- `remote_root = "/Notes"`

Do not append the notes folder directly to `webdav_url`. noteui combines `webdav_url` and `remote_root` for you.

1. Find your WebDAV user endpoint.
2. Choose a remote directory under that endpoint, such as `/Notes` or `/Notes/personal`.
3. Add a sync profile:

```toml
[sync]
default_profile = "cloud"

[sync.profiles.cloud]
kind = "webdav"
webdav_url = "https://cloud.example.com/remote.php/dav/files/alice"
remote_root = "/Notes"
auth = "basic"
username_env = "NOTEUI_NEXTCLOUD_USERNAME"
password_env = "NOTEUI_NEXTCLOUD_PASSWORD"
```

4. Export the environment variables before starting noteui:

```sh
export NOTEUI_NEXTCLOUD_USERNAME="alice"
export NOTEUI_NEXTCLOUD_PASSWORD="app-password-here"
noteui
```

5. Mark the notes you want synced with `sync: synced`, then run sync from noteui.

`remote_root` defaults to `"/noteui"` when omitted for WebDAV profiles.

### WebDAV path model

This is the exact relationship between the two path fields:

- `webdav_url` is the base endpoint for one authenticated WebDAV user
- `remote_root` is the subdirectory noteui uses under that endpoint
- `sync_remote_root` is a per-workspace override for `remote_root`; for WebDAV it uses the same semantics as `remote_root`

Examples:

| Goal | `webdav_url` | `remote_root` |
|------|--------------|---------------|
| Store notes in `Notes` | `https://cloud.example.com/remote.php/dav/files/alice` | `/Notes` |
| Store notes in `Notes/personal` | `https://cloud.example.com/remote.php/dav/files/alice` | `/Notes/personal` |
| Use the default noteui directory | `https://cloud.example.com/remote.php/dav/files/alice` | omit the field |

For WebDAV workspaces, `sync_remote_root` should also be a remote directory such as `/Notes/work`. It must not be a local filesystem path like `/home/alice/notes/work`.

### WebDAV auth modes

| Mode | Description |
|------|-------------|
| `basic` (default) | HTTP Basic Auth; `username_env` and `password_env` required |
| `bearer` | HTTP Bearer token; `token_env` required. Use for Nextcloud app tokens or OAuth access tokens |
| `none` | No authentication; for trusted LAN or pre-authenticated endpoints |

`username_env`, `password_env`, and `token_env` hold environment variable names, not the credentials themselves.

Example `bearer` profile:

```toml
[sync.profiles.cloud]
kind = "webdav"
webdav_url = "https://cloud.example.com/remote.php/dav/files/alice"
remote_root = "/Notes"
auth = "bearer"
token_env = "NOTEUI_WEBDAV_TOKEN"
```

Environment variables are resolved at sync time, not at config load. This means CI or headless environments can load the config even when credentials are not yet set, but it also means the variables must exist in the same environment that launches `noteui`.

In practice:

- exporting variables in one shell does not help if you start `noteui` from another shell, a desktop launcher, or a user service that does not inherit them
- for Nextcloud, an app password is usually the safest choice instead of your normal login password
- if noteui reports a missing WebDAV credential env var, verify the variable is exported before `noteui` starts

### How WebDAV storage works

The WebDAV backend stores real Markdown files at `<remote_root>/<rel_path>` on the server. This means:

- synced notes are viewable and editable through Nextcloud, ownCloud, or any WebDAV client
- noteui keeps its own metadata in `<remote_root>/.noteui-sync/` (note mappings and pins)
- the desktop noteui experience remains unchanged

More specifically:

- note bodies are stored as normal files at paths such as `<remote_root>/work/plan.md`
- note mappings are stored in `<remote_root>/.noteui-sync/notes/<id>.json`
- pinned-note metadata is stored in `<remote_root>/.noteui-sync/pins.json`
- noteui creates `remote_root` and `.noteui-sync/` automatically on first successful upload when they do not already exist
- an empty or newly created remote directory is valid; noteui does not require `.noteui-sync/` to exist before the first sync

Conflict copies are kept local-only, matching SSH behavior.

### WebDAV performance note

WebDAV is more request-heavy than SSH sync. A single sync run may need multiple HTTP requests for remote indexing, note content, metadata, and directory creation.

What to expect:

- the first sync to a new WebDAV `remote_root` is usually slower because noteui has to create the remote directory structure and metadata files
- later syncs are faster once that structure exists
- high-latency networks make WebDAV feel slower than SSH because WebDAV uses more round trips

If lowest latency matters more than WebDAV compatibility, the SSH backend is still the leaner option.

### Multiple backends in one config

You can define both SSH and WebDAV profiles in the same config. The profile picker (`F`) shows a `[ssh]` or `[webdav]` badge next to each profile name.

```toml
[sync]
default_profile = "cloud"

[sync.profiles.cloud]
kind = "webdav"
webdav_url = "https://cloud.example.com/remote.php/dav/files/alice"
auth = "basic"
username_env = "NOTEUI_WEBDAV_USER"
password_env = "NOTEUI_WEBDAV_PASSWORD"

[sync.profiles.backup]
kind = "ssh"
ssh_host = "backup-host"
remote_root = "/srv/noteui-backup"
remote_bin = "noteui-sync"
```

Only one profile is active per workspace at a time.

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
- `F` opens the in-app default sync profile picker and updates only `sync.default_profile`

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

For WebDAV profiles, `sync_remote_root` is still a remote directory under `webdav_url`, not a local directory on your machine. Example:

```toml
[workspaces.work]
root = "/home/alice/notes/work"
label = "Work"
sync_remote_root = "/Notes/work"

[workspaces.personal]
root = "/home/alice/notes/personal"
label = "Personal"
sync_remote_root = "/Notes/personal"
```

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
- If a note shows "Remote copy missing", sync again to recreate the remote copy. Use `ctrl+e` and then `u` only when you want to stop syncing that note and keep it local-only.

For more debugging steps, see [Troubleshooting](../reference/troubleshooting.md).
