package notes

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Category struct {
	Name    string
	RelPath string
	Depth   int
}

func DiscoverCategories(root string) ([]Category, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}

	var out []Category

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path == root {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		out = append(out, Category{
			Name:    d.Name(),
			RelPath: rel,
			Depth:   strings.Count(rel, string(filepath.Separator)),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].RelPath < out[j].RelPath
	})

	// Add a virtual root category.
	out = append([]Category{{
		Name:    "All notes",
		RelPath: "",
		Depth:   0,
	}}, out...)

	return out, nil
}

func CreateCategory(root, relPath string) error {
	relPath = filepath.Clean(strings.TrimSpace(relPath))
	if relPath == "" || relPath == "." {
		return errors.New("category name cannot be empty")
	}
	if strings.HasPrefix(relPath, "..") {
		return errors.New("category must stay inside notes root")
	}

	return os.MkdirAll(filepath.Join(root, relPath), 0o755)
}

func DeleteCategory(root, relPath string) error {
	relPath = filepath.Clean(strings.TrimSpace(relPath))
	if relPath == "" || relPath == "." {
		return errors.New("cannot delete root category")
	}
	if strings.HasPrefix(relPath, "..") {
		return errors.New("category must stay inside notes root")
	}

	return TrashPath(filepath.Join(root, relPath))
}

func MoveCategory(root, oldRelPath, newRelPath string) error {
	oldRelPath = filepath.Clean(strings.TrimSpace(oldRelPath))
	newRelPath = filepath.Clean(strings.TrimSpace(newRelPath))

	if oldRelPath == "" || oldRelPath == "." {
		return errors.New("cannot move root category")
	}
	if newRelPath == "" || newRelPath == "." {
		return errors.New("target category cannot be root")
	}
	if strings.HasPrefix(oldRelPath, "..") || strings.HasPrefix(newRelPath, "..") {
		return errors.New("category path must stay inside notes root")
	}
	if oldRelPath == newRelPath {
		return nil
	}
	if newRelPath == oldRelPath ||
		strings.HasPrefix(newRelPath, oldRelPath+string(filepath.Separator)) {
		return errors.New("cannot move a category inside itself")
	}

	oldPath := filepath.Join(root, oldRelPath)
	newPath := filepath.Join(root, newRelPath)

	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return err
	}

	if _, err := os.Stat(newPath); err == nil {
		return errors.New("target category already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.Rename(oldPath, newPath)
}
