---
satisfies: [R4]
---
# fn-237-activity-timeouts-heartbeat-and.4 Live timeout admission, cooperative cancellation evidence and SDK cancellation proof

## Description
Moved from fn-227.5 on 2026-10-21: the timeout-to-tick part of its admission table (Approach step 1), its pinned cooperative-cancellation protocol binding (step 2), the held-attempt server-cancellation path with pending heartbeat, acknowledgement and completion-after-cancel race (step 6) and the matching acceptance text.

Extend fn-227.5's transactional live World admission and realization, through the existing Temporal Dispatch facade, to the timeout fields and cooperative cancellation delivered in fn-237.3, then prove cooperative cancellation with real SDK/server evidence. fn-227's owner approval at `bbf0835fbf` is authoritative. No further envelope/core ABI changes, fabricated Start cancellation field or hidden default are authorized.

**Size:** M. Extensions of fn-227.5's admission table, evidence mapping and live harness; the admission transaction state machine, receipt, freeze and Reset remain fn-227.5's and are reused, not rebuilt. Another runtime-generation or acknowledgment protocol would exceed this scope.
**Dependencies:** Consume fn-237-activity-timeouts-heartbeat-and.3 (timeout and cooperative-cancellation Model). Cross-spec prerequisites fn-227.5 (transactional admission, receipt, freeze/Reset and the SDK retry proof) and fn-227.12 (T, actual-request transport), satisfied through this spec's dependency on fn-227. fn-237.5 follows.
**Files:** `model/temporal/realization/src/{annotations,support}.rs`; `model/temporal/feature/activity/{temporal,phase}.rs` at their actual post-fn-237.3 owners; `model/temporal/realization/tests/realization.rs`; `model/temporal/hosts/go/{allocation_test,parity_test}.go` only where the extended admission needs existing transaction controls; `tests/testpilot_model_slice_test.go` and annotation test integration where cancellation evidence is consumed; realization docs and `model/README.md`; the committed Temporal guest.
**Touches:** [model/temporal/realization/**, model/temporal/feature/activity/temporal.rs, model/temporal/feature/activity/phase.rs, model/temporal/hosts/go/**, tests/testpilot_model_slice_test.go, tests/testpilot_model_annotations_test.go, model/README.md]

### Approach

1. Extend fn-227.5's per-field admission table with the four request timeouts. Write a concrete per-field timeout-to-tick admission table; absent/zero wire values map to None only through that table. After exact quantization, use native fn-227.2's range-newtype try_new for each present timeout/countdown seed before World::new. Map an invalid present scalar to field-named Unsupported through fn-227.5's admission-freeze path. This is an explicit Temporal option admission function, not World::new or generic checker validation; never approximate a World. fn-227.5 records the timeout fields without admitting them; this task replaces that recording with admission.
2. Pin the explicit standalone cooperative-cancellation protocol binding to the API/SDK versions and local protocol evidence recorded in cancellation research, rechecking actual admitted revisions at execution. Retain its identity/evidence with the immutable Start receipt. Missing/unsupported binding names asynchronous_cancel and refuses admission; Start supplies no such field. Distinguish actual Cancel, pending Describe, heartbeat cancel notification, worker acknowledgement and terminal Describe. Completion/failure after pending cancellation remain admissible races. Do not infer cancellation from callback labels or the workflow-only WaitForCancellation policy.
3. Reuse fn-227.5's whole-Runtime replacement, receipt, freeze and Reset transactions unchanged for the extended World; the timeout/cancellation fields join the same prepared candidate. Existing transaction and publication/allocation controls must pass with the extended World.
4. Extend actual callback/Describe realization and live worker behavior with a distinct held-attempt server-cancellation path: actual Cancel, pending heartbeat and acknowledgement, plus a completion-after-cancel race control. Use real SDK attempt values and synchronized actual Describe transitions; do not reorder stale observations. Preserve fn-227.5's retryable-failure-then-completion proof and Complete, Fail, HostRefusal and context-Cancellation.

### Investigation targets

Required, reanchor baseline coordinates:

- `.flow/artifacts/fn-227-parameterized-models-declared-worlds/{start-admission-contract.md,cancellation-evidence-research.md}` for transaction, identity, trust and pinned evidence constraints under current approvals.
- fn-227.5's delivered admission table, receipt and facade in `model/temporal/realization/src/annotations.rs`.
- `model/temporal/realization/tests/realization.rs` for native rollback and malformed-evidence precedence.
- `tests/testpilot_model_slice_test.go` for actual callback/Describe/live sequencing and existing controls.

### Verification

Quick from model/: `mise exec -- cargo test --manifest-path temporal/realization/Cargo.toml`; `mise exec -- cargo test -p umpire-wasm -p umpire-temporal-tests`. From `model/temporal/hosts/go`: `go test -tags test_dep ./...`. Run current focused annotation/slice tests through their repository harness with required test_dep/integration tags. Scoped Rust fmt/strict clippy and changed-Go lint. Re-run fn-227.5's success/freeze/Reset publication and allocation controls with the extended World.

Full from root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`, `make umpire-guest`, `make umpire-check-model-slice`, `make umpire-check-model-gaps`. Report inherited failures accurately. Retain capture identities, the pinned binding identity, live Describe/heartbeat/acknowledgement evidence, privacy checks and actual timings; do not expose private values in diagnostics.

### Reworked for programmed workers (2026-10-24)

fn-243 replaces the Answer path this description inherited from fn-227.5. Read Approach steps 2 and 4 as follows:

- Worker behaviour is program rows, not Answer replies. Extend the activity key and response types (`model/temporal/proto/umpire/temporal/v1`) with `RecordActivityTaskHeartbeat` and `RespondActivityTaskCanceled` responses and the cancel-requested bit a heartbeat response reveals as a key field; the `WorkerProgram` envelope does not change. The Realization enumerates the new rows from the same reply policy; the coverage test extends to them. The raw executor sends them as is; the sdk executor's generic handler heartbeats and returns the canceled error.
- The held attempt and the schedule-to-start expiry come from the hold fault in the proposal's instruction (fn-243), never from a sleeping worker or `start_delay`.
- Judgment is from server history: the Describe projection reads outcome, last failure, heartbeat and cancel-request state and projects the multi-step paths. A worker that heartbeats or acknowledges where its row says otherwise is Unexplained.
- The live slice runs through `umpire explore` with the raw executor first, then the sdk executor with identical judgments: held-attempt cancellation with pending heartbeat and acknowledgement, completion-after-cancel race, schedule-to-start expiry under a hold, plus fn-227's retry proof and the Complete, Fail, HostRefusal and context-Cancellation controls.
- Paths: `umpire-wasm` is `umpire-runtime` (`model/umpire/protocol/runtime`); the Driver is `common/testing/testpilot/temporal/{act,control,provision}` as fn-243.2 leaves it; executors are `model/temporal/hosts/go/reference/worker/{raw,sdk}`; the live harness is whatever fn-243.2 leaves of `tests/testpilot_model_slice_test.go`. `model/temporal/**` paths are unchanged.
## Acceptance
- [ ] Bind actual cancellation-request acknowledgments, pending state, completion/failure races, worker acknowledgment and repeated-pending-request evidence to fn-237.3's independent contract. Supply committed cooperative proof for fn-243.5; do not silently remove the atomic checkpoint's unheld refusal or claim its separate no-hold admission obligation complete.
- [ ] Selected World admission uses actual filled same-invocation evidence, an explicit request timeout-to-tick quantization table and retained pinned cooperative protocol identity. Unsupported binding cannot install a World; out-of-domain timeouts and a missing binding name the offending field (`asynchronous_cancel` for the binding).
- [ ] The extended admission commits through fn-227.5's facade transactions only; its publication/allocation, freeze, Reset and receipt-replay controls pass with the extended World, and generic runtime production has no Temporal data or ABI change.
- [ ] Actual SDK held-attempt server cancellation is judged through real Cancel/Describe/heartbeat evidence; pending heartbeat does not close; explicit acknowledgement and completion-after-cancel are independently exercised.
- [ ] fn-227.5's retry proof and Complete/Fail/HostRefusal/context-Cancellation controls, refreshed guest slice/gaps and full gates/timings remain reported.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
