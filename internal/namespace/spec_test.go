package namespace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceSpecUsesDevboxYAMLName(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "devbox.yaml"), []byte("name: project-devbox\nimage: custom:image\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	path, name := WorkspaceSpec(workspace)
	if path != filepath.Join(workspace, "devbox.yaml") || name != "project-devbox" {
		t.Fatalf("got path %q and name %q", path, name)
	}
}

func TestWorkspaceSpecAllowsMissingFile(t *testing.T) {
	path, name := WorkspaceSpec(t.TempDir())
	if path != "" || name != "" {
		t.Fatalf("got path %q and name %q", path, name)
	}
}

func TestWorkspaceSpecLeavesValidationToDevbox(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "devbox.yaml"), []byte("image: custom:image\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, name := WorkspaceSpec(workspace)
	if path == "" || name != "" {
		t.Fatalf("got path %q and name %q", path, name)
	}
}
