package tui

import (
	"context"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
	notesync "atbuy/noteui/internal/sync"
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

func batchMoveCmd(root string, items []moveBatchItem) tea.Cmd {
	return func() tea.Msg {
		for _, item := range items {
			var err error
			switch item.kind {
			case moveTargetCategory:
				err = notes.MoveCategory(root, item.oldRelPath, item.newRelPath)
			case moveTargetNote:
				err = notes.MoveNote(root, item.oldRelPath, item.newRelPath)
			}
			if err != nil {
				return batchMovedMsg{items: items, err: err}
			}
		}
		return batchMovedMsg{items: items}
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

func addNoteTagsCmd(path string, tags []string) tea.Cmd {
	return func() tea.Msg {
		err := notes.AddTagsToNote(path, tags)
		return noteTaggedMsg{path: path, tags: tags, err: err}
	}
}

func createTodoNoteCmd(root, relDir string) tea.Cmd {
	return func() tea.Msg {
		path, err := notes.CreateTodoNote(root, relDir)
		return noteCreatedMsg{path: path, err: err}
	}
}

func toggleTodoCmd(path string, lineIdx int) tea.Cmd {
	return func() tea.Msg {
		err := notes.ToggleTodoLine(path, lineIdx)
		return todoModifiedMsg{path: path, err: err}
	}
}

func addTodoCmd(path, text string) tea.Cmd {
	return func() tea.Msg {
		err := notes.AddTodoItem(path, text)
		return todoModifiedMsg{path: path, err: err}
	}
}

func deleteTodoCmd(path string, lineIdx int) tea.Cmd {
	return func() tea.Msg {
		err := notes.DeleteTodoLine(path, lineIdx)
		return todoModifiedMsg{path: path, err: err}
	}
}

func editTodoCmd(path string, lineIdx int, newText string) tea.Cmd {
	return func() tea.Msg {
		err := notes.EditTodoLine(path, lineIdx, newText)
		return todoModifiedMsg{path: path, err: err}
	}
}

func encryptNoteCmd(path, passphrase string) tea.Cmd {
	return func() tea.Msg {
		err := notes.EncryptNoteFile(path, passphrase)
		return encryptNoteMsg{path: path, err: err}
	}
}

func decryptNoteCmd(path, passphrase string) tea.Cmd {
	return func() tea.Msg {
		err := notes.DecryptNoteFile(path, passphrase)
		return decryptNoteMsg{path: path, err: err}
	}
}

func openEncryptedNoteCmd(path, passphrase string) tea.Cmd {
	return func() tea.Msg {
		raw, err := notes.ReadAll(path)
		if err != nil {
			return openEncryptedNoteReadyMsg{origPath: path, err: err}
		}

		tempContent, err := notes.PrepareForEdit(raw, passphrase)
		if err != nil {
			return openEncryptedNoteReadyMsg{origPath: path, err: err}
		}

		tmpFile, err := os.CreateTemp("", "noteui-*.md")
		if err != nil {
			return openEncryptedNoteReadyMsg{origPath: path, err: err}
		}
		tmpFile.Close()

		if err := os.WriteFile(tmpFile.Name(), []byte(tempContent), 0o600); err != nil {
			os.Remove(tmpFile.Name())
			return openEncryptedNoteReadyMsg{origPath: path, err: err}
		}

		return openEncryptedNoteReadyMsg{origPath: path, tempPath: tmpFile.Name()}
	}
}

func reencryptFromTempCmd(origPath, tempPath, passphrase string) tea.Cmd {
	return func() tea.Msg {
		newPath, err := notes.ReencryptFromTemp(origPath, tempPath, passphrase)
		return reencryptFinishedMsg{newPath: newPath, err: err}
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

type syncStartMsg struct{}
type syncDebouncedMsg struct{ token int }

type syncFinishedMsg struct {
	result notesync.SyncResult
	err    error
}

type syncImportFinishedMsg struct {
	result notesync.SyncResult
	err    error
}

type noteSyncClassToggledMsg struct {
	path      string
	syncClass string
	err       error
}

type remoteNoteDeletedMsg struct {
	path string
	err  error
}

func startSyncCmd() tea.Cmd {
	return func() tea.Msg { return syncStartMsg{} }
}

func syncDebounceCmd(token int) tea.Cmd {
	return tea.Tick(750*time.Millisecond, func(time.Time) tea.Msg {
		return syncDebouncedMsg{token: token}
	})
}

func syncNowCmd(root string, cfg config.SyncConfig, pinnedNotes []string, pinnedCats []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := notesync.SyncRoot(ctx, root, cfg, pinnedNotes, pinnedCats, nil)
		return syncFinishedMsg{result: result, err: err}
	}
}

func toggleNoteSyncCmd(path string) tea.Cmd {
	return func() tea.Msg {
		syncClass, err := notes.ToggleNoteSyncClass(path)
		return noteSyncClassToggledMsg{path: path, syncClass: syncClass, err: err}
	}
}

func deleteRemoteNoteKeepLocalCmd(root, path string, cfg config.SyncConfig) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := notesync.DeleteRemoteNoteAndKeepLocal(ctx, root, path, cfg, nil)
		return remoteNoteDeletedMsg{path: path, err: err}
	}
}

func importCurrentSyncedNoteCmd(root string, cfg config.SyncConfig, noteID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := notesync.ImportRemoteNote(ctx, root, cfg, noteID, nil)
		return syncImportFinishedMsg{result: result, err: err}
	}
}

func importSyncedNotesCmd(root string, cfg config.SyncConfig) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := notesync.ImportRemoteNotes(ctx, root, cfg, nil)
		return syncImportFinishedMsg{result: result, err: err}
	}
}
