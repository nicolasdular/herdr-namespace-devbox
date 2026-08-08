package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
	if err := herdrClient.RunInPane(
		inputs.PaneID, inputs.PluginExecutable, "connect-session",
		"--name", devboxName,
		"--config-dir", inputs.ConfigDir,
	); err != nil {
		return err
	}
	printResult(actionID, "session-launched", inputs.PaneID, devboxName)
	return nil
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
