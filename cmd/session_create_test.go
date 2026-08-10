package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"herdr-namespace/internal/namespace"
)

type creationClientStub struct {
	fakePatchClient
	exists      bool
	existsErr   error
	createErr   error
	createdSpec *namespace.Spec
}

func (c *creationClientStub) Exists(context.Context, string) (bool, error) {
	return c.exists, c.existsErr
}

func (c *creationClientStub) Create(_ context.Context, spec namespace.Spec) error {
	c.createdSpec = &spec
	return c.createErr
}

func TestEnsureDevboxGeneratesPatchBeforeCreating(t *testing.T) {
	client := &creationClientStub{fakePatchClient: fakePatchClient{
		devboxes: []namespace.Devbox{{Name: "demo", DefaultDir: "/workspace/demo"}},
	}}
	changes := &localChangesStub{patch: []byte("binary patch")}
	spec := namespace.Spec{
		Name:       "demo",
		Repository: &namespace.Repository{URL: "github.com/acme/demo", Ref: "main"},
	}
	var output bytes.Buffer

	err := ensureDevbox(context.Background(), client, changes, "/local/demo", spec, true, &output)
	require.NoError(t, err)
	require.Equal(t, []string{"/local/demo"}, changes.generated)
	require.Equal(t, spec, *client.createdSpec)
	require.Equal(t, []byte("binary patch"), client.uploaded)
	require.Contains(t, output.String(), "Creating persistent Namespace Devbox demo")
	require.Contains(t, output.String(), "Uploading 12 B")
}

func TestEnsureDevboxDoesNotCreateWhenPatchGenerationFails(t *testing.T) {
	client := &creationClientStub{}
	changes := &localChangesStub{err: errors.New("wrong repository")}
	spec := namespace.Spec{
		Name:       "demo",
		Repository: &namespace.Repository{URL: "github.com/acme/demo"},
	}

	err := ensureDevbox(context.Background(), client, changes, "/local/demo", spec, true, &bytes.Buffer{})
	require.EqualError(t, err, "wrong repository")
	require.Nil(t, client.createdSpec)
}

func TestEnsureDevboxReconnectSkipsLocalChanges(t *testing.T) {
	client := &creationClientStub{exists: true}
	changes := &localChangesStub{err: errors.New("must not be called")}
	spec := namespace.Spec{Name: "demo"}
	var output bytes.Buffer

	err := ensureDevbox(context.Background(), client, changes, "/local/demo", spec, true, &output)
	require.NoError(t, err)
	require.Empty(t, changes.generated)
	require.Nil(t, client.createdSpec)
	require.Contains(t, output.String(), "Reconnecting to persistent Namespace Devbox demo")
}
