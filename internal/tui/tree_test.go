package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

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

func TestFuzzySequenceMatch(t *testing.T) {
	cases := []struct {
		pattern, target string
		want            bool
	}{
		{"ntui", "noteui", true},
		{"cfg", "config", true},
		{"abc", "a-b-c", true},
		{"abc", "acb", false}, // wrong order
		{"xyz", "noteui", false},
		{"", "anything", true}, // empty pattern always matches
		{"a", "", false},
	}
	for _, tc := range cases {
		got := fuzzySequenceMatch(tc.pattern, tc.target)
		if got != tc.want {
			t.Errorf("fuzzySequenceMatch(%q, %q) = %v, want %v", tc.pattern, tc.target, got, tc.want)
		}
	}
}

func TestNoteMatchesFuzzyTitle(t *testing.T) {
	n := notes.Note{
		TitleText: "project configuration",
		Name:      "config.md",
		RelPath:   "work/config.md",
		Preview:   "some body text",
	}
	m := modelWithNotes(nil)

	// fuzzy subsequence on title
	if !m.noteMatches(n, "pjcfg") {
		t.Fatal("expected fuzzy match on title via subsequence")
	}
	// exact substring still works
	if !m.noteMatches(n, "config") {
		t.Fatal("expected exact match on title")
	}
	// term with no possible match
	if m.noteMatches(n, "zzzzz") {
		t.Fatal("expected no match for absent term")
	}
}

func TestNoteMatchesFuzzyPath(t *testing.T) {
	n := notes.Note{
		TitleText: "untitled",
		Name:      "note.md",
		RelPath:   "work/projects/alpha.md",
		Preview:   "",
	}
	m := modelWithNotes(nil)

	// fuzzy subsequence on path
	if !m.noteMatches(n, "wkpalpha") {
		t.Fatal("expected fuzzy match on path")
	}
}

func TestFilterAndScoreNotesOrdering(t *testing.T) {
	// title-exact match should rank above body-only match
	titleMatch := notes.Note{
		TitleText: "configuration guide",
		Name:      "config-guide.md",
		RelPath:   "config-guide.md",
		Preview:   "body text",
	}
	bodyMatch := notes.Note{
		TitleText: "unrelated document",
		Name:      "unrelated.md",
		RelPath:   "unrelated.md",
		Preview:   "this note covers configuration settings",
	}

	result := filterAndScoreNotes([]notes.Note{bodyMatch, titleMatch}, "configuration")
	require.Len(t, result, 2)
	if result[0].RelPath != "config-guide.md" {
		t.Errorf("expected title-exact match first, got %q", result[0].RelPath)
	}
}

func TestFilterAndScoreNotesEmptyQuery(t *testing.T) {
	ns := []notes.Note{
		{RelPath: "a.md", TitleText: "A"},
		{RelPath: "b.md", TitleText: "B"},
	}
	result := filterAndScoreNotes(ns, "")
	require.Len(t, result, 2)
	// order preserved when no query
	if result[0].RelPath != "a.md" {
		t.Errorf("expected original order preserved for empty query")
	}
}

// treeItemSummary is the comparable projection of a treeItem used to assert the
// new single-pass build produces the same tree as the reference logic.
type treeItemSummary struct {
	Kind      treeItemKind
	Name      string
	RelPath   string
	Depth     int
	Expanded  bool
	MatchHint string
}

func summarizeTree(items []treeItem) []treeItemSummary {
	out := make([]treeItemSummary, len(items))
	for i, it := range items {
		out[i] = treeItemSummary{it.Kind, it.Name, it.RelPath, it.Depth, it.Expanded, it.MatchHint}
	}
	return out
}

// referenceTree reproduces the pre-optimization buildTree using the retained
// directChild*/categorySubtreeMatches/filterAndScoreNotes helpers, so the
// optimized build can be checked against it.
func referenceTree(m *Model, query string) []treeItem {
	var out []treeItem
	var walk func(parent string, depth int)
	walk = func(parent string, depth int) {
		effectiveExpanded := func(rel string) bool {
			if rel == "" || query != "" {
				return true
			}
			return m.expanded[rel]
		}
		if parent == "" && depth == 0 {
			out = append(out, treeItem{Kind: treeCategory, Name: "/", RelPath: "", Depth: 0, Expanded: true})
			depth = 1
		}
		for _, cat := range m.directChildCategories(parent) {
			include := query == "" || m.categoryMatches(cat, query) || m.categorySubtreeMatches(cat.RelPath, query)
			if !include {
				continue
			}
			catCopy := cat
			item := treeItem{Kind: treeCategory, Name: cat.Name, RelPath: cat.RelPath, Depth: depth, Expanded: effectiveExpanded(cat.RelPath), Category: &catCopy}
			out = append(out, item)
			if item.Expanded {
				walk(cat.RelPath, depth+1)
			}
		}
		childNotes := m.directChildNotes(parent)
		if query != "" {
			childNotes = filterAndScoreNotes(childNotes, query)
		}
		for _, n := range childNotes {
			noteCopy := n
			hint := ""
			if query != "" {
				hint = findMatchExcerpt(noteCopy, query)
			}
			out = append(out, treeItem{Kind: treeNote, Name: n.Title(), RelPath: n.RelPath, Depth: depth, Note: &noteCopy, MatchHint: hint})
		}
		for _, n := range m.directChildRemoteNotes(parent) {
			if query != "" && !m.remoteNoteMatches(n, query) {
				continue
			}
			remoteCopy := n
			out = append(out, treeItem{Kind: treeRemoteNote, Name: m.remoteOnlyDisplayTitle(n), RelPath: n.RelPath, Depth: depth, RemoteNote: &remoteCopy})
		}
	}
	walk("", 0)
	return out
}

