package tui

import (
	"path/filepath"
	"sort"
	"strings"

	"atbuy/noteui/internal/notes"
	"atbuy/noteui/internal/state"
	notesync "atbuy/noteui/internal/sync"
)

func normalizeCategoryRelPath(relPath string) string {
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	if relPath == "." {
		return ""
	}
	return strings.Trim(relPath, "/")
}

func (m *Model) togglePinCurrent() error {
	if m.listMode == listModePins {
		item := m.currentPinItem()
		if item == nil {
			return nil
		}

		switch item.Kind {
		case pinItemCategory:
			if m.pinnedCats[item.RelPath] {
				delete(m.pinnedCats, item.RelPath)
				m.status = "unpinned category: " + item.Name
			} else {
				m.pinnedCats[item.RelPath] = true
				m.status = "pinned category: " + item.Name
			}

		case pinItemNote:
			if m.pinnedNotes[item.RelPath] {
				delete(m.pinnedNotes, item.RelPath)
				m.status = "unpinned note: " + item.Name
			} else {
				m.pinnedNotes[item.RelPath] = true
				m.status = "pinned note: " + item.Name
			}

		case pinItemTemporaryNote:
			key := tempPinnedKey(item.RelPath)
			if m.pinnedNotes[key] {
				delete(m.pinnedNotes, key)
				m.status = "unpinned temporary note: " + item.Name
			} else {
				m.pinnedNotes[key] = true
				m.status = "pinned temporary note: " + item.Name
			}
		}

		if err := m.saveTreeState(); err != nil {
			return err
		}

		items := m.filteredPinnedItems()
		if len(items) == 0 {
			m.pinsCursor = 0
		} else if m.pinsCursor >= len(items) {
			m.pinsCursor = len(items) - 1
		}

		m.syncSelectedNote()
		return nil
	}

	if m.listMode == listModeTemporary {
		n := m.currentTempNote()
		if n == nil {
			return nil
		}
		pinKey := tempPinnedKey(n.RelPath)
		if m.pinnedNotes[pinKey] {
			delete(m.pinnedNotes, pinKey)
			m.status = "unpinned temporary note: " + n.Title()
		} else {
			m.pinnedNotes[pinKey] = true
			m.status = "pinned temporary note: " + n.Title()
		}
		if err := m.saveTreeState(); err != nil {
			return err
		}
		return nil
	}

	item := m.currentTreeItem()
	if item == nil {
		return nil
	}

	switch item.Kind {
	case treeCategory:
		if item.RelPath == "" {
			m.status = "cannot pin root category"
			return nil
		}
		if m.pinnedCats[item.RelPath] {
			delete(m.pinnedCats, item.RelPath)
			m.status = "unpinned category: " + item.Name
		} else {
			m.pinnedCats[item.RelPath] = true
			m.status = "pinned category: " + item.Name
		}

	case treeNote:
		if item.Note == nil {
			return nil
		}
		if m.pinnedNotes[item.Note.RelPath] {
			delete(m.pinnedNotes, item.Note.RelPath)
			m.status = "unpinned note: " + item.Note.Title()
		} else {
			m.pinnedNotes[item.Note.RelPath] = true
			m.status = "pinned note: " + item.Note.Title()
		}
	case treeRemoteNote:
		m.status = "note is only on the server; press i to import it or I to import all"
		return nil
	}

	if err := m.saveTreeState(); err != nil {
		return err
	}

	m.rebuildTree()
	return nil
}

func (m *Model) syncStateFromPins() {
	m.state.PinnedNotes = m.state.PinnedNotes[:0]
	for p := range m.pinnedNotes {
		m.state.PinnedNotes = append(m.state.PinnedNotes, p)
	}
	sort.Strings(m.state.PinnedNotes)

	m.state.PinnedCategories = m.state.PinnedCategories[:0]
	for p := range m.pinnedCats {
		if normalized := normalizeCategoryRelPath(p); normalized != "" {
			m.state.PinnedCategories = append(m.state.PinnedCategories, normalized)
		}
	}
	sort.Strings(m.state.PinnedCategories)
}

