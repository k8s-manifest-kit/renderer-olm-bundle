package olmbundle

import (
	"context"
	"fmt"

	registryv1 "github.com/joelanford/library-olm/bundle/registry/v1"
)

type bundleSource interface {
	Read(ctx context.Context) (registryv1.Bundle, error)
}

type sourceOptions struct {
	credentials *Credentials
	tlsVerify   bool
	certDir     string
}

func newBundleSource(ref transportRef, opts sourceOptions) (bundleSource, error) {
	switch ref.transport {
	case transportDocker, transportOCI, transportOCIArchive:
		return &imageBundleSource{ref: ref, opts: opts}, nil
	case transportDir:
		return &directoryBundleSource{path: expandPath(ref.ref)}, nil
	case transportTar:
		return &tarBundleSource{path: expandPath(ref.ref)}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownTransport, ref.ref)
	}
}
