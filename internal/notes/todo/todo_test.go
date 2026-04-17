package todo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/notes/meta"
)

func TestExtractItemsAndMetadata(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		"title: Tasks",
		"---",
		"",
		"- [ ] Plan sprint [p2] [due:2026-05-01]",
		"  - [x] Closed item [p1]",
		"not a todo line",
		"- [ ] Plain task",
	}, "\n")

	items := ExtractItems(raw, false)
	require.Len(t, items, 3)

	require.Equal(t, 4, items[0].Line)
	require.False(t, items[0].Checked)
	require.Equal(t, "Plan sprint [p2] [due:2026-05-01]", items[0].Text)
	require.Equal(t, "Plan sprint", items[0].DisplayText)
	require.Equal(t, 2, items[0].Metadata.Priority)
	require.Equal(t, "2026-05-01", items[0].Metadata.DueDate)

	require.True(t, items[1].Checked)
	require.Equal(t, "Closed item", items[1].DisplayText)

	openOnly := ExtractItems(raw, true)
	require.Len(t, openOnly, 2)
	require.False(t, openOnly[0].Checked)
	require.False(t, openOnly[1].Checked)

	display, metadata := ParseMetadata("ship release [p3] [due:2026-06-01] soon")
	require.Equal(t, "ship release soon", display)
	require.Equal(t, 3, metadata.Priority)
	require.Equal(t, "2026-06-01", metadata.DueDate)
}

func TestParsePriorityToken(t *testing.T) {
	tests := []struct {
		token    string
		priority int
		ok       bool
	}{
		{token: "[p1]", priority: 1, ok: true},
		{token: " [P12] ", priority: 12, ok: true},
		{token: "[p0]", ok: false},
		{token: "[px]", ok: false},
		{token: "p2", ok: false},
	}

	for _, tc := range tests {
		priority, ok := ParsePriorityToken(tc.token)
		require.Equal(t, tc.priority, priority, tc.token)
		require.Equal(t, tc.ok, ok, tc.token)
	}
}

func TestCreateNoteCreatesTemplate(t *testing.T) {
	root := t.TempDir()

	path, err := CreateNote(root, ".")
	require.NoError(t, err)
	require.DirExists(t, root)
	require.Equal(t, root, filepath.Dir(path))
	require.True(t, strings.HasPrefix(filepath.Base(path), ".new-"))
	require.True(t, strings.HasSuffix(filepath.Base(path), ".md"))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "# Todo\n\n- [ ] \n", string(content))

	subdirPath, err := CreateNote(root, "work")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "work"), filepath.Dir(subdirPath))
}

func TestToggleLine(t *testing.T) {
	path := writeTodoFile(t, strings.Join([]string{
		"- [ ] first",
		"- [x] second",
		"plain",
	}, "\n"))

	require.NoError(t, ToggleLine(path, 0))
	require.NoError(t, ToggleLine(path, 1))

	content := readTodoFile(t, path)
	require.Equal(t, strings.Join([]string{
		"- [x] first",
		"- [ ] second",
		"plain",
	}, "\n"), content)

	err := ToggleLine(path, 2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "is not a todo item")

	err = ToggleLine(path, 99)
	require.Error(t, err)
	require.Contains(t, err.Error(), "out of range")
}

func TestAddDeleteAndEditLine(t *testing.T) {
	t.Run("add item appends newline when missing", func(t *testing.T) {
		path := writeTodoFile(t, "# Todo\n\n- [ ] first")

		require.NoError(t, AddItem(path, "second"))

		require.Equal(t, "# Todo\n\n- [ ] first\n- [ ] second\n", readTodoFile(t, path))
	})

	t.Run("delete line removes the requested line", func(t *testing.T) {
		path := writeTodoFile(t, strings.Join([]string{
			"- [ ] first",
			"- [ ] second",
			"- [ ] third",
		}, "\n"))

		require.NoError(t, DeleteLine(path, 1))
		require.Equal(t, "- [ ] first\n- [ ] third", readTodoFile(t, path))
	})

	t.Run("edit line preserves state and indentation", func(t *testing.T) {
		path := writeTodoFile(t, "  - [x] old value\n")

		require.NoError(t, EditLine(path, 0, "new value"))
		require.Equal(t, "  - [x] new value\n", readTodoFile(t, path))
	})
}

func TestUpdatePriority(t *testing.T) {
	path := writeTodoFile(t, "- [ ] task [p1] [due:2026-05-01]\n")

	require.NoError(t, UpdatePriority(path, 0, "3"))
	require.Equal(t, "- [ ] task [due:2026-05-01] [p3]\n", readTodoFile(t, path))

	require.NoError(t, UpdatePriority(path, 0, ""))
	require.Equal(t, "- [ ] task [due:2026-05-01]\n", readTodoFile(t, path))

	err := UpdatePriority(path, 0, "0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid priority")

	plain := writeTodoFile(t, "plain\n")
	err = UpdatePriority(plain, 0, "2")
	require.Error(t, err)
	require.Contains(t, err.Error(), "is not a todo item")
}

func TestUpdateDueDate(t *testing.T) {
	path := writeTodoFile(t, "- [ ] task [p2] [due:2026-05-01]\n")

	require.NoError(t, UpdateDueDate(path, 0, "2026-06-10"))
	require.Equal(t, "- [ ] task [p2] [due:2026-06-10]\n", readTodoFile(t, path))

	require.NoError(t, UpdateDueDate(path, 0, ""))
	require.Equal(t, "- [ ] task [p2]\n", readTodoFile(t, path))

	err := UpdateDueDate(path, 0, "2026-02-30")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid due date")

	plain := writeTodoFile(t, "plain\n")
	err = UpdateDueDate(plain, 0, "2026-06-10")
	require.Error(t, err)
	require.Contains(t, err.Error(), "is not a todo item")
}

func TestTodoHelpers(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		"title: demo",
		"---",
		"",
		"- [ ] task",
	}, "\n")
	body := meta.StripFrontMatter(raw)
	require.Equal(t, 3, bodyLineOffset(raw, body))
	require.Equal(t, 0, bodyLineOffset(body, body))
	require.True(t, isTodoLine("- [X] task"))
	require.False(t, isTodoLine("task"))

	path := filepath.Join(t.TempDir(), "lines.md")
	require.NoError(t, writeLines(path, []string{"one", "two"}))
	lines, err := readLines(path)
	require.NoError(t, err)
	require.Equal(t, []string{"one", "two"}, lines)
}

func writeTodoFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "todo.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func readTodoFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}
