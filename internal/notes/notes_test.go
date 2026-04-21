package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
		require.Failf(t, "assertion failed", "expected heading title, got %q", got)
	}
	if got := ExtractTitleOrFirstLine("---\nfoo: bar\n---\n* first line"); got != "first line" {
		require.Failf(t, "assertion failed", "expected first line fallback, got %q", got)
	}
	if got := fallbackTitleFromFilename("project-notes.md"); got != "project notes" {
		require.Failf(t, "assertion failed", "unexpected fallback title: %q", got)
	}
	if got := fallbackTitleFromFilename(".md"); got != "Untitled" {
		require.Failf(t, "assertion failed", "expected Untitled fallback, got %q", got)
	}
}

func TestSlugifyAndWordCount(t *testing.T) {
	if got := Slugify("  Hello, World + Draft/1  "); got != "hello-world-draft-1" {
		require.Failf(t, "assertion failed", "unexpected slug: %q", got)
	}
	if got := Slugify("!!!"); got != "" {
		require.Failf(t, "assertion failed", "expected empty slug for punctuation-only input, got %q", got)
	}

	raw := strings.Join([]string{
		"---",
		"tags: alpha, beta",
		"---",
		"one two",
		"three",
	}, "\n")
	if got := WordCount(raw); got != 3 {
		require.Failf(t, "assertion failed", "expected word count 3, got %d", got)
	}
	if got := ReadingTimeMinutes(0); got != 0 {
		require.Failf(t, "assertion failed", "expected zero reading time, got %d", got)
	}
	if got := ReadingTimeMinutes(1); got != 1 {
		require.Failf(t, "assertion failed", "expected minimum reading time of 1, got %d", got)
	}
	if got := ReadingTimeMinutes(226); got != 2 {
		require.Failf(t, "assertion failed", "expected rounded reading time of 2, got %d", got)
	}

	// CharCount strips frontmatter and counts all non-newline runes including spaces.
	if got := CharCount(raw); got != 12 { // "one two\nthree" = 12 chars (space included, newline excluded)
		require.Failf(t, "assertion failed", "expected char count 13, got %d", got)
	}
	if got := CharCount("no frontmatter"); got != 14 {
		require.Failf(t, "assertion failed", "expected char count 14, got %d", got)
	}
	if got := CharCount(""); got != 0 {
		require.Failf(t, "assertion failed", "expected char count 0 for empty string, got %d", got)
	}
}

func TestCleanRelativePathAndReplaceOrInsertRootTitle(t *testing.T) {
	if got := cleanRelativePath(" ../docs/note ", true); got != "docs/note.md" {
		require.Failf(t, "assertion failed", "unexpected cleaned path: %q", got)
	}
	if got := cleanRelativePath(".", true); got != "" {
		require.Failf(t, "assertion failed", "expected empty cleaned path, got %q", got)
	}

	replaced := replaceOrInsertRootTitle("# Old\n\nBody", "New")
	if replaced != "# New\n\nBody" {
		require.Failf(t, "assertion failed", "unexpected replaced title content: %q", replaced)
	}
	inserted := replaceOrInsertRootTitle("Body", "Inserted")
	if inserted != "# Inserted\n\nBody" {
		require.Failf(t, "assertion failed", "unexpected inserted title content: %q", inserted)
	}
}

func TestRenameFromTitleDeletesEmptyTemporaryNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".new-20260401-120000.md")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}

	newPath, renamed, err := RenameFromTitle(path)
	if err != nil {
		require.Failf(t, "assertion failed", "RenameFromTitle returned error: %v", err)
	}
	if newPath != "" || renamed {
		require.Failf(t, "assertion failed", "expected deleted temporary note, got path=%q renamed=%v", newPath, renamed)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		require.Failf(t, "assertion failed", "expected temp note to be deleted, stat err=%v", err)
	}
}

