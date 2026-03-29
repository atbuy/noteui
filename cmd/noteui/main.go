package main

import (
	"fmt"
	"os"
	"path/filepath"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	root := os.Getenv("NOTES_ROOT")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to resolve home directory: %v\n", err)
			os.Exit(1)
		}
		root = filepath.Join(home, "notes")
	}

	cfg, cfgErr := config.Load()
	startupError := ""
	if cfgErr != nil {
		startupError = cfgErr.Error()
		fmt.Fprintf(os.Stderr, "config warning: %v\n", cfgErr)
	}

	tui.ApplyTheme(cfg)

	m := tui.New(root, startupError, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "program error: %v\n", err)
		os.Exit(1)
	}
}
