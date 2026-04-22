package editor

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandPrefersConfiguredEditor(t *testing.T) {
	t.Setenv("NOTEUI_EDITOR", "helix")
	t.Setenv("EDITOR", "vim")

	cmd, err := Command("/tmp/note.md")
	require.NoError(t, err)
	if cmd.Path != "helix" {
		require.Failf(t, "assertion failed", "expected NOTEUI_EDITOR to win, got %q", cmd.Path)
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "/tmp/note.md" {
		require.Failf(t, "assertion failed", "unexpected command args: %#v", cmd.Args)
	}
}

func TestCommandFallsBackToEditorAndDefault(t *testing.T) {
	t.Run("EDITOR fallback", func(t *testing.T) {
		t.Setenv("NOTEUI_EDITOR", "")
		t.Setenv("EDITOR", "vim")

		cmd, err := Command("/tmp/note.md")
		require.NoError(t, err)
		if filepath.Base(cmd.Path) != "vim" {
			require.Failf(t, "assertion failed", "expected EDITOR to be used, got %q", cmd.Path)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		t.Setenv("NOTEUI_EDITOR", "")
		t.Setenv("EDITOR", "")

		cmd, err := Command("/tmp/note.md")
		require.NoError(t, err)
		if filepath.Base(cmd.Path) != "nvim" {
			require.Failf(t, "assertion failed", "expected default editor to be nvim, got %q", cmd.Path)
		}
	})
}

func TestCommandSupportsEditorArgsAndQuotes(t *testing.T) {
	t.Run("space separated args", func(t *testing.T) {
		t.Setenv("NOTEUI_EDITOR", "code -w")
		t.Setenv("EDITOR", "")

		cmd, err := Command("/tmp/note.md")
		require.NoError(t, err)
		require.Equal(t, "code", filepath.Base(cmd.Path))
		require.Equal(t, []string{"code", "-w", "/tmp/note.md"}, cmd.Args)
	})

	t.Run("quoted args", func(t *testing.T) {
		t.Setenv("NOTEUI_EDITOR", "")
		t.Setenv("EDITOR", `emacsclient -c --alternate-editor=""`)

		cmd, err := Command("/tmp/note.md")
		require.NoError(t, err)
		require.Equal(t, "emacsclient", filepath.Base(cmd.Path))
		require.Equal(t, []string{"emacsclient", "-c", "--alternate-editor=", "/tmp/note.md"}, cmd.Args)
	})
}

func TestCommandRejectsInvalidQuotedEditorCommand(t *testing.T) {
	t.Setenv("NOTEUI_EDITOR", `"code -w`)
	t.Setenv("EDITOR", "")

	_, err := Command("/tmp/note.md")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unterminated quote")
}
