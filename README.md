# Provider Civo

`provider-civo` is a [Crossplane](https://crossplane.io/) provider for
[Civo](https://www.civo.com/), built with
[Upjet](https://github.com/crossplane/upjet) and backed by the
[`civo/civo`](https://github.com/civo/terraform-provider-civo)
Terraform provider.

It exposes XRM-conformant managed resources that let you manage Civo
infrastructure - compute instances, volumes, networks, firewalls, reserved
IPs, DNS, object stores, managed databases and Kubernetes clusters - directly
from Kubernetes or Upbound. Every resource is available in two flavors:
cluster-scoped (`*.civo.upbound.io`) and namespaced (`*.civo.m.upbound.io`).

## Authentication

The provider authenticates with a Civo API token read from the referenced
`Secret`. Create one in the [Civo dashboard](https://dashboard.civo.com/security)
under Security > API Keys and store it as JSON:

```json
{
  "token": "<Civo API token>"
}
```

### Credentials on disk

The upstream Terraform provider takes the token from a credentials file. For
every reconcile the provider writes the token to a file only its own user can
read under `/tmp/provider-civo` (`--credentials-dir` flag, `CREDENTIALS_DIR`
environment variable), configures the upstream provider with it and removes
the file again. The file exists for one configure call: a few milliseconds
plus the API round trip upstream makes to validate the token. To keep it off
the node's disk, or when the container runs with a read-only root filesystem,
mount a memory-backed volume at that path:

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: provider-civo
spec:
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
          - name: package-runtime
            volumeMounts:
            - name: credentials
              mountPath: /tmp/provider-civo
          volumes:
          - name: credentials
            emptyDir:
              medium: Memory
```

The provider refuses to start when `CIVO_TOKEN` is set in its environment: the
upstream provider reads that variable before the credentials file, so every
ProviderConfig would end up using it.

## Getting Started

### 1. Install the provider

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-civo
spec:
  package: xpkg.upbound.io/upbound/provider-civo:v1.0.0
```

### 2. Create a credentials Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: civo-creds
  namespace: upbound-system
type: Opaque
stringData:
  creds: |
    {
      "token": "<Civo API token>"
    }
```

### 3. Create a ProviderConfig

```yaml
apiVersion: civo.upbound.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: civo-creds
      namespace: upbound-system
      key: creds
```

### 4. Create a managed resource

More examples for each resource are available under [`examples/cluster/`](examples/cluster/)
and [`examples/namespaced/`](examples/namespaced/).

## ProviderConfig fields

| Field | Required | Description |
|---|---|---|
| `spec.credentials.source` | Yes | One of `Secret`, `InjectedIdentity`, `Environment`, `Filesystem` |
| `spec.credentials.secretRef` | When source=Secret | Reference to the credentials Secret |
| `spec.apiEndpoint` | No | Base URL of the Civo API; defaults to the public `https://api.civo.com`, set it only when Civo has given you a dedicated endpoint |
| `spec.reconciliationPolicy` | No | Rate-limiting policy for reconciliation |

## Regions

A `ProviderConfig` is not tied to a region. Every regional managed resource
takes its region in `spec.forProvider.region` (for example `LON1`, `NYC1`,
`FRA1` or `PHX1`); when omitted, the resource is created in the account's
default region. Global resources (`dns` `Domain` and `Record`, `compute`
`SSHKey`) have no region. Resource IDs such as networks and disk images are
region-scoped, so references between resources must stay within one region.

`kubernetes` `NodePool` also takes `spec.forProvider.region` and needs it
whenever its cluster is outside the account's default region: the upstream
Terraform resource has no region attribute, so the provider applies this value
to the Civo API client for every call made for that resource instead.

## Developing

### Code generation

```console
make generate
```

This runs the Upjet code generator against the pinned `civo/civo`
Terraform provider schema and docs, and writes the generated APIs,
controllers, and CRDs for both the cluster-scoped and namespaced variants.

### Run locally against a cluster

```console
make run
```

### Run end-to-end tests

```console
make e2e
```

## Reporting issues

Please open an [issue](https://github.com/upbound/provider-upjet-civo/issues)
for bug reports, feature requests, or questions.
