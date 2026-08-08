package namespace

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var invalidNameCharacters = regexp.MustCompile(`[^a-z0-9-]+`)

func WorkspaceDevboxName(workspace string) string {
	return devboxName(workspace, "")
}

func NewDevboxName(workspace string) string {
	return devboxName(workspace, rand.Text())
}

func devboxName(workspace, salt string) string {
	base := strings.ToLower(filepath.Base(workspace))
	base = invalidNameCharacters.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if len(base) > 24 {
		base = base[:24]
	}
	if base == "" {
		base = "workspace"
	}
	digest := sha256.Sum256([]byte(workspace + "\x00" + salt))
	return fmt.Sprintf("herdr-%s-%x", base, digest[:5])
}
