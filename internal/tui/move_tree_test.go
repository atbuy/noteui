package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/notes"
)

func TestPinPathHelpers(t *testing.T) {
	if !hasCategoryPrefix("work/projects", "work") {
		require.FailNow(t, "expected category prefix match")
	}
	if hasCategoryPrefix("workbench", "work") {
		require.FailNow(t, "expected prefix helper not to match partial segment")
	}
	if got := rewriteCategoryPrefix("work/projects", "work", "archive/work"); got != "archive/work/projects" {
		require.Failf(t, "assertion failed", "unexpected rewritten category path: %q", got)
	}
	if got := tempPinnedKey("daily/today.md"); got != ".tmp/daily/today.md" {
		require.Failf(t, "assertion failed", "unexpected temp pinned key: %q", got)
	}
	if rel, ok := tempRelFromPinnedKey(" .tmp/daily/today.md "); !ok || rel != "daily/today.md" {
		require.Failf(t, "assertion failed", "unexpected temp rel decode: rel=%q ok=%v", rel, ok)
	}
	if _, ok := tempRelFromPinnedKey(".tmp"); ok {
		require.FailNow(t, "expected bare .tmp key to be rejected")
	}
}

func TestDirectChildSortingAndMatching(t *testing.T) {
	now := time.Now()
	m := Model{
		notes: []notes.Note{
			{RelPath: "work/b.md", Name: "b.md", TitleText: "Beta", Preview: "contains roadmap", Tags: []string{"roadmap"}, ModTime: now.Add(-time.Hour)},
			{RelPath: "work/a.md", Name: "a.md", TitleText: "Alpha", Preview: "encrypted body", Encrypted: true, ModTime: now},
			{RelPath: "top.md", Name: "top.md", TitleText: "Top", Preview: "preview", ModTime: now},
		},
		categories: []notes.Category{
			{Name: "work", RelPath: "work"},
			{Name: "archive", RelPath: "archive"},
			{Name: "work-sub", RelPath: "work/sub"},
		},
		pinnedNotes: map[string]bool{"work/b.md": true},
		pinnedCats:  map[string]bool{"work": true},
	}

	children := m.directChildNotes("work")
	if len(children) != 2 || children[0].RelPath != "work/b.md" {
		require.Failf(t, "assertion failed", "expected pinned note first in child notes, got %#v", children)
	}

	m.sortByModTime = true
	children = m.directChildNotes("work")
	if children[0].RelPath != "work/b.md" || children[1].RelPath != "work/a.md" {
		require.Failf(t, "assertion failed", "expected pinned note first and remaining notes by mod time, got %#v", children)
	}

	cats := m.directChildCategories("")
	if len(cats) != 2 || cats[0].RelPath != "work" {
		require.Failf(t, "assertion failed", "expected pinned category first at root, got %#v", cats)
	}

	if !m.noteMatches(children[0], "road map") {
		require.FailNow(t, "expected multi-term note search to match preview text")
	}
	if !m.noteMatches(children[0], "#road") {
		require.FailNow(t, "expected tag search to match tags")
	}
	if !m.noteMatches(notes.Note{RelPath: "secret.md", Name: "secret.md", Preview: "private text", Encrypted: true}, "encrypted") {
		require.FailNow(t, "expected encrypted notes to match encrypted placeholder")
	}
	if m.noteMatches(children[0], "missing") {
		require.FailNow(t, "expected non-matching query to fail")
	}

	if !m.categoryMatches(notes.Category{Name: "Work", RelPath: "work"}, "work") {
		require.FailNow(t, "expected category match")
	}
	if !m.categorySubtreeMatches("work", "road") {
		require.FailNow(t, "expected subtree match from child note")
	}
}

func TestMoveSelectionHelpers(t *testing.T) {
	m := Model{
		categories: []notes.Category{{Name: "work", RelPath: "work", Depth: 0}},
		notes:      []notes.Note{{RelPath: "work/today.md", Name: "today.md", TitleText: "Today"}},
		markedTreeItems: map[string]bool{
			"c:work":          true,
			"n:work/today.md": true,
		},
		expanded: map[string]bool{"work": true},
	}

	selection := m.currentMoveSelection()
	if len(selection) != 2 {
		require.Failf(t, "assertion failed", "expected two selected items, got %#v", selection)
	}
	summary := m.moveSelectionSummary()
	if summary.categories != 1 || summary.notes != 1 {
		require.Failf(t, "assertion failed", "unexpected selection summary: %#v", summary)
	}

	m.treeItems = []treeItem{{Kind: treeNote, RelPath: "work/today.md", Name: "Today", Note: &m.notes[0]}}
	m.treeCursor = 0
	m.markedTreeItems = nil
	selection = m.currentMoveSelection()
	if len(selection) != 1 || selection[0].RelPath != "work/today.md" {
		require.Failf(t, "assertion failed", "expected current tree item fallback selection, got %#v", selection)
	}
}

