package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
	notesync "atbuy/noteui/internal/sync"
)

var (
	openURLGOOS        = runtime.GOOS
	openURLExecCommand = exec.Command
	openURLStart       = startDetachedProcess
)

func refreshAllCmd(root string, sessionToken int) tea.Cmd {
	return func() tea.Msg {
		n, err := notes.Discover(root)
		if err != nil {
			return dataLoadedMsg{err: err, sessionToken: sessionToken}
		}

		tmp, err := notes.DiscoverTemporary(root)
		if err != nil {
			return dataLoadedMsg{err: err, sessionToken: sessionToken}
		}

		cats, err := notes.DiscoverCategories(root)
		if err != nil {
			return dataLoadedMsg{err: err, sessionToken: sessionToken}
		}

		return dataLoadedMsg{
			notes:        n,
			tempNotes:    tmp,
			categories:   cats,
			sessionToken: sessionToken,
		}
	}
}

func createNoteCmd(root, relDir string) tea.Cmd {
	return func() tea.Msg {
		path, err := notes.CreateNote(root, relDir)
		return noteCreatedMsg{path: path, err: err}
	}
}

func createNoteFromTemplateCmd(root, relDir, templatePath string) tea.Cmd {
	return func() tea.Msg {
		path, err := notes.CreateNoteFromTemplate(root, relDir, templatePath)
		return noteCreatedMsg{path: path, err: err}
	}
}

func createTemplateCmd(root string) tea.Cmd {
	return func() tea.Msg {
		path, err := notes.CreateTemplate(root)
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
		result, err := notes.DeleteNote(path)
		return noteDeletedMsg{path: path, result: result, err: err}
	}
}

func deleteCategoryCmd(root, relPath string) tea.Cmd {
	return func() tea.Msg {
		result, err := notes.DeleteCategory(root, relPath)
		return categoryDeletedMsg{relPath: relPath, result: result, err: err}
	}
}

