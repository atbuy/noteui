// Package meta provides note frontmatter, tag, sync-class, and wikilink helpers.
package meta

import (
	"errors"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type FrontMatter map[string]string

const (
	SyncClassLocal  = "local"
	SyncClassSynced = "synced"
	SyncClassShared = "shared"
)

var wikilinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// WikilinkURLPrefix is the synthetic URL prefix used to mark wikilink destinations.
const WikilinkURLPrefix = "#wikilink:"

func ParseFrontMatter(raw string) (FrontMatter, string, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	if !strings.HasPrefix(raw, "---\n") {
		return nil, raw, nil
	}

	rest := strings.TrimPrefix(raw, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return nil, raw, nil
	}

	block := rest[:end]
	body := rest[end+len("\n---\n"):]

	fm := make(FrontMatter)
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		key = normalizeFrontMatterKey(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}

		fm[key] = value
	}

	return fm, body, nil
}

func FrontMatterBool(fm FrontMatter, key string) bool {
	if len(fm) == 0 {
		return false
	}

	v, ok := fm[normalizeFrontMatterKey(key)]
	if !ok {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func FrontMatterString(fm FrontMatter, key string) string {
	if len(fm) == 0 {
		return ""
	}
	return strings.TrimSpace(fm[normalizeFrontMatterKey(key)])
}

func ParseSyncClass(fm FrontMatter) string {
	switch strings.ToLower(FrontMatterString(fm, "sync")) {
	case SyncClassSynced:
		return SyncClassSynced
	case SyncClassShared:
		return SyncClassShared
	default:
		return SyncClassLocal
	}
}

func NoteIsEncrypted(raw string) bool {
	fm, _, err := ParseFrontMatter(raw)
	if err != nil || len(fm) == 0 {
		return false
	}
	return FrontMatterBool(fm, "encrypted")
}

func NoteIsPrivate(raw string) bool {
	fm, _, err := ParseFrontMatter(raw)
	if err != nil || len(fm) == 0 {
		return false
	}
	return FrontMatterBool(fm, "private")
}

func StripFrontMatter(raw string) string {
	_, body, err := ParseFrontMatter(raw)
	if err != nil {
		return raw
	}
	return body
}

func ParseTags(fm FrontMatter) []string {
	raw, ok := fm[normalizeFrontMatterKey("tags")]
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func AddTagsToNote(path string, tags []string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	normalizedRaw := strings.ReplaceAll(string(raw), "\r\n", "\n")
	fm, body, err := ParseFrontMatter(normalizedRaw)
	if err != nil {
		return err
	}

	existing := ParseTags(fm)
	merged := mergeTags(existing, tags)
	if len(merged) == 0 {
		return nil
	}

	updated := setFrontMatterField(normalizedRaw, body, "tags", "tags: "+strings.Join(merged, ", "))
	return os.WriteFile(path, []byte(updated), 0o644)
}

func SetNoteSyncClass(path, syncClass string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	normalizedRaw := strings.ReplaceAll(string(raw), "\r\n", "\n")
	_, body, err := ParseFrontMatter(normalizedRaw)
	if err != nil {
		return err
	}

	value := SyncClassLocal
	switch strings.ToLower(strings.TrimSpace(syncClass)) {
	case SyncClassSynced:
		value = SyncClassSynced
	case SyncClassShared:
		value = SyncClassShared
	}

	updated := setFrontMatterField(normalizedRaw, body, "sync", "sync: "+value)
	return os.WriteFile(path, []byte(updated), 0o644)
}

func ToggleNoteSyncClass(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	fm, _, err := ParseFrontMatter(strings.ReplaceAll(string(raw), "\r\n", "\n"))
	if err != nil {
		return "", err
	}

	if ParseSyncClass(fm) == SyncClassShared {
		return SyncClassShared, errors.New("shared notes cannot be toggled; edit frontmatter directly to change sync class")
	}

	next := SyncClassSynced
	if ParseSyncClass(fm) == SyncClassSynced {
		next = SyncClassLocal
	}

	return next, SetNoteSyncClass(path, next)
}

// RewriteWikilinks replaces [[target]] patterns with [target](#wikilink:encoded)
// so the standard markdown parser treats them as links. The target is
// percent-encoded so the URL is valid.
func RewriteWikilinks(content string) string {
	return wikilinkRe.ReplaceAllStringFunc(content, func(match string) string {
		inner := match[2 : len(match)-2]
		encoded := url.PathEscape(inner)
		return "[" + inner + "](" + WikilinkURLPrefix + encoded + ")"
	})
}

func DecodeWikilinkTarget(encoded string) string {
	decoded, err := url.PathUnescape(encoded)
	if err != nil {
		return encoded
	}
	return decoded
}

func ExtractWikilinks(content string) []string {
	matches := wikilinkRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		target := m[1]
		if !seen[target] {
			seen[target] = true
			out = append(out, target)
		}
	}
	return out
}

func normalizeFrontMatterKey(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

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

func setFrontMatterField(raw, body, key, fieldLine string) string {
	normalizedKey := normalizeFrontMatterKey(key)
	if raw == body {
		return "---\n" + fieldLine + "\n---\n" + body
	}

	rest := strings.TrimPrefix(raw, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return "---\n" + fieldLine + "\n---\n" + raw
	}

	block := rest[:end]
	lines := make([]string, 0, strings.Count(block, "\n")+1)
	for _, line := range strings.Split(block, "\n") {
		if idx := strings.Index(line, ":"); idx >= 0 {
			if normalizeFrontMatterKey(line[:idx]) == normalizedKey {
				continue
			}
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	lines = append(lines, fieldLine)
	return "---\n" + strings.Join(lines, "\n") + "\n---\n" + body
}
