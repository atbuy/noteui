package sync

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

const (
	SchemaVersion    = 1
	SyncDirName      = ".noteui-sync"
	NotesDirName     = "notes"
	ConflictsDirName = "conflicts"
	DefaultRemoteBin = "noteui-sync"
	ClassLocal       = "local"
	ClassSynced      = "synced"
	ErrCodeNotFound  = "not_found"
	ErrCodeConflict  = "revision_mismatch"
	ErrCodeInvalid   = "invalid_request"
	ErrCodeInternal  = "internal_error"
)

type RootConfig struct {
	SchemaVersion int    `json:"schema_version"`
	ClientID      string `json:"client_id"`
	Profile       string `json:"profile"`
}

type ConflictInfo struct {
	CopyPath   string    `json:"copy_path"`
	OccurredAt time.Time `json:"occurred_at"`
}

type NoteRecord struct {
	ID                string        `json:"id"`
	RelPath           string        `json:"rel_path"`
	Class             string        `json:"class"`
	RemoteRev         string        `json:"remote_rev"`
	LastSyncedHash    string        `json:"last_synced_hash"`
	Encrypted         bool          `json:"encrypted"`
	LastSyncAt        time.Time     `json:"last_sync_at,omitempty"`
	LastSyncAttemptAt time.Time     `json:"last_sync_attempt_at,omitempty"`
	LastSyncError     string        `json:"last_sync_error,omitempty"`
	Conflict          *ConflictInfo `json:"conflict,omitempty"`
}

type Pins struct {
	PinnedNoteIDs    []string `json:"pinned_note_ids"`
	PinnedCategories []string `json:"pinned_categories"`
}

type ConflictRecord struct {
	NoteID       string    `json:"note_id"`
	LocalPath    string    `json:"local_path"`
	RemoteRev    string    `json:"remote_rev"`
	LocalHash    string    `json:"local_hash"`
	ConflictPath string    `json:"conflict_copy_path"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type RemoteNoteMeta struct {
	ID        string `json:"id"`
	RelPath   string `json:"rel_path"`
	Title     string `json:"title,omitempty"`
	Revision  string `json:"revision"`
	Encrypted bool   `json:"encrypted"`
}

type RemoteNote struct {
	RemoteNoteMeta
	Content string `json:"content"`
}

type PullIndexRequest struct {
	RemoteRoot string `json:"remote_root"`
}
type PullIndexResponse struct {
	Notes []RemoteNoteMeta `json:"notes"`
	Pins  Pins             `json:"pins"`
	// SkippedCount is the number of remote notes that the backend
	// discovered but could not materialize (corrupt mapping, fetch error,
	// etc). It lets the sync engine report partial failures instead of
	// silently dropping notes. Zero for backends that do not track it.
	SkippedCount int `json:"skipped_count,omitempty"`
}

type FetchNoteRequest struct {
	RemoteRoot string `json:"remote_root"`
	NoteID     string `json:"note_id"`
}
type FetchNoteResponse struct {
	Note RemoteNote `json:"note"`
}

type RegisterNoteRequest struct {
	RemoteRoot string `json:"remote_root"`
	RelPath    string `json:"rel_path"`
	Content    string `json:"content"`
	Encrypted  bool   `json:"encrypted"`
}
type RegisterNoteResponse struct {
	ID       string `json:"id"`
	Revision string `json:"revision"`
}

type PushNoteRequest struct {
	RemoteRoot       string `json:"remote_root"`
	NoteID           string `json:"note_id"`
	ExpectedRevision string `json:"expected_revision"`
	RelPath          string `json:"rel_path"`
	Content          string `json:"content"`
	Encrypted        bool   `json:"encrypted"`
}
type PushNoteResponse struct {
	Revision string `json:"revision"`
}

type UpdateNotePathRequest struct {
	RemoteRoot       string `json:"remote_root"`
	NoteID           string `json:"note_id"`
	ExpectedRevision string `json:"expected_revision"`
	RelPath          string `json:"rel_path"`
}
type UpdateNotePathResponse struct {
	Revision string `json:"revision"`
}

type DeleteNoteRequest struct {
	RemoteRoot       string `json:"remote_root"`
	NoteID           string `json:"note_id"`
	ExpectedRevision string `json:"expected_revision"`
}

type DeleteNoteResponse struct{}

type PinsGetRequest struct {
	RemoteRoot string `json:"remote_root"`
}
type PinsGetResponse struct {
	Pins Pins `json:"pins"`
}

type PinsPutRequest struct {
	RemoteRoot string `json:"remote_root"`
	Pins       Pins   `json:"pins"`
}
type PinsPutResponse struct {
	Pins Pins `json:"pins"`
}

type RPCError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

type SyncResult struct {
	NotesChanged       bool
	PinsChanged        bool
	PinnedNoteRelPaths []string
	PinnedCategories   []string
	RemoteOnlyNotes    []RemoteNoteMeta
	ImportedNotes      int
	SkippedImports     int
	RegisteredNotes    int
	UpdatedNotes       int
	Conflicts          int
}

func NewClientID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "client-" + time.Now().UTC().Format("20060102150405")
	}
	return "client-" + hex.EncodeToString(buf[:])
}
