package notes

import (
	"strings"

	"atbuy/noteui/internal/notes/meta"
)

type FrontMatter = meta.FrontMatter

const (
	SyncClassLocal  = meta.SyncClassLocal
	SyncClassSynced = meta.SyncClassSynced
	SyncClassShared = meta.SyncClassShared
)

func ParseFrontMatter(raw string) (FrontMatter, string, error) { return meta.ParseFrontMatter(raw) }
func FrontMatterBool(fm FrontMatter, key string) bool          { return meta.FrontMatterBool(fm, key) }
func FrontMatterString(fm FrontMatter, key string) string      { return meta.FrontMatterString(fm, key) }
func ParseSyncClass(fm FrontMatter) string                     { return meta.ParseSyncClass(fm) }
func NoteIsEncrypted(raw string) bool                          { return meta.NoteIsEncrypted(raw) }
func NoteIsPrivate(raw string) bool                            { return meta.NoteIsPrivate(raw) }
func StripFrontMatter(raw string) string                       { return meta.StripFrontMatter(raw) }
func ParseTags(fm FrontMatter) []string                        { return meta.ParseTags(fm) }
func AddTagsToNote(path string, tags []string) error           { return meta.AddTagsToNote(path, tags) }
func SetNoteSyncClass(path, syncClass string) error            { return meta.SetNoteSyncClass(path, syncClass) }
func ToggleNoteSyncClass(path string) (string, error)          { return meta.ToggleNoteSyncClass(path) }

func mergeTags(existing, incoming []string) []string {
	out := make([]string, 0, len(existing)+len(incoming))
	seen := make(map[string]bool, len(existing)+len(incoming))
	appendTag := func(tag string) {
		tag = normalizeTag(tag)
		if tag == "" {
			return
		}
		key := strings.ToLower(tag)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, tag)
	}
	for _, tag := range existing {
		appendTag(tag)
	}
	for _, tag := range incoming {
		appendTag(tag)
	}
	return out
}

func normalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.TrimPrefix(tag, "#")
	return strings.TrimSpace(tag)
}
