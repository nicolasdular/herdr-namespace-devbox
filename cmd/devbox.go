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
) error {
	runner := command.OSRunner{}
	herdrClient := herdr.New(os.Getenv("HERDR_BIN_PATH"))
	tab, err := herdrClient.CreateTab(ctx, inputs.Workspace, tabTitle(inputs.Workspace))
	if err != nil {
		return err
	}
	args := []string{
		"connect-session",
		"--name", devboxName,
		"--workspace", inputs.Workspace,
	}
	if repository := repositoryURL(ctx, runner, inputs.Workspace); repository != "" {
		args = append(args, "--repository", repository)
	}
	if err := herdrClient.RunInPane(
		ctx, tab.RootPaneID, inputs.PluginExecutable, args...,
	); err != nil {
		return err
	}
	printResult(actionID, "session-launched", tab.RootPaneID, devboxName)
	return nil
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
