# Decision 005: Policy Persistence via Embedded OPA

**Status**: Accepted
**Date**: 2026-04-02

## Context

The policy manager stores policy data in two systems:

- **Database (PostgreSQL/SQLite)**: policy metadata (ID, display name, priority, package name, label selector, etc.)
- **OPA REST API**: Rego source code (via `PUT /v1/policies/{id}`)

OPA's policy store is ephemeral. When OPA restarts, all Rego code loaded via the REST API is lost. The database retains metadata, creating an inconsistent state: the service knows _about_ policies but cannot _evaluate_ them. OPA may restart independently of the policy manager (e.g., pod eviction, OOM kill, rolling update), so the policy manager cannot assume OPA restarts only when it does.

## Decision

**Embed OPA as a Go library instead of running it as a separate service. Store Rego code in the database. Load policies from the database into the embedded OPA engine on startup and on every CRUD mutation.**

This eliminates the persistence problem entirely: there is no external OPA service whose state can be lost. The database is the single source of truth. The embedded OPA engine is an in-process, derived cache rebuilt from the database.

Specifically:

1. Add a `rego_code` column to the `policies` table.
2. Replace the HTTP-based `opa.Client` implementation with an embedded engine using the `github.com/open-policy-agent/opa/v1/rego` Go library.
3. The embedded engine compiles all policies into a `PreparedEvalQuery` on startup and recompiles when policies change.
4. On startup, load all policies from the DB and compile them into the engine.
5. On create/update/delete, update the DB and recompile the engine.
6. Remove the OPA sidecar container from deployment infrastructure.

## Options Considered

### Option 1: Store Rego in DB + Epoch-Based Reconciliation

Add `rego_code` to the DB model. Keep OPA as an external service. On startup and when a periodic epoch check fails, iterate all policies and push them to OPA via REST API.

**Pros:**
- Minimal code change to the existing OPA HTTP client
- DB becomes the single source of truth
- No new Go dependencies

**Cons:**
- Still requires running OPA as a separate service
- Needs a reconciler with epoch detection logic (new `internal/reconciler` package)
- Periodic health check defines a maximum staleness window — policies may be unevaluable between OPA restart and next check
- OPA restarts between health checks require detection via epoch markers stored in OPA's Data API
- More failure modes: OPA down, network errors, partial sync failures
- Deployment complexity unchanged (two services + networking)

**Why not chosen:** Adds complexity (reconciler, epoch tracking, periodic sync) to work around a problem that embedding eliminates entirely.

### Option 2: Embed OPA as a Go Library (chosen)

Replace the HTTP-based OPA client with the OPA Go SDK (`github.com/open-policy-agent/opa/v1/rego`). OPA runs in-process. Policies are compiled from DB contents.

**Pros:**
- Eliminates the persistence problem entirely — no external state to lose
- No reconciler, epoch tracking, or health checks needed
- Single binary deployment — no OPA sidecar container
- Lower evaluation latency (in-process function call vs. HTTP round-trip)
- Fewer failure modes (no network errors, no OPA unavailability)
- Simpler deployment infrastructure (one fewer container, no OPA configuration)
- Rego validation via compilation replaces both the custom parser and OPA REST validation

**Cons:**
- Adds OPA as a Go dependency (significant dependency tree)
- Requires rewriting the `opa.Client` implementation (~275 lines of HTTP code → embedded engine)
- Recompilation on policy mutation adds latency to CRUD operations (typically milliseconds)
- Concurrent evaluations during recompilation need synchronization (read-write lock)
- OPA memory footprint moves into the policy manager process

**Why chosen:** Simpler overall architecture. Eliminates an entire class of problems (state synchronization between two services) rather than solving them with added complexity.

### Option 3: OPA Bundle API with a Bundle Server

Package all Rego into OPA bundles. The policy manager serves bundles; OPA polls for updates.

**Pros:**
- OPA's recommended production approach for external OPA
- Atomic bundle loading with built-in revision tracking

**Cons:**
- Major architectural change replacing all OPA REST API interactions
- Polling introduces latency between mutation and OPA pickup
- Still requires an external OPA service
- More complex than both Option 1 and Option 2

**Why not chosen:** Higher complexity than embedding, and still requires an external OPA service.

### Option 4: Rego Files on a Shared Volume

Write `.rego` files to a persistent filesystem. Configure OPA to watch the directory.

**Pros:**
- OPA natively supports loading from disk

**Cons:**
- Requires shared storage infrastructure (PersistentVolumes, NFS)
- File I/O is harder to make atomic/transactional
- Race conditions between file writes and OPA's file watcher

**Why not chosen:** Introduces infrastructure dependencies that embedding avoids.

### Option 5: OPA Disk-Based Storage Backend

Configure OPA's built-in disk storage with a persistent volume.

**Pros:**
- Zero code changes to the policy manager

**Cons:**
- Uncertain whether OPA's disk storage covers the Policy API (not just the Data API)
- Requires persistent volume infrastructure
- Less control over recovery

**Why not chosen:** Uncertain effectiveness; still requires infrastructure and an external OPA service.

## Consequences

- The `policies` table grows by one `TEXT` column per row (Rego source code)
- The OPA sidecar container is removed from `compose.yaml` and deployment manifests
- OPA configuration (`OPA_URL`, `OPA_TIMEOUT`) is removed from application config
- `GetPolicy` no longer requires a network call for Rego retrieval
- Policy evaluation latency decreases (no HTTP overhead)
- The custom Rego parser (`internal/rego/parser.go`) can be replaced by OPA's compiler for package name extraction
- The `opa.Client` interface changes: `StorePolicy`/`GetPolicy`/`DeletePolicy` are replaced by engine lifecycle methods; `EvaluatePolicy` remains
- CRUD operations include a recompilation step (milliseconds) protected by a read-write lock
- The policy manager binary size increases due to the OPA dependency