func TestRenameFromTitleUsesUniquePath(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "meeting-notes.md")
	if err := os.WriteFile(existing, []byte("existing"), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}

	path := filepath.Join(dir, ".new-20260401-120000.md")
	if err := os.WriteFile(path, []byte("# Meeting Notes\nBody"), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}

	newPath, renamed, err := RenameFromTitle(path)
	if err != nil {
		require.Failf(t, "assertion failed", "RenameFromTitle returned error: %v", err)
	}
	if !renamed {
		require.FailNow(t, "expected note to be renamed")
	}
	if filepath.Base(newPath) != "meeting-notes-2.md" {
		require.Failf(t, "assertion failed", "expected unique target path, got %q", newPath)
	}
}

func TestRenameNoteTitleUpdatesContentAndFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "draft.md")
	if err := os.WriteFile(path, []byte("# Old Title\n\nBody"), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}

	newPath, renamed, err := RenameNoteTitle(path, "Fresh Title")
	if err != nil {
		require.Failf(t, "assertion failed", "RenameNoteTitle returned error: %v", err)
	}
	if !renamed {
		require.FailNow(t, "expected note to be renamed")
	}
	if filepath.Base(newPath) != "fresh-title.md" {
		require.Failf(t, "assertion failed", "unexpected new path: %q", newPath)
	}

	data, err := os.ReadFile(newPath)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadFile returned error: %v", err)
	}
	if !strings.HasPrefix(string(data), "# Fresh Title") {
		require.Failf(t, "assertion failed", "expected updated title in file, got %q", string(data))
	}
}

func TestMoveNoteValidatesAndMovesFile(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "draft.md")
	if err := os.WriteFile(oldPath, []byte("body"), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}

	if err := MoveNote(root, "", "next.md"); err == nil {
		require.FailNow(t, "expected error for empty source path")
	}

	if err := MoveNote(root, "../draft.md", "nested/renamed"); err != nil {
		require.Failf(t, "assertion failed", "MoveNote returned error: %v", err)
	}
	newPath := filepath.Join(root, "nested", "renamed.md")
	if _, err := os.Stat(newPath); err != nil {
		require.Failf(t, "assertion failed", "expected moved note at %q: %v", newPath, err)
	}

	if err := MoveNote(root, "nested/renamed.md", "nested/renamed.md"); err != nil {
		require.Failf(t, "assertion failed", "expected no-op move to succeed, got %v", err)
	}
}

func TestCreateNoteAndReadHelpers(t *testing.T) {
	root := t.TempDir()

	path, err := CreateNote(root, "daily")
	if err != nil {
		require.Failf(t, "assertion failed", "CreateNote returned error: %v", err)
	}
	if filepath.Ext(path) != ".md" {
		require.Failf(t, "assertion failed", "expected markdown note, got %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		require.Failf(t, "assertion failed", "expected note to exist: %v", err)
	}

	content := "Line 1\n\tIndented\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}

	preview, err := ReadPreview(path)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadPreview returned error: %v", err)
	}
	if !strings.Contains(preview, "    Indented") {
		require.Failf(t, "assertion failed", "expected tabs to expand in preview, got %q", preview)
	}

	full, err := ReadFull(path)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadFull returned error: %v", err)
	}
	if full != strings.TrimSpace(strings.ReplaceAll(content, "\t", "    ")) {
		require.Failf(t, "assertion failed", "unexpected full content: %q", full)
	}

	all, err := ReadAll(path)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadAll returned error: %v", err)
	}
	if all != content {
		require.Failf(t, "assertion failed", "unexpected raw content: %q", all)
	}

	if !IsNoteFile("example.org") || !IsNoteFile("example.norg") {
		require.FailNow(t, "expected supported note extensions to return true")
	}
	if IsNoteFile("example.json") {
		require.FailNow(t, "expected unsupported extension to return false")
	}
}

