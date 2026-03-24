# Klarity — CLAUDE.md

This file gives Claude Code (and human contributors) full context about the Klarity project. Read this before making any changes.

---

## What is Klarity?

Klarity is a **Kubernetes-native Operator** that automatically detects, diagnoses, and explains Kubernetes failures using AI. It specialises deeply in two failure modes:

1. **OOMKill** — pods killed by the Linux kernel OOM killer due to exceeding memory limits
2. **CrashLoopBackOff** — pods stuck in a restart loop due to application failures

When a failure is detected, Klarity autonomously gathers context (pod logs, events, node state, resource metrics, topology), sends it to an AI model, and produces a structured, actionable diagnosis stored as a native Kubernetes object queryable via `kubectl`.

The longer term vision is to become the open source AI SRE for Kubernetes — autonomously diagnosing every class of cluster failure (Pending pods, node pressure, ImagePullBackOff, PVC binding failures, RBAC errors, network/DNS failures, and more). OOMKill and CrashLoopBackOff are the wedge.

---

## What Makes Klarity Different

- **Kubernetes-native**: results are stored as CRDs (`KlarityDiagnosis`), queryable via `kubectl get klaritydiagnoses`
- **Open source and self-hosted**: no data leaves the cluster, unlike SaaS alternatives
- **Single binary**: one Go binary, one Docker image, one Helm chart — `helm install klarity` just works
- **Expert-level diagnosis**: prompts encode real SRE operational knowledge, not generic AI output
- **Raw Anthropic Go SDK**: no Python, no LangChain, no unnecessary abstractions

---

## Architecture

Klarity is a Kubernetes Operator — a controller that encodes human SRE operational knowledge into software.

### Three-CRD model:

```
KlarityConfig   (cluster-scoped singleton)
  └── sets AI provider, model, API key secret ref, retention, concurrency limits

KlarityMonitor  (namespace-scoped, one per team/scope)
  └── defines what to watch: target namespaces, failure types, pod selector, severity

KlarityDiagnosis  (namespace-scoped, operator-created only)
  └── one per detected failure: immutable spec snapshot + mutable status with AI output
```

### Data flow:

```
User applies KlarityConfig CR (once, cluster-wide)
    ↓
User applies KlarityMonitor CR (per team/namespace)
    ↓
Event watcher starts watching configured namespaces
    ↓
Pod OOMKills or enters CrashLoopBackOff
    ↓
Event watcher detects K8s event (reason: OOMKilling / BackOff)
    ↓
Operator checks existence-based dedup:
  - Is there already a KlarityDiagnosis for this workload + container + failureType + revisionHash?
  - If yes → skip (same version, same failure already diagnosed)
  - If no → create KlarityDiagnosis CR in Pending phase
    ↓
KlarityDiagnosis controller reconciles → phase transitions to Gathering
    ↓
Collectors run in parallel, populating spec.context.sources:
  - "logs"     → previous container logs
  - "events"   → namespace events filtered by pod/node
  - "topology" → ownerRef chain, replica counts, related resources
  - "metrics"  → resource usage from metrics-server
    ↓
Phase transitions to Diagnosing → context sent to AI model
    ↓
AI returns structured diagnosis → written to status.diagnosis
    ↓
Phase transitions to Diagnosed
    ↓
Engineer runs: kubectl get klaritydiagnoses
```

No Redis, no Python worker, no separate processes. Controller-runtime provides the internal work queue. CRDs in etcd provide durability.

---

## Repository Structure

Modelled after prometheus-operator and cert-manager. Standard Go project layout.

