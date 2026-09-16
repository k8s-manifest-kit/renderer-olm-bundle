// Package orb renders OLM registry+v1 bundles into Kubernetes manifests.
package orb

import (
	"fmt"

	engine "github.com/k8s-manifest-kit/engine/pkg"
)

// NewEngine creates an Engine configured with a single Orb renderer.
func NewEngine(source Source, opts ...RendererOption) (*engine.Engine, error) {
	renderer, err := New([]Source{source}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create orb renderer: %w", err)
	}

	e, err := engine.New(engine.WithRenderer(renderer))
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	return e, nil
}
