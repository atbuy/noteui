package meta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFrontMatterAndHelpers(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		`Title: "Launch Plan"`,
		"private: yes",
		"encrypted: on",
		"custom_key:  spaced value  ",
		"sync: shared",
		"tags: work, personal",
		"# comment",
		"---",
		"Body line 1",
		"Body line 2",
	}, "\r\n")

	fm, body, err := ParseFrontMatter(raw)
	require.NoError(t, err)
	require.Equal(t, "Launch Plan", FrontMatterString(fm, "title"))
	require.Equal(t, "spaced value", FrontMatterString(fm, "custom-key"))
	require.True(t, FrontMatterBool(fm, "private"))
	require.True(t, FrontMatterBool(fm, "encrypted"))
	require.Equal(t, SyncClassShared, ParseSyncClass(fm))
	require.Equal(t, []string{"work", "personal"}, ParseTags(fm))
	require.Equal(t, "Body line 1\nBody line 2", body)
	require.True(t, NoteIsEncrypted(raw))
	require.True(t, NoteIsPrivate(raw))
	require.Equal(t, body, StripFrontMatter(raw))
}

func TestAddTagsToNoteMergesCaseInsensitiveTags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.md")
	initial := strings.Join([]string{
		"---",
		"tags: Work, #Personal",
		"sync: local",
		"---",
		"body",
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(initial), 0o644))

	require.NoError(t, AddTagsToNote(path, []string{"work", "urgent", "#Personal", "new"}))

	updated, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, strings.Join([]string{
		"---",
		"sync: local",
		"tags: Work, Personal, urgent, new",
		"---",
		"body",
	}, "\n"), string(updated))
}

func TestSetNoteSyncClassAndToggleNoteSyncClass(t *testing.T) {
	t.Run("set unknown sync class falls back to local", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "note.md")
		require.NoError(t, os.WriteFile(path, []byte("body"), 0o644))

		require.NoError(t, SetNoteSyncClass(path, "invalid"))

		updated, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "---\nsync: local\n---\nbody", string(updated))
	})

	t.Run("toggle local to synced", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "note.md")
		require.NoError(t, os.WriteFile(path, []byte("body"), 0o644))

		next, err := ToggleNoteSyncClass(path)
		require.NoError(t, err)
		require.Equal(t, SyncClassSynced, next)

		updated, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "---\nsync: synced\n---\nbody", string(updated))
	})

	t.Run("toggle synced to local", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "note.md")
		require.NoError(t, os.WriteFile(path, []byte("---\nsync: synced\n---\nbody"), 0o644))

		next, err := ToggleNoteSyncClass(path)
		require.NoError(t, err)
		require.Equal(t, SyncClassLocal, next)

		updated, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "---\nsync: local\n---\nbody", string(updated))
	})

	t.Run("shared notes cannot be toggled", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "note.md")
		require.NoError(t, os.WriteFile(path, []byte("---\nsync: shared\n---\nbody"), 0o644))

		next, err := ToggleNoteSyncClass(path)
		require.Error(t, err)
		require.Equal(t, SyncClassShared, next)
		require.Contains(t, err.Error(), "shared notes cannot be toggled")

		updated, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "---\nsync: shared\n---\nbody", string(updated))
	})
}

func TestWikilinkAndFrontMatterHelpers(t *testing.T) {
	content := "Review [[Plan/2026]] with [[Spaced Title]] and [[Plan/2026]]."

	rewritten := RewriteWikilinks(content)
	require.Contains(t, rewritten, "[Plan/2026](#wikilink:Plan%2F2026)")
	require.Contains(t, rewritten, "[Spaced Title](#wikilink:Spaced%20Title)")
	require.Equal(t, []string{"Plan/2026", "Spaced Title"}, ExtractWikilinks(content))
	require.Equal(t, "Spaced Title", DecodeWikilinkTarget("Spaced%20Title"))
	require.Equal(t, "%zz", DecodeWikilinkTarget("%zz"))

	require.Equal(t, "custom-key", normalizeFrontMatterKey(" Custom_Key "))
	require.Equal(t, []string{"Work", "Personal", "new"}, mergeTags([]string{"Work", "#Personal"}, []string{"work", "new", "#personal"}))
	require.Equal(t, "tag", normalizeTag(" #tag "))

	replaced := setFrontMatterField("---\nold_key: value\nkeep: yes\n---\nbody", "body", "old-key", "old_key: next")
	require.Equal(t, "---\nkeep: yes\nold_key: next\n---\nbody", replaced)

	inserted := setFrontMatterField("body", "body", "sync", "sync: synced")
	require.Equal(t, "---\nsync: synced\n---\nbody", inserted)
}
