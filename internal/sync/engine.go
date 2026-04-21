// Package sync implements the SSH-based note synchronisation protocol.
package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/fsutil"
	"atbuy/noteui/internal/notes"
)

func isSyncedClass(syncClass string) bool {
	return syncClass == notes.SyncClassSynced || syncClass == notes.SyncClassShared
}

func SyncRoot(ctx context.Context, root, remoteRootOverride string, cfg config.SyncConfig, localPinnedNotes []string, localPinnedCats []string, client Client) (SyncResult, error) {
	var result SyncResult
	if !HasSyncProfile(cfg) {
		return result, nil
	}
	startTime := time.Now()
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
	allNotes, err := notes.Discover(root)
	if err != nil {
		return result, err
	}
	if err := MigratePinsFromState(root, allNotes, localPinnedNotes, localPinnedCats); err != nil {
		return result, err
	}
	if err := SavePinsFromRelPaths(root, allNotes, localPinnedNotes, localPinnedCats); err != nil {
		return result, err
	}
	localPins, err := LoadPins(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}

	records, err := LoadNoteRecords(root)
	if err != nil {
		return result, err
	}
	syncedRelPaths := syncedRelPathsFromNotes(allNotes)

	remoteIndex, err := client.PullIndex(ctx, profile, PullIndexRequest{RemoteRoot: profile.RemoteRoot})
	if err != nil {
		_ = markSyncErrors(root, records, syncedRelPaths, err)
		return result, err
	}
	if len(localPins.PinnedNoteIDs) == 0 && len(localPins.PinnedCategories) == 0 {
		if len(remoteIndex.Pins.PinnedNoteIDs) > 0 || len(remoteIndex.Pins.PinnedCategories) > 0 {
			if err := SavePins(root, remoteIndex.Pins); err != nil {
				return result, err
			}
			result.PinsChanged = true
		}
	}

	noteByRelPath := make(map[string]notes.Note, len(allNotes))
	for _, note := range allNotes {
		noteByRelPath[filepath.ToSlash(note.RelPath)] = note
	}
	remoteByID := make(map[string]RemoteNoteMeta, len(remoteIndex.Notes))
	remoteByRelPath := make(map[string]RemoteNoteMeta, len(remoteIndex.Notes))
	for _, meta := range remoteIndex.Notes {
		remoteByID[meta.ID] = meta
		remoteByRelPath[filepath.ToSlash(meta.RelPath)] = meta
	}

	for id, rec := range records {
		remoteMeta, ok := remoteByID[id]
		localNote, hasLocal := noteByRelPath[filepath.ToSlash(rec.RelPath)]
		if !ok {
			if hasLocal && isSyncedClass(localNote.SyncClass) {
				if err := recordSyncFailure(root, rec, &RPCError{Code: ErrCodeNotFound, Message: "note missing on remote"}); err != nil {
					return result, err
				}
			}
			continue
		}
		if !hasLocal {
			continue
		}
		if !isSyncedClass(localNote.SyncClass) {
			continue
		}
		raw, err := notes.ReadAll(localNote.Path)
		if err != nil {
			return result, err
		}
		localHash := HashContent(raw)
		remoteChanged := remoteContentChanged(rec, remoteMeta.Revision)
		switch {
		case remoteChanged && rec.LastSyncedHash == localHash:
			if err := applyRemoteNote(ctx, client, profile, root, rec, remoteMeta, &result); err != nil {
				_ = recordSyncFailure(root, rec, err)
				return result, err
			}
		case remoteChanged && rec.LastSyncedHash != localHash:
			if localNote.SyncClass == notes.SyncClassShared {
				if err := applyRemoteNote(ctx, client, profile, root, rec, remoteMeta, &result); err != nil {
					_ = recordSyncFailure(root, rec, err)
					return result, err
				}
			} else {
				if err := createConflict(ctx, client, profile, root, rec, remoteMeta, localHash); err != nil {
					_ = recordSyncFailure(root, rec, err)
					return result, err
				}
				result.Conflicts++
			}
		case filepath.ToSlash(rec.RelPath) != filepath.ToSlash(remoteMeta.RelPath) && rec.LastSyncedHash == localHash:
			if err := moveLocalFile(root, rec.RelPath, remoteMeta.RelPath); err != nil {
				return result, err
			}
			rec.RelPath = filepath.ToSlash(remoteMeta.RelPath)
			rec.RemoteRev = remoteMeta.Revision
			rec.LastSyncAt = time.Now().UTC()
			rec.LastSyncAttemptAt = rec.LastSyncAt
			rec.LastSyncError = ""
			if err := SaveNoteRecord(root, rec); err != nil {
				return result, err
			}
			result.NotesChanged = true
		}
	}

	records, err = LoadNoteRecords(root)
	if err != nil {
		return result, err
	}
	recordsByRelPath := make(map[string]NoteRecord, len(records))
	for _, rec := range records {
		recordsByRelPath[filepath.ToSlash(rec.RelPath)] = rec
	}
	orphanedRecords := orphanedSyncedRecords(records, remoteByID, noteByRelPath)
	claimedOrphanedRecordIDs := make(map[string]bool, len(orphanedRecords))

	for _, note := range allNotes {
		if !isSyncedClass(note.SyncClass) {
			continue
		}
		raw, err := notes.ReadAll(note.Path)
		if err != nil {
			return result, err
		}
		rec, ok := recordsByRelPath[filepath.ToSlash(note.RelPath)]
		if !ok {
			rec, ok = matchOrphanedRecord(note, raw, orphanedRecords, claimedOrphanedRecordIDs)
		}
		if !ok {
			if remoteMeta, adoptable := remoteByRelPath[filepath.ToSlash(note.RelPath)]; adoptable {
				// Remote already has a note at this path. Adopt its ID instead of
				// registering a new one, which would create a duplicate on the server.
				// This happens when local sync records are missing (e.g. fresh install,
				// .noteui-sync deleted) but the note was previously synced from another device.
				rec = NoteRecord{
					ID:                remoteMeta.ID,
					RelPath:           filepath.ToSlash(note.RelPath),
					Class:             ClassSynced,
					RemoteRev:         remoteMeta.Revision,
					Encrypted:         note.Encrypted,
					LastSyncAttemptAt: time.Now().UTC(),
				}
				if err := SaveNoteRecord(root, rec); err != nil {
					return result, err
				}
				recordsByRelPath[filepath.ToSlash(note.RelPath)] = rec
				// Fall through to the push/pull logic below; do not continue.
			} else {
				resp, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{RemoteRoot: profile.RemoteRoot, RelPath: filepath.ToSlash(note.RelPath), Content: raw, Encrypted: note.Encrypted})
				if err != nil {
					return result, err
				}
				rec = NoteRecord{ID: resp.ID, RelPath: filepath.ToSlash(note.RelPath), Class: ClassSynced, RemoteRev: resp.Revision, LastSyncedHash: HashContent(raw), Encrypted: note.Encrypted, LastSyncAt: time.Now().UTC(), LastSyncAttemptAt: time.Now().UTC()}
				rec.LastSyncError = ""
				if err := SaveNoteRecord(root, rec); err != nil {
					return result, err
				}
				result.RegisteredNotes++
				continue
			}
		}
		if _, exists := remoteByID[rec.ID]; !exists {
			if remoteMeta, adoptable := remoteByRelPath[filepath.ToSlash(note.RelPath)]; adoptable {
				oldID := rec.ID
				rec.ID = remoteMeta.ID
				rec.RelPath = filepath.ToSlash(note.RelPath)
				rec.RemoteRev = remoteMeta.Revision
				rec.Encrypted = note.Encrypted
				rec.LastSyncAttemptAt = time.Now().UTC()
				rec.LastSyncError = ""
				if oldID != rec.ID {
					if err := DeleteNoteRecord(root, oldID); err != nil {
						return result, err
					}
				}
				if err := SaveNoteRecord(root, rec); err != nil {
					return result, err
				}
				recordsByRelPath[filepath.ToSlash(note.RelPath)] = rec
			} else {
				resp, err := client.RegisterNote(ctx, profile, RegisterNoteRequest{
					RemoteRoot: profile.RemoteRoot,
					RelPath:    filepath.ToSlash(note.RelPath),
					Content:    raw,
					Encrypted:  note.Encrypted,
				})
				if err != nil {
					_ = recordSyncFailure(root, rec, fmt.Errorf("remote note missing; re-register failed: %w", err))
					return result, err
				}
				oldID := rec.ID
				rec = NoteRecord{
					ID:                resp.ID,
					RelPath:           filepath.ToSlash(note.RelPath),
					Class:             ClassSynced,
					RemoteRev:         resp.Revision,
					LastSyncedHash:    HashContent(raw),
					Encrypted:         note.Encrypted,
					LastSyncAt:        time.Now().UTC(),
					LastSyncAttemptAt: time.Now().UTC(),
				}
				rec.LastSyncError = ""
				if oldID != rec.ID {
					if err := DeleteNoteRecord(root, oldID); err != nil {
						return result, err
					}
				}
				if err := SaveNoteRecord(root, rec); err != nil {
					return result, err
				}
				recordsByRelPath[filepath.ToSlash(note.RelPath)] = rec
				result.RegisteredNotes++
				continue
			}
		}
		localHash := HashContent(raw)
		if filepath.ToSlash(rec.RelPath) != filepath.ToSlash(note.RelPath) && rec.LastSyncedHash == localHash {
			resp, err := client.UpdateNotePath(ctx, profile, UpdateNotePathRequest{RemoteRoot: profile.RemoteRoot, NoteID: rec.ID, ExpectedRevision: rec.RemoteRev, RelPath: filepath.ToSlash(note.RelPath)})
			if err != nil {
				_ = recordSyncFailure(root, rec, err)
				return result, err
			}
			rec.RelPath = filepath.ToSlash(note.RelPath)
			rec.RemoteRev = resp.Revision
			rec.LastSyncAt = time.Now().UTC()
			rec.LastSyncAttemptAt = rec.LastSyncAt
			rec.LastSyncError = ""
			if err := SaveNoteRecord(root, rec); err != nil {
				return result, err
			}
			if err := DeleteConflictRecord(root, rec.ID); err != nil {
				return result, err
			}
			result.UpdatedNotes++
			continue
		}
		if rec.LastSyncedHash == localHash && rec.Encrypted == note.Encrypted {
			if rec.Conflict == nil && (strings.TrimSpace(rec.LastSyncError) != "" || rec.LastSyncAt.IsZero()) {
				rec.LastSyncAt = time.Now().UTC()
				rec.LastSyncAttemptAt = rec.LastSyncAt
				rec.LastSyncError = ""
				if err := SaveNoteRecord(root, rec); err != nil {
					return result, err
				}
			}
			continue
		}
		resp, err := client.PushNote(ctx, profile, PushNoteRequest{RemoteRoot: profile.RemoteRoot, NoteID: rec.ID, ExpectedRevision: rec.RemoteRev, RelPath: filepath.ToSlash(note.RelPath), Content: raw, Encrypted: note.Encrypted})
		if err != nil {
			_ = recordSyncFailure(root, rec, err)
			return result, err
		}
		rec.RelPath = filepath.ToSlash(note.RelPath)
		rec.RemoteRev = resp.Revision
		rec.LastSyncedHash = localHash
		rec.Encrypted = note.Encrypted
		rec.LastSyncAt = time.Now().UTC()
		rec.LastSyncAttemptAt = rec.LastSyncAt
		rec.LastSyncError = ""
		rec.Conflict = nil
		if err := SaveNoteRecord(root, rec); err != nil {
			return result, err
		}
		if err := DeleteConflictRecord(root, rec.ID); err != nil {
			return result, err
		}
		result.UpdatedNotes++
	}

	if err := SavePinsFromRelPaths(root, allNotes, localPinnedNotes, localPinnedCats); err != nil {
		return result, err
	}
	localPins, err = LoadPins(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	if len(localPins.PinnedNoteIDs) > 0 || len(localPins.PinnedCategories) > 0 {
		if _, err := client.PinsPut(ctx, profile, PinsPutRequest{RemoteRoot: profile.RemoteRoot, Pins: localPins}); err != nil {
			return result, err
		}
	}
	result.PinnedNoteRelPaths, result.PinnedCategories, err = LoadPinnedRelPaths(root)
	if err != nil {
		return result, err
	}
	result.RemoteOnlyNotes, err = remoteOnlyNotesForRoot(root, remoteIndex.Notes, records)
	_ = appendSyncResultEvent(root, profileName, startTime, result, err)
	return result, err
}

