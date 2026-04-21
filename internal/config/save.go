package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"atbuy/noteui/internal/fsutil"
)

func ResolvePath() (string, error) {
	return resolveConfigFile("config.toml")
}

func ResolveSecretsPath() (string, error) {
	return resolveConfigFile("secrets.toml")
}

func resolveConfigFile(name string) (string, error) {
	path := os.Getenv("NOTEUI_CONFIG")
	if name == "config.toml" && strings.TrimSpace(path) != "" {
		return path, nil
	}

	userCfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCfgDir, "noteui", name), nil
}

// SaveTheme updates theme.name in the config file and returns the previous
// theme name and the path of the file that was written.
func SaveTheme(name string) (oldName, configPath string, err error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if !IsValidThemeName(name) {
		return "", "", Validate(Config{Theme: ThemeConfig{Name: name}})
	}

	path, err := ResolvePath()
	if err != nil {
		return "", "", err
	}

	cfg := loadForMutation(path)
	oldName = strings.TrimSpace(cfg.Theme.Name)
	if oldName == "" {
		oldName = Default().Theme.Name
	}
	if err := updateConfigString(path, "theme", "name", name, Default().Theme.Name); err != nil {
		return "", "", err
	}
	return oldName, path, nil
}

func SaveDefaultSyncProfile(profile string) (Config, string, error) {
	profile = strings.TrimSpace(profile)

	path, err := ResolvePath()
	if err != nil {
		return Config{}, "", err
	}

	cfg := loadForMutation(path)
	cfg.Sync.DefaultProfile = profile
	if err := updateConfigString(path, "sync", "default_profile", profile, Default().Sync.DefaultProfile); err != nil {
		return Config{}, "", err
	}
	return cfg, path, nil
}

func SaveRelativeLineNumbers(enabled bool) error {
	path, err := ResolvePath()
	if err != nil {
		return err
	}
	return updateConfigBool(path, "preview", "relative_line_numbers", enabled, Default().Preview.RelativeLineNumbers)
}

func loadForMutation(path string) Config {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	_, _ = toml.Decode(string(data), &cfg)
	return cfg
}

func updateConfigBool(path, section, key string, value, defaultValue bool) error {
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		raw = nil
	default:
		return err
	}
	updated, changed := updateTOMLBoolKey(raw, section, key, value, defaultValue)
	if !changed {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, updated, 0o644)
}

func updateTOMLBoolKey(raw []byte, section, key string, value, defaultValue bool) ([]byte, bool) {
	newline := detectNewline(raw)
	lines := splitLinesPreserveEndings(string(raw))
	remove := value == defaultValue

	start, end, foundSection := findSection(lines, section)
	if !foundSection {
		if remove {
			return raw, false
		}
		lines = appendBoolSection(lines, section, key, value, newline)
		return []byte(strings.Join(lines, "")), true
	}

	val := "false"
	if value {
		val = "true"
	}
	for i := start + 1; i < end; i++ {
		_, _, ok := matchKeyLine(lines[i], key)
		if !ok {
			continue
		}
		if remove {
			lines = append(lines[:i], lines[i+1:]...)
			lines = pruneEmptySection(lines, start, end-1)
		} else {
			base := strings.TrimRight(lines[i], "\r\n")
			indent := base[:len(base)-len(strings.TrimLeft(base, " \t"))]
			lines[i] = indent + key + " = " + val + lineEnding(lines[i], newline)
		}
		return []byte(strings.Join(lines, "")), true
	}

	if remove {
		return raw, false
	}

	insertAt := end
	if insertAt > 0 && !hasLineEnding(lines[insertAt-1]) {
		lines[insertAt-1] += newline
	}
	lines = append(lines[:insertAt], append([]string{key + " = " + val + newline}, lines[insertAt:]...)...)
	return []byte(strings.Join(lines, "")), true
}

func appendBoolSection(lines []string, section, key string, value bool, newline string) []string {
	if len(lines) > 0 {
		last := len(lines) - 1
		if !hasLineEnding(lines[last]) {
			lines[last] += newline
		}
		if strings.TrimSpace(strings.TrimRight(lines[last], "\r\n")) != "" {
			lines = append(lines, newline)
		}
	}
	val := "false"
	if value {
		val = "true"
	}
	lines = append(lines,
		"["+section+"]"+newline,
		key+" = "+val+newline,
	)
	return lines
}

func updateConfigString(path, section, key, value, defaultValue string) error {
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		raw = nil
	default:
		return err
	}

	updated, changed := updateTOMLStringKey(raw, section, key, value, defaultValue)
	if !changed {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, updated, 0o644)
}

