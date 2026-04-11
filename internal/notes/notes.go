// Package notes provides note discovery, creation, mutation, and metadata parsing.
package notes

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

const PreviewBytes = 16 * 1024

type Note struct {
	Root      string
	Path      string
	RelPath   string
	Name      string
	TitleText string
	Category  string
	Preview   string
	ModTime   time.Time
	Tags      []string
	Encrypted bool
	SyncClass string
}

func TempRoot(root string) string {
	return filepath.Join(root, ".tmp")
}

const TemplatesDirName = ".templates"

func TemplatesRoot(root string) string {
	return filepath.Join(root, TemplatesDirName)
}

// Template describes a user-defined note template stored in .templates/.
type Template struct {
	Name    string // display name derived from the filename
	RelPath string // relative path inside .templates/
	Path    string // absolute path
}

// DiscoverTemplates returns all note files inside <root>/.templates/, sorted
// alphabetically by RelPath. Returns an empty slice with no error when the
// directory does not exist.
func DiscoverTemplates(root string) ([]Template, error) {
	templatesRoot := TemplatesRoot(root)

	if _, err := os.Stat(templatesRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Template{}, nil
		}
		return nil, err
	}

	var out []Template

	err := filepath.WalkDir(templatesRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != templatesRoot {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		if !IsNoteFile(path) {
			return nil
		}

		rel, err := filepath.Rel(templatesRoot, path)
		if err != nil {
			return err
		}

		out = append(out, Template{
			Name:    fallbackTitleFromFilename(filepath.Base(path)),
			RelPath: rel,
			Path:    path,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].RelPath < out[j].RelPath
	})

	return out, nil
}

// ApplyTemplateVars replaces template placeholders in content with values
// derived from t. Supported variables:
//
//	{{date}}  - current date in YYYY-MM-DD format
//	{{time}}  - current time in HH:MM (24-hour) format
//	{{title}} - replaced with empty string; the note title is set by the editor
func ApplyTemplateVars(content string, t time.Time) string {
	return strings.NewReplacer(
		"{{date}}", t.Format("2006-01-02"),
		"{{time}}", t.Format("15:04"),
		"{{title}}", "",
	).Replace(content)
}

// CreateTemplate creates a new blank template file inside <root>/.templates/.
// The directory is created if it does not exist. Returns the absolute path of
// the new file.
func CreateTemplate(root string) (string, error) {
	dir := TemplatesRoot(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := ".new-" + time.Now().Format("20060102-150405") + ".md"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("# Template title\n\n"), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// CreateNoteFromTemplate creates a new note pre-filled with the content of
// templatePath after applying variable substitution. The created file uses the
// same .new-* naming convention as CreateNote.
func CreateNoteFromTemplate(root, relDir, templatePath string) (string, error) {
	raw, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}
	content := ApplyTemplateVars(string(raw), time.Now())

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

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}

	return path, nil
}

func DiscoverTemporary(root string) ([]Note, error) {
	tempRoot := TempRoot(root)

	info, err := os.Stat(tempRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if mkErr := os.MkdirAll(tempRoot, 0o755); mkErr != nil {
				return nil, mkErr
			}
			return []Note{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("temporary notes root is not a directory: %s", tempRoot)
	}

	var out []Note

	err = filepath.WalkDir(tempRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != tempRoot {
				return filepath.SkipDir
			}
			return nil
		}

		if !IsNoteFile(path) {
			return nil
		}

		rel, err := filepath.Rel(tempRoot, path)
		if err != nil {
			return err
		}

		category := filepath.Dir(rel)
		if category == "." {
			category = ""
		}

		preview, _ := ReadFull(path)
		title := ExtractTitle(preview)
		if title == "" {
			title = fallbackTitleFromFilename(filepath.Base(path))
		}

		fm, _, _ := ParseFrontMatter(preview)
		tags := ParseTags(fm)
		encrypted := FrontMatterBool(fm, "encrypted")

		info, err := d.Info()
		if err != nil {
			return err
		}

		out = append(out, Note{
			Root:      tempRoot,
			Path:      path,
			RelPath:   rel,
			Name:      filepath.Base(path),
			TitleText: title,
			Category:  category,
			Preview:   preview,
			ModTime:   info.ModTime(),
			Tags:      tags,
			Encrypted: encrypted,
			SyncClass: SyncClassLocal,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].RelPath < out[j].RelPath
	})

	return out, nil
}

func CreateTemporaryNote(root string) (string, error) {
	return CreateNote(TempRoot(root), "")
}

func (n Note) Title() string {
	if strings.TrimSpace(n.TitleText) != "" {
		return n.TitleText
	}
	return n.Name
}

func (n Note) Description() string { return n.RelPath }
func (n Note) FilterValue() string {
	return n.Title() + " " + n.Name + " " + n.RelPath + " " + n.Preview + " " + strings.Join(
		n.Tags,
		" ",
	)
}

