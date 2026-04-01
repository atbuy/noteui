package editor

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandPrefersConfiguredEditor(t *testing.T) {
	t.Setenv("NOTEUI_EDITOR", "helix")
	t.Setenv("EDITOR", "vim")

	cmd := Command("/tmp/note.md")
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

		cmd := Command("/tmp/note.md")
		if filepath.Base(cmd.Path) != "vim" {
			require.Failf(t, "assertion failed", "expected EDITOR to be used, got %q", cmd.Path)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		t.Setenv("NOTEUI_EDITOR", "")
		t.Setenv("EDITOR", "")

		cmd := Command("/tmp/note.md")
		if filepath.Base(cmd.Path) != "nvim" {
			require.Failf(t, "assertion failed", "expected default editor to be nvim, got %q", cmd.Path)
		}
	})
}
