package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadActionInputsUsesWorkspaceRoot(t *testing.T) {
	configDirectory := t.TempDir()
	repository := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repository, ".git"), 0o755))
	subdirectory := filepath.Join(repository, "nested")
	require.NoError(t, os.Mkdir(subdirectory, 0o755))
	contextJSON, err := json.Marshal(map[string]string{
		"focused_pane_cwd": subdirectory,
		"focused_pane_id":  "pane-1",
	})
	require.NoError(t, err)
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", configDirectory)
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", string(contextJSON))

	inputs, err := loadActionInputs()
	require.NoError(t, err)
	workspaceRoot, err := filepath.EvalSymlinks(repository)
	require.NoError(t, err)
	require.Equal(t, workspaceRoot, inputs.Workspace)
}

func TestNormalizeRepositoryURL(t *testing.T) {
	tests := map[string]string{
		"git@github.com:acme/demo.git":     "github.com/acme/demo",
		"https://github.com/acme/demo.git": "github.com/acme/demo",
		"https://gitlab.com/acme/demo.git": "https://gitlab.com/acme/demo",
	}
	for input, want := range tests {
		require.Equal(t, want, normalizeRepositoryURL(input))
	}
}
