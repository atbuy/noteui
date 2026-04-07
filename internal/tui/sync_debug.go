package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/notes"
	notesync "atbuy/noteui/internal/sync"
)

var writeClipboard = clipboard.WriteAll

type syncDebugDetails struct {
	NoteTitle         string
	RelPath           string
	NoteID            string
	FriendlyTitle     string
	FriendlySummary   string
	SuggestedAction   string
	RawError          string
	ConflictCopyPath  string
	LastSyncAt        time.Time
	LastSyncAttemptAt time.Time
}

func classifySyncIssue(rec notesync.NoteRecord, rawErr string) (string, string, string) {
	rawLower := strings.ToLower(strings.TrimSpace(rawErr))
	switch {
	case rec.Conflict != nil:
		return "Sync conflict", "Both local and remote changed. Compare the two copies and keep the version you want.", "Press `" + keys.OpenConflictCopy.Help().Key + "` or `" + keys.ShowSyncDebug.Help().Key + "` to compare local and remote, then choose which version to keep."
	case strings.Contains(rawLower, "note missing on remote"):
		return "Remote copy missing", "This note is still linked locally, but its remote copy no longer exists.", "Review the local note, then sync again if you want to recreate it remotely."
	case strings.Contains(rawLower, "dial tcp") || strings.Contains(rawLower, "connection refused") || strings.Contains(rawLower, "timeout"):
		return "Sync host unreachable", "noteui could not reach the sync host for this note.", "Check the sync profile host, network access, and whether the remote sync service is available."
	case strings.Contains(rawLower, "permission denied") || strings.Contains(rawLower, "publickey") || strings.Contains(rawLower, "authentication"):
		return "Authentication failed", "noteui reached the host but could not authenticate the sync request.", "Check SSH credentials, agent forwarding, and the configured remote binary permissions."
	default:
		return "Sync failed", "The last sync attempt for this note failed.", "Open details to review the stored sync metadata and technical error text."
	}
}

func (m Model) currentSyncDebugDetails() (*syncDebugDetails, bool) {
	note := m.currentLocalNote()
	if note == nil || (note.SyncClass != notes.SyncClassSynced && note.SyncClass != notes.SyncClassShared) {
		return nil, false
	}
	return m.syncDebugDetailsForNote(*note)
}

func (m Model) syncDebugDetailsForNote(note notes.Note) (*syncDebugDetails, bool) {
	relPath := strings.TrimSpace(note.RelPath)
	if relPath == "" {
		return nil, false
	}
	rec, ok := m.syncRecords[relPath]
	if !ok {
		return nil, false
	}
	return buildSyncDebugDetails(note.Title(), relPath, rec)
}

func (m Model) syncDebugDetailsForRelPath(relPath string) (*syncDebugDetails, bool) {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return nil, false
	}
	rec, ok := m.syncRecords[relPath]
	if !ok {
		return nil, false
	}
	return buildSyncDebugDetails("", relPath, rec)
}

func buildSyncDebugDetails(noteTitle, relPath string, rec notesync.NoteRecord) (*syncDebugDetails, bool) {
	rawErr := strings.TrimSpace(rec.LastSyncError)
	if rec.Conflict == nil && rawErr == "" {
		return nil, false
	}
	title, summary, action := classifySyncIssue(rec, rawErr)
	details := &syncDebugDetails{
		NoteTitle:         strings.TrimSpace(noteTitle),
		RelPath:           strings.TrimSpace(relPath),
		NoteID:            strings.TrimSpace(rec.ID),
		FriendlyTitle:     title,
		FriendlySummary:   summary,
		SuggestedAction:   action,
		RawError:          rawErr,
		LastSyncAt:        rec.LastSyncAt,
		LastSyncAttemptAt: rec.LastSyncAttemptAt,
	}
	if rec.Conflict != nil {
		details.ConflictCopyPath = strings.TrimSpace(rec.Conflict.CopyPath)
	}
	return details, true
}

func formatSyncTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return "never"
	}
	return ts.UTC().Format("2006-01-02 15:04:05 UTC")
}

