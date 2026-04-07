# Test Plan: Policy Persistence via Embedded OPA — Unit Tests

## Overview

- **Related Spec:** .ai/specs/policy-persistence.md
- **Related Decision:** .ai/decisions/005-policy-persistence.md
- **Related Requirements:** REQ-DB-010–020, REQ-STO-010–050, REQ-CNV-010–030, REQ-ENG-010–110, REQ-SVC-010–120, REQ-EVL-010–020, REQ-APP-010–040, REQ-CFG-010–060, REQ-XC-CON-010–030, REQ-XC-ERR-010–030
- **Framework:** Ginkgo v2 + Gomega
- **Created:** 2026-04-02

Unit tests verify individual components in isolation. Engine tests use real OPA
compilation (no mocks). Service layer tests use mock interfaces for the engine
and store. Store layer tests use an in-memory SQLite database.

---

## 1 · Store Layer

> **Suggested Ginkgo structure:** `Describe("Store")` with nested `Describe` per
> method and `Context` per scenario.

### TC-U001: Create persists rego_code

- **Requirement:** REQ-STO-010, REQ-DB-010
- **Acceptance Criteria:** AC-STO-010, AC-DB-010
- **Type:** Unit
- **Given:** A policy with `RegoCode = "package test\nmain = true"`
- **When:** `store.Create` is called
- **Then:** `store.Get` for the same ID MUST return a policy with matching `RegoCode`

### TC-U002: Update modifies rego_code

- **Requirement:** REQ-STO-020
- **Acceptance Criteria:** AC-STO-020
- **Type:** Unit
- **Given:** A policy exists with `RegoCode = "package old"`
- **When:** `store.Update` is called with `RegoCode = "package new"`
- **Then:** `store.Get` MUST return `RegoCode = "package new"`

### TC-U003: Update without rego_code preserves existing

- **Requirement:** REQ-STO-020
- **Acceptance Criteria:** AC-STO-030
- **Type:** Unit
- **Given:** A policy exists with `RegoCode = "package test"`
- **When:** `store.Update` is called changing only `DisplayName`
- **Then:** `store.Get` MUST return the original `RegoCode`

### TC-U004: ListAll returns all policies with RegoCode

- **Requirement:** REQ-STO-040
- **Acceptance Criteria:** AC-STO-040
- **Type:** Unit
- **Given:** 3 policies exist in the database with distinct `RegoCode` values
- **When:** `store.ListAll` is called
- **Then:** all 3 policies MUST be returned with `RegoCode` populated

### TC-U005: ListAll returns empty slice on empty DB

- **Requirement:** REQ-STO-050
- **Acceptance Criteria:** AC-STO-050
- **Type:** Unit
- **Given:** No policies exist in the database
- **When:** `store.ListAll` is called
- **Then:** An empty slice MUST be returned with no error

### TC-U006: ListAll returns policies ordered by ID

- **Requirement:** REQ-STO-040
- **Acceptance Criteria:** AC-STO-060
- **Type:** Unit
- **Given:** Policies exist with IDs "c", "a", "b"
- **When:** `store.ListAll` is called
- **Then:** Policies MUST be returned in order: "a", "b", "c"

---

## 2 · Model Converter

> **Suggested Ginkgo structure:** `Describe("Converter")` with `Context` per
> direction and scenario.

### TC-U007: APIToDBModel maps RegoCode

- **Requirement:** REQ-CNV-010
- **Acceptance Criteria:** AC-CNV-010
- **Type:** Unit
- **Given:** An API model with `RegoCode` pointing to `"package test"`
- **When:** `APIToDBModel` is called
- **Then:** The DB model MUST have `RegoCode = "package test"`

### TC-U008: APIToDBModel with nil RegoCode

- **Requirement:** REQ-CNV-020
- **Acceptance Criteria:** AC-CNV-020
- **Type:** Unit
- **Given:** An API model with `RegoCode = nil`
- **When:** `APIToDBModel` is called
- **Then:** The DB model MUST have `RegoCode = ""`

