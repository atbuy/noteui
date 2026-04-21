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

func TestEditorModelCtrlWDeletesPreviousWordInInsertMode(t *testing.T) {
	e := NewEditorModel("", "note.md", "", "one two three", 80, 24, false, "", false)
	e.markLoaded(editorHashContent(e.Content()), time.Now())

	e, _ = updateEditorModel(e, keyMsg("A"))
	e, _ = updateEditorModel(e, keyMsg("ctrl+w"))

	require.Equal(t, "one two ", e.Content())
	require.Equal(t, editorModeInsert, e.mode)
	require.Equal(t, len([]rune("one two ")), e.col)
	require.True(t, e.dirty)
}

func TestEditorModelCtrlWDeletesPreviousWordAndWhitespace(t *testing.T) {
	e := NewEditorModel("", "note.md", "", "one   two", 80, 24, false, "", false)
	e.markLoaded(editorHashContent(e.Content()), time.Now())

	e.mode = editorModeInsert
	e.col = len([]rune("one   "))
	e.preferCol = e.col

	e, _ = updateEditorModel(e, keyMsg("ctrl+w"))

	require.Equal(t, "two", e.Content())
	require.Equal(t, editorModeInsert, e.mode)
	require.Equal(t, 0, e.col)
	require.True(t, e.dirty)
}
