package orb

import (
	"github.com/k8s-manifest-kit/engine/pkg/types"
	"github.com/k8s-manifest-kit/pkg/util"
	"github.com/k8s-manifest-kit/pkg/util/cache"
)

// RendererOption is a generic option for RendererOptions.
type RendererOption = util.Option[RendererOptions]

// RendererOptions contains renderer-level processing configuration.
type RendererOptions struct {
	Filters           []types.Filter
	Transformers      []types.Transformer
	PostRenderers     []types.PostRenderer
	SourceSelectors   []SourceSelector
	CacheOptions      *cache.Options
	SourceAnnotations bool
	ContentHash       bool
}

// ApplyTo applies the renderer options to the target configuration.
func (opts RendererOptions) ApplyTo(target *RendererOptions) {
	target.Filters = opts.Filters
	target.Transformers = opts.Transformers
	target.PostRenderers = opts.PostRenderers
	target.SourceSelectors = opts.SourceSelectors
	target.SourceAnnotations = opts.SourceAnnotations
	target.ContentHash = opts.ContentHash

	if opts.CacheOptions != nil {
		if target.CacheOptions == nil {
			target.CacheOptions = &cache.Options{}
		}
		opts.CacheOptions.ApplyTo(target.CacheOptions)
	}
}

// WithFilter adds a renderer-specific filter.
func WithFilter(filter types.Filter) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.Filters = append(opts.Filters, filter)
	})
}

// WithTransformer adds a renderer-specific transformer.
func WithTransformer(transformer types.Transformer) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.Transformers = append(opts.Transformers, transformer)
	})
}

// WithPostRenderer adds a renderer-specific post-renderer.
func WithPostRenderer(postRenderer types.PostRenderer) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.PostRenderers = append(opts.PostRenderers, postRenderer)
	})
}

// WithSourceSelector adds a source selector.
func WithSourceSelector(selector SourceSelector) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.SourceSelectors = append(opts.SourceSelectors, selector)
	})
}

// WithCache enables render result caching with the specified options.
func WithCache(opts ...cache.Option) RendererOption {
	return util.FunctionalOption[RendererOptions](func(rendererOpts *RendererOptions) {
		if rendererOpts.CacheOptions == nil {
			rendererOpts.CacheOptions = &cache.Options{}
		}

		for _, opt := range opts {
			opt.ApplyTo(rendererOpts.CacheOptions)
		}
	})
}

// WithSourceAnnotations enables or disables source tracking annotations.
func WithSourceAnnotations(enabled bool) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.SourceAnnotations = enabled
	})
}

// WithContentHash enables or disables content hashes.
func WithContentHash(enabled bool) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.ContentHash = enabled
	})
}
