package herdr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewUsesProvidedExecutable(t *testing.T) {
	client := New("/custom/bin/herdr")
	if client.executable != "/custom/bin/herdr" {
		t.Fatalf("got %q", client.executable)
	}
}

func TestNewFallsBackToHerdrOnPath(t *testing.T) {
	client := New("")
	if client.executable != "herdr" {
		t.Fatalf("got %q", client.executable)
	}
}

func TestContextPrefersFocusedPaneCWD(t *testing.T) {
	context, err := ParseContext(`{"workspace_cwd":"/workspace","focused_pane_cwd":"/workspace/subdir"}`)
	if err != nil {
		t.Fatal(err)
	}
	if got := context.Workspace(); got != "/workspace/subdir" {
		t.Fatalf("got %q", got)
	}
}

func TestWorkspaceRootFindsGitRootFromSubdirectory(t *testing.T) {
	repository := t.TempDir()
	if err := os.Mkdir(filepath.Join(repository, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	subdirectory := filepath.Join(repository, "some", "nested", "directory")
	if err := os.MkdirAll(subdirectory, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := WorkspaceRoot(subdirectory)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(repository)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWorkspaceRootFallsBackToDirectoryOutsideGit(t *testing.T) {
	workspace := t.TempDir()
	got, err := WorkspaceRoot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
