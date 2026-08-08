package command

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFailureIncludesOutputAndPreservesCause(t *testing.T) {
	wantErr := errors.New("exit status 1")
	err := failure("tool", []string{"check"}, []byte("  useful detail\n"), wantErr)

	require.ErrorIs(t, err, wantErr)
	require.EqualError(t, err, `run "tool" "check": exit status 1: useful detail`)
}

func TestFailureOmitsBlankOutput(t *testing.T) {
	err := failure("tool", nil, []byte("\n"), errors.New("not found"))
	require.EqualError(t, err, `run "tool": not found`)
}

func TestOSRunnerReturnsContextCancellation(t *testing.T) {
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = (OSRunner{}).CombinedOutput(ctx, executable, "-test.run=^$")
	require.ErrorIs(t, err, context.Canceled)
}
