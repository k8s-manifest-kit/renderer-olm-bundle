# Agent Guide: renderer-olm-bundle

`renderer-olm-bundle` renders Operator Lifecycle Manager registry+v1 bundles using
the public `github.com/joelanford/library-olm` API and returns
`[]unstructured.Unstructured` for the shared manifest pipeline.

## Documentation

- [README](README.md) — overview and quick start.
- [Design](docs/design.md) — transports, rendering, pipeline, and security.
- [Development](docs/development.md) — workflow, fixtures, and integration tests.

## Public API

The package is imported from `github.com/k8s-manifest-kit/renderer-olm-bundle/pkg`.

- `olmbundle.New([]olmbundle.Source{...}, opts...)` creates a renderer.
- `olmbundle.NewEngine(source, opts...)` creates an `engine.Engine` for one source.
- `Source.Bundle` accepts `docker://`, `oci:`, `oci-archive:`, `dir:`, and
  `tar:` references.
- Source rendering supports target namespaces, deployment configuration,
  explicit credentials or explicit ambient-credential opt-in, registry TLS
  settings, and source post-renderers. Install
  namespace is intentionally not overridden by this renderer.
- Renderer options match sibling renderers: filters, transformers,
  post-renderers, source selectors, caching, source annotations, and content
  hashes.

The renderer deliberately does not expose or invoke the upstream certificate-provider
conversion option. Rendering may produce webhook configurations from a bundle,
but it does not create certificate-management resources or inject CA-management
annotations.

## Development

Use Go 1.26.8 and run commands from this directory:

```bash
make test
make fmt
make lint
make lint/fix
make check
```

Unit tests use checked-in registry+v1 bundle fixtures and Gomega. Remote
registry validation is performed manually and is not part of the default test
target.

Do not import the upstream project's `internal` packages. Keep transport behavior
aligned with the upstream project and keep certificate-provider behavior absent.
