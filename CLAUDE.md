# provider-upjet-civo — Agent Guide

This is a **Crossplane v2 Upjet provider** for [Civo](https://www.civo.com/). It wraps the [`civo/civo`](https://github.com/civo/terraform-provider-civo) Terraform provider (terraform-plugin-sdk/v2, imported in-process via `civo.Provider()`) and generates cluster-scoped and namespaced Crossplane managed resources from a declarative config, with no hand-written controllers.

> **Adding a resource**: Use the `add-upjet-resource` skill.

---

## Architecture

### Dual-scope generation

Every resource is generated **twice** from the same `config/`:

| Scope | Root group | ProviderConfig kind | API version example |
|---|---|---|---|
| Cluster | `civo.upbound.io` | `ProviderConfig` | `compute.civo.upbound.io/v1beta1` |
| Namespaced | `civo.m.upbound.io` | `ClusterProviderConfig` | `compute.civo.m.upbound.io/v1beta1` |

`config/provider.go` calls `GetProvider()` (cluster) and `GetProviderNamespaced()` (namespaced). The generator in `cmd/generator/main.go` runs `pipeline.Run(GetProvider(), GetProviderNamespaced(), rootDir)`.

CRD group formula: `<shortGroup>.<rootGroup>` — e.g. short group `compute` + root group `civo.upbound.io` → `compute.civo.upbound.io`.

### Codegen pipeline

`make generate` runs these steps in order:
1. Downloads Terraform ≤1.5 via `build/` submodule tools.
2. Initialises a minimal `.work/terraform/main.tf.json` and runs `terraform providers schema -json` → **`config/schema.json`** (do not edit).
3. Clones the TF provider repo (sparse, tag pinned) and scrapes its docs → **`config/provider-metadata.yaml`** (do not edit).
4. Runs `go generate ./apis/...` which invokes:
   - `cmd/generator/main.go` → rewrites `apis/cluster/`, `apis/namespaced/`, `internal/controller/cluster/`, `internal/controller/namespaced/`, `examples-generated/`, `config/generated.lst`.
   - `controller-gen` → `package/crds/`.
   - `angryjet` → managed-resource methodsets.

---

## Repository structure

```
config/
  provider.go               ← resourcePrefix, modulePath, WithRootGroup, group imports  ✏️
  external_name.go          ← ExternalNameConfigs (also gates generation include list)  ✏️
  schema.json               ← TF provider schema                                        🚫 generated
  provider-metadata.yaml    ← scraped TF provider docs                                  🚫 generated
  generated.lst             ← list of generated resource names                          🚫 generated
  cluster/<group>/config.go ← per-group AddResourceConfigurator (cluster scope)         ✏️
  namespaced/<group>/config.go ← same, namespaced scope (must match cluster)            ✏️
  cluster/provider.go       ← registers group Configure functions (cluster)             ✏️
  namespaced/provider.go    ← registers group Configure functions (namespaced)          ✏️

apis/
  cluster/                  ← generated API types (zz_* files)                          🚫 generated
  namespaced/               ← generated API types (zz_* files)                          🚫 generated
  generate.go               ← //go:generate directives that drive make generate         ✏️ rarely

internal/
  clients/                  ← TF setup fn; provider credential extraction               ✏️
  controller/cluster/       ← generated controllers                                     🚫 generated
  controller/namespaced/    ← generated controllers                                     🚫 generated
  features/                 ← feature flags                                             ✏️ rarely

cmd/
  generator/main.go         ← runs pipeline.Run(GetProvider(), GetProviderNamespaced()) ✏️ rarely
  provider/main.go          ← provider binary entry point                              ✏️ rarely

package/
  crossplane.yaml           ← provider name in the OCI package                         ✏️
  crds/                     ← generated CRD manifests                                  🚫 generated

examples/
  cluster/<group>/          ← curated E2E examples (cluster scope)                     ✏️
  namespaced/<group>/       ← curated E2E examples (namespaced scope)                  ✏️
  cluster/providerconfig/   ← ProviderConfig + Secret template                         ✏️
  namespaced/providerconfig/← ProviderConfig + ClusterProviderConfig + Secret template ✏️
  install.yaml              ← provider install manifest                                ✏️

examples-generated/         ← raw generated examples (starting point only)             🚫 generated

cluster/test/setup.sh       ← E2E cluster setup: creates Secret + ProviderConfig       ✏️

Makefile                    ← provider knobs at the top (see below)                    ✏️
build/                      ← crossplane/build submodule — do not edit                 🚫 submodule
```

`✏️` = hand-written (safe to edit). `🚫` = generated or submodule (never edit directly).

---

## Makefile knobs

The provider-specific variables at the top of `Makefile`:

```makefile
PROVIDER_NAME                  # civo (PROJECT_NAME is derived as provider-$(PROVIDER_NAME))
TERRAFORM_PROVIDER_SOURCE      # civo/civo
TERRAFORM_PROVIDER_REPO        # https://github.com/civo/terraform-provider-civo (for pulling docs)
TERRAFORM_PROVIDER_VERSION     # pinned civo TF provider version
TERRAFORM_DOCS_PATH            # docs/resources (docs dir inside the TF provider repo)
```

`PROJECT_REPO` is derived as `github.com/upbound/$(PROJECT_NAME)` — matches the Go module path `github.com/upbound/provider-civo`.

---

## Key commands

```bash
# After any fresh clone or worktree — MUST run first
git submodule update --init --recursive

# Full codegen (schema pull + doc scrape + code generation)
make generate

# Run only the Go generator (schema + metadata already present)
go run cmd/generator/main.go "$PWD"

# Verify compilation
go build ./...

# Run provider out-of-cluster (needs a kubeconfig)
make run

# E2E test (builds provider, spins up a KinD cluster, runs uptest/chainsaw)
UPTEST_EXAMPLE_LIST="examples/cluster/<group>/<kind>.yaml" \
UPTEST_CLOUD_CREDENTIALS="$(cat path/to/creds.json)" \
UPTEST_DATASOURCE_PATH="path/to/datasource.ini" \
make e2e

# After any E2E run — ALWAYS clean up in this order
kubectl delete managed --all --all-namespaces
kind delete cluster --name local-dev
```

---

## Config key concepts

### ExternalNameConfigs gates generation

A resource absent from `config/external_name.go` is **never generated**, even if it is in `GroupMap`. Adding a resource here is always step one.

### Group and Kind naming

Upjet derives `shortGroup` and `Kind` from the TF resource name by default (strips the provider prefix, splits on `_`). Override via:
- `config/groups.go` `GroupMap` — if the provider uses versioned or multi-word group names.
- `r.ShortGroup` / `r.Kind` inside an `AddResourceConfigurator` call.

### References

Cross-resource references (`r.References["field"] = config.Reference{...}`) go in both `config/cluster/<group>/config.go` AND `config/namespaced/<group>/config.go`. These files must stay in sync.

### Sensitive fields

Any TF attribute marked `Sensitive: true` becomes `<field>SecretRef` in the CRD (references a `v1/Secret`). The plain field name is absent from `forProvider`. Check CRD `keys` when a field seems missing.

---

## E2E setup

`cluster/test/setup.sh` runs during `make e2e`. It:
1. Creates a `provider-secret` Secret in `upbound-system` from `$UPTEST_CLOUD_CREDENTIALS` (JSON with a single `token` key holding the Civo API token).
2. Applies a `ProviderConfig` (cluster scope) and a `ClusterProviderConfig` (namespaced scope).

The `UPTEST_DATASOURCE_PATH` ini file resolves `${data.<key>}` placeholders in example YAML files (e.g. `region: ${data.civo_default_region}`).

---

## Crossplane v2 notes

- Both scopes are `SafeStart`-capable (`package/crossplane.yaml`).
- Cluster-scoped ProviderConfig kind for namespaced resources is **`ClusterProviderConfig`** (not `ProviderConfig`).
- Managed resources in the namespaced scope reference it with `providerConfigRef.kind: ClusterProviderConfig`.
- Namespace for namespaced managed resources and Secrets: **`upbound-system`** in examples (or whatever namespace the user creates them in).
