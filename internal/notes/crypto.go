package notes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	saltLen      = 16
	nonceLen     = 12
	keyLen       = 32
	argonTime    = 1
	argonMem     = 32 * 1024
	argonThreads = 2
)

func deriveKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, argonTime, argonMem, argonThreads, keyLen)
}

// EncryptBody encrypts plaintext using AES-256-GCM with an Argon2id-derived key.
// Returns a base64-encoded blob of salt || nonce || ciphertext.
func EncryptBody(plaintext, passphrase string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	key := deriveKey(passphrase, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	blob := make([]byte, saltLen+nonceLen+len(ciphertext))
	copy(blob[:saltLen], salt)
	copy(blob[saltLen:saltLen+nonceLen], nonce)
	copy(blob[saltLen+nonceLen:], ciphertext)

	return base64.StdEncoding.EncodeToString(blob), nil
}

// DecryptBody decodes and decrypts a base64-encoded blob produced by EncryptBody.
func DecryptBody(encoded, passphrase string) (string, error) {
	blob, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", fmt.Errorf("decoding blob: %w", err)
	}

	if len(blob) < saltLen+nonceLen {
		return "", errors.New("encrypted content is too short")
	}

	salt := blob[:saltLen]
	nonce := blob[saltLen : saltLen+nonceLen]
	ciphertext := blob[saltLen+nonceLen:]

	key := deriveKey(passphrase, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("decryption failed: wrong passphrase or corrupted data")
	}

	return string(plaintext), nil
}

// DecryptForPreview decrypts the body and returns the full file content with plaintext body,
// suitable for rendering in the preview pane.
func DecryptForPreview(raw, passphrase string) (string, error) {
	body := strings.TrimSpace(StripFrontMatter(raw))
	plaintext, err := DecryptBody(body, passphrase)
	if err != nil {
		return "", err
	}
	return swapNoteBody(raw, plaintext), nil
}

// PrepareForEdit decrypts the body and strips the encrypted flag so the note can be
// opened in an external editor as plain text.
func PrepareForEdit(raw, passphrase string) (string, error) {
	body := strings.TrimSpace(StripFrontMatter(raw))
	plaintext, err := DecryptBody(body, passphrase)
	if err != nil {
		return "", err
	}
	withoutFlag := removeEncryptedFlag(raw)
	return swapNoteBody(withoutFlag, plaintext), nil
}

// EncryptNoteFile reads the note at path, encrypts its body, adds the encrypted flag
// to frontmatter, and writes the result back.
func EncryptNoteFile(path, passphrase string) error {
	raw, err := ReadAll(path)
	if err != nil {
		return err
	}

	body := StripFrontMatter(raw)
	encrypted, err := EncryptBody(body, passphrase)
	if err != nil {
		return err
	}

	withFlag := addEncryptedFlag(raw)
	final := swapNoteBody(withFlag, encrypted+"\n")

	return atomicWriteFile(path, []byte(final), 0o644)
}

// DecryptNoteFile reads the note at path, decrypts its body, removes the encrypted flag,
// and writes the plaintext back.
func DecryptNoteFile(path, passphrase string) error {
	raw, err := ReadAll(path)
	if err != nil {
		return err
	}

	body := strings.TrimSpace(StripFrontMatter(raw))
	plaintext, err := DecryptBody(body, passphrase)
	if err != nil {
		return err
	}

	withoutFlag := removeEncryptedFlag(raw)
	final := swapNoteBody(withoutFlag, plaintext)

	return atomicWriteFile(path, []byte(final), 0o644)
}

// ReencryptFromTemp re-encrypts content from a temp file back to the original path.
// It handles title-based renaming of the original file. The temp file is always deleted.
func ReencryptFromTemp(origPath, tempPath, passphrase string) (string, error) {
	defer func() { _ = os.Remove(tempPath) }()

	tempContent, err := ReadAll(tempPath)
	if err != nil {
		return origPath, fmt.Errorf("reading temp file: %w", err)
	}

	body := StripFrontMatter(tempContent)
	encrypted, err := EncryptBody(body, passphrase)
	if err != nil {
		return origPath, fmt.Errorf("encrypting: %w", err)
	}

	withFlag := addEncryptedFlag(tempContent)
	final := swapNoteBody(withFlag, encrypted+"\n")

	newPath := origPath
	newTitle := ExtractTitleOrFirstLine(tempContent)
	if newTitle != "" {
		dir := filepath.Dir(origPath)
		ext := filepath.Ext(origPath)
		if ext == "" {
			ext = ".md"
		}
		baseSlug := Slugify(newTitle)
		if baseSlug == "" {
			baseSlug = "untitled"
		}
		candidate := uniquePath(dir, baseSlug, ext, origPath)
		if candidate != origPath {
			newPath = candidate
		}
	}

	// Atomic write: the original file is not touched until the new content is
	// fully flushed to disk, so a crash or disk-full cannot corrupt origPath.
	if err := atomicWriteFile(newPath, []byte(final), 0o644); err != nil {
		return origPath, fmt.Errorf("writing encrypted file: %w", err)
	}

	if newPath != origPath {
		_ = os.Remove(origPath)
	}

	return newPath, nil
}

func addEncryptedFlag(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return "---\nencrypted: true\n---\n" + raw
	}
	rest := strings.TrimPrefix(raw, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return "---\nencrypted: true\n---\n" + raw
	}
	block := rest[:end]
	body := rest[end+len("\n---\n"):]

	var lines []string
	for _, line := range strings.Split(block, "\n") {
		if idx := strings.Index(line, ":"); idx >= 0 {
			k := strings.ToLower(strings.TrimSpace(line[:idx]))
			k = strings.ReplaceAll(k, "_", "-")
			if k == "encrypted" {
				continue
			}
		}
		lines = append(lines, line)
	}
	lines = append(lines, "encrypted: true")
	return "---\n" + strings.Join(lines, "\n") + "\n---\n" + body
}

func removeEncryptedFlag(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return raw
	}
	rest := strings.TrimPrefix(raw, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return raw
	}
	block := rest[:end]
	body := rest[end+len("\n---\n"):]

	var lines []string
	for _, line := range strings.Split(block, "\n") {
		if idx := strings.Index(line, ":"); idx >= 0 {
			k := strings.ToLower(strings.TrimSpace(line[:idx]))
			k = strings.ReplaceAll(k, "_", "-")
			if k == "encrypted" {
				continue
			}
		}
		lines = append(lines, line)
	}
	return "---\n" + strings.Join(lines, "\n") + "\n---\n" + body
}

func swapNoteBody(raw, newBody string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return newBody
	}
	rest := strings.TrimPrefix(raw, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return newBody
	}
	return "---\n" + rest[:end] + "\n---\n" + newBody
}
