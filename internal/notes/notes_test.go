package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractTitleAndFallbackHelpers(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		"title: ignored",
		"---",
		"",
		"# Heading Title",
		"Body",
	}, "\n")

	if got := ExtractTitle(raw); got != "Heading Title" {
		t.Fatalf("expected heading title, got %q", got)
	}
	if got := ExtractTitleOrFirstLine("---\nfoo: bar\n---\n* first line"); got != "first line" {
		t.Fatalf("expected first line fallback, got %q", got)
	}
	if got := fallbackTitleFromFilename("project-notes.md"); got != "project notes" {
		t.Fatalf("unexpected fallback title: %q", got)
	}
	if got := fallbackTitleFromFilename(".md"); got != "Untitled" {
		t.Fatalf("expected Untitled fallback, got %q", got)
	}
}

func TestSlugifyAndWordCount(t *testing.T) {
	if got := Slugify("  Hello, World + Draft/1  "); got != "hello-world-draft-1" {
		t.Fatalf("unexpected slug: %q", got)
	}
	if got := Slugify("!!!"); got != "" {
		t.Fatalf("expected empty slug for punctuation-only input, got %q", got)
	}

	raw := strings.Join([]string{
		"---",
		"tags: alpha, beta",
		"---",
		"one two",
		"three",
	}, "\n")
	if got := WordCount(raw); got != 3 {
		t.Fatalf("expected word count 3, got %d", got)
	}
	if got := ReadingTimeMinutes(0); got != 0 {
		t.Fatalf("expected zero reading time, got %d", got)
	}
	if got := ReadingTimeMinutes(1); got != 1 {
		t.Fatalf("expected minimum reading time of 1, got %d", got)
	}
	if got := ReadingTimeMinutes(226); got != 2 {
		t.Fatalf("expected rounded reading time of 2, got %d", got)
	}
}

func TestCleanRelativePathAndReplaceOrInsertRootTitle(t *testing.T) {
	if got := cleanRelativePath(" ../docs/note ", true); got != "docs/note.md" {
		t.Fatalf("unexpected cleaned path: %q", got)
	}
	if got := cleanRelativePath(".", true); got != "" {
		t.Fatalf("expected empty cleaned path, got %q", got)
	}

	replaced := replaceOrInsertRootTitle("# Old\n\nBody", "New")
	if replaced != "# New\n\nBody" {
		t.Fatalf("unexpected replaced title content: %q", replaced)
	}
	inserted := replaceOrInsertRootTitle("Body", "Inserted")
	if inserted != "# Inserted\n\nBody" {
		t.Fatalf("unexpected inserted title content: %q", inserted)
	}
}

func TestRenameFromTitleDeletesEmptyTemporaryNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".new-20260401-120000.md")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	newPath, renamed, err := RenameFromTitle(path)
	if err != nil {
		t.Fatalf("RenameFromTitle returned error: %v", err)
	}
	if newPath != "" || renamed {
		t.Fatalf("expected deleted temporary note, got path=%q renamed=%v", newPath, renamed)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected temp note to be deleted, stat err=%v", err)
	}
}