func (m Model) syncIssuePreviewMarkdown(relPath string) string {
	details, ok := m.syncDebugDetailsForRelPath(relPath)
	if !ok {
		return ""
	}
	lines := []string{
		"## Sync status",
		"",
		"- State: **" + details.FriendlyTitle + "**",
		"- Summary: " + details.FriendlySummary,
	}
	if !details.LastSyncAttemptAt.IsZero() {
		lines = append(lines, "- Last attempt: `"+formatSyncTimestamp(details.LastSyncAttemptAt)+"`")
	}
	if details.SuggestedAction != "" {
		lines = append(lines, "- Next step: "+details.SuggestedAction)
	}
	if details.ConflictCopyPath != "" {
		lines = append(lines, "- Conflict copy: `"+details.ConflictCopyPath+"`")
	} else {
		lines = append(lines, "- Details: Press `"+keys.ShowSyncDebug.Help().Key+"` for note-level sync details.")
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderSelectedSyncIssueHint() string {
	details, ok := m.currentSyncDebugDetails()
	if !ok || details.ConflictCopyPath != "" {
		return ""
	}
	return "sync issue: press " + keys.ShowSyncDebug.Help().Key + " for details"
}

func (m *Model) openCurrentSyncDebugModal() {
	details, ok := m.currentSyncDebugDetails()
	if !ok {
		m.status = "sync details only work on unhealthy synced notes"
		return
	}
	m.showSyncDebugModal = true
	m.conflictResolutionChoice = conflictResolutionKeepLocal
	if details.ConflictCopyPath != "" {
		m.status = "resolve conflict"
		return
	}
	m.status = "sync details"
}

func (m *Model) closeSyncDebugModal(status string) {
	m.showSyncDebugModal = false
	m.conflictResolutionChoice = conflictResolutionKeepLocal
	if strings.TrimSpace(status) != "" {
		m.status = status
	}
}

func (m *Model) copyCurrentSyncDebugRawError() {
	details, ok := m.currentSyncDebugDetails()
	if !ok {
		m.status = "sync details only work on unhealthy synced notes"
		return
	}
	if details.ConflictCopyPath != "" {
		m.status = "copy is only available for non-conflict sync errors"
		return
	}
	if strings.TrimSpace(details.RawError) == "" {
		m.status = "no technical sync detail to copy"
		return
	}
	if err := writeClipboard(details.RawError); err != nil {
		m.status = "sync debug copy failed: " + err.Error()
		return
	}
	m.status = "copied sync detail to clipboard"
}

func (m Model) selectedConflictChoiceLabel() string {
	if m.conflictResolutionChoice == conflictResolutionKeepRemote {
		return "remote"
	}
	return "local"
}

func (m Model) currentConflictResolutionRecord() (*notes.Note, notesync.NoteRecord, string, bool) {
	note := m.currentLocalNote()
	if note == nil || (note.SyncClass != notes.SyncClassSynced && note.SyncClass != notes.SyncClassShared) {
		return nil, notesync.NoteRecord{}, "", false
	}
	rec, ok := m.syncRecords[filepath.ToSlash(strings.TrimSpace(note.RelPath))]
	if !ok || rec.Conflict == nil {
		return nil, notesync.NoteRecord{}, "", false
	}
	conflictPath := m.currentConflictCopyPath()
	if conflictPath == "" {
		return nil, notesync.NoteRecord{}, "", false
	}
	noteCopy := *note
	return &noteCopy, rec, conflictPath, true
}

func (m *Model) confirmCurrentConflictResolution() tea.Cmd {
	note, rec, _, ok := m.currentConflictResolutionRecord()
	if !ok || note == nil {
		return nil
	}
	m.status = "resolving conflict: keep " + m.selectedConflictChoiceLabel()
	if m.conflictResolutionChoice == conflictResolutionKeepRemote {
		return resolveConflictKeepRemoteCmd(m.rootDir, rec)
	}
	return resolveConflictKeepLocalCmd(m.rootDir, m.activeWorkspaceSyncRemoteRoot(), m.cfg.Sync, note.Path, rec)
}

func readSyncDebugFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "[unavailable: " + err.Error() + "]"
	}
	if len(raw) == 0 {
		return "[empty note]"
	}
	return string(raw)
}

func truncateModalContent(raw string, maxLines int) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "[empty note]"
	}
	lines := strings.Split(raw, "\n")
	if maxLines > 0 && len(lines) > maxLines {
		lines = append(lines[:maxLines], fmt.Sprintf("... (%d more lines)", len(strings.Split(raw, "\n"))-maxLines))
	}
	return strings.Join(lines, "\n")
}

func renderSyncDebugPane(title, subtitle, body string, width int, selected bool) string {
	borderColor := modalBorderColor
	if selected {
		borderColor = accentColor
	}
	innerWidth := max(12, width-4)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(textColor).Background(modalBgColor)
	if selected {
		titleStyle = titleStyle.Foreground(accentColor)
	}
	subtitleStyle := lipgloss.NewStyle().Foreground(mutedColor).Background(modalBgColor)
	bodyStyle := lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Foreground(textColor)
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		fillWidthBackground(titleStyle.Render(title), innerWidth, modalBgColor),
		fillWidthBackground(subtitleStyle.Render(subtitle), innerWidth, modalBgColor),
		fillWidthBackground("", innerWidth, modalBgColor),
		fillWidthBackground(bodyStyle.Render(body), innerWidth, modalBgColor),
	)
	return lipgloss.NewStyle().
		Width(width).
		Background(modalBgColor).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(content)
}

