package namespace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceSpecNameUsesDevboxYAMLName(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "devbox.yaml"), []byte("name: project-devbox\nimage: custom:image\n"), 0o600))

	name, err := WorkspaceSpecName(workspace)
	require.NoError(t, err)
	require.Equal(t, "project-devbox", name)
}

func TestWorkspaceSpecNameGeneratesNameWithoutYAML(t *testing.T) {
	workspace := t.TempDir()

	name, err := WorkspaceSpecName(workspace)
	require.NoError(t, err)
	require.Equal(t, WorkspaceDevboxName(workspace), name)
}

func TestWorkspaceSpecNameGeneratesNameWhenYAMLOmitsIt(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "devbox.yaml"), []byte("image: custom:image\n"), 0o600))

	name, err := WorkspaceSpecName(workspace)
	require.NoError(t, err)
	require.Equal(t, WorkspaceDevboxName(workspace), name)
}

func TestNewSpecLoadsYAMLAndOverridesName(t *testing.T) {
	workspace := t.TempDir()
	yamlSpec := "name: project-devbox\nimage: custom:image\npurpose: coding\ndotfiles: github.com/acme/dotfiles\nprivate_features:\n  - fast-storage\nsessions:\n  - name: custom\n    command: zsh\n"
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "devbox.yaml"), []byte(yamlSpec), 0o600))

	spec, err := NewSpec(workspace, "unique-devbox", "github.com/acme/ignored")
	require.NoError(t, err)
	require.Equal(t, "unique-devbox", spec.Name)
	require.Equal(t, "custom:image", spec.Image)
	require.Equal(t, []Session{{Name: "custom", Command: "zsh"}}, spec.Sessions)
	require.Equal(t, "custom", spec.SessionName())
	require.Equal(t, "coding", spec.Purpose)
	require.Equal(t, "github.com/acme/dotfiles", spec.Dotfiles)
	require.Equal(t, []string{"fast-storage"}, spec.PrivateFeatures)

	contents, err := json.Marshal(spec)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(contents, &got))
	require.Equal(t, "unique-devbox", got["name"])
	require.Equal(t, "custom:image", got["image"])
	require.Equal(t, "coding", got["purpose"])
	require.Equal(t, "github.com/acme/dotfiles", got["dotfiles"])
	require.Equal(t, []any{"fast-storage"}, got["private_features"])
	require.NotContains(t, got, "repository")
}

func TestNewSpecRejectsUnknownFieldsInsteadOfDroppingThem(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "devbox.yaml"), []byte("image: custom:image\nfuture_field: value\n"), 0o600))

	_, err := NewSpec(workspace, "demo", "")
	require.ErrorContains(t, err, "field future_field not found")
}

func TestNewSpecUsesDefaultsWithoutYAML(t *testing.T) {
	workspace := t.TempDir()

	spec, err := NewSpec(workspace, "herdr-demo-123", "github.com/acme/demo")
	require.NoError(t, err)
	contents, err := json.Marshal(spec)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(contents, &got))
	require.Equal(t, "herdr-demo-123", got["name"])
	require.Equal(t, "builtin:agents", got["image"])
	require.Equal(t, "m", got["size"])
	require.Equal(t, "private", got["access_mode"])
	require.Equal(t, "1h", got["auto_stop_idle_timeout"])
	require.Equal(t, map[string]any{"url": "github.com/acme/demo"}, got["repository"])
	require.Equal(t, "herdr", spec.SessionName())
}

func TestNewSpecAddsDefaultSessionWhenYAMLOmitsSessions(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "devbox.yaml"), []byte("image: custom:image\n"), 0o600))

	spec, err := NewSpec(workspace, "demo", "")
	require.NoError(t, err)
	require.Equal(t, []Session{{Name: "herdr", Command: "bash"}}, spec.Sessions)
	require.Equal(t, "herdr", spec.SessionName())
}

func TestNewSpecRejectsInvalidYAML(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "devbox.yaml"), []byte("name: [invalid\n"), 0o600))

	_, err := NewSpec(workspace, "demo", "")
	require.ErrorContains(t, err, "parse")
	require.ErrorContains(t, err, "devbox.yaml")
}
