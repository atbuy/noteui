package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func updateEditorModel(e EditorModel, msg tea.Msg) (EditorModel, tea.Cmd) {
	return e.Update(msg)
}

func TestEditorModelGLRequestsLinkPicker(t *testing.T) {
	e := NewEditorModel("", "note.md", "", "body", 80, 24, false, "", false)

	var cmd tea.Cmd
	e, cmd = updateEditorModel(e, keyMsg("g"))
	require.Nil(t, cmd)

	_, cmd = updateEditorModel(e, keyMsg("l"))
	require.NotNil(t, cmd)

	msg := cmd()
	require.IsType(t, editorLinkPickerMsg{}, msg)
}

func TestEditorModelInsertWikilinkUsesVisualSelectionAsLabel(t *testing.T) {
	e := NewEditorModel("", "note.md", "", "hello world", 80, 24, false, "", false)
	e.markLoaded(editorHashContent(e.Content()), time.Now())

	e, _ = updateEditorModel(e, keyMsg("V"))
	e.InsertWikilink("Target Note")

	require.Equal(t, "[[Target Note|hello world]]", e.Content())
	require.Equal(t, editorModeNormal, e.mode)
	require.True(t, e.dirty)
}

func TestEditorModelInsertURLLinkWrapsVisualSelection(t *testing.T) {
	e := NewEditorModel("", "note.md", "", "hello world", 80, 24, false, "", false)
	e.markLoaded(editorHashContent(e.Content()), time.Now())

	e, _ = updateEditorModel(e, keyMsg("V"))
	e.InsertURLLink("https://example.com")

	require.Equal(t, "[hello world](https://example.com)", e.Content())
	require.Equal(t, editorModeNormal, e.mode)
	require.True(t, e.dirty)
}

func TestEditorModelQRefusesDirtyBuffer(t *testing.T) {
	e := NewEditorModel("", "note.md", "", "body", 80, 24, false, "", false)
	e.markLoaded(editorHashContent(e.Content()), time.Now())

	e, _ = updateEditorModel(e, keyMsg("i"))
	e, _ = updateEditorModel(e, keyMsg("x"))
	e, _ = updateEditorModel(e, keyMsg("esc"))
	e, _ = updateEditorModel(e, keyMsg(":"))
	e, _ = updateEditorModel(e, keyMsg("q"))
	e, cmd := updateEditorModel(e, keyMsg("enter"))

	require.Nil(t, cmd)
	require.Equal(t, editorModeNormal, e.mode)
	require.Contains(t, e.status, "no write since last change")
}
