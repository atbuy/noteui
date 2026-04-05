package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
	notesync "atbuy/noteui/internal/sync"
)

func TestPaletteCommandsOmitSyncWithoutProfile(t *testing.T) {
	m := newTestModel(t)
	cmds := paletteCommands(m)
	for _, cmd := range cmds {
		require.NotEqual(t, cmdSyncNow, cmd.action)
		require.NotEqual(t, cmdImportAll, cmd.action)
	}
}

func TestPaletteCommandsIncludeSyncWithProfile(t *testing.T) {
	cfg := config.Default()
	cfg.Dashboard = false
	cfg.Sync.DefaultProfile = "main"
	cfg.Sync.Profiles = map[string]config.SyncProfile{
		"main": {SSHHost: "notes", RemoteRoot: "/srv/notes", RemoteBin: "noteui-sync"},
	}
	m := New(t.TempDir(), "", cfg, "test")
	cmds := paletteCommands(m)
	actions := make(map[string]bool, len(cmds))
	for _, cmd := range cmds {
		actions[cmd.action] = true
	}
	require.True(t, actions[cmdSyncNow])
	require.True(t, actions[cmdImportAll])
}

func TestPaletteCommandsIncludeDeleteRemoteWhenApplicable(t *testing.T) {
	m := newTestModel(t)
	n := notes.Note{Path: m.rootDir + "/work/note.md", RelPath: "work/note.md", Name: "note.md", TitleText: "Note", SyncClass: notes.SyncClassSynced}
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.syncRecords = map[string]notesync.NoteRecord{n.RelPath: {RelPath: n.RelPath}}
	cmds := paletteCommands(m)
	actions := make(map[string]bool, len(cmds))
	for _, cmd := range cmds {
		actions[cmd.action] = true
	}
	require.True(t, actions[cmdDeleteRemoteKeepLocal])
}

func TestRebuildPaletteFilteredIncludesNotesAndCommands(t *testing.T) {
	m := newTestModel(t)
	m.notes = []notes.Note{{RelPath: "work/new-ideas.md", Name: "new-ideas.md", TitleText: "New Ideas"}}
	m.openCommandPalette()
	m.commandPaletteInput.SetValue("new")
	m.rebuildPaletteFiltered()

	var sawNote, sawCommand bool
	for _, item := range m.commandPaletteFiltered {
		if item.kind == paletteKindCommand && item.title == "New Note" {
			sawCommand = true
		}
		if item.kind == paletteKindNote && item.title == "New Ideas" {
			sawNote = true
		}
	}

	require.True(t, sawNote)
	require.True(t, sawCommand)
}

func TestRebuildPaletteFilteredDoesNotDuplicateNotePerCommand(t *testing.T) {
	m := newTestModel(t)
	m.notes = []notes.Note{{RelPath: "work/testing.md", Name: "testing.md", TitleText: "Testing"}}
	m.openCommandPalette()

	count := 0
	for _, item := range m.commandPaletteFiltered {
		if item.kind != paletteKindCommand && item.title == "Testing" {
			count++
		}
	}
	require.Equal(t, 1, count)
}

func TestTabCompletePaletteCommandUsesVisibleLabel(t *testing.T) {
	m := newTestModel(t)
	m.openCommandPalette()
	m.commandPaletteInput.SetValue("show h")
	m.rebuildPaletteFiltered()
	require.NotEmpty(t, m.commandPaletteFiltered)
	m.tabCompletePalette()
	require.Equal(t, "Show Help", m.commandPaletteInput.Value())
	require.Equal(t, paletteKindCommand, m.commandPaletteFiltered[0].kind)
}

func TestTabCompletePaletteNoteUsesTitle(t *testing.T) {
	m := newTestModel(t)
	m.notes = []notes.Note{{RelPath: "work/testing.md", Name: "testing.md", TitleText: "Testing"}}
	m.openCommandPalette()
	m.commandPaletteInput.SetValue("test")
	m.rebuildPaletteFiltered()
	require.NotEmpty(t, m.commandPaletteFiltered)
	m.tabCompletePalette()
	require.Equal(t, "Testing", m.commandPaletteInput.Value())
}

func TestCommitPaletteSelectionCommandOpensHelp(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	m.openCommandPalette()
	m.commandPaletteInput.SetValue("show help")
	m.rebuildPaletteFiltered()
	require.NotEmpty(t, m.commandPaletteFiltered)
	cmd := m.commitPaletteSelection()
	require.Nil(t, cmd)
	require.False(t, m.showCommandPalette)
	require.True(t, m.showHelp)
	require.Equal(t, "help", m.status)
}

