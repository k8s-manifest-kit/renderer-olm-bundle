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

Remote registry validation is intentionally not part of the default test
suite. The implementation was manually validated against an OperatorHub
bundle; committed tests use local fixtures so `make test` remains deterministic
and network-independent.

Native Podman/GPGME dependencies are required for image transport tests. Local
`dir:` and `tar:` tests do not need a registry. Keep the renderer dependent on
public OLM bundle/library-olm APIs; do not reach into `joelanford/orb/internal`.