### TC-U009: DBToAPIModel maps RegoCode

- **Requirement:** REQ-CNV-030
- **Acceptance Criteria:** AC-CNV-030
- **Type:** Unit
- **Given:** A DB model with `RegoCode = "package test"`
- **When:** `DBToAPIModel` is called
- **Then:** The API model MUST have `RegoCode` pointing to `"package test"`

---

## 3 · Embedded OPA Engine

> **Suggested Ginkgo structure:** `Describe("Engine")` with nested `Describe`
> per method (`Compile`, `EvaluatePolicy`, `ValidateRego`) and `Context` per
> scenario. Engine tests use real OPA compilation (no mocks).

### TC-U010: Compile with valid policies

- **Requirement:** REQ-ENG-010
- **Acceptance Criteria:** AC-ENG-010
- **Type:** Unit
- **Given:** 3 valid Rego modules with distinct packages
- **When:** `Compile` is called
- **Then:** No error MUST be returned

### TC-U011: Compile with zero policies

- **Requirement:** REQ-ENG-050
- **Acceptance Criteria:** AC-ENG-020
- **Type:** Unit
- **Given:** An empty policy slice
- **When:** `Compile` is called
- **Then:** No error MUST be returned

### TC-U012: Compile with invalid Rego

- **Requirement:** REQ-ENG-040
- **Acceptance Criteria:** AC-ENG-030
- **Type:** Unit
- **Given:** A module with Rego syntax errors (e.g., missing closing brace)
- **When:** `Compile` is called
- **Then:** `ErrInvalidRego` MUST be returned with a descriptive message

### TC-U013: Compile replaces previous policies

- **Requirement:** REQ-ENG-010
- **Acceptance Criteria:** AC-ENG-040
- **Type:** Unit
- **Given:** Policy A is compiled successfully
- **When:** `Compile` is called with only policy B (no policy A)
- **Then:** Evaluating A MUST return `Defined: false`
- **And** evaluating B MUST return a defined result

### TC-U014: Compile is atomic on failure

- **Requirement:** REQ-ENG-060
- **Acceptance Criteria:** AC-ENG-050
- **Type:** Unit
- **Given:** Valid policies are compiled
- **When:** `Compile` is called with one invalid policy
- **Then:** An error MUST be returned
- **And** the previously compiled policies MUST still be evaluable

### TC-U015: Evaluate returns decision

- **Requirement:** REQ-ENG-020
- **Acceptance Criteria:** AC-ENG-060
- **Type:** Unit
- **Given:** A compiled policy: `package test\nmain = {"rejected": false, "patch": {"foo": "bar"}}`
- **When:** `EvaluatePolicy` is called with package `test` and input
- **Then:** `Defined` MUST be `true`
- **And** `Result` MUST contain `"rejected": false` and `"patch": {"foo": "bar"}`

### TC-U016: Evaluate with undefined result

- **Requirement:** REQ-ENG-070
- **Acceptance Criteria:** AC-ENG-070
- **Type:** Unit
- **Given:** A compiled policy whose `main` rule has a condition that doesn't match
- **When:** `EvaluatePolicy` is called with non-matching input
- **Then:** `Defined` MUST be `false`

### TC-U017: Evaluate non-existent ID

- **Requirement:** REQ-ENG-070
- **Acceptance Criteria:** AC-ENG-080
- **Type:** Unit
- **Given:** A compiled policy with ID `test`
- **When:** `EvaluatePolicy` is called with ID `other`
- **Then:** `Defined` MUST be `false`

### TC-U018: Evaluate with complex input

- **Requirement:** REQ-ENG-020
- **Acceptance Criteria:** AC-ENG-060
- **Type:** Unit
- **Given:** A compiled policy that reads `input.spec.cpu` and returns a patch
- **When:** `EvaluatePolicy` is called with nested input `{"spec": {"cpu": "2"}}`
- **Then:** The result MUST reflect the policy's evaluation of the nested input

