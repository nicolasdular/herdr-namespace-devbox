package main

import (
	"fmt"
	"os"

	"herdr-namespace/internal/herdr"
	"herdr-namespace/internal/namespace"
)

type ActionInputs struct {
	ConfigDir        string
	PluginExecutable string
	Workspace        string
	PaneID           string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: herdr-namespace <start-devbox|new-devbox|connect-session>")
	}
	switch args[0] {
	case "start-devbox":
		inputs, err := loadActionInputs()
		if err != nil {
			return err
		}
		return openDevbox(inputs, "start-devbox", namespace.WorkspaceDevboxName(inputs.Workspace))
	case "new-devbox":
		inputs, err := loadActionInputs()
		if err != nil {
			return err
		}
		return openDevbox(inputs, "new-devbox", namespace.NewDevboxName(inputs.Workspace))
	case "connect-session":
		return connectSession(args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func loadActionInputs() (ActionInputs, error) {
	var inputs ActionInputs

	inputs.ConfigDir = os.Getenv("HERDR_PLUGIN_CONFIG_DIR")
	if inputs.ConfigDir == "" {
		return ActionInputs{}, fmt.Errorf("HERDR_PLUGIN_CONFIG_DIR is required")
	}

	var err error
	inputs.PluginExecutable, err = os.Executable()
	if err != nil {
		return ActionInputs{}, err
	}

	herdrContext, err := herdr.ParseContext(os.Getenv("HERDR_PLUGIN_CONTEXT_JSON"))
	if err != nil {
		return ActionInputs{}, err
	}

	inputs.Workspace = herdrContext.Workspace()
	inputs.PaneID = herdrContext.Pane(os.Getenv("HERDR_PANE_ID"))

	if inputs.Workspace == "" || inputs.PaneID == "" {
		return ActionInputs{}, fmt.Errorf("start a Namespace Devbox from a Herdr workspace or pane")
	}

	inputs.Workspace, err = herdr.WorkspaceRoot(inputs.Workspace)
	if err != nil {
		return ActionInputs{}, err
	}

	return inputs, nil
}
