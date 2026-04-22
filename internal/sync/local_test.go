package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
)

func TestHashContent(t *testing.T) {
	h1 := HashContent("hello")
	h2 := HashContent("hello")
	h3 := HashContent("world")

	if !isHashString(h1) {
		require.Failf(t, "assertion failed", "expected sha256: prefix, got %q", h1)
	}
	if h1 != h2 {
		require.FailNow(t, "expected same input to produce same hash")
	}
	if h1 == h3 {
		require.FailNow(t, "expected different inputs to produce different hashes")
	}
	if HashContent("") == HashContent("x") {
		require.FailNow(t, "expected empty and non-empty content to differ")
	}
}

func isHashString(s string) bool {
	return len(s) > 7 && s[:7] == "sha256:"
}

func TestSortedUnique(t *testing.T) {
	if got := sortedUnique(nil); got != nil {
		require.Failf(t, "assertion failed", "expected nil for nil input, got %v", got)
	}
	if got := sortedUnique([]string{}); got != nil {
		require.Failf(t, "assertion failed", "expected nil for empty input, got %v", got)
	}

	got := sortedUnique([]string{"b", "a", "b", " a ", "c"})
	require.Equal(t, []string{"a", "b", "c"}, got)

	// empty strings should be dropped
	got = sortedUnique([]string{"z", "", "  ", "a"})
	require.Equal(t, []string{"a", "z"}, got)
}

func TestRemoveSortedUniqueString(t *testing.T) {
	items := []string{"a", "b", "c"}

	got := removeSortedUniqueString(items, "b")
	require.Equal(t, []string{"a", "c"}, got)

	got = removeSortedUniqueString(items, "z")
	require.Equal(t, []string{"a", "b", "c"}, got)

	got = removeSortedUniqueString(nil, "a")
	if got != nil {
		require.Failf(t, "assertion failed", "expected nil for nil input, got %v", got)
	}

	got = removeSortedUniqueString(items, "")
	require.Equal(t, []string{"a", "b", "c"}, got)
}

func TestHasSyncProfile(t *testing.T) {
	if HasSyncProfile(config.SyncConfig{}) {
		require.FailNow(t, "expected false for empty config")
	}
	if HasSyncProfile(config.SyncConfig{DefaultProfile: "  "}) {
		require.FailNow(t, "expected false for whitespace-only profile name")
	}
	if !HasSyncProfile(config.SyncConfig{DefaultProfile: "main"}) {
		require.FailNow(t, "expected true for non-empty profile name")
	}
}

func TestActiveProfile(t *testing.T) {
	root := t.TempDir()
	cfg := config.SyncConfig{
		DefaultProfile: "home",
		Profiles: map[string]config.SyncProfile{
			"home":   {SSHHost: "host1", RemoteRoot: "/srv"},
			"office": {SSHHost: "host2", RemoteRoot: "/data"},
		},
	}

	profile, name, err := ActiveProfile(cfg, root)
	require.NoError(t, err)
	require.Equal(t, "home", name)
	require.Equal(t, "host1", profile.SSHHost)
	require.Equal(t, DefaultRemoteBin, profile.RemoteBin, "expected default bin to be set when empty")

	// root config overrides the default profile
	require.NoError(t, SaveRootConfig(root, RootConfig{
		SchemaVersion: SchemaVersion,
		ClientID:      "test-client",
		Profile:       "office",
	}))
	profile, name, err = ActiveProfile(cfg, root)
	require.NoError(t, err)
	require.Equal(t, "office", name)
	require.Equal(t, "host2", profile.SSHHost)

	// no profile configured
	_, _, err = ActiveProfile(config.SyncConfig{}, root)
	require.Error(t, err)

	// profile referenced but not present
	_, _, err = ActiveProfile(config.SyncConfig{DefaultProfile: "missing"}, t.TempDir())
	require.Error(t, err)
}

func TestResolvedRemoteRootIgnoresWebDAVOverrideThatMatchesLocalRoot(t *testing.T) {
	profile := config.SyncProfile{
		Kind:       config.SyncKindWebDAV,
		WebDAVURL:  "https://cloud.example.com/remote.php/dav/files/alice",
		RemoteRoot: "/Notes",
	}

	got := resolvedRemoteRoot(profile, "/home/alice/notes", "/home/alice/notes")
	require.Equal(t, "/Notes", got)
}

func TestResolvedRemoteRootKeepsDistinctWebDAVOverride(t *testing.T) {
	profile := config.SyncProfile{
		Kind:       config.SyncKindWebDAV,
		WebDAVURL:  "https://cloud.example.com/remote.php/dav/files/alice",
		RemoteRoot: "/Notes",
	}

	got := resolvedRemoteRoot(profile, "/home/alice/notes", "/Projects/Shared")
	require.Equal(t, "/Projects/Shared", got)
}

