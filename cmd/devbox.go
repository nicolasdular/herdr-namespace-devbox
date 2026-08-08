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

func workspaceDevbox(workspace string) (name, specPath string) {
	specPath, name = namespace.WorkspaceSpec(workspace)
	if name == "" {
		name = namespace.WorkspaceDevboxName(workspace)
	}
	return name, specPath
}

func openDevbox(
	ctx context.Context,
	inputs ActionInputs,
	actionID string,
	devboxName string,
	specPath string,
) error {
	runner := command.OSRunner{}
	herdrClient := herdr.New(os.Getenv("HERDR_BIN_PATH"))
	if err := herdrClient.RenamePane(ctx, inputs.PaneID, paneTitle(inputs.Workspace)); err != nil {
		return err
	}
	args := []string{
		"connect-session",
		"--name", devboxName,
		"--config-dir", inputs.ConfigDir,
	}
	if specPath != "" {
		args = append(args, "--devbox-spec", specPath)
	} else if repository := repositoryURL(ctx, runner, inputs.Workspace); repository != "" {
		args = append(args, "--repository", repository)
	}
	if err := herdrClient.RunInPane(
		ctx, inputs.PaneID, inputs.PluginExecutable, args...,
	); err != nil {
		return err
	}
	printResult(actionID, "session-launched", inputs.PaneID, devboxName)
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

func paneTitle(workspace string) string {
	return "Namespace · " + filepath.Base(workspace)
}

func printResult(actionID, phase, paneID, devboxName string) {
	result, _ := json.Marshal(map[string]any{
		"schemaVersion": 1, "action": actionID, "ok": true,
		"phase": phase, "paneId": paneID, "devboxName": devboxName,
	})
	fmt.Printf("HERDR_NAMESPACE_RESULT: %s\n", result)
}
