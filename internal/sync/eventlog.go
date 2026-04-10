package sync

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SyncEventType describes the outcome of a sync run.
type SyncEventType string

const (
	SyncEventSuccess  SyncEventType = "success"  // clean run, no conflicts
	SyncEventConflict SyncEventType = "conflict" // succeeded but had ≥1 conflict
	SyncEventError    SyncEventType = "error"    // run failed with an error
)

// SyncEvent records the outcome of a single sync run.
type SyncEvent struct {
	Timestamp       time.Time     `json:"timestamp"`
	Type            SyncEventType `json:"type"`
	ProfileName     string        `json:"profile,omitempty"`
	RegisteredNotes int           `json:"registered,omitempty"`
	UpdatedNotes    int           `json:"updated,omitempty"`
	Conflicts       int           `json:"conflicts,omitempty"`
	ErrorMsg        string        `json:"error,omitempty"`
	DurationMs      int64         `json:"duration_ms,omitempty"`
}

const (
	eventLogFileName = "sync-events.jsonl"
	eventLogMaxLines = 200
)

func eventLogPath(root string) string {
	return filepath.Join(SyncDir(root), eventLogFileName)
}

// AppendSyncEvent appends a SyncEvent to the workspace event log.
// The log is pruned to the last eventLogMaxLines entries on each write.
func AppendSyncEvent(root string, event SyncEvent) error {
	path := eventLogPath(root)
	existing, err := readEventLines(path)
	if err != nil {
		existing = nil
	}
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	existing = append(existing, string(line))
	if len(existing) > eventLogMaxLines {
		existing = existing[len(existing)-eventLogMaxLines:]
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(existing, "\n")+"\n"), 0o644)
}

// LoadSyncEvents reads up to limit events from the log, newest first.
// If limit <= 0, all events are returned.
func LoadSyncEvents(root string, limit int) ([]SyncEvent, error) {
	path := eventLogPath(root)
	lines, err := readEventLines(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	// Reverse so newest is first.
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	if limit > 0 && len(lines) > limit {
		lines = lines[:limit]
	}
	events := make([]SyncEvent, 0, len(lines))
	for _, line := range lines {
		var e SyncEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events, nil
}

func readEventLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}