func restoreFromTrashCmd(label string, results []notes.TrashResult) tea.Cmd {
	copyResults := append([]notes.TrashResult(nil), results...)
	return func() tea.Msg {
		for _, r := range copyResults {
			if err := notes.RestoreFromTrash(r); err != nil {
				return restoreFinishedMsg{label: label, err: err}
			}
		}
		return restoreFinishedMsg{label: label}
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

func removeNoteTagsCmd(path string, tags []string) tea.Cmd {
	return func() tea.Msg {
		err := notes.RemoveTagsFromNote(path, tags)
		return noteUntaggedMsg{path: path, tags: tags, err: err}
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

func updateTodoDueDateCmd(path string, lineIdx int, dueDate string) tea.Cmd {
	return func() tea.Msg {
		err := notes.UpdateTodoDueDate(path, lineIdx, dueDate)
		return todoModifiedMsg{path: path, err: err}
	}
}

func updateTodoPriorityCmd(path string, lineIdx int, priority string) tea.Cmd {
	return func() tea.Msg {
		err := notes.UpdateTodoPriority(path, lineIdx, priority)
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

func reencryptFromTempCmd(origPath, tempPath, passphrase string) tea.Cmd {
	return func() tea.Msg {
		newPath, err := notes.ReencryptFromTemp(origPath, tempPath, passphrase)
		return reencryptFinishedMsg{newPath: newPath, err: err}
	}
}

func startWatchTeaCmd(root string, sessionToken int) tea.Cmd {
	return func() tea.Msg {
		return startWatchCmd(root, sessionToken)()
	}
}

func waitForWatchTeaCmd(events <-chan teaMsg) tea.Cmd {
	return func() tea.Msg {
		return waitForWatchCmd(events)()
	}
}

type (
	syncStartMsg     struct{ sessionToken int }
	syncDebouncedMsg struct {
		token        int
		sessionToken int
	}
)

type syncFinishedMsg struct {
	result       notesync.SyncResult
	err          error
	sessionToken int
}

type syncImportFinishedMsg struct {
	result       notesync.SyncResult
	err          error
	sessionToken int
}

type noteSyncClassToggledMsg struct {
	path      string
	syncClass string
	err       error
}

type noteMadeSharedMsg struct {
	path      string
	syncClass string
	err       error
}

type remoteNoteDeletedMsg struct {
	path         string
	err          error
	sessionToken int
}

type conflictResolvedMsg struct {
	keepRemote bool
	err        error
}

type noteVersionSavedMsg struct{}

type noteHistoryLoadedMsg struct {
	relPath string
	entries []notes.HistoryEntry
	err     error
}

type noteVersionRestoredMsg struct {
	relPath string
	err     error
}

type syncEventsLoadedMsg struct {
	events []notesync.SyncEvent
}

type syncUnlinkLocalMsg struct {
	err error
}

func startSyncCmd(sessionToken int) tea.Cmd {
	return func() tea.Msg { return syncStartMsg{sessionToken: sessionToken} }
}

func syncDebounceCmd(token, sessionToken int) tea.Cmd {
	return tea.Tick(750*time.Millisecond, func(time.Time) tea.Msg {
		return syncDebouncedMsg{token: token, sessionToken: sessionToken}
	})
}

func syncNowCmd(root, remoteRootOverride string, cfg config.SyncConfig, pinnedNotes []string, pinnedCats []string, sessionToken int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := notesync.SyncRoot(ctx, root, remoteRootOverride, cfg, pinnedNotes, pinnedCats, nil)
		return syncFinishedMsg{result: result, err: err, sessionToken: sessionToken}
	}
}

func toggleNoteSyncCmd(path string) tea.Cmd {
	return func() tea.Msg {
		syncClass, err := notes.ToggleNoteSyncClass(path)
		return noteSyncClassToggledMsg{path: path, syncClass: syncClass, err: err}
	}
}

func toggleNoteSharedCmd(path, targetClass string) tea.Cmd {
	return func() tea.Msg {
		err := notes.SetNoteSyncClass(path, targetClass)
		return noteMadeSharedMsg{path: path, syncClass: targetClass, err: err}
	}
}

func deleteRemoteNoteKeepLocalCmd(root, path, remoteRootOverride string, cfg config.SyncConfig, sessionToken int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := notesync.DeleteRemoteNoteAndKeepLocal(ctx, root, path, remoteRootOverride, cfg, nil)
		return remoteNoteDeletedMsg{path: path, err: err, sessionToken: sessionToken}
	}
}

func importCurrentSyncedNoteCmd(root, remoteRootOverride string, cfg config.SyncConfig, noteID string, sessionToken int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := notesync.ImportRemoteNote(ctx, root, remoteRootOverride, cfg, noteID, nil)
		return syncImportFinishedMsg{result: result, err: err, sessionToken: sessionToken}
	}
}

func loadSyncEventsCmd(root string) tea.Cmd {
	return func() tea.Msg {
		events, _ := notesync.LoadSyncEvents(root, 50)
		return syncEventsLoadedMsg{events: events}
	}
}

func unlinkNoteLocallyCmd(root, notePath string) tea.Cmd {
	return func() tea.Msg {
		err := notesync.UnlinkNoteLocally(root, notePath)
		return syncUnlinkLocalMsg{err: err}
	}
}

func importSyncedNotesCmd(root, remoteRootOverride string, cfg config.SyncConfig, sessionToken int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := notesync.ImportRemoteNotes(ctx, root, remoteRootOverride, cfg, nil)
		return syncImportFinishedMsg{result: result, err: err, sessionToken: sessionToken}
	}
}

// saveNoteVersionCmd reads the note at absPath and saves its current content as a
// history version. Failures are silently ignored so they never block the main workflow.
func saveNoteVersionCmd(root, absPath string) tea.Cmd {
	return func() tea.Msg {
		content, err := notes.ReadAll(absPath)
		if err == nil {
			relPath, relErr := filepath.Rel(root, absPath)
			if relErr == nil {
				_ = notes.SaveVersion(root, filepath.ToSlash(relPath), content)
			}
		}
		return noteVersionSavedMsg{}
	}
}

// saveNoteVersionAndOpenEncryptedCmd saves the current encrypted blob as a history
// version, then decrypts and opens the note in an external editor. Combining both
// steps ensures the pre-edit snapshot is captured before the editor runs.
func saveNoteVersionAndOpenEncryptedCmd(root, path, passphrase string) tea.Cmd {
	return func() tea.Msg {
		// Save the current encrypted content before decrypting for editing.
		if content, err := notes.ReadAll(path); err == nil {
			if relPath, relErr := filepath.Rel(root, path); relErr == nil {
				_ = notes.SaveVersion(root, filepath.ToSlash(relPath), content)
			}
		}

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
		_ = tmpFile.Close()
		if err := os.WriteFile(tmpFile.Name(), []byte(tempContent), 0o600); err != nil {
			_ = os.Remove(tmpFile.Name())
			return openEncryptedNoteReadyMsg{origPath: path, err: err}
		}
		return openEncryptedNoteReadyMsg{origPath: path, tempPath: tmpFile.Name()}
	}
}

func loadNoteHistoryCmd(root, relPath string) tea.Cmd {
	return func() tea.Msg {
		entries, err := notes.Versions(root, relPath)
		return noteHistoryLoadedMsg{relPath: relPath, entries: entries, err: err}
	}
}

type trashBrowserLoadedMsg struct {
	items []notes.TrashedItem
	err   error
}

type trashRestoreMsg struct {
	item notes.TrashedItem
	err  error
}

func loadTrashBrowserCmd(root string) tea.Cmd {
	return func() tea.Msg {
		items, err := notes.ListTrashed(root)
		return trashBrowserLoadedMsg{items: items, err: err}
	}
}

func restoreTrashItemCmd(item notes.TrashedItem) tea.Cmd {
	return func() tea.Msg {
		result := notes.TrashResult{
			OriginalPath:  item.OriginalPath,
			TrashFilePath: item.TrashFilePath,
			TrashInfoPath: item.TrashInfoPath,
		}
		return trashRestoreMsg{item: item, err: notes.RestoreFromTrash(result)}
	}
}

func restoreNoteVersionCmd(root, absPath, relPath, versionID string) tea.Cmd {
	return func() tea.Msg {
		err := notes.RestoreVersion(root, absPath, relPath, versionID)
		return noteVersionRestoredMsg{relPath: relPath, err: err}
	}
}

type openURLMsg struct {
	url string
	err error
}

func buildOpenURLCommand(goos, url string) *exec.Cmd {
	switch goos {
	case "darwin":
		return openURLExecCommand("open", url)
	case "windows":
		return openURLExecCommand("cmd", "/c", "start", "", url)
	default:
		return openURLExecCommand("xdg-open", url)
	}
}

func startDetachedProcess(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Release()
}

func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		return openURLMsg{url: url, err: openURLStart(buildOpenURLCommand(openURLGOOS, url))}
	}
}
