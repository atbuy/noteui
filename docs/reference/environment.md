# Environment variables

This page documents the environment variables noteui reads and the WebDAV
credential fallback file that works alongside them.

## `NOTES_ROOT`

Overrides the default notes directory. When set, this bypasses configured workspace profiles for that session and disables in-app workspace switching.

Default:

```text
$HOME/notes
```

Use this when:

- your notes live somewhere else
- you want separate note collections
- you are testing noteui against a temporary notes tree

## `NOTEUI_CONFIG`

Points noteui at a specific config file path.

Use this when:

- you want to keep config outside the default config directory
- you maintain multiple noteui setups
- you want per-project config behavior

This only changes `config.toml` lookup. It does not move the WebDAV credential
fallback file; that still lives under `noteui/secrets.toml` inside the user
config directory.

## `NOTEUI_EDITOR`

Overrides the editor used to open notes.

Resolution order is:

1. `NOTEUI_EDITOR`
2. `EDITOR`
3. `nvim`

## `XDG_STATE_HOME`

Changes the location of the noteui state file.

Default state path:

```text
$HOME/.local/state/noteui/state.json
```

## `XDG_DATA_HOME`

Influences the user trash location used when deleting notes or categories.

This matters because noteui deletes into trash rather than immediately hard-deleting content.

## WebDAV credential fallback file

For WebDAV sync profiles, `username_env`, `password_env`, and `token_env` hold
environment variable names, not the secret values themselves.

When noteui resolves one of those names at sync time, it uses this order:

1. the current process environment
2. `noteui/secrets.toml` inside the user config directory, if the env var is
   unset or empty

The default `secrets.toml` path depends on the platform:

- Linux and other XDG-style systems: `$XDG_CONFIG_HOME/noteui/secrets.toml` or
  `$HOME/.config/noteui/secrets.toml`
- macOS: `$HOME/Library/Application Support/noteui/secrets.toml`
- Windows: `%AppData%\noteui\secrets.toml`

This file is separate from `config.toml`. Setting `NOTEUI_CONFIG` does not
change where noteui looks for `secrets.toml`.

### File format

Use a flat TOML file with top-level string keys whose names exactly match the
configured env var names.

Example:

```toml
NOTEUI_NEXTCLOUD_USERNAME = "alice"
NOTEUI_NEXTCLOUD_PASSWORD = "app-password-here"
NOTEUI_WEBDAV_TOKEN = "app-token-for-another-profile"
```

Practical rules:

- keep the file flat; noteui reads top-level string keys by exact name
- env vars win over `secrets.toml` when both define the same key
- you can keep credentials for multiple sync profiles in the same file
- noteui reads the file at sync time, not at startup or config load
- malformed TOML causes the WebDAV auth step to fail until the file is fixed
- noteui does not create or update this file for you

!!! warning

    `secrets.toml` stores plain-text credentials. Keep it readable only by your
    user account, and do not commit it into a repository or sync it to a shared
    location by accident.

## Example shell setup

=== "bash / zsh"

    ```bash
    export NOTES_ROOT="$HOME/my-notes"
    export NOTEUI_EDITOR="nvim"
    export NOTEUI_CONFIG="$HOME/.config/noteui/config.toml"
    ```

=== "PowerShell"

    ```powershell
    $env:NOTES_ROOT = "$HOME\my-notes"
    $env:NOTEUI_EDITOR = "nvim"
    $env:NOTEUI_CONFIG = "$HOME\.config\noteui\config.toml"
    ```
