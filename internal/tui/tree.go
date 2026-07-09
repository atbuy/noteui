package tui

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/editor"
	"atbuy/noteui/internal/notes"
	notesync "atbuy/noteui/internal/sync"
)

func (m *Model) rebuildTree() {
	var selectedKey string
	if current := m.currentTreeItem(); current != nil {
		selectedKey = current.key()
	}

	var out []treeItem
	ctx := m.buildTreeContext()
	m.buildTree("", 0, &ctx, &out)
	m.treeItems = out

	if len(m.treeItems) == 0 {
		m.treeCursor = 0
		m.selected = nil
		m.preserveCursor = -1
		return
	}

	restore := -1

	if selectedKey != "" {
		for i, item := range m.treeItems {
			if item.key() == selectedKey {
				restore = i
				break
			}
		}
	}

	if restore == -1 && m.preserveCursor >= 0 {
		restore = m.preserveCursor
		if restore >= len(m.treeItems) {
			restore = len(m.treeItems) - 1
		}
	}

	if restore == -1 {
		restore = 0
	}

	m.treeCursor = restore
	m.preserveCursor = -1
	m.syncSelectedNote()
}

// treeBuildContext holds the query-dependent work computed once per tree
// rebuild: the parent -> children groupings (so the tree is assembled without
// rescanning every note per category), the matched note scores for display,
// and the set of categories that must be shown because they or something in
// their subtree matches. This turns a rebuild from O(categories x notes) into a
// single pass over the collection.
type treeBuildContext struct {
	query          string
	notesByParent  map[string][]int
	remoteByParent map[string][]int
	catsByParent   map[string][]notes.Category
	noteScoreByRel map[string]int
	visibleCats    map[string]bool
}

func (m *Model) buildTreeContext() treeBuildContext {
	query := strings.TrimSpace(strings.ToLower(m.searchInput.Value()))
	ctx := treeBuildContext{
		query:          query,
		notesByParent:  make(map[string][]int),
		remoteByParent: make(map[string][]int),
		catsByParent:   make(map[string][]notes.Category),
	}

	for i := range m.notes {
		parent := parentKey(m.notes[i].RelPath)
		ctx.notesByParent[parent] = append(ctx.notesByParent[parent], i)
	}
	for i := range m.remoteOnlyNotes {
		parent := parentKey(m.remoteOnlyNotes[i].RelPath)
		ctx.remoteByParent[parent] = append(ctx.remoteByParent[parent], i)
	}
	seen := make(map[string]bool)
	for _, source := range [][]notes.Category{m.categories, m.remoteCategories} {
		for _, c := range source {
			if c.RelPath == "" || seen[c.RelPath] {
				continue
			}
			seen[c.RelPath] = true
			parent := parentKey(c.RelPath)
			ctx.catsByParent[parent] = append(ctx.catsByParent[parent], c)
		}
	}

	if query == "" {
		return ctx
	}

	ctx.noteScoreByRel = make(map[string]int)
	ctx.visibleCats = make(map[string]bool)

	for i := range m.notes {
		doc := m.docFor(m.notes[i])
		// Display eligibility mirrors filterAndScoreNotes (no tag prefix).
		if score := noteScoreDoc(doc, query); score >= 0 {
			ctx.noteScoreByRel[m.notes[i].RelPath] = score
		}
		// Subtree visibility mirrors categorySubtreeMatches, which uses
		// noteMatches and therefore honors the "#tag" prefix.
		if noteMatchesDoc(doc, query) {
			markAncestors(ctx.visibleCats, parentKey(m.notes[i].RelPath))
		}
	}
	for i := range m.remoteOnlyNotes {
		if m.remoteNoteMatches(m.remoteOnlyNotes[i], query) {
			markAncestors(ctx.visibleCats, parentKey(m.remoteOnlyNotes[i].RelPath))
		}
	}
	for _, cats := range ctx.catsByParent {
		for _, c := range cats {
			if m.categoryMatches(c, query) {
				markAncestors(ctx.visibleCats, parentKey(c.RelPath))
			}
		}
	}

	return ctx
}

