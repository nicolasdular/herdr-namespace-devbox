package main

import (
	"context"
	"fmt"

	"herdr-namespace/internal/namespace"
)

func stopDevbox(ctx context.Context, inputs ActionInputs, devboxName string) error {
	client := namespace.New()
	if err := client.Preflight(ctx); err != nil {
		return err
	}

	authenticated, err := client.IsAuthenticated(ctx)
	if err != nil {
		return err
	}
	if !authenticated {
		return fmt.Errorf("Namespace login is required; open a Devbox first to complete the login flow")
	}

	if err := client.Stop(ctx, devboxName); err != nil {
		return err
	}

	printResult("stop-devbox", "devbox-stopped", inputs.PaneID, devboxName)
	return nil
}
