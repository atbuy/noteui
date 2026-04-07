package notes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	HistoryDirName     = ".noteui-history"
	MaxVersionsPerNote = 50
)

// HistoryEntry describes one saved version of a note.
type HistoryEntry struct {
	ID        string // filename stem, e.g. "20260407-150405-a1b2c3d4"
	Timestamp time.Time
	Hash      string // first 8 hex chars of SHA-256
	Size      int
	FirstLine string // first non-blank line, truncated for display
}

// historyDir returns the per-note directory inside .noteui-history/.
// Slashes in relPath are replaced with "__" so the entire path maps to
// a single directory name.
func historyDir(root, relPath string) string {
	escaped := strings.ReplaceAll(filepath.ToSlash(relPath), "/", "__")
	return filepath.Join(root, HistoryDirName, escaped)
}

func versionPath(root, relPath, versionID string) string {
	return filepath.Join(historyDir(root, relPath), versionID+".md")
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// SaveVersion saves content as a new history version for the note at relPath.
// It is a no-op when content matches the most recent existing version.
// After saving, it prunes old versions so at most MaxVersionsPerNote are kept.
func SaveVersion(root, relPath, content string) error {
	dir := historyDir(root, relPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating history dir: %w", err)
	}

	hash := contentHash(content)

	// Skip if identical to the most recent version.
	existing, _ := Versions(root, relPath)
	if len(existing) > 0 && existing[0].Hash == hash[:8] {
		return nil
	}

	ts := time.Now().UTC()
	id := ts.Format("20060102-150405") + "-" + hash[:8]
	path := versionPath(root, relPath, id)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing history version: %w", err)
	}

	pruneVersions(dir, MaxVersionsPerNote)
	return nil
}

// Versions returns all history entries for the note at relPath, newest first.
func Versions(root, relPath string) ([]HistoryEntry, error) {
	dir := historyDir(root, relPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	out := make([]HistoryEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		stem := strings.TrimSuffix(e.Name(), ".md")
		ts, hash, ok := parseVersionID(stem)
		if !ok {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		firstLine := versionFirstLine(filepath.Join(dir, e.Name()))
		out = append(out, HistoryEntry{
			ID:        stem,
			Timestamp: ts,
			Hash:      hash,
			Size:      int(info.Size()),
			FirstLine: firstLine,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.After(out[j].Timestamp)
	})
	return out, nil
}

// VersionContent returns the stored content for a specific version ID.
func VersionContent(root, relPath, versionID string) (string, error) {
	path := versionPath(root, relPath, versionID)
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading history version: %w", err)
	}
	return string(b), nil
}

// RestoreVersion atomically overwrites the note at notePath with the content
// of the named history version. Before overwriting, the current content of
// notePath is saved as a new history version so the restore itself is undoable.
func RestoreVersion(root, notePath, relPath, versionID string) error {
	// Save current content before overwriting so the user can undo the restore.
	if current, err := ReadAll(notePath); err == nil {
		_ = SaveVersion(root, relPath, current)
	}

	content, err := VersionContent(root, relPath, versionID)
	if err != nil {
		return err
	}

	if err := atomicWriteFile(notePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("restoring version: %w", err)
	}
	return nil
}

// parseVersionID parses a version stem like "20260407-150405-a1b2c3d4"
// into a time.Time and the 8-char hash suffix.
func parseVersionID(id string) (time.Time, string, bool) {
	// Expected format: YYYYMMDD-HHMMSS-<hash8>
	parts := strings.SplitN(id, "-", 3)
	if len(parts) != 3 {
		return time.Time{}, "", false
	}
	ts, err := time.Parse("20060102-150405", parts[0]+"-"+parts[1])
	if err != nil {
		return time.Time{}, "", false
	}
	return ts.UTC(), parts[2], true
}

// versionFirstLine reads the first non-blank line from a history file for
// display in the rollback modal.
func versionFirstLine(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "---" {
			continue
		}
		// Show "encrypted" for encrypted blobs instead of garbled base64.
		if strings.HasPrefix(line, "encrypted: true") {
			return "encrypted"
		}
		const maxLen = 60
		if len(line) > maxLen {
			return line[:maxLen] + "…"
		}
		return line
	}
	return ""
}

// pruneVersions deletes the oldest versions in dir so that at most keep files remain.
func pruneVersions(dir string, keep int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type vf struct {
		name string
		ts   time.Time
	}
	var files []vf
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		stem := strings.TrimSuffix(e.Name(), ".md")
		ts, _, ok := parseVersionID(stem)
		if !ok {
			continue
		}
		files = append(files, vf{name: e.Name(), ts: ts})
	}
	if len(files) <= keep {
		return
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ts.After(files[j].ts)
	})
	for _, f := range files[keep:] {
		_ = os.Remove(filepath.Join(dir, f.name))
	}
}