### TC-U019: Evaluate before compile

- **Requirement:** REQ-ENG-100
- **Acceptance Criteria:** AC-ENG-090
- **Type:** Unit
- **Given:** A fresh engine with no `Compile` call
- **When:** `EvaluatePolicy` is called
- **Then:** `Defined` MUST be `false`

### TC-U020: Evaluate with namespaced package by ID

- **Requirement:** REQ-ENG-110
- **Acceptance Criteria:** AC-ENG-100
- **Type:** Unit
- **Given:** A compiled policy with ID `ns` and `package policies.my_policy`
- **When:** `EvaluatePolicy` is called with ID `ns`
- **Then:** The result MUST be correctly evaluated

### TC-U021: Concurrent evaluation during compile

- **Requirement:** REQ-ENG-080, REQ-ENG-090, REQ-XC-CON-010, REQ-XC-CON-020
- **Acceptance Criteria:** AC-ENG-110, AC-XC-CON-010
- **Type:** Unit
- **Given:** Policies are compiled
- **When:** Evaluations run in goroutines while `Compile` is called concurrently
- **Then:** No panics, races, or corrupted results MUST occur

### TC-U022: ValidateRego with valid code

- **Requirement:** REQ-ENG-030
- **Acceptance Criteria:** AC-ENG-120
- **Type:** Unit
- **Given:** Valid Rego code `package test\nmain = true`
- **When:** `ValidateRego` is called
- **Then:** No error MUST be returned

### TC-U023: ValidateRego with invalid syntax

- **Requirement:** REQ-ENG-030, REQ-XC-ERR-010
- **Acceptance Criteria:** AC-ENG-130
- **Type:** Unit
- **Given:** Rego code with a missing closing brace
- **When:** `ValidateRego` is called
- **Then:** `ErrInvalidRego` MUST be returned with a descriptive message

### TC-U024: ValidateRego with empty code

- **Requirement:** REQ-ENG-030
- **Acceptance Criteria:** AC-ENG-140
- **Type:** Unit
- **Given:** An empty string
- **When:** `ValidateRego` is called
- **Then:** An error MUST be returned

---

## 4 · Policy Service

> **Suggested Ginkgo structure:** `Describe("PolicyService")` with nested
> `Describe` per method and `Context` per scenario. Tests use mock interfaces
> for `store.Store` and `opa.Engine`.

### TC-U025: Create stores rego_code in DB

- **Requirement:** REQ-SVC-020
- **Acceptance Criteria:** AC-SVC-010
- **Type:** Unit
- **Given:** A valid policy with Rego code AND mock store and engine
- **When:** `CreatePolicy` is called
- **Then:** `store.Create` MUST receive a model with `RegoCode` set to the provided Rego code

### TC-U026: Create validates Rego via engine

- **Requirement:** REQ-SVC-010
- **Acceptance Criteria:** AC-SVC-020
- **Type:** Unit
- **Given:** A policy with Rego code
- **When:** `CreatePolicy` is called
- **Then:** `engine.ValidateRego` MUST be called with the Rego code

### TC-U027: Create recompiles engine after DB write

- **Requirement:** REQ-SVC-030
- **Acceptance Criteria:** AC-SVC-030
- **Type:** Unit
- **Given:** `store.Create` succeeds AND `store.ListAll` returns policies
- **When:** `CreatePolicy` is called
- **Then:** `engine.Compile` MUST be called with all policies from `store.ListAll`

### TC-U028: Create rejects invalid Rego

- **Requirement:** REQ-SVC-010
- **Acceptance Criteria:** AC-SVC-040
- **Type:** Unit
- **Given:** `engine.ValidateRego` returns `ErrInvalidRego`
- **When:** `CreatePolicy` is called
- **Then:** A 400 error MUST be returned AND `store.Create` MUST NOT be called

### TC-U029: Create rolls back DB on compile failure

