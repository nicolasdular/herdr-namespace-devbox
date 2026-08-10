package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"

	"herdr-namespace/internal/namespace"
)

type patchClient interface {
	List(context.Context) ([]namespace.Devbox, error)
	Upload(context.Context, string, string, string) error
	Exec(context.Context, string, ...string) error
}

func uploadTrackedChanges(ctx context.Context, client patchClient, devboxName string, patch []byte) error {
	devboxes, err := client.List(ctx)
	if err != nil {
		return err
	}
	remoteDirectory := ""
	for _, devbox := range devboxes {
		if devbox.Name == devboxName {
			remoteDirectory = devbox.DefaultDir
			break
		}
	}
	if remoteDirectory == "" {
		return fmt.Errorf("Namespace did not report the repository directory for Devbox %s", devboxName)
	}

	file, err := os.CreateTemp("", "herdr-namespace-*.patch")
	if err != nil {
		return fmt.Errorf("create temporary Git patch: %w", err)
	}
	localPath := file.Name()
	defer os.Remove(localPath)
	if _, err := file.Write(patch); err != nil {
		file.Close()
		return fmt.Errorf("write temporary Git patch: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary Git patch: %w", err)
	}

	digest := sha256.Sum256(patch)
	remotePath := fmt.Sprintf("/tmp/herdr-namespace-%x.patch", digest[:8])
	if err := client.Upload(ctx, devboxName, localPath, remotePath); err != nil {
		return err
	}
	defer func() {
		_ = client.Exec(ctx, devboxName, "rm", "-f", remotePath)
	}()
	if err := client.Exec(ctx, devboxName, "git", "-C", remoteDirectory, "apply", "--check", remotePath); err != nil {
		return fmt.Errorf("local changes do not apply cleanly in Devbox %s: %w", devboxName, err)
	}
	if err := client.Exec(ctx, devboxName, "git", "-C", remoteDirectory, "apply", remotePath); err != nil {
		return fmt.Errorf("apply local changes in Devbox %s: %w", devboxName, err)
	}
	return nil
}
