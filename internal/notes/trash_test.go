package notes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrashPathReturnsResult(t *testing.T) {
	root := t.TempDir()
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)

	filePath := filepath.Join(root, "note.md")
	if err := os.WriteFile(filePath, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := TrashPath(filePath)
	require.NoError(t, err)

	require.Equal(t, filePath, result.OriginalPath)
	require.NotEmpty(t, result.TrashFilePath)
	require.NotEmpty(t, result.TrashInfoPath)

	if _, err := os.Stat(result.TrashFilePath); err != nil {
		t.Fatalf("expected file in trash at %q: %v", result.TrashFilePath, err)
	}
	if _, err := os.Stat(result.TrashInfoPath); err != nil {
		t.Fatalf("expected .trashinfo at %q: %v", result.TrashInfoPath, err)
	}
}

func TestRestoreFromTrash(t *testing.T) {
	root := t.TempDir()
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)

	filePath := filepath.Join(root, "note.md")
	if err := os.WriteFile(filePath, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := TrashPath(filePath)
	require.NoError(t, err)

	// File should be gone from original location.
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected original to be gone after trash, stat err=%v", err)
	}

	require.NoError(t, RestoreFromTrash(result))

	// File should be back at original location.
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, "content", string(data))

	// .trashinfo should be removed.
	if _, err := os.Stat(result.TrashInfoPath); !os.IsNotExist(err) {
		t.Fatalf("expected .trashinfo to be removed after restore, stat err=%v", err)
	}
}

func TestRestoreFromTrashConflict(t *testing.T) {
	root := t.TempDir()
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)

	filePath := filepath.Join(root, "note.md")
	if err := os.WriteFile(filePath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := TrashPath(filePath)
	require.NoError(t, err)

	// Put a new file at the original path.
	if err := os.WriteFile(filePath, []byte("new file"), 0o600); err != nil {
		t.Fatal(err)
	}

	err = RestoreFromTrash(result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestRestoreFromTrashDirectory(t *testing.T) {
	root := t.TempDir()
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)

	dirPath := filepath.Join(root, "mycat")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, "note.md"), []byte("body"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := TrashPath(dirPath)
	require.NoError(t, err)

	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		t.Fatalf("expected directory to be gone after trash, stat err=%v", err)
	}

	require.NoError(t, RestoreFromTrash(result))

	if _, err := os.Stat(filepath.Join(dirPath, "note.md")); err != nil {
		t.Fatalf("expected directory and contents to be restored: %v", err)
	}
}

func TestRestoreFromTrashEmptyResult(t *testing.T) {
	err := RestoreFromTrash(TrashResult{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing path information")
}
