# Run serial bounded semantic exploration with umpire-fuzz

> HTML render lens (local): open `.flow/artifacts/fn-33-run-resumable-semantic-exploration/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Overview

Add one serial bounded `umpire-fuzz run` command. The Lean-owned exploration layer chooses one
candidate at a time from the retained fn-17 policy, the command executes it through the existing
runner and Run Evaluation path, and the Lean authority incorporates the admitted Result before
choosing the next candidate. The command reports semantic coverage and exhaustion honestly.

## Goal & Context
<!-- scope: business -->

The prototype needs to prove that model-owned candidate selection can drive real bounded Execution
without moving semantic policy into Go. A single-process serial loop is sufficient. Concurrency,
leases, crash-safe campaign state, checkpoints, and resume are deferred until serial exploration has
demonstrated useful findings and predictable cost.

## Architecture & Data Models
<!-- scope: technical -->

```text
checked Space + exploration policy + Limits
  -> Lean chooses one ExperimentSpec
  -> existing runner performs one Execution
  -> existing Run Evaluation admits one Result
  -> Lean updates semantic coverage
  -> next candidate or exhausted
  -> final bounded report
```

The Go orchestration is deliberately shallow. It holds one in-memory session, one active candidate,
and one admitted Result at a time. The Lean bridge owns candidate selection, semantic coverage,
exhaustion, and final report meaning. Go owns framing, process lifecycle, Limit accounting, the call
to the shared runner/Run Evaluation interfaces, and cleanup.

No campaign Artifact, checkpoint graph, lease, attempt ledger, corpus, adaptive mutation protocol,
or resumable state is introduced. Interruption returns a bounded tooling outcome and discards the
in-memory session; a later invocation starts from the same checked inputs.

## API Contracts
<!-- scope: technical -->

- `umpire-fuzz run` accepts one checked catalog subject and explicit candidate/wall Limits from the
  closed command shape. It has no `resume`, checkpoint, worker-count, lease, shard, corpus, or
  arbitrary executable option.
- The fixed sibling Lean bridge supports `initialize`, `next`, `observe`, and `finish`. Each call is
  bounded, canonical, and carries only checked identifiers, Limits, the selected v2
  `ExperimentSpec`, or one complete fn-18-admitted Result.
- `next` returns at most one candidate. Go must finish cleanup and Run Evaluation for that candidate
  before calling `observe` and cannot request another candidate concurrently.
- `observe` validates the complete Result closure and lets Lean update semantic coverage. Go cannot
  inspect, prioritize, or synthesize semantic coordinates.
- `finish` reports `exhausted|limit-reached|stopped|tooling-failure`, selected/executed/result counts,
  semantic coverage from Lean, and the first unexecuted candidate when known. The report is command
  output, not a persisted Artifact or restart token.
- Pinned regressions remain outside the exploration Limit and use their existing ordinary test path.

## Edge Cases & Constraints
<!-- scope: technical -->

- Candidate Limit zero, wall Limit exhaustion, cancellation, bridge failure, runner failure,
  incomplete cleanup, and non-accepted Run Evaluation remain distinct outcomes.
- A candidate is observed exactly once only after a complete admitted Result exists. A tooling error
  cannot be relabeled as semantic coverage or exhaustion.
- Determinism is measured from the same checked inputs and Limits; timestamps and progress rendering
  do not enter model identities.
- Only one bridge request and one Execution are active. There are no goroutine worker pools, leases,
  lock files, generation graphs, checkpoint publication, or crash recovery.
- Existing comments are preserved in reused source and documentation.

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

- **R1:** One bounded canonical Lean bridge keeps candidate selection, semantic coverage, and
  exhaustion Lean-owned while Go sees only checked bindings, Limits, v2 candidates, and complete
  admitted Results.
- **R2:** The command executes exactly one candidate at a time through the existing runner and Run
  Evaluation interfaces, with cleanup complete before the Result is observed.
- **R3:** Every candidate and Result remains bound to the same checked Space, policy, environment
  input, Definition IDs, Behavior Fingerprints, Artifact Checksums, and Limits; crossed or stale
  values reject before the next selection.
- **R4:** The final report distinguishes exhausted, Limit Reached, stopped, and tooling failure and
  reports semantic coverage without treating missing or failed work as coverage.
- **R5:** Pinned regressions remain independent of the exploration Limit, and serial deterministic
  fixtures prove the same checked inputs choose the same candidate sequence and report.
- **R6:** Concurrency, leases, crash-safe campaign state, checkpoints, resume, adaptive corpora, and
  generalized multi-environment orchestration are explicit deferred boundaries with no placeholder
  API or persisted format.

## Early proof point

Prove one candidate can cross the Lean bridge, execute through the shared runner, return as a complete
admitted Result, and update Lean-owned coverage before adding the public command.

## Boundaries
<!-- scope: business -->

- No concurrency, leases, worker pools, crash-safe state, checkpoint graph, or resume command.
- No private runtime or Run Evaluation implementation, Go-owned semantic coverage, corpus,
  prioritization, or mutation feedback.
- No new persisted Artifact family, generalized campaign service, remote staging, canary, or release
  Claim Assessment.

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

Serial execution exposes the semantic and operational seams directly and is enough to measure whether
live exploration deserves more infrastructure.

### Implementation Tradeoffs
<!-- scope: technical -->

Discarding in-memory progress on interruption repeats work, but avoids committing to checkpoint and
lease semantics before the retained loop has proved useful.

## References

- `.plans/UMPIRE4_ORDER.md` — retained serial fn-33 scope.
- `.flow/specs/fn-17-bounded-semantic-exploration-and.md` — Lean-owned selection and coverage.
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — admitted v2 Artifacts and Results.
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — shared runner.
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — shared Run Evaluation.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Bounded Lean bridge | `.1`, `.2` | — |
| R2 | Serial runner/Run Evaluation loop | `.3` | — |
| R3 | Binding and fail-closed admission | `.1`, `.3` | — |
| R4 | Honest terminal report | `.4` | — |
| R5 | Determinism and pinned regressions | `.5` | — |
| R6 | Deferred concurrency/resume boundary | `.6` | — |
