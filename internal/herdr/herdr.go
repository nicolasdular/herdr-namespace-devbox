package herdr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"herdr-namespace/internal/command"
)

const commandTimeout = 15 * time.Second

type Herdr struct {
	cmd command.Command
}

func New(executable string) Herdr {
	return newWithRunner(executable, command.OSRunner{})
}

func newWithRunner(executable string, runner command.Runner) Herdr {
	if executable == "" {
		executable = "herdr"
	}
	return Herdr{cmd: command.NewWithRunner(executable, runner)}
}

func (h Herdr) RenamePane(ctx context.Context, paneID, title string) error {
	return h.run(ctx, "pane", "rename", paneID, title)
}

func (h Herdr) RunInPane(ctx context.Context, paneID, executable string, args ...string) error {
	command := []string{"pane", "run", paneID, executable}
	return h.run(ctx, append(command, args...)...)
}

func (h Herdr) run(ctx context.Context, args ...string) error {
	_, err := h.cmd.Output(ctx, commandTimeout, args...)
	return err
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
