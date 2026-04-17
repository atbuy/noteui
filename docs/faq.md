# FAQ

## Can I try noteui without setting anything up?

Yes. Run `noteui --demo` to launch the UI against a bundle of built-in sample
notes copied into a throwaway temporary directory. Your real notes directory
is not touched. The demo data is removed automatically when you quit.

See the [Demo mode](guide/usage.md#demo-mode) section of the usage guide for
details about what is included.

## Where are my notes stored?

By default, under `$HOME/notes`, unless you set `NOTES_ROOT`.

## Does noteui use a database?

No. noteui works with normal files and directories.

## What file types are supported?

- `.md`
- `.txt`
- `.org`
- `.norg`

## Does noteui change my existing files?

It works directly with your note files for operations like create, rename, move, delete, todo actions, and encrypted note workflows.

## Where do deleted files go?

They are moved to the user trash.

## Where is the config file?

If `NOTEUI_CONFIG` is not set, noteui looks in your user config directory under `noteui/config.toml`.

## Where is the WebDAV credential fallback file?

If you use `secrets.toml`, noteui looks for it in your user config directory
under `noteui/secrets.toml`. On Linux that is usually
`~/.config/noteui/secrets.toml`.

This file is separate from `config.toml`. Setting `NOTEUI_CONFIG` does not move
it.

## Where is noteui’s state stored?

By default in:

```text
$HOME/.local/state/noteui/state.json
```

## How do I change the editor used to open notes?

Set `NOTEUI_EDITOR`, or fall back to `EDITOR`.

## Can I keep using my own editor and sync tools?

Yes. That is one of the main reasons noteui uses normal files on disk.

## Where do I configure sync?

In the `[sync]` section of your config file. noteui supports two sync backends: SSH (the original) and WebDAV (for Nextcloud or any WebDAV server). See the [Sync guide](guide/sync.md) and the [Configuration reference](reference/configuration.md).

## Can I sync through Nextcloud or another WebDAV server?

Yes. Set `kind = "webdav"` on a sync profile, provide the `webdav_url`, and configure authentication. Synced notes are stored as real Markdown files on the server, so you can view and edit them through Nextcloud or any WebDAV client.

The simple rule is:

- `webdav_url` points to your WebDAV user endpoint
- `remote_root` points to the notes directory under that endpoint

For example, to sync into a Nextcloud folder named `Notes`:

```toml
webdav_url = "https://cloud.example.com/remote.php/dav/files/alice"
remote_root = "/Notes"
```

Export the environment variables named by `username_env` and `password_env`
before starting `noteui`, or put the same variable names in `noteui/secrets.toml`
inside your user config directory if you launch noteui from a desktop
environment that drops shell env vars. See the [Sync guide](guide/sync.md#webdav-setup)
for the full setup.

## Why is WebDAV sync slower than SSH?

WebDAV usually needs more network round trips than SSH sync, especially on the first sync to a new remote root.

That is expected because noteui has to:

- inspect remote metadata
- fetch note and mapping information
- create the remote directory and `.noteui-sync/` structure when needed

Later syncs are usually faster once the remote structure exists.

## Why does a note appear as remote-only?

That means the note exists on the sync server but has not been imported into the current notes root yet.

Use `i` to import the selected remote-only note or `I` to import all missing synced notes.

## How do I resolve a sync conflict?

Select the conflicted synced note, press `O` to open the generated conflict copy, merge the content you want into the original note, then sync again.

The conflict copy is a safety file, not the canonical note path for future sync.

## Where should I look for common problems?

Start with [Troubleshooting](reference/troubleshooting.md). It covers editor launch problems, sync setup errors, remote-only notes, and contributor docs build issues.
