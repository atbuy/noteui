package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/fsutil"
)

func ImportRemoteNotes(ctx context.Context, root, remoteRootOverride string, cfg config.SyncConfig, client Client) (SyncResult, error) {
	return importRemoteNotes(ctx, root, remoteRootOverride, cfg, client, "")
}

func ImportRemoteNote(ctx context.Context, root, remoteRootOverride string, cfg config.SyncConfig, noteID string, client Client) (SyncResult, error) {
	return importRemoteNotes(ctx, root, remoteRootOverride, cfg, client, strings.TrimSpace(noteID))
}

func importRemoteNotes(ctx context.Context, root, remoteRootOverride string, cfg config.SyncConfig, client Client, onlyNoteID string) (SyncResult, error) {
	var result SyncResult
	profile, profileName, err := ActiveProfile(cfg, root)
	if err != nil {
		return result, err
	}
	if client == nil {
		client = NewClient(profile)
	}
	profile.RemoteRoot = resolvedRemoteRoot(profile, root, remoteRootOverride)
	rootCfg, err := EnsureRootConfig(root, cfg)
	if err != nil {
		return result, err
	}
	if rootCfg.Profile != profileName {
		rootCfg.Profile = profileName
		if err := SaveRootConfig(root, rootCfg); err != nil {
			return result, err
		}
	}

	records, err := LoadNoteRecords(root)
	if err != nil {
		return result, err
	}
	remoteIndex, err := client.PullIndex(ctx, profile, PullIndexRequest{RemoteRoot: profile.RemoteRoot})
	if err != nil {
		return result, err
	}

	reservedRelPaths := reservedImportRelPaths(root, records)
	now := time.Now().UTC()
	foundTarget := onlyNoteID == ""
	for _, meta := range remoteIndex.Notes {
		if onlyNoteID != "" && meta.ID != onlyNoteID {
			continue
		}
		foundTarget = true
		relPath := filepath.ToSlash(strings.TrimSpace(meta.RelPath))
		if relPath == "" || relPath == "." || strings.HasPrefix(relPath, "../") {
			return result, fmt.Errorf("invalid remote note path: %q", meta.RelPath)
		}

		importRelPath, ok := resolveImportRelPath(root, relPath, meta.ID, reservedRelPaths)
		if !ok {
			result.SkippedImports++
			continue
		}
		targetPath := filepath.Join(root, filepath.FromSlash(importRelPath))
		rec, hasRecord := records[meta.ID]
		if hasRecord {
			currentRelPath := filepath.ToSlash(strings.TrimSpace(rec.RelPath))
			currentPath := filepath.Join(root, filepath.FromSlash(currentRelPath))
			if currentRelPath != "" && fileExists(currentPath) {
				reservedRelPaths[currentRelPath] = rec.ID
				continue
			}
			content, err := fetchRemoteNoteContent(ctx, client, profile, meta)
			if err != nil {
				return result, err
			}
			if err := writeImportedNote(targetPath, content); err != nil {
				return result, err
			}
			rec.RelPath = importRelPath
			rec.Class = ClassSynced
			rec.RemoteRev = meta.Revision
			rec.LastSyncedHash = HashContent(content)
			rec.Encrypted = meta.Encrypted
			rec.LastSyncAt = now
			rec.LastSyncAttemptAt = now
			rec.LastSyncError = ""
			rec.Conflict = nil
			if err := SaveNoteRecord(root, rec); err != nil {
				return result, err
			}
			if err := DeleteConflictRecord(root, rec.ID); err != nil {
				return result, err
			}
			records[meta.ID] = rec
			result.ImportedNotes++
			continue
		}

		content, err := fetchRemoteNoteContent(ctx, client, profile, meta)
		if err != nil {
			return result, err
		}
		if err := writeImportedNote(targetPath, content); err != nil {
			return result, err
		}
		rec = NoteRecord{
			ID:                meta.ID,
			RelPath:           importRelPath,
			Class:             ClassSynced,
			RemoteRev:         meta.Revision,
			LastSyncedHash:    HashContent(content),
			Encrypted:         meta.Encrypted,
			LastSyncAt:        now,
			LastSyncAttemptAt: now,
		}
		if err := SaveNoteRecord(root, rec); err != nil {
			return result, err
		}
		if err := DeleteConflictRecord(root, rec.ID); err != nil {
			return result, err
		}
		records[meta.ID] = rec
		result.ImportedNotes++
	}

	if onlyNoteID != "" && !foundTarget {
		return result, &RPCError{Code: ErrCodeNotFound, Message: "remote note not found"}
	}

	if err := SavePins(root, remoteIndex.Pins); err != nil {
		return result, err
	}
	result.NotesChanged = result.ImportedNotes > 0
	result.PinsChanged = len(remoteIndex.Pins.PinnedNoteIDs) > 0 || len(remoteIndex.Pins.PinnedCategories) > 0
	result.PinnedNoteRelPaths, result.PinnedCategories, err = LoadPinnedRelPaths(root)
	if err != nil {
		return result, err
	}
	result.RemoteOnlyNotes, err = remoteOnlyNotesForRoot(root, remoteIndex.Notes, records)
	return result, err
}

