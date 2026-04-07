package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
)

func SyncDir(root string) string    { return filepath.Join(root, SyncDirName) }
func ConfigPath(root string) string { return filepath.Join(SyncDir(root), "config.json") }
func PinsPath(root string) string   { return filepath.Join(SyncDir(root), "pins.json") }
func NoteRecordPath(root, noteID string) string {
	return filepath.Join(SyncDir(root), NotesDirName, noteID+".json")
}

func ConflictPath(root, noteID string) string {
	return filepath.Join(SyncDir(root), ConflictsDirName, noteID+".json")
}

func HasSyncProfile(cfg config.SyncConfig) bool { return strings.TrimSpace(cfg.DefaultProfile) != "" }

func ActiveProfile(cfg config.SyncConfig, root string) (config.SyncProfile, string, error) {
	name := strings.TrimSpace(cfg.DefaultProfile)
	if name == "" {
		return config.SyncProfile{}, "", errors.New("sync is not configured")
	}
	rootCfg, err := LoadRootConfig(root)
	if err == nil && strings.TrimSpace(rootCfg.Profile) != "" {
		name = rootCfg.Profile
	}
	profile, ok := cfg.Profiles[name]
	if !ok {
		return config.SyncProfile{}, "", errors.New("sync profile not found: " + name)
	}
	if strings.TrimSpace(profile.RemoteBin) == "" {
		profile.RemoteBin = DefaultRemoteBin
	}
	return profile, name, nil
}

func EnsureRootConfig(root string, cfg config.SyncConfig) (RootConfig, error) {
	existing, err := LoadRootConfig(root)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return RootConfig{}, err
	}
	name := strings.TrimSpace(cfg.DefaultProfile)
	if name == "" {
		return RootConfig{}, errors.New("sync is not configured")
	}
	out := RootConfig{SchemaVersion: SchemaVersion, ClientID: NewClientID(), Profile: name}
	return out, SaveRootConfig(root, out)
}

func LoadRootConfig(root string) (RootConfig, error) {
	var cfg RootConfig
	path := ConfigPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := unmarshalLocalSyncJSON(path, "root config", data, &cfg); err != nil {
		return RootConfig{}, err
	}
	return cfg, nil
}

func SaveRootConfig(root string, cfg RootConfig) error { return writeJSON(ConfigPath(root), cfg) }

func LoadNoteRecords(root string) (map[string]NoteRecord, error) {
	dir := filepath.Join(SyncDir(root), NotesDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]NoteRecord{}, nil
		}
		return nil, err
	}
	out := make(map[string]NoteRecord, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var rec NoteRecord
		if err := unmarshalLocalSyncJSON(path, "note record", data, &rec); err != nil {
			return nil, err
		}
		if rec.ID != "" {
			out[rec.ID] = rec
		}
	}
	return out, nil
}

func SaveNoteRecord(root string, rec NoteRecord) error {
	if strings.TrimSpace(rec.ID) == "" {
		return errors.New("note record id cannot be empty")
	}
	return writeJSON(NoteRecordPath(root, rec.ID), rec)
}

func DeleteNoteRecord(root, noteID string) error {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return nil
	}
	if err := os.Remove(NoteRecordPath(root, noteID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(ConflictPath(root, noteID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func DeleteConflictRecord(root, noteID string) error {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return nil
	}
	if err := os.Remove(ConflictPath(root, noteID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func SaveConflictRecord(root string, rec ConflictRecord) error {
	if strings.TrimSpace(rec.NoteID) == "" {
		return errors.New("conflict record note id cannot be empty")
	}
	return writeJSON(ConflictPath(root, rec.NoteID), rec)
}

func LoadPins(root string) (Pins, error) {
	var pins Pins
	path := PinsPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		return pins, err
	}
	if len(data) == 0 {
		return pins, nil
	}
	if err := unmarshalLocalSyncJSON(path, "pins", data, &pins); err != nil {
		return Pins{}, err
	}
	pins.PinnedNoteIDs = sortedUnique(pins.PinnedNoteIDs)
	pins.PinnedCategories = sortedUnique(pins.PinnedCategories)
	return pins, nil
}

func SavePins(root string, pins Pins) error {
	pins.PinnedNoteIDs = sortedUnique(pins.PinnedNoteIDs)
	pins.PinnedCategories = sortedUnique(pins.PinnedCategories)
	return writeJSON(PinsPath(root), pins)
}

func RemovePinnedNoteID(root, noteID string) error {
	pins, err := LoadPins(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	pins.PinnedNoteIDs = removeSortedUniqueString(pins.PinnedNoteIDs, noteID)
	return SavePins(root, pins)
}

func LoadPinnedRelPaths(root string) ([]string, []string, error) {
	pins, err := LoadPins(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	records, err := LoadNoteRecords(root)
	if err != nil {
		return nil, nil, err
	}
	var relPaths []string
	for _, id := range pins.PinnedNoteIDs {
		rec, ok := records[id]
		if ok && strings.TrimSpace(rec.RelPath) != "" {
			relPaths = append(relPaths, filepath.ToSlash(rec.RelPath))
		}
	}
	return sortedUnique(relPaths), sortedUnique(pins.PinnedCategories), nil
}

func SavePinsFromRelPaths(root string, currentNotes []notes.Note, pinnedNotes []string, pinnedCats []string) error {
	records, err := LoadNoteRecords(root)
	if err != nil {
		return err
	}
	idByRelPath := make(map[string]string, len(records))
	for _, rec := range records {
		if strings.TrimSpace(rec.RelPath) != "" {
			idByRelPath[filepath.ToSlash(rec.RelPath)] = rec.ID
		}
	}
	syncedPaths := make(map[string]bool, len(currentNotes))
	for _, note := range currentNotes {
		if note.SyncClass == notes.SyncClassSynced {
			syncedPaths[filepath.ToSlash(note.RelPath)] = true
		}
	}
	var ids []string
	for _, relPath := range pinnedNotes {
		relPath = filepath.ToSlash(strings.TrimSpace(relPath))
		if strings.HasPrefix(relPath, ".tmp/") || !syncedPaths[relPath] {
			continue
		}
		if id := strings.TrimSpace(idByRelPath[relPath]); id != "" {
			ids = append(ids, id)
		}
	}
	return SavePins(root, Pins{PinnedNoteIDs: ids, PinnedCategories: pinnedCats})
}

func MigratePinsFromState(root string, currentNotes []notes.Note, pinnedNotes []string, pinnedCats []string) error {
	if _, err := os.Stat(PinsPath(root)); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return SavePinsFromRelPaths(root, currentNotes, pinnedNotes, pinnedCats)
}

func HashContent(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func unmarshalLocalSyncJSON(path, label string, data []byte, out any) error {
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("invalid local sync %s JSON at %s: %w", label, path, err)
	}
	return nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func sortedUnique(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = filepath.ToSlash(strings.TrimSpace(item))
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func removeSortedUniqueString(items []string, target string) []string {
	target = strings.TrimSpace(target)
	if target == "" || len(items) == 0 {
		return items
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			continue
		}
		out = append(out, item)
	}
	return sortedUnique(out)
}