func TestCreateTemporaryNoteAndDeleteNote(t *testing.T) {
	root := t.TempDir()
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)

	path, err := CreateTemporaryNote(root)
	if err != nil {
		require.Failf(t, "assertion failed", "CreateTemporaryNote returned error: %v", err)
	}
	if !strings.HasPrefix(path, TempRoot(root)) {
		require.Failf(t, "assertion failed", "expected temporary note under temp root, got %q", path)
	}

	if _, err := DeleteNote(path); err != nil {
		require.Failf(t, "assertion failed", "DeleteNote returned error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		require.Failf(t, "assertion failed", "expected original note to be gone, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(xdgData, "Trash", "files", filepath.Base(path))); err != nil {
		require.Failf(t, "assertion failed", "expected deleted note in trash, got %v", err)
	}
}

func TestNotePresentationHelpers(t *testing.T) {
	n := Note{
		Name:      "daily.md",
		RelPath:   "work/daily.md",
		Preview:   "preview body",
		Tags:      []string{"alpha", "beta"},
		TitleText: "",
	}

	if got := n.Title(); got != "daily.md" {
		require.Failf(t, "assertion failed", "expected file name fallback title, got %q", got)
	}
	if got := n.Description(); got != "work/daily.md" {
		require.Failf(t, "assertion failed", "unexpected description: %q", got)
	}
	filter := n.FilterValue()
	for _, fragment := range []string{"daily.md", "work/daily.md", "preview body", "alpha beta"} {
		if !strings.Contains(filter, fragment) {
			require.Failf(t, "assertion failed", "expected filter value to contain %q, got %q", fragment, filter)
		}
	}

	n.TitleText = "Daily"
	if got := n.Title(); got != "Daily" {
		require.Failf(t, "assertion failed", "expected explicit title, got %q", got)
	}
}

func TestCreateTodoNoteAndTodoMutations(t *testing.T) {
	root := t.TempDir()

	path, err := CreateTodoNote(root, "projects")
	if err != nil {
		require.Failf(t, "assertion failed", "CreateTodoNote returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadFile returned error: %v", err)
	}
	if string(data) != "# Todo\n\n- [ ] \n" {
		require.Failf(t, "assertion failed", "unexpected todo template: %q", string(data))
	}

	if err := ToggleTodoLine(path, 2); err != nil {
		require.Failf(t, "assertion failed", "ToggleTodoLine returned error: %v", err)
	}
	text := mustRead(t, path)
	if !strings.Contains(text, "- [x] ") {
		require.Failf(t, "assertion failed", "expected toggled todo, got %q", text)
	}

	if err := EditTodoLine(path, 2, "updated task"); err != nil {
		require.Failf(t, "assertion failed", "EditTodoLine returned error: %v", err)
	}
	text = mustRead(t, path)
	if !strings.Contains(text, "- [x] updated task") {
		require.Failf(t, "assertion failed", "expected edited todo line, got %q", text)
	}

	if err := AddTodoItem(path, "second task"); err != nil {
		require.Failf(t, "assertion failed", "AddTodoItem returned error: %v", err)
	}
	text = mustRead(t, path)
	if !strings.Contains(text, "- [ ] second task") {
		require.Failf(t, "assertion failed", "expected appended todo item, got %q", text)
	}

	if err := UpdateTodoDueDate(path, 2, "2026-04-12"); err != nil {
		require.Failf(t, "assertion failed", "UpdateTodoDueDate returned error: %v", err)
	}
	text = mustRead(t, path)
	if !strings.Contains(text, "- [x] updated task [due:2026-04-12]") {
		require.Failf(t, "assertion failed", "expected due date to be added, got %q", text)
	}

	if err := UpdateTodoPriority(path, 2, "1"); err != nil {
		require.Failf(t, "assertion failed", "UpdateTodoPriority returned error: %v", err)
	}
	text = mustRead(t, path)
	if !strings.Contains(text, "- [x] updated task [due:2026-04-12] [p1]") {
		require.Failf(t, "assertion failed", "expected priority to be added, got %q", text)
	}

	if err := UpdateTodoPriority(path, 2, ""); err != nil {
		require.Failf(t, "assertion failed", "UpdateTodoPriority clear returned error: %v", err)
	}
	text = mustRead(t, path)
	if strings.Contains(text, "[p1]") {
		require.Failf(t, "assertion failed", "expected priority to be cleared, got %q", text)
	}

	if err := UpdateTodoDueDate(path, 2, ""); err != nil {
		require.Failf(t, "assertion failed", "UpdateTodoDueDate clear returned error: %v", err)
	}
	text = mustRead(t, path)
	if strings.Contains(text, "[due:") {
		require.Failf(t, "assertion failed", "expected due date to be cleared, got %q", text)
	}

	if err := DeleteTodoLine(path, 3); err != nil {
		require.Failf(t, "assertion failed", "DeleteTodoLine returned error: %v", err)
	}
	text = mustRead(t, path)
	if strings.Contains(text, "second task") {
		require.Failf(t, "assertion failed", "expected appended todo item to be deleted, got %q", text)
	}
}

func TestTodoMutationsRejectInvalidLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.md")
	writeFile(t, path, "plain text\n")

	if err := ToggleTodoLine(path, 0); err == nil {
		require.FailNow(t, "expected non-todo line to be rejected")
	}
	if err := ToggleTodoLine(path, 5); err == nil {
		require.FailNow(t, "expected out-of-range line to be rejected")
	}
	if err := DeleteTodoLine(path, 5); err == nil {
		require.FailNow(t, "expected out-of-range delete to be rejected")
	}
	if err := EditTodoLine(path, 5, "nope"); err == nil {
		require.FailNow(t, "expected out-of-range edit to be rejected")
	}
	if err := UpdateTodoDueDate(path, 0, "2026-99-99"); err == nil {
		require.FailNow(t, "expected invalid due date to be rejected")
	}
	if err := UpdateTodoPriority(path, 0, "zero"); err == nil {
		require.FailNow(t, "expected invalid priority to be rejected")
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
		require.Failf(t, "assertion failed", "Discover returned error: %v", err)
	}
	if len(got) != 3 {
		require.Failf(t, "assertion failed", "expected 3 discovered notes, got %d", len(got))
	}

	if got[0].RelPath != "inbox.md" || got[0].Category != "uncategorized" {
		require.Failf(t, "assertion failed", "unexpected root note metadata: %#v", got[0])
	}
	if strings.Join(got[0].Tags, ",") != "alpha,beta" {
		require.Failf(t, "assertion failed", "unexpected root note tags: %v", got[0].Tags)
	}
	if got[2].RelPath != "work/project.md" || got[2].Category != "work" || !got[2].Encrypted {
		require.Failf(t, "assertion failed", "unexpected nested note metadata: %#v", got[2])
	}
}

func TestDiscoverTemporaryCreatesTempRootAndUsesEmptyCategoryForRoot(t *testing.T) {
	root := t.TempDir()

	got, err := DiscoverTemporary(root)
	if err != nil {
		require.Failf(t, "assertion failed", "DiscoverTemporary returned error: %v", err)
	}
	if len(got) != 0 {
		require.Failf(t, "assertion failed", "expected no temporary notes, got %d", len(got))
	}
	if _, err := os.Stat(TempRoot(root)); err != nil {
		require.Failf(t, "assertion failed", "expected temp root to be created: %v", err)
	}

	writeFile(t, filepath.Join(TempRoot(root), "scratch.md"), "# Scratch")
	writeFile(t, filepath.Join(TempRoot(root), "today", "todo.md"), "# Todo")

	got, err = DiscoverTemporary(root)
	if err != nil {
		require.Failf(t, "assertion failed", "DiscoverTemporary returned error: %v", err)
	}
	if len(got) != 2 {
		require.Failf(t, "assertion failed", "expected 2 temporary notes, got %d", len(got))
	}
	if got[0].Category != "" {
		require.Failf(t, "assertion failed", "expected root temp note to have empty category, got %q", got[0].Category)
	}
	if got[1].Category != "today" {
		require.Failf(t, "assertion failed", "expected nested temp note category to be today, got %q", got[1].Category)
	}
}

