---
satisfies: [R5, R9]
---
# fn-64-umpire-case-runtime.9 Complete abort, cleanup, and reusable PreparedCase execution

## Description
Complete R5 and R9 on top of the scheduler: public PreparedCase execution, per-Run Monitor creation,
unconditional stop, Host-effect drain/quarantine, cleanup, terminal precedence, and repeated-Run
isolation.

**Size:** M
**Files:** termination/facade/reuse implementation and race tests under `tools/umpire/execution/**`,
`tools/umpire/{prepare,prepared_case}.go`, public/execution package documentation
**Touches:** [tools/umpire/execution/**, tools/umpire/*.go, tools/umpire/README.md]

## Approach
- Validate live Host identity and MonitorFactory typed nil before Run creation; construct fresh Host
  session, stores, recorder, and Monitor per Run.
- Make Monitor observation the synchronized dispatch barrier. Pass an Executor-bounded context to
  every callback and require cancellation cooperation without wrapping callbacks in goroutines.
- Cancel Host-owned handles and boundedly drain accepted late outcomes. Quarantine unterminated
  handles behind the Profile ceiling without Executor-owned goroutine leaks.
- Run unsuppressible cleanup with a fresh bounded context and implement the spec precedence table.
- Stress one immutable PreparedCase across sequential and concurrent Runs.

## Investigation targets
**Required** (read before coding):
- `tools/umpire/executor/portable_executor.go:115-180` — legacy lifecycle orchestration
- `tools/umpire/runner/runner.go` — current cancellation/bounds shape
- `tools/umpire/executor/portable_executor_test.go` — current preflight tests
- `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md:284-390` — corrected Host/Monitor/precedence contract
- `.flow/memory/bug/integration/portable-execution-boundaries-must-2026-09-03.md` —
  cancellation/invariant lessons

**Optional** (reference as needed):
- `.flow/memory/bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04.md` — typed-nil
  cases

## Key context
A conforming Host returns effect handles within context, and a conforming Monitor returns when its
callback context is cancelled. Umpire cannot guarantee closure for a Host method or caller-supplied
Monitor that violates those contracts.

## Acceptance
- [ ] Run preflight rejects nil/typed-nil/mismatched Host and MonitorFactory before Run creation or
  I/O and creates fresh Monitor/state for every accepted Run.
- [ ] Race tests prove Stop prevents new ordinary dispatch, cancels handles, records bounded late
  outcomes, and cannot suppress fresh-context cleanup.
- [ ] Conforming Monitor timeout/cancellation returns incomplete/inconclusive unless a violation was
  proved; a finite late-returning Monitor that ignores cancellation proves Executor does not
  manufacture a timeout/goroutine and reports the contract violation only after return.
- [ ] Drain expiry quarantines the handle under a global ceiling; ceiling exhaustion and Host
  context violation have stable diagnostics and no unbounded Executor goroutines.
- [ ] Every terminal-precedence row is table-tested, including cleanup/Host-close independence and
  violation dominance.
- [ ] One PreparedCase drives many sequential/concurrent Runs without Slot/Event/Monitor/Host-session
  leakage under the race detector.
- [ ] `go test -race -count=1 -tags test_dep ./tools/umpire/execution/... ./tools/umpire/...` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
