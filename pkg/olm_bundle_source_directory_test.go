package olmbundle_test

import (
	"context"
	"testing"

	olmbundle "github.com/k8s-manifest-kit/renderer-olm-bundle/pkg"

	. "github.com/onsi/gomega"
)

const directoryBundleRef = "dir:../config/test/bundles/simple"

func TestDirectorySourceDoesNotResolveCredentials(t *testing.T) {
	g := NewWithT(t)
	called := false

	renderer, err := olmbundle.New([]olmbundle.Source{{
		Bundle: directoryBundleRef,
		Credentials: func(context.Context) (*olmbundle.Credentials, error) {
			called = true

			return &olmbundle.Credentials{}, nil
		},
	}})
	g.Expect(err).NotTo(HaveOccurred())

	_, err = renderer.Process(t.Context(), nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(called).To(BeFalse())
}
