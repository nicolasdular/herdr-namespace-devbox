package namespace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadCreateOptionsLoadsDotfiles(t *testing.T) {
	configDirectory := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(configDirectory, pluginConfigName),
		[]byte(`{
  "size": "m",
  "dotfiles": "github.com/acme/dotfiles"
}`),
		0o600,
	))

	options, err := LoadCreateOptions(configDirectory)
	require.NoError(t, err)
	require.Equal(t, CreateOptions{Dotfiles: "github.com/acme/dotfiles"}, options)
}

func TestLoadCreateOptionsAllowsMissingConfigDirectory(t *testing.T) {
	options, err := LoadCreateOptions("")
	require.NoError(t, err)
	require.Empty(t, options)
}

func TestLoadCreateOptionsAllowsMissingConfigFile(t *testing.T) {
	options, err := LoadCreateOptions(t.TempDir())
	require.NoError(t, err)
	require.Empty(t, options)
}

func TestLoadCreateOptionsRejectsInvalidJSON(t *testing.T) {
	configDirectory := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(configDirectory, pluginConfigName),
		[]byte("{"),
		0o600,
	))

	_, err := LoadCreateOptions(configDirectory)
	require.ErrorContains(t, err, "unexpected end of JSON input")
	require.ErrorContains(t, err, pluginConfigName)
}
