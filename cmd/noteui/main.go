package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"atbuy/noteui/internal/buildinfo"
	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/demo"
	"atbuy/noteui/internal/tui"
)

func main() {
	demoMode := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-version", "-v":
			fmt.Println(buildinfo.Version)
			return
		case "--demo":
			demoMode = true
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve home directory: %v\n", err)
		os.Exit(1)
	}
	fallbackRoot := filepath.Join(home, "notes")

	cfg, cfgErr := config.Load()
	startupError := ""
	if cfgErr != nil {
		startupError = cfgErr.Error()
		fmt.Fprintf(os.Stderr, "config warning: %v\n", cfgErr)
	}

	startup := config.ResolveStartupWorkspace(cfg, os.Getenv("NOTES_ROOT"), fallbackRoot)

	if demoMode {
		demoRoot, demoCleanup, demoErr := demo.Setup()
		if demoErr != nil {
			fmt.Fprintf(os.Stderr, "demo setup failed: %v\n", demoErr)
			os.Exit(1)
		}
		defer demoCleanup()
		startup = config.StartupWorkspace{Root: demoRoot, Label: "Demo", Name: "demo"}
		cfg.Sync = config.SyncConfig{}
		cfg.Dashboard = false
	}

	tui.ApplyTheme(cfg)
	tui.ApplyConfigKeys(cfg.Keys)

	if collisions := tui.ValidateKeyCollisions(); len(collisions) > 0 {
		msg := "keybinding conflicts: " + strings.Join(collisions, "; ")
		fmt.Fprintf(os.Stderr, "warning: %s\n", msg)
		if startupError == "" {
			startupError = msg
		} else {
			startupError = startupError + "; " + msg
		}
	}

	m := tui.NewWithSession(
		startup.Root,
		startupError,
		cfg,
		buildinfo.Version,
		tui.WorkspaceSession{
			Name:            strings.TrimSpace(startup.Name),
			Label:           strings.TrimSpace(startup.Label),
			Override:        startup.Override,
			StartWithPicker: startup.NeedsSelection,
		},
	)
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
