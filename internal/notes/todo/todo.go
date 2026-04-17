// Package todo provides todo parsing and file mutation helpers for note content.
package todo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"atbuy/noteui/internal/fsutil"
	"atbuy/noteui/internal/notes/meta"
)

type Metadata struct {
	Priority int
	DueDate  string
	DueTime  time.Time
}

type Item struct {
	Line        int
	Checked     bool
	Text        string
	DisplayText string
	Metadata    Metadata
}

func ExtractItems(raw string, openOnly bool) []Item {
	normalizedRaw := strings.ReplaceAll(raw, "\r\n", "\n")
	body := meta.StripFrontMatter(normalizedRaw)
	offset := bodyLineOffset(normalizedRaw, body)
	lines := strings.Split(body, "\n")
	items := make([]Item, 0, len(lines))
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
		display, metadata := ParseMetadata(text)
		items = append(items, Item{Line: offset + idx, Checked: checked, Text: strings.TrimSpace(text), DisplayText: display, Metadata: metadata})
	}
	return items
}

func ParseMetadata(text string) (string, Metadata) {
	fields := strings.Fields(text)
	metadata := Metadata{}
	displayFields := make([]string, 0, len(fields))
	for _, field := range fields {
		normalized := strings.ToLower(strings.TrimSpace(field))
		if priority, ok := ParsePriorityToken(normalized); ok {
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

func ParsePriorityToken(token string) (int, bool) {
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

func CreateNote(root, relDir string) (string, error) {
	relDir = strings.TrimSpace(relDir)
	if relDir == "." {
		relDir = ""
	}

	targetDir := root
	if relDir != "" {
		targetDir = filepath.Join(root, relDir)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}

	name := ".new-" + time.Now().Format("20060102-150405") + ".md"
	path := filepath.Join(targetDir, name)

	template := "# Todo\n\n- [ ] \n"
	if err := fsutil.WriteFileAtomic(path, []byte(template), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func ToggleLine(path string, lineIdx int) error {
	lines, err := readLines(path)
	if err != nil {
		return err
	}
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("line index %d out of range", lineIdx)
	}
	line := lines[lineIdx]
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]

	switch {
	case strings.HasPrefix(trimmed, "- [ ] "):
		lines[lineIdx] = indent + "- [x] " + trimmed[6:]
	case strings.HasPrefix(trimmed, "- [x] "), strings.HasPrefix(trimmed, "- [X] "):
		lines[lineIdx] = indent + "- [ ] " + trimmed[6:]
	default:
		return fmt.Errorf("line %d is not a todo item", lineIdx)
	}
	return writeLines(path, lines)
}

func AddItem(path, text string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	raw := string(content)
	if !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	raw += "- [ ] " + text + "\n"
	return fsutil.WriteFileAtomic(path, []byte(raw), 0o644)
}

func DeleteLine(path string, lineIdx int) error {
	lines, err := readLines(path)
	if err != nil {
		return err
	}
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("line index %d out of range", lineIdx)
	}
	lines = append(lines[:lineIdx], lines[lineIdx+1:]...)
	return writeLines(path, lines)
}

func EditLine(path string, lineIdx int, newText string) error {
	lines, err := readLines(path)
	if err != nil {
		return err
	}
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("line index %d out of range", lineIdx)
	}
	line := lines[lineIdx]
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]

	prefix := "- [ ] "
	if strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "- [X] ") {
		prefix = "- [x] "
	}
	lines[lineIdx] = indent + prefix + newText
	return writeLines(path, lines)
}

func UpdatePriority(path string, lineIdx int, priority string) error {
	lines, err := readLines(path)
	if err != nil {
		return err
	}
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("line index %d out of range", lineIdx)
	}
	line := lines[lineIdx]
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]

	if !isTodoLine(trimmed) {
		return fmt.Errorf("line %d is not a todo item", lineIdx)
	}

	priority = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(priority), "p"))
	if priority != "" {
		if _, ok := ParsePriorityToken("[p" + priority + "]"); !ok {
			return fmt.Errorf("invalid priority %q: expected a positive number", priority)
		}
	}

	body := trimmed[6:]
	fields := strings.Fields(body)
	kept := make([]string, 0, len(fields)+1)
	for _, field := range fields {
		if _, ok := ParsePriorityToken(field); ok {
			continue
		}
		kept = append(kept, field)
	}
	if priority != "" {
		kept = append(kept, "[p"+priority+"]")
	}

	lines[lineIdx] = indent + trimmed[:6] + strings.TrimSpace(strings.Join(kept, " "))
	return writeLines(path, lines)
}

func UpdateDueDate(path string, lineIdx int, dueDate string) error {
	lines, err := readLines(path)
	if err != nil {
		return err
	}
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("line index %d out of range", lineIdx)
	}
	line := lines[lineIdx]
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]

	if !isTodoLine(trimmed) {
		return fmt.Errorf("line %d is not a todo item", lineIdx)
	}

	dueDate = strings.TrimSpace(dueDate)
	if dueDate != "" {
		if _, err := time.Parse("2006-01-02", dueDate); err != nil {
			return fmt.Errorf("invalid due date %q: expected YYYY-MM-DD", dueDate)
		}
	}

	body := trimmed[6:]
	fields := strings.Fields(body)
	kept := make([]string, 0, len(fields)+1)
	for _, field := range fields {
		normalized := strings.ToLower(strings.TrimSpace(field))
		if strings.HasPrefix(normalized, "[due:") && strings.HasSuffix(normalized, "]") {
			continue
		}
		kept = append(kept, field)
	}
	if dueDate != "" {
		kept = append(kept, "[due:"+dueDate+"]")
	}

	lines[lineIdx] = indent + trimmed[:6] + strings.TrimSpace(strings.Join(kept, " "))
	return writeLines(path, lines)
}

func bodyLineOffset(raw, body string) int {
	if raw == body {
		return 0
	}
	prefixLen := len(raw) - len(body)
	if prefixLen <= 0 || prefixLen > len(raw) {
		return 0
	}
	return strings.Count(raw[:prefixLen], "\n")
}

func isTodoLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "- [ ] ") || strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "- [X] ")
}

func readLines(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

func writeLines(path string, lines []string) error {
	return fsutil.WriteFileAtomic(path, []byte(strings.Join(lines, "\n")), 0o644)
}
