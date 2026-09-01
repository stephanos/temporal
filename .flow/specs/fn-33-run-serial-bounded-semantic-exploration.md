# Run serial bounded semantic exploration with umpire-fuzz

> HTML render lens (local): open `.flow/artifacts/fn-33-run-serial-bounded-semantic-exploration/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Overview

Add one serial bounded `umpire-fuzz run` command over the exact two-choice caller-closure
duplicate-delivery negative-control Space. The Lean-owned fn-17 layer selects one candidate at a
time using fn-40's canonical PlannerPolicy surface. The command runs that candidate through the
existing caller-closure runner and Run Evaluation path, admits its complete Result, and then asks
Lean for the next candidate. The terminal output reports semantic coverage and finite exhaustion or
Limit Reached honestly.

## Goal & Context
<!-- scope: business -->

The prototype needs one live proof that model-owned selection can drive bounded Execution without
moving semantic policy into Go. A single-process serial campaign is sufficient. Broader campaign
coordination and recovery infrastructure remain deferred until this loop demonstrates useful
findings and predictable cost.

## Architecture & Data Models
<!-- scope: technical -->

```text
checked CallerClosureFault Space + fn-40 PlannerPolicy + Limits
  -> fn-17 in-memory session chooses one ExperimentSpec
  -> existing runner performs one bounded Execution and cleanup
  -> existing Run Evaluation admits one complete Result
  -> Lean observes the checked admission and chooses the next candidate
  -> terminal semantic coverage plus exhausted or Limit Reached
