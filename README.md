# Orb Renderer

`renderer-orb` renders OLM registry+v1 bundles through
`github.com/joelanford/library-olm` and returns Kubernetes
`unstructured.Unstructured` objects compatible with k8s-manifest-kit.

## Installation

```bash
go get github.com/k8s-manifest-kit/renderer-orb
```

## Quick start

```go
renderer, err := orb.New([]orb.Source{{
    Bundle: "docker://quay.io/example/operator-bundle:v1.0.0",
}})
if err != nil {
    return err
}

objects, err := renderer.Process(ctx, nil)
```

Supported source transports are `docker://`, `oci:`, `oci-archive:`, `dir:`,
and `tar:`. `dir:` and `tar:` references are useful for local bundle testing.
The renderer does not override the bundle's install namespace; the underlying
library derives the normal OLM namespace behavior.
Registry TLS verification defaults to Orb's current behavior (`false`); set
`TLSVerify: true` for verified registry TLS. This controls image transport
trust and is separate from certificate management.

Registry credentials are anonymous by default. A nil `Credentials` callback or
nil callback result never consults ambient container credentials. To explicitly
enable the container credential store, configure
`Credentials: orb.AmbientCredentials`; use `orb.StaticCredentials` or a custom
callback for explicit credentials.

The renderer does not configure a certificate provider. It never injects
cert-manager or OpenShift service-CA management resources or annotations.

See [`docs/design.md`](docs/design.md), [`docs/development.md`](docs/development.md),
and [`AGENTS.md`](AGENTS.md).

## License

Apache License 2.0. See [LICENSE](LICENSE).
