package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type outputRunner struct {
	executable string
	args       []string
	output     []byte
	err        error
}

func (r *outputRunner) CombinedOutput(_ context.Context, executable string, args ...string) ([]byte, error) {
	r.executable = executable
	r.args = append([]string(nil), args...)
	return r.output, r.err
}

func (*outputRunner) Run(context.Context, string, []string, io.Reader, io.Writer, io.Writer) error {
	panic("unexpected interactive command")
}

func TestLoadActionInputsUsesWorkspaceRoot(t *testing.T) {
	repository := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repository, ".git"), 0o755))
	subdirectory := filepath.Join(repository, "nested")
	require.NoError(t, os.Mkdir(subdirectory, 0o755))
	contextJSON, err := json.Marshal(map[string]string{
		"focused_pane_cwd": subdirectory,
		"focused_pane_id":  "pane-1",
	})
	require.NoError(t, err)
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", string(contextJSON))
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", "/plugin/config")

	inputs, err := loadActionInputs()
	require.NoError(t, err)
	workspaceRoot, err := filepath.EvalSymlinks(repository)
	require.NoError(t, err)
	require.Equal(t, workspaceRoot, inputs.Workspace)
	require.Equal(t, "/plugin/config", inputs.PluginConfigDir)
}

func TestLoadActionInputsDoesNotRequireFocusedPane(t *testing.T) {
	workspace := t.TempDir()
	contextJSON, err := json.Marshal(map[string]string{
		"workspace_cwd": workspace,
	})
	require.NoError(t, err)
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", string(contextJSON))
	t.Setenv("HERDR_PANE_ID", "")

	inputs, err := loadActionInputs()
	require.NoError(t, err)
	resolvedWorkspace, err := filepath.EvalSymlinks(workspace)
	require.NoError(t, err)
	require.Equal(t, resolvedWorkspace, inputs.Workspace)
	require.Empty(t, inputs.PaneID)
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

func TestTabTitleUsesWorkspaceDirectory(t *testing.T) {
	require.Equal(t, "Devbox · demo", tabTitle("/workspace/demo"))
}

func TestRepositoryURLBuildsGitCommand(t *testing.T) {
	runner := &outputRunner{output: []byte("git@github.com:acme/demo.git\n")}

	got := repositoryURL(context.Background(), runner, "/workspace/demo")

	require.Equal(t, "github.com/acme/demo", got)
	require.Equal(t, "git", runner.executable)
	require.Equal(t, []string{"-C", "/workspace/demo", "remote", "get-url", "origin"}, runner.args)
}
