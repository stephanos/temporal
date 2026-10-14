---
satisfies: [R2, R8, R10]
---
# fn-94-simplify-the-testpilot-go-runtime.17 Closing gates, deadcode sweep, measurement and docs

## Description
The finalization task: the full gate run, the `deadcode` check R2 names, the R10 measurement against fn-94.2's baseline, and the doc sweep the lanes deferred.

**Size:** M
**Files:** READMEs under `common/testing/testpilot/**` that still name removed or renamed declarations (root `README.md:27-29` aliases; `internal/verification/README.md` if touched by removals; `temporal/README.md:14-21`), `.plans/UMPIRE4_ORDER.md` (fn-70's resume note about `runCase`; fn-94's entry), any stragglers `deadcode` reports
**Touches:** [common/testing/testpilot/**/README.md, tests/testcore/testpilot/README.md, .plans/UMPIRE4_ORDER.md, common/testing/testpilot/**, tests/**]

### Approach
- Run `go run golang.org/x/tools/cmd/deadcode@v0.48.0 -test -tags 'test_dep integration' ./common/testing/testpilot/... ./tests/... ./tools/...` and filter to Testpilot packages; each reported function is deleted or given a production caller, never suppressed. Confirm candidates with `-whylive`.
- Run every R8 gate: `make umpire-check-case-runtime-conformance`, `umpire-check-testpilot-protocol`, `umpire-check-testpilot-authoring`, `umpire-check-retired-vocabulary`, `go test -race -tags test_dep ./common/testing/testpilot/...`, `make umpire-check-regression` (live identity count equals fn-94.2's), `make lint-code-fast`. Report anything not run with the reason.
- Measure with the spec's four commands; report each against fn-94.2's baseline and the R10 floors, with the reason for any miss.
- Docs: sweep the Testpilot READMEs for removed names; update fn-70's resume note in `.plans/UMPIRE4_ORDER.md` (it cites `runCase`). Do not edit `.plans/index.json` unless fn-94's dependencies changed.

### Quick commands
```sh
go run golang.org/x/tools/cmd/deadcode@v0.48.0 -test -tags 'test_dep integration' ./common/testing/testpilot/... ./tests/... ./tools/... | grep testpilot
make umpire-check-regression
make lint-code-fast
```

### Carried from fn-94.16 (2026-09-28) — must fix before the closing gates
- `TestTestpilotOwnsCaseProtocolAndRuntime` (tools/umpire/regression) fails: after fn-94.8–.12 the Temporal Drivers import `common/testing/testpilot/internal/execution` (`ProgramCeiling`) and `internal/ir` (`CheckCeilings`, `Invalid`, …), which the layering boundary forbids. Keep the boundary (do not loosen the test): expose what the Drivers need through the public `testpilot` facade (e.g. `testpilot.ProgramCeiling()`, `testpilot.CheckLimits(...)`) or move the shared pieces into the package the boundary already allows the Drivers to import, and re-point the Drivers. Behavior, rejection messages, goldens and identities stay unchanged.

## Acceptance
- [ ] `deadcode` reports no unreachable function in Testpilot packages.
- [ ] Every R8 gate passes, or is reported not run with the reason; live identity count unchanged.
- [ ] The receipt reports the four measurements against the baseline and the R10 floors.
- [ ] No README names a removed declaration; fn-70's resume note is current.


## Done summary
The Temporal Drivers no longer import `internal/ir` or `internal/execution`. A new facade function, `testpilot.WithinProgramCeiling(limits)`, wraps `ir.CheckCeilings(limits, execution.ProgramCeiling(), …)`, and the server's `validProfile` and the worker's `validWorkerProfile` call it. Behavior, messages, goldens and identities are unchanged, and `TestTestpilotOwnsCaseProtocolAndRuntime` passes again. `deadcode` reports no Testpilot function. The server Driver README names the new check. In `.plans/UMPIRE4_ORDER.md`, fn-70's resume note now cites `runCapturedCase` (fn-94 removed `runCase`), and fn-94 moved from the delivery queue to the delivered list, the only queue entry edited. No other README names a removed declaration: each backticked identifier was checked against the Go, proto and Lean sources.

R9 outcome, from fn-94.16: only the model-value reference arm was removed. `evidence_field_id` and `correlated_capture` stay, because Lean's correlated Producer emits them.

R10, against fn-94.2's baseline at `ea93a2bc8b`:

| Part | Baseline | Now | Change | Floor |
| --- | --- | --- | --- | --- |
| Production | 19,349 | 19,375 | +26 (+0.1%) | at least 6% smaller (18,188): missed |
| Tests | 20,572 | 20,337 | -235 (-1.1%) | at least 5% smaller (19,543): missed |
| Live tests | 2,399 | 2,369 | -30 | none |
| `.proto` | 1,414 | 1,411 | -3 (the removed arm) | none |

Why both floors were missed:
- Production code lost 584 lines across the lanes. `internal/testsupport`, 610 lines of shared test fakes and fixtures in non-`_test.go` files, is counted as production by the measurement command, so the net is +26. Counted as test code, production would be 18,765 (-3.0%).
- Test code lost about 1,000 duplicated lines to the shared helpers (fn-94.14/.15). About 760 lines of new pinning tests came back: identity goldens, R1 correlated-field tests, rejection-order pins, and rejection tests for the checks that stay.
- The spec's own estimate of about 2,100 removable lines was already close to the floors. D1 kept two of the three arms (R9) and D2 kept the wrapper structs.

Gates:
- `umpire-check-regression`: every prerequisite and every Go test step passed, with 45 live identities passing and 0 failing. The Lean tail step is INCONCLUSIVE. It builds fn-88's uncommitted `SearchDifferential.lean`, and a concurrent session terminated that build (exit 143).
- One earlier live run failed `TestTestpilotNexusCallerScheduleToStartTimeout/chasm` once: the handler ran and "completed without a reply" when its worker should already have been stopped. It passed with `-count=3` and in two later full runs. It looks like an intermittent outage race.

baseline: none recorded as a gate run before the edits. The boundary test was red on arrival, carried from fn-94.16.
Gate receipt: not written, because other sessions keep the worktree dirty.

stage: impl-review - ran [codex fan-out rid 1ebce22801d14931aff985092eaea31d: correctness SHIP, contracts NEEDS_WORK (order note overstated the identity bytes) -> fixed -> re-review SHIP]
## Evidence
- Commits: c9c72c50ec63d173d017d28ff629041690e3e118, 2170b03b485281d76e4be60ea9b688f5928d0d87, 9a22b11a8e825ce3a9a07f815bf08faf0d9a8552, d998457fc71e09863f139eaf0797adfdc023d219
- Tests: go test -count=1 -tags test_dep -run 'TestTestpilotOwnsCaseProtocolAndRuntime|TestTestpilotDependencyBoundary' ./tools/umpire/regression/ (green; red before the fix), go run golang.org/x/tools/cmd/deadcode@v0.48.0 -test -tags 'test_dep integration' ./common/testing/testpilot/... ./tests/... ./tools/... | grep testpilot (empty), go test -race -count=1 -tags test_dep ./common/testing/testpilot/... (green), make umpire-check-case-runtime-conformance (green, no fixture diff), make umpire-check-testpilot-protocol (green), make umpire-check-testpilot-authoring (green), make umpire-check-retired-vocabulary (green), make lint-code-fast (0 issues), make umpire-check-regression: every prerequisite and every Go test step green, live identities 45 passing / 0 failing; final Lean step (lake build Temporal UmpireTests TemporalModelTests ... and the umpire-inspect checks) INCONCLUSIVE: it compiles fn-88's uncommitted model/TemporalModelTests/SearchDifferential.lean, whose build a concurrent session terminated (exit 143), go test -v -count=1 -timeout 30m -tags 'test_dep integration' ./tests -run '^TestTestpilot' (45 passing, 0 failing; one earlier run failed TestTestpilotNexusCallerScheduleToStartTimeout/chasm once, intermittent, passed -count=3 in isolation and in two later full runs)
- PRs: