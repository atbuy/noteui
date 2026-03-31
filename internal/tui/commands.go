package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"atbuy/noteui/internal/notes"
)

func refreshAllCmd(root string) tea.Cmd {
	return func() tea.Msg {
		n, err := notes.Discover(root)
		if err != nil {
			return dataLoadedMsg{err: err}
		}

		tmp, err := notes.DiscoverTemporary(root)
		if err != nil {
			return dataLoadedMsg{err: err}
		}

		cats, err := notes.DiscoverCategories(root)
		if err != nil {
			return dataLoadedMsg{err: err}
		}

		return dataLoadedMsg{
			notes:      n,
			tempNotes:  tmp,
			categories: cats,
		}
	}
}

func createNoteCmd(root, relDir string) tea.Cmd {
	return func() tea.Msg {
		path, err := notes.CreateNote(root, relDir)
		return noteCreatedMsg{path: path, err: err}
	}
}

func createCategoryCmd(root, relPath string) tea.Cmd {
	return func() tea.Msg {
		err := notes.CreateCategory(root, relPath)
		return categoryCreatedMsg{relPath: relPath, err: err}
	}
}

func createTemporaryNoteCmd(root string) tea.Cmd {
	return func() tea.Msg {
		path, err := notes.CreateTemporaryNote(root)
		return noteCreatedMsg{path: path, err: err}
	}
}

func deleteNoteCmd(path string) tea.Cmd {
	return func() tea.Msg {
		err := notes.DeleteNote(path)
		return noteDeletedMsg{path: path, err: err}
	}
}

func deleteCategoryCmd(root, relPath string) tea.Cmd {
	return func() tea.Msg {
		err := notes.DeleteCategory(root, relPath)
		return categoryDeletedMsg{relPath: relPath, err: err}
	}
}

func moveNoteCmd(root, oldRelPath, newRelPath string) tea.Cmd {
	return func() tea.Msg {
		err := notes.MoveNote(root, oldRelPath, newRelPath)
		return noteMovedMsg{
			oldRelPath: oldRelPath,
			newRelPath: newRelPath,
			err:        err,
		}
	}
}

func moveCategoryCmd(root, oldRelPath, newRelPath string) tea.Cmd {
	return func() tea.Msg {
		err := notes.MoveCategory(root, oldRelPath, newRelPath)
		return categoryMovedMsg{
			oldRelPath: oldRelPath,
			newRelPath: newRelPath,
			err:        err,
		}
	}
}

func renameNoteCmd(path, newTitle string) tea.Cmd {
	return func() tea.Msg {
		newPath, _, err := notes.RenameNoteTitle(path, newTitle)
		return noteRenamedMsg{
			oldPath: path,
			newPath: newPath,
			err:     err,
		}
	}
}

func renameCategoryCmd(root, oldRelPath, newRelPath string) tea.Cmd {
	return func() tea.Msg {
		err := notes.MoveCategory(root, oldRelPath, newRelPath)
		return categoryRenamedMsg{
			oldRelPath: oldRelPath,
			newRelPath: newRelPath,
			err:        err,
		}
	}
}

func startWatchTeaCmd(root string) tea.Cmd {
	return func() tea.Msg {
		return startWatchCmd(root)()
	}
}

func waitForWatchTeaCmd(events <-chan teaMsg) tea.Cmd {
	return func() tea.Msg {
		return waitForWatchCmd(events)()
	}
}
