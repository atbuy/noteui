# Troubleshooting

This page collects the most common noteui problems and the fastest ways to verify them.

## Editor does not open

noteui resolves the editor in this order:

1. `NOTEUI_EDITOR`
2. `EDITOR`
3. `nvim`

Check:

- that the chosen editor command exists on your `PATH`
- that `NOTEUI_EDITOR` or `EDITOR` does not point to a shell alias that only works in interactive shells
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

## Config changes are ignored or startup falls back to defaults

If the config file is missing, noteui uses defaults.

If the config file contains unknown keys or invalid values, noteui rejects that load and falls back to defaults.

Check:

- the file path from `NOTEUI_CONFIG`, if set
- otherwise the default lookup path: `noteui/config.toml` inside your user config directory
- field names against the [Configuration reference](configuration.md)
- sync profile names, preview styles, border styles, and code styles for invalid values

## Sync does not start

Sync remains disabled when `sync.default_profile` is empty.

Check:

- that `[sync]` exists in the config
- that `default_profile` matches a real entry under `sync.profiles`
- that every configured profile includes `ssh_host`, `remote_root`, and `remote_bin`

See the [Sync guide](../guide/sync.md) for the expected config shape.

## Remote sync command fails

If noteui cannot run the remote helper, verify:

- SSH to the configured host works manually
- `remote_bin` points to the real `noteui-sync` path on the remote machine
- the remote user can write to `remote_root`
- the remote machine has the `noteui-sync` binary version you expect

If you installed only `noteui` and not `noteui-sync`, remote sync will not work.

## Notes show up as remote-only placeholders

That means noteui found synced note metadata on the server but no local file body yet.

Use:

- `i` to import the selected remote-only note
- `I` to import all missing synced notes

If imports are skipped, check whether a local file already exists at the target path. noteui avoids overwriting existing local files.

## Synced notes turn red after startup

Previously healthy synced notes start green from their saved local sync record. If they turn red after startup, the background sync found a real problem.

Check:

- that the sync profile is still valid
- that SSH access still works
- that the remote root is accessible
- whether the note now has a conflict copy beside it

## A synced note has a conflict

When noteui reports a conflict, it keeps the local note untouched and writes the remote body into a sibling conflict copy such as `note.conflict-YYYYMMDD-HHMMSS.md`.

Use this resolution flow:

- select the conflicted synced note
- press `O` to open the generated conflict copy
- open the original note too
- merge what you want into the original note
- save the original note and sync again

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

If a rerun fails with multiple `github-pages` artifacts, verify the workflow is using a unique artifact name per run attempt.

See [Docs maintenance](docs-maintenance.md) for the docs pipeline setup.
