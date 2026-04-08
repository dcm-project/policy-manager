# Policy Persistence via Embedded OPA — Implementation Checkpoint

## Overview

Replaced the external OPA sidecar service with an embedded OPA Go library
(`github.com/open-policy-agent/opa/v1/rego`). Rego source code is now stored in
the database alongside policy metadata. The embedded OPA engine compiles policies
from the DB on startup and after every CRUD mutation.

**Branch:** `persist-policies`
**Base commit:** `1d4d118` (docs: add decision, spec, and test plan)
**Date:** 2026-04-02

## Test Results

All 175 specs passing across 5 test suites (race detection clean):

- **Handler/engine tests:** 11/11
- **Handler/v1alpha1 tests:** 15/15
- **OPA engine tests:** 16/16 (NEW)
- **Service tests:** 104/104
- **Store tests:** 29/29

## Changes by Topic (following spec dependency graph)

### Topic 1: Database Model

- `internal/store/model/policy.go` — Added `RegoCode string` field with GORM tag
  `gorm:"column:rego_code;type:text;not null"`. Removed `PackageName` field
  (no longer needed — engine looks up by policy ID). AutoMigrate handles schema
  migration.

### Topic 2: Store Layer

- `internal/store/policy.go` — Added `ListAll(ctx) (PolicyList, error)` to the
  `Policy` interface. Returns all policies ordered by ID ASC, empty slice when
  none exist. Updated `Update` Select to include `rego_code` in mutable fields.
  Removed `package_name` from Update Select (field removed from model).

### Topic 3: Model Converter

- `internal/service/converter.go` — `APIToDBModel` maps `api.RegoCode` →
  `db.RegoCode`. `DBToAPIModel` reads `RegoCode` from DB model (was previously
  always empty string).

### Topic 4: Embedded OPA Engine

- `internal/opa/engine.go` — NEW file. `Engine` interface with `Compile`,
  `EvaluatePolicy`, `ValidateRego`. Uses two mutexes: `compileMu` (sync.Mutex)
  serializes `Compile` calls to prevent interleaving races, `mu` (sync.RWMutex)
  protects the query map swap so evaluations remain non-blocking during
  compilation. Builds `PreparedEvalQuery` per policy, keyed by policy ID
  (package name resolved from compiled AST). Atomic compilation (failed compile
  preserves previous state). Uses `ast.CompileModulesWithOpt` with `RegoV1`
  parser options.

- `internal/opa/engine_test.go` — NEW file. 16 test cases: valid/zero/invalid
  compilation, state replacement, atomic failure, evaluation (defined, ID differs
  from package name, undefined, non-existent ID, complex input, before compile,
  namespaced packages by ID), concurrent evaluation during compile, ValidateRego
  (valid, invalid, empty).

- `internal/opa/errors.go` — Removed dead HTTP-client sentinels
  (`ErrPolicyNotFound`, `ErrOPAUnavailable`, `OPAError` struct). Renamed
  `ErrClientInternal` → `ErrEngineInternal`. Updated comments to say
  "policy engine" instead of "OPA server/client".

- `internal/opa/client.go` — REMOVED (HTTP OPA client)
- `internal/opa/client_test.go` — REMOVED (HTTP client tests)
- `internal/rego/parser.go` — REMOVED (package name extraction no longer needed)
- `internal/rego/parser_test.go` — REMOVED (parser tests)

### Topic 5: Policy Service

- `internal/service/engine_errors.go` — Renamed from `opa_errors.go`. Renamed
  `handleOPAError` → `handleEngineError`. Removed dead branches for
  `ErrPolicyNotFound` and `ErrOPAUnavailable`. Updated user-facing messages
  from "communicating with OPA" to "policy engine".

