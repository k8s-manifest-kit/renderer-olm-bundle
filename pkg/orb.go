package orb

import (
	"context"
	"fmt"
	"slices"

	registryv1 "github.com/joelanford/library-olm/bundle/registry/v1"
	"github.com/k8s-manifest-kit/engine/pkg/pipeline"
	"github.com/k8s-manifest-kit/engine/pkg/types"
	"github.com/k8s-manifest-kit/pkg/util/cache"
	"github.com/k8s-manifest-kit/pkg/util/k8s"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const rendererType = "orb"

// Source defines a registry+v1 OLM bundle source.
type Source struct {
	// Bundle is an Orb transport reference. Supported transports are docker://,
	// oci:, oci-archive:, dir:, and tar:.
	Bundle string

	// TargetNamespaces are the namespaces watched by the operator. When empty,
	// library-olm derives the target namespaces from the CSV install modes.
	TargetNamespaces []string

	// DeploymentConfig contains optional CSV deployment customizations.
	DeploymentConfig *registryv1.DeploymentConfig

	// Credentials supplies registry credentials dynamically for image sources.
	// A nil callback or nil result uses anonymous access. Use AmbientCredentials
	// explicitly when the container credential store should be consulted.
	Credentials func(context.Context) (*Credentials, error)

	// TLSVerify controls registry TLS verification. It defaults to false to match
	// Orb's current source behavior.
	TLSVerify bool

	// CertDir is the directory containing Docker registry TLS certificates.
	CertDir string

	// PostRenderers are source-specific post-renderers applied before combining
	// this source with other sources.
	PostRenderers []types.PostRenderer
}

// Credentials contains username/password credentials for a registry source.
type Credentials struct {
	Username string
	Password string
	ambient  bool
}

// StaticCredentials returns a credentials callback for fixed registry credentials.
func StaticCredentials(username, password string) func(context.Context) (*Credentials, error) {
	return func(context.Context) (*Credentials, error) {
		return &Credentials{Username: username, Password: password}, nil
	}
}

// AmbientCredentials returns a credentials callback that explicitly enables
// the container image library's ambient credential lookup.
func AmbientCredentials(context.Context) (*Credentials, error) {
	return &Credentials{ambient: true}, nil
}

// SourceSelector decides whether a Source should be rendered.
type SourceSelector = func(ctx context.Context, source Source) (bool, error)

// Renderer handles Orb registry+v1 bundle rendering.
type Renderer struct {
	inputs []*sourceHolder
	opts   RendererOptions
	cache  cache.Interface[[]unstructured.Unstructured]
}

// New creates a new Orb renderer with the given inputs and options.
func New(inputs []Source, opts ...RendererOption) (*Renderer, error) {
	rendererOpts := RendererOptions{
		Filters:      make([]types.Filter, 0),
		Transformers: make([]types.Transformer, 0),
		ContentHash:  true,
	}

	for _, opt := range opts {
		opt.ApplyTo(&rendererOpts)
	}

	holders := make([]*sourceHolder, len(inputs))
	for i := range inputs {
		source := inputs[i]
		source.TargetNamespaces = slices.Clone(source.TargetNamespaces)
		holders[i] = &sourceHolder{Source: source}
		if err := holders[i].Validate(); err != nil {
			return nil, fmt.Errorf("invalid source at index %d: %w", i, err)
		}
	}

	return &Renderer{
		inputs: holders,
		opts:   rendererOpts,
		cache:  newCache(rendererOpts.CacheOptions),
	}, nil
}

// Process executes the rendering logic for all configured inputs.
// Render-time values are ignored because Orb bundles are not templates.
func (r *Renderer) Process(ctx context.Context, _ types.Values) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	for _, holder := range r.inputs {
		selected, err := pipeline.ApplySourceSelectors(ctx, holder.Source, r.opts.SourceSelectors)
		if err != nil {
			return nil, fmt.Errorf("source selector error for Orb bundle %s: %w", holder.Bundle, err)
		}
		if !selected {
			continue
		}

		objects, err := r.renderSingle(ctx, holder)
		if err != nil {
			return nil, fmt.Errorf("error rendering Orb bundle %s: %w", holder.Bundle, err)
		}

		objects, err = pipeline.ApplyPostRenderers(ctx, objects, holder.PostRenderers)
		if err != nil {
			return nil, fmt.Errorf("source post-renderer error for Orb bundle %s: %w", holder.Bundle, err)
		}

		allObjects = append(allObjects, objects...)
	}

	chain := types.BuildPostRendererChain(r.opts.Filters, r.opts.Transformers, r.opts.PostRenderers)
	result, err := pipeline.ApplyPostRenderers(ctx, allObjects, chain)
	if err != nil {
		return nil, fmt.Errorf("renderer post-renderer error: %w", err)
	}

	return result, nil
}

// Name returns the renderer type identifier.
func (r *Renderer) Name() string {
	return rendererType
}

func (r *Renderer) renderSingle(ctx context.Context, holder *sourceHolder) ([]unstructured.Unstructured, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before render: %w", err)
	}

	spec := orbSpec{
		Bundle:           holder.Bundle,
		TargetNamespaces: slices.Clone(holder.TargetNamespaces),
		DeploymentConfig: holder.DeploymentConfig,
		TLSVerify:        holder.TLSVerify,
		CertDir:          holder.CertDir,
	}

	if r.cache != nil {
		r.cache.Sync()
		if cached, found := r.cache.Get(spec); found {
			r.annotate(cached, "", types.RenderOriginCache)
			return cached, nil
		}
	}

	credentials, err := r.credentials(ctx, holder)
	if err != nil {
		return nil, err
	}

	source, err := newBundleSource(holder.ref, sourceOptions{
		credentials: credentials,
		tlsVerify:   holder.TLSVerify,
		certDir:     holder.CertDir,
	})
	if err != nil {
		return nil, fmt.Errorf("creating bundle source: %w", err)
	}

	bundle, err := source.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading bundle: %w", err)
	}

	clientObjects, err := registryv1.ToPlainManifests(bundle, holder.renderOptions()...)
	if err != nil {
		return nil, fmt.Errorf("converting bundle to plain manifests: %w", err)
	}

	objects, err := k8s.ToUnstructuredSlice(clientObjects)
	if err != nil {
		return nil, fmt.Errorf("converting rendered objects: %w", err)
	}

	r.annotate(objects, holder.Bundle, types.RenderOriginLive)
	if r.cache != nil {
		r.cache.Set(spec, objects)
	}

	return objects, nil
}

func (r *Renderer) credentials(ctx context.Context, holder *sourceHolder) (*Credentials, error) {
	if holder.Credentials == nil ||
		holder.ref.transport == transportDir ||
		holder.ref.transport == transportTar {
		return nil, nil
	}

	credentials, err := holder.Credentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting credentials for bundle %q: %w", holder.Bundle, err)
	}

	return credentials, nil
}

func (r *Renderer) annotate(objects []unstructured.Unstructured, bundle, origin string) {
	for i := range objects {
		if r.opts.SourceAnnotations && bundle != "" {
			k8s.SetAnnotation(&objects[i], types.AnnotationSourceType, rendererType)
			k8s.SetAnnotation(&objects[i], types.AnnotationSourcePath, bundle)
		}
		if r.opts.ContentHash && bundle != "" {
			types.SetContentHash(&objects[i])
		}
		if r.opts.SourceAnnotations && origin != "" {
			types.SetRenderOrigin(&objects[i], origin)
		}
	}
}
