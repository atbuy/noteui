package sync

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"atbuy/noteui/internal/notes"
)

type remoteNoteFile struct {
	ID        string `json:"id"`
	RelPath   string `json:"rel_path"`
	Title     string `json:"title,omitempty"`
	Revision  int    `json:"revision"`
	Encrypted bool   `json:"encrypted"`
}

type Server struct{}

func (Server) PullIndex(req PullIndexRequest) (PullIndexResponse, error) {
	notes, err := loadRemoteNotes(req.RemoteRoot)
	if err != nil {
		return PullIndexResponse{}, err
	}
	pins, err := loadRemotePins(req.RemoteRoot)
	if err != nil {
		return PullIndexResponse{}, err
	}
	resp := PullIndexResponse{Pins: pins}
	for _, note := range notes {
		resp.Notes = append(resp.Notes, RemoteNoteMeta{ID: note.ID, RelPath: filepath.ToSlash(note.RelPath), Title: note.Title, Revision: strconv.Itoa(note.Revision), Encrypted: note.Encrypted})
	}
	return resp, nil
}

func (Server) FetchNote(req FetchNoteRequest) (FetchNoteResponse, error) {
	note, err := loadRemoteNote(req.RemoteRoot, req.NoteID)
	if err != nil {
		return FetchNoteResponse{}, err
	}
	content, err := os.ReadFile(remoteContentPath(req.RemoteRoot, req.NoteID))
	if err != nil {
		return FetchNoteResponse{}, err
	}
	return FetchNoteResponse{Note: RemoteNote{RemoteNoteMeta: RemoteNoteMeta{ID: note.ID, RelPath: filepath.ToSlash(note.RelPath), Title: note.Title, Revision: strconv.Itoa(note.Revision), Encrypted: note.Encrypted}, Content: string(content)}}, nil
}

func (Server) RegisterNote(req RegisterNoteRequest) (RegisterNoteResponse, error) {
	id := "n_" + strings.ReplaceAll(NewClientID(), "client-", "")
	note := remoteNoteFile{ID: id, RelPath: filepath.Clean(req.RelPath), Title: titleFromContent(req.RelPath, req.Content), Revision: 1, Encrypted: req.Encrypted}
	if err := saveRemoteNote(req.RemoteRoot, note, req.Content); err != nil {
		return RegisterNoteResponse{}, err
	}
	return RegisterNoteResponse{ID: id, Revision: "1"}, nil
}

func (Server) PushNote(req PushNoteRequest) (PushNoteResponse, error) {
	note, err := loadRemoteNote(req.RemoteRoot, req.NoteID)
	if err != nil {
		return PushNoteResponse{}, err
	}
	if strconv.Itoa(note.Revision) != strings.TrimSpace(req.ExpectedRevision) {
		return PushNoteResponse{}, &RPCError{Code: ErrCodeConflict, Message: "revision mismatch"}
	}
	note.RelPath = filepath.Clean(req.RelPath)
	note.Title = titleFromContent(req.RelPath, req.Content)
	note.Revision++
	note.Encrypted = req.Encrypted
	if err := saveRemoteNote(req.RemoteRoot, note, req.Content); err != nil {
		return PushNoteResponse{}, err
	}
	return PushNoteResponse{Revision: strconv.Itoa(note.Revision)}, nil
}

func (Server) UpdateNotePath(req UpdateNotePathRequest) (UpdateNotePathResponse, error) {
	note, err := loadRemoteNote(req.RemoteRoot, req.NoteID)
	if err != nil {
		return UpdateNotePathResponse{}, err
	}
	if strconv.Itoa(note.Revision) != strings.TrimSpace(req.ExpectedRevision) {
		return UpdateNotePathResponse{}, &RPCError{Code: ErrCodeConflict, Message: "revision mismatch"}
	}
	note.RelPath = filepath.Clean(req.RelPath)
	note.Revision++
	content, err := os.ReadFile(remoteContentPath(req.RemoteRoot, req.NoteID))
	if err != nil {
		return UpdateNotePathResponse{}, err
	}
	if err := saveRemoteNote(req.RemoteRoot, note, string(content)); err != nil {
		return UpdateNotePathResponse{}, err
	}
	return UpdateNotePathResponse{Revision: strconv.Itoa(note.Revision)}, nil
}