func TestSaveAndLoadRootConfig(t *testing.T) {
	root := t.TempDir()

	_, err := LoadRootConfig(root)
	require.ErrorIs(t, err, os.ErrNotExist)

	cfg := RootConfig{SchemaVersion: SchemaVersion, ClientID: "abc123", Profile: "home"}
	require.NoError(t, SaveRootConfig(root, cfg))

	loaded, err := LoadRootConfig(root)
	require.NoError(t, err)
	require.Equal(t, cfg, loaded)
}

func TestLoadRootConfigReportsCorruptPath(t *testing.T) {
	root := t.TempDir()
	path := ConfigPath(root)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`{"schema_version":1}"`), 0o644))

	_, err := LoadRootConfig(root)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid local sync root config JSON at "+path)
	require.Contains(t, err.Error(), `invalid character '"' after top-level value`)
}

func TestEnsureRootConfig(t *testing.T) {
	root := t.TempDir()
	syncCfg := config.SyncConfig{
		DefaultProfile: "work",
		Profiles: map[string]config.SyncProfile{
			"work": {SSHHost: "whost", RemoteRoot: "/notes"},
		},
	}

	// first call creates a new config
	cfg, err := EnsureRootConfig(root, syncCfg)
	require.NoError(t, err)
	require.Equal(t, "work", cfg.Profile)
	require.NotEmpty(t, cfg.ClientID)

	// second call returns the same config without modifying it
	cfg2, err := EnsureRootConfig(root, syncCfg)
	require.NoError(t, err)
	require.Equal(t, cfg.ClientID, cfg2.ClientID)
}

func TestSaveAndLoadNoteRecords(t *testing.T) {
	root := t.TempDir()

	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Empty(t, records)

	rec := NoteRecord{
		ID:             "note-abc",
		RelPath:        "work/note.md",
		Class:          ClassSynced,
		RemoteRev:      "rev-1",
		LastSyncedHash: HashContent("body"),
		LastSyncAt:     time.Now().UTC().Truncate(time.Second),
	}
	require.NoError(t, SaveNoteRecord(root, rec))

	records, err = LoadNoteRecords(root)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, rec.RelPath, records["note-abc"].RelPath)
	require.Equal(t, rec.RemoteRev, records["note-abc"].RemoteRev)
}

func TestLoadNoteRecordsReportsCorruptPath(t *testing.T) {
	root := t.TempDir()
	path := NoteRecordPath(root, "n_bad")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`{"id":"n_bad"}"`), 0o644))

	_, err := LoadNoteRecords(root)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid local sync note record JSON at "+path)
	require.Contains(t, err.Error(), `invalid character '"' after top-level value`)
}

func TestSaveNoteRecordRejectsEmptyID(t *testing.T) {
	root := t.TempDir()
	err := SaveNoteRecord(root, NoteRecord{ID: "", RelPath: "note.md"})
	require.Error(t, err)
}

func TestDeleteNoteRecord(t *testing.T) {
	root := t.TempDir()
	rec := NoteRecord{ID: "del-note", RelPath: "a.md", Class: ClassSynced}
	require.NoError(t, SaveNoteRecord(root, rec))

	require.NoError(t, DeleteNoteRecord(root, "del-note"))
	records, err := LoadNoteRecords(root)
	require.NoError(t, err)
	require.Empty(t, records)

	// deleting a non-existent record is a no-op
	require.NoError(t, DeleteNoteRecord(root, "does-not-exist"))

	// empty id is a no-op
	require.NoError(t, DeleteNoteRecord(root, ""))
}

func TestSaveAndLoadPins(t *testing.T) {
	root := t.TempDir()

	_, err := LoadPins(root)
	require.ErrorIs(t, err, os.ErrNotExist)

	pins := Pins{
		PinnedNoteIDs:    []string{"id-b", "id-a", "id-a"},
		PinnedCategories: []string{"work", "personal"},
	}
	require.NoError(t, SavePins(root, pins))

	loaded, err := LoadPins(root)
	require.NoError(t, err)
	// duplicates should be removed and entries sorted
	require.Equal(t, []string{"id-a", "id-b"}, loaded.PinnedNoteIDs)
	require.Equal(t, []string{"personal", "work"}, loaded.PinnedCategories)
}

func TestLoadPinsReportsCorruptPath(t *testing.T) {
	root := t.TempDir()
	path := PinsPath(root)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`{"pinned_note_ids":[]}"`), 0o644))

	_, err := LoadPins(root)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid local sync pins JSON at "+path)
	require.Contains(t, err.Error(), `invalid character '"' after top-level value`)
}

func TestRemovePinnedNoteID(t *testing.T) {
	root := t.TempDir()

	// removing from a non-existent pins file is a no-op
	require.NoError(t, RemovePinnedNoteID(root, "id-1"))

	require.NoError(t, SavePins(root, Pins{PinnedNoteIDs: []string{"id-1", "id-2"}}))

	require.NoError(t, RemovePinnedNoteID(root, "id-1"))
	loaded, err := LoadPins(root)
	require.NoError(t, err)
	require.Equal(t, []string{"id-2"}, loaded.PinnedNoteIDs)
}

