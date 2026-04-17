package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomicCreatesAndOverwritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")

	require.NoError(t, WriteFileAtomic(path, []byte("first"), 0o600))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "first", string(data))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	require.NoError(t, WriteFileAtomic(path, []byte("second"), 0o644))

	data, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "second", string(data))

	info, err = os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestWriteFileAtomicReturnsErrorWhenParentDirectoryMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "note.txt")

	err := WriteFileAtomic(path, []byte("body"), 0o644)
	require.Error(t, err)
}

func TestSyncDirBestEffortIgnoresMissingDirectory(t *testing.T) {
	syncDirBestEffort(filepath.Join(t.TempDir(), "missing"))
}
