package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
)

type localClient struct{ server Server }

func (c localClient) PullIndex(_ context.Context, _ config.SyncProfile, req PullIndexRequest) (PullIndexResponse, error) {
	return c.server.PullIndex(req)
}

func (c localClient) FetchNote(_ context.Context, _ config.SyncProfile, req FetchNoteRequest) (FetchNoteResponse, error) {
	return c.server.FetchNote(req)
}

func (c localClient) RegisterNote(_ context.Context, _ config.SyncProfile, req RegisterNoteRequest) (RegisterNoteResponse, error) {
	return c.server.RegisterNote(req)
}

func (c localClient) PushNote(_ context.Context, _ config.SyncProfile, req PushNoteRequest) (PushNoteResponse, error) {
	return c.server.PushNote(req)
}

func (c localClient) UpdateNotePath(_ context.Context, _ config.SyncProfile, req UpdateNotePathRequest) (UpdateNotePathResponse, error) {
	return c.server.UpdateNotePath(req)
}

func (c localClient) DeleteNote(_ context.Context, _ config.SyncProfile, req DeleteNoteRequest) (DeleteNoteResponse, error) {
	return c.server.DeleteNote(req)
}

func (c localClient) PinsGet(_ context.Context, _ config.SyncProfile, req PinsGetRequest) (PinsGetResponse, error) {
	return c.server.PinsGet(req)
}

func (c localClient) PinsPut(_ context.Context, _ config.SyncProfile, req PinsPutRequest) (PinsPutResponse, error) {
	return c.server.PinsPut(req)
}

func testSyncConfig(remoteRoot string) config.SyncConfig {
	return config.SyncConfig{
		DefaultProfile: "local",
		Profiles: map[string]config.SyncProfile{
			"local": {
				SSHHost:    "localhost",
				RemoteRoot: remoteRoot,
				RemoteBin:  "noteui-sync",
			},
		},
	}
}

