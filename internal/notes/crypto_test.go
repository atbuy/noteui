package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptAndDecryptBodyRoundTrip(t *testing.T) {
	encoded, err := EncryptBody("secret note body", "passphrase")
	if err != nil {
		t.Fatalf("EncryptBody returned error: %v", err)
	}
	if encoded == "" {
		t.Fatal("expected encrypted payload")
	}

	plaintext, err := DecryptBody(encoded, "passphrase")
	if err != nil {
		t.Fatalf("DecryptBody returned error: %v", err)
	}
	if plaintext != "secret note body" {
		t.Fatalf("expected decrypted plaintext, got %q", plaintext)
	}

	if _, err := DecryptBody(encoded, "wrong"); err == nil {
		t.Fatal("expected wrong-passphrase error")
	}
	if _, err := DecryptBody("abc", "passphrase"); err == nil {
		t.Fatal("expected malformed input error")
	}
}

func TestPrepareForEditAndDecryptForPreview(t *testing.T) {
	encrypted, err := EncryptBody("Plain body", "passphrase")
	if err != nil {
		t.Fatalf("EncryptBody returned error: %v", err)
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
		t.Fatalf("DecryptForPreview returned error: %v", err)
	}
	if !strings.Contains(preview, "encrypted: true") || !strings.HasSuffix(preview, "Plain body") {
		t.Fatalf("unexpected preview content: %q", preview)
	}

	editable, err := PrepareForEdit(raw, "passphrase")
	if err != nil {
		t.Fatalf("PrepareForEdit returned error: %v", err)
	}
	if strings.Contains(editable, "encrypted: true") {
		t.Fatalf("expected encrypted flag to be removed, got %q", editable)
	}
	if !strings.HasSuffix(editable, "Plain body") {
		t.Fatalf("expected decrypted body, got %q", editable)
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
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if err := EncryptNoteFile(path, "passphrase"); err != nil {
		t.Fatalf("EncryptNoteFile returned error: %v", err)
	}
	encryptedData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(encryptedData), "encrypted: true") {
		t.Fatalf("expected encrypted flag, got %q", string(encryptedData))
	}
	if strings.Contains(string(encryptedData), "Visible body") {
		t.Fatalf("expected body to be encrypted, got %q", string(encryptedData))
	}

	if err := DecryptNoteFile(path, "passphrase"); err != nil {
		t.Fatalf("DecryptNoteFile returned error: %v", err)
	}
	decryptedData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.Contains(string(decryptedData), "encrypted: true") {
		t.Fatalf("expected encrypted flag to be removed, got %q", string(decryptedData))
	}
	if !strings.Contains(string(decryptedData), "Visible body") {
		t.Fatalf("expected plaintext body, got %q", string(decryptedData))
	}
}

func TestReencryptFromTempRenamesOriginalBasedOnEditedTitle(t *testing.T) {
	dir := t.TempDir()
	origPath := filepath.Join(dir, "draft.md")
	tempPath := filepath.Join(dir, "edit.md")

	if err := os.WriteFile(origPath, []byte("placeholder"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(tempPath, []byte("# Renamed Secret\n\nBody text"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	newPath, err := ReencryptFromTemp(origPath, tempPath, "passphrase")
	if err != nil {
		t.Fatalf("ReencryptFromTemp returned error: %v", err)
	}
	if filepath.Base(newPath) != "renamed-secret.md" {
		t.Fatalf("expected renamed encrypted note path, got %q", newPath)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("expected temp file to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(origPath); !os.IsNotExist(err) {
		t.Fatalf("expected original file to be replaced, stat err=%v", err)
	}

	raw := mustRead(t, newPath)
	if !strings.Contains(raw, "encrypted: true") {
		t.Fatalf("expected encrypted flag in rewritten note, got %q", raw)
	}
	decrypted, err := DecryptBody(strings.TrimSpace(StripFrontMatter(raw)), "passphrase")
	if err != nil {
		t.Fatalf("DecryptBody returned error: %v", err)
	}
	if decrypted != "# Renamed Secret\n\nBody text" {
		t.Fatalf("unexpected decrypted body: %q", decrypted)
	}
}

func TestUniqueTrashNameAndBuildTrashInfo(t *testing.T) {
	filesDir := t.TempDir()
	infoDir := t.TempDir()

	writeFile(t, filepath.Join(filesDir, "note.md"), "body")
	writeFile(t, filepath.Join(infoDir, "note-2.md.trashinfo"), "info")

	if got := uniqueTrashName(filesDir, infoDir, "note.md"); got != "note-3.md" {
		t.Fatalf("expected note-3.md, got %q", got)
	}

	info := buildTrashInfo("/tmp/path with spaces/note.md")
	if !strings.Contains(info, "[Trash Info]") {
		t.Fatalf("expected trash info header, got %q", info)
	}
	if !strings.Contains(info, "Path=/tmp/path%20with%20spaces/note.md") {
		t.Fatalf("expected escaped path, got %q", info)
	}
	if !strings.Contains(info, "DeletionDate=") {
		t.Fatalf("expected deletion date, got %q", info)
	}
}
