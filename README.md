[![CI](https://github.com/atbuy/noteui/actions/workflows/ci.yml/badge.svg)](https://github.com/atbuy/noteui/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/atbuy/noteui/branch/main/graph/badge.svg)](https://codecov.io/gh/atbuy/noteui)

# noteui

`noteui` is a terminal UI for browsing and editing a filesystem-backed notes directory. It organizes notes into categories, supports temporary notes, renders previews in the terminal, and keeps lightweight local UI state such as pinned items and collapsed categories.

## Requirements

- Go 1.26.1 or newer
- A notes directory, defaulting to `$HOME/notes`
- A terminal editor if you want to open notes directly from the app. `noteui` checks `NOTEUI_EDITOR`, then `EDITOR`, and falls back to `nvim`.

## Build and Run

```bash
make build
./bin/noteui
```

Run directly from source:

```bash
go run ./cmd/noteui
```

Install the binary into your Go bin directory:

```bash
make install
```

Show the embedded version string:

```bash
./bin/noteui --version
```

## Configuration

Runtime behavior is controlled through a small set of environment variables and config files.

- `NOTES_ROOT`: root directory for notes. Defaults to `$HOME/notes`.
- `NOTEUI_CONFIG`: explicit path to the TOML config file. If unset, the app uses `$XDG_CONFIG_HOME/noteui/config.toml` or the platform default from `os.UserConfigDir()`.
- `NOTEUI_EDITOR`: editor binary used to open notes. Overrides `EDITOR`.
- `XDG_STATE_HOME`: controls where UI state is stored. The default state file path is `$HOME/.local/state/noteui/state.json`.

## Testing

Run the unit test suite locally with:

```bash
make test
```

To collect local coverage output:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## CI

GitHub Actions runs the unit tests, validates that the project still builds with `go build -buildvcs=false ./...`, and uploads coverage reports to Codecov on pushes and pull requests.

Release packaging remains in `.github/workflows/release-build.yml` and is only triggered for published GitHub releases.
