package herdr

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Herdr struct {
	executable string
}

func New(executable string) Herdr {
	return Herdr{executable: executable}
}

func (h Herdr) Run(args ...string) error {
	command := exec.Command(h.executable, args...)
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s: %w", h.executable, err)
	}
	return nil
}

type Context struct {
	FocusedPaneCWD string `json:"focused_pane_cwd"`
	WorkspaceCWD   string `json:"workspace_cwd"`
	FocusedPaneID  string `json:"focused_pane_id"`
}

func ParseContext(raw string) (Context, error) {
	if raw == "" {
		return Context{}, nil
	}
	var context Context
	if err := json.Unmarshal([]byte(raw), &context); err != nil {
		return Context{}, fmt.Errorf("HERDR_PLUGIN_CONTEXT_JSON is invalid JSON: %w", err)
	}
	return context, nil
}

func (c Context) Workspace() string {
	if c.FocusedPaneCWD != "" {
		return c.FocusedPaneCWD
	}
	return c.WorkspaceCWD
}

func (c Context) Pane(fallback string) string {
	if c.FocusedPaneID != "" {
		return c.FocusedPaneID
	}
	return fallback
}

var invalidNameCharacters = regexp.MustCompile(`[^a-z0-9-]+`)

func GenerateDevboxName(workspace string) string {
	base := strings.ToLower(filepath.Base(workspace))
	base = invalidNameCharacters.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if len(base) > 24 {
		base = base[:24]
	}
	if base == "" {
		base = "workspace"
	}
	digest := sha256.Sum256([]byte(workspace + "\x00" + rand.Text()))
	return fmt.Sprintf("herdr-%s-%x", base, digest[:5])
}

func Environment(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
