package namespace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const pluginConfigName = "config.json"

// CreateOptions contains Herdr-specific options that are passed as flags to
// the Namespace CLI rather than included in the native Devbox specification.
type CreateOptions struct {
	Dotfiles string `json:"dotfiles,omitempty"`
}

func LoadCreateOptions(configDirectory string) (CreateOptions, error) {
	if configDirectory == "" {
		return CreateOptions{}, nil
	}
	path := filepath.Join(configDirectory, pluginConfigName)
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return CreateOptions{}, nil
	}
	if err != nil {
		return CreateOptions{}, fmt.Errorf("read %s: %w", path, err)
	}

	var options CreateOptions
	if err := json.Unmarshal(contents, &options); err != nil {
		return CreateOptions{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return options, nil
}