func appendSyncResultEvent(root, profileName string, startTime time.Time, result SyncResult, syncErr error) error {
	event := SyncEvent{
		Timestamp:       time.Now().UTC(),
		ProfileName:     profileName,
		RegisteredNotes: result.RegisteredNotes,
		UpdatedNotes:    result.UpdatedNotes,
		Conflicts:       result.Conflicts,
		DurationMs:      time.Since(startTime).Milliseconds(),
	}
	switch {
	case syncErr != nil:
		event.Type = SyncEventError
		event.ErrorMsg = syncErr.Error()
	case result.Conflicts > 0:
		event.Type = SyncEventConflict
	default:
		event.Type = SyncEventSuccess
	}
	return AppendSyncEvent(root, event)
}

func applyRemoteNote(ctx context.Context, client Client, profile config.SyncProfile, root string, rec NoteRecord, meta RemoteNoteMeta, result *SyncResult) error {
	resp, err := client.FetchNote(ctx, profile, FetchNoteRequest{RemoteRoot: profile.RemoteRoot, NoteID: meta.ID})
	if err != nil {
		return err
	}
	targetPath := filepath.Join(root, filepath.FromSlash(meta.RelPath))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomic(targetPath, []byte(resp.Note.Content), 0o644); err != nil {
		return err
	}
	if filepath.ToSlash(rec.RelPath) != filepath.ToSlash(meta.RelPath) && strings.TrimSpace(rec.RelPath) != "" {
		oldPath := filepath.Join(root, filepath.FromSlash(rec.RelPath))
		if oldPath != targetPath {
			_ = os.Remove(oldPath)
		}
	}
	rec.RelPath = filepath.ToSlash(meta.RelPath)
	rec.RemoteRev = meta.Revision
	rec.LastSyncedHash = HashContent(resp.Note.Content)
	rec.Encrypted = meta.Encrypted
	rec.LastSyncAt = time.Now().UTC()
	rec.LastSyncAttemptAt = rec.LastSyncAt
	rec.LastSyncError = ""
	rec.Conflict = nil
	if err := SaveNoteRecord(root, rec); err != nil {
		return err
	}
	if err := DeleteConflictRecord(root, rec.ID); err != nil {
		return err
	}
	result.NotesChanged = true
	return nil
}

