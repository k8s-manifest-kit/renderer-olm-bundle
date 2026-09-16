# Orb Renderer Design

## Source loading

The renderer follows Orb's registry+v1 source model while using only public
APIs. It accepts:

- `docker://` registry references;
- `oci:` OCI layout directories;
- `oci-archive:` OCI archive files;
- `dir:` extracted registry+v1 bundle directories; and
- `tar:` tar, tar.gz, or other formats recognized by Podman's automatic
  decompressor.

Image sources use the Podman image transport and Orb's signature behavior,
then unpack through `library-olm/image/bundle.RegistryV1Handler`. Local
archives are extracted into temporary directories and removed after parsing.
Tar extraction rejects absolute paths, parent traversal, links, and files over
100 MiB.

## Rendering pipeline

Each selected source is read and converted with
`registryv1.ToPlainManifests`. The source-specific post-renderer runs next;
selected source outputs are combined and passed through renderer-level filters,
transformers, and post-renderers. Render-time values are accepted by the
shared interface but do not template or otherwise modify an OLM bundle.

The renderer does not provide an install namespace override. The library
derives the install namespace and emits the corresponding OLM resources using
its normal bundle behavior. `TargetNamespaces`, when explicitly supplied,
continues to control operator scope and generated webhook selectors.

By default content hashes are enabled, source annotations are disabled, and
caching is disabled. `WithCache` enables the shared clone-safe cache.

Image credentials are explicit. A missing `Credentials` callback, or a
callback that returns nil, uses anonymous access. Set
`Credentials: orb.AmbientCredentials` to explicitly enable the container
image library's ambient credential lookup.

## Certificate behavior

The library's conversion options include a certificate provider, but this
renderer intentionally never supplies one and does not expose that option.
Webhook resources in a bundle can therefore be rendered, but the renderer
does not generate `Certificate`, `Issuer`, or `ClusterIssuer` resources and
does not add cert-manager or service-CA injection annotations. Registry
credentials and TLS trust settings are image-transport configuration, not
certificate management.

## Security and lifecycle

Credential callbacks run only when a non-local source is actually read; cache
hits do not invoke them. Temporary image and archive directories are cleaned
after each render. Context cancellation is checked before rendering and during
archive extraction.
