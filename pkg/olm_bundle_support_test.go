package olmbundle

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseTransportRef(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		transport transport
		ref       string
		err       error
	}{
		{name: "docker", value: "docker://quay.io/example/bundle:v1", transport: transportDocker, ref: "quay.io/example/bundle:v1"},
		{name: "oci", value: "oci:/tmp/layout", transport: transportOCI, ref: "/tmp/layout"},
		{name: "oci archive", value: "oci-archive:/tmp/bundle.oci", transport: transportOCIArchive, ref: "/tmp/bundle.oci"},
		{name: "directory", value: "dir:./bundle", transport: transportDir, ref: "./bundle"},
		{name: "tar", value: "tar:./bundle.tar.gz", transport: transportTar, ref: "./bundle.tar.gz"},
		{name: "empty", value: " ", err: ErrBundleEmpty},
		{name: "unsupported", value: "helm:./bundle", err: ErrUnknownTransport},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)
			ref, err := parseTransportRef(test.value)
			if test.err != nil {
				g.Expect(err).To(HaveOccurred())
				g.Expect(errors.Is(err, test.err)).To(BeTrue())

				return
			}

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(ref.transport).To(Equal(test.transport))
			g.Expect(ref.ref).To(Equal(test.ref))
		})
	}
}