func createConflict(ctx context.Context, client Client, profile config.SyncProfile, root string, rec NoteRecord, meta RemoteNoteMeta, localHash string) error {
	resp, err := client.FetchNote(ctx, profile, FetchNoteRequest{RemoteRoot: profile.RemoteRoot, NoteID: meta.ID})
	if err != nil {
		return err
	}
	ext := filepath.Ext(rec.RelPath)
	base := strings.TrimSuffix(rec.RelPath, ext)
	if ext == "" {
		ext = ".md"
	}
	conflictRelPath := fmt.Sprintf("%s.conflict-%s%s", base, time.Now().UTC().Format("20060102-150405"), ext)
	conflictPath := filepath.Join(root, filepath.FromSlash(conflictRelPath))
	if err := os.MkdirAll(filepath.Dir(conflictPath), 0o755); err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomic(conflictPath, []byte(resp.Note.Content), 0o644); err != nil {
		return err
	}
	rec.Conflict = &ConflictInfo{CopyPath: filepath.ToSlash(conflictRelPath), OccurredAt: time.Now().UTC()}
	rec.RemoteRev = meta.Revision
	rec.LastSyncedHash = localHash
	rec.LastSyncAttemptAt = time.Now().UTC()
	rec.LastSyncError = "conflict"
	if err := SaveNoteRecord(root, rec); err != nil {
		return err
	}
	return SaveConflictRecord(root, ConflictRecord{NoteID: rec.ID, LocalPath: filepath.ToSlash(rec.RelPath), RemoteRev: meta.Revision, LocalHash: localHash, ConflictPath: filepath.ToSlash(conflictRelPath), OccurredAt: time.Now().UTC()})
}

