# Run serial bounded semantic exploration with umpire-fuzz

> HTML render lens (local): open `.flow/artifacts/fn-33-run-serial-bounded-semantic-exploration/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Overview

Add one serial bounded `umpire-fuzz run` command. The Lean-owned fn-17 layer selects one candidate at
a time from one checked finite Space using fn-40's canonical PlannerPolicy surface. The command runs
that candidate through the existing runner and Run Evaluation path, admits its complete Result, and
then asks Lean for the next candidate. The terminal output reports semantic coverage and finite
exhaustion or Limit Reached honestly.

## Goal & Context
<!-- scope: business -->

The prototype needs one live proof that model-owned selection can drive bounded Execution without
moving semantic policy into Go. A single-process serial campaign is sufficient. Broader campaign
coordination and recovery infrastructure remain deferred until this loop demonstrates useful
findings and predictable cost.

## Architecture & Data Models
<!-- scope: technical -->

```text
checked Space + fn-40 PlannerPolicy + Limits
  -> fn-17 in-memory session chooses one ExperimentSpec
  -> existing runner performs one bounded Execution and cleanup
  -> existing Run Evaluation admits one complete Result
  -> Lean observes the checked admission and chooses the next candidate
  -> terminal semantic coverage plus exhausted or Limit Reached
```

The Go orchestration is deliberately shallow. It holds one process-local campaign session, one
active candidate, and one complete admitted Result at a time. The Lean bridge owns candidate order,
the requested uncovered coordinate, semantic coverage, and finite exhaustion. Go owns framing,
process lifecycle, Limit accounting, the existing runner and Run Evaluation calls, and cleanup.

No new persisted Artifact family or restart contract is introduced. Interruption returns a bounded
tooling outcome and discards the process-local session; a later invocation begins again from the
same checked inputs.

## API Contracts
<!-- scope: technical -->

- `umpire-fuzz run` accepts one checked Nexus Space binding, one retained fn-17 policy, canonical
  fn-40 PlannerPolicy input, and explicit candidate/wall-clock Limits from a closed command shape.
- The fixed Lean bridge supports `initialize`, `next`, `observe`, and `finish`. Each frame is
  canonical and carries only checked identities, Limits, one selected v2 ExperimentSpec, or one
  complete fn-18-admitted Result binding.
- `next` returns at most one candidate. Cleanup and Run Evaluation for that candidate must finish
  before `observe`; another `next` is unavailable while a candidate is outstanding.
- `observe` validates the complete Result closure and lets Lean update the selected-coordinate and
  finite-exhaustion state. Go cannot inspect, prioritize, or synthesize Model Coordinates.
- `finish` reports `exhausted|limit-reached|stopped|tooling-failure`, selected/executed/admitted
  counts, Lean-owned semantic coverage, and the first unexecuted candidate when known. This is
  terminal command output, not a persisted Artifact or restart token.
- Pinned Regressions use their existing ordinary Regression path independently of exploration
  Limits; the campaign neither installs nor owns them.

## Edge Cases & Constraints
<!-- scope: technical -->

- Candidate Limit zero, wall-clock Limit exhaustion, cancellation, bridge failure, runner failure,
  incomplete cleanup, and rejected Run Evaluation remain distinct outcomes.
- A candidate is observed exactly once only after a complete admitted Result exists. Operational
  failure cannot be relabeled as semantic coverage or finite exhaustion.
- Every candidate and Result remains bound to the same checked Space, policy, environment input,
  Definition IDs, Behavior Fingerprints, Artifact Checksum, and Limits.
- Determinism depends only on checked inputs and Limits; timestamps and progress rendering do not
  enter semantic identities.
- Only one bridge request and one Execution are active at any instant.

## Quick commands
<!-- scope: technical -->

```bash
cd model && mise exec -- lake build Temporal.Tool.ExplorationBridgeTests
mise exec -- go test ./tools/umpire/campaign/...
mise exec -- go test ./tools/umpire/cmd/umpire-fuzz/...
mise exec -- make umpire-check-regression
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** One bounded canonical Lean bridge keeps candidate selection, the requested uncovered
  coordinate, semantic coverage, and exhaustion Lean-owned while Go sees only checked bindings,
  Limits, one v2 candidate, and one complete admitted Result.
- **R2:** The command executes exactly one candidate at a time through the existing runner and Run
  Evaluation interfaces, with cleanup complete before the Result is observed or another candidate
  is requested.
- **R3:** Every candidate and Result remains bound to the same checked Space, fn-17 policy, fn-40
  PlannerPolicy, environment input, Definition IDs, Behavior Fingerprints, Artifact Checksum, and
  Limits. Crossed, stale, incomplete, or duplicate values reject before the next selection.
- **R4:** Terminal output distinguishes exhausted, Limit Reached, stopped, and tooling failure and
  reports Lean-owned semantic coverage without treating unexecuted, failed, or rejected work as
  coverage.
- **R5:** Identical checked inputs and Limits select the same serial sequence and terminal output;
  pinned Regressions remain on the ordinary Regression path outside the exploration Limit.
- **R6:** Campaign coordination beyond this one process-local serial loop, durable recovery, and
  adaptive selection are explicit deferred boundaries with no placeholder API or persisted format.

## Early proof point

Prove one candidate can cross the Lean bridge, execute through the shared runner, complete cleanup,
return as one admitted Result, and update Lean-owned coverage before adding the public command.

## Boundaries
<!-- scope: business -->

- No concurrency, worker pools, leases, shards, crash-safe campaign state, checkpoint graph, or
  resume command.
- No private runner or Run Evaluation implementation, Go-owned semantic coverage, adaptive corpus,
  prioritization, or mutation feedback.
- No new persisted Artifact family, generalized campaign service, remote staging, canary, release
  Claim Assessment, or automatic Regression installation.

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

Serial execution exposes the semantic and operational seams directly and is enough to measure
whether live exploration deserves more infrastructure.

### Implementation Tradeoffs
<!-- scope: technical -->

Discarding process-local progress on interruption repeats work, but keeps the retained prototype
within the existing Artifact, runner, Run Evaluation, and fn-17 ownership boundaries.

## References

- `.plans/UMPIRE4_ORDER.md` — retained serial fn-33 scope.
- `.flow/specs/fn-17-bounded-semantic-exploration-and.md` — Lean-owned bounded selection.
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — admitted v2 Artifacts and Results.
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — shared runner.
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — shared Run Evaluation.
- `.flow/specs/fn-40-centralize-plannerpolicy-constructors.md` — canonical PlannerPolicy surface.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Bounded Lean bridge | `.1`, `.2` | — |
| R2 | Serial runner and Run Evaluation loop | `.3`, `.6` | — |
| R3 | Complete binding and fail-closed admission | `.1`, `.3` | — |
| R4 | Honest terminal output | `.4` | — |
| R5 | Determinism and pinned independence | `.5` | — |
| R6 | Strict serial process-local boundary | `.6` | — |
