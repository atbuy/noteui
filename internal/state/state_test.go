package state

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestStatePathUsesXDGStateHomeWhenSet(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdg)

	path, err := statePath()
	if err != nil {
		t.Fatalf("statePath returned error: %v", err)
	}

	want := filepath.Join(xdg, "noteui", "state.json")
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestStatePathFallsBackToHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", home)

	path, err := statePath()
	if err != nil {
		t.Fatalf("statePath returned error: %v", err)
	}

	want := filepath.Join(home, ".local", "state", "noteui", "state.json")
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestLoadReturnsZeroValueWhenFileMissing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	s, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !reflect.DeepEqual(s, State{}) {
		t.Fatalf("expected zero-value state, got %#v", s)
	}
}

func TestLoadReturnsZeroValueWhenFileEmpty(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdg)

	path := filepath.Join(xdg, "noteui", "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	s, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !reflect.DeepEqual(s, State{}) {
		t.Fatalf("expected zero-value state, got %#v", s)
	}
}

func TestLoadRejectsInvalidJSON(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdg)

	path := filepath.Join(xdg, "noteui", "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected JSON error, got nil")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdg)

	want := State{
		PinnedNotes:         []string{"inbox/today.md", "ideas.md"},
		PinnedCategories:    []string{"inbox", "work/projects"},
		CollapsedCategories: []string{"archive"},
		SortByModTime:       true,
	}

	if err := Save(want); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	path := filepath.Join(xdg, "noteui", "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	text := string(data)
	for _, fragment := range []string{
		`"pinned_notes"`,
		`"pinned_categories"`,
		`"collapsed_categories"`,
		`"sort_by_mod_time": true`,
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected saved state to contain %q, got %q", fragment, text)
		}
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if strings.Join(got.PinnedNotes, ",") != strings.Join(want.PinnedNotes, ",") {
		t.Fatalf("expected pinned notes %v, got %v", want.PinnedNotes, got.PinnedNotes)
	}
	if strings.Join(got.PinnedCategories, ",") != strings.Join(want.PinnedCategories, ",") {
		t.Fatalf("expected pinned categories %v, got %v", want.PinnedCategories, got.PinnedCategories)
	}
	if strings.Join(got.CollapsedCategories, ",") != strings.Join(want.CollapsedCategories, ",") {
		t.Fatalf("expected collapsed categories %v, got %v", want.CollapsedCategories, got.CollapsedCategories)
	}
	if got.SortByModTime != want.SortByModTime {
		t.Fatalf("expected SortByModTime %v, got %v", want.SortByModTime, got.SortByModTime)
	}
}