func (m *Model) syncStateFromExpanded() {
	m.state.CollapsedCategories = m.state.CollapsedCategories[:0]

	for relPath, expanded := range m.expanded {
		relPath = normalizeCategoryRelPath(relPath)
		if relPath == "" {
			continue
		}
		if !expanded {
			m.state.CollapsedCategories = append(m.state.CollapsedCategories, relPath)
		}
	}

	sort.Strings(m.state.CollapsedCategories)
}

func (m *Model) syncStateForSave() {
	m.syncStateFromPins()
	m.syncStateFromExpanded()
	m.state.SortByModTime = m.sortByModTime
	m.state.RecentCommands = normalizePaletteRecentCommands(m.state.RecentCommands)
}

func (m *Model) saveLocalState() error {
	m.syncStateForSave()
	return state.Save(m.state)
}

func (m *Model) saveTreeState() error {
	if err := m.saveLocalState(); err != nil {
		return err
	}
	if notesync.HasSyncProfile(m.cfg.Sync) && len(m.notes) > 0 {
		return notesync.SavePinsFromRelPaths(m.rootDir, m.notes, m.state.PinnedNotes, m.state.PinnedCategories)
	}
	return nil
}

func (m *Model) pruneCategoryStateToExisting() {
	existing := make(map[string]bool, len(m.categories)+len(m.remoteCategories))
	for _, source := range [][]notes.Category{m.categories, m.remoteCategories} {
		for _, c := range source {
			if normalized := normalizeCategoryRelPath(c.RelPath); normalized != "" {
				existing[normalized] = true
			}
		}
	}

	for k, expanded := range m.expanded {
		normalized := normalizeCategoryRelPath(k)
		if normalized == "" {
			continue
		}
		if !existing[normalized] {
			delete(m.expanded, k)
			continue
		}
		if normalized != k {
			delete(m.expanded, k)
			m.expanded[normalized] = expanded
		}
	}

	for k := range m.pinnedCats {
		normalized := normalizeCategoryRelPath(k)
		if normalized == "" || !existing[normalized] {
			delete(m.pinnedCats, k)
			continue
		}
		if normalized != k {
			delete(m.pinnedCats, k)
			m.pinnedCats[normalized] = true
		}
	}
}

func (m *Model) removeCategoryStateSubtree(relPath string) {
	if relPath == "" {
		return
	}

	for k := range m.expanded {
		if hasCategoryPrefix(k, relPath) {
			delete(m.expanded, k)
		}
	}

	for k := range m.pinnedCats {
		if hasCategoryPrefix(k, relPath) {
			delete(m.pinnedCats, k)
		}
	}
}

func (m *Model) rewriteCategoryStateSubtree(oldRelPath, newRelPath string) {
	if oldRelPath == "" || newRelPath == "" || oldRelPath == newRelPath {
		return
	}

	newExpanded := make(map[string]bool, len(m.expanded))
	for k, v := range m.expanded {
		if hasCategoryPrefix(k, oldRelPath) {
			k = rewriteCategoryPrefix(k, oldRelPath, newRelPath)
		}
		newExpanded[k] = v
	}
	m.expanded = newExpanded

	newPinnedCats := make(map[string]bool, len(m.pinnedCats))
	for k, v := range m.pinnedCats {
		if hasCategoryPrefix(k, oldRelPath) {
			k = rewriteCategoryPrefix(k, oldRelPath, newRelPath)
		}
		newPinnedCats[k] = v
	}
	m.pinnedCats = newPinnedCats

	newPinnedNotes := make(map[string]bool, len(m.pinnedNotes))
	oldPrefix := oldRelPath + "/"
	newPrefix := newRelPath + "/"

	for k, v := range m.pinnedNotes {
		if after, ok := strings.CutPrefix(k, oldPrefix); ok {
			k = newPrefix + after
		}
		newPinnedNotes[k] = v
	}
	m.pinnedNotes = newPinnedNotes
}

func (m Model) isPinnedCategory(relPath string) bool {
	return m.pinnedCats[relPath]
}

