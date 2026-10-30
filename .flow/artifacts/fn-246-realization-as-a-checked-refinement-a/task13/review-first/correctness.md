**Scope read.** Task spec, full protocol Model (`model.rs`, `parameters.rs`, `adapter.rs`, `view.rs`, `mutants.rs`, tests), feature correction hunks, fixture registration, and the resume-20261030 evidence (proof-handoff, negative-control map, input-refresh, guest-publication).

## Correctness assessment

**Protocol Model** (`model/temporal/shared/protocol/activity/model.rs`): I traced every handler against the invariants. Budget arming/clearing is exact (`exact_budget_arming` holds at start, Deliver, retry, finish); `consume` decrements before selection so no reached state carries a zero budget; priority ScheduleToClose > StartToClose > Heartbeat > ScheduleToStart is total because queued and attempt/heartbeat budgets are mutually exclusive by lifecycle. Pending cancellation + retryable Fail closes Failed without increment; pending + local expiry closes TimedOut with the exact cause; scheduled RequestCancel and current-attempt AcknowledgeCancel close Canceled. Terminal outcome and `retained_failure` are separate and the view keeps them separate. Tick with no armed budget emits no edge (no `stay!`), matching the test at `tests.rs:179`. No logic errors found.

**Feature correction** (`feature/activity/model.rs:205-213, 301-311`): only the two approved cells changed; nonpending retry path, atomic Cancel, RespondCanceled and the one-attempt atomic checkpoint World (which can never be pending) are untouched, so `compatibility/checkpoint.rs` is unaffected.

**Parameters/adapter**: `inspect_world` checks declared schema, exported names, kinds, bounds and None vs present-zero; `RawStart::into_effective` is the only defaulting site and only defaults schedule_to_start from schedule_to_close. The `scope()` raw-realizable label in `tests.rs:242` is consistent with that rule (world 10 with absent schedule_to_start and present schedule_to_close is labelled abstract).

**Manifest wording**: the task's "removed original-artifact hash manifest" is stale as physical-removal language. The fn-227 `original-artifacts.sha256` is still tracked, nothing in `model/` reads it, and `compatibility/README.md:5-6` plus the feature README now describe it as historical with the active gate on semantics. Evidence in `negative-control-and-checkpoint-map.json` states this plainly. No code problem.

## Findings

- **Severity**: P3 · **Confidence**: 75 · **Classification**: introduced · **File:Line**: `model/temporal/shared/protocol/activity/mutants.rs:111-117` · **R-IDs**: [] 
  **Problem**: `wrong-pending-timeout-outcome` rewrites the single `Terminal::TimedOut` in `attempt_expired`, whose else-branch serves both the pending case and the nonpending at-cap case. The mutant therefore also turns ordinary at-cap local expiry into Canceled, so it is not a pending-specific control, though it still disagrees with the graph.
  **Suggestion**: Split `attempt_expired` into explicit pending/at-cap arms (or target the mutant with a guard-specific anchor) so the control isolates the approved correction.

- **Severity**: P3 · **Confidence**: 75 · **Classification**: introduced · **File:Line**: `model/temporal/shared/protocol/activity/tests.rs:183-187` · **R-IDs**: []
  **Problem**: `parameters.rs` is compiled twice: once in `umpire_temporal::activity_protocol` and again via `#[path]` into the testing crate. Tests then mix the two copies (`scope()` uses the local one at line 240; the adapter test at line 198 uses the library one). Two distinct `Parameters` types that happen to agree.
  **Suggestion**: Drop the `#[path]` module and import `umpire_temporal::activity_protocol::parameters` everywhere in the tests.

- **Severity**: P3 · **Confidence**: 75 · **Classification**: introduced · **File:Line**: `model/temporal/feature/activity/proof_checks.rs:649-896` · **R-IDs**: []
  **Problem**: The "independent law" that refutes the pending-expiry Canceled mutant is `tick_priority` line 608, which this same diff edited from `Canceled` to `TimedOut`. The test proves the law and the code were co-edited consistently, not that the law is independent of the code.
  **Suggestion**: FYI only; the protocol-side `pending_expiry_is_not_acknowledgement` provides the independent cross-check.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R5 | met (this task's slice) | Independent protocol Model, exact parameter/schema contract, 12-World native + TLC agreement (`expected/worlds.txt.snap`, `tlc.txt.snap`), 20 source mutants, approved feature correction with renewed evidence. Online binding and generic refinement consumer remain with .3/.4 per the task description. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 3 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>