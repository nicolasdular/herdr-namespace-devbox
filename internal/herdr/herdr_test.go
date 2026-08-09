package herdr

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type recordingRunner struct {
	executable string
	args       []string
	output     []byte
	err        error
}

func (r *recordingRunner) CombinedOutput(_ context.Context, executable string, args ...string) ([]byte, error) {
	r.executable = executable
	r.args = append([]string(nil), args...)
	return r.output, r.err
}

func (*recordingRunner) Run(context.Context, string, []string, io.Reader, io.Writer, io.Writer) error {
	panic("unexpected interactive command")
}

func TestNewUsesProvidedExecutable(t *testing.T) {
	runner := &recordingRunner{output: tabCreatedOutput("tab-1", "pane-1")}
	client := newWithRunner("/custom/bin/herdr", runner)
	_, err := client.CreateTab(context.Background(), "/workspace", "title")
	require.NoError(t, err)
	require.Equal(t, "/custom/bin/herdr", runner.executable)
}

func TestNewFallsBackToHerdrOnPath(t *testing.T) {
	runner := &recordingRunner{output: tabCreatedOutput("tab-1", "pane-1")}
	client := newWithRunner("", runner)
	_, err := client.CreateTab(context.Background(), "/workspace", "title")
	require.NoError(t, err)
	require.Equal(t, "herdr", runner.executable)
}

func TestCreateTabBuildsCommandAndReturnsIDs(t *testing.T) {
	runner := &recordingRunner{output: tabCreatedOutput("tab-1", "pane-1")}
	client := newWithRunner("/custom/bin/herdr", runner)

	tab, err := client.CreateTab(context.Background(), "/workspace/demo", "Devbox · demo")
	require.NoError(t, err)
	require.Equal(t, Tab{ID: "tab-1", RootPaneID: "pane-1"}, tab)
	require.Equal(t, "/custom/bin/herdr", runner.executable)
	require.Equal(t, []string{
		"tab", "create",
		"--cwd", "/workspace/demo",
		"--label", "Devbox · demo",
		"--focus",
	}, runner.args)
}

func TestCreateTabRejectsInvalidResponse(t *testing.T) {
	runner := &recordingRunner{output: []byte(`{"result":{"type":"tab_created"}}`)}
	client := newWithRunner("herdr", runner)

	_, err := client.CreateTab(context.Background(), "/workspace", "Devbox · demo")
	require.ErrorContains(t, err, "missing tab or pane ID")
}

func TestRunInPaneBuildsCommand(t *testing.T) {
	runner := &recordingRunner{}
	client := newWithRunner("/custom/bin/herdr", runner)

	require.NoError(t, client.RunInPane(
		context.Background(),
		"pane-1",
		"/plugins/namespace",
		"connect-session",
	))
	require.Equal(t, []string{
		"pane", "run", "pane-1", "/plugins/namespace", "connect-session",
	}, runner.args)
}

func TestCommandFailureIncludesCapturedOutput(t *testing.T) {
	wantErr := errors.New("exit status 1")
	runner := &recordingRunner{output: []byte("pane not found\n"), err: wantErr}
	client := newWithRunner("herdr", runner)

	_, err := client.CreateTab(context.Background(), "/workspace", "title")
	require.ErrorIs(t, err, wantErr)
	require.ErrorContains(t, err, "pane not found")
}

func tabCreatedOutput(tabID, paneID string) []byte {
	return []byte(`{"result":{"root_pane":{"pane_id":"` + paneID + `"},"tab":{"tab_id":"` + tabID + `"},"type":"tab_created"}}`)
}

func TestContextPrefersFocusedPaneCWD(t *testing.T) {
	context, err := ParseContext(`{"workspace_cwd":"/workspace","focused_pane_cwd":"/workspace/subdir"}`)
	require.NoError(t, err)
	require.Equal(t, "/workspace/subdir", context.Workspace())
}

func TestWorkspaceRootFindsGitRootFromSubdirectory(t *testing.T) {
	repository := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repository, ".git"), 0o755))
	subdirectory := filepath.Join(repository, "some", "nested", "directory")
	require.NoError(t, os.MkdirAll(subdirectory, 0o755))

	got, err := WorkspaceRoot(subdirectory)
	require.NoError(t, err)
	want, err := filepath.EvalSymlinks(repository)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestWorkspaceRootFallsBackToDirectoryOutsideGit(t *testing.T) {
	workspace := t.TempDir()
	got, err := WorkspaceRoot(workspace)
	require.NoError(t, err)
	want, err := filepath.EvalSymlinks(workspace)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