func updateTOMLStringKey(raw []byte, section, key, value, defaultValue string) ([]byte, bool) {
	newline := detectNewline(raw)
	lines := splitLinesPreserveEndings(string(raw))
	remove := strings.TrimSpace(value) == strings.TrimSpace(defaultValue)

	start, end, foundSection := findSection(lines, section)
	if !foundSection {
		if remove {
			return raw, false
		}
		lines = appendSection(lines, section, key, value, newline)
		return []byte(strings.Join(lines, "")), true
	}

	for i := start + 1; i < end; i++ {
		indent, comment, ok := matchKeyLine(lines[i], key)
		if !ok {
			continue
		}
		if remove {
			lines = append(lines[:i], lines[i+1:]...)
			lines = pruneEmptySection(lines, start, end-1)
		} else {
			lines[i] = buildKeyLine(indent, key, value, comment, lineEnding(lines[i], newline))
		}
		return []byte(strings.Join(lines, "")), true
	}

	if remove {
		return raw, false
	}

	insertAt := end
	if insertAt > 0 && !hasLineEnding(lines[insertAt-1]) {
		lines[insertAt-1] += newline
	}
	inserted := buildKeyLine("", key, value, "", newline)
	lines = append(lines[:insertAt], append([]string{inserted}, lines[insertAt:]...)...)
	return []byte(strings.Join(lines, "")), true
}

func detectNewline(raw []byte) string {
	if bytes.Contains(raw, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func splitLinesPreserveEndings(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.SplitAfter(raw, "\n")
}

func findSection(lines []string, section string) (start int, end int, found bool) {
	start = -1
	end = len(lines)
	for i, line := range lines {
		name, ok := parseTableHeader(line)
		if !ok {
			continue
		}
		if start >= 0 {
			end = i
			break
		}
		if name == section {
			start = i
			found = true
		}
	}
	if !found {
		return -1, len(lines), false
	}
	return start, end, true
}

func parseTableHeader(line string) (string, bool) {
	trimmed := strings.TrimSpace(strings.TrimRight(line, "\r\n"))
	if strings.HasPrefix(trimmed, "[[") || !strings.HasPrefix(trimmed, "[") {
		return "", false
	}
	end := strings.Index(trimmed, "]")
	if end <= 1 {
		return "", false
	}
	if strings.Contains(trimmed[1:end], "[") {
		return "", false
	}
	rest := strings.TrimSpace(trimmed[end+1:])
	if rest != "" && !strings.HasPrefix(rest, "#") {
		return "", false
	}
	return strings.TrimSpace(trimmed[1:end]), true
}

func matchKeyLine(line, key string) (indent string, comment string, ok bool) {
	base := strings.TrimRight(line, "\r\n")
	leftTrimmed := strings.TrimLeft(base, " \t")
	if leftTrimmed == "" || strings.HasPrefix(leftTrimmed, "#") {
		return "", "", false
	}
	if !strings.HasPrefix(leftTrimmed, key) {
		return "", "", false
	}
	rest := leftTrimmed[len(key):]
	rest = strings.TrimLeft(rest, " \t")
	if !strings.HasPrefix(rest, "=") {
		return "", "", false
	}
	indent = base[:len(base)-len(leftTrimmed)]
	_, comment = splitInlineComment(base)
	return indent, comment, true
}

func splitInlineComment(line string) (string, string) {
	inBasicString := false
	inLiteralString := false
	escaped := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\\':
			if inBasicString && !escaped {
				escaped = true
				continue
			}
		case '"':
			if !inLiteralString && !escaped {
				inBasicString = !inBasicString
			}
		case '\'':
			if !inBasicString && !escaped {
				inLiteralString = !inLiteralString
			}
		case '#':
			if !inBasicString && !inLiteralString {
				return line[:i], line[i:]
			}
		}
		escaped = false
	}
	return line, ""
}

func buildKeyLine(indent, key, value, comment, newline string) string {
	line := indent + key + " = " + tomlQuote(value)
	if comment != "" {
		if !strings.HasPrefix(comment, " ") && !strings.HasPrefix(comment, "\t") {
			line += " "
		}
		line += comment
	}
	return line + newline
}

func appendSection(lines []string, section, key, value, newline string) []string {
	if len(lines) > 0 {
		last := len(lines) - 1
		if !hasLineEnding(lines[last]) {
			lines[last] += newline
		}
		if strings.TrimSpace(strings.TrimRight(lines[last], "\r\n")) != "" {
			lines = append(lines, newline)
		}
	}
	lines = append(lines,
		"["+section+"]"+newline,
		buildKeyLine("", key, value, "", newline),
	)
	return lines
}

func pruneEmptySection(lines []string, start, end int) []string {
	if start < 0 || start >= len(lines) {
		return lines
	}
	if end > len(lines) {
		end = len(lines)
	}
	for i := start + 1; i < end; i++ {
		if strings.TrimSpace(strings.TrimRight(lines[i], "\r\n")) != "" {
			return lines
		}
	}
	removeEnd := end
	for removeEnd < len(lines) && strings.TrimSpace(strings.TrimRight(lines[removeEnd], "\r\n")) == "" {
		removeEnd++
	}
	return append(lines[:start], lines[removeEnd:]...)
}

func lineEnding(line, fallback string) string {
	switch {
	case strings.HasSuffix(line, "\r\n"):
		return "\r\n"
	case strings.HasSuffix(line, "\n"):
		return "\n"
	default:
		return fallback
	}
}

func hasLineEnding(line string) bool {
	return strings.HasSuffix(line, "\n")
}

func tomlQuote(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	)
	return `"` + replacer.Replace(value) + `"`
}
