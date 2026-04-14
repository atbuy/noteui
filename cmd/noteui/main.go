package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"atbuy/noteui/internal/buildinfo"
	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/demo"
	"atbuy/noteui/internal/notes"
	"atbuy/noteui/internal/tui"
)

func main() {
	args := os.Args[1:]
	captureMode := false
	captureText := ""
	demoMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			printHelp(os.Stdout)
			return
		case "--version", "-version", "-v":
			fmt.Println(buildinfo.Version)
			return
		case "+themes":
			printThemes(os.Stdout)
			return
		case "+set-theme":
			if i+1 >= len(args) {
				printError(os.Stderr, "usage: noteui +set-theme <name>  (run 'noteui +themes' to see available themes)")
				os.Exit(1)
			}
			name := args[i+1]
			if !config.IsValidThemeName(name) {
				printError(os.Stderr, fmt.Sprintf("unknown theme %q - run 'noteui +themes' to see available themes", name))
				os.Exit(1)
			}
			canonical := tui.NormalizeThemeName(name)
			oldName, configPath, saveErr := config.SaveTheme(canonical)
			if saveErr != nil {
				printError(os.Stderr, fmt.Sprintf("failed to save config: %v", saveErr))
				os.Exit(1)
			}
			var newEntry tui.BuiltinThemeEntry
			for _, entry := range tui.BuiltinThemes() {
				if entry.Name == canonical {
					newEntry = entry
					break
				}
			}
			printThemeChanged(os.Stdout, oldName, canonical, configPath, newEntry)
			return
		case "--capture", "-w":
			captureMode = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				captureText = args[i+1]
				i++
			}
		case "--demo":
			demoMode = true
		default:
			printError(os.Stderr, fmt.Sprintf("unknown option %q", args[i]))
			os.Exit(1)
		}
	}

	if captureMode {
		if captureText == "" {
			stat, _ := os.Stdin.Stat()
			if stat.Mode()&os.ModeCharDevice != 0 {
				fmt.Fprintln(os.Stderr, "error: provide text as argument or pipe via stdin")
				os.Exit(1)
			}
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
				os.Exit(1)
			}
			captureText = strings.TrimRight(string(b), "\n")
		}
		if strings.TrimSpace(captureText) == "" {
			fmt.Fprintln(os.Stderr, "error: capture text is empty")
			os.Exit(1)
		}
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		cfg, _ := config.Load()
		startup := config.ResolveStartupWorkspace(cfg, os.Getenv("NOTES_ROOT"), filepath.Join(home, "notes"))
		if err := notes.AppendCapture(startup.Root, "inbox.md", captureText); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
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