func TestSaveAndDeleteConflictRecord(t *testing.T) {
	root := t.TempDir()

	rec := ConflictRecord{
		NoteID:       "conflict-note",
		LocalPath:    "notes/a.md",
		RemoteRev:    "rev-2",
		ConflictPath: "notes/a.conflict-20260101-120000.md",
		OccurredAt:   time.Now().UTC(),
	}
	require.NoError(t, SaveConflictRecord(root, rec))
	require.FileExists(t, ConflictPath(root, "conflict-note"))

	require.NoError(t, DeleteConflictRecord(root, "conflict-note"))
	if _, err := os.Stat(ConflictPath(root, "conflict-note")); !os.IsNotExist(err) {
		require.Failf(t, "assertion failed", "expected conflict record to be deleted, stat err=%v", err)
	}

	// deleting non-existent is a no-op
	require.NoError(t, DeleteConflictRecord(root, "does-not-exist"))
	// empty id is a no-op
	require.NoError(t, DeleteConflictRecord(root, ""))
}

func TestSaveConflictRecordRejectsEmptyID(t *testing.T) {
	root := t.TempDir()
	err := SaveConflictRecord(root, ConflictRecord{NoteID: ""})
	require.Error(t, err)
}

func TestLoadPinnedRelPaths(t *testing.T) {
	root := t.TempDir()

	// no pins file: returns empty slices
	notePaths, catPaths, err := LoadPinnedRelPaths(root)
	require.NoError(t, err)
	require.Nil(t, notePaths)
	require.Nil(t, catPaths)

	// save a note record and a pins entry
	rec := NoteRecord{ID: "n1", RelPath: "work/meeting.md", Class: ClassSynced}
	require.NoError(t, SaveNoteRecord(root, rec))
	require.NoError(t, SavePins(root, Pins{
		PinnedNoteIDs:    []string{"n1"},
		PinnedCategories: []string{"work"},
	}))

	notePaths, catPaths, err = LoadPinnedRelPaths(root)
	require.NoError(t, err)
	require.Equal(t, []string{"work/meeting.md"}, notePaths)
	require.Equal(t, []string{"work"}, catPaths)
}

func TestSavePinsFromRelPaths(t *testing.T) {
	root := t.TempDir()

	syncedNote := notes.Note{RelPath: "work/note.md"}
	syncedNote.SyncClass = notes.SyncClassSynced

	sharedNote := notes.Note{RelPath: "shared/note.md"}
	sharedNote.SyncClass = notes.SyncClassShared

	localNote := notes.Note{RelPath: "local/note.md"}

	rec := NoteRecord{ID: "wn1", RelPath: "work/note.md", Class: ClassSynced}
	require.NoError(t, SaveNoteRecord(root, rec))
	require.NoError(t, SaveNoteRecord(root, NoteRecord{ID: "sn1", RelPath: "shared/note.md", Class: ClassSynced}))

	require.NoError(t, SavePinsFromRelPaths(root, []notes.Note{syncedNote, sharedNote, localNote},
		[]string{"work/note.md", "shared/note.md", "local/note.md"}, []string{"work"}))

	pins, err := LoadPins(root)
	require.NoError(t, err)
	require.Equal(t, []string{"sn1", "wn1"}, pins.PinnedNoteIDs)
	require.Equal(t, []string{"work"}, pins.PinnedCategories)
}

func TestMigratePinsFromState(t *testing.T) {
	root := t.TempDir()

	syncedNote := notes.Note{RelPath: "daily.md"}
	syncedNote.SyncClass = notes.SyncClassSynced

	rec := NoteRecord{ID: "d1", RelPath: "daily.md", Class: ClassSynced}
	require.NoError(t, SaveNoteRecord(root, rec))

	// first call: pins file doesn't exist yet, so migration runs
	require.NoError(t, MigratePinsFromState(root, []notes.Note{syncedNote}, []string{"daily.md"}, nil))
	pins, err := LoadPins(root)
	require.NoError(t, err)
	require.Equal(t, []string{"d1"}, pins.PinnedNoteIDs)

	// second call: pins file already exists, migration is skipped
	require.NoError(t, MigratePinsFromState(root, []notes.Note{syncedNote}, nil, nil))
	pins, err = LoadPins(root)
	require.NoError(t, err)
	require.Equal(t, []string{"d1"}, pins.PinnedNoteIDs, "expected pins unchanged on second migration call")
}

func TestPathHelpers(t *testing.T) {
	root := "/notes/root"
	require.Equal(t, filepath.Join(root, SyncDirName), SyncDir(root))
	require.Equal(t, filepath.Join(SyncDir(root), "config.json"), ConfigPath(root))
	require.Equal(t, filepath.Join(SyncDir(root), "pins.json"), PinsPath(root))
	require.Equal(t, filepath.Join(SyncDir(root), NotesDirName, "abc.json"), NoteRecordPath(root, "abc"))
	require.Equal(t, filepath.Join(SyncDir(root), ConflictsDirName, "abc.json"), ConflictPath(root, "abc"))
}
