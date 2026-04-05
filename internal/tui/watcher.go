package tui

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"atbuy/noteui/internal/notes"
)

type watchStartedMsg struct {
	watcher      *fsnotify.Watcher
	events       chan teaMsg
	err          error
	sessionToken int
}

type (
	watchRefreshMsg struct{ sessionToken int }
	watchErrorMsg   struct {
		err          error
		sessionToken int
	}
)

type teaMsg interface{}

func startWatchCmd(root string, sessionToken int) func() teaMsg {
	return func() teaMsg {
		watcher, events, err := newRecursiveWatcher(root, sessionToken)
		if err != nil {
			return watchStartedMsg{err: err, sessionToken: sessionToken}
		}
		return watchStartedMsg{
			watcher:      watcher,
			events:       events,
			sessionToken: sessionToken,
		}
	}
}

func waitForWatchCmd(events <-chan teaMsg) func() teaMsg {
	return func() teaMsg {
		msg, ok := <-events
		if !ok {
			return watchErrorMsg{err: nil}
		}
		return msg
	}
}

func newRecursiveWatcher(root string, sessionToken int) (*fsnotify.Watcher, chan teaMsg, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}

	if err := addWatchTree(watcher, root); err != nil {
		_ = watcher.Close()
		return nil, nil, err
	}

	out := make(chan teaMsg, 8)

	go func() {
		defer close(out)

		var (
			timer   *time.Timer
			timerCh <-chan time.Time
			pending bool
		)

		schedule := func() {
			pending = true
			if timer == nil {
				timer = time.NewTimer(250 * time.Millisecond)
				timerCh = timer.C
				return
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(250 * time.Millisecond)
			timerCh = timer.C
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
					continue
				}

				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						_ = addWatchTree(watcher, event.Name)
					}
				}

				schedule()

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				out <- watchErrorMsg{err: err, sessionToken: sessionToken}

			case <-timerCh:
				if pending {
					out <- watchRefreshMsg{sessionToken: sessionToken}
					pending = false
				}
				timerCh = nil
			}
		}
	}()

	return watcher, out, nil
}

func addWatchTree(watcher *fsnotify.Watcher, root string) error {
	root = filepath.Clean(root)
	tempRoot := filepath.Clean(notes.TempRoot(root))

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		name := d.Name()
		cleanPath := filepath.Clean(path)

		if cleanPath != root && strings.HasPrefix(name, ".") && cleanPath != tempRoot {
			return filepath.SkipDir
		}

		return watcher.Add(cleanPath)
	})
}