func TestDiscoverCategoriesAndMoveCategory(t *testing.T) {
	root := t.TempDir()
	if err := CreateCategory(root, "work/projects"); err != nil {
		require.Failf(t, "assertion failed", "CreateCategory returned error: %v", err)
	}
	if err := CreateCategory(root, "personal"); err != nil {
		require.Failf(t, "assertion failed", "CreateCategory returned error: %v", err)
	}
	if err := CreateCategory(root, ".hidden"); err != nil {
		require.Failf(t, "assertion failed", "CreateCategory returned error: %v", err)
	}

	cats, err := DiscoverCategories(root)
	if err != nil {
		require.Failf(t, "assertion failed", "DiscoverCategories returned error: %v", err)
	}
	if len(cats) != 4 {
		require.Failf(t, "assertion failed", "expected 4 categories including virtual root, got %d", len(cats))
	}
	if cats[0].RelPath != "" || cats[0].Name != "All notes" {
		require.Failf(t, "assertion failed", "unexpected virtual root category: %#v", cats[0])
	}
	for _, cat := range cats {
		if strings.HasPrefix(cat.RelPath, ".hidden") {
			require.Failf(t, "assertion failed", "expected hidden category to be skipped, got %#v", cat)
		}
	}

	if err := MoveCategory(root, "work", "archive/work"); err != nil {
		require.Failf(t, "assertion failed", "MoveCategory returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "archive", "work", "projects")); err != nil {
		require.Failf(t, "assertion failed", "expected moved category tree: %v", err)
	}
	if err := MoveCategory(root, "archive", "archive/subdir"); err == nil {
		require.FailNow(t, "expected self-nesting move to be rejected")
	}
}

func TestDeleteCategoryMovesDirectoryToTrash(t *testing.T) {
	root := t.TempDir()
	xdgData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgData)

	writeFile(t, filepath.Join(root, "projects", "note.md"), "body")
	if _, err := DeleteCategory(root, "projects"); err != nil {
		require.Failf(t, "assertion failed", "DeleteCategory returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "projects")); !os.IsNotExist(err) {
		require.Failf(t, "assertion failed", "expected category to be removed from notes root, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(xdgData, "Trash", "files", "projects")); err != nil {
		require.Failf(t, "assertion failed", "expected category to be moved to trash: %v", err)
	}
	if _, err := DeleteCategory(root, "."); err == nil {
		require.FailNow(t, "expected root delete to be rejected")
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadFile returned error: %v", err)
	}
	return string(data)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		require.Failf(t, "assertion failed", "MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}
}

func TestMoveNoteBetweenRootsMovesAcrossRoots(t *testing.T) {
	root := t.TempDir()
	srcRoot := TempRoot(root)
	dstRoot := filepath.Join(root, "archive")
	require.NoError(t, os.MkdirAll(srcRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "draft.md"), []byte("# Draft\n"), 0o644))

	err := MoveNoteBetweenRoots(srcRoot, "draft.md", dstRoot, "tmp/draft.md")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(srcRoot, "draft.md"))
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(dstRoot, "tmp", "draft.md"))
	require.NoError(t, err)
}

