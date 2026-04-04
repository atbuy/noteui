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
	// Build a relPath→ID map so we can tell whether a local file is already claimed
	// by a different sync record (a legitimate path collision) vs. unclaimed (orphaned
	// local file whose records were deleted).
	recordByRelPath := make(map[string]string, len(records))
	for _, rec := range records {
		relPath := filepath.ToSlash(strings.TrimSpace(rec.RelPath))
		if relPath != "" {
			recordByRelPath[relPath] = rec.ID
		}
	}

	out := make([]RemoteNoteMeta, 0, len(remote))
	for _, meta := range remote {
		rec, ok := records[meta.ID]
		if ok && localByRelPath[filepath.ToSlash(strings.TrimSpace(rec.RelPath))] {
			continue
		}
		// Defensive: skip if a local note exists at this path AND no other record
		// already owns that path. This prevents phantom "remote only" entries when
		// sync records are missing (e.g. .noteui-sync deleted) but the local file is
		// present — the note will be reconciled on the next full sync via adopt logic.
		// We do NOT skip when another record owns the path, because in that case the
		// remote note is a genuine path collision and the user needs to handle it.
		metaRelPath := filepath.ToSlash(strings.TrimSpace(meta.RelPath))
		if localByRelPath[metaRelPath] {
			ownerID := recordByRelPath[metaRelPath]
			if ownerID == "" || ownerID == meta.ID {
				continue
			}
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
