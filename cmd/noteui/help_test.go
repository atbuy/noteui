package main

import (
	"strings"
	"testing"
)

func TestPrintHelpContainsExpectedSections(t *testing.T) {
	var buf strings.Builder
	printHelp(&buf)
	out := buf.String()

	for _, want := range []string{"USAGE", "FLAGS", "ENVIRONMENT", "EXAMPLES", "--help", "--version", "--demo", "NOTES_ROOT", "NOTEUI_CONFIG"} {
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
