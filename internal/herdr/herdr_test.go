package herdr

import (
	"regexp"
	"testing"
)

func TestContextPrefersFocusedPaneCWD(t *testing.T) {
	context, err := ParseContext(`{"workspace_cwd":"/workspace","focused_pane_cwd":"/workspace/subdir"}`)
	if err != nil {
		t.Fatal(err)
	}
	if got := context.Workspace(); got != "/workspace/subdir" {
		t.Fatalf("got %q", got)
	}
}

func TestGenerateDevboxNameIsValid(t *testing.T) {
	name := GenerateDevboxName("/Users/me/My Project")
	if !regexp.MustCompile(`^herdr-my-project-[a-f0-9]{10}$`).MatchString(name) {
		t.Fatalf("invalid name %q", name)
	}
}
