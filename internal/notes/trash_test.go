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

func TestListTrashedEmpty(t *testing.T) {
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)
	root := t.TempDir()

	items, err := ListTrashed(root)
	require.NoError(t, err)
	require.Nil(t, items)
}

func TestListTrashedFiltersToRoot(t *testing.T) {
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)

	root1 := t.TempDir()
	root2 := t.TempDir()

	file1 := filepath.Join(root1, "alpha.md")
	file2 := filepath.Join(root2, "beta.md")
	require.NoError(t, os.WriteFile(file1, []byte("a"), 0o600))
	require.NoError(t, os.WriteFile(file2, []byte("b"), 0o600))

	_, err := TrashPath(file1)
	require.NoError(t, err)
	_, err = TrashPath(file2)
	require.NoError(t, err)

	items, err := ListTrashed(root1)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "alpha.md", items[0].Name)
}

func TestListTrashedSortedNewestFirst(t *testing.T) {
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)
	root := t.TempDir()

	// Write two .trashinfo files manually with different dates so we can control order.
	infoDir := filepath.Join(xdgData, "Trash", "info")
	filesDir := filepath.Join(xdgData, "Trash", "files")
	require.NoError(t, os.MkdirAll(infoDir, 0o700))
	require.NoError(t, os.MkdirAll(filesDir, 0o700))

	older := filepath.Join(root, "older.md")
	newer := filepath.Join(root, "newer.md")

	writeInfo := func(name, origPath, date string) {
		content := "[Trash Info]\nPath=" + origPath + "\nDeletionDate=" + date + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(infoDir, name+".trashinfo"), []byte(content), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(filesDir, name), []byte("x"), 0o600))
	}

	writeInfo("older.md", filepath.ToSlash(older), "2025-01-01T10:00:00")
	writeInfo("newer.md", filepath.ToSlash(newer), "2025-06-01T10:00:00")

	items, err := ListTrashed(root)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "newer.md", items[0].Name)
	require.Equal(t, "older.md", items[1].Name)
}

func TestListTrashedURLDecodedPath(t *testing.T) {
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)

	root := filepath.Join(t.TempDir(), "my notes")
	require.NoError(t, os.MkdirAll(root, 0o755))

	filePath := filepath.Join(root, "foo bar.md")
	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o600))

	_, err := TrashPath(filePath)
	require.NoError(t, err)

	items, err := ListTrashed(root)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, filePath, items[0].OriginalPath)
	require.Equal(t, "foo bar.md", items[0].Name)
}

func TestListTrashedSkipsMalformedEntries(t *testing.T) {
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)
	root := t.TempDir()

	infoDir := filepath.Join(xdgData, "Trash", "info")
	filesDir := filepath.Join(xdgData, "Trash", "files")
	require.NoError(t, os.MkdirAll(infoDir, 0o700))
	require.NoError(t, os.MkdirAll(filesDir, 0o700))

	// Valid entry.
	validPath := filepath.Join(root, "valid.md")
	validInfo := "[Trash Info]\nPath=" + filepath.ToSlash(validPath) + "\nDeletionDate=2025-01-01T12:00:00\n"
	require.NoError(t, os.WriteFile(filepath.Join(infoDir, "valid.md.trashinfo"), []byte(validInfo), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "valid.md"), []byte("ok"), 0o600))

	// Malformed entry (no Path= line).
	require.NoError(t, os.WriteFile(filepath.Join(infoDir, "bad.md.trashinfo"), []byte("[Trash Info]\n"), 0o600))

	items, err := ListTrashed(root)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "valid.md", items[0].Name)
}

func TestListTrashedName(t *testing.T) {
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)
	root := t.TempDir()

	filePath := filepath.Join(root, "note.md")
	require.NoError(t, os.WriteFile(filePath, []byte("body"), 0o600))

	result, err := TrashPath(filePath)
	require.NoError(t, err)

	items, err := ListTrashed(root)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "note.md", items[0].Name)
	require.Equal(t, result.TrashFilePath, items[0].TrashFilePath)
	require.Equal(t, result.TrashInfoPath, items[0].TrashInfoPath)
}
