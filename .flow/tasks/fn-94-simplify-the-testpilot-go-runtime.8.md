---
satisfies: [R3]
---
# fn-94-simplify-the-testpilot-go-runtime.8 Driver helper package and standard-library replacements

## Description
Lane B for the Drivers: one new helper package under `temporal/internal` for the primitives the Drivers copy, and stdlib replacements for the hand-written `slices`/`maps`/`cmp` equivalents.

**Size:** M
**Files:** new `common/testing/testpilot/temporal/internal/<helper>/` package (name chosen by the task, for example `runtime`), `temporal/internal/delivery/{ledger,carrier}.go`, `temporal/server/{driver,session}.go`, `temporal/worker/{driver,registry,reservation,routing,sdk,values}.go`, `temporal/driver.go`, `temporal/profile.go`, `common/testing/testpilot/internal/execution/evidence.go` (method-path builder only if it can import the helper without a cycle)
**Touches:** [common/testing/testpilot/temporal/internal/runtime/**, common/testing/testpilot/temporal/internal/delivery/ledger.go, common/testing/testpilot/temporal/internal/delivery/carrier.go, common/testing/testpilot/temporal/server/driver.go, common/testing/testpilot/temporal/server/session.go, common/testing/testpilot/temporal/worker/**, common/testing/testpilot/temporal/driver.go, common/testing/testpilot/temporal/profile.go]

### Approach
- Move to the helper: `nilValue` (`ledger.go:777`, `server/driver.go:242`, `worker/driver.go:553` — unify on the version that handles `UnsafePointer`), `contextError` (`ledger.go:790`, `server/driver.go:254`; takes the caller's sentinel), the context-aware mutex (`ledgerMutex` `ledger.go:797`, `hostMutex` `server/driver.go:263`; the worker's `contextMutex` `registry.go:71` keeps its nil-context rejection, via an option or a thin wrapper), the effect-result clone (`ledger.go:774`, `server/session.go:242`, `worker/reservation.go:152`), `nexusHeaderBytes` (`delivery/carrier.go:422`, `worker/routing.go:459`), the method-path builder (`delivery/carrier.go:430`, `server/session.go:85,125`), the StartWorkflow path constant (`delivery/carrier.go:19`, `worker/driver.go:21`, `temporal/profile.go:79`), `hasWorkerEntrypoint` (`temporal/driver.go:119`, `worker/driver.go:171`).
- Stdlib: `sessionIn`/`removeSessionFrom` (`routing.go:440,449`) → `slices.Contains`/`DeleteFunc` (use the returned slice); `slicesContains` (`sdk.go:272`); `setKeys`/`nexusSetKeys` (`worker/driver.go:529,537`) → `slices.Sorted(maps.Keys(...))` if order is observable; `firstError` (`registry.go:295`) → `cmp.Or` only if its arguments are side-effect free. Keep nil-vs-empty results as they are.
- Preserve comments on moved code.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/internal/delivery/ledger.go:770-820`
- `common/testing/testpilot/temporal/server/driver.go:240-290`
- `common/testing/testpilot/temporal/worker/registry.go:60-100,290-300`
- `common/testing/testpilot/temporal/worker/driver.go:520-560`

### Quick commands
```sh
go test -race -tags test_dep ./common/testing/testpilot/temporal/...
make lint-code-fast
```

## Acceptance
- [ ] Each listed primitive has one definition; each package still returns its own sentinel errors.
- [ ] No hand-written `slices`/`maps`/`cmp` equivalent remains in the Drivers.
- [ ] Route golden unchanged; `-race` tests and lint pass.


## Done summary
New package `common/testing/testpilot/temporal/internal/primitive` holds the one definition of `NilValue` (the UnsafePointer-aware version), `ContextError` and the context-aware `Mutex` (both take the caller's sentinel), `CloneEffectResult`, `NexusHeaderBytes`, `MethodPath`, `StartWorkflowPath` and `HasWorkerEntrypoint`. delivery, server, worker and the composite Driver dropped their copies, including the worker's `contextMutex`, `cloneOutcome`/`values.go` and delivery_test's clone. The worker's hand-written helpers became `slices.Contains`, `slices.DeleteFunc`, `slices.AppendSeq(make(...), maps.Keys(...))` (this keeps non-nil empty results, pinned by `TestNexusCandidatesWithoutRouteIsEmptyNotNil`) and `cmp.Or`.

Deviations: the package is named `primitive`, not `runtime`, because revive rejects a package that shadows stdlib `runtime`. `server/handle.go` and `delivery/delivery_test.go` were edited outside the declared Touches because they call the removed copies. `cmp.Or[error]` trips staticcheck SA4023 (a false positive), so it carries two reasoned nolints. `execution/evidence.go` cannot import `temporal/internal` because of Go's internal-package rule, so it is unchanged. The route and catalog goldens are unchanged and pass.

stage: impl-review - ran [round 1 fan-out NEEDS_WORK (2 findings fixed)..round 2 SHIP]
## Evidence
- Commits: 5ac32da7360c5ca29a1914f68573e502831c4494, fc979ab84168c244cd262c324a911d1423af3cbe
- Tests: baseline: green (go test -race -tags test_dep ./common/testing/testpilot/temporal/...), go test -race -tags test_dep ./common/testing/testpilot/temporal/..., make lint-code-fast GOLANGCI_LINT_BASE_REV=944c6a4d4b5927deee64da32428789be28e7ad10
- PRs: