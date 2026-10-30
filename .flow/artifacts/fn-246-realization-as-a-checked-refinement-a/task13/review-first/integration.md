## Review: fn-246.13 independent standalone-activity protocol Model (integration axis)

**Outcome: SHIP.** The new shared protocol module meets the unchanged fixture runner, build-script discovery, snapshot-ownership gate, realization crate, guest and canary policy correctly. The approved pending-cancellation correction is narrow and the atomic compatibility checkpoint is provably untouched by it. I found no behavioural defect, no regression in unchanged callers, and no security exposure. Two non-blocking items below.

### Integration points verified against unchanged code

- **Fixture discovery and runner.** `discovery::generate` in MODEL_AND_TESTS mode compiles `model.rs` as `activity` and `tests.rs` as `activity_tests`, and `registered()` points at `activity_tests::run`. The new `tests.rs` defines `run`, `lifted`, `check`, `scopes` with the signatures the three new `Consumer`s and `run_at_skipping` expect (`Fixture = (&str, fn() -> RunResult)`, `RunResult = Result<Files, Box<dyn Error>>`; the `String` error from `lifted` converts). `Consumer::owns` pattern `worlds/*.txt` matches `worlds/00-report.txt`. The committed `expected/` set is exactly the union of the four consumers' outputs.
- **Build script rerun semantics.** `build_at` already printed `cargo:rerun-if-changed=../feature`, so the new directory line for `../shared/protocol` does not newly disable Cargo's default rerun behaviour. No regression for feature fixtures.
- **`stay!()` in the Heartbeat arm.** `outcome::stay()` only sets a flag and does not return, so the heartbeat budget reset after it executes. The `wrong-heartbeat-reset` mutant being refuted (protocol-mutants.txt) confirms this empirically.
- **Snapshot-ownership gate.** Adding `temporal/shared/protocol` to `FIXTURES` makes every file under `activity/expected/` (including `worlds/`) owned via `starts_with`. Separate target passed (snapshot-ownership.log, 2 tests).
- **Realization crate / guest.** `#[path = "../../shared/protocol/activity/mod.rs"] pub mod activity_protocol;` resolves `mod model;` relative to the `mod.rs` directory. Checker-only code is cfg-gated off wasm32. Guest bytes reproduce exactly and `tools/canary/policy/policy_test.go` plus `preflight.go` compare `modelHash` to the committed guest SHA-256, which matches `8b9ea3…`. No other file pins the old hash.
- **Renamed feature claim.** No consumer outside `.flow/` references `pending_retryable_failure_cancels`.
- **Atomic checkpoint.** `compatibility/checkpoint.rs` explores `World::new(1, Some(1), Some(1), None, None, false)` and asserts `!cancel_requested` on every state, so neither corrected cell (pending Failed / pending TimedOut) is reachable there. The 1/6/11 semantic comparison and the changed-edge negative control stand unchanged.
- **No feature import in the shared module.** `model.rs`, `adapter.rs`, `parameters.rs`, `view.rs`, `mod.rs` import only `umpire_dsl`. The feature World appears only in test files compiled into `umpire-temporal-tests`, which is the task's explicit "validate against both independent World schemas" requirement.
- **Task wording on the hash manifest.** `.flow/artifacts/fn-227-…/original-artifacts.sha256` still exists; `compatibility/README.md` (unchanged) already describes it as historical hashes with the active gate comparing semantics; no active test reads it. The change's reading ("hash authority removed from the gate, file preserved as provenance") is consistent with the repo state and the spec's "preserve historical evidence" instruction. The feature README edit removing "independent SHA256 authority" is the right correction.
- **Raw Start defaults.** The pinned `go.temporal.io/api` module cache is not readable in this session. The server's own `chasm/lib/activity/proto/v1/activity_state.proto` documents the same two rules `RawStart::into_effective` implements (waiting defaults to schedule_to_close; one of start_to_close/schedule_to_close required). Treated as consistent, not independently confirmed against the pinned revision.

### Findings

**1.**
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/temporal/shared/protocol/activity/tests.rs:203-207`
- **R-IDs**: []
- **Problem**: `parameters.rs` is compiled a second time into the test crate via `#[allow(unreachable_pub)] #[path = "parameters.rs"] mod parameters;`, producing a second, distinct `Parameters`/`validate_effective_world` alongside the public `umpire_temporal::activity_protocol::parameters` that the very same file already imports at `tests.rs:218`. The lib functions are generic over `Values + Lifts`, so they accept the test-crate `World` and the feature `World` without the duplicate. This is duplicated machinery the task does not need.
- **Suggestion**: Delete the `#[path = "parameters.rs"] mod parameters;` block, point `parameters_tests.rs` and `scope()`/`scopes()` at `umpire_temporal::activity_protocol::parameters`.

**2.**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/temporal/shared/protocol/activity/README.md:150-151`
- **R-IDs**: []
- **Problem**: The README states the required protocol run "passed 22 tests in 45.96 seconds (66.566 seconds including recompilation)". The committed `proof-handoff.json` limitations say those figures have no recovered log and the renewed proof measured 96.08 s. The spec forbids unevidenced timings.
- **Suggestion**: Replace with the recovered 96.08 s figure (or drop the timing sentence) so the README matches the committed evidence.

**FYI (not findings):**
- The `UMPIRE_PROTOCOL_SCOPE_REPORT` / `UMPIRE_PROTOCOL_MUTANT_REPORT` env-var file writes inside tests are a new convention in this tree; harmless, but it is evidence-capture living in test bodies.
- `discover()` refuses any non-directory entry directly under `shared/protocol/`, so a future top-level README there would break the build script. Pre-existing discovery behaviour, now reachable from a second root.
- Code-smell baseline: `edits()` in `mutants.rs` carries an `expect(clippy::too_many_lines)` with reason; the per-field `unwrap_or(Remaining { ticks: 0 })` chains in `model.rs` are the DSL's liftable-subset idiom (the feature Model uses the same), not a new smell.

### Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R5 | partial | Independent full protocol Model with latent timers and separate view (`model.rs`, `view.rs`); native fixtures and required TLC own-invariant/state-set agreement for 12 Worlds (`expected/tlc.txt.snap`, protocol-scopes.txt); approved cancellation correction with renewed native/TLC evidence (feature-policy-required.log, required-complement-seven.log). Native refinement, observable-view commutation with online matching and the mapped-edge/stutter controls are owned by tasks .3/.4 per the task description and are not claimed here. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>