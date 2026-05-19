# Zote Pulumi

Go library of [Pulumi `ComponentResource`](https://www.pulumi.com/docs/iac/concepts/resources/components/) factories and pure-Go helpers for building reusable Kubernetes-based infrastructure stacks.

This module lives in its own `go.mod` (`github.com/milagre/zote/pulumi`) so Pulumi/Kubernetes/Helm provider dependencies do not leak into the main [`zote/go`](../go) module.

## Layout

### Bootstrap (program wiring)

- `env`, `config` — environment identity and merged per-env YAML loading.
- `cloud` — multi-provider handle passed into components (`Cloud.ForDatabase`, `ForObjectStorage`, load-balancer annotations). Each service that needs cloud placement defines its own small interface (e.g. `svc/mysql/digitalocean`, `svc/objectstorage/digitalocean`); `cloud/digitalocean` implements them.

### Utilities (`util/`)

Stateless helpers imported by components and consumer programs:

- `util/annotations`, `util/endpoint`, `util/profile`, `util/stringdata`, `util/tokens`

### Components

Deployable Pulumi component families (each exposes a `New` and registers a `ComponentResource`):

- **`infra/`** — cluster-scoped systems (once or fixed per environment): `keda`, `metrics_server`, `grafana_stack`, `cert_manager`, `nginx_ingress`, `cloudflare_tunnel`, `prometheus`, … Pass a shared `*infra.Cluster` into each component `New`; it records capability discovery data (ingress class names, `HasKeda`, Pulumiverse `grafana.Provider`, …) without importing sibling infra packages.
- **`svc/`** — namespace-scoped backing services: `rabbitmq`, `redis`, `mysql`, `influxdb`, `timescaledb`, `objectstorage`, …
- **`k8s/`** — workloads: `deployment` (http/proc), `job`, `cronjob`

Shared chart machinery: `internal/helm`.

Package conventions:

- `svc/<name>/internal/<backend>` — backends are internal; parents re-export caller-facing types via aliases.
- `svc/<name>/digitalocean` — optional placement interface for that service’s cloud backend (duplicate per service when shapes match; do not share a global placement package).
- `infra/grafana_stack` may depend on `svc/objectstorage`; `svc` does not depend on `infra`.

## Conventions

- **Type tokens** use the `zote:` prefix with a family segment (e.g. `zote:infra:Grafana`, `zote:svc:Rabbitmq`, `zote:svc:Mysql`, `zote:k8s:Deployment`).
- **Providers are supplied by the caller.** Components accept `opts ...pulumi.ResourceOption` but never construct Kubernetes or Helm providers internally.
- **Child resources** use explicit, stable logical names that match the Kubernetes/Helm names they materialize.
- **Polymorphic backends** (e.g. `svc/mysql` container vs cloud) use a small internal Go interface, optional pointer-typed args, and explicit precedence rules.
