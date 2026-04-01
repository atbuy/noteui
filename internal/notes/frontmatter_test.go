package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		t.Fatalf("ParseFrontMatter returned error: %v", err)
	}
	if body != "# Heading\nBody" {
		t.Fatalf("unexpected body: %q", body)
	}
	if fm["title"] != "Hello" {
		t.Fatalf("expected title to be parsed, got %#v", fm)
	}
	if fm["encrypted-flag"] != "yes" {
		t.Fatalf("expected normalized key, got %#v", fm)
	}

	tags := ParseTags(fm)
	if strings.Join(tags, ",") != "work,personal,urgent" {
		t.Fatalf("unexpected tags: %v", tags)
	}
	if !FrontMatterBool(fm, "encrypted_flag") {
		t.Fatal("expected boolean frontmatter lookup to normalize key names")
	}
}

func TestParseFrontMatterLeavesBodyUntouchedWhenBlockIncomplete(t *testing.T) {
	raw := "---\ntitle: Missing close\n# Heading"

	fm, body, err := ParseFrontMatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontMatter returned error: %v", err)
	}
	if fm != nil {
		t.Fatalf("expected nil frontmatter, got %#v", fm)
	}
	if body != raw {
		t.Fatalf("expected body to be unchanged, got %q", body)
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
		t.Fatal("expected encrypted flag to be detected")
	}
	if !NoteIsPrivate(raw) {
		t.Fatal("expected private flag to be detected")
	}
	if StripFrontMatter(raw) != "secret body" {
		t.Fatalf("expected frontmatter to be stripped, got %q", StripFrontMatter(raw))
	}
}

func TestMergeTagsNormalizesAndDeduplicates(t *testing.T) {
	got := mergeTags([]string{"work", "Urgent"}, []string{"#urgent", " personal ", "", "Work"})
	if strings.Join(got, ",") != "work,Urgent,personal" {
		t.Fatalf("unexpected merged tags: %v", got)
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
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if err := AddTagsToNote(path, []string{"beta", "#alpha", "gamma"}); err != nil {
		t.Fatalf("AddTagsToNote returned error: %v", err)
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	text := string(updated)
	if !strings.Contains(text, "tags: alpha, beta, gamma") {
		t.Fatalf("expected updated tag list, got %q", text)
	}
}