// markAncestors marks dir and each of its parent directories as visible, so a
// matching note or category surfaces every category on the path to the root.
func markAncestors(visible map[string]bool, dir string) {
	for dir != "" && dir != "." {
		visible[dir] = true
		parent := filepath.Dir(dir)
		if parent == "." {
			parent = ""
		}
		dir = parent
	}
}

// parentKey returns the parent directory of a relPath in the same normalized
// form the tree groups by ("" for the root).
func parentKey(relPath string) string {
	dir := filepath.Dir(relPath)
	if dir == "." {
		return ""
	}
	return dir
}

func (m *Model) buildTree(parent string, depth int, ctx *treeBuildContext, out *[]treeItem) {
	query := ctx.query

	effectiveExpanded := func(rel string) bool {
		if rel == "" {
			return true
		}
		if query != "" {
			return true
		}
		return m.expanded[rel]
	}

	// Add a synthetic root item at the top of the tree.
	if parent == "" && depth == 0 {
		*out = append(*out, treeItem{
			Kind:     treeCategory,
			Name:     "/",
			RelPath:  "",
			Depth:    0,
			Expanded: true,
			Category: nil,
		})

		// Root should show its direct children one level below it.
		depth = 1
	}

	for _, cat := range m.sortedChildCategories(ctx.catsByParent[parent]) {
		include := query == "" || m.categoryMatches(cat, query) || ctx.visibleCats[cat.RelPath]
		if !include {
			continue
		}

		item := treeItem{
			Kind:     treeCategory,
			Name:     cat.Name,
			RelPath:  cat.RelPath,
			Depth:    depth,
			Expanded: effectiveExpanded(cat.RelPath),
			Category: &cat,
		}
		*out = append(*out, item)

		if item.Expanded {
			m.buildTree(cat.RelPath, depth+1, ctx, out)
		}
	}

	childNotes := m.sortedChildNotes(ctx.notesByParent[parent])
	if query != "" {
		childNotes = filterByPrecomputedScore(childNotes, ctx.noteScoreByRel)
	}
	for _, n := range childNotes {
		noteCopy := n
		hint := ""
		if query != "" {
			hint = findMatchExcerpt(noteCopy, query)
		}
		*out = append(*out, treeItem{
			Kind:      treeNote,
			Name:      n.Title(),
			RelPath:   n.RelPath,
			Depth:     depth,
			Note:      &noteCopy,
			MatchHint: hint,
		})
	}

	for _, n := range m.sortedChildRemoteNotes(ctx.remoteByParent[parent]) {
		if query != "" && !m.remoteNoteMatches(n, query) {
			continue
		}
		remoteCopy := n
		*out = append(*out, treeItem{
			Kind:       treeRemoteNote,
			Name:       m.remoteOnlyDisplayTitle(n),
			RelPath:    n.RelPath,
			Depth:      depth,
			RemoteNote: &remoteCopy,
		})
	}
}

// filterByPrecomputedScore keeps the notes whose relpath scored >= 0 (present
// in scores) and orders them by descending score, matching
// filterAndScoreNotes. The stable sort preserves the incoming sort order for
// ties.
func filterByPrecomputedScore(ns []notes.Note, scores map[string]int) []notes.Note {
	type scored struct {
		note  notes.Note
		score int
	}
	matched := make([]scored, 0, len(ns))
	for _, n := range ns {
		if score, ok := scores[n.RelPath]; ok {
			matched = append(matched, scored{n, score})
		}
	}
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].score > matched[j].score
	})
	out := make([]notes.Note, len(matched))
	for i, sm := range matched {
		out[i] = sm.note
	}
	return out
}

func (m Model) directChildNotes(parent string) []notes.Note {
	out := make([]notes.Note, 0)
	for _, n := range m.notes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == parent {
			out = append(out, n)
		}
	}
	sortNotes(out, m.sortMethod, m.sortReverse, m.isPinnedNote)
	return out
}

func sortNotes(out []notes.Note, method string, reverse bool, isPinned func(string) bool) {
	sort.SliceStable(out, func(i, j int) bool {
		if isPinned != nil {
			pi := isPinned(out[i].RelPath)
			pj := isPinned(out[j].RelPath)
			if pi != pj {
				return pi
			}
		}
		var less bool
		switch method {
		case sortModified:
			less = out[i].ModTime.After(out[j].ModTime)
		case sortCreated:
			less = out[i].CreatedAt.After(out[j].CreatedAt)
		case sortSize:
			less = out[i].Size > out[j].Size
		default:
			less = out[i].RelPath < out[j].RelPath
		}
		if reverse {
			return !less
		}
		return less
	})
}

