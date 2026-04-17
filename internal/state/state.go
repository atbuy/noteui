// Package state persists local UI state such as pins, sort preference, and recent commands.
package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"atbuy/noteui/internal/fsutil"
)

type WorkspaceState struct {
	PinnedNotes         []string `json:"pinned_notes,omitempty"`
	PinnedCategories    []string `json:"pinned_categories,omitempty"`
	CollapsedCategories []string `json:"collapsed_categories,omitempty"`
	RecentCommands      []string `json:"recent_commands,omitempty"`
	SortMethod          string   `json:"sort_method,omitempty"`
	SortReverse         bool     `json:"sort_reverse,omitempty"`
	SortByModTime       bool     `json:"sort_by_mod_time,omitempty"` // legacy: migrated to SortMethod on load
}

type State struct {
	Workspaces map[string]WorkspaceState
}

type persistedState struct {
	Workspaces          map[string]WorkspaceState `json:"workspaces,omitempty"`
	PinnedNotes         []string                  `json:"pinned_notes,omitempty"`
	PinnedCategories    []string                  `json:"pinned_categories,omitempty"`
	CollapsedCategories []string                  `json:"collapsed_categories,omitempty"`
	RecentCommands      []string                  `json:"recent_commands,omitempty"`
	SortByModTime       bool                      `json:"sort_by_mod_time,omitempty"`
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

	var persisted persistedState
	if err := json.Unmarshal(data, &persisted); err != nil {
		return State{}, err
	}

	s.Workspaces = persisted.Workspaces
	migrateSortMethod(s.Workspaces)
	if len(s.Workspaces) == 0 {
		sortMethod := ""
		if persisted.SortByModTime {
			sortMethod = "modified"
		}
		legacy := WorkspaceState{
			PinnedNotes:         persisted.PinnedNotes,
			PinnedCategories:    persisted.PinnedCategories,
			CollapsedCategories: persisted.CollapsedCategories,
			RecentCommands:      persisted.RecentCommands,
			SortMethod:          sortMethod,
		}
		if !isZeroWorkspaceState(legacy) {
			s.Workspaces = map[string]WorkspaceState{defaultWorkspaceKey: legacy}
		}
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

	persisted := persistedState{Workspaces: normalizeWorkspaceMap(s.Workspaces)}
	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}

	return fsutil.WriteFileAtomic(path, data, 0o644)
}

func (s State) Workspace(name string) WorkspaceState {
	if len(s.Workspaces) == 0 {
		return WorkspaceState{}
	}
	return s.Workspaces[normalizeWorkspaceKey(name)]
}

func (s *State) SetWorkspace(name string, ws WorkspaceState) {
	if s.Workspaces == nil {
		s.Workspaces = make(map[string]WorkspaceState)
	}
	key := normalizeWorkspaceKey(name)
	if isZeroWorkspaceState(ws) {
		delete(s.Workspaces, key)
		return
	}
	s.Workspaces[key] = ws
}

func normalizeWorkspaceMap(items map[string]WorkspaceState) map[string]WorkspaceState {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]WorkspaceState, len(items))
	for name, ws := range items {
		key := normalizeWorkspaceKey(name)
		if isZeroWorkspaceState(ws) {
			continue
		}
		out[key] = ws
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

const defaultWorkspaceKey = "default"

func normalizeWorkspaceKey(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultWorkspaceKey
	}
	return name
}

func isZeroWorkspaceState(ws WorkspaceState) bool {
	return len(ws.PinnedNotes) == 0 &&
		len(ws.PinnedCategories) == 0 &&
		len(ws.CollapsedCategories) == 0 &&
		len(ws.RecentCommands) == 0 &&
		ws.SortMethod == "" &&
		!ws.SortReverse &&
		!ws.SortByModTime
}

func migrateSortMethod(workspaces map[string]WorkspaceState) {
	for key, ws := range workspaces {
		if ws.SortMethod == "" && ws.SortByModTime {
			ws.SortMethod = "modified"
			ws.SortByModTime = false
			workspaces[key] = ws
		}
	}
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
