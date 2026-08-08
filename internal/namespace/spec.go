package namespace

import (
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

func WorkspaceSpec(workspace string) (path, name string) {
	path = filepath.Join(workspace, "devbox.yaml")
	contents, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ""
		}
		return path, ""
	}

	var spec struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal(contents, &spec); err != nil {
		return path, ""
	}
	return path, spec.Name
}