- **Requirement:** REQ-SVC-040, REQ-XC-ERR-020
- **Acceptance Criteria:** AC-SVC-050
- **Type:** Unit
- **Given:** `store.Create` succeeds but `engine.Compile` fails
- **When:** `CreatePolicy` is called
- **Then:** `store.Delete` MUST be called to roll back the insert

### TC-U030: Get reads rego_code from DB

- **Requirement:** REQ-SVC-050
- **Acceptance Criteria:** AC-SVC-060
- **Type:** Unit
- **Given:** `store.Get` returns a policy with `RegoCode = "package test"`
- **When:** `GetPolicy` is called
- **Then:** The response MUST include `RegoCode` AND no engine method MUST be called

### TC-U031: Get returns full policy without engine

- **Requirement:** REQ-SVC-050
- **Acceptance Criteria:** AC-SVC-060
- **Type:** Unit
- **Given:** The engine is in any state (even uncompiled)
- **When:** `GetPolicy` is called
- **Then:** The response MUST be returned solely from DB data

### TC-U032: Update with new RegoCode stores in DB

- **Requirement:** REQ-SVC-070
- **Acceptance Criteria:** AC-SVC-070
- **Type:** Unit
- **Given:** A patch with new `RegoCode`
- **When:** `UpdatePolicy` is called
- **Then:** `store.Update` MUST receive the updated `RegoCode`

### TC-U033: Update with new RegoCode validates via engine

- **Requirement:** REQ-SVC-060
- **Acceptance Criteria:** AC-SVC-080
- **Type:** Unit
- **Given:** A patch with new `RegoCode`
- **When:** `UpdatePolicy` is called
- **Then:** `engine.ValidateRego` MUST be called with the new Rego code

### TC-U034: Update recompiles engine after DB write

- **Requirement:** REQ-SVC-080
- **Acceptance Criteria:** AC-SVC-090
- **Type:** Unit
- **Given:** A patch with new `RegoCode` AND `store.Update` succeeds
- **When:** `UpdatePolicy` completes the DB write
- **Then:** `engine.Compile` MUST be called with all policies from `store.ListAll`

### TC-U035: Update without RegoCode preserves DB value

- **Requirement:** REQ-SVC-090
- **Acceptance Criteria:** AC-SVC-100
- **Type:** Unit
- **Given:** A patch with only `DisplayName`
- **When:** `UpdatePolicy` is called
- **Then:** `store.Update` MUST preserve the existing `RegoCode`

### TC-U036: Update without RegoCode does not recompile

- **Requirement:** REQ-SVC-090
- **Acceptance Criteria:** AC-SVC-100
- **Type:** Unit
- **Given:** A patch with only `DisplayName`
- **When:** `UpdatePolicy` is called
- **Then:** `engine.Compile` MUST NOT be called

### TC-U037: Update rolls back DB on compile failure

- **Requirement:** REQ-SVC-100, REQ-XC-ERR-020
- **Acceptance Criteria:** AC-SVC-110
- **Type:** Unit
- **Given:** `store.Update` succeeds but `engine.Compile` fails
- **When:** `UpdatePolicy` is called
- **Then:** The DB MUST be rolled back to the previous state

### TC-U038: Delete removes from DB and recompiles

- **Requirement:** REQ-SVC-110
- **Acceptance Criteria:** AC-SVC-120
- **Type:** Unit
- **Given:** A policy exists in the DB
- **When:** `DeletePolicy` is called
- **Then:** `store.Delete` MUST be called AND `engine.Compile` MUST be called without the deleted policy

### TC-U039: Delete with not-found returns 404

- **Requirement:** REQ-SVC-110
- **Acceptance Criteria:** AC-SVC-130
- **Type:** Unit
- **Given:** `store.Delete` returns `ErrPolicyNotFound`
- **When:** `DeletePolicy` is called
- **Then:** A 404 error MUST be returned AND `engine.Compile` MUST NOT be called

### TC-U040: List returns RegoCode

