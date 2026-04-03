# FAQ

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

In the `[sync]` section of your config file. See the [Sync guide](guide/sync.md) and the [Configuration reference](reference/configuration.md).

## Why does a note appear as remote-only?

That means the note exists on the sync server but has not been imported into the current notes root yet.

Use `i` to import the selected remote-only note or `I` to import all missing synced notes.

## Where should I look for common problems?

Start with [Troubleshooting](reference/troubleshooting.md). It covers editor launch problems, sync setup errors, remote-only notes, and contributor docs build issues.
