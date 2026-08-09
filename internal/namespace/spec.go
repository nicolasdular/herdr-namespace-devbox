package namespace

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const defaultSessionName = "herdr"

// Spec is the normalized Namespace Devbox specification. It can be decoded
// from devbox.yaml and encoded directly as the JSON accepted by the Devbox CLI.
type Spec struct {
	Name                string        `json:"name" yaml:"name"`
	Purpose             string        `json:"purpose,omitempty" yaml:"purpose,omitempty"`
	Image               string        `json:"image,omitempty" yaml:"image,omitempty"`
	Size                string        `json:"size,omitempty" yaml:"size,omitempty"`
	AccessMode          string        `json:"access_mode,omitempty" yaml:"access_mode,omitempty"`
	AutoStopIdleTimeout string        `json:"auto_stop_idle_timeout,omitempty" yaml:"auto_stop_idle_timeout,omitempty"`
	Repository          *Repository   `json:"repository,omitempty" yaml:"repository,omitempty"`
	Dotfiles            string        `json:"dotfiles,omitempty" yaml:"dotfiles,omitempty"`
	PrivateFeatures     []string      `json:"private_features,omitempty" yaml:"private_features,omitempty"`
	Sessions            []Session     `json:"sessions,omitempty" yaml:"sessions,omitempty"`
	Integrations        *Integrations `json:"integrations,omitempty" yaml:"integrations,omitempty"`
	VolumeSizeGB        *int          `json:"volume_size_gb,omitempty" yaml:"volume_size_gb,omitempty"`
	Site                *string       `json:"site,omitempty" yaml:"site,omitempty"`
	Ephemeral           bool          `json:"ephemeral,omitempty" yaml:"ephemeral,omitempty"`
	Privileged          bool          `json:"privileged,omitempty" yaml:"privileged,omitempty"`
}

type Repository struct {
	Disabled bool   `json:"disabled,omitempty" yaml:"disabled,omitempty"`
	URL      string `json:"url,omitempty" yaml:"url,omitempty"`
	Ref      string `json:"ref,omitempty" yaml:"ref,omitempty"`
}

type Session struct {
	Name    string `json:"name" yaml:"name"`
	Command string `json:"command" yaml:"command"`
}

type Integrations struct {
	GitHub *GitHubIntegration `json:"github,omitempty" yaml:"github,omitempty"`
}

type GitHubIntegration struct {
	ShareAuth bool `json:"share_auth,omitempty" yaml:"share_auth,omitempty"`
}

// NewSpec loads devbox.yaml from workspace when present. Otherwise it uses the
// built-in defaults. name is always authoritative so new Devboxes can reuse a
// workspace YAML file under a unique name.
func NewSpec(workspace, name, repositoryURL string) (Spec, error) {
	spec, found, err := loadSpec(workspace)
	if err != nil {
		return Spec{}, err
	}
	if found {
		spec.Name = name
		if len(spec.Sessions) == 0 {
			spec.Sessions = defaultSessions()
		}
		return spec, nil
	}

	return defaultSpec(name, repositoryURL), nil
}

func (s Spec) SessionName() string {
	if len(s.Sessions) == 0 || s.Sessions[0].Name == "" {
		return defaultSessionName
	}
	return s.Sessions[0].Name
}

// WorkspaceSpecName returns the name declared by devbox.yaml, or the stable
// workspace name when the file is absent or does not declare one.
func WorkspaceSpecName(workspace string) (string, error) {
	spec, found, err := loadSpec(workspace)
	if err != nil {
		return "", err
	}
	if found && spec.Name != "" {
		return spec.Name, nil
	}
	return WorkspaceDevboxName(workspace), nil
}

func loadSpec(workspace string) (Spec, bool, error) {
	path := filepath.Join(workspace, "devbox.yaml")
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Spec{}, false, nil
	}
	if err != nil {
		return Spec{}, false, fmt.Errorf("read %s: %w", path, err)
	}

	var spec Spec
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	decoder.KnownFields(true)
	if err := decoder.Decode(&spec); err != nil {
		return Spec{}, false, fmt.Errorf("parse %s: %w", path, err)
	}
	return spec, true, nil
}

func defaultSpec(name, repositoryURL string) Spec {
	repository := &Repository{Disabled: true}
	if repositoryURL != "" {
		repository = &Repository{URL: repositoryURL}
	}

	spec := Spec{
		Name:                name,
		Image:               "builtin:agents",
		Size:                "m",
		AccessMode:          "private",
		AutoStopIdleTimeout: "1h",
		Repository:          repository,
		Sessions:            defaultSessions(),
	}
	return spec
}

func defaultSessions() []Session {
	return []Session{{Name: defaultSessionName, Command: "bash"}}
}
