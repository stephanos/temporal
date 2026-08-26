---
satisfies: [R1, R2, R3, R5, R7]
---
# fn-14-milestone-a-pilot-baseline-and-lean.2 Measure isolated mutations, coverage, and feedback cost

## Description
### Umpire4 reconciliation (normative)

This task is retained only as historical Milestone A research design. The spec is superseded as an Umpire4 roadmap gate: do not implement it, do not use Agentworkflow evidence or `LEAN_FIRST_GO` for runtime/qualification admission, and do not add it as a dependency. Current Target, Refinement, artifact, runner, conformance, verification, and qualification specs proceed independently.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Build the provider-free source-snapshot, mutation-validity, detection, coverage, and timing runner for R1/R2/R3/R5/R7.

**Size:** M
**Files:** `tools/umpire/pilot/snapshot.go`, `tools/umpire/pilot/baseline.go`, `tools/umpire/pilot/mutation.go`, `tools/umpire/pilot/coverage.go`, `tools/umpire/pilot/timing.go`, `tools/umpire/pilot/mutation_test.go`, `tools/umpire/pilot/coverage_test.go`, `tools/umpire/pilot/testdata/mutations/**`
**Touches:** [tools/umpire/pilot/snapshot.go, tools/umpire/pilot/baseline.go, tools/umpire/pilot/mutation.go, tools/umpire/pilot/coverage.go, tools/umpire/pilot/timing.go, tools/umpire/pilot/mutation_test.go, tools/umpire/pilot/coverage_test.go, tools/umpire/pilot/testdata/mutations/**]

### Approach

- Materialize the recorded source tree in a contained temporary snapshot without Git worktrees, apply one canonical mutation, run its exact declared command, and verify the caller tree digest remains unchanged.
- Classify compile/setup/invalid outcomes separately from expected semantic detections and survived mutations; require proof that the declared semantic seam and reason were reached.
- Aggregate the fixed family-by-layer coverage matrix and warm timing samples with nearest-rank percentiles.
- Retain normalized command/output/timing/tree records and prove two provider-free reruns agree on classifications, coverage, and all non-duration inputs.
- Gate full expansion on the timeout-classification and handler-failure/caller-closure early mutations.

### Investigation targets

**Required:**
- `model/Temporal/Tool/Inspect.lean:46-77` — canonical model baseline command surface.
- `model/Temporal/Tool/InspectTests.lean:10-57` — exact result and failure checks.
- `tools/umpire/internal/generate/regression/generate_test.go:17-98` — isolated repository-shaped temp roots and fail-closed validation.
- `Makefile:1015-1030` — current focused regression generation/check contract.

### Quick command

`go test -count=1 -tags test_dep ./tools/umpire/pilot -run 'TestMutation|TestCoverage|TestTiming'`
## Acceptance

- [ ] Both early mutations are valid, detected for the exact expected semantic reason, and meet the focused feedback threshold before the remaining ten run.
- [ ] All twelve mutations use isolated non-worktree snapshots and leave the caller repository byte-identical.
- [ ] Invalid/setup/compile/timeout/survived/detected outcomes remain distinct and cannot inflate detection coverage.
- [ ] Mandatory/overall detection, family-by-layer coverage, and feedback/execution percentiles are computed exactly from retained records.
- [ ] Two provider-free reruns reproduce classifications, coverage, and non-duration decision inputs; tests use `require` and whole-value comparisons.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
