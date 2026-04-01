# Installation

The recommended way to install noteui is from the pre-built release archives on GitHub:

<https://github.com/atbuy/noteui/releases>

## Release archive formats

- Linux: `.tar.gz`
- macOS: `.tar.gz`
- Windows: `.zip`

!!! tip

    If you just want to use noteui, prefer release binaries over building from source.

## Install from a release archive

=== "Linux"

    Extract the release archive:

    ```bash
    tar -xzf noteui-<version>-linux-amd64.tar.gz
    ```

    Run the binary:

    ```bash
    chmod +x noteui-<version>-linux-amd64
    ./noteui-<version>-linux-amd64
    ```

    Install it on your `PATH`:

    ```bash
    mv noteui-<version>-linux-amd64 noteui
    sudo mv noteui /usr/local/bin/
    ```

=== "macOS"

    Extract the archive:

    ```bash
    tar -xzf noteui-<version>-darwin-arm64.tar.gz
    ```

    Run it:

    ```bash
    chmod +x noteui-<version>-darwin-arm64
    ./noteui-<version>-darwin-arm64
    ```

    Install it globally:

    ```bash
    mv noteui-<version>-darwin-arm64 noteui
    sudo mv noteui /usr/local/bin/
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

    If you want easier access, rename it to `noteui.exe` and place it in a directory on your `PATH`.

## Build from source

!!! note

    Building from source is mainly useful for contributors or users who specifically want to build the binary themselves. Most users should prefer the release archives above.

If you prefer to build noteui yourself:

```bash
make build
./bin/noteui
```

Or run directly from source:

```bash
go run ./cmd/noteui
```

Install from source:

```bash
make install
```

## Verify the version

```bash
noteui --version
```

## Next step

Continue with [Getting started](getting-started.md) or [Your first notes workflow](first-notes.md).
