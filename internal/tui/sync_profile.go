package tui

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/config"
	notesync "atbuy/noteui/internal/sync"
)

type syncProfileSavedMsg struct {
	cfg      config.Config
	path     string
	profile  string
	rebound  bool
	showInfo string
	err      error
}

type syncProfileChange struct {
	previousDefault string
	selectedDefault string
	boundProfile    string
}

const (
	syncProfileMigrationKeepCurrent = iota
	syncProfileMigrationRebindRoot
	syncProfileMigrationCancel
)

func sortedSyncProfileNames(cfg config.SyncConfig) []string {
	if len(cfg.Profiles) == 0 {
		return nil
	}
	out := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func saveDefaultSyncProfileCmd(profile string, rebound bool, showInfo string) tea.Cmd {
	return func() tea.Msg {
		cfg, path, err := config.SaveDefaultSyncProfile(profile)
		return syncProfileSavedMsg{
			cfg:      cfg,
			path:     path,
			profile:  profile,
			rebound:  rebound,
			showInfo: showInfo,
			err:      err,
		}
	}
}

func saveAndRebindDefaultSyncProfileCmd(root, profile string) tea.Cmd {
	return func() tea.Msg {
		cfg, path, err := config.SaveDefaultSyncProfile(profile)
		if err != nil {
			return syncProfileSavedMsg{profile: profile, err: err}
		}

		rootCfg, loadErr := notesync.LoadRootConfig(root)
		switch {
		case loadErr == nil:
			rootCfg.Profile = profile
		case errors.Is(loadErr, os.ErrNotExist):
			rootCfg = notesync.RootConfig{
				SchemaVersion: notesync.SchemaVersion,
				ClientID:      notesync.NewClientID(),
				Profile:       profile,
			}
		default:
			return syncProfileSavedMsg{profile: profile, err: loadErr}
		}

		if err := notesync.SaveRootConfig(root, rootCfg); err != nil {
			return syncProfileSavedMsg{profile: profile, err: err}
		}

		return syncProfileSavedMsg{
			cfg:      cfg,
			path:     path,
			profile:  profile,
			rebound:  true,
			showInfo: "root rebound",
		}
	}
}

func (m *Model) openSyncProfilePicker() {
	m.syncProfileNames = sortedSyncProfileNames(m.cfg.Sync)
	if len(m.syncProfileNames) == 0 {
		m.status = "sync profiles are not configured"
		return
	}
	m.showSyncProfilePicker = true
	m.syncProfileCursor = 0
	current := strings.TrimSpace(m.cfg.Sync.DefaultProfile)
	for i, name := range m.syncProfileNames {
		if name == current {
			m.syncProfileCursor = i
			break
		}
	}
	m.status = "select sync profile"
}

func (m *Model) closeSyncProfilePicker(status string) {
	m.showSyncProfilePicker = false
	m.syncProfileNames = nil
	m.syncProfileCursor = 0
	if strings.TrimSpace(status) != "" {
		m.status = status
	}
}

func (m *Model) moveSyncProfileCursor(delta int) {
	if len(m.syncProfileNames) == 0 {
		m.syncProfileCursor = 0
		return
	}
	m.syncProfileCursor += delta
	if m.syncProfileCursor < 0 {
		m.syncProfileCursor = 0
	}
	if m.syncProfileCursor >= len(m.syncProfileNames) {
		m.syncProfileCursor = len(m.syncProfileNames) - 1
	}
}

func (m Model) selectedSyncProfileName() string {
	if len(m.syncProfileNames) == 0 {
		return ""
	}
	cursor := m.syncProfileCursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(m.syncProfileNames) {
		cursor = len(m.syncProfileNames) - 1
	}
	return m.syncProfileNames[cursor]
}

func (m *Model) openSyncProfileMigration(boundProfile, selectedDefault string) {
	m.pendingSyncProfileChange = &syncProfileChange{
		previousDefault: strings.TrimSpace(m.cfg.Sync.DefaultProfile),
		selectedDefault: strings.TrimSpace(selectedDefault),
		boundProfile:    strings.TrimSpace(boundProfile),
	}
	m.showSyncProfileMigration = true
	m.syncProfileMigrationChoice = syncProfileMigrationKeepCurrent
	m.showSyncProfilePicker = false
	m.status = fmt.Sprintf("root is still bound to %s", boundProfile)
}

func (m *Model) closeSyncProfileMigration(status string) {
	m.showSyncProfileMigration = false
	m.syncProfileMigrationChoice = syncProfileMigrationKeepCurrent
	m.pendingSyncProfileChange = nil
	if strings.TrimSpace(status) != "" {
		m.status = status
	}
}

func (m *Model) moveSyncProfileMigrationChoice(delta int) {
	m.syncProfileMigrationChoice += delta
	if m.syncProfileMigrationChoice < syncProfileMigrationKeepCurrent {
		m.syncProfileMigrationChoice = syncProfileMigrationKeepCurrent
	}
	if m.syncProfileMigrationChoice > syncProfileMigrationCancel {
		m.syncProfileMigrationChoice = syncProfileMigrationCancel
	}
}

func (m *Model) confirmSelectedSyncProfile() tea.Cmd {
	profile := strings.TrimSpace(m.selectedSyncProfileName())
	if profile == "" {
		m.closeSyncProfilePicker("sync profiles are not configured")
		return nil
	}
	if profile == strings.TrimSpace(m.cfg.Sync.DefaultProfile) {
		m.closeSyncProfilePicker("sync profile already selected")
		return nil
	}

	rootCfg, err := notesync.LoadRootConfig(m.rootDir)
	if err == nil && strings.TrimSpace(rootCfg.Profile) != "" && strings.TrimSpace(rootCfg.Profile) != profile {
		m.openSyncProfileMigration(rootCfg.Profile, profile)
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		m.closeSyncProfilePicker("sync profile check failed: " + err.Error())
		return nil
	}

	m.closeSyncProfilePicker("")
	m.status = "saving sync profile..."
	return saveDefaultSyncProfileCmd(profile, false, "")
}

func (m *Model) confirmSyncProfileMigration() tea.Cmd {
	if m.pendingSyncProfileChange == nil {
		m.closeSyncProfileMigration("sync profile change cancelled")
		return nil
	}
	change := *m.pendingSyncProfileChange
	switch m.syncProfileMigrationChoice {
	case syncProfileMigrationKeepCurrent:
		m.closeSyncProfileMigration("")
		m.status = "saving sync profile..."
		return saveDefaultSyncProfileCmd(change.selectedDefault, false, change.boundProfile)
	case syncProfileMigrationRebindRoot:
		m.closeSyncProfileMigration("")
		m.status = "rebinding sync root..."
		return saveAndRebindDefaultSyncProfileCmd(m.rootDir, change.selectedDefault)
	default:
		m.closeSyncProfileMigration("sync profile change cancelled")
		return nil
	}
}

func (m Model) renderSyncProfilePickerModal() string {
	modalWidth, innerWidth := m.modalDimensions(54, 80)
	current := strings.TrimSpace(m.cfg.Sync.DefaultProfile)
	lines := make([]string, 0, len(m.syncProfileNames))
	for i, name := range m.syncProfileNames {
		prefix := "  "
		if i == m.syncProfileCursor {
			prefix = "› "
		}
		detail := name
		if name == current {
			detail += " (current default)"
		}
		bg := modalBgColor
		fg := textColor
		if i == m.syncProfileCursor {
			bg = selectedBgColor
			fg = selectedFgColor
		}
		lines = append(lines, lipgloss.NewStyle().Width(innerWidth).Padding(0, 1).Foreground(fg).Background(bg).Render(prefix+detail))
	}
	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render("(no sync profiles configured)"))
	}

	content := lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderModalTitle("Select sync profile", innerWidth),
			m.renderModalBlank(innerWidth),
			m.renderModalHint("Choose the default sync profile from config.toml. Existing roots can stay bound to a different profile.", innerWidth),
			m.renderModalBlank(innerWidth),
			lipgloss.JoinVertical(lipgloss.Left, lines...),
			m.renderModalBlank(innerWidth),
			m.renderModalFooter("up/down to choose • Enter to confirm • Esc to cancel", innerWidth),
		),
	)
	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderSyncProfileMigrationModal() string {
	modalWidth, innerWidth := m.modalDimensions(58, 86)
	change := m.pendingSyncProfileChange
	if change == nil {
		return ""
	}

	choices := []string{
		fmt.Sprintf("Keep current root binding on %s", change.boundProfile),
		fmt.Sprintf("Rebind this root to %s", change.selectedDefault),
		"Cancel",
	}
	rows := make([]string, 0, len(choices))
	for i, choice := range choices {
		bg := modalBgColor
		fg := textColor
		prefix := "  "
		if i == m.syncProfileMigrationChoice {
			bg = selectedBgColor
			fg = selectedFgColor
			prefix = "› "
		}
		rows = append(rows, lipgloss.NewStyle().Width(innerWidth).Padding(0, 1).Foreground(fg).Background(bg).Render(prefix+choice))
	}

	bodyLines := []string{
		fmt.Sprintf("Current root binding: %s", change.boundProfile),
		fmt.Sprintf("Selected default profile: %s", change.selectedDefault),
		"This is a root migration decision, not a note conflict resolution flow.",
	}
	body := make([]string, 0, len(bodyLines))
	for _, line := range bodyLines {
		body = append(body, lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(line))
	}

	content := lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderModalTitle("Sync root already bound", innerWidth),
			m.renderModalBlank(innerWidth),
			m.renderModalHint("Choose whether this notes root should keep its current host or be rebound to the newly selected default profile.", innerWidth),
			m.renderModalBlank(innerWidth),
			lipgloss.JoinVertical(lipgloss.Left, body...),
			m.renderModalBlank(innerWidth),
			lipgloss.JoinVertical(lipgloss.Left, rows...),
			m.renderModalBlank(innerWidth),
			m.renderModalFooter("up/down to choose • Enter to confirm • Esc to cancel", innerWidth),
		),
	)
	return modalCardStyle(modalWidth).Render(content)
}
