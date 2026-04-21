# Zote Pulumi

Go library of [Pulumi `ComponentResource`](https://www.pulumi.com/docs/iac/concepts/resources/components/) factories and pure-Go helpers for building reusable Kubernetes-based infrastructure stacks.

This module lives in its own `go.mod` (`github.com/milagre/zote/pulumi`) so Pulumi/Kubernetes/Helm provider dependencies do not leak into the main [`zote/go`](../go) module.

## Layout

- `env`, `config`, `profile` — pure Go structures, validation, and configuration loading.
- `cloud` — the umbrella `Cloud` interface every provider satisfies (currently the load-balancer annotation methods). Per-resource-family capabilities (database placement, object-storage endpoints, …) are *not* on this interface; they live in resource-family-and-provider-specific packages (e.g. `database/digitalocean`) so each provider can expose exactly what that resource family needs.
  - `cloud/<provider>` — concrete provider implementation (e.g. `cloud/digitalocean.Cloud`, constructed via `digitalocean.New()`). The consumer Pulumi program selects one provider per stack and passes it to cloud-polymorphic components. Per-instance context that varies between resources (e.g. the VPC/project a specific database cluster lives in) is obtained from the provider via explicit factory methods (`Cloud.ForDatabase(vpcID, projectID)`), so one provider value can serve many instances living in different networks.
- `database/<provider>` — per-provider interfaces describing the handle a database backend on that provider needs (e.g. `database/digitalocean.Cloud`, with `VPCID`/`ProjectID`). Callers never implement these directly; the matching `cloud/<provider>` factory method returns a value that satisfies them implicitly.
- `infra`
  - `infra/<name>` — one subpackage per piece of shared cluster infrastructure. Each parent package is the sole public entry point for deploying that piece. It accepts caller args, picks a backend, and wires shared client resources (ConfigMap/Secret) around whatever the backend emits.
    - `infra/<name>/internal/<backend>` — backend implementations (container, cloud provider) live under `internal/` so they cannot be imported by consumers. Caller-facing argument types are re-exported from the parent via Go type aliases (e.g. `influxdb.ContainerArgs = container.Args`) so callers only ever depend on `infra/<name>`.
- `k8s`
  - `k8s/deployment` — polymorphic workload facade: callers describe intent through a `Mode` (HTTP or background process) and the facade materializes the matching underlying component, synthesizing public/private hostnames from `Name`, `Namespace`, `PublicDomains`, and `Veneers`.
    - `k8s/deployment/http`, `k8s/deployment/proc` — the concrete workload components the facade delegates to; they may also be used directly when a caller only needs one shape.
  - `k8s/cronjob`, `k8s/job` — scheduled and one-shot task components.
- `database/<engine>` — top-level components with polymorphic backends (container vs cloud provider) selected via explicit args, same `internal/<backend>` convention as `infra/<name>`.

## Conventions

- **Type tokens** always use the `zote:` prefix (e.g. `zote:infra:Grafana`)
- **Providers are supplied by the caller.** Components accept `opts ...pulumi.ResourceOption` (and an explicit provider arg where appropriate) but never construct Kubernetes or Helm providers internally.
- **Child resources** use explicit, stable logical names that match the Kubernetes/Helm names they materialize.
- **Polymorphic backends** (e.g. `database/mysql` with container or cloud implementations) use a small internal Go interface, optional pointer-typed args, and explicit precedence rules. See the plan for details.
