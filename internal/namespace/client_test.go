package namespace

import (
	"testing"

	"github.com/stretchr/testify/require"

	"herdr-namespace/internal/config"
)

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
	client := Client{executable: "/path/that/does/not/exist/devbox"}
	authenticated, err := client.IsAuthenticated()
	require.Error(t, err)
	require.False(t, authenticated)
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
