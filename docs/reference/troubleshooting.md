# Troubleshooting

This page collects the most common noteui problems and the fastest ways to verify them.

## Editor does not open

noteui resolves the editor in this order:

1. `NOTEUI_EDITOR`
2. `EDITOR`
3. `nvim`

Check:

- that the chosen editor command exists on your `PATH`
- that `NOTEUI_EDITOR` or `EDITOR` is a real command string such as `code -w`, not a shell alias or shell function that only exists in interactive shells
- that the editor command can be launched manually from the same terminal environment

See [Environment variables](environment.md) for the full environment lookup behavior.

## Notes are not where I expected

By default, noteui uses:

```text
$HOME/notes
```

Check:

- whether `NOTES_ROOT` is set
- whether you started noteui from an environment that sets a different notes root
- whether you are looking for temporary notes under `.tmp` inside the notes root

See [Storage and state](storage-and-state.md) for the full storage layout.

## Config changes are ignored or startup warns about config errors

If the config file is missing, noteui uses defaults.

If the config file contains unknown keys or invalid values, noteui warns at startup and keeps the decoded portion of the file where possible instead of rewriting it.

Run `noteui +check-config` to validate the file without opening the TUI. It prints the resolved path, any errors, warnings, and keybinding conflicts, including ambiguous sort-menu and preview/todo chord overrides, and exits with code 1 if something is wrong.

Check:

- the file path from `NOTEUI_CONFIG`, if set
- otherwise the default lookup path: `noteui/config.toml` inside your user config directory
- field names against the [Configuration reference](configuration.md)
- sync profile names, preview styles, border styles, and code styles for invalid values
- that the specific key written from the UI is the one you expect:
  - theme picker / `noteui +set-theme`: `theme.name`
  - sync profile picker: `sync.default_profile`

## Sync does not start

Sync remains disabled when `sync.default_profile` is empty.

Check:

- that `[sync]` exists in the config
- that `default_profile` matches a real entry under `sync.profiles`
- that every configured profile includes the required fields for its backend kind:
  - SSH (`kind = "ssh"` or omitted): `ssh_host`, `remote_root`, `remote_bin`
  - WebDAV (`kind = "webdav"`): `webdav_url`, and `username_env`/`password_env` when `auth = "basic"`

See the [Sync guide](../guide/sync.md) for the expected config shape.

## SSH remote sync command fails

If noteui cannot run the remote helper, verify:

- SSH to the configured host works manually
- `remote_bin` points to the real `noteui-sync` path on the remote machine
- the remote user can write to `remote_root`
- the remote machine has the `noteui-sync` binary version you expect

If you installed only `noteui` and not `noteui-sync`, remote sync will not work.

## WebDAV sync fails

When WebDAV sync fails, first confirm the effective remote path:

- `webdav_url` should point to the WebDAV user endpoint
- `remote_root` should point to the directory under that endpoint where noteui should sync

For Nextcloud, the most common working pattern is:

```toml
webdav_url = "https://cloud.example.com/remote.php/dav/files/alice"
remote_root = "/Notes"
```

That means noteui syncs into:

```text
https://cloud.example.com/remote.php/dav/files/alice/Notes
```

Check these in order:

- verify the `webdav_url` is reachable
- verify `remote_root` is a remote directory under that endpoint, not a local filesystem path
- if using `sync_remote_root`, verify it also uses WebDAV-style remote paths such as `/Notes/work`
- check that the variable named in `token_env` or the variables named in `username_env` and `password_env` are set in the same environment that launches `noteui`, or that `noteui/secrets.toml` inside the user config directory contains those same variable names
- if you use `NOTEUI_CONFIG`, remember that it does not move `secrets.toml`
- if `secrets.toml` exists, verify it is valid TOML with flat top-level string keys
- if using Nextcloud, prefer an app password instead of your normal account password
- `http://` URLs are accepted but `https://` is recommended for anything outside a trusted LAN

Manual check with the exact same target noteui uses:

```sh
curl -u "$NOTEUI_NEXTCLOUD_USERNAME:$NOTEUI_NEXTCLOUD_PASSWORD" \
  -X PROPFIND \
  -H 'Depth: 1' \
  https://cloud.example.com/remote.php/dav/files/alice/Notes
```

How to interpret common failures:

