package sync

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
)

func DeleteRemoteNoteAndKeepLocal(ctx context.Context, root, path string, cfg config.SyncConfig, client Client) error {
	if client == nil {
		client = SSHClient{}
	}
	profile, profileName, err := ActiveProfile(cfg, root)
	if err != nil {
		return err
	}
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

	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))

	records, err := LoadNoteRecords(root)
	if err != nil {
		return err
	}
	var rec NoteRecord
	found := false
	for _, candidate := range records {
		if filepath.ToSlash(strings.TrimSpace(candidate.RelPath)) == relPath {
			rec = candidate
			found = true
			break
		}
	}
	if !found {
		return errors.New("note is not linked to remote sync")
	}

	_, err = client.DeleteNote(ctx, profile, DeleteNoteRequest{
		RemoteRoot:       profile.RemoteRoot,
		NoteID:           rec.ID,
		ExpectedRevision: rec.RemoteRev,
	})
	if err != nil {
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != ErrCodeNotFound {
			return err
		}
	}

	if err := notes.SetNoteSyncClass(path, notes.SyncClassLocal); err != nil {
		return err
	}
	if err := DeleteNoteRecord(root, rec.ID); err != nil {
		return err
	}
	if err := RemovePinnedNoteID(root, rec.ID); err != nil {
		return err
	}
	return nil
}
