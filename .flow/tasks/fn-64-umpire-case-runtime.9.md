---
satisfies: [R2, R5, R9]
---
# fn-64-umpire-case-runtime.9 Complete abort, cleanup, and reusable PreparedCase execution

## Description
Complete R5 and R9 on top of the scheduler: root-facade PreparedCase execution, authoritative per-Run
Contract Monitor creation,
unconditional stop, Host-effect drain/quarantine, cleanup, terminal precedence, and repeated-Run
isolation.

**Size:** M
**Files:** termination/facade/reuse implementation and race tests under `tools/umpire/internal/execution/**`,
`tools/umpire/{prepare,prepared_case}.go`, root/internal package documentation
**Touches:** [tools/umpire/internal/execution/**, tools/umpire/*.go, tools/umpire/README.md]

## Approach
- Add the working public `PreparedCase.Run(ctx, host)` method to task 13's prepared facade and reuse
  its private preflight. Validate live Host identity before Run creation, then
  construct a fresh Host session, stores, recorder, and Monitor from the prepared Contract per Run;
  internal factory failure is a pre-Run invariant with no target effects.
- Make Monitor observation the synchronized barrier for controller dispatch and worker activation
  reservations. Already-reserved activations remain in flight, including delayed delivery and SDK
  commands racing cancellation. Pass an Executor-bounded context to
  every callback and require cancellation cooperation without wrapping callbacks in goroutines.
- Cancel Host-owned handles and boundedly drain accepted late outcomes. Quarantine unterminated
  handles behind the Profile ceiling without Executor-owned goroutine leaks.
- Run unsuppressible cleanup with a fresh bounded context and implement the spec precedence table.
- Route post-close arrivals to bounded Host diagnostics; keep returned Run/Verdict immutable and
  release quarantine capacity when a late handle finishes.
- Stress one immutable PreparedCase across sequential and concurrent Runs.

## Investigation targets
**Required** (read before coding):
- `tools/umpire/executor/portable_executor.go:115-180` — legacy lifecycle orchestration
- `tools/umpire/runner/runner.go` — current cancellation/bounds shape
- `tools/umpire/executor/portable_executor_test.go` — current preflight tests
- `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md` — Executor, Host, and Monitor boundary; Run and Verdict;
  Abort and cleanup semantics
- `.flow/memory/bug/integration/portable-execution-boundaries-must-2026-09-03.md` —
  cancellation/invariant lessons

**Optional** (reference as needed):
- `.flow/memory/bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04.md` — typed-nil
  cases

## Key context
A conforming Host returns effect handles within context, and the internal Monitor returns when its
callback context is cancelled. Umpire cannot guarantee closure for a Host method or internal test
Monitor that violates those contracts.

## Acceptance
- [ ] The root API is exactly `PreparedCase.Run(ctx, host)`; preflight rejects nil/typed-nil/
  mismatched Host and internal MonitorFactory failure before Run creation or I/O and creates fresh
  authoritative Monitor/state for every accepted Run.
- [ ] Race tests prove Stop prevents new controller dispatch and activation reservations, cancels
  existing handles, records bounded pre-close late outcomes, and cannot suppress fresh-context
  cleanup. Coordinate with Task 6 SDK tests for commands racing activation cancellation.
- [ ] Quarantined completion and Slot publication after return leave serialized Run/Verdict
  snapshots unchanged, emit bounded Host diagnostics, and release completed quarantine capacity.
- [ ] Conforming Monitor timeout/cancellation returns incomplete/inconclusive unless a violation was
  proved; a finite late-returning Monitor that ignores cancellation proves Executor does not
  manufacture a timeout/goroutine and reports the contract violation only after return.
- [ ] Drain expiry quarantines the handle under a global ceiling; ceiling exhaustion and Host
  context violation have stable diagnostics and no unbounded Executor goroutines.
- [ ] Every terminal-precedence row is table-tested, including cleanup/Host-close independence and
  violation dominance and completed early closure with pending liveness yielding inconclusive.
- [ ] One PreparedCase drives many sequential/concurrent Runs without Slot/Event/Monitor/Host-session
  leakage under the race detector.
- [ ] `go test -race -count=1 -tags test_dep ./tools/umpire/internal/execution/...
  ./tools/umpire/...` passes.

## Done summary
Implemented reusable `PreparedCase.Run` execution with preflight identity/factory validation, fresh per-Run Monitor/session/value/recording state, synchronized abort settlement, phase-aware cancellation, bounded drain/quarantine, unsuppressible cleanup, terminal precedence, and immutable returned snapshots. Added focused lifecycle, deadline, diagnostics, terminal-table, and stateful sequential/concurrent reuse race coverage; no Temporal worker/server source was changed.

Validation passed with repeated focused normal/race tests, the exact full Umpire race acceptance suite using the physical repository TMPDIR and Clang, `make fmt-imports`, and scoped no-fix `make lint-code` including errortype.

stage: impl-review - ran [2026-09-05T07:50:09Z..2026-09-05T08:10:33Z]
## Evidence
- Commits:
- Tests: baseline: green, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=0 go test -count=10 -tags test_dep -timeout 90s ./tools/umpire/internal/execution ./tools/umpire, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=1 CC=/usr/bin/clang go test -race -count=10 -tags test_dep -timeout 180s ./tools/umpire/internal/execution ./tools/umpire, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=1 CC=/usr/bin/clang go test -race -count=1 -tags test_dep ./tools/umpire/internal/execution/... ./tools/umpire/..., make fmt-imports, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=1 CC=/usr/bin/clang make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false, impl-review codex:gpt-5.6-sol:high SHIP (round 4; reviewed staged tree f54ca8dd60d570ba43d78e636d7fcc63df26bfca; receipt /tmp/impl-review-receipt-fn-64-umpire-case-runtime.9.json)
- PRs: