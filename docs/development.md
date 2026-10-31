# OLM Bundle Renderer Development

Use Go 1.26.8 and run:

```bash
make deps
make test
make fmt
make lint
make check
```

The unit suite covers local bundle rendering, namespace options, the shared
pipeline, caching, tar input, cancellation, and the certificate-free webhook
path. Fixtures live under `config/test/bundles` and follow registry+v1 bundle
layout.

## Package layout

- `pkg/olm_bundle.go` — bundle rendering and renderer orchestration.
- `pkg/olm_bundle_option.go` — renderer options and defaults.
- `pkg/olm_bundle_support.go` — shared implementation helpers.
- `pkg/olm_bundle_cache.go` — output-cache construction and cache keys.
- `pkg/olm_bundle_source.go` — the source model and transport dispatch.
- `pkg/olm_bundle_source_directory.go`, `pkg/olm_bundle_source_image.go`,
  `pkg/olm_bundle_source_tar.go` — directory, image, and tar transports.
- `pkg/engine.go` — the `NewEngine` convenience constructor.

Renderer options are typed by the shared
`github.com/k8s-manifest-kit/pkg/util/option` package. The layout follows
the shared [`project-layout.md`](../../docs/project-layout.md) convention
(workspace navigation).

Remote registry validation is intentionally not part of the default test
suite. The implementation was manually validated against an OperatorHub
bundle; committed tests use local fixtures so `make test` remains deterministic
and network-independent.

Native Podman/GPGME dependencies are required for image transport tests. Local
`dir:` and `tar:` tests do not need a registry. Keep the renderer dependent on
public OLM bundle/library-olm APIs; do not reach into `joelanford/orb/internal`.
