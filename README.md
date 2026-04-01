[![CI](https://github.com/atbuy/noteui/actions/workflows/ci.yml/badge.svg)](https://github.com/atbuy/noteui/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/atbuy/noteui/branch/main/graph/badge.svg)](https://codecov.io/gh/atbuy/noteui)

# noteui

`noteui` is a terminal notes application for people who keep their notes as normal files and directories. It adds a fast keyboard-driven interface for browsing, searching, previewing, organizing, and editing those files without moving them into a database or a proprietary format.

If you want a TUI note app but still want your notes to remain plain files you can edit with any editor, `noteui` is built for that workflow.

## Highlights

- browse notes and categories in a tree view
- preview notes directly in the terminal
- search by title, path, content preview, and tags
- create, rename, move, and delete notes or categories
- work with temporary notes separately from your main notes
- create and manage todo notes
- pin important notes and categories
- customize theme, preview behavior, icons, and keybindings
- keep your notes as regular files on disk

## Installation

### Recommended: download a release

Pre-built binaries are available on the GitHub releases page:

<https://github.com/atbuy/noteui/releases>

That is the easiest way to install `noteui` if you just want to use the application.

Release files are published as archives.

- Linux and macOS releases are published as `.tar.gz`
- Windows releases are published as `.zip`

#### Linux

Download the archive for your platform, then extract it:

```bash
tar -xzf noteui-<version>-linux-amd64.tar.gz
```

Make the binary executable if needed and run it:

```bash
chmod +x noteui-<version>-linux-amd64
./noteui-<version>-linux-amd64
```

If you want it on your `PATH`, rename and move it somewhere like `/usr/local/bin`:

```bash
mv noteui-<version>-linux-amd64 noteui
sudo mv noteui /usr/local/bin/
```

#### macOS

Download the matching macOS archive and extract it:

```bash
tar -xzf noteui-<version>-darwin-arm64.tar.gz
```

Run it directly:

```bash
chmod +x noteui-<version>-darwin-arm64
./noteui-<version>-darwin-arm64
```

To install it globally:

```bash
mv noteui-<version>-darwin-arm64 noteui
sudo mv noteui /usr/local/bin/
```

On Apple Silicon, prefer the `darwin-arm64` build. On Intel Macs, use the `darwin-amd64` build.

#### Windows

Windows releases are provided as `.zip` archives.

You can extract them using File Explorer:

1. Download the `noteui-<version>-windows-*.zip` archive.
2. Right-click it.
3. Choose `Extract All...`.
4. Open the extracted folder and run `noteui.exe`.

Or extract from PowerShell:

```powershell
Expand-Archive .\noteui-<version>-windows-amd64.zip -DestinationPath .\noteui
```

Then run:

```powershell
.\noteui\noteui-<version>-windows-amd64.exe
```

If you want, you can rename it to `noteui.exe` and place it in a directory that is already on your `PATH`.

### Build from source

If you prefer to build it yourself:

```bash
make build
./bin/noteui
```

Or run it directly from source:

```bash
go run ./cmd/noteui
```

To install it into your Go bin directory:

```bash
make install
```

To check the binary version:

```bash
./bin/noteui --version
```

## How noteui stores notes

noteui works with normal files under a root notes directory.

- folders are treated as categories
- supported note file types are `.md`, `.txt`, `.org`, and `.norg`
- notes at the root are treated as uncategorized in the main notes view
- hidden directories are skipped
- temporary notes are stored under a `.tmp` directory inside the notes root

Because the files stay on disk in a normal structure, you can still:

- edit them directly with your editor outside noteui
- sync them with git, Syncthing, iCloud, Dropbox, or similar tools
- reorganize them using your normal shell or file manager

## First run

By default, noteui looks for notes in:

```text
$HOME/notes
```

If that directory does not exist, noteui creates it automatically when needed.

Launch the app with:

```bash
noteui
```

Or, if you built it locally:

```bash
./bin/noteui
```

## Everyday usage

The app is built around a few simple workflows.

### Browse and preview

- move through categories and notes with the keyboard
- preview the selected note in the right-hand pane
- switch focus between the tree and preview with `tab`

### Search

- press `/` to search
- search matches note titles, paths, preview text, and tags

### Create and organize

- create a new note with `n`
- create a temporary note with `N`
- create a todo note with `T`
- create a category with `C`
- rename with `R`
- move with `m`
- delete with `d`

### Pin important items

- pin the current note or category with `p`
- open the pins view with `P`

### Get help inside the app

- press `?` to open the in-app help view

## Default keybindings

The most important default keys are:

- `j` / `k` or arrow keys: move selection
- `h` / `l` or left/right: collapse or expand categories
- `enter` or `o`: open note in editor
- `/`: search
- `tab`: switch focused pane
- `n`: new note
- `N`: new temporary note
- `T`: new todo list
- `C`: create category
- `R`: rename
- `m`: move
- `d`: delete
- `A`: add tags
- `p`: pin current item
- `P`: show pins
- `B`: toggle preview privacy
- `L`: toggle preview line numbers
- `s`: toggle sort order
- `?`: show help
- `q`: quit

There are more keys for preview navigation, todo actions, heading jumps, encryption flows, and paging, and the in-app help is the best place to see the full list.

## Configuration

noteui can be used with no config file at all, but it also supports a TOML config file for customizing the interface and behavior.

### Environment variables

- `NOTES_ROOT`: choose a custom notes directory instead of `$HOME/notes`
- `NOTEUI_CONFIG`: point to a specific config file
- `NOTEUI_EDITOR`: choose which editor opens notes from noteui
- `XDG_STATE_HOME`: change where noteui stores local UI state
- `XDG_DATA_HOME`: affects the trash location used for deletions

Editor resolution order is:

- `NOTEUI_EDITOR`
- `EDITOR`
- `nvim`

### Config file location

If `NOTEUI_CONFIG` is not set, noteui looks for:

```text
<user config dir>/noteui/config.toml
```

### What you can customize

The config supports:

- theme selection and color overrides
- typography settings
- icons
- modal styling
- markdown preview behavior
- syntax highlighting style
- preview privacy defaults
- line number defaults
- keybinding overrides

Built-in theme names currently include:

- `default`
- `nord`
- `gruvbox`
- `catppuccin`
- `catppuccin-mocha`
- `catppuccin-latte`
- `solarized-light`
- `paper`
- `onedark`
- `kanagawa`
- `dracula`
- `everforest`
- `everforest-dark`
- `tokyo-night-storm`
- `tokyonight-storm`
- `github-light`
- `github-dark`
- `carbonfox`

Example config:

```toml
dashboard = true

[theme]
name = "nord"
border_style = "rounded"

[preview]
render_markdown = true
style = "dark"
syntax_highlight = true
code_style = "monokai"
privacy = false
line_numbers = true

[keys]
open = ["enter", "o"]
search = ["/"]
show_help = ["?"]
```

## Notes, tags, todos, and encryption

### Plain files first

noteui is built around normal note files, not a custom storage engine. You can keep using your preferred editor and existing note-management setup.

### Frontmatter

noteui understands simple frontmatter blocks delimited by `---`.

Recognized fields include:

- `tags`
- `encrypted`
- `private`

Example:

```yaml
---
tags: work, project-x
encrypted: false
private: false
---
```

### Todo support

Todo notes are supported directly in the app. You can create todo note templates and perform todo actions such as toggling, adding, editing, and deleting items.

### Encrypted notes

noteui supports encrypted note bodies for workflows where you want notes stored in encrypted form on disk but still opened and previewed through the app when needed.

## Temporary notes

Temporary notes are useful for scratch work, quick capture, or short-lived drafts that you do not want mixed into your main note tree immediately.

They live under the note root’s `.tmp` directory and have their own list mode in the UI.

## What gets saved locally

noteui stores local UI state separately from your notes.

That state includes:

- pinned notes
- pinned categories
- collapsed categories
- sort mode

By default, the state file lives at:

```text
$HOME/.local/state/noteui/state.json
```

Your actual notes remain in your notes directory.

## Updating and releases

New builds are published on the releases page:

<https://github.com/atbuy/noteui/releases>

If you installed from a release binary, the normal update path is simply downloading a newer release and replacing the old binary.

## Development and project status

If you are contributing or building locally, useful commands are:

```bash
make build
make test
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

GitHub Actions runs CI for tests and builds, and Codecov is used for coverage reporting.
