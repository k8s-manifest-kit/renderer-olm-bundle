package orb

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
)

const directoryBundleRef = "dir:../config/test/bundles/simple"

func TestDirectorySourceDoesNotResolveCredentials(t *testing.T) {
	g := NewWithT(t)
	called := false

	renderer, err := New([]Source{{
		Bundle: directoryBundleRef,
		Credentials: func(context.Context) (*Credentials, error) {
			called = true

			return &Credentials{}, nil
		},
	}})
	g.Expect(err).NotTo(HaveOccurred())

	_, err = renderer.Process(t.Context(), nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(called).To(BeFalse())
}