func Discover(root string) ([]Note, error) {
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if mkErr := os.MkdirAll(root, 0o755); mkErr != nil {
				return nil, mkErr
			}
			return []Note{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("notes root is not a directory: %s", root)
	}

	var out []Note

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}

		if !IsNoteFile(path) {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		category := filepath.Dir(rel)
		if category == "." {
			category = "uncategorized"
		}

		preview, _ := ReadFull(path)
		title := ExtractTitle(preview)
		if title == "" {
			title = fallbackTitleFromFilename(filepath.Base(path))
		}

		fm, _, _ := ParseFrontMatter(preview)
		tags := ParseTags(fm)
		encrypted := FrontMatterBool(fm, "encrypted")
		syncClass := ParseSyncClass(fm)

		info, err := d.Info()
		if err != nil {
			return err
		}

		out = append(out, Note{
			Root:      root,
			Path:      path,
			RelPath:   rel,
			Name:      filepath.Base(path),
			TitleText: title,
			Category:  category,
			Preview:   preview,
			ModTime:   info.ModTime(),
			Tags:      tags,
			Encrypted: encrypted,
			SyncClass: syncClass,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].RelPath < out[j].RelPath
	})

	return out, nil
}

func ReadPreview(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, PreviewBytes)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	text := string(buf[:n])
	text = strings.ReplaceAll(text, "\t", "    ")
	return strings.TrimSpace(text), nil
}

func ReadFull(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text := string(b)
	text = strings.ReplaceAll(text, "\t", "    ")
	return strings.TrimSpace(text), nil
}

func ReadAll(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func IsNoteFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt", ".org", ".norg":
		return true
	default:
		return false
	}
}

// AppendCapture appends a timestamped bullet entry to the note at relPath
// inside root. If the file does not exist it is created with a heading derived
// from the filename. The write is atomic so the file is never left partially
// written.
func AppendCapture(root, relPath, text string) error {
	path := filepath.Join(root, relPath)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var raw string
	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		title := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
		raw = "# " + title + "\n"
	} else if err != nil {
		return err
	} else {
		raw = string(existing)
	}

	if !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	raw += "- [" + time.Now().Format("2006-01-02 15:04") + "] " + text + "\n"
	return atomicWriteFile(path, []byte(raw), 0o644)
}

// atomicWriteFile writes data to path by first writing to a sibling temp file
// in the same directory and then renaming it into place. This ensures the
// target is never left in a partially-written state if the process is
// interrupted or the disk runs out of space mid-write.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".noteui-tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	// Always remove the temp file if we never reach the rename.
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
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

	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		return "", err
	}

	return path, nil
}

func DeleteNote(path string) (TrashResult, error) {
	return TrashPath(path)
}

func MoveNote(root, oldRelPath, newRelPath string) error {
	return MoveNoteBetweenRoots(root, oldRelPath, root, newRelPath)
}

func MoveNoteBetweenRoots(srcRoot, oldRelPath, dstRoot, newRelPath string) error {
	oldRelPath = cleanRelativePath(oldRelPath, true)
	newRelPath = cleanRelativePath(newRelPath, true)

	if oldRelPath == "" || newRelPath == "" {
		return errors.New("note path cannot be empty")
	}

	oldPath := filepath.Join(srcRoot, oldRelPath)
	newPath := filepath.Join(dstRoot, newRelPath)
	if oldPath == newPath {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return err
	}

	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("target already exists: %s", newRelPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.Rename(oldPath, newPath)
}

func ExtractTitle(content string) string {
	content = StripFrontMatter(content)

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(line, "# "))
			if title != "" {
				return title
			}
		}
	}
	return ""
}

func fallbackTitleFromFilename(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = strings.TrimSpace(strings.ReplaceAll(base, "-", " "))
	if base == "" {
		return "Untitled"
	}
	return base
}

func RenameFromTitle(path string) (string, bool, error) {
	content, err := ReadAll(path)
	if err != nil {
		return "", false, err
	}

	// If the file is empty and still has the temp name, delete it.
	if strings.TrimSpace(content) == "" && isTempNoteName(filepath.Base(path)) {
		_ = os.Remove(path)
		return "", false, nil
	}

	title := ExtractTitleOrFirstLine(content)
	if title == "" {
		return path, false, nil
	}

	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	if ext == "" {
		ext = ".md"
	}

	baseSlug := Slugify(title)
	if baseSlug == "" {
		baseSlug = "untitled"
	}

	target := uniquePath(dir, baseSlug, ext, path)
	if target == path {
		return path, false, nil
	}

	if err := os.Rename(path, target); err != nil {
		return "", false, err
	}
	return target, true, nil
}

func isTempNoteName(name string) bool {
	return strings.HasPrefix(name, ".new-")
}