func (Server) DeleteNote(req DeleteNoteRequest) (DeleteNoteResponse, error) {
	note, err := loadRemoteNote(req.RemoteRoot, req.NoteID)
	if err != nil {
		return DeleteNoteResponse{}, err
	}
	if strconv.Itoa(note.Revision) != strings.TrimSpace(req.ExpectedRevision) {
		return DeleteNoteResponse{}, &RPCError{Code: ErrCodeConflict, Message: "revision mismatch"}
	}
	if err := os.Remove(remoteMetaPath(req.RemoteRoot, req.NoteID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return DeleteNoteResponse{}, err
	}
	if err := os.Remove(remoteContentPath(req.RemoteRoot, req.NoteID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return DeleteNoteResponse{}, err
	}
	pins, err := loadRemotePins(req.RemoteRoot)
	if err != nil {
		return DeleteNoteResponse{}, err
	}
	pins.PinnedNoteIDs = removeString(pins.PinnedNoteIDs, req.NoteID)
	if err := saveRemotePins(req.RemoteRoot, pins); err != nil {
		return DeleteNoteResponse{}, err
	}
	return DeleteNoteResponse{}, nil
}

func (Server) PinsGet(req PinsGetRequest) (PinsGetResponse, error) {
	pins, err := loadRemotePins(req.RemoteRoot)
	if err != nil {
		return PinsGetResponse{}, err
	}
	return PinsGetResponse{Pins: pins}, nil
}

func (Server) PinsPut(req PinsPutRequest) (PinsPutResponse, error) {
	if err := saveRemotePins(req.RemoteRoot, req.Pins); err != nil {
		return PinsPutResponse{}, err
	}
	return PinsPutResponse{Pins: req.Pins}, nil
}

func HandleRPC(op string, payload []byte) ([]byte, error) {
	server := Server{}
	var data any
	var err error
	switch op {
	case "pull_index":
		var req PullIndexRequest
		if err = json.Unmarshal(payload, &req); err == nil {
			data, err = server.PullIndex(req)
		}
	case "fetch_note":
		var req FetchNoteRequest
		if err = json.Unmarshal(payload, &req); err == nil {
			data, err = server.FetchNote(req)
		}
	case "register_note":
		var req RegisterNoteRequest
		if err = json.Unmarshal(payload, &req); err == nil {
			data, err = server.RegisterNote(req)
		}
	case "push_note":
		var req PushNoteRequest
		if err = json.Unmarshal(payload, &req); err == nil {
			data, err = server.PushNote(req)
		}
	case "update_note_path":
		var req UpdateNotePathRequest
		if err = json.Unmarshal(payload, &req); err == nil {
			data, err = server.UpdateNotePath(req)
		}
	case "delete_note":
		var req DeleteNoteRequest
		if err = json.Unmarshal(payload, &req); err == nil {
			data, err = server.DeleteNote(req)
		}
	case "pins_get":
		var req PinsGetRequest
		if err = json.Unmarshal(payload, &req); err == nil {
			data, err = server.PinsGet(req)
		}
	case "pins_put":
		var req PinsPutRequest
		if err = json.Unmarshal(payload, &req); err == nil {
			data, err = server.PinsPut(req)
		}
	default:
		err = &RPCError{Code: ErrCodeInvalid, Message: "unknown operation"}
	}
	return marshalRPCResponse(data, err)
}

func marshalRPCResponse(data any, err error) ([]byte, error) {
	type response struct {
		Error *RPCError `json:"error,omitempty"`
		Data  any       `json:"data,omitempty"`
	}
	resp := response{Data: data}
	if err != nil {
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) {
			rpcErr = &RPCError{Code: ErrCodeInternal, Message: err.Error()}
		}
		resp.Error = rpcErr
		resp.Data = nil
	}
	return json.Marshal(resp)
}

func loadRemoteNotes(root string) ([]remoteNoteFile, error) {
	dir := filepath.Join(root, "notes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []remoteNoteFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var note remoteNoteFile
		if err := json.Unmarshal(data, &note); err != nil {
			return nil, err
		}
		out = append(out, note)
	}
	return out, nil
}

func loadRemoteNote(root, noteID string) (remoteNoteFile, error) {
	var note remoteNoteFile
	data, err := os.ReadFile(remoteMetaPath(root, noteID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return note, &RPCError{Code: ErrCodeNotFound, Message: "note not found"}
		}
		return note, err
	}
	if err := json.Unmarshal(data, &note); err != nil {
		return remoteNoteFile{}, err
	}
	return note, nil
}

func saveRemoteNote(root string, note remoteNoteFile, content string) error {
	if strings.TrimSpace(note.ID) == "" {
		return errors.New("note id cannot be empty")
	}
	if strings.TrimSpace(note.RelPath) == "" {
		return errors.New("note rel_path cannot be empty")
	}
	if err := os.MkdirAll(filepath.Dir(remoteMetaPath(root, note.ID)), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(remoteContentPath(root, note.ID)), 0o755); err != nil {
		return err
	}
	meta, err := json.MarshalIndent(note, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(remoteMetaPath(root, note.ID), meta, 0o644); err != nil {
		return err
	}
	return os.WriteFile(remoteContentPath(root, note.ID), []byte(content), 0o644)
}

func loadRemotePins(root string) (Pins, error) {
	var pins Pins
	data, err := os.ReadFile(filepath.Join(root, "pins.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return pins, nil
		}
		return pins, err
	}
	if len(data) == 0 {
		return pins, nil
	}
	if err := json.Unmarshal(data, &pins); err != nil {
		return Pins{}, err
	}
	return pins, nil
}

func saveRemotePins(root string, pins Pins) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(pins, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "pins.json"), data, 0o644)
}

func titleFromContent(relPath, content string) string {
	title := strings.TrimSpace(notes.ExtractTitle(content))
	if title != "" {
		return title
	}
	return filepath.Base(filepath.ToSlash(relPath))
}

func remoteMetaPath(root, noteID string) string { return filepath.Join(root, "notes", noteID+".json") }
func remoteContentPath(root, noteID string) string {
	return filepath.Join(root, "content", noteID+".note")
}

func removeString(items []string, target string) []string {
	target = strings.TrimSpace(target)
	if target == "" || len(items) == 0 {
		return items
	}
	out := items[:0]
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
