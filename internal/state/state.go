package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type State struct {
	PinnedNotes         []string `json:"pinned_notes"`
	PinnedCategories    []string `json:"pinned_categories"`
	CollapsedCategories []string `json:"collapsed_categories"`
	SortByModTime       bool     `json:"sort_by_mod_time"`
}

func Load() (State, error) {
	var s State

	path, err := statePath()
	if err != nil {
		return s, err
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return s, nil
	} else if err != nil {
		return s, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	if len(data) == 0 {
		return s, nil
	}

	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, err
	}

	return s, nil
}

func Save(s State) error {
	path, err := statePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func statePath() (string, error) {
	xdgStateHome := os.Getenv("XDG_STATE_HOME")
	if strings.TrimSpace(xdgStateHome) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdgStateHome = filepath.Join(home, ".local", "state")
	}

	return filepath.Join(xdgStateHome, "noteui", "state.json"), nil
}