func TestAppendCapture(t *testing.T) {
	dir := t.TempDir()

	if err := AppendCapture(dir, "inbox.md", "first entry"); err != nil {
		t.Fatalf("unexpected error on create: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "inbox.md"))
	require.NoError(t, err)
	raw := string(content)
	if !strings.Contains(raw, "# inbox") {
		t.Errorf("expected heading, got %q", raw)
	}
	if !strings.Contains(raw, "first entry") {
		t.Errorf("expected captured text, got %q", raw)
	}
	if !strings.Contains(raw, "- [") {
		t.Errorf("expected timestamped bullet, got %q", raw)
	}

	if err := AppendCapture(dir, "inbox.md", "second entry"); err != nil {
		t.Fatalf("unexpected error on append: %v", err)
	}
	content, err = os.ReadFile(filepath.Join(dir, "inbox.md"))
	require.NoError(t, err)
	raw = string(content)
	if !strings.Contains(raw, "first entry") || !strings.Contains(raw, "second entry") {
		t.Errorf("expected both entries, got %q", raw)
	}
}

func TestAppendCaptureNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	require.NoError(t, os.WriteFile(path, []byte("existing content"), 0o644))

	require.NoError(t, AppendCapture(dir, "notes.md", "appended"))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	if !strings.Contains(string(raw), "existing content") {
		t.Error("original content was lost")
	}
	if !strings.Contains(string(raw), "appended") {
		t.Error("appended content not found")
	}
}

func TestOpenOrCreateDailyNoteCreatesWithDefaultHeading(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC)

	path, created, err := OpenOrCreateDailyNote(root, "daily", "", now)
	if err != nil {
		require.Failf(t, "assertion failed", "OpenOrCreateDailyNote returned error: %v", err)
	}
	if !created {
		require.FailNow(t, "expected created=true for new daily note")
	}
	expected := filepath.Join(root, "daily", "2026-04-13.md")
	if path != expected {
		require.Failf(t, "assertion failed", "expected path %q, got %q", expected, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadFile returned error: %v", err)
	}
	if string(data) != "# 2026-04-13\n\n" {
		require.Failf(t, "assertion failed", "unexpected default content: %q", string(data))
	}
}

func TestOpenOrCreateDailyNoteExistingFileNotRecreated(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC)

	path, created, err := OpenOrCreateDailyNote(root, "daily", "", now)
	if err != nil || !created {
		require.Failf(t, "assertion failed", "first call failed: err=%v created=%v", err, created)
	}

	// Write custom content so we can verify the file is not overwritten.
	if err := os.WriteFile(path, []byte("existing content"), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}

	path2, created2, err := OpenOrCreateDailyNote(root, "daily", "", now)
	if err != nil {
		require.Failf(t, "assertion failed", "second call returned error: %v", err)
	}
	if created2 {
		require.FailNow(t, "expected created=false when daily note already exists")
	}
	if path2 != path {
		require.Failf(t, "assertion failed", "expected same path on second call, got %q", path2)
	}

	data, err := os.ReadFile(path2)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadFile returned error: %v", err)
	}
	if string(data) != "existing content" {
		require.Failf(t, "assertion failed", "expected existing content to be preserved, got %q", string(data))
	}
}

func TestOpenOrCreateDailyNoteUsesTemplate(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 4, 13, 10, 30, 0, 0, time.UTC)

	tmplDir := filepath.Join(root, TemplatesDirName)
	require.NoError(t, os.MkdirAll(tmplDir, 0o755))
	tmplPath := filepath.Join(tmplDir, "daily.md")
	require.NoError(t, os.WriteFile(tmplPath, []byte("# Daily: {{date}}\n\nTime: {{time}}\n"), 0o644))

	path, created, err := OpenOrCreateDailyNote(root, "journal", tmplPath, now)
	if err != nil {
		require.Failf(t, "assertion failed", "OpenOrCreateDailyNote returned error: %v", err)
	}
	if !created {
		require.FailNow(t, "expected created=true for new daily note")
	}

	expected := filepath.Join(root, "journal", "2026-04-13.md")
	if path != expected {
		require.Failf(t, "assertion failed", "expected path %q, got %q", expected, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadFile returned error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# Daily: 2026-04-13") {
		require.Failf(t, "assertion failed", "expected substituted date in heading, got %q", content)
	}
	if !strings.Contains(content, "Time: 10:30") {
		require.Failf(t, "assertion failed", "expected substituted time, got %q", content)
	}
	if strings.Contains(content, "{{") {
		require.Failf(t, "assertion failed", "expected all template vars to be substituted, got %q", content)
	}
}

