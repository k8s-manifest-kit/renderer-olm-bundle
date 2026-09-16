package olmbundle

import (
	"context"
	"fmt"
	"os"

	registryv1 "github.com/joelanford/library-olm/bundle/registry/v1"
	"github.com/joelanford/library-olm/image"
	imagebundle "github.com/joelanford/library-olm/image/bundle"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	dockerTransport "go.podman.io/image/v5/docker"
	archiveTransport "go.podman.io/image/v5/oci/archive"
	layoutTransport "go.podman.io/image/v5/oci/layout"
	imageTypes "go.podman.io/image/v5/types"
)

type imageBundleSource struct {
	ref  transportRef
	opts sourceOptions
}

func (s *imageBundleSource) Read(ctx context.Context) (registryv1.Bundle, error) {
	return s.readImage(ctx)
}

func (s *imageBundleSource) readImage(ctx context.Context) (registryv1.Bundle, error) {
	imgRef, err := parseImageReference(s.ref)
	if err != nil {
		return registryv1.Bundle{}, err
	}

	return readFromImage(ctx, imgRef, buildSystemContext(s.opts))
}

func parseImageReference(ref transportRef) (imageTypes.ImageReference, error) {
	switch ref.transport {
	case transportDocker:
		imgRef, err := dockerTransport.ParseReference("//" + ref.ref)
		if err != nil {
			return nil, fmt.Errorf("parsing docker reference: %w", err)
		}

		return imgRef, nil
	case transportOCI:
		imgRef, err := layoutTransport.ParseReference(ref.ref)
		if err != nil {
			return nil, fmt.Errorf("parsing oci layout reference: %w", err)
		}

		return imgRef, nil
	case transportOCIArchive:
		imgRef, err := archiveTransport.ParseReference(ref.ref)
		if err != nil {
			return nil, fmt.Errorf("parsing oci-archive reference: %w", err)
		}

		return imgRef, nil
	case transportDir, transportTar:
		return nil, fmt.Errorf("%w: local transport %q is not an image reference", ErrUnknownTransport, ref.ref)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownTransport, ref.ref)
	}
}

func buildSystemContext(opts sourceOptions) *imageTypes.SystemContext {
	sysCtx := &imageTypes.SystemContext{}

	if !opts.tlsVerify {
		sysCtx.DockerInsecureSkipTLSVerify = imageTypes.OptionalBoolTrue
		sysCtx.OCIInsecureSkipTLSVerify = true
	}
	if opts.certDir != "" {
		sysCtx.DockerCertPath = opts.certDir
	}
	if opts.credentials == nil {
		// A non-nil auth config prevents Podman from consulting ambient stores.
		sysCtx.DockerAuthConfig = &imageTypes.DockerAuthConfig{}
	} else if !opts.credentials.ambient {
		sysCtx.DockerAuthConfig = &imageTypes.DockerAuthConfig{
			Username: opts.credentials.Username,
			Password: opts.credentials.Password,
		}
	}

	return sysCtx
}

func readFromImage(
	ctx context.Context,
	imgRef imageTypes.ImageReference,
	sysCtx *imageTypes.SystemContext,
) (registryv1.Bundle, error) {
	client, err := image.NewContainersImageRepository(
		ctx,
		imgRef,
		sysCtx,
		image.WithSignatureVerification(image.VerifyIfPresent),
	)
	if err != nil {
		return registryv1.Bundle{}, fmt.Errorf("creating image repository: %w", err)
	}

	repo, err := image.NewCachingRepository(client)
	if err != nil {
		_ = client.Close()

		return registryv1.Bundle{}, fmt.Errorf("creating caching repository: %w", err)
	}
	defer func() { _ = repo.Close() }()

	handler := &imagebundle.RegistryV1Handler{}
	desc, manifestBytes, err := resolveAndMatch(ctx, repo, handler)
	if err != nil {
		return registryv1.Bundle{}, err
	}

	tmpDir, err := os.MkdirTemp("", "olm-bundle-")
	if err != nil {
		return registryv1.Bundle{}, fmt.Errorf("creating temporary bundle directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := handler.Unpack(ctx, repo, desc, manifestBytes, tmpDir); err != nil {
		return registryv1.Bundle{}, fmt.Errorf("unpacking image: %w", err)
	}

	bundle, err := registryv1.FromFS(os.DirFS(tmpDir))
	if err != nil {
		return registryv1.Bundle{}, fmt.Errorf("parsing bundle: %w", err)
	}

	return bundle, nil
}

func resolveAndMatch(
	ctx context.Context,
	repo image.Repository,
	handler *imagebundle.RegistryV1Handler,
) (ocispecv1.Descriptor, []byte, error) {
	desc, err := repo.Resolve(ctx)
	if err != nil {
		return ocispecv1.Descriptor{}, nil, fmt.Errorf("resolving image: %w", err)
	}

	manifestBytes, mediaType, err := repo.FetchManifest(ctx, desc)
	if err != nil {
		return ocispecv1.Descriptor{}, nil, fmt.Errorf("fetching manifest: %w", err)
	}
	desc.MediaType = mediaType

	matched, err := handler.Matches(ctx, repo, desc, manifestBytes)
	if err != nil {
		return ocispecv1.Descriptor{}, nil, fmt.Errorf("checking image type: %w", err)
	}
	if !matched {
		return ocispecv1.Descriptor{}, nil, fmt.Errorf("image does not match %s handler", handler.Name())
	}

	return desc, manifestBytes, nil
}
