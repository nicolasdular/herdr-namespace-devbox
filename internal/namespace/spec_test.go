package namespace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceSpecUsesDevboxYAMLName(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "devbox.yaml"), []byte("name: project-devbox\nimage: custom:image\n"), 0o600))

	path, name := WorkspaceSpec(workspace)
	require.Equal(t, filepath.Join(workspace, "devbox.yaml"), path)
	require.Equal(t, "project-devbox", name)
}

func TestWorkspaceSpecAllowsMissingFile(t *testing.T) {
	path, name := WorkspaceSpec(t.TempDir())
	require.Empty(t, path)
	require.Empty(t, name)
}

func TestWorkspaceSpecLeavesValidationToDevbox(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "devbox.yaml"), []byte("image: custom:image\n"), 0o600))
	path, name := WorkspaceSpec(workspace)
	require.NotEmpty(t, path)
	require.Empty(t, name)
}