func (m Model) directChildRemoteNotes(parent string) []notesync.RemoteNoteMeta {
	out := make([]notesync.RemoteNoteMeta, 0)
	for _, n := range m.remoteOnlyNotes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == parent {
			out = append(out, n)
		}
	}
	sortRemoteNotes(out)
	return out
}

// sortedChildNotes gathers the notes at the given indices into m.notes and
// sorts them exactly like directChildNotes. It is the hot-path equivalent used
// with the precomputed parent groupings.
func (m Model) sortedChildNotes(idxs []int) []notes.Note {
	out := make([]notes.Note, 0, len(idxs))
	for _, i := range idxs {
		out = append(out, m.notes[i])
	}
	sortNotes(out, m.sortMethod, m.sortReverse, m.isPinnedNote)
	return out
}

func (m Model) sortedChildRemoteNotes(idxs []int) []notesync.RemoteNoteMeta {
	out := make([]notesync.RemoteNoteMeta, 0, len(idxs))
	for _, i := range idxs {
		out = append(out, m.remoteOnlyNotes[i])
	}
	sortRemoteNotes(out)
	return out
}

func (m Model) sortedChildCategories(cats []notes.Category) []notes.Category {
	out := make([]notes.Category, len(cats))
	copy(out, cats)
	sortCategories(out, m.isPinnedCategory)
	return out
}

func sortRemoteNotes(out []notesync.RemoteNoteMeta) {
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RelPath == out[j].RelPath {
			return out[i].ID < out[j].ID
		}
		return out[i].RelPath < out[j].RelPath
	})
}

func sortCategories(out []notes.Category, isPinned func(string) bool) {
	sort.SliceStable(out, func(i, j int) bool {
		pi := isPinned(out[i].RelPath)
		pj := isPinned(out[j].RelPath)
		if pi != pj {
			return pi
		}
		return out[i].RelPath < out[j].RelPath
	})
}

func (m Model) remoteNoteMatches(n notesync.RemoteNoteMeta, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	for term := range strings.FieldsSeq(q) {
		if !strings.Contains(strings.ToLower(remoteOnlyNoteTitle(n)), term) && !strings.Contains(strings.ToLower(n.RelPath), term) {
			return false
		}
	}
	return true
}

func (m Model) directChildCategories(parent string) []notes.Category {
	out := make([]notes.Category, 0)
	seen := make(map[string]bool)
	for _, source := range [][]notes.Category{m.categories, m.remoteCategories} {
		for _, c := range source {
			if c.RelPath == "" || seen[c.RelPath] {
				continue
			}
			dir := filepath.Dir(c.RelPath)
			if dir == "." {
				dir = ""
			}
			if dir == parent {
				out = append(out, c)
				seen[c.RelPath] = true
			}
		}
	}

	sortCategories(out, m.isPinnedCategory)

	return out
}

// fuzzySequenceMatch reports whether every rune of pattern appears in target
// in order (subsequence match). Both strings must already be lower-cased by
// the caller.
func fuzzySequenceMatch(pattern, target string) bool {
	if pattern == "" {
		return true
	}
	pi := 0
	pr := []rune(pattern)
	for _, r := range target {
		if r == pr[pi] {
			pi++
			if pi == len(pr) {
				return true
			}
		}
	}
	return false
}

// noteSearchDoc holds a note's search fields pre-lowercased, so scoring a note
// against a query does not repeat the same strings.ToLower work on every term,
// every keystroke, and every ancestor category.
type noteSearchDoc struct {
	title string
	name  string
	rel   string
	body  string
	tags  []string
}

type cachedNoteDoc struct {
	doc     noteSearchDoc
	modTime time.Time
}

