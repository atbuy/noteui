// Package editor opens files in the user's preferred terminal editor.
package editor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func Command(path string) (*exec.Cmd, error) {
	bin := os.Getenv("NOTEUI_EDITOR")
	if bin == "" {
		bin = os.Getenv("EDITOR")
	}
	if bin == "" {
		bin = "nvim"
	}
	parts, err := splitCommand(bin)
	if err != nil {
		return nil, fmt.Errorf("invalid editor command %q: %w", bin, err)
	}
	return exec.Command(parts[0], append(parts[1:], path)...), nil
}

func Open(path string) tea.Cmd {
	cmd, err := Command(path)
	if err != nil {
		return func() tea.Msg {
			return FinishedMsg{
				Err:  err,
				Path: path,
			}
		}
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return FinishedMsg{
			Err:  err,
			Path: path,
		}
	})
}

func splitCommand(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("empty command")
	}

	var (
		parts       []string
		buf         strings.Builder
		quote       rune
		escaped     bool
		wordStarted bool
	)

	flush := func() {
		parts = append(parts, buf.String())
		buf.Reset()
		wordStarted = false
	}

	for _, r := range raw {
		if escaped {
			buf.WriteRune(r)
			escaped = false
			wordStarted = true
			continue
		}
		switch quote {
		case '\'':
			if r == '\'' {
				quote = 0
			} else {
				buf.WriteRune(r)
			}
			wordStarted = true
			continue
		case '"':
			switch r {
			case '"':
				quote = 0
			case '\\':
				escaped = true
			default:
				buf.WriteRune(r)
			}
			wordStarted = true
			continue
		}

		switch r {
		case ' ', '\t', '\n', '\r':
			if wordStarted {
				flush()
			}
		case '\'', '"':
			quote = r
			wordStarted = true
		case '\\':
			escaped = true
			wordStarted = true
		default:
			buf.WriteRune(r)
			wordStarted = true
		}
	}

	switch {
	case escaped:
		return nil, errors.New("unfinished escape")
	case quote != 0:
		return nil, errors.New("unterminated quote")
	case wordStarted:
		flush()
	}

	if len(parts) == 0 {
		return nil, errors.New("empty command")
	}
	return parts, nil
}

type FinishedMsg struct {
	Err  error
	Path string
}
