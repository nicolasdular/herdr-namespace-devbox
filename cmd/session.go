package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"herdr-namespace/internal/command"
	"herdr-namespace/internal/namespace"
)

func prepareSession(ctx context.Context) (namespace.Client, error) {
	client := namespace.New()
	if err := client.Preflight(ctx); err != nil {
		return namespace.Client{}, err
	}

	authenticated, err := client.IsAuthenticated(ctx)
	if err != nil {
		return namespace.Client{}, err
	}

	if !authenticated {
		fmt.Println("Namespace login is required. Complete the login flow below; the Devbox connection will continue afterwards.")

		if err := client.Login(ctx); err != nil {
			return namespace.Client{}, err
		}

		authenticated, err = client.IsAuthenticated(ctx)
		if err != nil {
			return namespace.Client{}, err
		}

		if !authenticated {
			return namespace.Client{}, fmt.Errorf("Namespace authentication could not be verified after login")
		}
	}

	return client, nil
}

func connectSession(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("connect-session", flag.ContinueOnError)
	name := flags.String("name", "", "Devbox name")
	repository := flags.String("repository", "", "Git repository to clone when creating the Devbox")
	workspace := flags.String("workspace", "", "workspace directory")
	dotfiles := flags.String("dotfiles", "", "dotfiles repository configured by Herdr")
	existing := flags.Bool("existing", false, "connect without loading a workspace Devbox specification")
	uploadLocalChanges := flags.Bool("upload-local-changes", false, "apply tracked local changes to a newly created Devbox")
	encodedCreatePlan := flags.String("create-plan", "", "encoded creation form overrides")

	if err := flags.Parse(args); err != nil {
		return err
	}

	if *name == "" || *workspace == "" || flags.NArg() != 0 {
		return fmt.Errorf("usage: herdr-namespace connect-session --name NAME --workspace DIR")
	}

	client, err := prepareSession(ctx)
	if err != nil {
		return err
	}
	if *existing {
		return attachToSession(ctx, client, "", *name)
	}
	spec, err := namespace.NewSpec(*workspace, *name, *repository)
	if err != nil {
		return err
	}
	if *encodedCreatePlan != "" {
		encodedPlan, err := base64.RawURLEncoding.DecodeString(*encodedCreatePlan)
		if err != nil {
			return fmt.Errorf("decode creation form values: %w", err)
		}
		var plan DevboxCreatePlan
		if err := json.Unmarshal(encodedPlan, &plan); err != nil {
			return fmt.Errorf("parse creation form values: %w", err)
		}
		spec = plan.apply(spec)
	}
	createOptions := namespace.CreateOptions{Dotfiles: *dotfiles}
	localChanges := newGitLocalChangesService(command.OSRunner{})
	if err := ensureDevbox(
		ctx,
		client,
		localChanges,
		*workspace,
		spec,
		createOptions,
		*uploadLocalChanges,
		os.Stdout,
	); err != nil {
		return err
	}

	return attachToSession(ctx, client, spec.SessionName(), *name)
}

func attachToSession(ctx context.Context, client namespace.Client, sessionName, devboxName string) error {
	if sessionName == "" {
		fmt.Printf("\nConnecting to %s...\n", devboxName)
	} else {
		fmt.Printf("\nConnecting to session %s on %s...\n", sessionName, devboxName)
	}
	exitCode, err := client.Connect(ctx, devboxName, sessionName)

	if err != nil {
		return err
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}