func TestSyncRootRegistersUpdatesAndPersistsPins(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	notePath := filepath.Join(root, "work", "plan.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, []byte(`---
sync: synced
---
# Plan

Body
`), 0o644))

	client := localClient{}
	cfg := testSyncConfig(remote)
	result, err := SyncRoot(context.Background(), root, cfg, []string{"work/plan.md"}, []string{"work"}, client)
	require.NoError(t, err)
	require.Equal(t, 1, result.RegisteredNotes)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Len(t, records, 1)
	var noteID string
	for id := range records {
		noteID = id
	}

	pins, err := LoadPins(root)
	require.NoError(t, err)
	require.Equal(t, []string{noteID}, pins.PinnedNoteIDs)
	require.Equal(t, []string{"work"}, pins.PinnedCategories)

	require.NoError(t, os.WriteFile(notePath, []byte(`---
sync: synced
---
# Plan

Updated body
`), 0o644))
	result, err = SyncRoot(context.Background(), root, cfg, []string{"work/plan.md"}, []string{"work"}, client)
	require.NoError(t, err)
	require.Equal(t, 1, result.UpdatedNotes)

	fetched, err := client.FetchNote(context.Background(), cfg.Profiles["local"], FetchNoteRequest{RemoteRoot: remote, NoteID: noteID})
	require.NoError(t, err)
	require.Contains(t, fetched.Note.Content, "Updated body")
}

func TestSyncRootPullsRemoteChangeWhenLocalIsUnmodified(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	notePath := filepath.Join(root, "note.md")
	require.NoError(t, os.WriteFile(notePath, []byte(`---
sync: synced
---
# Note

Local
`), 0o644))

	client := localClient{}
	cfg := testSyncConfig(remote)
	_, err := SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	var rec NoteRecord
	for _, r := range records {
		rec = r
	}

	_, err = client.PushNote(context.Background(), cfg.Profiles["local"], PushNoteRequest{
		RemoteRoot:       remote,
		NoteID:           rec.ID,
		ExpectedRevision: rec.RemoteRev,
		RelPath:          rec.RelPath,
		Content: `---
sync: synced
---
# Note

Remote change
`,
		Encrypted: false,
	})
	require.NoError(t, err)

	result, err := SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)
	require.True(t, result.NotesChanged)

	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Contains(t, string(data), "Remote change")
}

func TestSyncRootCreatesAndClearsConflictCopy(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	notePath := filepath.Join(root, "work", "plan.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, []byte(`---
sync: synced
---
# Plan

Local body
`), 0o644))

	client := localClient{}
	cfg := testSyncConfig(remote)
	_, err := SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Len(t, records, 1)
	var rec NoteRecord
	for _, candidate := range records {
		rec = candidate
	}

	_, err = client.PushNote(context.Background(), cfg.Profiles["local"], PushNoteRequest{
		RemoteRoot:       remote,
		NoteID:           rec.ID,
		ExpectedRevision: rec.RemoteRev,
		RelPath:          rec.RelPath,
		Content: `---
sync: synced
---
# Plan

Remote body
`,
		Encrypted: false,
	})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(notePath, []byte(`---
sync: synced
---
# Plan

Local edit
`), 0o644))

	result, err := SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)
	require.Equal(t, 1, result.Conflicts)

	localRaw, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Contains(t, string(localRaw), "Local edit")
	require.NotContains(t, string(localRaw), "Remote body")

	records, err = LoadNoteRecords(root)
	require.NoError(t, err)
	rec = records[rec.ID]
	require.NotNil(t, rec.Conflict)
	require.Equal(t, "conflict", rec.LastSyncError)
	require.FileExists(t, ConflictPath(root, rec.ID))

	conflictCopyPath := filepath.Join(root, filepath.FromSlash(rec.Conflict.CopyPath))
	conflictRaw, err := os.ReadFile(conflictCopyPath)
	require.NoError(t, err)
	require.Contains(t, string(conflictRaw), "Remote body")

	require.NoError(t, os.WriteFile(notePath, []byte(`---
sync: synced
---
# Plan

Merged body
`), 0o644))

	result, err = SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)
	require.Equal(t, 1, result.UpdatedNotes)

	records, err = LoadNoteRecords(root)
	require.NoError(t, err)
	rec = records[rec.ID]
	require.Nil(t, rec.Conflict)
	require.Empty(t, rec.LastSyncError)
	require.NoFileExists(t, ConflictPath(root, rec.ID))
	require.FileExists(t, conflictCopyPath)

	fetched, err := client.FetchNote(context.Background(), cfg.Profiles["local"], FetchNoteRequest{
		RemoteRoot: remote,
		NoteID:     rec.ID,
	})
	require.NoError(t, err)
	require.Contains(t, fetched.Note.Content, "Merged body")
}

func TestSyncRootPreservesRemoteIdentityForUnchangedLocalMove(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	oldPath := filepath.Join(root, "work", "plan.md")
	newPath := filepath.Join(root, "archive", "plan.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(oldPath), 0o755))
	require.NoError(t, os.WriteFile(oldPath, []byte(`---
sync: synced
---
# Plan

Body
`), 0o644))

	client := localClient{}
	cfg := testSyncConfig(remote)
	_, err := SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Len(t, records, 1)
	var original NoteRecord
	for _, candidate := range records {
		original = candidate
	}

	require.NoError(t, os.MkdirAll(filepath.Dir(newPath), 0o755))
	require.NoError(t, os.Rename(oldPath, newPath))

	result, err := SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)
	require.Equal(t, 1, result.UpdatedNotes)
	require.Zero(t, result.RegisteredNotes)

	records, err = LoadNoteRecords(root)
	require.NoError(t, err)
	require.Len(t, records, 1)
	updated := records[original.ID]
	require.Equal(t, "archive/plan.md", updated.RelPath)

	index, err := client.PullIndex(context.Background(), cfg.Profiles["local"], PullIndexRequest{RemoteRoot: remote})
	require.NoError(t, err)
	require.Len(t, index.Notes, 1)
	require.Equal(t, original.ID, index.Notes[0].ID)
	require.Equal(t, "archive/plan.md", index.Notes[0].RelPath)
}

func TestSSHClientSendsPayloadOverStdin(t *testing.T) {
	var gotStdin []byte
	var gotName string
	var gotArgs []string
	client := SSHClient{
		Run: func(_ context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
			gotStdin = append([]byte(nil), stdin...)
			gotName = name
			gotArgs = append([]string(nil), args...)
			return []byte(`{"data":{"pins":{}}}`), nil
		},
	}
	profile := config.SyncProfile{SSHHost: "notes-prod", RemoteBin: "/usr/local/bin/noteui-sync"}
	_, err := client.PinsGet(context.Background(), profile, PinsGetRequest{RemoteRoot: "/srv/noteui"})
	require.NoError(t, err)
	require.Equal(t, "ssh", gotName)
	require.Equal(t, []string{"notes-prod", "/usr/local/bin/noteui-sync", "pins_get"}, gotArgs)
	require.JSONEq(t, `{"remote_root":"/srv/noteui"}`, string(gotStdin))
}

func TestSSHClientFallsBackToLegacyArgPayload(t *testing.T) {
	calls := 0
	client := SSHClient{
		Run: func(_ context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
			calls++
			require.Equal(t, "ssh", name)
			if calls == 1 {
				require.Equal(t, []string{"notes-prod", "/usr/local/bin/noteui-sync", "pins_get"}, args)
				require.JSONEq(t, `{"remote_root":"/srv/noteui"}`, string(stdin))
				return nil, fmt.Errorf("exit status 2: usage: noteui-sync <operation> <json-payload>")
			}
			require.Nil(t, stdin)
			require.Equal(t, []string{"notes-prod", "/usr/local/bin/noteui-sync", "pins_get", `{"remote_root":"/srv/noteui"}`}, args)
			return []byte(`{"data":{"pins":{}}}`), nil
		},
	}
	profile := config.SyncProfile{SSHHost: "notes-prod", RemoteBin: "/usr/local/bin/noteui-sync"}
	_, err := client.PinsGet(context.Background(), profile, PinsGetRequest{RemoteRoot: "/srv/noteui"})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
}

func TestSSHClientDeleteNoteFallsBackForUnknownOperation(t *testing.T) {
	calls := 0
	client := SSHClient{
		Run: func(_ context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
			calls++
			require.Equal(t, "ssh", name)
			switch calls {
			case 1:
				require.Equal(t, []string{"notes-prod", "/usr/local/bin/noteui-sync", "delete_note"}, args)
				require.JSONEq(t, `{"remote_root":"/srv/noteui","note_id":"n1","expected_revision":"7"}`, string(stdin))
				return []byte(`{"error":{"code":"invalid_request","message":"unknown operation"}}`), nil
			case 2:
				require.Equal(t, []string{"notes-prod", "/usr/local/bin/noteui-sync", "pins_get"}, args)
				require.JSONEq(t, `{"remote_root":"/srv/noteui"}`, string(stdin))
				return []byte(`{"data":{"pins":{"pinned_note_ids":["n1","n2"],"pinned_categories":["work"]}}}`), nil
			case 3:
				require.Equal(t, []string{"notes-prod", "/usr/local/bin/noteui-sync", "pins_put"}, args)
				require.JSONEq(t, `{"remote_root":"/srv/noteui","pins":{"pinned_note_ids":["n2"],"pinned_categories":["work"]}}`, string(stdin))
				return []byte(`{"data":{"pins":{"pinned_note_ids":["n2"],"pinned_categories":["work"]}}}`), nil
			case 4:
				require.Nil(t, stdin)
				require.Equal(t, []string{"notes-prod", "sh", "-lc", "rm -f '/srv/noteui/notes/n1.json' '/srv/noteui/content/n1.note'"}, args)
				return nil, nil
			default:
				require.FailNow(t, "unexpected extra fallback call")
				return nil, nil
			}
		},
	}
	profile := config.SyncProfile{SSHHost: "notes-prod", RemoteBin: "/usr/local/bin/noteui-sync"}
	_, err := client.DeleteNote(context.Background(), profile, DeleteNoteRequest{RemoteRoot: "/srv/noteui", NoteID: "n1", ExpectedRevision: "7"})
	require.NoError(t, err)
	require.Equal(t, 4, calls)
}

func TestImportRemoteNotesBootstrapsFreshRoot(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	client := localClient{}
	cfg := testSyncConfig(remote)

	first, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "work/plan.md",
		Content: `---
sync: synced
---
# Plan

Remote body
`,
		Encrypted: false,
	})
	require.NoError(t, err)
	second, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "ideas.md",
		Content: `---
sync: synced
---
# Ideas
`,
		Encrypted: false,
	})
	require.NoError(t, err)
	_, err = client.PinsPut(context.Background(), cfg.Profiles["local"], PinsPutRequest{
		RemoteRoot: remote,
		Pins:       Pins{PinnedNoteIDs: []string{first.ID}, PinnedCategories: []string{"work"}},
	})
	require.NoError(t, err)

	result, err := ImportRemoteNotes(context.Background(), root, cfg, client)
	require.NoError(t, err)
	require.Equal(t, 2, result.ImportedNotes)
	require.True(t, result.NotesChanged)
	require.ElementsMatch(t, []string{"work/plan.md"}, result.PinnedNoteRelPaths)
	require.Equal(t, []string{"work"}, result.PinnedCategories)

	data, err := os.ReadFile(filepath.Join(root, "work", "plan.md"))
	require.NoError(t, err)
	require.Contains(t, string(data), "Remote body")
	_, err = os.Stat(filepath.Join(root, "ideas.md"))
	require.NoError(t, err)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Contains(t, records, first.ID)
	require.Contains(t, records, second.ID)

	pins, err := LoadPins(root)
	require.NoError(t, err)
	require.Equal(t, []string{first.ID}, pins.PinnedNoteIDs)
	require.Equal(t, []string{"work"}, pins.PinnedCategories)
}

func TestImportRemoteNoteImportsOnlySelectedNote(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	client := localClient{}
	cfg := testSyncConfig(remote)
	first, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "work/plan.md",
		Content: `---
sync: synced
---
# Plan

Remote body
`,
		Encrypted: false,
	})
	require.NoError(t, err)
	_, err = client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "ideas.md",
		Content: `---
sync: synced
---
# Ideas
`,
		Encrypted: false,
	})
	require.NoError(t, err)

	result, err := ImportRemoteNote(context.Background(), root, cfg, first.ID, client)
	require.NoError(t, err)
	require.Equal(t, 1, result.ImportedNotes)
	require.Zero(t, result.SkippedImports)
	require.Len(t, result.RemoteOnlyNotes, 1)
	require.Equal(t, "ideas.md", result.RemoteOnlyNotes[0].RelPath)

	_, err = os.Stat(filepath.Join(root, "work", "plan.md"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(root, "ideas.md"))
	require.Error(t, err)
}

func TestImportRemoteNoteImportsOnlySelectedDuplicatePath(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	client := localClient{}
	cfg := testSyncConfig(remote)
	first, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "work/plan.md",
		Content: `---
sync: synced
---
# Plan

First body
`,
		Encrypted: false,
	})
	require.NoError(t, err)
	second, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "work/plan.md",
		Content: `---
sync: synced
---
# Plan

Second body
`,
		Encrypted: false,
	})
	require.NoError(t, err)

	result, err := ImportRemoteNote(context.Background(), root, cfg, second.ID, client)
	require.NoError(t, err)
	require.Equal(t, 1, result.ImportedNotes)
	require.Zero(t, result.SkippedImports)
	require.Len(t, result.RemoteOnlyNotes, 1)
	require.Equal(t, first.ID, result.RemoteOnlyNotes[0].ID)

	data, err := os.ReadFile(filepath.Join(root, "work", "plan.md"))
	require.NoError(t, err)
	require.Contains(t, string(data), "Second body")

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Contains(t, records, second.ID)
	require.Equal(t, "work/plan.md", records[second.ID].RelPath)
	require.NotContains(t, records, first.ID)

	result, err = ImportRemoteNote(context.Background(), root, cfg, first.ID, client)
	require.NoError(t, err)
	require.Equal(t, 1, result.ImportedNotes)
	require.Zero(t, result.SkippedImports)

	records, err = LoadNoteRecords(root)
	require.NoError(t, err)
	require.Contains(t, records, first.ID)
	require.Equal(t, "work/plan~"+duplicateImportSuffixCandidates(first.ID)[0]+".md", records[first.ID].RelPath)

	data, err = os.ReadFile(filepath.Join(root, filepath.FromSlash(records[first.ID].RelPath)))
	require.NoError(t, err)
	require.Contains(t, string(data), "First body")
}

func TestImportRemoteNotesImportsDuplicatePathsWithIDSuffix(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	client := localClient{}
	cfg := testSyncConfig(remote)
	_, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "work/plan.md",
		Content: `---
sync: synced
---
# Plan

First body
`,
		Encrypted: false,
	})
	require.NoError(t, err)
	_, err = client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "work/plan.md",
		Content: `---
sync: synced
---
# Plan

Second body
`,
		Encrypted: false,
	})
	require.NoError(t, err)

	result, err := ImportRemoteNotes(context.Background(), root, cfg, client)
	require.NoError(t, err)
	require.Equal(t, 2, result.ImportedNotes)
	require.Zero(t, result.SkippedImports)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Len(t, records, 2)
	paths := make(map[string]bool, len(records))
	for _, rec := range records {
		paths[rec.RelPath] = true
	}
	require.True(t, paths["work/plan.md"])

	globbed, err := filepath.Glob(filepath.Join(root, "work", "plan*.md"))
	require.NoError(t, err)
	require.Len(t, globbed, 2)
}

func TestImportRemoteNotesRestoresMissingTrackedNote(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	client := localClient{}
	cfg := testSyncConfig(remote)

	resp, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "work/plan.md",
		Content: `---
sync: synced
---
# Plan

Remote body
`,
		Encrypted: false,
	})
	require.NoError(t, err)
	require.NoError(t, SaveNoteRecord(root, NoteRecord{ID: resp.ID, RelPath: "work/plan.md", Class: ClassSynced, RemoteRev: resp.Revision}))

	result, err := ImportRemoteNotes(context.Background(), root, cfg, client)
	require.NoError(t, err)
	require.Equal(t, 1, result.ImportedNotes)
	require.Zero(t, result.SkippedImports)

	data, err := os.ReadFile(filepath.Join(root, "work", "plan.md"))
	require.NoError(t, err)
	require.Contains(t, string(data), "Remote body")
}

func TestImportRemoteNotesSkipsExistingLocalCollision(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	client := localClient{}
	cfg := testSyncConfig(remote)
	_, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "local.md",
		Content: `---
sync: synced
---
# Remote
`,
		Encrypted: false,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "local.md"), []byte("# Local\n"), 0o644))

	result, err := ImportRemoteNotes(context.Background(), root, cfg, client)
	require.NoError(t, err)
	require.Zero(t, result.ImportedNotes)
	require.Equal(t, 1, result.SkippedImports)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Empty(t, records)
}

func TestImportRemoteNotesHandlesEmptyRemote(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()

	result, err := ImportRemoteNotes(context.Background(), root, testSyncConfig(remote), localClient{})
	require.NoError(t, err)
	require.Zero(t, result.ImportedNotes)
	require.False(t, result.NotesChanged)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Empty(t, records)
}

func TestPullIndexIncludesRemoteTitle(t *testing.T) {
	remote := t.TempDir()
	client := localClient{}
	cfg := testSyncConfig(remote)
	_, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "work/plan.md",
		Content: `---
sync: synced
---
# Plan Title
`,
		Encrypted: false,
	})
	require.NoError(t, err)

	index, err := client.PullIndex(context.Background(), cfg.Profiles["local"], PullIndexRequest{RemoteRoot: remote})
	require.NoError(t, err)
	require.Len(t, index.Notes, 1)
	require.Equal(t, "Plan Title", index.Notes[0].Title)
}

func TestSyncRootReturnsRemoteOnlyPlaceholderInsteadOfMaterializingMissingNote(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	client := localClient{}
	cfg := testSyncConfig(remote)
	resp, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "work/plan.md",
		Content: `---
sync: synced
---
# Plan Title

Remote body
`,
		Encrypted: false,
	})
	require.NoError(t, err)
	require.NoError(t, SaveNoteRecord(root, NoteRecord{ID: resp.ID, RelPath: "work/plan.md", Class: ClassSynced, RemoteRev: resp.Revision}))

	result, err := SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)
	require.False(t, result.NotesChanged)
	require.Len(t, result.RemoteOnlyNotes, 1)
	require.Equal(t, resp.ID, result.RemoteOnlyNotes[0].ID)
	_, err = os.Stat(filepath.Join(root, "work", "plan.md"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestSyncRootReturnsUntrackedRemoteNotesAsPlaceholders(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	client := localClient{}
	cfg := testSyncConfig(remote)
	_, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "ideas.md",
		Content: `---
sync: synced
---
# Ideas
`,
		Encrypted: false,
	})
	require.NoError(t, err)

	result, err := SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)
	require.Len(t, result.RemoteOnlyNotes, 1)
	require.Equal(t, "ideas.md", result.RemoteOnlyNotes[0].RelPath)
	require.Equal(t, "Ideas", result.RemoteOnlyNotes[0].Title)
}

func TestDeleteRemoteNoteAndKeepLocalRemovesRemoteAndUnlinksLocal(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	notePath := filepath.Join(root, "work", "plan.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, []byte(`---
sync: synced
---
# Plan

Body
`), 0o644))

	client := localClient{}
	cfg := testSyncConfig(remote)
	result, err := SyncRoot(context.Background(), root, cfg, []string{"work/plan.md"}, []string{"work"}, client)
	require.NoError(t, err)
	require.Equal(t, 1, result.RegisteredNotes)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Len(t, records, 1)
	var rec NoteRecord
	for _, candidate := range records {
		rec = candidate
	}

	require.NoError(t, DeleteRemoteNoteAndKeepLocal(context.Background(), root, notePath, cfg, client))

	_, err = client.FetchNote(context.Background(), cfg.Profiles["local"], FetchNoteRequest{RemoteRoot: remote, NoteID: rec.ID})
	require.Error(t, err)
	var rpcErr *RPCError
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, ErrCodeNotFound, rpcErr.Code)

	records, err = LoadNoteRecords(root)
	require.NoError(t, err)
	require.Empty(t, records)

	pins, err := LoadPins(root)
	require.NoError(t, err)
	require.Empty(t, pins.PinnedNoteIDs)

	raw, err := os.ReadFile(notePath)
	require.NoError(t, err)
	fm, _, err := notes.ParseFrontMatter(string(raw))
	require.NoError(t, err)
	require.Equal(t, notes.SyncClassLocal, notes.ParseSyncClass(fm))
}

func TestServerDeleteNoteRemovesPinnedID(t *testing.T) {
	remote := t.TempDir()
	client := localClient{}
	cfg := testSyncConfig(remote)
	resp, err := client.RegisterNote(context.Background(), cfg.Profiles["local"], RegisterNoteRequest{
		RemoteRoot: remote,
		RelPath:    "work/plan.md",
		Content: `---
sync: synced
---
# Plan
`,
		Encrypted: false,
	})
	require.NoError(t, err)
	_, err = client.PinsPut(context.Background(), cfg.Profiles["local"], PinsPutRequest{RemoteRoot: remote, Pins: Pins{PinnedNoteIDs: []string{resp.ID}, PinnedCategories: []string{"work"}}})
	require.NoError(t, err)

	_, err = client.DeleteNote(context.Background(), cfg.Profiles["local"], DeleteNoteRequest{RemoteRoot: remote, NoteID: resp.ID, ExpectedRevision: resp.Revision})
	require.NoError(t, err)

	index, err := client.PullIndex(context.Background(), cfg.Profiles["local"], PullIndexRequest{RemoteRoot: remote})
	require.NoError(t, err)
	require.Empty(t, index.Notes)
	require.Empty(t, index.Pins.PinnedNoteIDs)
	require.Equal(t, []string{"work"}, index.Pins.PinnedCategories)
}

func TestSyncRootMarksMissingRemoteNoteAsError(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	notePath := filepath.Join(root, "work", "plan.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, []byte(`---
sync: synced
---
# Plan

Body
`), 0o644))

	client := localClient{}
	cfg := testSyncConfig(remote)
	result, err := SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)
	require.Equal(t, 1, result.RegisteredNotes)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Len(t, records, 1)
	var rec NoteRecord
	for _, candidate := range records {
		rec = candidate
	}
	require.NoError(t, os.Remove(filepath.Join(remote, "notes", rec.ID+".json")))
	require.NoError(t, os.Remove(filepath.Join(remote, "content", rec.ID+".note")))

	result, err = SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)
	require.Zero(t, result.RegisteredNotes)
	require.Zero(t, result.UpdatedNotes)

	records, err = LoadNoteRecords(root)
	require.NoError(t, err)
	require.Len(t, records, 1)
	rec = records[rec.ID]
	require.Contains(t, rec.LastSyncError, "note missing on remote")
}

func TestResolveConflictKeepRemoteReplacesLocalAndCleansUpConflict(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	notePath := filepath.Join(root, "work", "plan.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, []byte("---\nsync: synced\n---\nlocal body\n"), 0o644))

	client := localClient{}
	cfg := testSyncConfig(remote)
	_, err := SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	var rec NoteRecord
	for _, candidate := range records {
		rec = candidate
	}

	_, err = client.PushNote(context.Background(), cfg.Profiles["local"], PushNoteRequest{
		RemoteRoot:       remote,
		NoteID:           rec.ID,
		ExpectedRevision: rec.RemoteRev,
		RelPath:          rec.RelPath,
		Content:          "---\nsync: synced\n---\nremote body\n",
		Encrypted:        false,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(notePath, []byte("---\nsync: synced\n---\nlocal edited\n"), 0o644))
	_, err = SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)

	records, err = LoadNoteRecords(root)
	require.NoError(t, err)
	rec = records[rec.ID]
	require.NotNil(t, rec.Conflict)
	conflictCopyPath := filepath.Join(root, filepath.FromSlash(rec.Conflict.CopyPath))
	require.FileExists(t, conflictCopyPath)

	require.NoError(t, ResolveConflictKeepRemote(root, rec))

	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	require.Equal(t, "---\nsync: synced\n---\nremote body\n", string(data))
	require.NoFileExists(t, conflictCopyPath)
	require.NoFileExists(t, ConflictPath(root, rec.ID))

	records, err = LoadNoteRecords(root)
	require.NoError(t, err)
	rec = records[rec.ID]
	require.Nil(t, rec.Conflict)
	require.Empty(t, rec.LastSyncError)
}

func TestResolveConflictKeepLocalPushesAndCleansUpConflict(t *testing.T) {
	root := t.TempDir()
	remote := t.TempDir()
	notePath := filepath.Join(root, "work", "plan.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(notePath), 0o755))
	require.NoError(t, os.WriteFile(notePath, []byte("---\nsync: synced\n---\nlocal body\n"), 0o644))

	client := localClient{}
	cfg := testSyncConfig(remote)
	_, err := SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	var rec NoteRecord
	for _, candidate := range records {
		rec = candidate
	}

	_, err = client.PushNote(context.Background(), cfg.Profiles["local"], PushNoteRequest{
		RemoteRoot:       remote,
		NoteID:           rec.ID,
		ExpectedRevision: rec.RemoteRev,
		RelPath:          rec.RelPath,
		Content:          "---\nsync: synced\n---\nremote body\n",
		Encrypted:        false,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(notePath, []byte("---\nsync: synced\n---\nlocal edited\n"), 0o644))
	_, err = SyncRoot(context.Background(), root, cfg, nil, nil, client)
	require.NoError(t, err)

	records, err = LoadNoteRecords(root)
	require.NoError(t, err)
	rec = records[rec.ID]
	require.NotNil(t, rec.Conflict)
	conflictCopyPath := filepath.Join(root, filepath.FromSlash(rec.Conflict.CopyPath))
	require.FileExists(t, conflictCopyPath)

	require.NoError(t, ResolveConflictKeepLocal(context.Background(), root, notePath, cfg, rec, client))

	require.NoFileExists(t, conflictCopyPath)
	require.NoFileExists(t, ConflictPath(root, rec.ID))
	records, err = LoadNoteRecords(root)
	require.NoError(t, err)
	rec = records[rec.ID]
	require.Nil(t, rec.Conflict)
	require.Empty(t, rec.LastSyncError)

	fetched, err := client.FetchNote(context.Background(), cfg.Profiles["local"], FetchNoteRequest{RemoteRoot: remote, NoteID: rec.ID})
	require.NoError(t, err)
	require.Equal(t, "---\nsync: synced\n---\nlocal edited\n", fetched.Note.Content)
}
