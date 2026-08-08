package namespace

import (
	"reflect"
	"testing"

	"herdr-namespace/internal/config"
)

func TestMakeDevboxSpec(t *testing.T) {
	got := makeDevboxSpec("herdr-demo-123", config.Default)
	want := devboxSpec{
		Name: "herdr-demo-123", Image: "builtin:agents", Size: "m", AccessMode: "private",
		AutoStopIdleTimeout: "1h", Repository: repository{Disabled: true},
		Sessions: []session{{Name: "herdr", Command: "bash"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestMakeDevboxSpecAddsGitHubIntegration(t *testing.T) {
	cfg := config.Default
	cfg.SetupGitHub = true
	got := makeDevboxSpec("herdr-demo-123", cfg)
	if got.Integrations == nil || !got.Integrations.GitHub.ShareAuth {
		t.Fatalf("missing integration: %#v", got)
	}
}

func TestAuthenticationCheckReportsExecutionErrors(t *testing.T) {
	client := Client{executable: "/path/that/does/not/exist/devbox"}
	authenticated, err := client.IsAuthenticated()
	if err == nil {
		t.Fatal("expected an execution error")
	}
	if authenticated {
		t.Fatal("unexpected authenticated result")
	}
}

func TestParseDevboxListAllowsCLIStatusMessages(t *testing.T) {
	output := []byte("No devbox available yet.\n[{\"name\":\"herdr-project-123\"}]\nA new version is available.\n")
	devboxes, err := parseDevboxList(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(devboxes) != 1 || devboxes[0].Name != "herdr-project-123" {
		t.Fatalf("got %#v", devboxes)
	}
}

func TestParseDevboxListRejectsMissingJSON(t *testing.T) {
	if _, err := parseDevboxList([]byte("not JSON")); err == nil {
		t.Fatal("expected invalid output to fail")
	}
}
