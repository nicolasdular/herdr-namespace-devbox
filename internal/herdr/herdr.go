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

type Tab struct {
	ID         string
	RootPaneID string
}

type Pane struct {
	ID          string            `json:"pane_id"`
	TabID       string            `json:"tab_id"`
	WorkspaceID string            `json:"workspace_id"`
	Tokens      map[string]string `json:"tokens"`
}

func (h Herdr) CreateTab(ctx context.Context, cwd, label string) (Tab, error) {
	output, err := h.cmd.Output(
		ctx,
		commandTimeout,
		"tab", "create",
		"--cwd", cwd,
		"--label", label,
		"--focus",
	)
	if err != nil {
		return Tab{}, err
	}

	var response struct {
		Result struct {
			RootPane struct {
				PaneID string `json:"pane_id"`
			} `json:"root_pane"`
			Tab struct {
				TabID string `json:"tab_id"`
			} `json:"tab"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return Tab{}, fmt.Errorf("parse Herdr tab creation response: %w", err)
	}
	if response.Result.Tab.TabID == "" || response.Result.RootPane.PaneID == "" {
		return Tab{}, errors.New("Herdr tab creation response is missing tab or pane ID")
	}
	return Tab{
		ID:         response.Result.Tab.TabID,
		RootPaneID: response.Result.RootPane.PaneID,
	}, nil
}

func (h Herdr) RunInPane(ctx context.Context, paneID, executable string, args ...string) error {
	command := []string{"pane", "run", paneID, executable}
	return h.run(ctx, append(command, args...)...)
}

func (h Herdr) MarkDevboxPane(ctx context.Context, paneID, devboxName string) error {
	return h.run(
		ctx,
		"pane", "report-metadata", paneID,
		"--source", "namespace-devbox",
		"--token", "devbox="+devboxName,
	)
}

func (h Herdr) FindDevboxPane(ctx context.Context, devboxName string) (*Pane, error) {
	output, err := h.cmd.Output(ctx, commandTimeout, "pane", "list")
	if err != nil {
		return nil, err
	}
	var response struct {
		Result struct {
			Panes []Pane `json:"panes"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("parse Herdr pane list response: %w", err)
	}
	for _, pane := range response.Result.Panes {
		if pane.Tokens["devbox"] == devboxName {
			return &pane, nil
		}
	}

	// Panes opened by plugin versions before tokens were introduced can be
	// migrated while their Devbox connector is still running.
	for _, pane := range response.Result.Panes {
		if pane.Tokens["devbox"] != "" {
			continue
		}
		name, processErr := h.paneDevboxName(ctx, pane.ID)
		if processErr != nil || name != devboxName {
			continue
		}
		if err := h.MarkDevboxPane(ctx, pane.ID, name); err != nil {
			return nil, err
		}
		return &pane, nil
	}
	return nil, nil
}

func (h Herdr) paneDevboxName(ctx context.Context, paneID string) (string, error) {
	output, err := h.cmd.Output(ctx, commandTimeout, "pane", "process-info", "--pane", paneID)
	if err != nil {
		return "", err
	}
	var response struct {
		Result struct {
			ProcessInfo struct {
				ForegroundProcesses []struct {
					Argv []string `json:"argv"`
				} `json:"foreground_processes"`
			} `json:"process_info"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return "", fmt.Errorf("parse Herdr pane process response: %w", err)
	}
	for _, process := range response.Result.ProcessInfo.ForegroundProcesses {
		isConnector := false
		for _, argument := range process.Argv {
			if argument == "connect-session" {
				isConnector = true
				break
			}
		}
		if !isConnector {
			continue
		}
		for index, argument := range process.Argv {
			if argument == "--name" && index+1 < len(process.Argv) {
				return process.Argv[index+1], nil
			}
		}
	}
	return "", nil
}

func (h Herdr) FocusTab(ctx context.Context, workspaceID, tabID string) error {
	if err := h.run(ctx, "workspace", "focus", workspaceID); err != nil {
		return err
	}
	return h.run(ctx, "tab", "focus", tabID)
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
