package main

import (
	"flag"
	"fmt"
	"os"

	"herdr-namespace/internal/config"
	"herdr-namespace/internal/namespace"
)

func prepareSession(configDir string) (config.Config, namespace.Client, error) {
	cfg, err := config.Load(configDir)
	if err != nil {
		return config.Config{}, namespace.Client{}, err
	}

	client := namespace.New()
	if err := client.Preflight(); err != nil {
		return config.Config{}, namespace.Client{}, err
	}

	authenticated, err := client.IsAuthenticated()
	if err != nil {
		return config.Config{}, namespace.Client{}, err
	}

	if !authenticated {
		fmt.Println("Namespace login is required. Complete the login flow below; the Devbox connection will continue afterwards.")

		if err := client.Login(); err != nil {
			return config.Config{}, namespace.Client{}, err
		}

		authenticated, err = client.IsAuthenticated()
		if err != nil {
			return config.Config{}, namespace.Client{}, err
		}

		if !authenticated {
			return config.Config{}, namespace.Client{}, fmt.Errorf("Namespace authentication could not be verified after login")
		}
	}

	return cfg, client, nil
}

func connectSession(args []string) error {
	flags := flag.NewFlagSet("connect-session", flag.ContinueOnError)
	name := flags.String("name", "", "Devbox name")
	configDir := flags.String("config-dir", "", "plugin configuration directory")

	if err := flags.Parse(args); err != nil {
		return err
	}

	if *name == "" || *configDir == "" || flags.NArg() != 0 {
		return fmt.Errorf("usage: herdr-namespace connect-session --name NAME --config-dir DIR")
	}

	cfg, client, err := prepareSession(*configDir)
	if err != nil {
		return err
	}

	exists, err := client.Exists(*name)
	if err != nil {
		return err
	}

	if exists {
		fmt.Printf("Reconnecting to persistent Namespace Devbox %s...\n", *name)
	} else {
		fmt.Printf("Creating persistent Namespace Devbox %s...\n", *name)
		if err := client.Create(*name, cfg); err != nil {
			return err
		}
	}

	return attachToSession(client, cfg, *name)
}

func attachToSession(client namespace.Client, cfg config.Config, devboxName string) error {
	fmt.Printf("\nConnecting to session %s on %s...\n", cfg.SessionName, devboxName)
	exitCode, err := client.Connect(devboxName, cfg.SessionName)

	if err != nil {
		return err
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}
