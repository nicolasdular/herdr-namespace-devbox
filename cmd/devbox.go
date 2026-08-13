package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"herdr-namespace/internal/command"
	"herdr-namespace/internal/herdr"
	"herdr-namespace/internal/namespace"
)

func workspaceDevbox(workspace string) (string, error) {
	return namespace.WorkspaceSpecName(workspace)
}

func openDevbox(
	ctx context.Context,
	inputs ActionInputs,
	actionID string,
	devboxName string,
	uploadLocalChanges bool,
) error {
	runner := command.OSRunner{}
	herdrClient := herdr.New(os.Getenv("HERDR_BIN_PATH"))
	createOptions, err := namespace.LoadCreateOptions(inputs.PluginConfigDir)
	if err != nil {
		return err
	}
	args := []string{
		"connect-session",
		"--name", devboxName,
		"--workspace", inputs.Workspace,
	}
	if createOptions.Dotfiles != "" {
		args = append(args, "--dotfiles", createOptions.Dotfiles)
	}
	if repository := repositoryURL(ctx, runner, inputs.Workspace); repository != "" {
		args = append(args, "--repository", repository)
	}
	if uploadLocalChanges {
		args = append(args, "--upload-local-changes")
	}
	tab, err := launchDevboxTab(ctx, herdrClient, inputs, tabTitle(inputs.Workspace), devboxName, args...)
	if err != nil {
		return err
	}
	printResult(actionID, "session-launched", tab.RootPaneID, devboxName)
	return nil
}

func openOrFocusDevbox(ctx context.Context, inputs *ActionInputs, devboxName string) error {
	herdrClient := herdr.New(os.Getenv("HERDR_BIN_PATH"))
	pane, err := herdrClient.FindDevboxPane(ctx, devboxName)
	if err != nil {
		return err
	}
	if pane != nil {
		return herdrClient.FocusTab(ctx, pane.WorkspaceID, pane.TabID)
	}
	if inputs == nil {
		return fmt.Errorf("open a Devbox from a Herdr workspace")
	}
	_, err = launchDevboxTab(
		ctx, herdrClient, *inputs, "Devbox · "+devboxName, devboxName,
		"connect-session", "--name", devboxName, "--workspace", inputs.Workspace, "--existing",
	)
	return err
}

func launchDevboxTab(
	ctx context.Context,
	herdrClient herdr.Herdr,
	inputs ActionInputs,
	label string,
	devboxName string,
	args ...string,
) (herdr.Tab, error) {
	tab, err := herdrClient.CreateTab(ctx, inputs.Workspace, label)
	if err != nil {
		return herdr.Tab{}, err
	}
	if err := herdrClient.RunInPane(ctx, tab.RootPaneID, inputs.PluginExecutable, args...); err != nil {
		return herdr.Tab{}, err
	}
	if err := herdrClient.MarkDevboxPane(ctx, tab.RootPaneID, devboxName); err != nil {
		return herdr.Tab{}, err
	}
	return tab, nil
}

func repositoryURL(ctx context.Context, runner command.Runner, workspace string) string {
	git := command.NewWithRunner("git", runner)
	output, err := git.Output(ctx, 5*time.Second, "-C", workspace, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return normalizeRepositoryURL(strings.TrimSpace(string(output)))
}

func normalizeRepositoryURL(repository string) string {
	const githubSSH = "git@github.com:"
	if strings.HasPrefix(repository, githubSSH) {
		repository = "github.com/" + strings.TrimPrefix(repository, githubSSH)
	}
	if strings.HasPrefix(repository, "https://github.com/") {
		repository = strings.TrimPrefix(repository, "https://")
	}
	return strings.TrimSuffix(repository, ".git")
}

func tabTitle(workspace string) string {
	return "Devbox · " + filepath.Base(workspace)
}

func printResult(actionID, phase, paneID, devboxName string) {
	result, _ := json.Marshal(map[string]any{
		"schemaVersion": 1, "action": actionID, "ok": true,
		"phase": phase, "paneId": paneID, "devboxName": devboxName,
	})
	fmt.Printf("HERDR_NAMESPACE_RESULT: %s\n", result)
}
