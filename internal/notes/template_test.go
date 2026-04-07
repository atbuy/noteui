package notes

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDiscoverTemplatesEmptyWhenNoDirExists(t *testing.T) {
	root := t.TempDir()
	tmpl, err := DiscoverTemplates(root)
	require.NoError(t, err)
	require.Empty(t, tmpl)
}

func TestDiscoverTemplatesReturnsFilesAlphabetically(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, TemplatesDirName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "standup.md"), []byte("# Standup\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "meeting.md"), []byte("# Meeting\n"), 0o644))

	tmpl, err := DiscoverTemplates(root)
	require.NoError(t, err)
	require.Len(t, tmpl, 2)
	require.Equal(t, "meeting.md", tmpl[0].RelPath)
	require.Equal(t, "standup.md", tmpl[1].RelPath)
	require.Equal(t, "meeting", tmpl[0].Name)
	require.Equal(t, "standup", tmpl[1].Name)
}

func TestDiscoverTemplatesIgnoresDotFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, TemplatesDirName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden.md"), []byte("hidden"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "visible.md"), []byte("visible"), 0o644))

	tmpl, err := DiscoverTemplates(root)
	require.NoError(t, err)
	require.Len(t, tmpl, 1)
	require.Equal(t, "visible.md", tmpl[0].RelPath)
}

func TestApplyTemplateVarsSubstitutesAll(t *testing.T) {
	fixed := time.Date(2026, 4, 8, 14, 30, 0, 0, time.UTC)
	input := "Date: {{date}}\nTime: {{time}}\nTitle: {{title}}"
	got := ApplyTemplateVars(input, fixed)
	require.Equal(t, "Date: 2026-04-08\nTime: 14:30\nTitle: ", got)
}

func TestApplyTemplateVarsNoVarsUnchanged(t *testing.T) {
	content := "# My Note\n\nSome body text."
	got := ApplyTemplateVars(content, time.Now())
	require.Equal(t, content, got)
}

func TestCreateTemplateCreatesFileInTemplatesDir(t *testing.T) {
	root := t.TempDir()
	path, err := CreateTemplate(root)
	require.NoError(t, err)
	require.NotEmpty(t, path)

	// File must be inside .templates/.
	templatesDir := filepath.Join(root, TemplatesDirName)
	require.True(t, len(path) > len(templatesDir), "path should be inside templates dir")
	require.Equal(t, templatesDir, filepath.Dir(path))

	// File must have non-empty default content.
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotEmpty(t, content)
}

func TestCreateTemplateCreatesDirectoryIfMissing(t *testing.T) {
	root := t.TempDir()
	// Verify .templates/ does not exist yet.
	_, err := os.Stat(filepath.Join(root, TemplatesDirName))
	require.True(t, os.IsNotExist(err))

	_, err = CreateTemplate(root)
	require.NoError(t, err)

	// Now .templates/ should exist.
	info, err := os.Stat(filepath.Join(root, TemplatesDirName))
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestCreateNoteFromTemplateWritesSubstitutedContent(t *testing.T) {
	root := t.TempDir()

	// Create a template file with a date variable.
	tmplDir := filepath.Join(root, TemplatesDirName)
	require.NoError(t, os.MkdirAll(tmplDir, 0o755))
	tmplPath := filepath.Join(tmplDir, "meeting.md")
	require.NoError(t, os.WriteFile(tmplPath, []byte("# Meeting\n\nDate: {{date}}\n"), 0o644))

	path, err := CreateNoteFromTemplate(root, "", tmplPath)
	require.NoError(t, err)
	require.NotEmpty(t, path)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(content), "# Meeting")
	require.NotContains(t, string(content), "{{date}}")
	// The substituted date should match today's date format.
	today := time.Now().Format("2006-01-02")
	require.Contains(t, string(content), today)
}
