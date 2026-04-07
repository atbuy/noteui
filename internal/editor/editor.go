// Package editor opens files in the user's preferred terminal editor.
package editor

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

func Command(path string) *exec.Cmd {
	bin := os.Getenv("NOTEUI_EDITOR")
	if bin == "" {
		bin = os.Getenv("EDITOR")
	}
	if bin == "" {
		bin = "nvim"
	}
	return exec.Command(bin, path)
}

func Open(path string) tea.Cmd {
	return tea.ExecProcess(Command(path), func(err error) tea.Msg {
		return FinishedMsg{
			Err:  err,
			Path: path,
		}
	})
}

type FinishedMsg struct {
	Err  error
	Path string
}
