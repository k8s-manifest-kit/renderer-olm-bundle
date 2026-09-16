package olmbundle

import (
	"errors"
	"fmt"
	"strings"

	registryv1 "github.com/joelanford/library-olm/bundle/registry/v1"
)

var (
	// ErrBundleEmpty is returned when a source has no bundle reference.
	ErrBundleEmpty = errors.New("bundle reference is required")

	// ErrUnknownTransport is returned for unsupported bundle transport prefixes.
	ErrUnknownTransport = errors.New("unknown bundle transport")
)

const renderOptionCapacity = 2

type transport int

const (
	transportDocker transport = iota
	transportOCI
	transportOCIArchive
	transportDir
	transportTar
)

type transportRef struct {
	transport transport
	ref       string
}

func parseTransportRef(value string) (transportRef, error) {
	if strings.TrimSpace(value) == "" {
		return transportRef{}, ErrBundleEmpty
	}

	prefixes := []struct {
		prefix string
		kind   transport
	}{
		{prefix: "docker://", kind: transportDocker},
		{prefix: "oci-archive:", kind: transportOCIArchive},
		{prefix: "oci:", kind: transportOCI},
		{prefix: "tar:", kind: transportTar},
		{prefix: "dir:", kind: transportDir},
	}

	for _, prefix := range prefixes {
		if ref, ok := strings.CutPrefix(value, prefix.prefix); ok {
			if strings.TrimSpace(ref) == "" {
				return transportRef{}, fmt.Errorf("%w: %q has an empty reference", ErrUnknownTransport, value)
			}

			return transportRef{transport: prefix.kind, ref: ref}, nil
		}
	}

	return transportRef{}, fmt.Errorf("%w in %q", ErrUnknownTransport, value)
}

type sourceHolder struct {
	Source

	ref transportRef
}

func (h *sourceHolder) Validate() error {
	ref, err := parseTransportRef(h.Bundle)
	if err != nil {
		return err
	}

	h.ref = ref

	return nil
}

func (s Source) renderOptions() []registryv1.RenderOption {
	options := make([]registryv1.RenderOption, 0, renderOptionCapacity)
	if len(s.TargetNamespaces) > 0 {
		options = append(options, registryv1.WithTargetNamespaces(s.TargetNamespaces...))
	}
	if s.DeploymentConfig != nil {
		options = append(options, registryv1.WithDeploymentConfig(s.DeploymentConfig))
	}

	return options
}
