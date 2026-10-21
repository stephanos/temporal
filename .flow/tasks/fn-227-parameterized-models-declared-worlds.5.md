---
satisfies: [R6]
---
# fn-227-parameterized-models-declared-worlds.5 Transactional live World admission, cooperative evidence and SDK retry proof

## Description

Keep original .5 after .4 and T. Consume completed actual-request transport/capture; implement its live admission and realization through the existing Temporal Dispatch facade, then prove retry and cooperative cancellation with real SDK/server evidence. Owner approval at `bbf0835fbf` is authoritative. No further envelope/core ABI changes, fabricated Start cancellation field or hidden default are authorized.

**Size:** M after T. One facade admission transaction state machine plus extensions of the existing realization/live harness. The native/Wasm publication controls exercise that state machine. The full unsplit protocol/capture/facade/live scope remains L; T removes an independently testable contract. Commit must remain owned moves; another runtime-generation or acknowledgment protocol would exceed this scope.
**Dependencies:** Keep fn-227-parameterized-models-declared-worlds.4 and consume .12 (T). Keep `.5 -> .6`. T needs no `.4` edge. Use completed fn217 catalog and fn231 declarative realization; no unfinished-catalog fallback.
**Files:** `model/temporal/realization/src/{annotations,support}.rs`; `model/temporal/feature/activity/{temporal,phase}.rs` at their actual post-.4 owners; `model/temporal/realization/tests/realization.rs`; `model/temporal/hosts/go/{allocation_test,parity_test}.go`; `model/umpire/wasm/tests/{stateful,abi_staging}.rs` only for generic existing transaction contract controls; `tests/testpilot_model_slice_test.go` and annotation test integration only where admission consumes T; realization docs and `model/README.md`; the committed Temporal guest.
**Touches:** [model/temporal/realization/**, model/temporal/feature/activity/temporal.rs, model/temporal/feature/activity/phase.rs, model/temporal/hosts/go/**, model/umpire/wasm/tests/**, tests/testpilot_model_slice_test.go, tests/testpilot_model_annotations_test.go, model/README.md]

## Approach

1. Consume T's immutable captured request/raw result. Validate canonical types, successful new-run result and namespace/activity/request/run identity before domains; cross-check optional response Link. Use fn217 validation without repairing captured bytes. Write a concrete per-field retry/timeout-to-tick admission table; absent/zero wire values map to None only through that table. After exact quantization, use native .2's range-newtype try_new for each present timeout/countdown seed and explicitly check maximum_attempts before World::new. Map an invalid present scalar to field-named Unsupported through the admission-freeze path. This is an explicit Temporal option admission function, not World::new or generic checker validation; never approximate a World.
2. Pin the explicit standalone cooperative-cancellation protocol binding to the API/SDK versions and local protocol evidence recorded in cancellation research, rechecking actual admitted revisions at execution. Retain its identity/evidence with the immutable Start receipt. Missing/unsupported binding names asynchronous_cancel and refuses admission; Start supplies no such field. Distinguish actual Cancel, pending Describe, heartbeat cancel notification, worker acknowledgement and terminal Describe. Completion/failure after pending cancellation remain admissible races. Do not infer cancellation from callback labels or the workflow-only WaitForCancellation policy.
3. Keep one execution per guest lifetime. Only started=true installs a new live binding. Initial started=false refuses; exact original receipt replay after progress/Reset is zero-step, including an idempotent started=false response with original successful provenance and same captured request/run. Changed request bytes under one ID, crossed execution/run IDs or another execution in a bound guest refuse state-preservingly. Later actual RPC requests and Describe/callback namespace/activity_run_id/attempt must match the binding before projection. An omitted run ID cannot identify this run.
4. Extend AnnotationRuntime's existing Dispatch facade with owned preparation variants for whole-Runtime replacement plus receipt, admission freeze, delegated Model effects and composed Reset. A prospective valid World has its unique Scheduled/attempt-one start and no exploration. Construct/validate/allocate before publication; commit performs owned moves. Generic Runtime/Realize gets no Temporal fields or mutable World escape. The existing explicit fixture World remains checker-only before admission and cannot justify live Judge/Answer.
5. Domain failure prepares Unsupported plus Some(transaction) to install the first admission freeze while preserving old machine/state/exploration. Ordinary malformed/failed/crossed evidence changes nothing and cannot overwrite that first freeze. Published freeze blocks advancement/reconfiguration while stateless Annotate/Probe remains independent. Reset retains selected World, receipt and execution identity, restores the unique start and clears freeze transactionally. A rejected prebinding candidate never becomes an admitted World after Reset; retain its attempted identity, and require a fresh guest for a new actual execution.
6. Extend actual callback/Describe realization and live worker behavior for retryable failure then completion. Capture/judge successful Start before worker startup. Use real SDK attempt values and synchronized actual Describe transitions; do not reorder stale observations. Preserve Complete, Fail, HostRefusal and context-Cancellation, and add a distinct held-attempt server-cancellation path with actual Cancel/pending heartbeat/acknowledgement plus completion-after-cancel race control.

## Investigation targets

Required, reanchor baseline coordinates:

- `.flow/artifacts/fn-227-parameterized-models-declared-worlds/{start-admission-contract.md,cancellation-evidence-research.md}` for transaction, identity, trust and pinned evidence constraints under current approvals.
- `model/temporal/realization/src/annotations.rs` for facade ownership and Dispatch outcomes.
- `model/umpire/wasm/src/lib.rs` and `abi.rs` for Refusal-plus-transaction and exact publication boundaries; read-only production interfaces.
- `model/temporal/realization/tests/realization.rs` for native rollback and malformed-evidence precedence.
- `model/temporal/hosts/go/allocation_test.go` for real 16 MiB response-buffer exhaustion.
- `tests/testpilot_model_slice_test.go` for actual callback/Describe/live sequencing and existing controls.

## Verification

Quick from model/: `mise exec -- cargo test --manifest-path temporal/realization/Cargo.toml`; `mise exec -- cargo test -p umpire-wasm -p umpire-temporal-tests`. From `model/temporal/hosts/go`: `go test -tags test_dep ./...`. Run current focused annotation/slice tests through their repository harness with required test_dep/integration tags. Scoped Rust fmt/strict clippy and changed-Go lint.

For each success/freeze/Reset transaction, test native false/panicking publication, oversized response and actual Wasm response-allocation exhaustion. First Explore fixture World, reject admission publication and retain every old field/row; publish and require Rows NotExplored until re-exploration under the selected World. Reject freeze/Reset publication similarly; retry commits once. Guest-owned response publication/allocation is the rollback boundary; later host-read failure cannot roll back guest commit under this ABI. An already successful remote Start cannot be undone; retain its actual capture for idempotent retry.

Full from root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`, `make umpire-guest`, `make umpire-check-model-slice`, `make umpire-check-model-gaps`. Preserve the umbrella's unchanged legacy scope and report inherited failures accurately. Retain capture identities, publication/allocation evidence, live Describe attempts, privacy checks and actual timings; do not expose private values in diagnostics.

## Acceptance

- [ ] Selected World admission uses actual filled same-invocation evidence, an explicit request quantization table and retained pinned cooperative protocol identity. Missing/failed/crossed evidence, initial started=false and unsupported binding cannot install a World; domains name the offending field.
- [ ] Original receipt replay after progress/Reset is zero-step. Namespace/activity/request/run/attempt and Link controls detect crossed or contradictory evidence, and a fresh guest is required for another execution.
- [ ] Whole-Runtime success, first Unsupported freeze and retained-World Reset commit through facade ownership only. Exploration invalidates only on committed replacement; generic runtime production has no Temporal data or ABI change.
- [ ] Native and actual Wasm publication/allocation failures preserve World/state/exploration/receipt/freeze/Reset atomically; retry commits once. Malformed-evidence precedence, absent-request wire compatibility and privacy controls remain discriminating.
- [ ] Actual SDK retryable failure then completion is judged through real attempts/Describe; pending heartbeat does not close; explicit acknowledgement and completion-after-cancel are independently exercised. Complete/Fail/HostRefusal/context-Cancellation, refreshed guest slice/gaps and full gates/timings remain reported.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