func TestRenameFromTitleUsesUniquePath(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "meeting-notes.md")
	if err := os.WriteFile(existing, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	path := filepath.Join(dir, ".new-20260401-120000.md")
	if err := os.WriteFile(path, []byte("# Meeting Notes\nBody"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	newPath, renamed, err := RenameFromTitle(path)
	if err != nil {
		t.Fatalf("RenameFromTitle returned error: %v", err)
	}
	if !renamed {
		t.Fatal("expected note to be renamed")
	}
	if filepath.Base(newPath) != "meeting-notes-2.md" {
		t.Fatalf("expected unique target path, got %q", newPath)
	}
}

func TestRenameNoteTitleUpdatesContentAndFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "draft.md")
	if err := os.WriteFile(path, []byte("# Old Title\n\nBody"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	newPath, renamed, err := RenameNoteTitle(path, "Fresh Title")
	if err != nil {
		t.Fatalf("RenameNoteTitle returned error: %v", err)
	}
	if !renamed {
		t.Fatal("expected note to be renamed")
	}
	if filepath.Base(newPath) != "fresh-title.md" {
		t.Fatalf("unexpected new path: %q", newPath)
	}

	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.HasPrefix(string(data), "# Fresh Title") {
		t.Fatalf("expected updated title in file, got %q", string(data))
	}
}

func TestMoveNoteValidatesAndMovesFile(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "draft.md")
	if err := os.WriteFile(oldPath, []byte("body"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if err := MoveNote(root, "", "next.md"); err == nil {
		t.Fatal("expected error for empty source path")
	}

	if err := MoveNote(root, "../draft.md", "nested/renamed"); err != nil {
		t.Fatalf("MoveNote returned error: %v", err)
	}
	newPath := filepath.Join(root, "nested", "renamed.md")
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected moved note at %q: %v", newPath, err)
	}

	if err := MoveNote(root, "nested/renamed.md", "nested/renamed.md"); err != nil {
		t.Fatalf("expected no-op move to succeed, got %v", err)
	}
}

func TestCreateNoteAndReadHelpers(t *testing.T) {
	root := t.TempDir()

	path, err := CreateNote(root, "daily")
	if err != nil {
		t.Fatalf("CreateNote returned error: %v", err)
	}
	if filepath.Ext(path) != ".md" {
		t.Fatalf("expected markdown note, got %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected note to exist: %v", err)
	}

	content := "Line 1\n\tIndented\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	preview, err := ReadPreview(path)
	if err != nil {
		t.Fatalf("ReadPreview returned error: %v", err)
	}
	if !strings.Contains(preview, "    Indented") {
		t.Fatalf("expected tabs to expand in preview, got %q", preview)
	}

	full, err := ReadFull(path)
	if err != nil {
		t.Fatalf("ReadFull returned error: %v", err)
	}
	if full != strings.TrimSpace(strings.ReplaceAll(content, "\t", "    ")) {
		t.Fatalf("unexpected full content: %q", full)
	}

	all, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if all != content {
		t.Fatalf("unexpected raw content: %q", all)
	}

	if !IsNoteFile("example.org") || !IsNoteFile("example.norg") {
		t.Fatal("expected supported note extensions to return true")
	}
	if IsNoteFile("example.json") {
		t.Fatal("expected unsupported extension to return false")
	}
}

func TestCreateTodoNoteAndTodoMutations(t *testing.T) {
	root := t.TempDir()

	path, err := CreateTodoNote(root, "projects")
	if err != nil {
		t.Fatalf("CreateTodoNote returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != "# Todo\n\n- [ ] \n" {
		t.Fatalf("unexpected todo template: %q", string(data))
	}

	if err := ToggleTodoLine(path, 2); err != nil {
		t.Fatalf("ToggleTodoLine returned error: %v", err)
	}
	text := mustRead(t, path)
	if !strings.Contains(text, "- [x] ") {
		t.Fatalf("expected toggled todo, got %q", text)
	}

	if err := EditTodoLine(path, 2, "updated task"); err != nil {
		t.Fatalf("EditTodoLine returned error: %v", err)
	}
	text = mustRead(t, path)
	if !strings.Contains(text, "- [x] updated task") {
		t.Fatalf("expected edited todo line, got %q", text)
	}

	if err := AddTodoItem(path, "second task"); err != nil {
		t.Fatalf("AddTodoItem returned error: %v", err)
	}
	text = mustRead(t, path)
	if !strings.Contains(text, "- [ ] second task") {
		t.Fatalf("expected appended todo item, got %q", text)
	}

	if err := DeleteTodoLine(path, 3); err != nil {
		t.Fatalf("DeleteTodoLine returned error: %v", err)
	}
	text = mustRead(t, path)
	if strings.Contains(text, "second task") {
		t.Fatalf("expected appended todo item to be deleted, got %q", text)
	}
}

func TestTodoMutationsRejectInvalidLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.md")
	writeFile(t, path, "plain text\n")

	if err := ToggleTodoLine(path, 0); err == nil {
		t.Fatal("expected non-todo line to be rejected")
	}
	if err := ToggleTodoLine(path, 5); err == nil {
		t.Fatal("expected out-of-range line to be rejected")
	}
	if err := DeleteTodoLine(path, 5); err == nil {
		t.Fatal("expected out-of-range delete to be rejected")
	}
	if err := EditTodoLine(path, 5, "nope"); err == nil {
		t.Fatal("expected out-of-range edit to be rejected")
	}
}

func TestDiscoverFindsNotesAndSkipsHiddenDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "inbox.md"), strings.Join([]string{
		"---",
		"tags: alpha, beta",
		"---",
		"# Inbox",
		"Body",
	}, "\n"))
	writeFile(t, filepath.Join(root, "work", "project.md"), strings.Join([]string{
		"---",
		"encrypted: true",
		"tags: secret",
		"---",
		"# Project",
		"Hidden body",
	}, "\n"))
	writeFile(t, filepath.Join(root, ".hidden", "skip.md"), "# Skip me")
	writeFile(t, filepath.Join(root, "notes.txt"), "plain text body")

	got, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 discovered notes, got %d", len(got))
	}

	if got[0].RelPath != "inbox.md" || got[0].Category != "uncategorized" {
		t.Fatalf("unexpected root note metadata: %#v", got[0])
	}
	if strings.Join(got[0].Tags, ",") != "alpha,beta" {
		t.Fatalf("unexpected root note tags: %v", got[0].Tags)
	}
	if got[2].RelPath != "work/project.md" || got[2].Category != "work" || !got[2].Encrypted {
		t.Fatalf("unexpected nested note metadata: %#v", got[2])
	}
}

