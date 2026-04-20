package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptAndDecryptBodyRoundTrip(t *testing.T) {
	encoded, err := EncryptBody("secret note body", "passphrase")
	if err != nil {
		require.Failf(t, "assertion failed", "EncryptBody returned error: %v", err)
	}
	if encoded == "" {
		require.FailNow(t, "expected encrypted payload")
	}

	plaintext, err := DecryptBody(encoded, "passphrase")
	if err != nil {
		require.Failf(t, "assertion failed", "DecryptBody returned error: %v", err)
	}
	if plaintext != "secret note body" {
		require.Failf(t, "assertion failed", "expected decrypted plaintext, got %q", plaintext)
	}

	if _, err := DecryptBody(encoded, "wrong"); err == nil {
		require.FailNow(t, "expected wrong-passphrase error")
	}
	if _, err := DecryptBody("abc", "passphrase"); err == nil {
		require.FailNow(t, "expected malformed input error")
	}
}

func TestPrepareForEditAndDecryptForPreview(t *testing.T) {
	encrypted, err := EncryptBody("Plain body", "passphrase")
	if err != nil {
		require.Failf(t, "assertion failed", "EncryptBody returned error: %v", err)
	}
	raw := strings.Join([]string{
		"---",
		"title: Example",
		"encrypted: true",
		"private: true",
		"---",
		encrypted,
	}, "\n")

	preview, err := DecryptForPreview(raw, "passphrase")
	if err != nil {
		require.Failf(t, "assertion failed", "DecryptForPreview returned error: %v", err)
	}
	if !strings.Contains(preview, "encrypted: true") || !strings.HasSuffix(preview, "Plain body") {
		require.Failf(t, "assertion failed", "unexpected preview content: %q", preview)
	}

	editable, err := PrepareForEdit(raw, "passphrase")
	if err != nil {
		require.Failf(t, "assertion failed", "PrepareForEdit returned error: %v", err)
	}
	if strings.Contains(editable, "encrypted: true") {
		require.Failf(t, "assertion failed", "expected encrypted flag to be removed, got %q", editable)
	}
	if !strings.HasSuffix(editable, "Plain body") {
		require.Failf(t, "assertion failed", "expected decrypted body, got %q", editable)
	}
}

func TestEncryptAndDecryptNoteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	raw := strings.Join([]string{
		"---",
		"title: Note",
		"---",
		"Visible body",
	}, "\n")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}

	if err := EncryptNoteFile(path, "passphrase"); err != nil {
		require.Failf(t, "assertion failed", "EncryptNoteFile returned error: %v", err)
	}
	encryptedData, err := os.ReadFile(path)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(encryptedData), "encrypted: true") {
		require.Failf(t, "assertion failed", "expected encrypted flag, got %q", string(encryptedData))
	}
	if strings.Contains(string(encryptedData), "Visible body") {
		require.Failf(t, "assertion failed", "expected body to be encrypted, got %q", string(encryptedData))
	}

	if err := DecryptNoteFile(path, "passphrase"); err != nil {
		require.Failf(t, "assertion failed", "DecryptNoteFile returned error: %v", err)
	}
	decryptedData, err := os.ReadFile(path)
	if err != nil {
		require.Failf(t, "assertion failed", "ReadFile returned error: %v", err)
	}
	if strings.Contains(string(decryptedData), "encrypted: true") {
		require.Failf(t, "assertion failed", "expected encrypted flag to be removed, got %q", string(decryptedData))
	}
	if !strings.Contains(string(decryptedData), "Visible body") {
		require.Failf(t, "assertion failed", "expected plaintext body, got %q", string(decryptedData))
	}
}

func TestReencryptFromTempRenamesOriginalBasedOnEditedTitle(t *testing.T) {
	dir := t.TempDir()
	origPath := filepath.Join(dir, "draft.md")
	tempPath := filepath.Join(dir, "edit.md")

	if err := os.WriteFile(origPath, []byte("placeholder"), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(tempPath, []byte("# Renamed Secret\n\nBody text"), 0o644); err != nil {
		require.Failf(t, "assertion failed", "WriteFile returned error: %v", err)
	}

	newPath, err := ReencryptFromTemp(origPath, tempPath, "passphrase")
	if err != nil {
		require.Failf(t, "assertion failed", "ReencryptFromTemp returned error: %v", err)
	}
	if filepath.Base(newPath) != "renamed-secret.md" {
		require.Failf(t, "assertion failed", "expected renamed encrypted note path, got %q", newPath)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		require.Failf(t, "assertion failed", "expected temp file to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(origPath); !os.IsNotExist(err) {
		require.Failf(t, "assertion failed", "expected original file to be replaced, stat err=%v", err)
	}

	raw := mustRead(t, newPath)
	if !strings.Contains(raw, "encrypted: true") {
		require.Failf(t, "assertion failed", "expected encrypted flag in rewritten note, got %q", raw)
	}
	decrypted, err := DecryptBody(strings.TrimSpace(StripFrontMatter(raw)), "passphrase")
	if err != nil {
		require.Failf(t, "assertion failed", "DecryptBody returned error: %v", err)
	}
	if decrypted != "# Renamed Secret\n\nBody text" {
		require.Failf(t, "assertion failed", "unexpected decrypted body: %q", decrypted)
	}
}

func TestReencryptBodyRenamesOriginalBasedOnEditedTitle(t *testing.T) {
	dir := t.TempDir()
	origPath := filepath.Join(dir, "draft.md")
	encrypted, err := EncryptBody("# Draft\n\nSecret body", "passphrase")
	require.NoError(t, err)

	raw := strings.Join([]string{
		"---",
		"title: Draft",
		"encrypted: true",
		"---",
		encrypted,
	}, "\n")
	require.NoError(t, os.WriteFile(origPath, []byte(raw), 0o644))

	newPath, err := ReencryptBody(origPath, "# Renamed Secret\n\nBody text", "passphrase")
	require.NoError(t, err)
	require.Equal(t, "renamed-secret.md", filepath.Base(newPath))
	_, statErr := os.Stat(origPath)
	require.True(t, os.IsNotExist(statErr))

	updated := mustRead(t, newPath)
	require.Contains(t, updated, "encrypted: true")
	decrypted, err := DecryptBody(strings.TrimSpace(StripFrontMatter(updated)), "passphrase")
	require.NoError(t, err)
	require.Equal(t, "# Renamed Secret\n\nBody text", decrypted)
}

func TestUniqueTrashNameAndBuildTrashInfo(t *testing.T) {
	filesDir := t.TempDir()
	infoDir := t.TempDir()

	writeFile(t, filepath.Join(filesDir, "note.md"), "body")
	writeFile(t, filepath.Join(infoDir, "note-2.md.trashinfo"), "info")

	if got := uniqueTrashName(filesDir, infoDir, "note.md"); got != "note-3.md" {
		require.Failf(t, "assertion failed", "expected note-3.md, got %q", got)
	}

	info := buildTrashInfo("/tmp/path with spaces/note.md")
	if !strings.Contains(info, "[Trash Info]") {
		require.Failf(t, "assertion failed", "expected trash info header, got %q", info)
	}
	if !strings.Contains(info, "Path=/tmp/path%20with%20spaces/note.md") {
		require.Failf(t, "assertion failed", "expected escaped path, got %q", info)
	}
	if !strings.Contains(info, "DeletionDate=") {
		require.Failf(t, "assertion failed", "expected deletion date, got %q", info)
	}
}
