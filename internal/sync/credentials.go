package sync

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"

	"atbuy/noteui/internal/config"
)

func resolveCredentialValue(envName string) (string, error) {
	envName = strings.TrimSpace(envName)
	if envName == "" {
		return "", nil
	}
	if value := os.Getenv(envName); value != "" {
		return value, nil
	}

	values, err := loadCredentialFallbacks()
	if err != nil {
		return "", err
	}
	return values[envName], nil
}

func loadCredentialFallbacks() (map[string]string, error) {
	path, err := config.ResolveSecretsPath()
	if err != nil {
		return nil, fmt.Errorf("resolve secrets file path: %w", err)
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read secrets file %s: %w", path, err)
	}
	values, err := decodeCredentialFallbacks(data)
	if err != nil {
		return nil, fmt.Errorf("parse secrets file %s: %w", path, err)
	}

	return values, nil
}

func decodeCredentialFallbacks(data []byte) (map[string]string, error) {
	values := make(map[string]string)
	if len(data) == 0 {
		return values, nil
	}
	if _, err := toml.Decode(string(data), &values); err != nil {
		return nil, err
	}
	return values, nil
}