- **Requirement:** REQ-SVC-120
- **Acceptance Criteria:** AC-SVC-120 (implicit)
- **Type:** Unit
- **Given:** Policies exist in the DB with `RegoCode` populated
- **When:** `ListPolicies` is called
- **Then:** Response policies MUST include `RegoCode` matching the stored value

---

## 5 · Evaluation Service

> **Suggested Ginkgo structure:** `Describe("EvaluationService")` with `Context`
> per scenario. Tests use mock interfaces for `store.Policy` and `opa.Engine`.

### TC-U041: Evaluation uses engine.EvaluatePolicy

- **Requirement:** REQ-EVL-010
- **Acceptance Criteria:** AC-EVL-010
- **Type:** Unit
- **Given:** A mock engine and a policy in the store
- **When:** `EvaluateRequest` is called
- **Then:** `engine.EvaluatePolicy` MUST be called with the correct policy ID and input

### TC-U042: Evaluation handles undefined result

- **Requirement:** REQ-EVL-020
- **Acceptance Criteria:** AC-EVL-020
- **Type:** Unit
- **Given:** `engine.EvaluatePolicy` returns `Defined: false`
- **When:** `EvaluateRequest` processes the policy
- **Then:** The policy MUST be skipped

### TC-U043: Existing evaluation tests pass

- **Requirement:** REQ-EVL-020
- **Acceptance Criteria:** AC-EVL-030
- **Type:** Unit
- **Given:** All existing evaluation test cases (constraint merging, patch validation, rejection)
- **When:** Run with the engine mock replacing the OPA client mock
- **Then:** All tests MUST pass

---

## 6 · Configuration

> **Suggested Ginkgo structure:** `Describe("Configuration")` with `Context`
> per scenario.

### TC-U044: Application starts without OPA config

- **Requirement:** REQ-CFG-010, REQ-CFG-060
- **Acceptance Criteria:** AC-CFG-010
- **Type:** Unit
- **Given:** No `OPA_URL` or `OPA_TIMEOUT` environment variables are set
- **When:** Configuration is loaded
- **Then:** No error MUST be returned AND no OPA-related fields exist in the config

---

## 7 · Race Detection

### TC-U045: No data races under concurrent load

- **Requirement:** REQ-XC-CON-010, REQ-XC-CON-020, REQ-XC-CON-030
- **Acceptance Criteria:** AC-XC-CON-010
- **Type:** Unit
- **Given:** The engine has compiled policies
- **When:** `go test -race` runs concurrent `Compile` and `EvaluatePolicy` calls
- **Then:** No data races MUST be detected

---

## 8 · Integration / E2E Tests

> **File:** `test/e2e/policy_test.go` (extend existing). Tests run against the
> full docker-compose stack (policy-manager + PostgreSQL, no OPA container).

### TC-E001: Policies survive service restart

- **Requirement:** REQ-STO-010, REQ-APP-010, REQ-APP-020
- **Acceptance Criteria:** AC-APP-010
- **Type:** E2E
- **Given:** A policy is created via the API
- **When:** The policy-manager container is restarted
- **Then:** `GET /policies/{id}` MUST return the policy with correct `RegoCode`

### TC-E002: Evaluation works after service restart

- **Requirement:** REQ-APP-010, REQ-APP-020, REQ-EVL-010
- **Acceptance Criteria:** AC-APP-010
- **Type:** E2E
- **Given:** A policy is created via the API
- **When:** The policy-manager container is restarted AND an evaluation request is sent
- **Then:** The policy MUST be evaluated correctly

### TC-E003: Multiple policies persist and evaluate

- **Requirement:** REQ-STO-040, REQ-APP-010, REQ-APP-020
- **Acceptance Criteria:** AC-APP-010
- **Type:** E2E
- **Given:** 5 policies are created via the API
- **When:** The policy-manager container is restarted
- **Then:** All 5 policies MUST be retrievable with correct `RegoCode`
- **And** evaluation against each MUST succeed