func buildNoteDoc(n notes.Note) noteSearchDoc {
	// Encrypted notes keep the "<encrypted>" placeholder as their body so the
	// literal word "encrypted" still matches; this mirrors the previous scorer.
	body := "<encrypted>"
	if !n.Encrypted {
		body = strings.ToLower(notes.StripFrontMatter(n.Preview))
	}
	var tags []string
	if len(n.Tags) > 0 {
		tags = make([]string, len(n.Tags))
		for i, t := range n.Tags {
			tags[i] = strings.ToLower(t)
		}
	}
	return noteSearchDoc{
		title: strings.ToLower(n.Title()),
		name:  strings.ToLower(n.Name),
		rel:   strings.ToLower(n.RelPath),
		body:  body,
		tags:  tags,
	}
}

// docFor returns the cached search doc for a note, rebuilding it only when the
// note is new or its modification time changed. The cache persists across
// keystrokes and is refreshed lazily whenever the notes collection changes.
func (m *Model) docFor(n notes.Note) noteSearchDoc {
	if m.docCache == nil {
		m.docCache = make(map[string]cachedNoteDoc)
	} else if len(m.docCache) > 2*len(m.notes)+16 {
		// Bound growth from renames/deletions leaving stale entries behind.
		m.docCache = make(map[string]cachedNoteDoc, len(m.notes))
	}
	if cached, ok := m.docCache[n.RelPath]; ok && cached.modTime.Equal(n.ModTime) {
		return cached.doc
	}
	doc := buildNoteDoc(n)
	m.docCache[n.RelPath] = cachedNoteDoc{doc: doc, modTime: n.ModTime}
	return doc
}

// scoreTermDoc returns a relevance score (>= 0) for a single lower-cased search
// term against a pre-lowercased doc, or -1 when the term does not match.
// Higher scores indicate a better match.
func scoreTermDoc(term string, d noteSearchDoc) int {
	if strings.Contains(d.title, term) {
		return 1000
	}
	if strings.Contains(d.name, term) {
		return 800
	}
	if strings.Contains(d.rel, term) {
		return 600
	}
	for _, t := range d.tags {
		if strings.Contains(t, term) {
			return 500
		}
	}
	if strings.Contains(d.body, term) {
		return 400
	}
	if fuzzySequenceMatch(term, d.title) {
		return 200
	}
	if fuzzySequenceMatch(term, d.rel) {
		return 100
	}
	return -1
}

// noteScoreDoc returns the total relevance score for a doc against a query, or
// -1 if any term does not match.
func noteScoreDoc(d noteSearchDoc, query string) int {
	total := 0
	for term := range strings.FieldsSeq(query) {
		s := scoreTermDoc(term, d)
		if s < 0 {
			return -1
		}
		total += s
	}
	return total
}

// noteMatchesDoc reports whether a doc matches the query, honoring the "#tag"
// prefix that restricts matching to tags.
func noteMatchesDoc(d noteSearchDoc, query string) bool {
	if query == "" {
		return true
	}
	if after, ok := strings.CutPrefix(query, "#"); ok {
		if after == "" {
			return true
		}
		for _, t := range d.tags {
			if strings.Contains(t, after) {
				return true
			}
		}
		return false
	}
	return noteScoreDoc(d, query) >= 0
}

// filterAndScoreNotes returns the subset of ns that match query, sorted by
// descending relevance score. When query is empty the original slice is
// returned unchanged.
func filterAndScoreNotes(ns []notes.Note, query string) []notes.Note {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return ns
	}
	type scored struct {
		note  notes.Note
		score int
	}
	matched := make([]scored, 0, len(ns))
	for _, n := range ns {
		if s := noteScoreDoc(buildNoteDoc(n), q); s >= 0 {
			matched = append(matched, scored{n, s})
		}
	}
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].score > matched[j].score
	})
	out := make([]notes.Note, len(matched))
	for i, sm := range matched {
		out[i] = sm.note
	}
	return out
}

func (m Model) noteMatches(n notes.Note, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	return noteMatchesDoc(buildNoteDoc(n), q)
}

func (m Model) categoryMatches(c notes.Category, query string) bool {
	return strings.Contains(strings.ToLower(c.Name), query) ||
		strings.Contains(strings.ToLower(c.RelPath), query)
}

