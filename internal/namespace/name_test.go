package namespace

import (
	"regexp"
	"testing"
)

func TestWorkspaceDevboxNameIsValid(t *testing.T) {
	name := WorkspaceDevboxName("/Users/me/My Project")
	if !regexp.MustCompile(`^herdr-my-project-[a-f0-9]{10}$`).MatchString(name) {
		t.Fatalf("invalid name %q", name)
	}
}

func TestWorkspaceDevboxNameIsStable(t *testing.T) {
	first := WorkspaceDevboxName("/Users/me/My Project")
	second := WorkspaceDevboxName("/Users/me/My Project")
	if first != second {
		t.Fatalf("got %q and %q", first, second)
	}
}

func TestNewDevboxNameIsUnique(t *testing.T) {
	first := NewDevboxName("/Users/me/My Project")
	second := NewDevboxName("/Users/me/My Project")
	if first == second {
		t.Fatalf("got duplicate name %q", first)
	}
}