- `internal/service/policy.go` — `PolicyServiceImpl` now holds `opa.Engine`
  instead of `opa.Client`. Key changes:
  - `CreatePolicy`: validates Rego via `engine.ValidateRego`, stores RegoCode in
    DB, recompiles engine after DB write, rolls back DB on compile failure.
    No longer extracts or stores package name.
  - `GetPolicy`: reads RegoCode from DB only (no engine call needed).
  - `UpdatePolicy`: validates new Rego if changed, updates DB, recompiles engine
    if Rego changed, rolls back DB state on compile failure. No longer extracts
    or stores package name.
  - `DeletePolicy`: deletes from DB, recompiles engine.
  - Added `recompileEngine` helper (ListAll → Compile).
  - Added `CompileAll` public method (exposed via `PolicyService` interface) for
    startup compilation, reusing `recompileEngine`.

### Topic 6: Evaluation Service

- `internal/service/evaluation.go` — `evaluationService` now holds `opa.Engine`
  instead of `opa.Client`. `evaluatePolicy` calls `engine.EvaluatePolicy` with
  the policy ID (not package name).

### Topic 7: Application Wiring

- `cmd/policy-manager/main.go` — Creates `opa.NewEngine()`, creates services,
  then calls `policyService.CompileAll()` for startup compilation. Exits
  non-zero if startup compilation fails. Passes engine to both `PolicyService`
  and `EvaluationService`. Removed OPA client initialization and OPA config
  references.

### Topic 8: Configuration & Cleanup

- `internal/config/config.go` — Removed `OPAConfig` struct and its env var
  loading (`OPA_URL`, `OPA_TIMEOUT`).
- `compose.yaml` — Removed `opa` service, `policy_policies` volume, OPA
  dependency and env vars from `policy-manager` service.

### Test Updates

- `internal/service/policy_test.go` — Uses real `opa.NewEngine()` instead of
  `MockOPAClient`. Tests verify Rego stored/retrieved from DB. Added tests for
  invalid Rego rejection and Rego persistence through create/get flow.
- `internal/service/evaluation_test.go` — `mockOPAClient` → `mockEngine`
  (implements `opa.Engine`). `mockOPAClientWithCapture` →
  `mockEngineWithCapture`.
- `internal/store/policy_test.go` — Added `ListAll` tests (all policies, empty
  DB, ID ordering). Added `RegoCode` persistence tests (create, update). Updated
  `newPolicy` helper to include `RegoCode` and `PackageName`.
- `test/e2e/suite_test.go` — Removed `opaClient` and OPA URL setup.
- `test/e2e/policy_test.go` — Removed direct OPA verification in delete test
  (replaced with API-level verification). Removed `opa` and `errors` imports.

## Dependencies Added

- `github.com/open-policy-agent/opa/v1` (OPA Go library for embedded engine)

## Files Changed

```
MODIFIED:
  cmd/policy-manager/main.go
  compose.yaml
  go.mod / go.sum
  internal/config/config.go
  internal/opa/errors.go          (cleaned up — dead sentinels removed, renamed)
  internal/opa/evaluation.go      (kept — types still used)
  internal/service/converter.go
  internal/service/evaluation.go
  internal/service/evaluation_test.go
  internal/service/policy.go
  internal/service/policy_test.go
  internal/store/model/policy.go
  internal/store/policy.go
  internal/store/policy_test.go
  test/e2e/policy_test.go
  test/e2e/suite_test.go

NEW:
  internal/opa/engine.go
  internal/opa/engine_test.go
  internal/service/engine_errors.go  (renamed from opa_errors.go)

REMOVED:
  internal/opa/client.go
  internal/opa/client_test.go
  internal/rego/parser.go
  internal/rego/parser_test.go
  internal/service/opa_errors.go     (renamed to engine_errors.go)
```

## What's NOT Done (out of scope per spec)

- Incremental compilation (compile only changed modules)
- Multi-instance coordination
- Bundle API support
- Policy compilation caching to disk
- E2E tests for persistence across restarts (TC-E001–TC-E003) — requires
  docker-compose stack
