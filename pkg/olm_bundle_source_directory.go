package olmbundle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	registryv1 "github.com/joelanford/library-olm/bundle/registry/v1"
)

type directoryBundleSource struct {
	path string
}

func (s *directoryBundleSource) Read(context.Context) (registryv1.Bundle, error) {
	bundle, err := registryv1.FromFS(os.DirFS(s.path))
	if err != nil {
		return registryv1.Bundle{}, fmt.Errorf("reading bundle directory: %w", err)
	}

	return bundle, nil
}

func expandPath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, path[2:])
}