```
klarity/
├── CLAUDE.md
├── README.md
├── LICENSE                                ← Apache 2.0
├── CHANGELOG.md
├── CONTRIBUTING.md
├── SECURITY.md
├── Dockerfile
├── Taskfile.yml
├── go.mod
├── go.sum
├── .gitignore
├── .github/
│   └── workflows/                         ← CI pipelines
├── cmd/
│   └── operator/
│       └── main.go                        ← entrypoint, manager setup
├── api/
│   └── v1alpha1/
│       ├── klarityconfig_types.go         ← KlarityConfig CRD spec (cluster-scoped singleton)
│       ├── klaritymonitor_types.go        ← KlarityMonitor CRD spec (namespace-scoped)
│       ├── klaritydiagnosis_types.go      ← KlarityDiagnosis CRD spec (operator-created)
│       ├── groupversion_info.go           ← registers API group klarity.io/v1alpha1
│       └── zz_generated.deepcopy.go      ← auto-generated, do not edit
├── internal/
│   ├── controller/
│   │   ├── klarityconfig_controller.go
│   │   ├── klaritymonitor_controller.go
│   │   └── klaritydiagnosis_controller.go
│   ├── watcher/
│   │   └── event_watcher.go              ← watches native K8s events
│   ├── collector/
│   │   ├── logs.go
│   │   ├── events.go
│   │   ├── topology.go
│   │   └── metrics.go
│   └── diagnosis/
│       ├── engine.go                     ← Anthropic Go SDK agent loop
│       ├── tools.go                      ← kubectl tool implementations
│       └── prompts.go                    ← expert SRE system prompts
├── config/
│   ├── crd/                              ← generated, do not hand-edit
│   └── rbac/
├── helm/
│   └── klarity/
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
├── hack/
│   └── update-codegen.sh                 ← runs controller-gen
├── test/
│   └── e2e/
└── examples/
    ├── klarityconfig.yaml
    ├── klaritymonitor.yaml
    └── oomkill-test.yaml
```

Key conventions:
- `internal/` packages cannot be imported externally — all business logic lives here
- `api/` contains only type definitions — no logic
- `config/` is generated, never hand-edited
- `hack/` contains codegen scripts, mirrors kubernetes/kubernetes convention
- `examples/` is what users copy-paste to get started

---

## Technology Decisions

| Decision | Choice | Reason |
|---|---|---|
| Language | Go only | Single binary, K8s ecosystem native |
| Operator framework | controller-runtime | Industry standard (ArgoCD, cert-manager, Prometheus Operator) |
| AI SDK | Raw Anthropic Go SDK | No abstraction, full control, debuggable |
| AI framework | None | 5 tools + ~150 line loop = no framework needed |
| Queue | None | CRDs in etcd + controller-runtime work queue |
| Deduplication | Existence-based (revisionHash) | No time windows to tune; naturally handles rollouts |
| Build tool | Taskfile | Simpler than Makefile |
| License | Apache 2.0 | K8s ecosystem standard, CNCF compatible |
| Deployment | Helm | Standard for K8s operators |

---

## CRD Specifications

### KlarityConfig — cluster-wide singleton, must be named `"klarity"`

```yaml
apiVersion: klarity.io/v1alpha1
kind: KlarityConfig
metadata:
  name: klarity
spec:
  ai:
    provider: anthropic
    model: claude-opus-4-6
    apiKeySecretRef:
      name: klarity-secrets
      key: anthropic-api-key
  diagnosisRetention: "72h"
  maxConcurrentDiagnoses: 5
status:
  active: true
  connectedMonitors: 3
  lastHealthCheck: "2026-03-23T10:00:00Z"
```

### KlarityMonitor — namespace-scoped, one per team/scope

```yaml
apiVersion: klarity.io/v1alpha1
kind: KlarityMonitor
metadata:
  name: payments-monitor
  namespace: payments
spec:
  targetNamespaces:
    - payments
    - payments-staging
  failureTypes:
    - OOMKill
    - CrashLoopBackOff
  selector:
    matchLabels:
      app: payments-api
  severity: critical
  enabled: true
status:
  phase: Active
  watchedPods: 12
  diagnosesCreated: 4
  lastFailureDetected: "2026-03-23T09:45:00Z"
```

### KlarityDiagnosis — auto-created by Operator, never manually

