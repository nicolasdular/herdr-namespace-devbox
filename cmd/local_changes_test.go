package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"herdr-namespace/internal/namespace"
)

type patchRunner struct {
	calls   [][]string
	outputs [][]byte
	err     error
}

func (r *patchRunner) CombinedOutput(_ context.Context, _ string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	if len(r.outputs) == 0 {
		return nil, r.err
	}
	output := r.outputs[0]
	r.outputs = r.outputs[1:]
	return output, r.err
}

func (r *patchRunner) Run(
	context.Context,
	string,
	[]string,
	io.Reader,
	io.Writer,
	io.Writer,
) error {
	return r.err
}

func TestTrackedChangeCountUsesNULTerminatedGitNames(t *testing.T) {
	runner := &patchRunner{outputs: [][]byte{
		[]byte("git@github.com:acme/demo.git\n"),
		[]byte("head123\n"),
		[]byte("first.go\x00name\nwith-newline.go\x00"),
	}}
	service := newGitLocalChangesService(runner)

	info, err := service.Inspect(context.Background(), "/workspace", namespace.Repository{
		URL: "github.com/acme/demo",
	})
	require.NoError(t, err)
	require.Equal(t, LocalChangesInfo{BaseCommit: "head123", FileCount: 2}, info)
	require.Equal(t, []string{
		"-C", "/workspace", "diff", "--name-only", "-z", "--no-ext-diff", "head123", "--",
	}, runner.calls[2])
}

func TestTrackedChangesPatchIncludesBinaryTrackedChanges(t *testing.T) {
	runner := &patchRunner{outputs: [][]byte{
		[]byte("git@github.com:acme/demo.git\n"),
		[]byte("head123\n"),
		[]byte("binary patch"),
	}}
	service := newGitLocalChangesService(runner)

	patch, err := service.GeneratePatch(context.Background(), "/workspace", namespace.Repository{
		URL: "github.com/acme/demo",
	})
	require.NoError(t, err)
	require.Equal(t, []byte("binary patch"), patch)
	require.Equal(t, []string{
		"-C", "/workspace", "diff", "--binary", "--full-index", "--no-ext-diff", "head123", "--",
	}, runner.calls[2])
}

func TestTrackedChangesPatchUsesConfiguredRepositoryRef(t *testing.T) {
	runner := &patchRunner{outputs: [][]byte{
		[]byte("https://github.com/acme/demo.git\n"),
		[]byte("abc123\n"),
		[]byte("patch from release"),
	}}
	service := newGitLocalChangesService(runner)

	patch, err := service.GeneratePatch(context.Background(), "/workspace", namespace.Repository{
		URL: "github.com/acme/demo",
		Ref: "release",
	})
	require.NoError(t, err)
	require.Equal(t, []byte("patch from release"), patch)
	require.Equal(t, []string{
		"-C", "/workspace", "diff", "--binary", "--full-index", "--no-ext-diff", "abc123", "--",
	}, runner.calls[2])
}

func TestTrackedChangesRejectsDifferentConfiguredRepository(t *testing.T) {
	runner := &patchRunner{outputs: [][]byte{[]byte("git@github.com:acme/local.git\n")}}
	service := newGitLocalChangesService(runner)

	_, err := service.GeneratePatch(context.Background(), "/workspace", namespace.Repository{
		URL: "github.com/acme/other",
	})
	require.ErrorContains(t, err, "not workspace origin")
	require.Len(t, runner.calls, 1)
}

func TestResolveCreatePlanShowsEffectiveSpecWithoutInspectingChanges(t *testing.T) {
	runner := &patchRunner{outputs: [][]byte{
		[]byte("git@github.com:acme/demo.git\n"),
	}}

	plan, err := resolveCreatePlan(context.Background(), ActionInputs{Workspace: t.TempDir()}, runner)
	require.NoError(t, err)
	require.Equal(t, "github.com/acme/demo", plan.Repository.URL)
	require.Equal(t, "builtin:agents", plan.Image)
	require.Equal(t, "m", plan.Size)
	require.Equal(t, "automatic", plan.Site)
	require.Len(t, runner.calls, 1)
}

type fakePatchClient struct {
	devboxes []namespace.Devbox
	uploaded []byte
	upload   []string
	exec     [][]string
	err      error
}

func (c *fakePatchClient) List(context.Context) ([]namespace.Devbox, error) {
	return c.devboxes, c.err
}

func (c *fakePatchClient) Upload(_ context.Context, name, localPath, remotePath string) error {
	contents, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	c.uploaded = contents
	c.upload = []string{name, localPath, remotePath}
	return c.err
}

func (c *fakePatchClient) Exec(_ context.Context, name string, args ...string) error {
	c.exec = append(c.exec, append([]string{name}, args...))
	return c.err
}

func TestUploadTrackedChangesChecksAndAppliesPatch(t *testing.T) {
	client := &fakePatchClient{devboxes: []namespace.Devbox{{
		Name:       "box-one",
		DefaultDir: "/workspaces/demo",
	}}}
	patch := []byte("diff --git a/demo b/demo\n")

	require.NoError(t, uploadTrackedChanges(context.Background(), client, "box-one", patch))
	require.True(t, bytes.Equal(patch, client.uploaded))
	require.Len(t, client.exec, 3)
	remotePath := client.upload[2]
	require.Equal(t, []string{
		"box-one", "git", "-C", "/workspaces/demo", "apply", "--check", remotePath,
	}, client.exec[0])
	require.Equal(t, []string{
		"box-one", "git", "-C", "/workspaces/demo", "apply", remotePath,
	}, client.exec[1])
	require.Equal(t, []string{"box-one", "rm", "-f", remotePath}, client.exec[2])
}

func TestUploadTrackedChangesRequiresRemoteRepositoryDirectory(t *testing.T) {
	client := &fakePatchClient{devboxes: []namespace.Devbox{{Name: "box-one"}}}

	err := uploadTrackedChanges(context.Background(), client, "box-one", []byte("patch"))
	require.EqualError(t, err, "Namespace did not report the repository directory for Devbox box-one")
	require.Empty(t, client.upload)
}

func TestTrackedChangesPatchReportsGitFailure(t *testing.T) {
	runner := &patchRunner{err: errors.New("git failed")}
	service := newGitLocalChangesService(runner)

	_, err := service.GeneratePatch(context.Background(), "/workspace", namespace.Repository{
		URL: "github.com/acme/demo",
	})
	require.ErrorContains(t, err, "verify local changes repository")
}
