# Roadmap

The Keeper Kubernetes Injector is a **focused, Keeper-native tool** — it integrates only with Keeper
Secrets Manager (it is intentionally *not* a generic multi-backend operator). This file tracks planned
work and explicitly-deferred items.

## Planned (near-term)

- **ConfigMap templates** — render non-secret configuration alongside injected secrets.
- **Secret validation** — validate fetched records/fields before injection.

## Backlog (ESO-parity, by demand)

- **Write-back to Keeper** (ESO `PushSecret`-style — push a pod/K8s Secret value *into* Keeper).
  Deferred: the injector is read-only today; revisit if customers ask. Lower priority for an injector.
- **Generators** — generate a password/token and store it (parity with ESO generators).
- **Cluster-wide fan-out** — distribute one injection spec across many namespaces
  (parity with ESO `ClusterExternalSecret`). The current model is per-pod annotation.

## Not planned

- **Tag / regex record selection.** The injector targets specific records by title or Keeper Notation,
  or a whole folder — which fits per-pod injection. Bulk tag/regex discovery is an ESO sync pattern that
  adds ambiguity to *what gets injected*; folder support already covers grouping.

## Known limitations (to wire up before GA)

- **Signal-on-refresh delivery.** The `keeper.security/signal` annotation is accepted and no longer
  crashes the sidecar, but the sidecar does not yet actually send the signal to the application
  container on refresh (cross-container signaling needs the pod's `shareProcessNamespace`). Treated as
  not-yet-implemented in the docs.
- **Helm chart values not consumed by the webhook.** `excludedNamespaces`, `defaults.*`,
  `sidecarResources`, and `metrics.port` are rendered by the chart but the webhook never reads them
  (the ConfigMap is unused). Wire these through the controller (flags/env) so the chart knobs take effect.
- **Folder → K8s-Secret injection** (`FolderRef.InjectAsK8sSecret` / `K8sSecretNamePrefix`) is parsed
  but not wired into the webhook or sidecar; either implement or remove.

## Release hardening (toward GA / 1.0)

- **Supply chain** — image signing (cosign), SBOM (syft), vulnerability scanning (govulncheck / Trivy) in CI.
- **CI-gated E2E** — run the Kind-based end-to-end suite automatically (currently manual).
- **API stability** — commit to an annotation-stability / deprecation policy before 1.0.
- **Maintainership** — add a second maintainer to reduce bus-factor before declaring GA.
