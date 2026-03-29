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
)

const PreviewBytes = 16 * 1024

type Note struct {
	Root     string
	Path     string
	RelPath  string
	Name     string
	Category string
	Preview  string
	ModTime  time.Time
}

func (n Note) Title() string       { return n.Name }
func (n Note) Description() string { return n.RelPath }
func (n Note) FilterValue() string { return n.Name + " " + n.RelPath + " " + n.Preview }

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

		preview, _ := ReadPreview(path)

		info, err := d.Info()
		if err != nil {
			return err
		}

		out = append(out, Note{
			Root:     root,
			Path:     path,
			RelPath:  rel,
			Name:     filepath.Base(path),
			Category: category,
			Preview:  preview,
			ModTime:  info.ModTime(),
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
	defer f.Close()

	buf := make([]byte, PreviewBytes)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	text := string(buf[:n])
	text = strings.ReplaceAll(text, "\t", "    ")
	return strings.TrimSpace(text), nil
}

func IsNoteFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt", ".org", ".norg":
		return true
	default:
		return false
	}
}

func CreateInboxNote(root string) (string, error) {
	return CreateNote(root, "inbox")
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

	name := time.Now().Format("2006-01-02-150405") + ".md"
	path := filepath.Join(targetDir, name)

	content := "# New note\n\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}

	return path, nil
}
