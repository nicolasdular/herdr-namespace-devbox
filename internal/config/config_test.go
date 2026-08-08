package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAppliesTerminalFirstDefaults(t *testing.T) {
	got, err := Parse([]byte(`{}`))
	require.NoError(t, err)
	require.Equal(t, Default, got)
}

func TestParseRejectsUnknownKeys(t *testing.T) {
	_, err := Parse([]byte(`{"agentKind":"codex"}`))
	require.ErrorContains(t, err, "unknown field")
}

func TestParseRejectsUnsafeSessionNames(t *testing.T) {
	for _, input := range []string{`{"sessionName":"hello world"}`, `{"sessionName":123}`} {
		_, err := Parse([]byte(input))
		require.Error(t, err, input)
	}
}

func TestParseRejectsNonObject(t *testing.T) {
	for _, input := range []string{`null`, `[]`, `"hello"`} {
		_, err := Parse([]byte(input))
		require.Error(t, err, input)
	}
}

func TestParseRejectsNullValues(t *testing.T) {
	for _, input := range []string{`{"image":null}`, `{"setupGithub":null}`, `{"site":null}`} {
		_, err := Parse([]byte(input))
		require.Error(t, err, input)
	}
}
