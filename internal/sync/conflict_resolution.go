package sync

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
)

func ResolveConflictKeepRemote(root string, rec NoteRecord) error {
	if rec.Conflict == nil {
		return errors.New("note does not have an active conflict")
	}
	conflictRelPath := filepath.ToSlash(strings.TrimSpace(rec.Conflict.CopyPath))
	if conflictRelPath == "" {
		return errors.New("conflict copy path is missing")
	}
	conflictPath := filepath.Join(root, filepath.FromSlash(conflictRelPath))
	raw, err := os.ReadFile(conflictPath)
	if err != nil {
		return err
	}

	notePath := filepath.Join(root, filepath.FromSlash(strings.TrimSpace(rec.RelPath)))
	if err := os.MkdirAll(filepath.Dir(notePath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(notePath, raw, 0o644); err != nil {
		return err
	}

	now := time.Now().UTC()
	rec.LastSyncedHash = HashContent(string(raw))
	rec.Encrypted = notes.NoteIsEncrypted(string(raw))
	rec.LastSyncAt = now
	rec.LastSyncAttemptAt = now
	rec.LastSyncError = ""
	rec.Conflict = nil
	if err := SaveNoteRecord(root, rec); err != nil {
		return err
	}
	if err := DeleteConflictRecord(root, rec.ID); err != nil {
		return err
	}
	if err := os.Remove(conflictPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func ResolveConflictKeepLocal(ctx context.Context, root, notePath, remoteRootOverride string, cfg config.SyncConfig, rec NoteRecord, client Client) error {
	if rec.Conflict == nil {
		return errors.New("note does not have an active conflict")
	}
	profile, profileName, err := ActiveProfile(cfg, root)
	if err != nil {
		return err
	}
	if client == nil {
		client = NewClient(profile)
	}
	profile.RemoteRoot = resolvedRemoteRoot(profile, root, remoteRootOverride)
	rootCfg, err := EnsureRootConfig(root, cfg)
	if err != nil {
		return err
	}
	if rootCfg.Profile != profileName {
		rootCfg.Profile = profileName
		if err := SaveRootConfig(root, rootCfg); err != nil {
			return err
		}
	}

	raw, err := notes.ReadAll(notePath)
	if err != nil {
		return err
	}
	resp, err := client.PushNote(ctx, profile, PushNoteRequest{
		RemoteRoot:       profile.RemoteRoot,
		NoteID:           rec.ID,
		ExpectedRevision: rec.RemoteRev,
		RelPath:          filepath.ToSlash(strings.TrimSpace(rec.RelPath)),
		Content:          raw,
		Encrypted:        notes.NoteIsEncrypted(raw),
	})
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	rec.RemoteRev = resp.Revision
	rec.LastSyncedHash = HashContent(raw)
	rec.Encrypted = notes.NoteIsEncrypted(raw)
	rec.LastSyncAt = now
	rec.LastSyncAttemptAt = now
	rec.LastSyncError = ""
	conflictRelPath := ""
	if rec.Conflict != nil {
		conflictRelPath = filepath.ToSlash(strings.TrimSpace(rec.Conflict.CopyPath))
	}
	rec.Conflict = nil
	if err := SaveNoteRecord(root, rec); err != nil {
		return err
	}
	if err := DeleteConflictRecord(root, rec.ID); err != nil {
		return err
	}
	if conflictRelPath != "" {
		conflictPath := filepath.Join(root, filepath.FromSlash(conflictRelPath))
		if err := os.Remove(conflictPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
