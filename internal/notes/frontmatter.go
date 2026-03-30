package notes

import (
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
