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

func TestEditorModelZZCentersCursorInRawMode(t *testing.T) {
	body := ""
	for i := 0; i < 40; i++ {
		if i > 0 {
			body += "\n"
		}
		body += "line"
	}
	e := NewEditorModel("", "note.txt", "", body, 80, 14, false, "", false)
	e.renderMode = false
	e.row = 20
	e.preferCol = 0
	e.col = 0
	e.viewTop = 0

	var cmd tea.Cmd
	e, cmd = updateEditorModel(e, keyMsg("z"))
	require.Nil(t, cmd)
	require.Equal(t, rune('z'), e.pendingPrefix)

	e, cmd = updateEditorModel(e, keyMsg("z"))
	require.Nil(t, cmd)
	require.Equal(t, rune(0), e.pendingPrefix)
	require.Equal(t, 20, e.row)
	require.Equal(t, e.row-e.contentHeight/2, e.viewTop)
	require.GreaterOrEqual(t, e.row, e.viewTop)
	require.Less(t, e.row, e.viewTop+e.contentHeight)
}

func TestEditorModelZZClampsAtTop(t *testing.T) {
	e := NewEditorModel("", "note.txt", "", "a\nb\nc\nd\ne", 80, 14, false, "", false)
	e.renderMode = false
	e.row = 0
	e.viewTop = 0

	e, _ = updateEditorModel(e, keyMsg("z"))
	e, _ = updateEditorModel(e, keyMsg("z"))

	require.Equal(t, 0, e.viewTop)
}

func TestEditorModelZZCentersInRenderMode(t *testing.T) {
	body := ""
	for i := 0; i < 30; i++ {
		if i > 0 {
			body += "\n\n"
		}
		body += "paragraph line"
	}
	e := NewEditorModel("", "note.md", "", body, 80, 14, false, "", false)
	if !e.renderMode || e.renderedDoc == nil {
		t.Skip("rendered doc unavailable")
	}
	e.row = 30
	e.renderViewTop = 0

	e, _ = updateEditorModel(e, keyMsg("z"))
	e, _ = updateEditorModel(e, keyMsg("z"))

	vStart := e.rowToVisualStart(30)
	require.Equal(t, max(0, vStart-e.contentHeight/2), e.renderViewTop)
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
