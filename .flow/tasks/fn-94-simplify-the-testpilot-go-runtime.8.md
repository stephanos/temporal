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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
