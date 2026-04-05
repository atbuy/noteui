# Environment variables

This page documents the environment variables noteui reads.

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
