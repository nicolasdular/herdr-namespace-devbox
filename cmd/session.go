package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"herdr-namespace/internal/config"
	"herdr-namespace/internal/namespace"
)

func prepareSession(ctx context.Context, configDir string) (config.Config, namespace.Client, error) {
	cfg, err := config.Load(configDir)
	if err != nil {
		return config.Config{}, namespace.Client{}, err
	}

	client := namespace.New()
	if err := client.Preflight(ctx); err != nil {
		return config.Config{}, namespace.Client{}, err
	}

	authenticated, err := client.IsAuthenticated(ctx)
	if err != nil {
		return config.Config{}, namespace.Client{}, err
	}

	if !authenticated {
		fmt.Println("Namespace login is required. Complete the login flow below; the Devbox connection will continue afterwards.")

		if err := client.Login(ctx); err != nil {
			return config.Config{}, namespace.Client{}, err
		}

		authenticated, err = client.IsAuthenticated(ctx)
		if err != nil {
			return config.Config{}, namespace.Client{}, err
		}

		if !authenticated {
			return config.Config{}, namespace.Client{}, fmt.Errorf("Namespace authentication could not be verified after login")
		}
	}

	return cfg, client, nil
}

func connectSession(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("connect-session", flag.ContinueOnError)
	name := flags.String("name", "", "Devbox name")
	configDir := flags.String("config-dir", "", "plugin configuration directory")
	repository := flags.String("repository", "", "Git repository to clone when creating the Devbox")
	specPath := flags.String("devbox-spec", "", "workspace devbox.yaml path")

	if err := flags.Parse(args); err != nil {
		return err
	}

	if *name == "" || *configDir == "" || flags.NArg() != 0 {
		return fmt.Errorf("usage: herdr-namespace connect-session --name NAME --config-dir DIR")
	}

	cfg, client, err := prepareSession(ctx, *configDir)
	if err != nil {
		return err
	}

	exists, err := client.Exists(ctx, *name)
	if err != nil {
		return err
	}

	if exists {
		fmt.Printf("Reconnecting to persistent Namespace Devbox %s...\n", *name)
	} else {
		fmt.Printf("Creating persistent Namespace Devbox %s...\n", *name)
		if *specPath != "" {
			if err := client.CreateFromSpec(ctx, *name, *specPath); err != nil {
				return err
			}
		} else {
			if err := client.Create(ctx, *name, cfg, *repository); err != nil {
				return err
			}
		}
	}

	return attachToSession(ctx, client, cfg, *name)
}

func attachToSession(ctx context.Context, client namespace.Client, cfg config.Config, devboxName string) error {
	fmt.Printf("\nConnecting to session %s on %s...\n", cfg.SessionName, devboxName)
	exitCode, err := client.Connect(ctx, devboxName, cfg.SessionName)

	if err != nil {
		return err
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}
