package editor

import (
	"path/filepath"
	"testing"
)

func TestCommandPrefersConfiguredEditor(t *testing.T) {
	t.Setenv("NOTEUI_EDITOR", "helix")
	t.Setenv("EDITOR", "vim")

	cmd := Command("/tmp/note.md")
	if cmd.Path != "helix" {
		t.Fatalf("expected NOTEUI_EDITOR to win, got %q", cmd.Path)
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "/tmp/note.md" {
		t.Fatalf("unexpected command args: %#v", cmd.Args)
	}
}

func TestCommandFallsBackToEditorAndDefault(t *testing.T) {
	t.Run("EDITOR fallback", func(t *testing.T) {
		t.Setenv("NOTEUI_EDITOR", "")
		t.Setenv("EDITOR", "vim")

		cmd := Command("/tmp/note.md")
		if filepath.Base(cmd.Path) != "vim" {
			t.Fatalf("expected EDITOR to be used, got %q", cmd.Path)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		t.Setenv("NOTEUI_EDITOR", "")
		t.Setenv("EDITOR", "")

		cmd := Command("/tmp/note.md")
		if cmd.Path != "nvim" {
			t.Fatalf("expected default editor to be nvim, got %q", cmd.Path)
		}
	})
}
