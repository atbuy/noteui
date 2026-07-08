package sync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidRemoteRelPathAndSafeJoin(t *testing.T) {
	root := t.TempDir()

	safe := []struct {
		in   string
		want string
	}{
		{"work/plan.md", "work/plan.md"},
		{"ideas.md", "ideas.md"},
		{"a/b/../c.md", "a/c.md"},
		{"  spaced.md  ", "spaced.md"},
	}
	for _, tc := range safe {
		got, ok := validRemoteRelPath(tc.in)
		require.Truef(t, ok, "expected %q to be accepted", tc.in)
		require.Equal(t, tc.want, got)
		joined, err := safeJoin(root, tc.in)
		require.NoErrorf(t, err, "safeJoin rejected safe path %q", tc.in)
		require.Equal(t, filepath.Join(root, filepath.FromSlash(tc.want)), joined)
	}

	unsafe := []string{
		"",
		".",
		"..",
		"../secret.md",
		"../../etc/passwd",
		"a/../../../.bashrc", // bypasses a naive "../" prefix check
		"foo/../..",
		"/etc/passwd",
		"/",
	}
	for _, in := range unsafe {
		_, ok := validRemoteRelPath(in)
		require.Falsef(t, ok, "expected %q to be rejected", in)
		_, err := safeJoin(root, in)
		require.Errorf(t, err, "expected safeJoin to reject %q", in)
	}
}

// TestSyncRootRefusesRemoteNoteEscapingRoot proves that a malicious or
// compromised remote cannot make a pull write outside the notes root by
// serving a note whose rel_path escapes it.
func TestSyncRootRefusesRemoteNoteEscapingRoot(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	notePath := filepath.Join(root, "note.md")
	original := "---\nsync: synced\n---\n# Note\n\nlocal\n"
	require.NoError(t, os.WriteFile(notePath, []byte(original), 0o644))

	client := localClient{}
	cfg := testSyncConfig(remote)

	// First sync registers the note and creates the local record + remote files.
	_, err := SyncRoot(context.Background(), root, "", cfg, nil, nil, client)
	require.NoError(t, err)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Len(t, records, 1)
	var rec NoteRecord
	for _, r := range records {
		rec = r
	}

	// Simulate a compromised remote: rewrite the stored note metadata with a
	// rel_path that escapes the notes root and bump the revision so the next
	// sync treats it as a remote change to pull.
	poisoned, err := loadRemoteNote(remote, rec.ID)
	require.NoError(t, err)
	poisoned.RelPath = "../pwned.md"
	poisoned.Revision = 99
	require.NoError(t, saveRemoteNote(remote, poisoned, "---\nsync: synced\n---\nowned\n"))

	_, err = SyncRoot(context.Background(), root, "", cfg, nil, nil, client)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsafe remote note path")

	// The location the unpatched code would have written to must stay empty,
	// and the local note must be untouched.
	wouldEscapeTo := filepath.Join(root, filepath.FromSlash("../pwned.md"))
	_, statErr := os.Stat(wouldEscapeTo)
	require.Truef(t, os.IsNotExist(statErr), "content escaped the notes root to %s", wouldEscapeTo)

	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Equal(t, original, string(data))
}

// TestImportRemoteNotesRejectsUnsafeRelPath covers the bootstrap import path,
// whose original guard only rejected a literal "../" prefix.
func TestImportRemoteNotesRejectsUnsafeRelPath(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	client := localClient{}
	cfg := testSyncConfig(remote)

	require.NoError(t, saveRemoteNote(remote, remoteNoteFile{
		ID:       "n_malicious",
		RelPath:  "a/../../../pwned.md",
		Revision: 1,
	}, "---\nsync: synced\n---\nowned\n"))

	_, err := ImportRemoteNotes(context.Background(), root, "", cfg, client)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid remote note path")

	wouldEscapeTo := filepath.Join(root, filepath.FromSlash("a/../../../pwned.md"))
	_, statErr := os.Stat(wouldEscapeTo)
	require.Truef(t, os.IsNotExist(statErr), "content escaped the notes root to %s", wouldEscapeTo)
}

// TestServerRejectsUnsafeRelPath is the server-side defense in depth: an SSH
// remote must not store a poisoned rel_path that would later be replayed to
// every client.
func TestServerRejectsUnsafeRelPath(t *testing.T) {
	remote := t.TempDir()
	server := Server{}

	var rpcErr *RPCError

	_, err := server.RegisterNote(RegisterNoteRequest{RemoteRoot: remote, RelPath: "../escape.md", Content: "x"})
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, ErrCodeInvalid, rpcErr.Code)

	reg, err := server.RegisterNote(RegisterNoteRequest{RemoteRoot: remote, RelPath: "ok.md", Content: "x"})
	require.NoError(t, err)

	_, err = server.PushNote(PushNoteRequest{
		RemoteRoot:       remote,
		NoteID:           reg.ID,
		ExpectedRevision: reg.Revision,
		RelPath:          "../evil.md",
		Content:          "y",
	})
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, ErrCodeInvalid, rpcErr.Code)

	_, err = server.UpdateNotePath(UpdateNotePathRequest{
		RemoteRoot:       remote,
		NoteID:           reg.ID,
		ExpectedRevision: reg.Revision,
		RelPath:          "a/../../../evil.md",
	})
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, ErrCodeInvalid, rpcErr.Code)
}
