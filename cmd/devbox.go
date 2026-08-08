package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"herdr-namespace/internal/herdr"
)

func openDevbox(
	inputs ActionInputs,
	actionID string,
	devboxName string,
) error {
	herdrClient := herdr.New(os.Getenv("HERDR_BIN_PATH"))
	if err := herdrClient.RenamePane(inputs.PaneID, paneTitle(inputs.Workspace)); err != nil {
		return err
	}
	args := []string{
		"connect-session",
		"--name", devboxName,
		"--config-dir", inputs.ConfigDir,
	}
	if repository := repositoryURL(inputs.Workspace); repository != "" {
		args = append(args, "--repository", repository)
	}
	if err := herdrClient.RunInPane(
		inputs.PaneID, inputs.PluginExecutable, args...,
	); err != nil {
		return err
	}
	printResult(actionID, "session-launched", inputs.PaneID, devboxName)
	return nil
}

func repositoryURL(workspace string) string {
	output, err := exec.Command("git", "-C", workspace, "remote", "get-url", "origin").Output()
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
