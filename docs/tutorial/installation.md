# Installation

The recommended way to install noteui is from the pre-built release archives on GitHub:

<https://github.com/atbuy/noteui/releases>

## Release archive formats

- Linux: `.tar.gz`
- macOS: `.tar.gz`
- Windows: `.zip`

Each release archive includes both `noteui` and `noteui-sync`. Most users only need to run `noteui`; `noteui-sync` is used when you configure SSH-based sync.

!!! tip

    If you just want to use noteui, prefer release binaries over building from source.

## Choosing an install method

Use release archives if you want the fastest and simplest install.

Use `make install` if you are building from a local checkout and want both binaries installed through Go tooling.

Use `go run ./cmd/noteui` mainly for development or quick local testing, not as a long-term install path.

## Install from a release archive

=== "Linux"

    Extract the release archive:

    ```bash
    tar -xzf noteui-<version>-linux-amd64.tar.gz
    ```

    Run the binaries:

    ```bash
    chmod +x noteui-<version>-linux-amd64 noteui-sync-<version>-linux-amd64
    ./noteui-<version>-linux-amd64
    ```

    Install them on your `PATH`:

    ```bash
    mv noteui-<version>-linux-amd64 noteui
    sudo mv noteui /usr/local/bin/
    mv noteui-sync-<version>-linux-amd64 noteui-sync
    sudo mv noteui-sync /usr/local/bin/
    ```

=== "macOS"

    Extract the archive:

    ```bash
    tar -xzf noteui-<version>-darwin-arm64.tar.gz
    ```

    Run it:

    ```bash
    chmod +x noteui-<version>-darwin-arm64 noteui-sync-<version>-darwin-arm64
    ./noteui-<version>-darwin-arm64
    ```

    Install both binaries globally:

    ```bash
    mv noteui-<version>-darwin-arm64 noteui
    sudo mv noteui /usr/local/bin/
    mv noteui-sync-<version>-darwin-arm64 noteui-sync
    sudo mv noteui-sync /usr/local/bin/
    ```

    Use `darwin-arm64` on Apple Silicon and `darwin-amd64` on Intel Macs.

=== "Windows"

    Download the matching `.zip` archive and extract it using File Explorer or PowerShell.

    PowerShell example:

    ```powershell
    Expand-Archive .\noteui-<version>-windows-amd64.zip -DestinationPath .\noteui
    ```

    Run it:

    ```powershell
    .\noteui\noteui-<version>-windows-amd64.exe
    ```

    If you want easier access, rename `noteui-<version>-windows-amd64.exe` to `noteui.exe` and place it in a directory on your `PATH`. If you plan to use sync, also place `noteui-sync-<version>-windows-amd64.exe` somewhere available for your remote sync setup.

## Build from source

!!! note

    Building from source is mainly useful for contributors or users who specifically want to build the binary themselves. Most users should prefer the release archives above.

If you prefer to build noteui yourself:

```bash
make build
./bin/noteui
```

Install both binaries from source:

```bash
make install
```

Or run directly from source without installing:

```bash
go run ./cmd/noteui
```

## Verify the version

```bash
noteui --version
```

## Next step

Continue with [Getting started](getting-started.md), [Your first notes workflow](first-notes.md), or the [Sync guide](../guide/sync.md) if you want remote sync.
