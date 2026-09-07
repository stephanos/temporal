---
satisfies: [R3, R6, R7]
---
# fn-73-explicit-environment-binding-for.4 Teach the shared Temporal Driver symbolic binding mode

## Description
Implement the two explicit shared-Driver modes and Temporal-specific no-I/O validation (R3, R6). Symbolic worker activation and request carriers must consume the same prepared role bindings while legacy 1.0 behavior remains explicit.

**Size:** M
**Files:** `common/testing/temporaltestpilot/driver.go`, `common/testing/temporaltestpilot/sdkworker/{api,driver,registry,interpreter}.go`, `common/testing/temporaltestpilot/internal/delivery/carrier.go`, focused tests
**Touches:** [common/testing/temporaltestpilot/driver.go, common/testing/temporaltestpilot/driver_test.go, common/testing/temporaltestpilot/sdkworker/**, common/testing/temporaltestpilot/internal/delivery/**]

### Approach
- Freeze symbolic Profile bindings at construction and reject partial or mixed legacy physical namespace/task-queue/Nexus maps; retain logical WorkerRoleID and explicit transport/lifecycle inputs.
- Reuse the pure worker-definition path during Validate before registry acquisition; recomputation in Open is acceptable and avoids mutable validation caches.
- Enforce the exact StartWorkflow, GetHistory and StartNexus symbolic field/role matrix using binding IDs as well as resolved values.
- Derive worker namespace, queue registrations and named Nexus endpoint routes from prepared role resources in symbolic mode.
- Retain runtime carrier checks as defense in depth and preserve the existing explicit physical path for literal 1.0 Programs.

### Investigation targets
**Required** (read before coding):
- `common/testing/temporaltestpilot/driver.go:28-109,216-231` — options, composite Open and workflow carrier extraction
- `common/testing/temporaltestpilot/sdkworker/api.go:25-38` — SDK worker option surface
- `common/testing/temporaltestpilot/sdkworker/driver.go:47-76,184-287` — construction and pure role/queue/endpoint resolution
- `common/testing/temporaltestpilot/sdkworker/registry.go:117-141` — first worker side effects
- `common/testing/temporaltestpilot/internal/delivery/carrier.go:84-105,328-352` — existing physical carrier checks

**Optional** (reference as needed):
- `common/testing/temporaltestpilot/sdkworker/interpreter.go:79-95` — Nexus endpoint consumption
- `common/testing/temporaltestpilot/server/driver.go:152-163` — Profile copy to replace with the central snapshot

### Key context
Do not redesign worker activation lifecycle owned by fn-74 or introduce a competing error taxonomy. Opaque SDK-client namespace mismatch remains an execution failure.

## Acceptance
- [ ] Symbolic and legacy modes have unambiguous construction/Validate rules; mixed or incomplete physical maps reject locally without Open.
- [ ] Validate rejects inconsistent namespace/queue references even when they resolve to equal text, resource-bearing RPC transport roles, missing endpoint resources and unsupported role/resource combinations.
- [ ] All symbolic validation is no-I/O and zero worker registry acquisition/registration is observed on rejection.
- [ ] Open derives namespace, task queues and Nexus route names only from prepared resources in symbolic mode and never consults legacy maps as fallback.
- [ ] Literal 1.0 cases retain the explicit physical configuration path and existing runtime carrier validation.
- [ ] `go test -count=1 -tags test_dep ./common/testing/temporaltestpilot/...` passes.

## Done summary
Implemented strict symbolic and legacy Temporal worker modes. Symbolic construction freezes Profile bindings and rejects legacy resource options; Validate and Open share pure prepared-role resolution for namespace, task queue, and Nexus routes, enforce the exact StartWorkflow/GetHistory/StartNexus binding-ID matrix, and preserve the legacy 1.0 carrier path. Tests cover equal-value/different-ID rejection, unsupported endpoint roles, frozen configuration, cross-mode rejection, and zero registry side effects. Independent implementation review returned SHIP. Plan sync was skipped because `planSync.enabled` is false.
## Evidence
- Commits:
- Tests: TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./common/testing/temporaltestpilot/..., targeted symbolic validation tests with -count=10, gofmt -d on task-changed files, git diff --check, implementation review SHIP: /tmp/impl-review-receipt-fn-73-explicit-environment-binding-for.4.json
- PRs:
