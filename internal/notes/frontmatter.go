package notes

import (
	"os"
	"strings"
)

type FrontMatter map[string]string

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
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
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

func normalizeFrontMatterKey(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "_", "-")
	return s
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
	raw, err := ReadAll(path)
	if err != nil {
		return err
	}

	normalizedRaw := strings.ReplaceAll(raw, "\r\n", "\n")
	fm, body, err := ParseFrontMatter(normalizedRaw)
	if err != nil {
		return err
	}

	existing := ParseTags(fm)
	merged := mergeTags(existing, tags)
	if len(merged) == 0 {
		return nil
	}

	line := "tags: " + strings.Join(merged, ", ")
	updated := setFrontMatterField(normalizedRaw, body, "tags", line)
	return os.WriteFile(path, []byte(updated), 0o644)
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
	var lines []string
	for _, line := range strings.Split(block, "\n") {
		if idx := strings.Index(line, ":"); idx >= 0 {
			k := normalizeFrontMatterKey(line[:idx])
			if k == normalizedKey {
				continue
			}
		}
		lines = append(lines, line)
	}
	lines = append(lines, fieldLine)
	return "---\n" + strings.Join(lines, "\n") + "\n---\n" + body
}
