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
		if priority, ok := parseTodoPriorityToken(normalized); ok {
			if metadata.Priority == 0 {
				metadata.Priority = priority
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

func parseTodoPriorityToken(token string) (int, bool) {
	token = strings.ToLower(strings.TrimSpace(token))
	if !strings.HasPrefix(token, "[p") || !strings.HasSuffix(token, "]") {
		return 0, false
	}
	digits := token[2 : len(token)-1]
	if digits == "" {
		return 0, false
	}
	priority := 0
	for _, r := range digits {
		if r < '0' || r > '9' {
			return 0, false
		}
		priority = priority*10 + int(r-'0')
	}
	if priority <= 0 {
		return 0, false
	}
	return priority, true
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
