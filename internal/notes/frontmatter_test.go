package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFrontMatterParsesAndNormalizesFields(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		`title: "Hello"`,
		"encrypted_flag: yes",
		"tags: work, personal,  , urgent",
		"# comment",
		"---",
		"# Heading",
		"Body",
	}, "\n")

	fm, body, err := ParseFrontMatter(raw)
	if err != nil {
		require.Failf(t, "assertion failed", "ParseFrontMatter returned error: %v", err)
	}
	if body != "# Heading\nBody" {
		require.Failf(t, "assertion failed", "unexpected body: %q", body)
	}
	if fm["title"] != "Hello" {
		require.Failf(t, "assertion failed", "expected title to be parsed, got %#v", fm)
	}
	if fm["encrypted-flag"] != "yes" {
		require.Failf(t, "assertion failed", "expected normalized key, got %#v", fm)
	}

	tags := ParseTags(fm)
	if strings.Join(tags, ",") != "work,personal,urgent" {
		require.Failf(t, "assertion failed", "unexpected tags: %v", tags)
	}
	if !FrontMatterBool(fm, "encrypted_flag") {
		require.FailNow(t, "expected boolean frontmatter lookup to normalize key names")
	}
}

func TestParseFrontMatterLeavesBodyUntouchedWhenBlockIncomplete(t *testing.T) {
	raw := "---\ntitle: Missing close\n# Heading"

	fm, body, err := ParseFrontMatter(raw)
	if err != nil {
		require.Failf(t, "assertion failed", "ParseFrontMatter returned error: %v", err)
	}
	if fm != nil {
		require.Failf(t, "assertion failed", "expected nil frontmatter, got %#v", fm)
	}
	if body != raw {
		require.Failf(t, "assertion failed", "expected body to be unchanged, got %q", body)
	}
}

func TestNotePrivacyAndEncryptionFlags(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		"encrypted: on",
		"private: true",
		"---",
		"secret body",
	}, "\n")

	if !NoteIsEncrypted(raw) {
		require.FailNow(t, "expected encrypted flag to be detected")
	}
	if !NoteIsPrivate(raw) {
		require.FailNow(t, "expected private flag to be detected")
	}
	if StripFrontMatter(raw) != "secret body" {
		require.Failf(t, "assertion failed", "expected frontmatter to be stripped, got %q", StripFrontMatter(raw))
	}
}

func TestParseSyncClassDefaultsToLocal(t *testing.T) {
	fm := FrontMatter{}
	require.Equal(t, SyncClassLocal, ParseSyncClass(fm))
	fm["sync"] = "synced"
	require.Equal(t, SyncClassSynced, ParseSyncClass(fm))
	fm["sync"] = "shared"
	require.Equal(t, SyncClassShared, ParseSyncClass(fm))
	fm["sync"] = "bogus"
	require.Equal(t, SyncClassLocal, ParseSyncClass(fm))
}

func TestToggleNoteSyncClassBlocksSharedNotes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shared.md")
	require.NoError(t, os.WriteFile(path, []byte("---\nsync: shared\n---\n# Shared\n"), 0o644))

	result, err := ToggleNoteSyncClass(path)
	require.Error(t, err)
	require.Equal(t, SyncClassShared, result)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(raw), "sync: shared")
}

func TestSetNoteSyncClassAcceptsShared(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	require.NoError(t, os.WriteFile(path, []byte("# Body\n"), 0o644))
	require.NoError(t, SetNoteSyncClass(path, SyncClassShared))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(raw), "sync: shared")
}

func TestToggleNoteSyncClassRewritesFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	require.NoError(t, os.WriteFile(path, []byte("# Heading\n\nBody\n"), 0o644))

	next, err := ToggleNoteSyncClass(path)
	require.NoError(t, err)
	require.Equal(t, SyncClassSynced, next)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(raw), "sync: synced")

	next, err = ToggleNoteSyncClass(path)
	require.NoError(t, err)
	require.Equal(t, SyncClassLocal, next)

	raw, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(raw), "sync: local")
}

func TestMergeTagsNormalizesAndDeduplicates(t *testing.T) {
	got := mergeTags([]string{"work", "Urgent"}, []string{"#urgent", " personal ", "", "Work"})
	if strings.Join(got, ",") != "work,Urgent,personal" {
		require.Failf(t, "assertion failed", "unexpected merged tags: %v", got)
	}
}

func TestAddTagsToNoteCreatesAndReplacesFrontMatterField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	raw := strings.Join([]string{
		"---",
		"title: Example",
		"tags: alpha",
		"---",
		"body",
	}, "\n")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}

	if err := AddTagsToNote(path, []string{"beta", "#alpha", "gamma"}); err != nil {
		require.Failf(t, "assertion failed", "AddTagsToNote returned error: %v", err)
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadFile returned error: %v", err)
	}
	text := string(updated)
	if !strings.Contains(text, "tags: alpha, beta, gamma") {
		require.Failf(t, "assertion failed", "expected updated tag list, got %q", text)
	}
}
