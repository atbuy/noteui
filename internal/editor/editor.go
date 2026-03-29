package editor

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

func Command(path string) *exec.Cmd {
	bin := os.Getenv("EDITOR")
	if bin == "" {
		bin = "nvim"
	}
	return exec.Command(bin, path)
}

func Open(path string) tea.Cmd {
	return tea.ExecProcess(Command(path), func(err error) tea.Msg {
		return FinishedMsg{Err: err}
	})
}

type FinishedMsg struct {
	Err error
}

