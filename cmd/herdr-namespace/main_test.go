package main

import (
	"testing"
)

func TestMissingActionIDHasNoDefault(t *testing.T) {
	t.Setenv("HERDR_PLUGIN_ACTION_ID", "")
	if _, err := prepareEnv(); err == nil || err.Error() != "HERDR_PLUGIN_ACTION_ID is required" {
		t.Fatalf("got %v", err)
	}
}
