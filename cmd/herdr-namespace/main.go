package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"herdr-namespace/internal/config"
	"herdr-namespace/internal/herdr"
	"herdr-namespace/internal/namespace"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: herdr-namespace <start-devbox|session>")
	}
	switch args[0] {
	case "start-devbox":
		return startDevbox()
	case "session":
		return runSession(args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func startDevbox() error {
	env, err := prepareEnv()
	if err != nil {
		return err
	}
	herdrClient := herdr.New(herdr.Environment("HERDR_BIN_PATH", "herdr"))

	if err := herdrClient.Run("pane", "rename", env.PaneID, "Namespace · "+filepath.Base(env.Workspace)); err != nil {
		return err
	}
	if err := herdrClient.Run(
		"pane", "run", env.PaneID, env.Executable, "session",
		"--name", env.DevboxName, "--config-dir", env.ConfigDir,
	); err != nil {
		return err
	}
	result, _ := json.Marshal(map[string]any{
		"schemaVersion": 1, "action": env.ActionID, "ok": true,
		"phase": "setup-launched", "paneId": env.PaneID, "devboxName": env.DevboxName,
	})
	fmt.Printf("HERDR_NAMESPACE_RESULT: %s\n", result)
	return nil
}

type Env struct {
	ActionID   string
	ConfigDir  string
	Executable string
	Workspace  string
	PaneID     string
	DevboxName string
}

func prepareEnv() (Env, error) {
	var env Env
	env.ActionID = os.Getenv("HERDR_PLUGIN_ACTION_ID")
	if env.ActionID == "" {
		return Env{}, fmt.Errorf("HERDR_PLUGIN_ACTION_ID is required")
	}
	if env.ActionID != "start-devbox" {
		return Env{}, fmt.Errorf("unknown plugin action: %s", env.ActionID)
	}

	env.ConfigDir = os.Getenv("HERDR_PLUGIN_CONFIG_DIR")
	if env.ConfigDir == "" {
		return Env{}, fmt.Errorf("HERDR_PLUGIN_CONFIG_DIR is required")
	}
	var err error
	env.Executable, err = os.Executable()
	if err != nil {
		return Env{}, err
	}

	context, err := herdr.ParseContext(os.Getenv("HERDR_PLUGIN_CONTEXT_JSON"))
	if err != nil {
		return Env{}, err
	}

	env.Workspace = context.Workspace()
	env.PaneID = context.Pane(os.Getenv("HERDR_PANE_ID"))
	if env.Workspace == "" || env.PaneID == "" {
		return Env{}, fmt.Errorf("start a Namespace Devbox from a Herdr workspace or pane")
	}

	env.DevboxName = herdr.GenerateDevboxName(env.Workspace)
	return env, nil
}

func runSession(args []string) error {
	flags := flag.NewFlagSet("session", flag.ContinueOnError)
	name := flags.String("name", "", "Devbox name")
	configDir := flags.String("config-dir", "", "plugin configuration directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *name == "" || *configDir == "" || flags.NArg() != 0 {
		return fmt.Errorf("usage: herdr-namespace session --name NAME --config-dir DIR")
	}

	cfg, err := config.Load(*configDir)
	if err != nil {
		return err
	}
	client := namespace.New(herdr.Environment("NAMESPACE_DEVBOX_BIN", "devbox"))
	if err := client.Preflight(); err != nil {
		return err
	}
	authenticated, err := client.IsAuthenticated()
	if err != nil {
		return err
	}
	if !authenticated {
		fmt.Println("Namespace login is required. Complete the login flow below; Devbox creation will continue afterwards.")
		if err := client.Login(); err != nil {
			return err
		}
		authenticated, err = client.IsAuthenticated()
		if err != nil {
			return err
		}
		if !authenticated {
			return fmt.Errorf("Namespace authentication could not be verified after login")
		}
	}
	fmt.Printf("Creating persistent Namespace Devbox %s...\n", *name)
	if err := client.Create(*name, cfg); err != nil {
		return err
	}
	fmt.Printf("\nConnecting to session %s on %s...\n", cfg.SessionName, *name)
	exitCode, err := client.Connect(*name, cfg.SessionName)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}