```yaml
apiVersion: klarity.io/v1alpha1
kind: KlarityDiagnosis
metadata:
  name: oomkill-payments-api-7b4f9-1711188000
  namespace: payments
  labels:
    klarity.io/monitor: payments-monitor
    klarity.io/failure-type: OOMKill
    klarity.io/owner-kind: Deployment
    klarity.io/owner-name: payments-api
    klarity.io/container: api
    klarity.io/revision-hash: 7b4f9c8d6
  annotations:
    klarity.io/retry: "false"
spec:
  failureType: OOMKill
  podName: payments-api-7b4f9c8d6-xkp2m
  containerName: api
  namespace: payments
  nodeName: k3d-klarity-agent-0
  ownerRef:
    kind: Deployment
    name: payments-api
  revisionHash: 7b4f9c8d6
  monitorRef:
    name: payments-monitor
    namespace: payments
  detectedAt: "2026-03-23T09:45:00Z"
  context:
    restartCount: 3
    exitCode: 137
    resources:
      requests: {"cpu": "250m", "memory": "256Mi"}
      limits: {"cpu": "500m", "memory": "512Mi"}
    sources:
      - name: logs
        data: "..."
      - name: events
        data: "..."
      - name: topology
        data: "..."
      - name: metrics
        data: "..."
status:
  phase: Diagnosed          # Pending | Gathering | Diagnosing | Diagnosed | Error
  diagnosedAt: "2026-03-23T09:45:47Z"
  retryCount: 0
  diagnosis:
    summary: "OOMKill due to memory leak in database connection pool"
    rootCause: "Memory grew linearly from 180Mi to 512Mi over 4 hours..."
    category: application   # application | infrastructure | configuration | dependency
    confidence: 0.92
    recommendations:
      - action: "Increase memory limit to 768Mi"
        type: resource
        priority: immediate
      - action: "Cap connection pool at 25 connections"
        type: code
        priority: short-term
    affectedResources:
      - kind: Deployment
        name: payments-api
        namespace: payments
```

---

## RBAC Requirements

```yaml
- apiGroups: [""]
  resources: ["pods", "events", "nodes", "namespaces"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods/log"]
  verbs: ["get"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods", "nodes"]
  verbs: ["get", "list"]
- apiGroups: ["klarity.io"]
  resources: ["klarityconfigs", "klaritymonitors", "klaritydiagnoses"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
- apiGroups: ["klarity.io"]
  resources: ["klaritydiagnoses/status"]
  verbs: ["get", "update", "patch"]
```

No write access to core K8s resources. Klarity is read-only on the cluster. Diagnosis only, no auto-remediation in v1.

---

## Code Style and Conventions

- Standard Go formatting (`gofmt`, `goimports`)
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Structured logging with `slog` (Go 1.21+)
- Context propagation everywhere — every function that does I/O takes `context.Context`
- No global state — everything injected via struct fields
- Interfaces for testability — `DiagnosisEngine`, `Collector`, `KubectlRunner` are interfaces
- Table-driven tests

---

## What to Build Next (in order)

1. `cmd/operator/main.go` — manager setup with controller-runtime, register all three CRDs
2. `internal/controller/klarityconfig_controller.go` — validate config, initialise AI client
3. `internal/controller/klaritymonitor_controller.go` — configure event watcher per monitor
4. `internal/watcher/event_watcher.go` — native K8s event watcher (OOMKilling / BackOff reasons)
5. `internal/collector/logs.go` — previous container log collector
6. `internal/collector/events.go` — K8s events collector
7. `internal/collector/topology.go` — ownerRef chain + replica state collector
8. `internal/collector/metrics.go` — metrics-server resource usage collector
9. `internal/diagnosis/prompts.go` — expert SRE system prompts
10. `internal/diagnosis/tools.go` — kubectl tool implementations
11. `internal/diagnosis/engine.go` — Claude agent loop
12. `internal/controller/klaritydiagnosis_controller.go` — drives full diagnosis lifecycle
13. `config/rbac/` — RBAC manifests
14. `helm/klarity/` — Helm chart
15. `examples/` — sample YAMLs
16. End to end test with real OOMKill on k3d

---

## What NOT to Do

- Do not add Python to this repo
- Do not add Redis or any external queue
- Do not use LangChain, LangGraph, or Claude Agent SDK
- Do not add auto-remediation in v1 (diagnosis and recommendations only)
- Do not create a web UI in v1 (kubectl is the UI)
- Do not add telemetry or analytics that sends data outside the cluster
- Do not use `panic()` in production code paths
- Do not store the Anthropic API key anywhere except a Kubernetes Secret reference
- Do not hand-edit files in `config/crd/` — generated by controller-gen
- Do not hand-edit `api/v1alpha1/zz_generated.deepcopy.go` — auto-generated
