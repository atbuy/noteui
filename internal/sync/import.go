package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"atbuy/noteui/internal/config"
)

func ImportRemoteNotes(ctx context.Context, root string, cfg config.SyncConfig, client Client) (SyncResult, error) {
	return importRemoteNotes(ctx, root, cfg, client, "")
}

func ImportRemoteNote(ctx context.Context, root string, cfg config.SyncConfig, noteID string, client Client) (SyncResult, error) {
	return importRemoteNotes(ctx, root, cfg, client, strings.TrimSpace(noteID))
}

func importRemoteNotes(ctx context.Context, root string, cfg config.SyncConfig, client Client, onlyNoteID string) (SyncResult, error) {
	var result SyncResult
	if client == nil {
		client = SSHClient{}
	}
	profile, profileName, err := ActiveProfile(cfg, root)
	if err != nil {
		return result, err
	}
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

		targetPath := filepath.Join(root, filepath.FromSlash(relPath))
		rec, hasRecord := records[meta.ID]
		if hasRecord {
			currentPath := filepath.Join(root, filepath.FromSlash(filepath.ToSlash(strings.TrimSpace(rec.RelPath))))
			if fileExists(currentPath) {
				continue
			}
			if currentPath != targetPath && fileExists(targetPath) {
				result.SkippedImports++
				continue
			}
			content, err := fetchRemoteNoteContent(ctx, client, profile, meta)
			if err != nil {
				return result, err
			}
			if err := writeImportedNote(targetPath, content); err != nil {
				return result, err
			}
			rec.RelPath = relPath
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
			result.ImportedNotes++
			continue
		}

		if fileExists(targetPath) {
			result.SkippedImports++
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
			RelPath:           relPath,
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
	return os.WriteFile(path, []byte(content), 0o644)
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