func (m Model) categorySubtreeMatches(relPath, query string) bool {
	prefix := relPath + string(filepath.Separator)

	for _, source := range [][]notes.Category{m.categories, m.remoteCategories} {
		for _, c := range source {
			if c.RelPath != relPath && strings.HasPrefix(c.RelPath, prefix) && m.categoryMatches(c, query) {
				return true
			}
		}
	}
	for _, n := range m.notes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == relPath || strings.HasPrefix(dir, prefix) {
			if m.noteMatches(n, query) {
				return true
			}
		}
	}
	for _, n := range m.remoteOnlyNotes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == relPath || strings.HasPrefix(dir, prefix) {
			if m.remoteNoteMatches(n, query) {
				return true
			}
		}
	}
	return false
}

func (m *Model) moveTreeCursor(delta int) {
	if len(m.treeItems) == 0 {
		return
	}
	next := max(m.treeCursor+delta, 0)
	if next >= len(m.treeItems) {
		next = len(m.treeItems) - 1
	}
	m.treeCursor = next
	m.syncSelectedNote()
}

func (m *Model) syncSelectedNote() {
	switch m.listMode {
	case listModeTodos:
		item := m.currentTodoItem()
		if item == nil {
			m.selected = nil
			m.refreshPreview()
			return
		}
		noteCopy := item.Note
		m.selected = &noteCopy
		m.refreshPreview()
		return

	case listModeTemporary:
		n := m.currentTempNote()
		if n == nil {
			m.selected = nil
			m.refreshPreview()
			return
		}
		m.selected = n
		m.refreshPreview()
		return

	case listModePins:
		item := m.currentPinItem()
		if item == nil {
			m.selected = nil
			m.refreshPreview()
			return
		}

		switch item.Kind {
		case pinItemNote:
			for _, n := range m.notes {
				if n.RelPath == item.RelPath {
					noteCopy := n
					m.selected = &noteCopy
					m.refreshPreview()
					return
				}
			}
		case pinItemTemporaryNote:
			for _, n := range m.tempNotes {
				if n.RelPath == item.RelPath {
					noteCopy := n
					m.selected = &noteCopy
					m.refreshPreview()
					return
				}
			}
		}

		m.selected = nil
		m.refreshPreview()
		return

	default:
		item := m.currentTreeItem()
		if item == nil || item.Kind != treeNote || item.Note == nil {
			m.selected = nil
			m.refreshPreview()
			return
		}
		m.selected = item.Note
		m.refreshPreview()
	}
}

func (m Model) currentTreeItem() *treeItem {
	if len(m.treeItems) == 0 || m.treeCursor < 0 || m.treeCursor >= len(m.treeItems) {
		return nil
	}
	item := m.treeItems[m.treeCursor]
	return &item
}

func (m *Model) activateCurrentItem() tea.Cmd {
	if m.listMode == listModeTodos {
		item := m.currentTodoItem()
		if item == nil {
			return nil
		}
		previewTodos := notes.ExtractTodoItems(item.Note.Preview, false)
		m.pendingTodoCursor = -1
		for i, todo := range previewTodos {
			if todo.Line == item.Todo.Line {
				m.pendingTodoCursor = i
				break
			}
		}
		m.previewTodoNavMode = m.pendingTodoCursor >= 0
		if item.IsTemp {
			m.switchToTemporaryMode()
			m.selectTemporaryNote(item.RelPath)
			m.status = "jumped to todo note"
			return nil
		}
		m.switchToNotesMode()
		m.selectTreeNote(item.RelPath)
		m.status = "jumped to todo note"
		return nil
	}

	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil {
			return nil
		}

		switch item.Kind {
		case pinItemCategory:
			m.switchToNotesMode()
			m.selectTreeCategory(item.RelPath)
			m.status = "jumped to pinned category"
			return nil

		case pinItemNote:
			m.switchToNotesMode()
			m.selectTreeNote(item.RelPath)
			m.status = "jumped to pinned note"
			return nil

		case pinItemTemporaryNote:
			m.switchToTemporaryMode()
			m.selectTemporaryNote(item.RelPath)
			m.status = "jumped to pinned temporary note"
			return nil
		}
	}

	if m.listMode == listModeTemporary {
		n := m.currentTempNote()
		if n == nil {
			return nil
		}
		if n.Encrypted {
			m.status = "opening encrypted note: " + n.RelPath
			return m.armOpenEncrypted(n.Path)
		}
		m.status = "opening in nvim: " + n.RelPath
		return editor.Open(n.Path)
	}

	item := m.currentTreeItem()
	if item == nil {
		return nil
	}

	if item.Kind == treeCategory {
		m.toggleCurrentCategory()
		return nil
	}
	if item.Kind == treeRemoteNote {
		m.status = "note is only on the server; press i to import it or I to import all"
		return nil
	}

	if item.Note != nil {
		if item.Note.Encrypted {
			m.status = "opening encrypted note: " + item.Note.RelPath
			return m.armOpenEncrypted(item.Note.Path)
		}
		m.status = "opening in nvim: " + item.Note.RelPath
		return editor.Open(item.Note.Path)
	}

	return nil
}

