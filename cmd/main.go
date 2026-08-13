package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"herdr-namespace/internal/herdr"
	"herdr-namespace/internal/namespace"
)

type ActionInputs struct {
	PluginExecutable string
	PluginConfigDir  string
	Workspace        string
	PaneID           string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: herdr-namespace <start-devbox|new-devbox|stop-devbox|manage-devboxes|connect-session>")
	}
	switch args[0] {
	case "start-devbox":
		inputs, err := loadActionInputs()
		if err != nil {
			return err
		}
		devboxName, err := workspaceDevbox(inputs.Workspace)
		if err != nil {
			return err
		}
		return openDevbox(ctx, inputs, "start-devbox", devboxName, false)
	case "new-devbox":
		inputs, err := loadActionInputs()
		if err != nil {
			return err
		}
		devboxName := namespace.NewDevboxName(inputs.Workspace)
		return openDevbox(ctx, inputs, "new-devbox", devboxName, false)
	case "stop-devbox":
		inputs, err := loadActionInputs()
		if err != nil {
			return err
		}
		devboxName, err := workspaceDevbox(inputs.Workspace)
		if err != nil {
			return err
		}
		return stopDevbox(ctx, inputs, devboxName)
	case "manage-devboxes":
		return manageDevboxes(ctx)
	case "connect-session":
		return connectSession(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func loadActionInputs() (ActionInputs, error) {
	var inputs ActionInputs

	var err error
	inputs.PluginExecutable, err = os.Executable()
	if err != nil {
		return ActionInputs{}, err
	}
	inputs.PluginConfigDir = os.Getenv("HERDR_PLUGIN_CONFIG_DIR")

	herdrContext, err := herdr.ParseContext(os.Getenv("HERDR_PLUGIN_CONTEXT_JSON"))
	if err != nil {
		return ActionInputs{}, err
	}

	inputs.Workspace = herdrContext.Workspace()
	inputs.PaneID = herdrContext.Pane(os.Getenv("HERDR_PANE_ID"))

	if inputs.Workspace == "" {
		return ActionInputs{}, fmt.Errorf("run this Namespace action from a Herdr workspace or pane")
	}

	inputs.Workspace, err = herdr.WorkspaceRoot(inputs.Workspace)
	if err != nil {
		return ActionInputs{}, err
	}

	return inputs, nil
}
