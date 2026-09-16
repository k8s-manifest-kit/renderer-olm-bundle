package olmbundle_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/k8s-manifest-kit/engine/pkg/types"
	"github.com/k8s-manifest-kit/pkg/util/cache"
	olmbundle "github.com/k8s-manifest-kit/renderer-olm-bundle/pkg"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	simpleBundleRef  = "dir:../config/test/bundles/simple"
	webhookBundleRef = "dir:../config/test/bundles/webhook"
	testValue        = "true"
)

var (
	configMapGVK     = schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}
	namespaceGVK     = schema.GroupVersionKind{Version: "v1", Kind: "Namespace"}
	validatingGVK    = schema.GroupVersionKind{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"}
	certificateGVK   = schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"}
	issuerGVK        = schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Issuer"}
	clusterIssuerGVK = schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "ClusterIssuer"}
)

func TestNewAndProcessReturnsUnstructuredObjects(t *testing.T) {
	g := NewWithT(t)

	renderer, err := olmbundle.New([]olmbundle.Source{{
		Bundle: simpleBundleRef,
	}})
	g.Expect(err).NotTo(HaveOccurred())

	objects, err := renderer.Process(t.Context(), nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(objects).NotTo(BeEmpty())
	g.Expect(renderer.Name()).To(Equal("olm-bundle"))

	for _, object := range objects {
		g.Expect(object.GetAPIVersion()).NotTo(BeEmpty())
		g.Expect(object.GetKind()).NotTo(BeEmpty())
	}

	g.Expect(findObject(objects, configMapGVK, "simple-operator-config")).NotTo(BeNil())
	g.Expect(findGVK(objects, namespaceGVK)).To(HaveLen(1))
}

func TestProcessAppliesPipelineOptions(t *testing.T) {
	g := NewWithT(t)
	selectorCalled := false

	renderer, err := olmbundle.New([]olmbundle.Source{{Bundle: simpleBundleRef}},
		olmbundle.WithSourceSelector(func(context.Context, olmbundle.Source) (bool, error) {
			selectorCalled = true

			return true, nil
		}),
		olmbundle.WithFilter(func(_ context.Context, object unstructured.Unstructured) (bool, error) {
			return object.GroupVersionKind() == configMapGVK, nil
		}),
		olmbundle.WithTransformer(func(_ context.Context, object unstructured.Unstructured) (unstructured.Unstructured, error) {
			object.SetLabels(map[string]string{"test": testValue})

			return object, nil
		}),
		olmbundle.WithPostRenderer(func(_ context.Context, objects []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
			for i := range objects {
				annotations := objects[i].GetAnnotations()
				if annotations == nil {
					annotations = make(map[string]string)
				}
				annotations["post-rendered"] = "true"
				objects[i].SetAnnotations(annotations)
			}

			return objects, nil
		}),
		olmbundle.WithSourceAnnotations(true),
	)
	g.Expect(err).NotTo(HaveOccurred())

	objects, err := renderer.Process(t.Context(), types.Values{"ignored": "by OLM bundle renderer"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(selectorCalled).To(BeTrue())
	g.Expect(objects).To(HaveLen(1))
	g.Expect(objects[0].GroupVersionKind()).To(Equal(configMapGVK))
	g.Expect(objects[0].GetLabels()).To(Equal(map[string]string{"test": testValue}))
	g.Expect(objects[0].GetAnnotations()).To(And(
		HaveKeyWithValue("post-rendered", "true"),
		HaveKeyWithValue(types.AnnotationSourceType, "olm-bundle"),
		HaveKey(types.AnnotationContentHash),
	))
}

func TestProcessDoesNotInjectCertificateManagement(t *testing.T) {
	g := NewWithT(t)

	renderer, err := olmbundle.New([]olmbundle.Source{{
		Bundle: webhookBundleRef,
	}})
	g.Expect(err).NotTo(HaveOccurred())

	objects, err := renderer.Process(t.Context(), nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(findGVK(objects, validatingGVK)).NotTo(BeEmpty())

	for _, object := range objects {
		g.Expect(object.GroupVersionKind()).NotTo(BeElementOf(certificateGVK, issuerGVK, clusterIssuerGVK))
		for key := range object.GetAnnotations() {
			g.Expect(strings.HasPrefix(key, "cert-manager.io/")).To(BeFalse())
			g.Expect(key).NotTo(Equal("service.beta.openshift.io/inject-cabundle"))
		}
	}
}

func TestCacheReturnsIndependentObjects(t *testing.T) {
	g := NewWithT(t)

	renderer, err := olmbundle.New([]olmbundle.Source{{
		Bundle: simpleBundleRef,
	}}, olmbundle.WithCache(cache.WithTTL(0)), olmbundle.WithSourceAnnotations(true))
	g.Expect(err).NotTo(HaveOccurred())

	first, err := renderer.Process(t.Context(), nil)
	g.Expect(err).NotTo(HaveOccurred())
	first[0].SetName("mutated")

	second, err := renderer.Process(t.Context(), nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(second[0].GetName()).NotTo(Equal("mutated"))
	g.Expect(second[0].GetAnnotations()).To(HaveKeyWithValue(types.AnnotationRenderOrigin, types.RenderOriginCache))
}

func TestContextCancellation(t *testing.T) {
	g := NewWithT(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	renderer, err := olmbundle.New([]olmbundle.Source{{Bundle: simpleBundleRef}})
	g.Expect(err).NotTo(HaveOccurred())

	_, err = renderer.Process(ctx, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("context"))
}

func findObject(objects []unstructured.Unstructured, gvk schema.GroupVersionKind, name string) *unstructured.Unstructured {
	index := slices.IndexFunc(objects, func(object unstructured.Unstructured) bool {
		return object.GroupVersionKind() == gvk && object.GetName() == name
	})
	if index >= 0 {
		return &objects[index]
	}

	return nil
}

func findGVK(objects []unstructured.Unstructured, gvk schema.GroupVersionKind) []unstructured.Unstructured {
	result := make([]unstructured.Unstructured, 0)
	for _, object := range objects {
		if object.GroupVersionKind() == gvk {
			result = append(result, object)
		}
	}

	return result
}
