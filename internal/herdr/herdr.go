package herdr

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Herdr struct {
	executable string
}

func New(executable string) Herdr {
	if executable == "" {
		executable = "herdr"
	}
	return Herdr{executable: executable}
}

func (h Herdr) RenamePane(paneID, title string) error {
	return h.run("pane", "rename", paneID, title)
}

func (h Herdr) RunInPane(paneID, executable string, args ...string) error {
	command := []string{"pane", "run", paneID, executable}
	return h.run(append(command, args...)...)
}

func (h Herdr) run(args ...string) error {
	command := exec.Command(h.executable, args...)
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s: %w", h.executable, err)
	}
	return nil
}

type Context struct {
	FocusedPaneCWD string `json:"focused_pane_cwd"`
	WorkspaceCWD   string `json:"workspace_cwd"`
	FocusedPaneID  string `json:"focused_pane_id"`
}

func ParseContext(raw string) (Context, error) {
	if raw == "" {
		return Context{}, nil
	}
	var context Context
	if err := json.Unmarshal([]byte(raw), &context); err != nil {
		return Context{}, fmt.Errorf("HERDR_PLUGIN_CONTEXT_JSON is invalid JSON: %w", err)
	}
	return context, nil
}

func (c Context) Workspace() string {
	if c.FocusedPaneCWD != "" {
		return c.FocusedPaneCWD
	}
	return c.WorkspaceCWD
}

func (c Context) Pane(fallback string) string {
	if c.FocusedPaneID != "" {
		return c.FocusedPaneID
	}
	return fallback
}

func WorkspaceRoot(workspace string) (string, error) {
	absolute, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("resolve workspace path: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
		absolute = resolved
	}

	current := absolute
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect workspace: %w", err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return absolute, nil
		}
		current = parent
	}
}