func TestRewriteWikilinks(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{
			in:  "See [[my note]] for details.",
			out: "See [my note](#wikilink:my%20note) for details.",
		},
		{
			in:  "[[first]] and [[second]]",
			out: "[first](#wikilink:first) and [second](#wikilink:second)",
		},
		{
			in:  "No wikilinks here.",
			out: "No wikilinks here.",
		},
		{
			in:  "[regular link](https://example.com) stays unchanged",
			out: "[regular link](https://example.com) stays unchanged",
		},
		{
			in:  "[[note with spaces]]",
			out: "[note with spaces](#wikilink:note%20with%20spaces)",
		},
		{
			in:  "Use [[target note|Shown Label]] here.",
			out: "Use [Shown Label](#wikilink:target%20note) here.",
		},
	}

	for _, tc := range cases {
		got := RewriteWikilinks(tc.in)
		if got != tc.out {
			require.Failf(t, "assertion failed", "RewriteWikilinks(%q) = %q; want %q", tc.in, got, tc.out)
		}
	}
}

func TestExtractWikilinks(t *testing.T) {
	content := "See [[alpha]] and [[beta|Beta Label]].\nAlso [[alpha]] again and [[gamma]]."
	got := ExtractWikilinks(content)
	want := []string{"alpha", "beta", "gamma"}
	require.Equal(t, want, got)
}

func TestSplitWikilinkTargetLabel(t *testing.T) {
	target, label := SplitWikilinkTargetLabel("[[Plan/2026|Quarterly Plan]]")
	require.Equal(t, "Plan/2026", target)
	require.Equal(t, "Quarterly Plan", label)

	target, label = SplitWikilinkTargetLabel("Standalone Title")
	require.Equal(t, "Standalone Title", target)
	require.Equal(t, "Standalone Title", label)
}

func TestFindNoteByWikilink(t *testing.T) {
	ns := []Note{
		{Name: "project-notes.md", TitleText: "Project Notes"},
		{Name: "ideas.md", TitleText: "Ideas"},
		{Name: "meeting-2026.md", TitleText: "Meeting 2026"},
	}

	// Exact title match (case-insensitive)
	n := FindNoteByWikilink(ns, "Project Notes")
	if n == nil || n.Name != "project-notes.md" {
		require.Failf(t, "assertion failed", "expected project-notes.md, got %v", n)
	}

	n = FindNoteByWikilink(ns, "project notes")
	if n == nil || n.Name != "project-notes.md" {
		require.Failf(t, "assertion failed", "expected project-notes.md for lowercase match, got %v", n)
	}

	// Filename stem match
	n = FindNoteByWikilink(ns, "ideas")
	if n == nil || n.Name != "ideas.md" {
		require.Failf(t, "assertion failed", "expected ideas.md for stem match, got %v", n)
	}

	// Prefix title match
	n = FindNoteByWikilink(ns, "Meeting")
	if n == nil || n.Name != "meeting-2026.md" {
		require.Failf(t, "assertion failed", "expected meeting-2026.md for prefix match, got %v", n)
	}

	// Not found
	n = FindNoteByWikilink(ns, "nonexistent")
	if n != nil {
		require.Failf(t, "assertion failed", "expected nil for nonexistent target, got %v", n)
	}
}

func TestDiscoverNotePopulatesSize(t *testing.T) {
	root := t.TempDir()
	content := "hello world"
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	notes, err := Discover(root)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	require.Equal(t, int64(len(content)), notes[0].Size)
}

func TestDiscoverNoteCreatedAtFromFrontmatter(t *testing.T) {
	root := t.TempDir()
	content := "---\ndate: 2024-03-15\n---\nbody"
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	notes, err := Discover(root)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	want := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	require.Equal(t, want, notes[0].CreatedAt)
}

func TestDiscoverNoteCreatedAtFallsBackToModTime(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("body"), 0o644); err != nil {
		t.Fatal(err)
	}
	notes, err := Discover(root)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	require.Equal(t, notes[0].ModTime, notes[0].CreatedAt)
}
