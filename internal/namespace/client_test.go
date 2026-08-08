package namespace

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"herdr-namespace/internal/command"
	"herdr-namespace/internal/config"
)

type runnerCall struct {
	executable string
	args       []string
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
}

type recordingRunner struct {
	calls  []runnerCall
	output []byte
	err    error
}

func (r *recordingRunner) CombinedOutput(_ context.Context, executable string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, runnerCall{executable: executable, args: append([]string(nil), args...)})
	return r.output, r.err
}

func (r *recordingRunner) Run(
	_ context.Context,
	executable string,
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	r.calls = append(r.calls, runnerCall{
		executable: executable,
		args:       append([]string(nil), args...),
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
	})
	return r.err
}

func testClient(runner *recordingRunner) Client {
	cmd := command.NewWithRunner("devbox-test", runner).
		WithStreams(bytes.NewBufferString("input"), io.Discard, io.Discard)
	return Client{
		cmd: cmd,
	}
}

func TestMakeDevboxSpec(t *testing.T) {
	got := makeDevboxSpec("herdr-demo-123", config.Default, "")
	want := devboxSpec{
		Name: "herdr-demo-123", Image: "builtin:agents", Size: "m", AccessMode: "private",
		AutoStopIdleTimeout: "1h", Repository: repository{Disabled: true},
		Sessions: []session{{Name: "herdr", Command: "bash"}},
	}
	require.Equal(t, want, got)
}

func TestMakeDevboxSpecAddsGitHubIntegration(t *testing.T) {
	cfg := config.Default
	cfg.SetupGitHub = true
	got := makeDevboxSpec("herdr-demo-123", cfg, "")
	require.NotNil(t, got.Integrations)
	require.True(t, got.Integrations.GitHub.ShareAuth)
}

func TestMakeDevboxSpecAddsRepository(t *testing.T) {
	got := makeDevboxSpec("herdr-demo-123", config.Default, "github.com/acme/demo")
	want := repository{URL: "github.com/acme/demo"}
	require.Equal(t, want, got.Repository)
}

func TestAuthenticationCheckReportsExecutionErrors(t *testing.T) {
	client := Client{cmd: command.New("/path/that/does/not/exist/devbox")}
	authenticated, err := client.IsAuthenticated(context.Background())
	require.Error(t, err)
	require.False(t, authenticated)
}

func TestPreflightIncludesCommandOutput(t *testing.T) {
	wantErr := errors.New("exit status 1")
	runner := &recordingRunner{output: []byte("unsupported installation\n"), err: wantErr}

	err := testClient(runner).Preflight(context.Background())
	require.ErrorIs(t, err, wantErr)
	require.ErrorContains(t, err, "unsupported installation")
	require.Equal(t, []string{"version"}, runner.calls[0].args)
}

func TestLoginUsesInteractiveStreams(t *testing.T) {
	runner := &recordingRunner{}
	client := testClient(runner)

	require.NoError(t, client.Login(context.Background()))
	require.Len(t, runner.calls, 1)
	require.Equal(t, []string{"login"}, runner.calls[0].args)
	require.NotNil(t, runner.calls[0].stdin)
	require.Equal(t, io.Discard, runner.calls[0].stdout)
	require.Equal(t, io.Discard, runner.calls[0].stderr)
}

func TestCreateStreamsGeneratedSpecToDevbox(t *testing.T) {
	runner := &recordingRunner{}
	client := testClient(runner)

	require.NoError(t, client.Create(context.Background(), "demo", config.Default, "github.com/acme/demo"))
	require.Equal(t, []string{"create", "--from", "-", "--from_format", "json"}, runner.calls[0].args)
	spec, err := io.ReadAll(runner.calls[0].stdin)
	require.NoError(t, err)
	require.Contains(t, string(spec), `"name":"demo"`)
	require.Contains(t, string(spec), `"url":"github.com/acme/demo"`)
}

func TestStopForcesNamedDevboxToStop(t *testing.T) {
	runner := &recordingRunner{}
	client := testClient(runner)

	require.NoError(t, client.Stop(context.Background(), "herdr-demo-123"))
	require.Len(t, runner.calls, 1)
	require.Equal(t, []string{"stop", "herdr-demo-123", "--force"}, runner.calls[0].args)
}

func TestParseDevboxListAllowsCLIStatusMessages(t *testing.T) {
	output := []byte("No devbox available yet.\n[{\"name\":\"herdr-project-123\"}]\nA new version is available.\n")
	devboxes, err := parseDevboxList(output)
	require.NoError(t, err)
	require.Equal(t, []devboxSummary{{Name: "herdr-project-123"}}, devboxes)
}

func TestParseDevboxListRejectsMissingJSON(t *testing.T) {
	_, err := parseDevboxList([]byte("not JSON"))
	require.Error(t, err)
}