func (m *Model) selectTreeCategory(relPath string) {
	m.rebuildTree()
	for i, item := range m.treeItems {
		if item.Kind == treeCategory && item.RelPath == relPath {
			m.treeCursor = i
			m.syncSelectedNote()
			return
		}
	}
}

func (m *Model) selectTreeNote(relPath string) {
	m.rebuildTree()
	for i, item := range m.treeItems {
		if (item.Kind == treeNote || item.Kind == treeRemoteNote) && item.RelPath == relPath {
			m.treeCursor = i
			m.syncSelectedNote()
			return
		}
	}
}

func (m *Model) selectTemporaryNote(relPath string) {
	items := m.filteredTempNotes()
	for i, n := range items {
		if n.RelPath == relPath {
			m.tempCursor = i
			m.syncSelectedNote()
			return
		}
	}
}

func (m *Model) toggleCurrentCategory() {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeCategory {
		return
	}
	if !m.categoryHasChildren(item.RelPath) {
		m.status = "category: " + item.Name
		return
	}

	m.expanded[item.RelPath] = !m.expanded[item.RelPath]
	if m.expanded[item.RelPath] {
		m.status = "expanded: " + item.Name
	} else {
		m.status = "collapsed: " + item.Name
	}

	_ = m.saveTreeState()
	m.rebuildTree()
}

func (m *Model) expandCurrentCategory() {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeCategory {
		return
	}
	if m.categoryHasChildren(item.RelPath) {
		m.expanded[item.RelPath] = true
		m.status = "expanded: " + item.Name
		_ = m.saveTreeState()
		m.rebuildTree()
	}
}

func (m *Model) collapseCurrentCategory() {
	item := m.currentTreeItem()
	if item == nil || item.Kind != treeCategory {
		return
	}

	if m.categoryHasChildren(item.RelPath) && m.expanded[item.RelPath] {
		m.expanded[item.RelPath] = false
		m.status = "collapsed: " + item.Name
		_ = m.saveTreeState()
		m.rebuildTree()
		return
	}

	parent := filepath.Dir(item.RelPath)
	if parent == "." {
		parent = ""
	}
	for i, t := range m.treeItems {
		if t.Kind == treeCategory && t.RelPath == parent {
			m.treeCursor = i
			m.syncSelectedNote()
			return
		}
	}
}

func (m Model) categoryHasChildren(relPath string) bool {
	if relPath == "" {
		return len(m.directChildCategories("")) > 0 || len(m.directChildNotes("")) > 0 || len(m.directChildRemoteNotes("")) > 0
	}
	prefix := relPath + string(filepath.Separator)
	for _, source := range [][]notes.Category{m.categories, m.remoteCategories} {
		for _, c := range source {
			if c.RelPath != relPath && strings.HasPrefix(c.RelPath, prefix) {
				return true
			}
		}
	}
	for _, n := range m.notes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == relPath {
			return true
		}
	}
	for _, n := range m.remoteOnlyNotes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == relPath {
			return true
		}
	}
	return false
}

