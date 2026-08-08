package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type Config struct {
	Image               string  `json:"image"`
	Size                string  `json:"size"`
	AccessMode          string  `json:"accessMode"`
	AutoStopIdleTimeout string  `json:"autoStopIdleTimeout"`
	SessionName         string  `json:"sessionName"`
	Shell               string  `json:"shell"`
	SetupGitHub         bool    `json:"setupGithub"`
	VolumeSizeGB        *int    `json:"volumeSizeGb,omitempty"`
	Site                *string `json:"site,omitempty"`
}

var Default = Config{
	Image:               "builtin:agents",
	Size:                "m",
	AccessMode:          "private",
	AutoStopIdleTimeout: "1h",
	SessionName:         "herdr",
	Shell:               "bash",
	SetupGitHub:         false,
}

var sessionNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

func Parse(data []byte) (Config, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return Config{}, errors.New("Namespace Devbox config must be a JSON object")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return Config{}, fmt.Errorf("invalid Namespace Devbox config: %w", err)
	}
	for key, value := range raw {
		if bytes.Equal(value, []byte("null")) {
			return Config{}, fmt.Errorf("config.%s has an invalid type", key)
		}
	}
	cfg := Default
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, normalizeJSONError(err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Config{}, err
	}
	if err := Validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Load(configDir string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if errors.Is(err, os.ErrNotExist) {
		return Default, nil
	}
	if err != nil {
		return Config{}, err
	}
	return Parse(data)
}

func Validate(cfg Config) error {
	if cfg.Image == "" {
		return errors.New("config.image must be a non-empty string")
	}
	if cfg.Size != "s" && cfg.Size != "m" && cfg.Size != "l" && cfg.Size != "xl" {
		return errors.New("config.size must be s, m, l, or xl")
	}
	if cfg.AccessMode != "private" && cfg.AccessMode != "shared" {
		return errors.New("config.accessMode must be private or shared")
	}
	if cfg.AutoStopIdleTimeout == "" {
		return errors.New("config.autoStopIdleTimeout must be a duration string")
	}
	if !sessionNamePattern.MatchString(cfg.SessionName) {
		return errors.New("config.sessionName contains unsupported characters")
	}
	if cfg.Shell == "" {
		return errors.New("config.shell must be a non-empty string")
	}
	if cfg.VolumeSizeGB != nil && *cfg.VolumeSizeGB <= 0 {
		return errors.New("config.volumeSizeGb must be a positive integer")
	}
	if cfg.Site != nil && *cfg.Site == "" {
		return errors.New("config.site must be a non-empty string")
	}
	return nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("Namespace Devbox config must contain one JSON object")
}

func normalizeJSONError(err error) error {
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		return fmt.Errorf("config.%s has an invalid type", typeError.Field)
	}
	return fmt.Errorf("invalid Namespace Devbox config: %w", err)
}
