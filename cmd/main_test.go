package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadActionInputsUsesWorkspaceRoot(t *testing.T) {
	configDirectory := t.TempDir()
	repository := t.TempDir()
	if err := os.Mkdir(filepath.Join(repository, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	subdirectory := filepath.Join(repository, "nested")
	if err := os.Mkdir(subdirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	contextJSON, err := json.Marshal(map[string]string{
		"focused_pane_cwd": subdirectory,
		"focused_pane_id":  "pane-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", configDirectory)
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", string(contextJSON))

	inputs, err := loadActionInputs()
	if err != nil {
		t.Fatal(err)
	}
	workspaceRoot, err := filepath.EvalSymlinks(repository)
	if err != nil {
		t.Fatal(err)
	}
	if inputs.Workspace != workspaceRoot {
		t.Fatalf("unexpected action inputs: %#v", inputs)
	}
}

func TestNormalizeRepositoryURL(t *testing.T) {
	tests := map[string]string{
		"git@github.com:acme/demo.git":     "github.com/acme/demo",
		"https://github.com/acme/demo.git": "github.com/acme/demo",
		"https://gitlab.com/acme/demo.git": "https://gitlab.com/acme/demo",
	}
	for input, want := range tests {
		if got := normalizeRepositoryURL(input); got != want {
			t.Errorf("normalizeRepositoryURL(%q) = %q, want %q", input, got, want)
		}
	}
}
