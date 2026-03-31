package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"atbuy/noteui/internal/buildinfo"
	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/tui"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-version" || arg == "-v" {
			fmt.Println(buildinfo.Version)
			return
		}
	}

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
	tui.ApplyConfigKeys(cfg.Keys)

	m := tui.New(root, startupError, cfg, buildinfo.Version)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "program error: %v\n", err)
		os.Exit(1)
	}
}
