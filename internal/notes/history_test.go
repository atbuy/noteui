package notes

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSaveVersionDeduplicates(t *testing.T) {
	root := t.TempDir()
	relPath := "work/plan.md"
	content := "# Plan\n\nBody"

	require.NoError(t, SaveVersion(root, relPath, content))

	entries, err := Versions(root, relPath)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// Saving the same content again should be a no-op.
	require.NoError(t, SaveVersion(root, relPath, content))
	entries, err = Versions(root, relPath)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestSaveVersionSavesDistinctVersions(t *testing.T) {
	root := t.TempDir()
	relPath := "note.md"

	require.NoError(t, SaveVersion(root, relPath, "version one"))
	// Sleep briefly to ensure distinct timestamps in the version ID.
	time.Sleep(2 * time.Millisecond)
	require.NoError(t, SaveVersion(root, relPath, "version two"))

	entries, err := Versions(root, relPath)
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestVersionsReturnedNewestFirst(t *testing.T) {
	root := t.TempDir()
	relPath := "note.md"

	// Write version files directly with known timestamps to avoid relying on
	// sub-second sleep (version IDs have second-level precision).
	dir := historyDir(root, relPath)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	ids := []string{
		"20260401-100000-aaaaaaaa",
		"20260401-100001-bbbbbbbb",
		"20260401-100002-cccccccc",
	}
	for _, id := range ids {
		require.NoError(t, os.WriteFile(filepath.Join(dir, id+".md"), []byte("content "+id), 0o644))
	}

	entries, err := Versions(root, relPath)
	require.NoError(t, err)
	require.Len(t, entries, 3)

	// Newest first.
	require.True(t, entries[0].Timestamp.After(entries[1].Timestamp))
	require.True(t, entries[1].Timestamp.After(entries[2].Timestamp))
}

func TestVersionsEmptyWhenNoneExist(t *testing.T) {
	root := t.TempDir()
	entries, err := Versions(root, "nonexistent.md")
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestSaveVersionPrunes(t *testing.T) {
	root := t.TempDir()
	relPath := "note.md"

	// Save MaxVersionsPerNote+5 distinct versions.
	for i := range MaxVersionsPerNote + 5 {
		content := "version " + string(rune('A'+i%26)) + "_" + time.Now().String()
		require.NoError(t, SaveVersion(root, relPath, content))
		time.Sleep(time.Millisecond)
	}

	entries, err := Versions(root, relPath)
	require.NoError(t, err)
	require.LessOrEqual(t, len(entries), MaxVersionsPerNote)
}

func TestRestoreVersionAtomicAndUndoable(t *testing.T) {
	root := t.TempDir()
	notePath := filepath.Join(root, "note.md")
	relPath := "note.md"

	// Write two version files directly with known timestamps.
	dir := historyDir(root, relPath)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	oldContent := "# Original\n\nOriginal body"
	newContent := "# Updated\n\nUpdated body"
	oldID := "20260401-090000-aaaaaaaa"
	newID := "20260401-100000-bbbbbbbb"
	require.NoError(t, os.WriteFile(filepath.Join(dir, oldID+".md"), []byte(oldContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, newID+".md"), []byte(newContent), 0o644))

	// The note on disk currently holds a third distinct content (not in history).
	currentContent := "# Current\n\nCurrent body"
	require.NoError(t, os.WriteFile(notePath, []byte(currentContent), 0o644))

	entries, err := Versions(root, relPath)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	// Restore the oldest version.
	require.NoError(t, RestoreVersion(root, notePath, relPath, oldID))

	// File should now contain the original content.
	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Equal(t, oldContent, string(data))

	// RestoreVersion should have saved currentContent as a new version (undoable).
	entries2, err := Versions(root, relPath)
	require.NoError(t, err)
	require.Greater(t, len(entries2), len(entries))

	// Verify the newest entry contains the pre-restore content.
	require.Equal(t, "# Current", entries2[0].FirstLine)
}

func TestRestoreVersionErrorOnMissingVersion(t *testing.T) {
	root := t.TempDir()
	notePath := filepath.Join(root, "note.md")
	require.NoError(t, os.WriteFile(notePath, []byte("content"), 0o644))

	err := RestoreVersion(root, notePath, "note.md", "nonexistent-id")
	require.Error(t, err)

	// File should be untouched.
	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Equal(t, "content", string(data))
}

func TestVersionFirstLineEncrypted(t *testing.T) {
	root := t.TempDir()
	relPath := "secret.md"

	encryptedContent := "---\nencrypted: true\n---\nAGFiY2Q="
	require.NoError(t, SaveVersion(root, relPath, encryptedContent))

	entries, err := Versions(root, relPath)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "encrypted", entries[0].FirstLine)
}

func TestHistoryDirHiddenFromNoteDiscover(t *testing.T) {
	root := t.TempDir()
	relPath := "note.md"
	notePath := filepath.Join(root, relPath)
	require.NoError(t, os.WriteFile(notePath, []byte("# Note\n\nBody"), 0o644))

	// Save a version so the history dir is created.
	require.NoError(t, SaveVersion(root, relPath, "# Note\n\nBody"))

	// Discover should find the note but not any file inside .noteui-history/.
	discovered, err := Discover(root)
	require.NoError(t, err)

	for _, n := range discovered {
		require.NotContains(t, n.RelPath, HistoryDirName)
	}
}
