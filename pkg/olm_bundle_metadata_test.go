package olmbundle_test

import (
	"testing"

	"github.com/k8s-manifest-kit/engine/pkg/types"
	utilcache "github.com/k8s-manifest-kit/pkg/util/cache"
	"github.com/k8s-manifest-kit/pkg/util/k8s"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	olmbundle "github.com/k8s-manifest-kit/renderer-olm-bundle/pkg"

	. "github.com/onsi/gomega"
)

// renderSimpleBundle processes the simple bundle fixture with the given
// renderer options and returns the single ConfigMap object.
func renderSimpleBundle(t *testing.T, opts ...olmbundle.RendererOption) *unstructured.Unstructured {
	t.Helper()

	g := NewWithT(t)

	renderer, err := olmbundle.New([]olmbundle.Source{{
		Bundle: simpleBundleRef,
	}}, opts...)
	g.Expect(err).NotTo(HaveOccurred())

	objects, err := renderer.Process(t.Context(), nil)
	g.Expect(err).NotTo(HaveOccurred())

	return findSimpleConfigMap(t, objects)
}

// findSimpleConfigMap locates the single ConfigMap of the simple bundle.
func findSimpleConfigMap(t *testing.T, objects []unstructured.Unstructured) *unstructured.Unstructured {
	t.Helper()

	g := NewWithT(t)

	configMaps := findGVK(objects, configMapGVK())
	g.Expect(configMaps).To(HaveLen(1))

	return &configMaps[0]
}

// TestAnnotationApplication characterizes how the OLM bundle renderer applies
// source metadata, content hashes, and render origins on live and cached
// renders.
func TestAnnotationApplication(t *testing.T) {

	t.Run("live render exact annotation map with hash over source metadata before origin", func(t *testing.T) {
		g := NewWithT(t)

		bare := renderSimpleBundle(t, olmbundle.WithContentHash(false))
		g.Expect(bare.GetAnnotations()).To(BeNil())

		expected := bare.DeepCopy()
		expected.SetAnnotations(map[string]string{
			types.AnnotationSourceType: "olm-bundle",
			types.AnnotationSourcePath: simpleBundleRef,
		})

		annotated := renderSimpleBundle(t, olmbundle.WithSourceAnnotations(true))

		g.Expect(annotated.GetAnnotations()).To(Equal(map[string]string{
			types.AnnotationSourceType:   "olm-bundle",
			types.AnnotationSourcePath:   simpleBundleRef,
			types.AnnotationContentHash:  k8s.ContentHash(expected),
			types.AnnotationRenderOrigin: types.RenderOriginLive,
		}))
	})

	t.Run("source annotations without content hash leave no hash key", func(t *testing.T) {
		g := NewWithT(t)

		annotated := renderSimpleBundle(t,
			olmbundle.WithSourceAnnotations(true),
			olmbundle.WithContentHash(false),
		)

		annotations := annotated.GetAnnotations()
		g.Expect(annotations).To(HaveLen(3))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "olm-bundle"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourcePath, simpleBundleRef))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationRenderOrigin, types.RenderOriginLive))
		g.Expect(annotations).NotTo(HaveKey(types.AnnotationContentHash))
	})

	t.Run("disabled annotations hash by default and inject no type, path, or origin", func(t *testing.T) {
		g := NewWithT(t)

		bare := renderSimpleBundle(t, olmbundle.WithContentHash(false))

		defaults := renderSimpleBundle(t)

		annotations := defaults.GetAnnotations()
		g.Expect(annotations).To(HaveKeyWithValue(
			types.AnnotationContentHash,
			k8s.ContentHash(bare.DeepCopy()),
		))
		g.Expect(annotations).NotTo(HaveKey(types.AnnotationSourceType))
		g.Expect(annotations).NotTo(HaveKey(types.AnnotationSourcePath))
		g.Expect(annotations).NotTo(HaveKey(types.AnnotationRenderOrigin))
	})

	t.Run("warm hit retains stored source fields and hash bytes and swaps the origin only", func(t *testing.T) {
		g := NewWithT(t)

		renderer, err := olmbundle.New([]olmbundle.Source{{
			Bundle: simpleBundleRef,
		}},
			olmbundle.WithCache(utilcache.WithTTL(0)),
			olmbundle.WithSourceAnnotations(true),
		)
		g.Expect(err).NotTo(HaveOccurred())

		live, err := renderer.Process(t.Context(), nil)
		g.Expect(err).NotTo(HaveOccurred())

		liveConfigMap := findSimpleConfigMap(t, live)

		warm, err := renderer.Process(t.Context(), nil)
		g.Expect(err).NotTo(HaveOccurred())

		warmConfigMap := findSimpleConfigMap(t, warm)

		liveAnnotations := liveConfigMap.GetAnnotations()
		warmAnnotations := warmConfigMap.GetAnnotations()

		g.Expect(warmAnnotations).To(HaveKeyWithValue(types.AnnotationRenderOrigin, types.RenderOriginCache))
		g.Expect(warmAnnotations[types.AnnotationSourceType]).To(Equal("olm-bundle"))
		g.Expect(warmAnnotations[types.AnnotationSourcePath]).To(Equal(simpleBundleRef))
		// The empty-bundle hit path applies no hash option: the stored hash
		// bytes are re-served without recomputation.
		g.Expect(warmAnnotations[types.AnnotationContentHash]).To(Equal(liveAnnotations[types.AnnotationContentHash]))
	})

	t.Run("warm hit with disabled annotations injects no origin and keeps the stored hash", func(t *testing.T) {
		g := NewWithT(t)

		renderer, err := olmbundle.New([]olmbundle.Source{{
			Bundle: simpleBundleRef,
		}},
			olmbundle.WithCache(utilcache.WithTTL(0)),
		)
		g.Expect(err).NotTo(HaveOccurred())

		live, err := renderer.Process(t.Context(), nil)
		g.Expect(err).NotTo(HaveOccurred())

		warm, err := renderer.Process(t.Context(), nil)
		g.Expect(err).NotTo(HaveOccurred())

		liveConfigMap := findSimpleConfigMap(t, live)
		warmConfigMap := findSimpleConfigMap(t, warm)

		warmAnnotations := warmConfigMap.GetAnnotations()
		g.Expect(warmAnnotations).NotTo(HaveKey(types.AnnotationRenderOrigin))
		g.Expect(warmAnnotations[types.AnnotationContentHash]).To(Equal(
			liveConfigMap.GetAnnotations()[types.AnnotationContentHash],
		))
	})
}