```

The Go orchestration is deliberately shallow. It holds one process-local campaign session, one
active candidate, and one complete admitted Result at a time. The Lean bridge owns candidate order,
the requested uncovered coordinate, semantic coverage, and finite exhaustion. Go owns framing,
process lifecycle, Limit accounting, the existing runner and Run Evaluation calls, and cleanup. The
only admitted Space is
`Temporal.Feature.Nexus.Experimental.CallerClosureFault.preparedResult`: its baseline and requested
duplicate-delivery candidates map to the two exact RuntimeConfiguration closures already admitted
by `tools/umpire/temporal/nexus.Binding`. Run Evaluation already admits the duplicate-delivery point;
task `.3` adds one exact third profile for the compiled baseline point's complete Experiment binding,
paired only with the ordinary caller-closure RuntimeConfiguration. The command does not accept the
basic-lifecycle `VariationSpace` or a caller-supplied adapter.

No new persisted Artifact family or restart contract is introduced. Interruption returns a bounded
tooling outcome and discards the process-local session; a later invocation begins again from the
same checked inputs.

## API Contracts
<!-- scope: technical -->

- `umpire-fuzz run` accepts the literal caller-closure fault Space ID, one retained fn-17 policy,
  canonical fn-40 PlannerPolicy input, the fixed local-loopback Nexus binding, a candidate Limit from
  one through two, and a positive wall-clock Limit no greater than 240 seconds. Other Space,
  environment, adapter, executable, and zero/oversized Limit inputs reject before selection.
- The fixed Lean bridge supports `initialize`, `next`, `observe`, and `finish`. Each frame is
  canonical and carries only checked identities, Limits, one selected v2 ExperimentSpec, or one
  complete fn-18-admitted Result binding.
- `next` returns at most one candidate. Cleanup and Run Evaluation for that candidate must finish
  before `observe`; another `next` is unavailable while a candidate is outstanding.
- `observe` validates the complete Result closure and lets Lean update the selected-coordinate and
  finite-exhaustion state. Go cannot inspect, prioritize, or synthesize Model Coordinates.
- Before campaign execution, task `.3` extends the Go and Lean Run Evaluation allowlists with the
  exact compiled baseline-point Experiment binding from `CallerClosureFault.batchResult`. Its
  checksum, query ID, Behavior Fingerprint, provenance, properties, and ordinary caller-closure
  RuntimeConfiguration must all match. Cross-pairing it with duplicate-delivery configuration or
  admitting any other Space point remains `unsupported-profile`; this is not a generic profile seam.
- `finish` reports `exhausted|limit-reached|stopped|tooling-failure`, selected/executed/admitted
  counts, Lean-owned semantic coverage, and the first unexecuted candidate when known. This is
  terminal command output, not a persisted Artifact or restart token.
- Pinned Regressions use their existing ordinary Regression path independently of exploration
  Limits; the campaign neither installs nor owns them.

The command writes exactly one compact JSON object plus LF to stdout and nothing to stderr after a
campaign reaches `finish`. Its closed field order and nullability are:

| Field | Type and invariant |
| --- | --- |
| `formatVersion` | Literal `umpire-fuzz-run-summary/v1`. |
| `status` | `exhausted|limit-reached|stopped|tooling-failure`. |
| `limitKind` | `candidate|wall-clock|null`; non-null exactly for `limit-reached`. |
| `selectedCount` | Non-negative integer; increments when `next` returns a candidate. |
| `executedCount` | Non-negative integer no greater than selected; increments only when the runner starts. |
| `admittedCount` | Non-negative integer no greater than executed; increments only for a complete Run Evaluation Result. |
| `coveredCoordinates` | Canonically sorted unique checked Model Coordinate IDs from successful `observe` transitions only. |
| `requestedCoordinateOutcome` | `coordinate-selected|coordinate-uncovered|null`, copied from the final Lean state. |
| `firstUnexecutedCandidate` | Canonical ExperimentSpec identity or `null`. |
| `failure` | `null` unless `tooling-failure`; otherwise exactly `{phase,code,candidateIdentity}` with phase `bridge|runner|cleanup|evaluation` and a stable closed code. |

JSON uses the table's field order, no insignificant whitespace, canonical coordinate ordering, UTF-8,
and one terminal LF. `exhausted` exits 0; `limit-reached` and `stopped` exit 2; `tooling-failure`
exits 1. Argument/admission failure or inability to encode/write that summary instead produces no
stdout and one compact `umpire-fuzz-run-error/v1` JSON object plus LF on stderr with exactly
`formatVersion`, `phase=arguments|admission|reporting`, and a stable closed `code`, then exits 1.
There is no in-flight progress stream; timestamps, durations, and arbitrary error text appear in
neither terminal format.

## Edge Cases & Constraints
<!-- scope: technical -->

- Candidate Limit zero is an admission error. Candidate/wall-clock Limit exhaustion, cancellation,
  bridge failure, runner failure, incomplete cleanup, and rejected Run Evaluation remain distinct
  outcomes through `status`, `limitKind`, and `failure`.
- `selectedCount` advances on a successful `next`; `executedCount` advances only after runner start;
  `admittedCount` advances only after Run Evaluation returns a complete Result. A Result reaches
  `observe` only when cleanup completed, its operational status is `succeeded`, and Observation
  Evaluation is `accepted`; satisfied and unsatisfied semantic outcomes are both observations.
  Operationally failed Results remain admitted and increment `admittedCount`, but terminate with
  `tooling-failure`, do not call `observe`, and add no semantic coverage. Cleanup or evaluation
  rejection similarly adds no coverage.
- Exhaustion before another `next` reports `exhausted`; exhausting the candidate or wall-clock bound
  reports `limit-reached` with the matching `limitKind`; caller cancellation reports `stopped` after
  bounded cleanup; bridge, runner, cleanup, and evaluation failures report `tooling-failure` with
  their phase/code. An active candidate is the first unexecuted candidate only if runner start never
  occurred; otherwise `firstUnexecutedCandidate` comes from Lean's next known selection or is null.
- Every candidate and Result remains bound to the same checked Space, policy, environment input,
  Definition IDs, Behavior Fingerprints, Artifact Checksum, and Limits.
- For a fixed checked input and admitted Result/outcome stream, candidate selection and ordering are
  deterministic until a runtime stop. Wall-clock scheduling can change where that sequence stops,
  while timestamps do not enter semantic identities or canonical output.
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

- **R1:** One bounded canonical Lean bridge over the exact two-choice caller-closure fault Space keeps candidate selection, the requested uncovered
  coordinate, semantic coverage, and exhaustion Lean-owned while Go sees only checked bindings,
  Limits, one v2 candidate, and one complete admitted Result.
- **R2:** The command executes exactly one candidate at a time through the existing runner and Run
  Evaluation interfaces, with cleanup complete before the Result is observed or another candidate
  is requested.
- **R3:** Every candidate and Result remains bound to the same checked Space, fn-17 policy, fn-40
  PlannerPolicy, environment input, Definition IDs, Behavior Fingerprints, Artifact Checksum, and
  Limits. Crossed, stale, incomplete, or duplicate values reject before the next selection.
- **R4:** The closed versioned stdout/stderr and exit-code contract distinguishes exhausted, Limit
  Reached, stopped, and tooling failure and reports Lean-owned semantic coverage without treating
  unexecuted, failed, or rejected work as coverage.
- **R5:** Identical checked inputs, candidate Limit, admitted Result/outcome stream, and terminal stop
  reason select the same serial sequence and canonical terminal encoding. Wall-clock scheduling is
  outside that equality claim; pinned Regressions remain on the ordinary
  Regression path outside the exploration Limit.
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
- No general Run Evaluation profile expansion: only the exact caller-closure fault baseline point
  added by task `.3` joins the already admitted ordinary and duplicate-delivery bindings.
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
