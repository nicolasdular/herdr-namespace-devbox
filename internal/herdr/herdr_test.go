package herdr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUsesProvidedExecutable(t *testing.T) {
	client := New("/custom/bin/herdr")
	require.Equal(t, "/custom/bin/herdr", client.executable)
}

func TestNewFallsBackToHerdrOnPath(t *testing.T) {
	client := New("")
	require.Equal(t, "herdr", client.executable)
}

func TestContextPrefersFocusedPaneCWD(t *testing.T) {
	context, err := ParseContext(`{"workspace_cwd":"/workspace","focused_pane_cwd":"/workspace/subdir"}`)
	require.NoError(t, err)
	require.Equal(t, "/workspace/subdir", context.Workspace())
}

func TestWorkspaceRootFindsGitRootFromSubdirectory(t *testing.T) {
	repository := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repository, ".git"), 0o755))
	subdirectory := filepath.Join(repository, "some", "nested", "directory")
	require.NoError(t, os.MkdirAll(subdirectory, 0o755))

	got, err := WorkspaceRoot(subdirectory)
	require.NoError(t, err)
	want, err := filepath.EvalSymlinks(repository)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestWorkspaceRootFallsBackToDirectoryOutsideGit(t *testing.T) {
	workspace := t.TempDir()
	got, err := WorkspaceRoot(workspace)
	require.NoError(t, err)
	want, err := filepath.EvalSymlinks(workspace)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
