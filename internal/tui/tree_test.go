package tui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/notes"
	notesync "atbuy/noteui/internal/sync"
)

func modelWithNotes(ns []notes.Note) Model {
	m := Model{expanded: make(map[string]bool)}
	m.notes = ns
	return m
}

func TestNoteMatchesPlainTerms(t *testing.T) {
	n := notes.Note{
		Name:      "meeting.md",
		RelPath:   "work/meeting.md",
		Preview:   "discussed the roadmap",
		Tags:      []string{"work", "planning"},
		TitleText: "Q1 Meeting",
	}
	m := modelWithNotes(nil)

	// matches title
	if !m.noteMatches(n, "q1") {
		require.FailNow(t, "expected match on title")
	}
	// matches filename
	if !m.noteMatches(n, "meeting") {
		require.FailNow(t, "expected match on filename")
	}
	// matches rel path
	if !m.noteMatches(n, "work/meeting") {
		require.FailNow(t, "expected match on relpath")
	}
	// matches preview
	if !m.noteMatches(n, "roadmap") {
		require.FailNow(t, "expected match on preview")
	}
	// multi-term: all terms must match
	if !m.noteMatches(n, "q1 roadmap") {
		require.FailNow(t, "expected match for both terms present")
	}
	if m.noteMatches(n, "q1 budget") {
		require.FailNow(t, "expected no match when second term is absent")
	}
	// empty query matches everything
	if !m.noteMatches(n, "") {
		require.FailNow(t, "expected empty query to match everything")
	}
}

func TestNoteMatchesTagSearch(t *testing.T) {
	n := notes.Note{
		Name:    "note.md",
		RelPath: "note.md",
		Tags:    []string{"urgent", "alpha"},
	}
	m := modelWithNotes(nil)

	if !m.noteMatches(n, "#urg") {
		require.FailNow(t, "expected partial tag match")
	}
	if m.noteMatches(n, "#missing") {
		require.FailNow(t, "expected no match for absent tag")
	}
	// bare # matches everything
	if !m.noteMatches(n, "#") {
		require.FailNow(t, "expected bare # to match everything")
	}
}

func TestNoteMatchesEncryptedHidesPreview(t *testing.T) {
	n := notes.Note{
		RelPath:   "secret.md",
		Preview:   "top secret content",
		Encrypted: true,
	}
	m := modelWithNotes(nil)

	// should not match on actual preview content when encrypted
	if m.noteMatches(n, "secret content") {
		require.FailNow(t, "expected encrypted preview content not to match")
	}
	// should match on the placeholder
	if !m.noteMatches(n, "encrypted") {
		require.FailNow(t, "expected match on encrypted placeholder")
	}
}

func TestRemoteNoteMatches(t *testing.T) {
	n := notesync.RemoteNoteMeta{
		ID:      "abc",
		RelPath: "work/plan.md",
		Title:   "Q2 Plan",
	}
	m := modelWithNotes(nil)

	if !m.remoteNoteMatches(n, "plan") {
		require.FailNow(t, "expected match on relpath")
	}
	if !m.remoteNoteMatches(n, "work") {
		require.FailNow(t, "expected match on directory in relpath")
	}
	if m.remoteNoteMatches(n, "budget") {
		require.FailNow(t, "expected no match for absent term")
	}
	if !m.remoteNoteMatches(n, "") {
		require.FailNow(t, "expected empty query to match everything")
	}
}

func TestCategoryMatches(t *testing.T) {
	m := modelWithNotes(nil)
	cat := notes.Category{Name: "Projects", RelPath: "work/projects"}

	if !m.categoryMatches(cat, "proj") {
		require.FailNow(t, "expected match on category name")
	}
	if !m.categoryMatches(cat, "work") {
		require.FailNow(t, "expected match on relpath")
	}
	if m.categoryMatches(cat, "personal") {
		require.FailNow(t, "expected no match for absent term")
	}
}

func TestCategorySubtreeMatchesNote(t *testing.T) {
	n := notes.Note{
		RelPath:   "work/projects/alpha.md",
		TitleText: "Alpha Project",
		Preview:   "project body",
	}
	m := modelWithNotes([]notes.Note{n})
	m.categories = []notes.Category{}

	// "work" subtree contains "alpha" which matches the note
	if !m.categorySubtreeMatches("work", "alpha") {
		require.FailNow(t, "expected subtree match via note")
	}
	// "personal" subtree has nothing
	if m.categorySubtreeMatches("personal", "alpha") {
		require.FailNow(t, "expected no subtree match for unrelated parent")
	}
}

func TestCategorySubtreeMatchesNestedCategory(t *testing.T) {
	m := modelWithNotes(nil)
	m.categories = []notes.Category{
		{Name: "Alpha", RelPath: "work/alpha"},
	}

	if !m.categorySubtreeMatches("work", "alpha") {
		require.FailNow(t, "expected subtree match via nested category name")
	}
	if m.categorySubtreeMatches("work", "personal") {
		require.FailNow(t, "expected no match when term is absent from subtree")
	}
}

func TestDirectChildNotesSortsAlphabetically(t *testing.T) {
	ns := []notes.Note{
		{RelPath: "b.md", TitleText: "B"},
		{RelPath: "a.md", TitleText: "A"},
		{RelPath: "work/c.md", TitleText: "C"},
	}
	m := modelWithNotes(ns)
	children := m.directChildNotes("")
	require.Len(t, children, 2)
	if children[0].RelPath != "a.md" || children[1].RelPath != "b.md" {
		require.Failf(t, "assertion failed", "expected alpha sort, got %q %q", children[0].RelPath, children[1].RelPath)
	}
}

func TestDirectChildCategoriesDeduplicates(t *testing.T) {
	m := modelWithNotes(nil)
	m.categories = []notes.Category{
		{Name: "work", RelPath: "work"},
		{Name: "work", RelPath: "work"}, // duplicate
	}
	m.remoteCategories = []notes.Category{
		{Name: "work", RelPath: "work"}, // also a duplicate
	}

	children := m.directChildCategories("")
	require.Len(t, children, 1)
}

func TestDirectChildRemoteNotesFiltersParent(t *testing.T) {
	m := modelWithNotes(nil)
	m.remoteOnlyNotes = []notesync.RemoteNoteMeta{
		{ID: "1", RelPath: "work/plan.md"},
		{ID: "2", RelPath: "personal/diary.md"},
		{ID: "3", RelPath: "work/goals.md"},
	}

	children := m.directChildRemoteNotes("work")
	require.Len(t, children, 2)

	children = m.directChildRemoteNotes("personal")
	require.Len(t, children, 1)

	children = m.directChildRemoteNotes("")
	require.Len(t, children, 0)
}