func moveLocalFile(root, oldRelPath, newRelPath string) error {
	oldPath := filepath.Join(root, filepath.FromSlash(oldRelPath))
	newPath := filepath.Join(root, filepath.FromSlash(newRelPath))
	if oldPath == newPath {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return err
	}
	return os.Rename(oldPath, newPath)
}

func orphanedSyncedRecords(records map[string]NoteRecord, remoteByID map[string]RemoteNoteMeta, localByRelPath map[string]notes.Note) []NoteRecord {
	out := make([]NoteRecord, 0, len(records))
	for _, rec := range records {
		if _, exists := remoteByID[rec.ID]; !exists {
			continue
		}
		localNote, hasLocal := localByRelPath[filepath.ToSlash(rec.RelPath)]
		if hasLocal && isSyncedClass(localNote.SyncClass) {
			continue
		}
		out = append(out, rec)
	}
	return out
}

func matchOrphanedRecord(note notes.Note, raw string, orphaned []NoteRecord, claimed map[string]bool) (NoteRecord, bool) {
	localHash := HashContent(raw)
	var matched NoteRecord
	matchCount := 0
	for _, rec := range orphaned {
		if claimed[rec.ID] {
			continue
		}
		if rec.LastSyncedHash != localHash || rec.Encrypted != note.Encrypted {
			continue
		}
		matched = rec
		matchCount++
		if matchCount > 1 {
			return NoteRecord{}, false
		}
	}
	if matchCount != 1 {
		return NoteRecord{}, false
	}
	claimed[matched.ID] = true
	return matched, true
}

func syncedRelPathsFromNotes(allNotes []notes.Note) []string {
	out := make([]string, 0, len(allNotes))
	for _, note := range allNotes {
		if isSyncedClass(note.SyncClass) {
			out = append(out, filepath.ToSlash(note.RelPath))
		}
	}
	return out
}

func recordSyncFailure(root string, rec NoteRecord, syncErr error) error {
	rec.LastSyncAttemptAt = time.Now().UTC()
	rec.LastSyncError = syncErr.Error()
	return SaveNoteRecord(root, rec)
}

func markSyncErrors(root string, records map[string]NoteRecord, relPaths []string, syncErr error) error {
	if len(records) == 0 || len(relPaths) == 0 {
		return nil
	}
	want := make(map[string]bool, len(relPaths))
	for _, relPath := range relPaths {
		want[filepath.ToSlash(relPath)] = true
	}
	for _, rec := range records {
		if !want[filepath.ToSlash(rec.RelPath)] {
			continue
		}
		rec.LastSyncAttemptAt = time.Now().UTC()
		rec.LastSyncError = syncErr.Error()
		if err := SaveNoteRecord(root, rec); err != nil {
			return err
		}
	}
	return nil
}
