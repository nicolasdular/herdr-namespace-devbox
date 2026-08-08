package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseAppliesTerminalFirstDefaults(t *testing.T) {
	got, err := Parse([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, Default) {
		t.Fatalf("got %#v, want %#v", got, Default)
	}
}

func TestParseRejectsUnknownKeys(t *testing.T) {
	_, err := Parse([]byte(`{"agentKind":"codex"}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("got error %v", err)
	}
}

func TestParseRejectsUnsafeSessionNames(t *testing.T) {
	for _, input := range []string{`{"sessionName":"hello world"}`, `{"sessionName":123}`} {
		if _, err := Parse([]byte(input)); err == nil {
			t.Fatalf("expected %s to fail", input)
		}
	}
}

func TestParseRejectsNonObject(t *testing.T) {
	for _, input := range []string{`null`, `[]`, `"hello"`} {
		if _, err := Parse([]byte(input)); err == nil {
			t.Fatalf("expected %s to fail", input)
		}
	}
}

func TestParseRejectsNullValues(t *testing.T) {
	for _, input := range []string{`{"image":null}`, `{"setupGithub":null}`, `{"site":null}`} {
		if _, err := Parse([]byte(input)); err == nil {
			t.Fatalf("expected %s to fail", input)
		}
	}
}
