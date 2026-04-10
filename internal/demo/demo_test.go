package demo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetup(t *testing.T) {
	root, cleanup, err := Setup()
	if err != nil {
		t.Fatalf("Setup() error: %v", err)
	}

	expected := []string{
		"journal.md",
		"inbox.md",
		"work/project-alpha.md",
		"work/meeting-notes.md",
		"personal/ideas.md",
		"personal/reading-list.md",
		"reference/commands.md",
		".tmp/scratch.md",
	}
	for _, rel := range expected {
		path := filepath.Join(root, rel)
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("expected file %s not found: %v", rel, statErr)
		}
	}

	cleanup()

	if _, err = os.Stat(root); !os.IsNotExist(err) {
		t.Errorf("cleanup() did not remove temp dir %s", root)
	}
}
