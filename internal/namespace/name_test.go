package namespace

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceDevboxNameIsValid(t *testing.T) {
	name := WorkspaceDevboxName("/Users/me/My Project")
	require.Regexp(t, regexp.MustCompile(`^herdr-my-project-[a-f0-9]{10}$`), name)
}

func TestWorkspaceDevboxNameIsStable(t *testing.T) {
	first := WorkspaceDevboxName("/Users/me/My Project")
	second := WorkspaceDevboxName("/Users/me/My Project")
	require.Equal(t, first, second)
}

func TestNewDevboxNameIsUnique(t *testing.T) {
	first := NewDevboxName("/Users/me/My Project")
	second := NewDevboxName("/Users/me/My Project")
	require.NotEqual(t, first, second)
}
