package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/buildinfo"
	"atbuy/noteui/internal/tui"
)

const banner = `
███╗   ██╗ ██████╗ ████████╗███████╗██╗   ██╗██╗
████╗  ██║██╔═══██╗╚══██╔══╝██╔════╝██║   ██║██║
██╔██╗ ██║██║   ██║   ██║   █████╗  ██║   ██║██║
██║╚██╗██║██║   ██║   ██║   ██╔══╝  ██║   ██║██║
██║ ╚████║╚██████╔╝   ██║   ███████╗╚██████╔╝██║
╚═╝  ╚═══╝ ╚═════╝    ╚═╝   ╚══════╝ ╚═════╝ ╚═╝`

type flagDef struct {
	names string
	desc  string
}

type envDef struct {
	name string
	desc string
}

var helpFlags = []flagDef{
	{"-h, --help", "Show this help message"},
	{"-v, --version", "Print version and exit"},
	{"    --demo", "Launch in demo mode with sample notes"},
	{"-w, --capture", "Append text to inbox.md without opening the TUI"},
	{"   +themes", "List all available themes with color previews"},
}

var helpEnvs = []envDef{
	{"NOTES_ROOT", "Override the default notes root directory"},
	{"NOTEUI_CONFIG", "Path to a custom config.toml"},
}

func printHelp(w io.Writer) {
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color("#8866CC"))
	bold := lipgloss.NewStyle().Bold(true)
	flagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9E7CC0"))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#A8A8A8"))
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#6AAF6A"))

	// strings.Builder.Write never returns an error; discard explicitly.
	var sb strings.Builder
	p := func(format string, args ...any) { _, _ = fmt.Fprintf(&sb, format, args...) }

	p("%s\n\n", accent.Render(banner))
	p("  %s  %s\n\n", muted.Render("v"+buildinfo.Version), muted.Render("keyboard-driven terminal notes"))

	p("%s\n\n  noteui %s\n\n", bold.Render("USAGE"), muted.Render("[flags]"))

	p("%s\n\n", bold.Render("FLAGS"))
	for _, f := range helpFlags {
		p("  %s  %s\n", flagStyle.Render(fmt.Sprintf("%-18s", f.names)), muted.Render(f.desc))
	}
	p("\n")

	p("%s\n\n", bold.Render("ENVIRONMENT"))
	for _, e := range helpEnvs {
		p("  %s  %s\n", flagStyle.Render(fmt.Sprintf("%-16s", e.name)), muted.Render(e.desc))
	}
	p("\n")

	p("%s\n\n", bold.Render("EXAMPLES"))
	p("  %s  %s\n", green.Render(fmt.Sprintf("%-26s", "noteui")), muted.Render("Start noteui"))
	p("  %s  %s\n", green.Render(fmt.Sprintf("%-26s", "noteui --demo")), muted.Render("Try noteui with sample notes"))
	p("  %s  %s\n", green.Render(fmt.Sprintf("%-26s", "NOTES_ROOT=~/work noteui")), muted.Render("Use a custom notes directory"))
	p("  %s  %s\n", green.Render(fmt.Sprintf("%-26s", `noteui --capture "buy milk"`)), muted.Render("Append a quick note to inbox.md"))
	p("\n")

	_, _ = io.WriteString(w, sb.String())
}

func printError(w io.Writer, msg string) {
	errorLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#D75F5F")).
		Render("error:")
	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E5E5")).
		Render(msg)
	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render("Run 'noteui --help' for usage.")

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %s\n%s\n", errorLabel, msgStyle, hint)
	_, _ = io.WriteString(w, sb.String())
}

func printThemes(w io.Writer) {
	bold := lipgloss.NewStyle().Bold(true)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#A8A8A8"))
	subtle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	var sb strings.Builder
	p := func(format string, args ...any) { _, _ = fmt.Fprintf(&sb, format, args...) }

	p("%s\n\n", bold.Render("THEMES"))
	p("  %s\n\n", muted.Render(`Set a theme in config.toml with:  [theme]  name = "..."`))

	for _, entry := range tui.BuiltinThemes() {
		pal := entry.Palette

		// Build color swatches from key palette colors.
		swatch := func(hex string) string {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("███")
		}
		swatches := swatch(pal.BgColor) +
			swatch(pal.PanelBgColor) +
			swatch(pal.AccentColor) +
			swatch(pal.AccentSoftColor) +
			swatch(pal.TextColor) +
			swatch(pal.SuccessColor) +
			swatch(pal.ErrorColor)

		// Name styled with the theme's own accent color.
		nameStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(pal.AccentColor))
		nameField := fmt.Sprintf("%-20s", entry.Name)

		// Aliases, if any.
		aliasLine := ""
		if len(entry.Aliases) > 0 {
			aliasLine = subtle.Render("  also: "+strings.Join(entry.Aliases, ", ")) + "\n"
		}

		p("  %s  %s\n", nameStyle.Render(nameField), swatches)
		p("  %s%s\n", muted.Render(entry.Description), "")
		if aliasLine != "" {
			p("%s", aliasLine)
		}
		p("\n")
	}

	_, _ = io.WriteString(w, sb.String())
}