func TestCommandPaletteEnterReturnsCommand(t *testing.T) {
	m := newTestModel(t)
	m.openCommandPalette()
	m.commandPaletteInput.SetValue("refresh")
	m.rebuildPaletteFiltered()
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)
	require.NotNil(t, cmd)
	require.False(t, updated.showCommandPalette)
}

func TestCommandPalettePrintableKeysStayInInput(t *testing.T) {
	m := newTestModel(t)
	m.notes = []notes.Note{{RelPath: "work/testing.md", Name: "testing.md", TitleText: "Testing"}}
	m.openCommandPalette()

	next, _ := m.Update(keyMsg("q"))
	updated := next.(Model)
	require.True(t, updated.showCommandPalette)
	require.Equal(t, "q", updated.commandPaletteInput.Value())

	next, _ = updated.Update(keyMsg("j"))
	updated = next.(Model)
	require.True(t, updated.showCommandPalette)
	require.Equal(t, "qj", updated.commandPaletteInput.Value())
	require.Equal(t, 0, updated.commandPaletteCursor)
}

func TestRenderCommandPaletteModalShowsNewTitleAndPrompt(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	m.notes = []notes.Note{{RelPath: "work/testing.md", Name: "testing.md", TitleText: "Testing"}}
	m.openCommandPalette()
	m.commandPaletteInput.SetValue("test")
	m.rebuildPaletteFiltered()

	rendered := stripANSI(m.renderCommandPaletteModal())
	require.Contains(t, rendered, "Command palette")
	require.NotContains(t, rendered, "Quick open")

	foundInputLine := false
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "> test") {
			foundInputLine = true
			break
		}
	}
	require.True(t, foundInputLine)
}

func TestPaletteFuzzyQueryMatchesCommand(t *testing.T) {
	m := newTestModel(t)
	m.openCommandPalette()
	m.commandPaletteInput.SetValue("shlp")
	m.rebuildPaletteFiltered()
	require.NotEmpty(t, m.commandPaletteFiltered)
	require.Equal(t, paletteKindCommand, m.commandPaletteFiltered[0].kind)
	require.Equal(t, "Show Help", m.commandPaletteFiltered[0].title)
}

func TestPaletteRecentCommandsBoostMatchingCommands(t *testing.T) {
	m := newTestModel(t)
	m.state.RecentCommands = []string{cmdShowPins}
	m.openCommandPalette()
	m.commandPaletteInput.SetValue("show")
	m.rebuildPaletteFiltered()
	require.NotEmpty(t, m.commandPaletteFiltered)
	require.Equal(t, "Show Pins", m.commandPaletteFiltered[0].title)
}

func TestCommitPaletteSelectionRecordsRecentCommand(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	m.openCommandPalette()
	m.commandPaletteInput.SetValue("show help")
	m.rebuildPaletteFiltered()
	require.NotEmpty(t, m.commandPaletteFiltered)
	_ = m.commitPaletteSelection()
	require.Equal(t, []string{cmdShowHelp}, m.state.RecentCommands)
}

func TestRenderCommandPaletteModalShowsGroupedSections(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	m.notes = []notes.Note{{RelPath: "work/testing.md", Name: "testing.md", TitleText: "Testing"}}
	m.tempNotes = []notes.Note{{RelPath: "scratch.md", Name: "scratch.md", TitleText: "Scratch"}}
	n := m.notes[0]
	m.treeItems = []treeItem{{Kind: treeNote, RelPath: n.RelPath, Name: n.Title(), Note: &n}}
	m.openCommandPalette()

	sections := make(map[paletteSection]bool)
	for _, item := range m.commandPaletteFiltered {
		sections[item.section] = true
	}
	require.True(t, sections[paletteSectionSuggested])
	require.True(t, sections[paletteSectionCommand])
	require.True(t, sections[paletteSectionNote])
	require.True(t, sections[paletteSectionTempNote])

	rendered := stripANSI(m.renderCommandPaletteModal())
	require.Contains(t, rendered, "Suggested actions")
	require.Contains(t, rendered, "Commands")
}

func TestRenderCommandPaletteModalKeepsTypedTextOnPromptLine(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	m.height = 40
	m.openCommandPalette()
	m.commandPaletteInput.SetValue("q")
	m.rebuildPaletteFiltered()

	rendered := stripANSI(m.renderCommandPaletteModal())
	found := false
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "> q") {
			found = true
			break
		}
	}
	require.True(t, found)
}