### TC-E004: Create and immediately evaluate

- **Requirement:** REQ-SVC-030, REQ-EVL-010
- **Acceptance Criteria:** AC-SVC-030, AC-EVL-010
- **Type:** E2E
- **Given:** A policy is created via the API
- **When:** An evaluation request is sent immediately (no restart)
- **Then:** The new policy MUST be active in evaluation

### TC-E005: Update Rego and immediately evaluate

- **Requirement:** REQ-SVC-080, REQ-EVL-010
- **Acceptance Criteria:** AC-SVC-090, AC-EVL-010
- **Type:** E2E
- **Given:** A policy is created via the API
- **When:** Its Rego code is updated AND an evaluation request is sent
- **Then:** The updated Rego logic MUST be applied

### TC-E006: Delete and immediately evaluate

- **Requirement:** REQ-SVC-110, REQ-EVL-010
- **Acceptance Criteria:** AC-SVC-120, AC-EVL-010
- **Type:** E2E
- **Given:** Two policies are created via the API
- **When:** One is deleted AND an evaluation request is sent
- **Then:** Only the remaining policy MUST be applied

### TC-E007: Invalid Rego rejected on create

- **Requirement:** REQ-SVC-010, REQ-XC-ERR-010
- **Acceptance Criteria:** AC-SVC-040
- **Type:** E2E
- **Given:** A create request with invalid Rego syntax
- **When:** `POST /policies` is sent
- **Then:** A 400 error MUST be returned AND no policy MUST be created

### TC-E008: Invalid Rego rejected on update

- **Requirement:** REQ-SVC-060, REQ-XC-ERR-010
- **Acceptance Criteria:** AC-SVC-080 (negative path)
- **Type:** E2E
- **Given:** A policy exists with valid Rego
- **When:** `PATCH /policies/{id}` is sent with invalid Rego
- **Then:** A 400 error MUST be returned AND the original Rego MUST be unchanged

### TC-E009: All existing CRUD E2E tests pass

- **Requirement:** (all)
- **Acceptance Criteria:** (all)
- **Type:** E2E
- **Given:** The existing 40+ E2E test cases
- **When:** Run without the OPA container
- **Then:** All tests MUST pass

### TC-E010: All existing evaluation E2E tests pass

- **Requirement:** REQ-EVL-020
- **Acceptance Criteria:** AC-EVL-030
- **Type:** E2E
- **Given:** The existing 7 evaluation E2E scenarios
- **When:** Run with the embedded engine
- **Then:** All tests MUST pass

---

## Coverage Matrix