func (m Model) currentTargetDir() string {
	if m.listMode == listModeTodos {
		item := m.currentTodoItem()
		if item == nil {
			return ""
		}
		dir := filepath.Dir(item.RelPath)
		if dir == "." {
			return ""
		}
		return dir
	}

	item := m.currentTreeItem()
	if item == nil {
		return ""
	}

	switch item.Kind {
	case treeCategory:
		return item.RelPath
	case treeNote:
		if item.Note == nil {
			return ""
		}
		dir := filepath.Dir(item.Note.RelPath)
		if dir == "." {
			return ""
		}
		return dir
	case treeRemoteNote:
		dir := filepath.Dir(item.RelPath)
		if dir == "." {
			return ""
		}
		return dir
	default:
		return ""
	}
}

func (m Model) countNotesUnder(relPath string) int {
	if relPath == "" {
		return len(m.notes) + len(m.remoteOnlyNotes)
	}
	prefix := relPath + string(filepath.Separator)
	count := 0
	for _, n := range m.notes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == relPath || strings.HasPrefix(dir, prefix) {
			count++
		}
	}
	for _, n := range m.remoteOnlyNotes {
		dir := filepath.Dir(n.RelPath)
		if dir == "." {
			dir = ""
		}
		if dir == relPath || strings.HasPrefix(dir, prefix) {
			count++
		}
	}
	return count
}

func (m Model) countChildCategories(relPath string) int {
	count := 0
	seen := make(map[string]bool)
	for _, source := range [][]notes.Category{m.categories, m.remoteCategories} {
		for _, c := range source {
			if c.RelPath == "" {
				continue
			}
			dir := filepath.Dir(c.RelPath)
			if dir == "." {
				dir = ""
			}
			if dir == relPath && !seen[c.RelPath] {
				seen[c.RelPath] = true
				count++
			}
		}
	}
	return count
}

func (m Model) treeInnerWidth() int {
	leftWidth, _ := m.panelWidths()
	// Panel inner = max(20, leftWidth-2) - 2*panelPaddingX; tree items use
	// Padding(0,1) internally so subtract 2 more for their own side padding.
	innerWidth := max(20, leftWidth-2) - 2*panelPaddingX
	return max(16, innerWidth-2)
}

func (m Model) panelWidths() (int, int) {
	usableWidth := max(40, m.width-6)

	leftWidth := int(float64(usableWidth) * treePaneRatio)
	leftWidth = max(minTreeWidth, leftWidth)

	rightWidth := usableWidth - leftWidth - panelGapWidth
	rightWidth = max(minPreviewWidth, rightWidth)

	// Rebalance if minimums pushed things too far.
	if leftWidth+rightWidth+panelGapWidth > usableWidth {
		leftWidth = max(minTreeWidth, usableWidth-rightWidth-panelGapWidth)
	}

	return leftWidth, rightWidth
}

func trimOrPad(s string, width int) string {
	w := lipgloss.Width(s)
	if w == width {
		return s
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}

	// Simple trim for now.
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	cur := 0
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if cur+rw > width {
			break
		}
		out = append(out, r)
		cur += rw
	}
	if cur < width {
		out = append(out, []rune(strings.Repeat(" ", width-cur))...)
	}
	return string(out)
}

func findMatchExcerpt(n notes.Note, query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return ""
	}

	// Tag search.
	if after, ok := strings.CutPrefix(q, "#"); ok {
		tag := after
		if tag == "" {
			return ""
		}
		for _, t := range n.Tags {
			if strings.Contains(strings.ToLower(t), tag) {
				return "tag:" + t
			}
		}
		return ""
	}

	if n.Encrypted {
		return "<encrypted>"
	}

	terms := strings.Fields(q)
	content := notes.StripFrontMatter(n.Preview)
	lines := strings.Split(content, "\n")

	lastSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Track nearest section heading for context
		if strings.HasPrefix(trimmed, "# ") {
			lastSection = ""
			continue
		}
		if after, ok := strings.CutPrefix(trimmed, "### "); ok {
			lastSection = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(trimmed, "## "); ok {
			lastSection = strings.TrimSpace(after)
		}

		lower := strings.ToLower(trimmed)
		matched := false
		for _, term := range terms {
			if strings.Contains(lower, term) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		// Strip leading # markers for display
		displayLine := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))

		if lastSection != "" && displayLine != lastSection {
			return lastSection + " › " + displayLine
		}
		return displayLine
	}

	return ""
}