- `401 Unauthorized`: the credentials are missing from the `noteui` process environment and `secrets.toml`, or the values are rejected by the server
- `404` or similar "could not be located": the effective path is wrong, or the server does not allow access to that path
- sync succeeds locally but notes do not appear where expected: double-check the combined `webdav_url + remote_root` path rather than either field in isolation
- a WebDAV auth error that mentions `secrets.toml`: the fallback file exists but cannot be read or parsed
- Nextcloud reports "Strict cookie not set" or intermittent connection resets on the first request: noteui keeps an HTTP cookie jar per sync client so the Nextcloud `nc_session_id` cookie set on the initial redirect is replayed on follow-up requests. If you still see this against an old build, update noteui
- `read tcp [IPv6]:port -> [IPv6]:443: read: connection reset by peer`: the server's IPv6 path resets connections while IPv4 works. Set `force_ipv4 = true` in the sync profile.
- `config warning: sync profile "x": force_ipv4 has no effect on SSH profiles` (same for `insecure_skip_tls_verify`, `ca_cert`): these fields only apply to WebDAV profiles. Remove them from the SSH profile or switch the profile to `kind = "webdav"`.
- `tls: failed to verify certificate for <IP> because it doesn't contain any IP SANs`: the certificate has no IP Subject Alternative Name so Go rejects it when connecting by IP address. Either connect by hostname (requires DNS to resolve it) or set `insecure_skip_tls_verify = true` in the profile.
- `x509: certificate signed by unknown authority`: the server uses a certificate signed by a private or internal CA not in the system trust store. Point `ca_cert` at your internal CA's PEM file, or set `insecure_skip_tls_verify = true` if you cannot install the cert.
- `sync profile "x" ca_cert: ...`: the `ca_cert` path does not exist or the file contains no valid PEM certificates. noteui validates `ca_cert` at startup so the error appears before sync is attempted. Fix the path or regenerate the PEM file.

An empty or brand-new remote root is valid. noteui creates the target directory and its `.noteui-sync/` metadata directory on first successful upload.

## WebDAV sync feels slow

WebDAV usually takes more round trips than SSH sync.

This is normal when:

- the remote root is new and noteui is creating directories and metadata
- the sync run needs to read remote mappings and note bodies
- the connection to the server has noticeable latency

In practice:

- the first sync to a new WebDAV root is usually the slowest
- later syncs are faster once the remote layout exists
- SSH sync is still the lower-latency backend when both are available

If WebDAV feels unusually slow even on a local network, compare noteui with a manual `curl` against the same WebDAV endpoint to rule out server-side latency.

## Notes show up as remote-only placeholders

That means noteui found synced note metadata on the server but no local file body yet.

Use:

- `i` to import the selected remote-only note
- `I` to import all missing synced notes

If imports are skipped, check whether a local file already exists at the target path. noteui avoids overwriting existing local files.

## Synced notes turn red after startup

Previously healthy synced notes start green from their saved local sync record. If they turn red after startup, the background sync found a real problem.

Press `ctrl+e` on the affected note to open the sync details modal. It shows the exact issue category, the active sync profile and effective remote root, how long ago the note was last successfully synced, and a suggested recovery step. From the modal you can press `r` to retry the sync immediately, or `u` to unlink the note locally if you want to stop syncing that note.

If the issue is "Remote copy missing", sync again first. noteui now recreates the remote copy when the local synced note still exists and the remote copy is gone.

To see a history of recent sync runs and when the problem first appeared, press `Y` to open the sync timeline.

Also check:

- that the sync profile is still valid
- that SSH access still works
- that the remote root is accessible
- whether the note now has a conflict copy beside it

## A synced note has a conflict

When noteui reports a conflict, it keeps the local note untouched and writes the remote body into a sibling conflict copy such as `note.conflict-YYYYMMDD-HHMMSS.md`.

Use this resolution flow:

- select the conflicted synced note
- press `ctrl+e` to open the sync details modal, which shows both versions side-by-side and reports how long ago the conflict occurred
- use `h`/`l` or left/right to choose which version to keep, then press `Enter` to apply
- alternatively, press `O` to open the conflict copy in your editor for manual merging, then save the original note and sync again

If the conflict does not clear, check:

- that you edited the original note rather than only the conflict copy
- that the next sync actually succeeded
- that the remote host and profile are still reachable

Repeated conflicts usually mean the same note is still being changed in two places before either side completes a clean sync cycle.

## Encryption is confusing across machines

Sync can move encrypted note files, but it does not move your passphrase.

Check:

- whether the note is encrypted through `encrypted: true`
- whether the other machine has imported the synced note yet
- whether you have entered the passphrase in the current session on that machine

For behavior details, see [Encrypted notes](../advanced/encryption.md).

## Docs render differently on GitHub Pages

If local docs look correct but GitHub Pages does not:

- confirm the `Documentation` workflow actually deployed the latest commit
- confirm GitHub Pages is publishing from GitHub Actions rather than a branch source
- confirm the docs workflow uses the pinned `zensical` version
- hard-refresh the browser or test in a private window to rule out cached CSS and JS

If Pages deploys only after a manual rerun, verify the workflow is still using separate `build` and `deploy` jobs rather than building and deploying in a single job.

See [Docs maintenance](docs-maintenance.md) for the docs pipeline setup.