func renderSyncDebugMetadata(details *syncDebugDetails) string {
	lines := []string{}
	if details.NoteTitle != "" {
		lines = append(lines, "- Note: `"+details.NoteTitle+"`")
	}
	lines = append(lines, "- Path: `"+details.RelPath+"`")
	if details.NoteID != "" {
		lines = append(lines, "- Remote ID: `"+details.NoteID+"`")
	}
	lines = append(lines,
		"- Last attempt: `"+formatSyncTimestamp(details.LastSyncAttemptAt)+"`",
		"- Last success: `"+formatSyncTimestamp(details.LastSyncAt)+"`",
	)
	return strings.Join(lines, "\n")
}

func (m Model) renderConflictResolutionModal(details *syncDebugDetails) string {
	note, _, conflictPath, ok := m.currentConflictResolutionRecord()
	if !ok || note == nil {
		return ""
	}
	modalWidth, innerWidth := m.modalDimensions(100, 148)
	paneGap := 2
	paneWidth := max(22, (innerWidth-paneGap-8)/2)
	maxLines := max(8, min(24, m.height-20))
	leftBody := truncateModalContent(readSyncDebugFile(note.Path), maxLines)
	rightBody := truncateModalContent(readSyncDebugFile(conflictPath), maxLines)
	leftPane := renderSyncDebugPane("Keep local", filepath.ToSlash(note.RelPath), leftBody, paneWidth, m.conflictResolutionChoice == conflictResolutionKeepLocal)
	rightPane := renderSyncDebugPane("Keep remote", filepath.ToSlash(details.ConflictCopyPath), rightBody, paneWidth, m.conflictResolutionChoice == conflictResolutionKeepRemote)
	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.renderModalTitle("Resolve conflict", innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalHint(details.FriendlySummary, innerWidth),
				m.renderModalBlank(innerWidth),
				lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Render(m.renderPreviewMarkdown("<sync-debug-meta>", renderSyncDebugMetadata(details))),
				m.renderModalBlank(innerWidth),
				lipgloss.JoinHorizontal(lipgloss.Top, leftPane, lipgloss.NewStyle().Width(paneGap).Background(modalBgColor).Render(strings.Repeat(" ", paneGap)), rightPane),
				m.renderModalBlank(innerWidth),
				m.renderModalFooter("left/right or h/l choose • Enter apply • Esc close", innerWidth),
			),
		)
	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderGenericSyncDebugModal(details *syncDebugDetails) string {
	modalWidth, innerWidth := m.modalDimensions(76, 112)
	bodyLines := []string{
		"## " + details.FriendlyTitle,
		"",
		renderSyncDebugMetadata(details),
		"",
		"### What this means",
		"",
		details.FriendlySummary,
	}
	if details.SuggestedAction != "" {
		bodyLines = append(bodyLines, "", "### Next step", "", details.SuggestedAction)
	}
	if strings.TrimSpace(details.RawError) != "" {
		bodyLines = append(bodyLines, "", "### Technical detail", "", details.RawError)
	}
	body := lipgloss.NewStyle().Width(innerWidth).Background(modalBgColor).Foreground(textColor).Render(strings.Join(bodyLines, "\n"))
	content := lipgloss.NewStyle().
		Width(innerWidth).
		Background(modalBgColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.renderModalTitle("Sync details", innerWidth),
				m.renderModalBlank(innerWidth),
				m.renderModalHint("Friendly sync status for the selected note. Use copy only when you need the exact stored error text.", innerWidth),
				m.renderModalBlank(innerWidth),
				body,
				m.renderModalBlank(innerWidth),
				m.renderModalFooter("y copy detail • Esc close", innerWidth),
			),
		)
	return modalCardStyle(modalWidth).Render(content)
}

func (m Model) renderSyncDebugModal() string {
	details, ok := m.currentSyncDebugDetails()
	if !ok {
		return ""
	}
	if details.ConflictCopyPath != "" {
		return m.renderConflictResolutionModal(details)
	}
	return m.renderGenericSyncDebugModal(details)
}

func resolveConflictKeepRemoteCmd(root string, rec notesync.NoteRecord) tea.Cmd {
	return func() tea.Msg {
		err := notesync.ResolveConflictKeepRemote(root, rec)
		return conflictResolvedMsg{keepRemote: true, err: err}
	}
}

func resolveConflictKeepLocalCmd(root, remoteRootOverride string, cfg config.SyncConfig, notePath string, rec notesync.NoteRecord) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := notesync.ResolveConflictKeepLocal(ctx, root, notePath, remoteRootOverride, cfg, rec, nil)
		return conflictResolvedMsg{keepRemote: false, err: err}
	}
}