func TestMoveDestinationHelpers(t *testing.T) {
	m := Model{
		categories: []notes.Category{
			{Name: "work", RelPath: "work"},
			{Name: "projects", RelPath: "work/projects"},
			{Name: "archive", RelPath: "archive"},
		},
		expanded: map[string]bool{"work": true},
	}

	items := m.moveDestinationItems()
	if len(items) < 3 || items[0].RelPath != "" {
		require.Failf(t, "assertion failed", "expected root move destination item first, got %#v", items)
	}

	m.setMoveDestinationCursor("work/projects")
	if got := m.currentMoveDestinationPath(); got != "work/projects" {
		require.Failf(t, "assertion failed", "unexpected move destination cursor path: %q", got)
	}

	m.moveDestCursor = 999
	m.clampMoveDestinationCursor()
	if m.currentMoveDestination() == nil {
		require.FailNow(t, "expected clamped destination cursor to point at an item")
	}

	m.jumpMoveDestinationTop()
	if m.moveDestCursor != 0 {
		require.Failf(t, "assertion failed", "expected top jump to reset cursor, got %d", m.moveDestCursor)
	}
	m.jumpMoveDestinationBottom()
	if m.moveDestCursor != len(m.moveDestinationItems())-1 {
		require.Failf(t, "assertion failed", "expected bottom jump to select last item, got %d", m.moveDestCursor)
	}

	selection := []treeItem{{Kind: treeCategory, RelPath: "work/projects", Name: "projects"}}
	if got := m.preferredMoveDestinationPath(selection); got != "work" {
		require.Failf(t, "assertion failed", "unexpected preferred category destination: %q", got)
	}
	selection = []treeItem{{Kind: treeNote, Note: &notes.Note{RelPath: "work/today.md"}}}
	if got := m.preferredMoveDestinationPath(selection); got != "work" {
		require.Failf(t, "assertion failed", "unexpected preferred note destination: %q", got)
	}
}

func TestBuildMoveBatchValidatesSelections(t *testing.T) {
	root := t.TempDir()
	writeTUINote(t, filepath.Join(root, "work", "today.md"), "body")
	writeTUINote(t, filepath.Join(root, "archive", "old.md"), "body")
	if err := os.MkdirAll(filepath.Join(root, "work", "projects"), 0o755); err != nil {
		require.Failf(t, "assertion failed", "MkdirAll returned error: %v", err)
	}

	m := Model{
		rootDir: root,
		categories: []notes.Category{
			{Name: "work", RelPath: "work"},
			{Name: "projects", RelPath: "work/projects"},
			{Name: "archive", RelPath: "archive"},
		},
		notes:           []notes.Note{{RelPath: "work/today.md", Name: "today.md", TitleText: "Today"}},
		markedTreeItems: map[string]bool{"n:work/today.md": true},
	}

	batch, err := m.buildMoveBatch("archive")
	if err != nil {
		require.Failf(t, "assertion failed", "buildMoveBatch returned error: %v", err)
	}
	if len(batch) != 1 || batch[0].newRelPath != "archive/today.md" {
		require.Failf(t, "assertion failed", "unexpected move batch: %#v", batch)
	}

	if _, err := m.buildMoveBatch("../outside"); err == nil {
		require.FailNow(t, "expected outside-root destination to be rejected")
	}

	m.markedTreeItems = map[string]bool{"c:work": true, "c:work/projects": true}
	if _, err := m.buildMoveBatch("archive"); err == nil || !strings.Contains(err.Error(), "descendant") {
		require.Failf(t, "assertion failed", "expected ancestor+descendant category move to be rejected, got %v", err)
	}

	m.markedTreeItems = map[string]bool{"c:work": true, "n:work/today.md": true}
	if _, err := m.buildMoveBatch("archive"); err == nil || !strings.Contains(err.Error(), "inside it") {
		require.Failf(t, "assertion failed", "expected category+nested-note move to be rejected, got %v", err)
	}

	m.markedTreeItems = map[string]bool{"n:work/today.md": true}
	if _, err := m.buildMoveBatch("work"); err == nil || !strings.Contains(err.Error(), "already in that category") {
		require.Failf(t, "assertion failed", "expected no-op destination to be rejected, got %v", err)
	}

	m.notes = append(m.notes, notes.Note{RelPath: "archive/today.md", Name: "today.md", TitleText: "Archive Today"})
	writeTUINote(t, filepath.Join(root, "archive", "today.md"), "body")
	if _, err := m.buildMoveBatch("archive"); err == nil || !strings.Contains(err.Error(), "target already exists") {
		require.Failf(t, "assertion failed", "expected existing-target conflict, got %v", err)
	}
}

func TestRewritePinnedNotePath(t *testing.T) {
	m := Model{pinnedNotes: map[string]bool{"work/today.md": true}}
	m.rewritePinnedNotePath("work/today.md", "archive/today.md")
	if m.pinnedNotes["work/today.md"] || !m.pinnedNotes["archive/today.md"] {
		require.Failf(t, "assertion failed", "expected pinned note path to be rewritten, got %#v", m.pinnedNotes)
	}
}

func writeTUINote(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		require.Failf(t, "assertion failed", "MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}
}