func TestBuildTreeMatchesReferenceAcrossQueries(t *testing.T) {
	ns := []notes.Note{
		{RelPath: "work/projects/alpha.md", Name: "alpha.md", TitleText: "Alpha Plan", Preview: "roadmap details", Tags: []string{"urgent"}},
		{RelPath: "work/projects/beta.md", Name: "beta.md", TitleText: "Beta", Preview: "planning notes"},
		{RelPath: "work/meeting.md", Name: "meeting.md", TitleText: "Meeting", Preview: "agenda"},
		{RelPath: "personal/journal.md", Name: "journal.md", TitleText: "Journal", Preview: "today"},
		{RelPath: "inbox.md", Name: "inbox.md", TitleText: "Inbox", Preview: "capture"},
		{RelPath: "secret.md", Name: "secret.md", TitleText: "Secret", Preview: "", Encrypted: true},
	}
	cats := []notes.Category{
		{Name: "work", RelPath: "work"},
		{Name: "projects", RelPath: "work/projects"},
		{Name: "personal", RelPath: "personal"},
	}
	remote := []notesync.RemoteNoteMeta{
		{ID: "r1", RelPath: "work/remote-only.md", Title: "Remote Plan"},
	}
	queries := []string{"", "alpha", "plan", "work", "#urgent", "#missing", "beta plan", "zzz", "encrypted", "meet"}

	for _, q := range queries {
		m := newTestModel(t)
		m.notes = ns
		m.categories = cats
		m.remoteOnlyNotes = remote
		m.expanded = map[string]bool{}
		m.searchInput.SetValue(q)
		m.rebuildTree()

		ql := strings.TrimSpace(strings.ToLower(q))
		want := summarizeTree(referenceTree(&m, ql))
		got := summarizeTree(m.treeItems)
		require.Equalf(t, want, got, "tree mismatch for query %q", q)
	}
}

func treeHasNote(items []treeItem, relPath string) bool {
	for _, it := range items {
		if it.Kind == treeNote && it.RelPath == relPath {
			return true
		}
	}
	return false
}

func TestRebuildTreeReflectsNoteContentChange(t *testing.T) {
	m := newTestModel(t)
	m.expanded = map[string]bool{}
	t1 := time.Unix(1000, 0)
	m.notes = []notes.Note{{RelPath: "a.md", Name: "a.md", TitleText: "Apple", ModTime: t1}}

	m.searchInput.SetValue("banana")
	m.rebuildTree()
	require.False(t, treeHasNote(m.treeItems, "a.md"), "apple note should not match banana")

	// Simulate the note being edited: same path, new title, newer modtime. The
	// cached search doc must refresh so search reflects the new content.
	m.notes = []notes.Note{{RelPath: "a.md", Name: "a.md", TitleText: "Banana", ModTime: time.Unix(2000, 0)}}
	m.rebuildTree()
	require.True(t, treeHasNote(m.treeItems, "a.md"), "edited note should match its new title")
}

func largeVaultModel() Model {
	m := Model{expanded: map[string]bool{}}
	var ns []notes.Note
	var cats []notes.Category
	for c := 0; c < 200; c++ {
		cat := fmt.Sprintf("cat%03d", c)
		cats = append(cats, notes.Category{Name: cat, RelPath: cat})
		for i := 0; i < 20; i++ {
			ns = append(ns, notes.Note{
				RelPath:   fmt.Sprintf("%s/note%03d.md", cat, i),
				Name:      fmt.Sprintf("note%03d.md", i),
				TitleText: fmt.Sprintf("Note %d %d", c, i),
				Preview:   "some body text about topics and other things worth indexing",
			})
		}
	}
	m.notes = ns
	m.categories = cats
	return m
}

func BenchmarkRebuildTreeLargeVault(b *testing.B) {
	m := largeVaultModel()
	m.searchInput.SetValue("beta") // selective query, matches nothing

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.rebuildTree()
	}
}

// BenchmarkRebuildTreeReferenceLargeVault exercises the pre-optimization
// per-category rescan logic on the same vault, for a before/after comparison.
func BenchmarkRebuildTreeReferenceLargeVault(b *testing.B) {
	m := largeVaultModel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = referenceTree(&m, "beta")
	}
}
