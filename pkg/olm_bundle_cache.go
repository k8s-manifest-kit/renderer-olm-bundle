package olmbundle

import (
	"github.com/k8s-manifest-kit/pkg/util/cache"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type olmBundleSpec struct {
	Bundle           string
	TargetNamespaces []string
	DeploymentConfig any
	TLSVerify        bool
	CertDir          string
}

func newCache(opts *cache.Options) cache.Interface[[]unstructured.Unstructured] {
	if opts == nil {
		return nil
	}

	return cache.NewRenderCache(*opts)
}