func fetchRemoteNoteContent(ctx context.Context, client Client, profile config.SyncProfile, meta RemoteNoteMeta) (string, error) {
	resp, err := client.FetchNote(ctx, profile, FetchNoteRequest{RemoteRoot: profile.RemoteRoot, NoteID: meta.ID})
	if err != nil {
		return "", err
	}
	return resp.Note.Content, nil
}

func writeImportedNote(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, []byte(content), 0o644)
}

func reservedImportRelPaths(root string, records map[string]NoteRecord) map[string]string {
	out := make(map[string]string, len(records))
	for _, rec := range records {
		relPath := filepath.ToSlash(strings.TrimSpace(rec.RelPath))
		if relPath == "" {
			continue
		}
		if fileExists(filepath.Join(root, filepath.FromSlash(relPath))) {
			out[relPath] = rec.ID
		}
	}
	return out
}

func resolveImportRelPath(root, desiredRelPath, noteID string, reserved map[string]string) (string, bool) {
	desiredRelPath = filepath.ToSlash(strings.TrimSpace(desiredRelPath))
	if desiredRelPath == "" {
		return "", false
	}
	if ownerID, exists := reserved[desiredRelPath]; exists && strings.TrimSpace(ownerID) != strings.TrimSpace(noteID) {
		candidate := duplicateImportRelPath(root, desiredRelPath, noteID, reserved)
		return candidate, candidate != ""
	}
	if fileExists(filepath.Join(root, filepath.FromSlash(desiredRelPath))) {
		return "", false
	}
	reserved[desiredRelPath] = noteID
	return desiredRelPath, true
}

func duplicateImportRelPath(root, desiredRelPath, noteID string, reserved map[string]string) string {
	ext := filepath.Ext(desiredRelPath)
	base := strings.TrimSuffix(desiredRelPath, ext)
	for _, suffix := range duplicateImportSuffixCandidates(noteID) {
		candidate := base + "~" + suffix + ext
		if ownerID, exists := reserved[candidate]; exists && strings.TrimSpace(ownerID) != strings.TrimSpace(noteID) {
			continue
		}
		if fileExists(filepath.Join(root, filepath.FromSlash(candidate))) {
			continue
		}
		reserved[candidate] = noteID
		return candidate
	}
	return ""
}

func duplicateImportSuffixCandidates(noteID string) []string {
	sanitized := sanitizeImportIDSuffix(noteID)
	if sanitized == "" {
		sanitized = "remote"
	}
	lengths := []int{6, 8, 12, len(sanitized)}
	seen := make(map[string]bool)
	out := make([]string, 0, len(lengths)+4)
	for _, n := range lengths {
		if n <= 0 {
			continue
		}
		if n > len(sanitized) {
			n = len(sanitized)
		}
		candidate := sanitized[:n]
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	base := out[len(out)-1]
	for i := 2; i <= 9; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	return out
}

func sanitizeImportIDSuffix(noteID string) string {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range noteID {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_-")
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
