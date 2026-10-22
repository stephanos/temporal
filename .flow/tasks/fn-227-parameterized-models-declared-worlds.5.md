---
satisfies: [R6]
---
# fn-227-parameterized-models-declared-worlds.5 Transactional live World admission and SDK retry proof

## Description
Keep original .5 after .3 and T. Consume completed actual-request transport/capture; implement its live admission and realization through the existing Temporal Dispatch facade, then prove retry with real SDK/server evidence. Owner approval at `bbf0835fbf` is authoritative. No further envelope/core ABI changes, fabricated Start cancellation field or hidden default are authorized.

On 2026-10-21 the owner moved the cooperative-cancellation part of this task to fn-237-activity-timeouts-heartbeat-and.4: the pinned cooperative-cancellation protocol binding, the held-attempt server-cancellation path with pending heartbeat, acknowledgement and completion-after-cancel race, and the timeout-to-tick part of the admission table. Admission here covers `maximum_attempts` only. Request timeout fields (schedule-to-start, start-to-close, schedule-to-close, heartbeat) are recorded with the immutable Start receipt but not admitted as World parameters before fn-237 lands; present values are not refused and not quantized, and no timeout-derived World field exists to bind. fn-237.4 replaces that recording with admission through this task's transaction path.

**Size:** M after T. One facade admission transaction state machine plus extensions of the existing realization/live harness. The native/Wasm publication controls exercise that state machine. The full unsplit protocol/capture/facade/live scope remains L; T removes an independently testable contract. Commit must remain owned moves; another runtime-generation or acknowledgment protocol would exceed this scope.
**Dependencies:** Consume fn-227-parameterized-models-declared-worlds.3 (retry Model) and .12 (T). The former .4 dependency moved with .4 to fn-237. Keep `.5 -> .6`. Use completed fn217 catalog and fn231 declarative realization; no unfinished-catalog fallback.
**Files:** `model/temporal/realization/src/{annotations,support}.rs`; `model/temporal/feature/activity/{temporal,phase}.rs` at their actual post-.3 owners; `model/temporal/realization/tests/realization.rs`; `model/temporal/hosts/go/{allocation_test,parity_test}.go`; `model/umpire/wasm/tests/{stateful,abi_staging}.rs` only for generic existing transaction contract controls; `tests/testpilot_model_slice_test.go` and annotation test integration only where admission consumes T; realization docs and `model/README.md`; the committed Temporal guest. Refresh the exact activity realization coverage mirror `model/temporal/feature/activity/expected/realization.snap.txt` when actual attempt/identity classifications change; preserve unrelated and semantic goldens.
**Touches:** [model/temporal/realization/**, model/temporal/feature/activity/temporal.rs, model/temporal/feature/activity/phase.rs, model/temporal/hosts/go/**, model/umpire/wasm/tests/**, tests/testpilot_model_slice_test.go, tests/testpilot_model_annotations_test.go, model/README.md, model/temporal/feature/activity/expected/realization.snap.txt]

### Approach

1. Consume T's immutable captured request/raw result. Validate canonical types, successful new-run result and namespace/activity/request/run identity before domains; cross-check optional response Link. Use fn217 validation without repairing captured bytes. Write a concrete per-field admission table: `maximum_attempts` is admitted; the four timeout fields are listed as recorded-not-admitted, with fn-237.4 named as their future owner. Explicitly check maximum_attempts before World::new. Map an invalid present maximum_attempts to field-named Unsupported through the admission-freeze path. This is an explicit Temporal option admission function, not World::new or generic checker validation; never approximate a World.
2. Keep one execution per guest lifetime. Only started=true installs a new live binding. Initial started=false refuses; exact original receipt replay after progress/Reset is zero-step, including an idempotent started=false response with original successful provenance and same captured request/run. Changed request bytes under one ID, crossed execution/run IDs or another execution in a bound guest refuse state-preservingly. Later actual RPC requests and Describe/callback namespace/activity_run_id/attempt must match the binding before projection. An omitted run ID cannot identify this run.
3. Extend AnnotationRuntime's existing Dispatch facade with owned preparation variants for whole-Runtime replacement plus receipt, admission freeze, delegated Model effects and composed Reset. A prospective valid World has its unique Scheduled/attempt-one start and no exploration. Construct/validate/allocate before publication; commit performs owned moves. Generic Runtime/Realize gets no Temporal fields or mutable World escape. The existing explicit fixture World remains checker-only before admission and cannot justify live Judge/Answer.
4. Domain failure prepares Unsupported plus Some(transaction) to install the first admission freeze while preserving old machine/state/exploration. Ordinary malformed/failed/crossed evidence changes nothing and cannot overwrite that first freeze. Published freeze blocks advancement/reconfiguration while stateless Annotate/Probe remains independent. Reset retains selected World, receipt and execution identity, restores the unique start and clears freeze transactionally. A rejected prebinding candidate never becomes an admitted World after Reset; retain its attempted identity, and require a fresh guest for a new actual execution.
5. Extend actual callback/Describe realization and live worker behavior for retryable failure then completion. Capture/judge successful Start before worker startup. Use real SDK attempt values and synchronized actual Describe transitions; do not reorder stale observations. Preserve Complete, Fail, HostRefusal and context-Cancellation.

### Investigation targets

Required, reanchor baseline coordinates:

- `.flow/artifacts/fn-227-parameterized-models-declared-worlds/start-admission-contract.md` for transaction, identity and trust constraints under current approvals.
- `model/temporal/realization/src/annotations.rs` for facade ownership and Dispatch outcomes.
- `model/umpire/wasm/src/lib.rs` and `abi.rs` for Refusal-plus-transaction and exact publication boundaries; read-only production interfaces.
- `model/temporal/realization/tests/realization.rs` for native rollback and malformed-evidence precedence.
- `model/temporal/hosts/go/allocation_test.go` for real 16 MiB response-buffer exhaustion.
- `tests/testpilot_model_slice_test.go` for actual callback/Describe/live sequencing and existing controls.

### Verification

Quick from model/: `mise exec -- cargo test --manifest-path temporal/realization/Cargo.toml`; `mise exec -- cargo test -p umpire-wasm -p umpire-temporal-tests`. From `model/temporal/hosts/go`: `go test -tags test_dep ./...`. Run current focused annotation/slice tests through their repository harness with required test_dep/integration tags. Scoped Rust fmt/strict clippy and changed-Go lint.

For each success/freeze/Reset transaction, test native false/panicking publication, oversized response and actual Wasm response-allocation exhaustion. First Explore fixture World, reject admission publication and retain every old field/row; publish and require Rows NotExplored until re-exploration under the selected World. Reject freeze/Reset publication similarly; retry commits once. Guest-owned response publication/allocation is the rollback boundary; later host-read failure cannot roll back guest commit under this ABI. An already successful remote Start cannot be undone; retain its actual capture for idempotent retry. Add a control that present timeout values are recorded and neither refused nor admitted.

Full from root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`, `make umpire-guest`, `make umpire-check-model-slice`, `make umpire-check-model-gaps`. Preserve the umbrella's unchanged legacy scope and report inherited failures accurately. Retain capture identities, publication/allocation evidence, live Describe attempts, privacy checks and actual timings; do not expose private values in diagnostics.
## Acceptance

- [ ] Selected World admission uses actual filled same-invocation evidence and an explicit per-field admission table covering maximum_attempts. Missing/failed/crossed evidence and initial started=false cannot install a World; an out-of-domain maximum_attempts names the field. Present request timeout values are recorded with the receipt and neither refused nor admitted.
- [ ] Original receipt replay after progress/Reset is zero-step. Namespace/activity/request/run/attempt and Link controls detect crossed or contradictory evidence, and a fresh guest is required for another execution.
- [ ] Whole-Runtime success, first Unsupported freeze and retained-World Reset commit through facade ownership only. Exploration invalidates only on committed replacement; generic runtime production has no Temporal data or ABI change.
- [ ] Native and actual Wasm publication/allocation failures preserve World/state/exploration/receipt/freeze/Reset atomically; retry commits once. Malformed-evidence precedence, absent-request wire compatibility and privacy controls remain discriminating.
- [ ] Actual SDK retryable failure then completion is judged through real attempts/Describe. Complete/Fail/HostRefusal/context-Cancellation, refreshed guest slice/gaps and full gates/timings remain reported.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
