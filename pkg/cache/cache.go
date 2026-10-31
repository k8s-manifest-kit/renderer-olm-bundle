package cache

import (
	utilcache "github.com/k8s-manifest-kit/pkg/util/cache"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// OLMBundleSpec contains the data used to generate cache keys for rendered
// OLM bundles. Keys are evaluated against the renderer-wide cache shared by
// all sources of a renderer.
type OLMBundleSpec struct {
	Bundle           string
	TargetNamespaces []string
	DeploymentConfig any
	TLSVerify        bool
	CertDir          string
}

// New creates the renderer-wide cache instance from the shared cache options.
// It returns nil when caching is disabled, and otherwise preserves the shared
// cache's default key behavior without injecting a renderer-specific default.
func New(opts *utilcache.Options) utilcache.Interface[[]unstructured.Unstructured] {
	if opts == nil {
		return nil
	}

	return utilcache.NewRenderCache(*opts)
}
