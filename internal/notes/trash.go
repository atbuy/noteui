package notes

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TrashResult records where a trashed item ended up, enabling in-app restore.
type TrashResult struct {
	OriginalPath  string // absolute path before trashing
	TrashFilePath string // path inside Trash/files/
	TrashInfoPath string // path of the .trashinfo metadata file
}

func TrashPath(path string) (TrashResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return TrashResult{}, errors.New("path cannot be empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return TrashResult{}, err
	}

	trashRoot, err := userTrashRoot()
	if err != nil {
		return TrashResult{}, err
	}

	filesDir := filepath.Join(trashRoot, "files")
	infoDir := filepath.Join(trashRoot, "info")

	if err := os.MkdirAll(filesDir, 0o700); err != nil {
		return TrashResult{}, err
	}
	if err := os.MkdirAll(infoDir, 0o700); err != nil {
		return TrashResult{}, err
	}

	base := filepath.Base(absPath)
	name := uniqueTrashName(filesDir, infoDir, base)

	targetPath := filepath.Join(filesDir, name)
	infoPath := filepath.Join(infoDir, name+".trashinfo")

	infoContent := buildTrashInfo(absPath)

	if err := os.WriteFile(infoPath, []byte(infoContent), 0o600); err != nil {
		return TrashResult{}, err
	}

	if err := os.Rename(absPath, targetPath); err != nil {
		_ = os.Remove(infoPath)
		return TrashResult{}, err
	}

	return TrashResult{
		OriginalPath:  absPath,
		TrashFilePath: targetPath,
		TrashInfoPath: infoPath,
	}, nil
}

// RestoreFromTrash moves a previously trashed item back to its original path.
// It returns an error if the original path is already occupied.
func RestoreFromTrash(result TrashResult) error {
	if strings.TrimSpace(result.OriginalPath) == "" || strings.TrimSpace(result.TrashFilePath) == "" {
		return errors.New("restore failed: missing path information")
	}
	if pathExists(result.OriginalPath) {
		return fmt.Errorf("restore failed: %q already exists", result.OriginalPath)
	}
	if err := os.MkdirAll(filepath.Dir(result.OriginalPath), 0o755); err != nil {
		return err
	}
	if err := os.Rename(result.TrashFilePath, result.OriginalPath); err != nil {
		return err
	}
	if result.TrashInfoPath != "" {
		_ = os.Remove(result.TrashInfoPath)
	}
	return nil
}

func userTrashRoot() (string, error) {
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if strings.TrimSpace(xdgDataHome) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdgDataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(xdgDataHome, "Trash"), nil
}

func uniqueTrashName(filesDir, infoDir, base string) string {
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	candidate := base
	i := 2

	for {
		fileExists := pathExists(filepath.Join(filesDir, candidate))
		infoExists := pathExists(filepath.Join(infoDir, candidate+".trashinfo"))
		if !fileExists && !infoExists {
			return candidate
		}

		if ext == "" {
			candidate = fmt.Sprintf("%s-%d", stem, i)
		} else {
			candidate = fmt.Sprintf("%s-%d%s", stem, i, ext)
		}
		i++
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func buildTrashInfo(absPath string) string {
	u := url.URL{Path: filepath.ToSlash(absPath)}
	deletionDate := time.Now().Format("2006-01-02T15:04:05")

	return strings.Join([]string{
		"[Trash Info]",
		"Path=" + u.String(),
		"DeletionDate=" + deletionDate,
		"",
	}, "\n")
}