func (m Model) isPinnedNote(relPath string) bool {
	return m.pinnedNotes[relPath]
}

func (m Model) isPinnedTemporaryNote(relPath string) bool {
	return m.pinnedNotes[tempPinnedKey(relPath)]
}

func (m Model) pinnedItems() []pinItem {
	var out []pinItem

	for _, c := range m.categories {
		if c.RelPath == "" {
			continue
		}
		if !m.isPinnedCategory(c.RelPath) {
			continue
		}
		out = append(out, pinItem{
			Kind:    pinItemCategory,
			Name:    c.Name,
			RelPath: c.RelPath,
		})
	}

	for _, n := range m.notes {
		if !m.isPinnedNote(n.RelPath) {
			continue
		}
		out = append(out, pinItem{
			Kind:    pinItemNote,
			Name:    n.Title(),
			RelPath: n.RelPath,
			Path:    n.Path,
			Tags:    n.Tags,
		})
	}

	for _, n := range m.tempNotes {
		if !m.isPinnedTemporaryNote(n.RelPath) {
			continue
		}
		out = append(out, pinItem{
			Kind:    pinItemTemporaryNote,
			Name:    n.Title(),
			RelPath: n.RelPath,
			Path:    n.Path,
			Tags:    n.Tags,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].RelPath < out[j].RelPath
	})

	return out
}

func (m Model) findNoteByRelPath(relPath string) *notes.Note {
	for i := range m.notes {
		if m.notes[i].RelPath == relPath {
			return &m.notes[i]
		}
	}
	return nil
}

func (m Model) findTempNoteByRelPath(relPath string) *notes.Note {
	for i := range m.tempNotes {
		if m.tempNotes[i].RelPath == relPath {
			return &m.tempNotes[i]
		}
	}
	return nil
}

func (m Model) filteredPinnedItems() []pinItem {
	query := strings.TrimSpace(strings.ToLower(m.searchInput.Value()))
	items := m.pinnedItems()
	if query == "" {
		return items
	}

	out := make([]pinItem, 0, len(items))
	for _, item := range items {
		switch item.Kind {
		case pinItemCategory:
			if strings.Contains(strings.ToLower(item.Name), query) ||
				strings.Contains(strings.ToLower(item.RelPath), query) {
				out = append(out, item)
			}
		case pinItemNote:
			if n := m.findNoteByRelPath(item.RelPath); n != nil && m.noteMatches(*n, query) {
				out = append(out, item)
			}
		case pinItemTemporaryNote:
			if n := m.findTempNoteByRelPath(item.RelPath); n != nil && m.noteMatches(*n, query) {
				out = append(out, item)
			}
		}
	}

	return out
}

func (m Model) currentPinItem() *pinItem {
	items := m.filteredPinnedItems()
	if len(items) == 0 || m.pinsCursor < 0 || m.pinsCursor >= len(items) {
		return nil
	}
	item := items[m.pinsCursor]
	return &item
}

func (m *Model) movePinsCursor(delta int) {
	items := m.filteredPinnedItems()
	if len(items) == 0 {
		return
	}
	next := max(m.pinsCursor+delta, 0)
	if next >= len(items) {
		next = len(items) - 1
	}
	m.pinsCursor = next
	m.syncSelectedNote()
}

func hasCategoryPrefix(path, prefix string) bool {
	if prefix == "" {
		return false
	}
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func rewriteCategoryPrefix(path, oldPrefix, newPrefix string) string {
	if path == oldPrefix {
		return newPrefix
	}
	if strings.HasPrefix(path, oldPrefix+"/") {
		return newPrefix + strings.TrimPrefix(path, oldPrefix)
	}
	return path
}

func tempPinnedKey(relPath string) string {
	return ".tmp/" + filepath.ToSlash(relPath)
}

func tempRelFromPinnedKey(key string) (string, bool) {
	key = filepath.ToSlash(strings.TrimSpace(key))
	if key == ".tmp" {
		return "", false
	}
	if after, ok := strings.CutPrefix(key, ".tmp/"); ok {
		return after, true
	}
	return "", false
}