func RenameNoteTitle(path, newTitle string) (string, bool, error) {
	newTitle = strings.TrimSpace(newTitle)
	if newTitle == "" {
		return path, false, errors.New("title cannot be empty")
	}

	content, err := ReadAll(path)
	if err != nil {
		return "", false, err
	}

	updated := replaceOrInsertRootTitle(content, newTitle)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return "", false, err
	}

	return RenameFromTitle(path)
}

func ExtractTitleOrFirstLine(content string) string {
	content = StripFrontMatter(content)

	firstNonEmpty := ""
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(line, "# "))
			if title != "" {
				return title
			}
		}
		if firstNonEmpty == "" {
			firstNonEmpty = strings.TrimLeft(line, "#~*`->")
			firstNonEmpty = strings.TrimSpace(firstNonEmpty)
		}
	}

	return firstNonEmpty
}

func Slugify(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}

	var b strings.Builder
	lastDash := false

	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || strings.ContainsRune("-_./\\:+&", r):
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		default:
			// skip punctuation and unsafe chars
		}
	}

	out := strings.Trim(b.String(), "-")
	return out
}

func uniquePath(dir, slug, ext, currentPath string) string {
	candidate := filepath.Join(dir, slug+ext)
	if candidate == currentPath {
		return candidate
	}

	if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
		return candidate
	}

	for i := 2; ; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s-%d%s", slug, i, ext))
		if candidate == currentPath {
			return candidate
		}
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
}

func cleanRelativePath(rel string, keepExt bool) string {
	rel = filepath.Clean(strings.TrimSpace(rel))
	if rel == "." {
		return ""
	}
	rel = strings.TrimPrefix(rel, string(filepath.Separator))
	for strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		rel = strings.TrimPrefix(rel, ".."+string(filepath.Separator))
		if rel == ".." {
			rel = ""
			break
		}
	}
	if keepExt && rel != "" && filepath.Ext(rel) == "" {
		rel += ".md"
	}
	return rel
}

func replaceOrInsertRootTitle(content, title string) string {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			lines[i] = "# " + title
			return strings.Join(lines, "\n")
		}
	}

	// No root heading found, prepend one.
	if strings.TrimSpace(content) == "" {
		return "# " + title + "\n\n"
	}
	return "# " + title + "\n\n" + content
}

func WordCount(content string) int {
	content = StripFrontMatter(content)
	return len(strings.Fields(content))
}

func ReadingTimeMinutes(wordCount int) int {
	// Average adult reading speed is ~200-250 words per minute.
	const wordsPerMinute = 225
	if wordCount <= 0 {
		return 0
	}
	return max(1, (wordCount+wordsPerMinute-1)/wordsPerMinute)
}

func CreateTodoNote(root, relDir string) (string, error) {
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
	if err := os.WriteFile(path, []byte(template), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func ToggleTodoLine(path string, lineIdx int) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
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
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func AddTodoItem(path, text string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	raw := string(content)
	if !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	raw += "- [ ] " + text + "\n"
	return os.WriteFile(path, []byte(raw), 0o644)
}

func DeleteTodoLine(path string, lineIdx int) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("line index %d out of range", lineIdx)
	}
	lines = append(lines[:lineIdx], lines[lineIdx+1:]...)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func EditTodoLine(path string, lineIdx int, newText string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
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
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func UpdateTodoPriority(path string, lineIdx int, priority string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("line index %d out of range", lineIdx)
	}
	line := lines[lineIdx]
	trimmed := strings.TrimLeft(line, " 	")
	indent := line[:len(line)-len(trimmed)]

	if !strings.HasPrefix(trimmed, "- [ ] ") && !strings.HasPrefix(trimmed, "- [x] ") && !strings.HasPrefix(trimmed, "- [X] ") {
		return fmt.Errorf("line %d is not a todo item", lineIdx)
	}

	priority = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(priority), "p"))
	if priority != "" {
		if _, ok := parseTodoPriorityToken("[p" + priority + "]"); !ok {
			return fmt.Errorf("invalid priority %q: expected a positive number", priority)
		}
	}

	body := trimmed[6:]
	fields := strings.Fields(body)
	kept := make([]string, 0, len(fields)+1)
	for _, field := range fields {
		if _, ok := parseTodoPriorityToken(field); ok {
			continue
		}
		kept = append(kept, field)
	}
	if priority != "" {
		kept = append(kept, "[p"+priority+"]")
	}

	lines[lineIdx] = indent + trimmed[:6] + strings.TrimSpace(strings.Join(kept, " "))
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func UpdateTodoDueDate(path string, lineIdx int, dueDate string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("line index %d out of range", lineIdx)
	}
	line := lines[lineIdx]
	trimmed := strings.TrimLeft(line, " 	")
	indent := line[:len(line)-len(trimmed)]

	if !strings.HasPrefix(trimmed, "- [ ] ") && !strings.HasPrefix(trimmed, "- [x] ") && !strings.HasPrefix(trimmed, "- [X] ") {
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
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}
