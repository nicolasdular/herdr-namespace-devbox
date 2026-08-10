package main

import (
	"context"
	"fmt"
	"io"

	"herdr-namespace/internal/namespace"
)

type devboxCreationClient interface {
	patchClient
	Exists(context.Context, string) (bool, error)
	Create(context.Context, namespace.Spec) error
}

func ensureDevbox(
	ctx context.Context,
	client devboxCreationClient,
	localChanges LocalChangesService,
	workspace string,
	spec namespace.Spec,
	uploadLocalChanges bool,
	output io.Writer,
) error {
	exists, err := client.Exists(ctx, spec.Name)
	if err != nil {
		return err
	}
	if exists {
		fmt.Fprintf(output, "Reconnecting to persistent Namespace Devbox %s...\n", spec.Name)
		return nil
	}

	var patch []byte
	if uploadLocalChanges {
		if spec.Repository == nil || spec.Repository.Disabled || spec.Repository.URL == "" {
			return fmt.Errorf("upload local changes requires a repository in the Devbox specification")
		}
		patch, err = localChanges.GeneratePatch(ctx, workspace, *spec.Repository)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(output, "Creating persistent Namespace Devbox %s...\n", spec.Name)
	if err := client.Create(ctx, spec); err != nil {
		return err
	}
	if len(patch) == 0 {
		return nil
	}

	fmt.Fprintf(output, "Uploading %s of tracked local changes...\n", formatByteSize(len(patch)))
	return uploadTrackedChanges(ctx, client, spec.Name, patch)
}
