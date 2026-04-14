package main

import (
	"strings"
	"testing"
)

func TestPrintHelpContainsExpectedSections(t *testing.T) {
	var buf strings.Builder
	printHelp(&buf)
	out := buf.String()

	for _, want := range []string{"USAGE", "FLAGS", "ENVIRONMENT", "EXAMPLES", "--help", "--version", "--demo", "+themes", "NOTES_ROOT", "NOTEUI_CONFIG"} {
		if !strings.Contains(out, want) {
			t.Errorf("printHelp output missing %q", want)
		}
	}
}

func TestPrintHelpContainsBanner(t *testing.T) {
	var buf strings.Builder
	printHelp(&buf)
	out := buf.String()

	if !strings.Contains(out, "noteui") {
		t.Error("printHelp output missing application name in banner")
	}
	if !strings.Contains(out, "keyboard-driven terminal notes") {
		t.Error("printHelp output missing tagline")
	}
}

func TestPrintThemesContainsAllBuiltinThemes(t *testing.T) {
	var buf strings.Builder
	printThemes(&buf)
	out := buf.String()

	for _, name := range []string{
		"default", "nord", "gruvbox", "catppuccin", "latte", "solarized-light", "paper",
		"onedark", "kanagawa", "dracula", "everforest", "tokyo-night-storm", "github-light",
		"github-dark", "carbonfox", "crimson", "dusk",
		"rose-pine", "monokai", "solarized-dark", "ayu-dark", "material", "nightfox",
	} {
		if !strings.Contains(out, name) {
			t.Errorf("printThemes output missing theme %q", name)
		}
	}

	if !strings.Contains(out, "THEMES") {
		t.Error("printThemes output missing THEMES header")
	}
	if !strings.Contains(out, "config.toml") {
		t.Error("printThemes output missing config.toml usage hint")
	}
}

func TestHelpFlagsAndEnvsAreNonEmpty(t *testing.T) {
	if len(helpFlags) == 0 {
		t.Error("helpFlags must not be empty")
	}
	if len(helpEnvs) == 0 {
		t.Error("helpEnvs must not be empty")
	}
	for _, f := range helpFlags {
		if f.names == "" || f.desc == "" {
			t.Errorf("flagDef has empty field: %+v", f)
		}
	}
	for _, e := range helpEnvs {
		if e.name == "" || e.desc == "" {
			t.Errorf("envDef has empty field: %+v", e)
		}
	}
}