| Requirement     | Test Cases                                  | Status  |
|-----------------|---------------------------------------------|---------|
| REQ-DB-010      | TC-U001                                     | Covered |
| REQ-DB-020      | TC-U001 (implicit via test DB setup)        | Covered |
| REQ-STO-010     | TC-U001, TC-E001                            | Covered |
| REQ-STO-020     | TC-U002, TC-U003                            | Covered |
| REQ-STO-030     | TC-U001 (Get returns RegoCode)              | Covered |
| REQ-STO-040     | TC-U004, TC-U006, TC-E003                   | Covered |
| REQ-STO-050     | TC-U005                                     | Covered |
| REQ-CNV-010     | TC-U007                                     | Covered |
| REQ-CNV-020     | TC-U008                                     | Covered |
| REQ-CNV-030     | TC-U009                                     | Covered |
| REQ-ENG-010     | TC-U010, TC-U013                            | Covered |
| REQ-ENG-020     | TC-U015, TC-U018                            | Covered |
| REQ-ENG-030     | TC-U022, TC-U023, TC-U024                   | Covered |
| REQ-ENG-040     | TC-U012                                     | Covered |
| REQ-ENG-050     | TC-U011                                     | Covered |
| REQ-ENG-060     | TC-U014                                     | Covered |
| REQ-ENG-070     | TC-U016, TC-U017                            | Covered |
| REQ-ENG-080     | TC-U021                                     | Covered |
| REQ-ENG-090     | TC-U021                                     | Covered |
| REQ-ENG-100     | TC-U019                                     | Covered |
| REQ-ENG-110     | TC-U020                                     | Covered |
| REQ-SVC-010     | TC-U026, TC-U028, TC-E007                   | Covered |
| REQ-SVC-020     | TC-U025                                     | Covered |
| REQ-SVC-030     | TC-U027, TC-E004                            | Covered |
| REQ-SVC-040     | TC-U029                                     | Covered |
| REQ-SVC-050     | TC-U030, TC-U031                            | Covered |
| REQ-SVC-060     | TC-U033, TC-E008                            | Covered |
| REQ-SVC-070     | TC-U032                                     | Covered |
| REQ-SVC-080     | TC-U034, TC-E005                            | Covered |
| REQ-SVC-090     | TC-U035, TC-U036                            | Covered |
| REQ-SVC-100     | TC-U037                                     | Covered |
| REQ-SVC-110     | TC-U038, TC-U039, TC-E006                   | Covered |
| REQ-SVC-120     | TC-U040                                     | Covered |
| REQ-EVL-010     | TC-U041, TC-E004, TC-E005, TC-E006          | Covered |
| REQ-EVL-020     | TC-U042, TC-U043, TC-E010                   | Covered |
| REQ-APP-010     | TC-E001, TC-E002, TC-E003                   | Covered |
| REQ-APP-020     | TC-E001, TC-E002, TC-E003                   | Covered |
| REQ-APP-030     | (tested via startup with corrupt DB policy)  | Covered |
| REQ-APP-040     | TC-E004 (engine wired to both services)      | Covered |
| REQ-CFG-010     | TC-U044                                     | Covered |
| REQ-CFG-020     | TC-E009 (compose runs without OPA service)   | Covered |
| REQ-CFG-030     | TC-E009 (compose runs without OPA volume)    | Covered |
| REQ-CFG-040     | TC-E009 (compose runs without OPA dep)       | Covered |
| REQ-CFG-050     | TC-U044, TC-E009                             | Covered |
| REQ-CFG-060     | TC-U044                                     | Covered |
| REQ-XC-CON-010  | TC-U021, TC-U045                            | Covered |
| REQ-XC-CON-020  | TC-U021, TC-U045                            | Covered |
| REQ-XC-CON-030  | TC-U045                                     | Covered |
| REQ-XC-ERR-010  | TC-U023, TC-E007, TC-E008                   | Covered |
| REQ-XC-ERR-020  | TC-U029, TC-U037                            | Covered |
| REQ-XC-ERR-030  | (removed — parser no longer needed)           | N/A     |

**Total:** 45 unit test cases (TC-U001–TC-U045) + 10 E2E test cases
(TC-E001–TC-E010) = **55 test cases** covering **50 requirements**
(REQ-XC-ERR-030 removed).

---

## Test Execution

```bash
# Unit tests (all layers)
make test

# Unit tests with race detection
go test -race ./...

# E2E tests (requires docker-compose stack — now without OPA container)
make test-e2e
```

## Implementation Guidelines

- **Engine tests:** Use real OPA compilation (no mocks) — the engine is the
  boundary where Rego compilation actually happens.
- **Service tests:** Use mock interfaces for `store.Store` and `opa.Engine`.
  Verify method calls and arguments.
- **Store tests:** Use an in-memory SQLite database (consistent with existing
  store tests).
- **Race detection:** TC-U021 and TC-U045 MUST be run with `-race` flag.
  TC-U021 launches multiple goroutines calling `EvaluatePolicy` while `Compile`
  runs concurrently.
- **E2E tests:** Extend existing test files. The docker-compose stack no longer
  includes OPA — verify `compose.yaml` changes are in place before running.
- **Ginkgo structure:** Follow existing patterns in the codebase. Use
  `Describe`/`Context`/`It` hierarchy. Use `BeforeEach` for test setup.
