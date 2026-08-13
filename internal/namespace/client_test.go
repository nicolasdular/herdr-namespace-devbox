package namespace

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"herdr-namespace/internal/command"
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

func TestConnectWithoutSessionLetsNamespaceChoose(t *testing.T) {
	runner := &recordingRunner{}
	client := testClient(runner)

	exitCode, err := client.Connect(context.Background(), "box-one", "")
	require.NoError(t, err)
	require.Zero(t, exitCode)
	require.Equal(t, []string{"session", "connect", "box-one"}, runner.calls[0].args)
}

func TestCreateStreamsGeneratedSpecToDevbox(t *testing.T) {
	runner := &recordingRunner{}
	client := testClient(runner)
	spec := Spec{
		Name:       "demo",
		Repository: &Repository{URL: "github.com/acme/demo"},
		Env:        []EnvironmentVariable{{Name: "MISE_DISABLE_TOOLS", Value: "postgres"}},
	}
	options := CreateOptions{Dotfiles: "github.com/acme/dotfiles"}

	require.NoError(t, client.Create(context.Background(), spec, options))
	require.Equal(t, []string{
		"create", "--from", "-", "--from_format", "json",
		"--dotfiles", "github.com/acme/dotfiles",
	}, runner.calls[0].args)
	contents, err := io.ReadAll(runner.calls[0].stdin)
	require.NoError(t, err)
	require.Contains(t, string(contents), `"name":"demo"`)
	require.Contains(t, string(contents), `"url":"github.com/acme/demo"`)
	require.Contains(t, string(contents), `"env":[{"name":"MISE_DISABLE_TOOLS","value":"postgres"}]`)
	require.NotContains(t, string(contents), `"dotfiles"`)
}

func TestUploadCopiesPatchToNamedDevbox(t *testing.T) {
	runner := &recordingRunner{}
	client := testClient(runner)

	require.NoError(t, client.Upload(context.Background(), "box-one", "/tmp/local.patch", "/tmp/remote.patch"))
	require.Equal(t, []string{"upload", "box-one", "/tmp/local.patch", "/tmp/remote.patch"}, runner.calls[0].args)
}

func TestExecRunsCommandInNamedDevbox(t *testing.T) {
	runner := &recordingRunner{}
	client := testClient(runner)

	require.NoError(t, client.Exec(context.Background(), "box-one", "git", "status", "--short"))
	require.Equal(t, []string{"exec", "box-one", "--", "git", "status", "--short"}, runner.calls[0].args)
}

func TestStopForcesNamedDevboxToStop(t *testing.T) {
	runner := &recordingRunner{}
	client := testClient(runner)

	require.NoError(t, client.Stop(context.Background(), "herdr-demo-123"))
	require.Len(t, runner.calls, 1)
	require.Equal(t, []string{"stop", "herdr-demo-123", "--force"}, runner.calls[0].args)
}

func TestDeleteForcesNamedDevboxToExpire(t *testing.T) {
	runner := &recordingRunner{}
	client := testClient(runner)

	require.NoError(t, client.Delete(context.Background(), "herdr-demo-123"))
	require.Len(t, runner.calls, 1)
	require.Equal(t, []string{"expire", "herdr-demo-123", "--force"}, runner.calls[0].args)
}

func TestListReturnsDevboxDetails(t *testing.T) {
	runner := &recordingRunner{output: []byte(`[{"id":"box-1","name":"demo","repository":"github.com/acme/demo","site":"zrh","default_dir":"/workspaces/demo","volume_size_gb":"150","instance_shape":{"virtual_cpu":8,"memory_megabytes":16384}}]`)}
	client := testClient(runner)

	devboxes, err := client.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []Devbox{{
		ID:         "box-1",
		Name:       "demo",
		Repository: "github.com/acme/demo",
		Site:       "zrh",
		DefaultDir: "/workspaces/demo",
		InstanceShape: InstanceShape{
			VirtualCPU:      8,
			MemoryMegabytes: 16384,
		},
	}}, devboxes)
	require.Equal(t, []string{"list", "-o", "json"}, runner.calls[0].args)
}

func TestParseDevboxListAllowsCLIStatusMessages(t *testing.T) {
	output := []byte("No devbox available yet.\n[{\"name\":\"herdr-project-123\"}]\nA new version is available.\n")
	devboxes, err := parseDevboxList(output)
	require.NoError(t, err)
	require.Equal(t, []Devbox{{Name: "herdr-project-123"}}, devboxes)
}

func TestParseDevboxListRejectsMissingJSON(t *testing.T) {
	_, err := parseDevboxList([]byte("not JSON"))
	require.Error(t, err)
}
