// Package demo provides embedded sample notes for the --demo flag.
package demo

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:fixtures
var fixturesFS embed.FS

// Setup copies the embedded demo notes into a temporary directory and returns
// the directory path and a cleanup function. The caller must call cleanup when
// the session ends.
func Setup() (root string, cleanup func(), err error) {
	root, err = os.MkdirTemp("", "noteui-demo-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(root) }

	err = fs.WalkDir(fixturesFS, "fixtures", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel("fixtures", path)
		if relErr != nil {
			return relErr
		}
		dest := filepath.Join(root, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, readErr := fixturesFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(dest, data, 0o644)
	})
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return root, cleanup, nil
}
