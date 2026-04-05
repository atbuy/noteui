package notes

import (
	"strings"
	"time"
)

type TodoMetadata struct {
	Priority int
	DueDate  string
	DueTime  time.Time
}

type TodoItem struct {
	Line        int
	Checked     bool
	Text        string
	DisplayText string
	Metadata    TodoMetadata
}

func ExtractTodoItems(raw string, openOnly bool) []TodoItem {
	normalizedRaw := strings.ReplaceAll(raw, "\r\n", "\n")
	body := StripFrontMatter(normalizedRaw)
	offset := todoBodyLineOffset(normalizedRaw, body)
	lines := strings.Split(body, "\n")
	items := make([]TodoItem, 0, len(lines))
	for idx, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		checked := false
		text := ""
		switch {
		case strings.HasPrefix(trimmed, "- [ ] "):
			text = trimmed[6:]
		case strings.HasPrefix(trimmed, "- [x] "), strings.HasPrefix(trimmed, "- [X] "):
			checked = true
			text = trimmed[6:]
		default:
			continue
		}
		if openOnly && checked {
			continue
		}
		display, metadata := ParseTodoMetadata(text)
		items = append(items, TodoItem{
			Line:        offset + idx,
			Checked:     checked,
			Text:        strings.TrimSpace(text),
			DisplayText: display,
			Metadata:    metadata,
		})
	}
	return items
}

func ParseTodoMetadata(text string) (string, TodoMetadata) {
	fields := strings.Fields(text)
	metadata := TodoMetadata{}
	displayFields := make([]string, 0, len(fields))
	for _, field := range fields {
		normalized := strings.ToLower(strings.TrimSpace(field))
		switch normalized {
		case "[p1]", "[p2]", "[p3]":
			if metadata.Priority == 0 {
				metadata.Priority = int(normalized[2] - '0')
			}
			continue
		}
		if strings.HasPrefix(normalized, "[due:") && strings.HasSuffix(normalized, "]") {
			rawDate := strings.TrimSuffix(strings.TrimPrefix(normalized, "[due:"), "]")
			if dueTime, err := time.Parse("2006-01-02", rawDate); err == nil {
				metadata.DueDate = rawDate
				metadata.DueTime = dueTime
				continue
			}
		}
		displayFields = append(displayFields, field)
	}
	display := strings.TrimSpace(strings.Join(displayFields, " "))
	if display == "" {
		display = strings.TrimSpace(text)
	}
	return display, metadata
}

func todoBodyLineOffset(raw, body string) int {
	if raw == body {
		return 0
	}
	prefixLen := len(raw) - len(body)
	if prefixLen <= 0 || prefixLen > len(raw) {
		return 0
	}
	return strings.Count(raw[:prefixLen], "\n")
}
