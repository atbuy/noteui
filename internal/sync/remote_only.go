package sync

import (
	"path/filepath"
	"sort"
	"strings"

	"atbuy/noteui/internal/notes"
)

func remoteOnlyNotesForRoot(root string, remote []RemoteNoteMeta, records map[string]NoteRecord) ([]RemoteNoteMeta, error) {
	allNotes, err := notes.Discover(root)
	if err != nil {
		return nil, err
	}
	localByRelPath := make(map[string]bool, len(allNotes))
	for _, note := range allNotes {
		localByRelPath[filepath.ToSlash(note.RelPath)] = true
	}
	out := make([]RemoteNoteMeta, 0, len(remote))
	for _, meta := range remote {
		rec, ok := records[meta.ID]
		if ok && localByRelPath[filepath.ToSlash(strings.TrimSpace(rec.RelPath))] {
			continue
		}
		out = append(out, meta)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RelPath == out[j].RelPath {
			return out[i].ID < out[j].ID
		}
		return out[i].RelPath < out[j].RelPath
	})
	return out, nil
}
