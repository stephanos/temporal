---
satisfies: [R4, R5, R9]
---
# fn-64-umpire-case-runtime.4 Implement generic DAG scheduling and Run recording

## Description
Implement the portable scheduling and data plane for R4. This task stops at deterministic
entrypoint execution, typed requests/outcomes/Slots, projections, and append-only Run recording;
Task 9 owns termination, cleanup, facade completion, and repeated-Run race guarantees.

**Size:** M
**Files:** scheduler, typed stores, request/projection, recorder, and fake-Host tests under
`tools/umpire/execution/**`
**Touches:** [tools/umpire/execution/**]

## Approach
- Schedule bounded nodes by entrypoint activation and dependencies with stable activation,
  instruction, attempt, and fan-out coordinates.
- Build typed requests and apply response Slot/Observation projections in execution/shared IR;
  Hosts receive prepared requests and return raw typed outcome data.
- Expose generic protocol/SDK failure and timeout outcomes to guards; classify missing required
  Slots, recorder failure, invariants, and global-limit breach as execution failures.
- Assign the central observation sequence and recorded monotonic elapsed coordinate. Deduplicate
  identical source events and reject conflicting source-ID reuse.
- Add the synchronized Monitor observation barrier primitive without implementing abort/cleanup.

## Investigation targets
**Required** (read before coding):
- `tools/umpire/executor/portable_executor.go:115-180` — current orchestration seam to replace
- `tools/umpire/executor/executor.go:68-150` — current execution/evaluation coupling
- `tools/umpire/runner/runner.go` — current bounded runner shape
- `tools/umpire/executor/portable_executor_test.go` — existing preflight/execution test patterns
- `.flow/memory/bug/integration/portable-execution-boundaries-must-2026-09-03.md` — invariant and
  cancellation regressions

**Optional** (reference as needed):
- `.flow/memory/bug/integration/behavior-neutral-refactors-must-not-2026-09-04.md` — keep extraction
  distinct from semantic hardening

## Key context
The server Host must not assign Slots, emit Observations, or reach into mutable Executor state.
Projection and ordering semantics remain portable execution concerns.

## Acceptance
- [ ] Fake-Host tests cover dependency order, permitted concurrency, entrypoint activation
  isolation, guarded result branching, typed request construction, immutable Slot assignment, and
  deterministic fan-out.
- [ ] Run Events contain stable source/causal coordinates, central sequence, and recorded monotonic
  elapsed values for instruction timeout and closure facts.
- [ ] Exact duplicate source events deduplicate; conflicting duplicates, missing Slots, post-close
  events, recorder/invariant/limit failure become incomplete with stable diagnostics.
- [ ] The Monitor barrier is synchronous with append and exposes no scheduling/evidence mutation.
- [ ] Execution imports neither verification nor a Temporal Host implementation.
- [ ] `go test -race -count=1 -tags test_dep ./tools/umpire/execution/...` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
