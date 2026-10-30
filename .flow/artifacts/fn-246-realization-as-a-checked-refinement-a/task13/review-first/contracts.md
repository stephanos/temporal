**Scope read:** task `.13` spec, the full protocol module (`model.rs`, `parameters.rs`, `adapter.rs`, `view.rs`, tests, mutants), the feature-side correction hunks, both READMEs, fixture/build/ownership registration, and the committed evidence under `task13/resume-20261030`.

## Contract assessment

The code matches its stated promises on every semantic point I traced:

- **Approved correction is exactly scoped.** Feature `model.rs:206` (pending retryable Fail → Failed) and `model.rs:303` (pending local expiry → TimedOut) are the only policy edits; the retry branches, scheduled Cancel → Canceled and `RespondCanceled` → Canceled are untouched. The `tick_priority` law and the renamed progress claim encode the new outcome, and the old name `pending_retryable_failure_cancels` has no remaining references.
- **Protocol Model agrees with its README.** Tick order ScheduleToClose > StartToClose > Heartbeat > ScheduleToStart, decrement-before-select, spanning budget surviving retry, heartbeat resetting only its own interval, pending cancel preserving timers, FailedPrecondition/NotFound rejection cells, and terminal clearing all match the handlers at `protocol/activity/model.rs`. Hand-traced the `[3,2,3,1]` and `[2,1,None,1]` test walks; they are consistent with the code.
- **Latent state is genuinely latent.** `View::from` drops all four budgets; `Snapshot` has no path to `Model`; view tests show distinct full states with equal views; the 20 source mutants edit only pre-invariant code and are refuted by full-graph disagreement, not by view comparison.
- **Parameter contract is exact.** `inspect_world` checks declared/exported field count, name, type (incl. `Bounded` name/field/backing/low/high) and export order against a fixed six-field schema, then value kinds; `Parameters::new` refuses zero and out-of-range without substitution; `RawStart::into_effective` is the only defaulting site. Feature World field order/types verified identical, so cross-validation against both Worlds is sound.
- **Fixture wiring.** Optional encoding in `Protocol.cfg.snap` (`Some(1)`→2, `None`→0) matches the existing feature encoding. `worlds.txt.snap`, `report.txt.snap`, `tlc.txt.snap` match the README's 12 Worlds / 16 laws / 23 states / 80 edges / 90 states / 750 edges claims. Snapshot-ownership gate includes `temporal/shared/protocol`.
- **Evidence.** The initial required run failed only on a missing pinned TLC jar; `fetch-tlc.log` then `protocol-required-renewed.log` (22 passed, real TLC) supersede it and the handoff records this honestly. Native run: 62 passed, 2 inherited ignores. Guest/canary hash `8b9ea3…` reproduced byte-exact per `guest-publication.json`.
- **Manifest wording (focus item).** The task says the hash manifest is "removed"; in the repo `.flow/artifacts/fn-227…/original-artifacts.sha256` still exists and `checkpoint.rs` does not read it. The feature README now says "Original bytes and the semantic verifier remain unchanged" and the compatibility README already says the active gate compares semantics. The change's interpretation (no active hash authority, historical record preserved, no deletion of a protected `.flow` artifact) is consistent and correct; the stale phrasing is in the task text, not the code.

## Findings

**1.**
- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/temporal/shared/protocol/activity/README.md:151`
- **R-IDs**: [R5]
- **Problem**: README states "The required protocol test run passed 22 tests in 45.96 seconds (66.566 seconds including recompilation)". The committed evidence contradicts it: `proof-handoff.json:106` says those timings "have no recovered final log", and the only passing required run (`protocol-required-renewed.log:31`) took 96.08s. A doc number that the evidence itself disowns.
- **Suggestion**: Cite the renewed run (96.08s) or drop the timing sentence; keep the 22-test count.

**2.**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/temporal/shared/protocol/activity/view.rs:34`
- **R-IDs**: []
- **Problem**: `Status::Canceled` is documented as "Cancellation was acknowledged or closed a queued execution", but `Client::RequestCancel` on an Executing attempt with `asynchronous_cancel == false` also yields `Terminal::Canceled` (model.rs `client` handler, else-branch). The doc omits the atomic-compatibility path the README explicitly names.
- **Suggestion**: Add "or atomic-compatibility running cancellation" to the variant doc (the `Terminal::Canceled` doc already says this).

**3.**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/temporal/shared/protocol/activity/tests.rs:183-185`
- **R-IDs**: []
- **Problem**: Overengineering / Duplicated Code. `#[allow(unreachable_pub)] #[path = "parameters.rs"] mod parameters;` compiles a second copy of the parameter contract into the test binary, while the same file also imports `umpire_temporal::activity_protocol::{adapter, parameters::Parameters}` at line 198. `validate_effective_world`/`validate_world` are generic over `M: Values + Lifts`, so the library copy already accepts the test-crate `World`, the feature `World`, and the `Control<N>` mocks. Two compiled copies of one contract can silently diverge in behaviour if `cfg` gates or the `#[path]` target change.
- **Suggestion**: Remove the path include; have `parameters_tests.rs` and `scope()` use `umpire_temporal::activity_protocol::parameters`.

## FYI (non-blocking)

- `View` carries both `status` and `terminal_outcome`, which are a bijection over `Terminal`. The README justifies it (Describe may omit the outcome payload while status is always present), so it is a deliberate observational distinction rather than speculative generality; noting only because a `Snapshot` can express a contradictory pair.
- The two READMEs cite different line ranges of the same pinned API files for the same facts (`service_grpc.pb.go` 799–805 vs 2771–2777; `activity.pb.go` 32–63 vs 100–114). Plausibly client-vs-server interface copies, but unverifiable offline. Suppressed.
- Process: `realization/src/lib.rs` touch is within the declared `.2` overlap guard; conductor's call.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R5 | partial (remainder deferred to .3/.4 by task scope) | This task's portion is met: independent full protocol Model with latent timers and separate view (`protocol/activity/model.rs`, `view.rs`), native fixtures for 12 Worlds (`expected/worlds/*`), required TLC own-invariant and state-set agreement (`tlc.txt.snap`, `protocol-required-renewed.log`), exact every-field World/state accounting (`parameters.rs`, `adapter.rs`), approved pending-cancellation correction with renewed proof (`feature-policy-required.log`, `required-complement-seven.log`). Native refinement, online view-commutation and stutter/gap reports are assigned to .3/.4 per the task description and are not claimed here. |

Unaddressed R-IDs: []

Suppressed findings: 1
Classification counts: 3 introduced, 0 pre_existing.

```json
{"suppressed_count":{"25":1},"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>