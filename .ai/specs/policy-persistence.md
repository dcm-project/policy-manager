# Spec: Policy Persistence via Embedded OPA

## 1. Overview

The policy manager currently stores policy data across two systems: metadata in
the database (PostgreSQL/SQLite) and Rego source code in an external OPA service
(via REST API). OPA's policy store is ephemeral — when OPA restarts, all Rego
code is lost, creating an inconsistent state where the service knows about
policies but cannot evaluate them.

This spec replaces the external OPA service with an embedded OPA Go library
(`github.com/open-policy-agent/opa/v1/rego`). Rego source code is stored in the
database alongside policy metadata. On startup and after every CRUD mutation,
policies are compiled from the database into an in-process OPA engine for
evaluation.

**Version scope:**

- Store Rego source code in the database
- Embedded OPA engine for policy compilation and evaluation
- Startup compilation from database
- Recompilation on policy create, update, and delete
- Concurrent-safe evaluation during recompilation
- Remove external OPA service dependency

**Out of scope:**

- Incremental compilation (compile only changed modules)
- Multi-instance coordination (shared engine state across replicas)
- Bundle API support
- Policy compilation caching to disk

**Reference documents:**

- [Decision 005: Policy Persistence](../decisions/005-policy-persistence.md)
- [Policy Engine Enhancement](https://github.com/dcm-project/enhancements/blob/1f357c1213ccfbb8638f9b5baed82ada86114c15/enhancements/policy-engine/policy-engine.md)
- [OPA Go Integration](https://www.openpolicyagent.org/docs/latest/integration/#integrating-with-the-go-api)

---

## 2. Architecture

### Before (External OPA)

```
┌──────────────────┐         ┌──────────────────┐
│  Policy Manager  │──HTTP──▶│  OPA (sidecar)   │
│                  │         │  (port 8181)      │
│  ┌────────────┐  │         │  Rego in memory   │
│  │ PostgreSQL │  │         │  (ephemeral)      │
│  │ metadata   │  │         └──────────────────┘
│  └────────────┘  │
└──────────────────┘
```

### After (Embedded OPA)

```
┌──────────────────────────────┐
│  Policy Manager              │
│                              │
│  ┌────────────┐  ┌────────┐ │
│  │ PostgreSQL │  │  OPA   │ │
│  │ metadata + │  │ engine │ │
│  │ rego_code  │  │ (lib)  │ │
│  └────────────┘  └────────┘ │
└──────────────────────────────┘
```

### Directory Structure (changes only)

```
internal/
├── opa/
│   ├── engine.go          ← NEW: embedded OPA engine
│   ├── evaluation.go      ← KEEP: EvaluationResult, PolicyDecision types
│   ├── errors.go          ← KEEP: sentinel errors
│   └── client.go          ← REMOVE: HTTP client
├── rego/                    ← REMOVE: package name parser (no longer needed)
├── store/
│   ├── model/
│   │   └── policy.go      ← MODIFY: add RegoCode field (PackageName removed)
│   └── policy.go          ← MODIFY: add ListAll, update Update select
├── service/
│   ├── converter.go       ← MODIFY: map RegoCode
│   ├── policy.go          ← MODIFY: use Engine, store Rego in DB
│   └── evaluation.go      ← MODIFY: use Engine
├── config/
│   └── config.go          ← MODIFY: remove OPAConfig
cmd/policy-manager/
└── main.go                ← MODIFY: engine init, remove OPA client
compose.yaml               ← MODIFY: remove OPA service
go.mod                     ← MODIFY: add OPA dependency
```

---

## 3. Topic Dependency Graph

| # | Topic                        | Prefix | Depends On |
|---|------------------------------|--------|------------|
| 1 | Database Model               | DB     | -          |
| 2 | Store Layer                  | STO    | 1          |
| 3 | Model Converter              | CNV    | 1          |
| 4 | Embedded OPA Engine          | ENG    | -          |
| 5 | Policy Service               | SVC    | 2, 3, 4    |
| 6 | Evaluation Service           | EVL    | 4          |
| 7 | Application Wiring           | APP    | 2, 4, 5, 6 |
| 8 | Configuration & Cleanup      | CFG    | -          |

```
Topic 1: Database Model        (independent)
Topic 4: Embedded OPA Engine    (independent)
Topic 8: Config & Cleanup       (independent)
  |         |
  +----+----+
  |    |
  v    v
Topic 2: Store Layer            (depends on 1)
Topic 3: Model Converter        (depends on 1)
  |    |    |
  +----+----+------> Topic 5: Policy Service    (depends on 2, 3, 4)
       |
       +-----------> Topic 6: Evaluation Service (depends on 4)
                          |         |
                          +----+----+
                               |
                               v
                     Topic 7: Application Wiring (depends on 2, 4, 5, 6)
```

Topics 1, 4, and 8 can be delivered in parallel. Topic 7 is the final
integration step.

---

## 4. Topic Specifications

### 4.1 Database Model

#### Overview

Add a `rego_code` column to the `policies` table to store Rego source code
alongside policy metadata. GORM's `AutoMigrate` handles the schema migration.

#### Requirements

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| REQ-DB-010 | The `Policy` model MUST include a `RegoCode` field of type `text`, not null | MUST | GORM tag: `gorm:"column:rego_code;type:text;not null"` |
| REQ-DB-020 | The database schema MUST be migrated automatically on startup via GORM `AutoMigrate` | MUST | Existing behavior, no new code needed |

#### Acceptance Criteria

##### AC-DB-010: RegoCode field in model

- **Validates:** REQ-DB-010
- **Given** the `Policy` struct in `internal/store/model/policy.go`
- **When** a policy is created with `RegoCode` set
- **Then** the value MUST be persisted in the `rego_code` column
- **And** the value MUST be retrievable via `Get`

##### AC-DB-020: Schema migration adds column

- **Validates:** REQ-DB-020
- **Given** an existing database without the `rego_code` column
- **When** the application starts and runs `AutoMigrate`
- **Then** the `rego_code` column MUST be added to the `policies` table

#### Dependencies

None — independently deliverable.

---

### 4.2 Store Layer

#### Overview

Extend the store layer to persist and retrieve Rego code. Add a `ListAll` method
for the engine's startup and recompilation loading.

#### Requirements

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| REQ-STO-010 | `Create` MUST persist `rego_code` alongside other policy fields | MUST | Already handled by `Select("*")` |
| REQ-STO-020 | `Update` MUST include `rego_code` in the mutable field set | MUST | Add to `Select(...)` list |
| REQ-STO-030 | `Get` and `List` MUST return `rego_code` in the model | MUST | Already handled by full row scan |
| REQ-STO-040 | The `Policy` interface MUST expose a `ListAll` method that returns all policies without pagination or filtering, ordered by ID | MUST | Used by engine for compilation |
| REQ-STO-050 | `ListAll` MUST return an empty slice (not nil) when no policies exist | MUST | |

#### Acceptance Criteria

##### AC-STO-010: Create persists rego_code

- **Validates:** REQ-STO-010
- **Given** a policy with `RegoCode` set to a Rego string
- **When** `store.Create` is called
- **Then** `store.Get` for the same ID MUST return the policy with matching `RegoCode`

##### AC-STO-020: Update modifies rego_code

- **Validates:** REQ-STO-020
- **Given** a policy exists with `RegoCode = "package old"`
- **When** `store.Update` is called with `RegoCode = "package new"`
- **Then** `store.Get` MUST return `RegoCode = "package new"`

##### AC-STO-030: Update without rego_code preserves existing

- **Validates:** REQ-STO-020
- **Given** a policy exists with `RegoCode = "package test"`
- **When** `store.Update` is called changing only `DisplayName`
- **Then** `store.Get` MUST return the original `RegoCode`

##### AC-STO-040: ListAll returns all policies

- **Validates:** REQ-STO-040
- **Given** 3 policies exist in the database
- **When** `store.ListAll` is called
- **Then** all 3 policies MUST be returned with `RegoCode` populated

##### AC-STO-050: ListAll returns empty slice on empty DB

- **Validates:** REQ-STO-050
- **Given** no policies exist in the database
- **When** `store.ListAll` is called
- **Then** an empty slice MUST be returned with no error

##### AC-STO-060: ListAll returns policies ordered by ID

- **Validates:** REQ-STO-040
- **Given** policies exist with IDs "c", "a", "b"
- **When** `store.ListAll` is called
- **Then** policies MUST be returned in order: "a", "b", "c"

#### Dependencies

Depends on Topic 1 (Database Model).

---

### 4.3 Model Converter

#### Overview

Update the API-to-DB and DB-to-API converters to map `RegoCode` between the
API model and the database model.

#### Requirements

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| REQ-CNV-010 | `APIToDBModel` MUST set `db.RegoCode` from `api.RegoCode` when non-nil | MUST | |
| REQ-CNV-020 | `APIToDBModel` MUST set `db.RegoCode` to empty string when `api.RegoCode` is nil | MUST | |
| REQ-CNV-030 | `DBToAPIModel` MUST set `api.RegoCode` from `db.RegoCode` | MUST | |

#### Acceptance Criteria

##### AC-CNV-010: APIToDBModel maps RegoCode

- **Validates:** REQ-CNV-010
- **Given** an API model with `RegoCode` set to `"package test"`
- **When** `APIToDBModel` is called
- **Then** the DB model MUST have `RegoCode = "package test"`

##### AC-CNV-020: APIToDBModel with nil RegoCode

- **Validates:** REQ-CNV-020
- **Given** an API model with `RegoCode = nil`
- **When** `APIToDBModel` is called
- **Then** the DB model MUST have `RegoCode = ""`

##### AC-CNV-030: DBToAPIModel maps RegoCode

- **Validates:** REQ-CNV-030
- **Given** a DB model with `RegoCode = "package test"`
- **When** `DBToAPIModel` is called
- **Then** the API model MUST have `RegoCode` pointing to `"package test"`

#### Dependencies

Depends on Topic 1 (Database Model).

---

### 4.4 Embedded OPA Engine

#### Overview

Replace the HTTP-based OPA client with an in-process OPA engine using the
`github.com/open-policy-agent/opa/v1/rego` package. The engine compiles Rego
modules and evaluates policies without network calls.

Out of scope: incremental compilation, bundle support, decision logging.

#### Requirements

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| REQ-ENG-010 | The `Engine` interface MUST expose `Compile(ctx, policies)` to load and compile all Rego modules | MUST | Replaces previous compiled state entirely |
| REQ-ENG-020 | The `Engine` interface MUST expose `EvaluatePolicy(ctx, policyID, input)` to evaluate a policy by ID | MUST | Returns `*EvaluationResult` (same as current) |
| REQ-ENG-030 | The `Engine` interface MUST expose `ValidateRego(ctx, regoCode)` to check Rego syntax without persisting | MUST | Returns `ErrInvalidRego` on failure |
| REQ-ENG-040 | `Compile` MUST return `ErrInvalidRego` with details when any module fails compilation | MUST | |
| REQ-ENG-050 | `Compile` with zero modules MUST succeed (empty engine) | MUST | |
| REQ-ENG-060 | `Compile` MUST be atomic: a failed compilation MUST NOT corrupt the previously compiled state | MUST | |
| REQ-ENG-070 | `EvaluatePolicy` MUST return `Defined: false` when the policy ID is not found or `main` rule does not exist | MUST | |
| REQ-ENG-080 | `EvaluatePolicy` MUST be safe for concurrent use from multiple goroutines | MUST | Uses `sync.RWMutex` read lock |
| REQ-ENG-090 | `Compile` MUST acquire a write lock; concurrent evaluations MUST complete with the previous compiled state | MUST | |
| REQ-ENG-100 | `EvaluatePolicy` on a fresh engine (before any `Compile`) MUST return `Defined: false` | MUST | |
| REQ-ENG-110 | `EvaluatePolicy` MUST support policies with namespaced Rego packages (e.g., `package policies.my_policy`) | MUST | Engine resolves package name from compiled AST |

#### Engine Interface

```go
type Engine interface {
    Compile(ctx context.Context, policies []PolicyModule) error
    EvaluatePolicy(ctx context.Context, policyID string, input map[string]any) (*EvaluationResult, error)
    ValidateRego(ctx context.Context, regoCode string) error
}

type PolicyModule struct {
    ID       string
    RegoCode string
}
```

#### Implementation Notes

- On `Compile`, build one `PreparedEvalQuery` per policy ID. The package name
  is resolved from the compiled AST (`compiler.Modules[policyID]`). Store in a
  `map[string]*rego.PreparedEvalQuery` keyed by policy ID.
- Compilation happens outside the write lock. The lock is acquired only to swap
  the prepared query map.
- `ValidateRego` compiles a throwaway module; the result is discarded.

#### Acceptance Criteria

##### AC-ENG-010: Compile with valid policies

- **Validates:** REQ-ENG-010
- **Given** 3 valid Rego modules
- **When** `Compile` is called
- **Then** no error MUST be returned

##### AC-ENG-020: Compile with zero policies

- **Validates:** REQ-ENG-050
- **Given** an empty policy slice
- **When** `Compile` is called
- **Then** no error MUST be returned

##### AC-ENG-030: Compile with invalid Rego

- **Validates:** REQ-ENG-040
- **Given** a module with Rego syntax errors
- **When** `Compile` is called
- **Then** `ErrInvalidRego` MUST be returned with a descriptive message

##### AC-ENG-040: Compile replaces previous state

- **Validates:** REQ-ENG-010
- **Given** policy A is compiled
- **When** `Compile` is called with only policy B
- **Then** evaluating A MUST return `Defined: false`
- **And** evaluating B MUST return a defined result

##### AC-ENG-050: Compile is atomic on failure

- **Validates:** REQ-ENG-060
- **Given** valid policies are compiled
- **When** `Compile` is called with one invalid policy
- **Then** an error MUST be returned
- **And** the previously compiled policies MUST still be evaluable

##### AC-ENG-060: Evaluate returns decision

- **Validates:** REQ-ENG-020
- **Given** a compiled policy that returns `{rejected: false, patch: {foo: "bar"}}`
- **When** `EvaluatePolicy` is called with matching input
- **Then** the result MUST contain `Defined: true` and the expected result map

##### AC-ENG-070: Evaluate with undefined result

- **Validates:** REQ-ENG-070
- **Given** a compiled policy whose `main` rule does not match input
- **When** `EvaluatePolicy` is called
- **Then** `Defined` MUST be `false`

##### AC-ENG-080: Evaluate non-existent ID

- **Validates:** REQ-ENG-070
- **Given** a compiled policy with ID `test`
- **When** `EvaluatePolicy` is called with a non-existent policy ID
- **Then** `Defined` MUST be `false`

##### AC-ENG-090: Evaluate before compile

- **Validates:** REQ-ENG-100
- **Given** a fresh engine with no `Compile` call
- **When** `EvaluatePolicy` is called
- **Then** `Defined` MUST be `false`

##### AC-ENG-100: Evaluate with namespaced package

- **Validates:** REQ-ENG-110
- **Given** a compiled policy with `package policies.my_policy`
- **When** `EvaluatePolicy` is called with the policy's ID
- **Then** the result MUST be correctly evaluated

##### AC-ENG-110: Concurrent evaluation during compile

- **Validates:** REQ-ENG-080, REQ-ENG-090
- **Given** policies are compiled
- **When** evaluations and a new `Compile` run concurrently
- **Then** no panics, races, or corrupted results MUST occur

##### AC-ENG-120: ValidateRego with valid code

- **Validates:** REQ-ENG-030
- **Given** valid Rego code
- **When** `ValidateRego` is called
- **Then** no error MUST be returned

##### AC-ENG-130: ValidateRego with invalid code

- **Validates:** REQ-ENG-030
- **Given** Rego code with syntax errors
- **When** `ValidateRego` is called
- **Then** `ErrInvalidRego` MUST be returned

##### AC-ENG-140: ValidateRego with empty code

- **Validates:** REQ-ENG-030
- **Given** an empty string
- **When** `ValidateRego` is called
- **Then** an error MUST be returned

#### Dependencies

None — independently deliverable.

---

### 4.5 Policy Service

#### Overview

Update the policy service to use the embedded engine instead of the HTTP OPA
client. Store Rego code in the database on create and update. Read Rego code
from the database on get. Recompile the engine after mutations.

#### Requirements

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| REQ-SVC-010 | `CreatePolicy` MUST validate Rego via `engine.ValidateRego` before storing | MUST | |
| REQ-SVC-020 | `CreatePolicy` MUST store `RegoCode` in the database | MUST | |
| REQ-SVC-030 | `CreatePolicy` MUST recompile the engine after successful DB write | MUST | Via `store.ListAll` → `engine.Compile` |
| REQ-SVC-040 | `CreatePolicy` MUST roll back the DB insert if engine recompilation fails | MUST | |
| REQ-SVC-050 | `GetPolicy` MUST read `RegoCode` from the database, not from OPA | MUST | No engine call needed |
| REQ-SVC-060 | `UpdatePolicy` MUST validate new Rego via `engine.ValidateRego` when `RegoCode` is in the patch | MUST | |
| REQ-SVC-070 | `UpdatePolicy` MUST store updated `RegoCode` in the database | MUST | |
| REQ-SVC-080 | `UpdatePolicy` MUST recompile the engine after successful DB write when Rego changed | MUST | |
| REQ-SVC-090 | `UpdatePolicy` MUST NOT recompile the engine when Rego is not in the patch | MUST | |
| REQ-SVC-100 | `UpdatePolicy` MUST roll back the DB update if engine recompilation fails | MUST | |
| REQ-SVC-110 | `DeletePolicy` MUST recompile the engine after successful DB delete | MUST | |
| REQ-SVC-120 | `ListPolicies` MUST return `RegoCode` in the response | MUST | Consistent with Get |

#### Acceptance Criteria

##### AC-SVC-010: Create stores rego_code in DB

- **Validates:** REQ-SVC-020
- **Given** a valid policy with Rego code
- **When** `CreatePolicy` is called
- **Then** `store.Create` MUST receive a model with `RegoCode` set

##### AC-SVC-020: Create validates Rego via engine

- **Validates:** REQ-SVC-010
- **Given** a policy with Rego code
- **When** `CreatePolicy` is called
- **Then** `engine.ValidateRego` MUST be called with the Rego code

##### AC-SVC-030: Create recompiles engine

- **Validates:** REQ-SVC-030
- **Given** a policy is created successfully in the DB
- **When** `CreatePolicy` completes the DB write
- **Then** `engine.Compile` MUST be called with all policies from `store.ListAll`

##### AC-SVC-040: Create rejects invalid Rego

- **Validates:** REQ-SVC-010
- **Given** `engine.ValidateRego` returns `ErrInvalidRego`
- **When** `CreatePolicy` is called
- **Then** a 400 error MUST be returned
- **And** `store.Create` MUST NOT be called

##### AC-SVC-050: Create rolls back on compile failure

- **Validates:** REQ-SVC-040
- **Given** `store.Create` succeeds but `engine.Compile` fails
- **When** `CreatePolicy` is called
- **Then** `store.Delete` MUST be called to roll back

##### AC-SVC-060: Get reads rego_code from DB

- **Validates:** REQ-SVC-050
- **Given** a policy exists in the DB with `RegoCode`
- **When** `GetPolicy` is called
- **Then** the response MUST include `RegoCode` from the DB
- **And** no engine method MUST be called

##### AC-SVC-070: Update with new RegoCode stores in DB

- **Validates:** REQ-SVC-070
- **Given** a patch with new `RegoCode`
- **When** `UpdatePolicy` is called
- **Then** `store.Update` MUST receive the updated `RegoCode`

##### AC-SVC-080: Update validates Rego via engine

- **Validates:** REQ-SVC-060
- **Given** a patch with new `RegoCode`
- **When** `UpdatePolicy` is called
- **Then** `engine.ValidateRego` MUST be called

##### AC-SVC-090: Update recompiles engine

- **Validates:** REQ-SVC-080
- **Given** a patch with new `RegoCode` and successful DB update
- **When** `UpdatePolicy` completes the DB write
- **Then** `engine.Compile` MUST be called

##### AC-SVC-100: Update without RegoCode preserves DB value

- **Validates:** REQ-SVC-090
- **Given** a patch with only `DisplayName`
- **When** `UpdatePolicy` is called
- **Then** `store.Update` MUST preserve the existing `RegoCode`
- **And** `engine.Compile` MUST NOT be called

##### AC-SVC-110: Update rolls back on compile failure

- **Validates:** REQ-SVC-100
- **Given** `store.Update` succeeds but `engine.Compile` fails
- **When** `UpdatePolicy` is called
- **Then** the DB MUST be rolled back to the previous state

##### AC-SVC-120: Delete recompiles engine

- **Validates:** REQ-SVC-110
- **Given** a policy is deleted from the DB
- **When** `DeletePolicy` is called
- **Then** `engine.Compile` MUST be called without the deleted policy

##### AC-SVC-130: Delete not-found returns 404

- **Validates:** REQ-SVC-110
- **Given** `store.Delete` returns `ErrPolicyNotFound`
- **When** `DeletePolicy` is called
- **Then** a 404 error MUST be returned
- **And** `engine.Compile` MUST NOT be called

#### Dependencies

Depends on Topic 2 (Store Layer), Topic 3 (Model Converter), Topic 4 (Embedded
OPA Engine).

---

### 4.6 Evaluation Service

#### Overview

Update the evaluation service to use the embedded engine instead of the HTTP OPA
client. The evaluation logic (constraint merging, patch application, rejection
handling) is unchanged.

#### Requirements

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| REQ-EVL-010 | The evaluation service MUST use `engine.EvaluatePolicy` instead of `opaClient.EvaluatePolicy` | MUST | |
| REQ-EVL-020 | All existing evaluation behavior (constraint merging, patch validation, rejection) MUST be preserved | MUST | |

#### Acceptance Criteria

##### AC-EVL-010: Evaluation uses engine

- **Validates:** REQ-EVL-010
- **Given** a mock engine
- **When** `EvaluateRequest` is called
- **Then** `engine.EvaluatePolicy` MUST be called with the correct policy ID and input

##### AC-EVL-020: Undefined result skips policy

- **Validates:** REQ-EVL-020
- **Given** `engine.EvaluatePolicy` returns `Defined: false`
- **When** `EvaluateRequest` processes the policy
- **Then** the policy MUST be skipped

##### AC-EVL-030: Existing evaluation tests pass

- **Validates:** REQ-EVL-020
- **Given** all existing evaluation test cases
- **When** run with the engine mock replacing the OPA client mock
- **Then** all tests MUST pass

#### Dependencies

Depends on Topic 4 (Embedded OPA Engine).

---

### 4.7 Application Wiring

#### Overview

Update `cmd/policy-manager/main.go` to initialize the embedded engine instead
of the HTTP OPA client. Load all policies from the database and compile them
on startup.

#### Requirements

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| REQ-APP-010 | On startup, the application MUST compile all policies via `PolicyService.CompileAll` | MUST | |
| REQ-APP-020 | `CompileAll` MUST load all policies from the database and compile them into the engine | MUST | |
| REQ-APP-030 | If startup compilation fails, the application MUST log the error and exit with a non-zero code | MUST | |
| REQ-APP-040 | The engine MUST be passed to both `PolicyService` and `EvaluationService` | MUST | |

#### Acceptance Criteria

##### AC-APP-010: Startup loads and compiles policies

- **Validates:** REQ-APP-010, REQ-APP-020
- **Given** policies exist in the database
- **When** the application starts
- **Then** all policies MUST be loaded from the DB and compiled into the engine

##### AC-APP-020: Startup failure on invalid policy

- **Validates:** REQ-APP-030
- **Given** a policy in the database has invalid Rego (e.g., corrupted data)
- **When** the application starts
- **Then** a compilation error MUST be logged
- **And** the application MUST exit with a non-zero code

#### Dependencies

Depends on Topic 2 (Store Layer), Topic 4 (Embedded OPA Engine), Topic 5
(Policy Service), Topic 6 (Evaluation Service).

---

### 4.8 Configuration & Infrastructure Cleanup

#### Overview

Remove OPA-related configuration and the external OPA container from the
deployment infrastructure.

#### Requirements

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| REQ-CFG-010 | `OPAConfig` struct and its fields (`OPA_URL`, `OPA_TIMEOUT`) MUST be removed from `config.go` | MUST | |
| REQ-CFG-020 | The OPA service MUST be removed from `compose.yaml` | MUST | |
| REQ-CFG-030 | The `policy_policies` volume MUST be removed from `compose.yaml` | MUST | |
| REQ-CFG-040 | The `opa` dependency MUST be removed from the `policy-manager` service in `compose.yaml` | MUST | |
| REQ-CFG-050 | `OPA_URL` and `OPA_TIMEOUT` environment variables MUST be removed from the `policy-manager` service in `compose.yaml` | MUST | |
| REQ-CFG-060 | The application MUST start without `OPA_URL` or `OPA_TIMEOUT` environment variables | MUST | |

#### Acceptance Criteria

##### AC-CFG-010: OPA config removed

- **Validates:** REQ-CFG-010, REQ-CFG-060
- **Given** no OPA environment variables are set
- **When** the application starts
- **Then** the application MUST start successfully without OPA configuration

##### AC-CFG-020: OPA container removed from compose

- **Validates:** REQ-CFG-020, REQ-CFG-030, REQ-CFG-040, REQ-CFG-050
- **Given** the updated `compose.yaml`
- **When** `podman compose up` is run
- **Then** only `postgres` and `policy-manager` services MUST start
- **And** no OPA-related services or volumes MUST exist

#### Dependencies

None — independently deliverable.

---

## 5. Cross-Cutting Concerns

### 5.1 Concurrency

#### Requirements

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| REQ-XC-CON-010 | The engine MUST be safe for concurrent evaluation from multiple goroutines | MUST | |
| REQ-XC-CON-020 | Recompilation MUST NOT corrupt in-flight evaluations | MUST | |
| REQ-XC-CON-030 | All engine operations MUST pass `go test -race` | MUST | |

#### Acceptance Criteria

##### AC-XC-CON-010: No races under concurrent load

- **Validates:** REQ-XC-CON-010, REQ-XC-CON-020, REQ-XC-CON-030
- **Given** the engine has compiled policies
- **When** concurrent `Compile` and `EvaluatePolicy` calls are made from multiple goroutines
- **Then** no data races MUST be detected by `go test -race`

### 5.2 Error Handling

#### Requirements

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| REQ-XC-ERR-010 | Invalid Rego MUST be reported via `ErrInvalidRego` with a descriptive message | MUST | |
| REQ-XC-ERR-020 | Compilation failures after DB writes MUST trigger a rollback | MUST | |
| REQ-XC-ERR-030 | ~~REMOVED~~ The custom Rego parser is no longer needed; package name is resolved from the compiled AST | N/A | |

---

## 6. Design Decisions

### DD-010: Embedded OPA over external service

**Decision:** Embed OPA as a Go library instead of running it as a sidecar.

**Rationale:** Eliminates the persistence problem entirely — no external state
to lose, no reconciliation needed. Simplifies deployment (single binary),
reduces evaluation latency (in-process call), and reduces failure modes (no
network errors). See [Decision 005](../decisions/005-policy-persistence.md) for
full analysis of alternatives.

### DD-020: Per-policy PreparedEvalQuery map keyed by ID

**Decision:** On `Compile`, build one `PreparedEvalQuery` per policy, keyed by
policy ID. The Rego package name is resolved from the compiled AST.

**Rationale:** `PrepareForEval` binds to a single query. Since we need
per-policy queries (`data.<pkg>.main`), we prepare one per policy at compile
time (amortized cost) rather than creating a new `rego.Rego` per evaluation call
(repeated cost). Keying by policy ID (instead of package name) means the
evaluation service can look up policies by ID directly, eliminating the need to
extract and store the Rego package name separately.

### DD-030: Remove custom Rego parser

**Decision:** Remove `internal/rego/parser.go`. Package name extraction is no
longer needed since the engine looks up policies by ID, not by package name.

**Rationale:** The engine resolves each policy's package name from the compiled
AST during `Compile`, so there is no need to extract it separately. This
eliminates the custom parser, its tests, and the `PackageName` field from the
database model.

### DD-040: Full recompilation on every mutation

**Decision:** Recompile all policies (via `ListAll` + `Compile`) on every
create, update, or delete.

**Rationale:** Simplest correct approach. Incremental compilation would be an
optimization for thousands of policies, but the expected workload is tens to low
hundreds. Full recompilation is sub-second at this scale. Premature optimization
is avoided.

### DD-050: Ginkgo + Gomega for testing

**Decision:** Use Ginkgo as the test framework with Gomega matchers, consistent
with the rest of the codebase.

**Rationale:** All existing tests use Ginkgo. Maintaining consistency reduces
cognitive overhead. Engine tests use mock interfaces; E2E tests use the full
docker-compose stack (minus OPA).

---

## 7. Requirement ID Index

| Prefix | Topic | Count |
|--------|-------|-------|
| REQ-DB-NNN | 4.1: Database Model | 2 |
| REQ-STO-NNN | 4.2: Store Layer | 5 |
| REQ-CNV-NNN | 4.3: Model Converter | 3 |
| REQ-ENG-NNN | 4.4: Embedded OPA Engine | 11 |
| REQ-SVC-NNN | 4.5: Policy Service | 12 |
| REQ-EVL-NNN | 4.6: Evaluation Service | 2 |
| REQ-APP-NNN | 4.7: Application Wiring | 4 |
| REQ-CFG-NNN | 4.8: Configuration & Cleanup | 6 |
| REQ-XC-CON-NNN | 5.1: Concurrency | 3 |
| REQ-XC-ERR-NNN | 5.2: Error Handling | 3 |
| **Total** | | **51** |