func TestDiscoverTemporaryCreatesTempRootAndUsesEmptyCategoryForRoot(t *testing.T) {
	root := t.TempDir()

	got, err := DiscoverTemporary(root)
	if err != nil {
		t.Fatalf("DiscoverTemporary returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no temporary notes, got %d", len(got))
	}
	if _, err := os.Stat(TempRoot(root)); err != nil {
		t.Fatalf("expected temp root to be created: %v", err)
	}

	writeFile(t, filepath.Join(TempRoot(root), "scratch.md"), "# Scratch")
	writeFile(t, filepath.Join(TempRoot(root), "today", "todo.md"), "# Todo")

	got, err = DiscoverTemporary(root)
	if err != nil {
		t.Fatalf("DiscoverTemporary returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 temporary notes, got %d", len(got))
	}
	if got[0].Category != "" {
		t.Fatalf("expected root temp note to have empty category, got %q", got[0].Category)
	}
	if got[1].Category != "today" {
		t.Fatalf("expected nested temp note category to be today, got %q", got[1].Category)
	}
}

func TestDiscoverCategoriesAndMoveCategory(t *testing.T) {
	root := t.TempDir()
	if err := CreateCategory(root, "work/projects"); err != nil {
		t.Fatalf("CreateCategory returned error: %v", err)
	}
	if err := CreateCategory(root, "personal"); err != nil {
		t.Fatalf("CreateCategory returned error: %v", err)
	}
	if err := CreateCategory(root, ".hidden"); err != nil {
		t.Fatalf("CreateCategory returned error: %v", err)
	}

	cats, err := DiscoverCategories(root)
	if err != nil {
		t.Fatalf("DiscoverCategories returned error: %v", err)
	}
	if len(cats) != 4 {
		t.Fatalf("expected 4 categories including virtual root, got %d", len(cats))
	}
	if cats[0].RelPath != "" || cats[0].Name != "All notes" {
		t.Fatalf("unexpected virtual root category: %#v", cats[0])
	}
	for _, cat := range cats {
		if strings.HasPrefix(cat.RelPath, ".hidden") {
			t.Fatalf("expected hidden category to be skipped, got %#v", cat)
		}
	}

	if err := MoveCategory(root, "work", "archive/work"); err != nil {
		t.Fatalf("MoveCategory returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "archive", "work", "projects")); err != nil {
		t.Fatalf("expected moved category tree: %v", err)
	}
	if err := MoveCategory(root, "archive", "archive/subdir"); err == nil {
		t.Fatal("expected self-nesting move to be rejected")
	}
}

func TestDeleteCategoryMovesDirectoryToTrash(t *testing.T) {
	root := t.TempDir()
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)

	writeFile(t, filepath.Join(root, "projects", "note.md"), "body")
	if err := DeleteCategory(root, "projects"); err != nil {
		t.Fatalf("DeleteCategory returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "projects")); !os.IsNotExist(err) {
		t.Fatalf("expected category to be removed from notes root, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(xdgData, "Trash", "files", "projects")); err != nil {
		t.Fatalf("expected category to be moved to trash: %v", err)
	}
	if err := DeleteCategory(root, "."); err == nil {
		t.Fatal("expected root delete to be rejected")
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	return string(data)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
