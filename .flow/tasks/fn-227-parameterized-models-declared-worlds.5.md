---
satisfies: [R6]
---
# fn-227-parameterized-models-declared-worlds.5 After .4: realization reads activity options into the World; Answer selects retryable; live SDK retry proof

## Description
In temporal/realization, build activity::World from the observed StartActivityExecutionRequest activity options (retry policy maximum attempts, the four timeouts mapped to tick budgets through a declared domain, cancellation type); options outside the admitted domains are Unsupported naming the field and freeze the run. Answer maps the callback payload to RespondCompleted or RespondFailed { retryable }. Extend the live SDK slice (make umpire-check-model-slice) with a worker that fails retryably once and completes on the second attempt, judged through real Describe observations showing attempt advance; keep Complete, Fail, HostRefusal and Cancellation passing. Use fn-217's catalog fill/validate for the start request if fn-217 is closed, otherwise construct the request directly and note it.

**Admission gate:** The unchanged envelope has no faithful request/result transport, and Start has no cancellation-type field. The owner has not approved the proposed Observation request field or the independent cancellation contract/source. The original R6 and no-envelope-change constraint remain in force; do not implement .5, invent a field/default, or treat the proposals below as authorization. fn217 and fn231 are now complete; use their actual catalog and declarative Realization surfaces, without the old unfinished-catalog fallback.

**Size:** Reassess and split before admission if capture/protocol generation, facade transactions and live retry controls do not fit one iteration. No implementation is admitted by this engineering brief.
**Files:** `model/temporal/proto/umpire/temporal/v1/temporal.proto` only after authority, its generated consumers; `model/temporal/realization/src/{support,annotations}.rs`; `model/temporal/feature/activity/temporal.rs`; `model/temporal/realization/tests/realization.rs`; `model/temporal/hosts/go/{allocation_test,parity_test}.go`; `tests/testpilot_model_{annotations,slice}_test.go`; existing guest/protocol generation targets and authoring docs.
**Touches:** [model/temporal/proto/**, model/temporal/realization/**, model/temporal/feature/activity/temporal.rs, model/temporal/hosts/go/**, model/umpire/wasm/tests/**, tests/testpilot_model_annotations_test.go, tests/testpilot_model_slice_test.go, model/README.md]

## Proposed transport after explicit owner authority

The smallest proposal adds optional request Any at fresh Observation tag 4, preserving method/result tags 1/2/3 and absent-field bytes. Callback, Reply, the core oracle and the three-envelope count remain unchanged. This is still an envelope change requiring approval. Do not hide a fourth bundle inside response/Annotate or use successive annotation calls as a pairing cache.

At the existing SDK interceptor, clone the fully filled request immediately before the actual invoke, then clone its successful raw result from that same invocation. Keep RPC failure distinct from subsequent local response-validation errors. Verify the innermost invoker sees the captured bytes and later interceptors cannot mutate them. Preserve ownership across overlapping invocations and reversed responses. Do not reconstruct from intended SDK options or a global last-request slot. The transport owner supplies same-invocation trust; Start response bytes do not independently echo/authenticate request ID.

## Transaction and identity contract after the gate

Keep Temporal admission in the existing AnnotationRuntime Dispatch facade. Stage an owned prospective whole Runtime plus immutable actual Start receipt; keep generic Runtime/Realize free of Temporal request data. Success constructs the selected World and unique Scheduled/attempt-one start, drops old exploration only on commit, and clears prior live-state freeze consistently. Commit performs owned moves after publication; all fallible validation, construction and encoding happen first.

Validate message types and namespace/activity/request/run identity before option domains. Only actual successful started=true evidence installs a new binding. Without retained original provenance, started=false cannot establish a World. An exact request/result replay for the original identity, including idempotent started=false with retained original started=true provenance, is a zero-step check after progress. Different bytes under the same request ID, crossed identities/run IDs or another execution in a bound guest refuse state-preservingly. Check optional response Link when present. Pin one execution per live guest; a different execution uses a fresh guest.

Carry actual requests on subsequent observed RPCs under the approved transport and match namespace/activity/run before projection. Check Describe response identities and callback namespace/activity_run_id/attempt from actual SDK activity.Info. Do not infer a specific run from an omitted run ID, reorder stale Describe results, or manufacture callback fields. Preserve existing Complete/Fail/HostRefusal and context-Cancellation controls; the latter is distinct from a server-cancellation test.

Out-of-domain World options return field-named Unsupported plus an admission-freeze transaction through Dispatch's existing outcome-plus-optional-transaction pair. Keep the previous machine/state/exploration unchanged; never approximate a World. Do not use generic Runtime.prepare's Err-to-no-transaction path. Malformed/failed/crossed ordinary evidence stays state-preserving and cannot overwrite the first admission freeze. A published freeze blocks advancement/reconfiguration; stateless annotation remains independent. An existing explicit checker fixture World is checker-only and cannot stand in for live Start admission.

Reset retains the admitted World, immutable execution identity and original receipt, restores that World's unique start, and clears freeze transactionally. It does not install another execution. For a rejected prebinding admission, Reset cannot convert rejected options into a World or permit changed bytes for the same attempted identity. Use a fresh guest for another actual execution. Preserve generic Reset semantics.

For success, freeze and Reset, response encoding/publication failure or response-buffer allocation failure commits none of World/state/exploration/receipt/freeze effects. Native dispatch_to acceptance and Wasm guest-owned response allocation are the existing boundaries. Host read happens after guest commit and cannot be promised to roll it back under this ABI. Retain the actual successful remote Start for idempotent retry; model-publication failure cannot undo that RPC. Diagnostics name fields/preconditions without private request/response/error values.

## Investigation targets

**Required**:
- `model/temporal/realization/src/annotations.rs:10` - Temporal Dispatch facade to extend.
- `model/umpire/wasm/src/lib.rs:39` - Refusal plus Some(transaction) contract.
- `model/umpire/wasm/src/abi.rs:100` - native and Wasm publication boundaries.
- `tests/testpilot_model_annotations_test.go:39` - after-fill actual request/result capture and retry ownership.
- `model/temporal/hosts/go/allocation_test.go:138` - real response-allocation exhaustion controls.
**Optional**:
- `model/temporal/realization/tests/realization.rs:491` - existing rollback tests.
- `tests/testpilot_model_slice_test.go:214` - live run and callback sequencing.

## Verification

Require actual after-fill/same-invocation capture, wrong/missing types, failed Start, initial started=false, exact replay after progress/Reset, conflicting request bytes, overlapping crossed captures, wrong callback attempt and namespace/activity/run mismatches. Validate captured bytes without repairing them. Quantization needs an explicit request-field admission table; absent/zero wire timeout maps to None only through that table, never a generic Option coercion. The approved cancellation evidence must enter the same immutable receipt.

Publication controls first Explore the fixture World, reject another World's admission publication and retain old rows, then publish admission and require NotExplored until re-exploration under the selected World. Repeat native false/panicking publisher, oversized response and actual Wasm 16 MiB response-allocation exhaustion for replacement, Unsupported freeze and Reset. Rejected publication must retain every old field; successful retry commits once. Pin old absent-request bytes and fresh tag roundtrip only after schema authority and regeneration. Preserve ordinary malformed-observation precedence and privacy.

Quick: focused Temporal realization native/Go-host parity/allocation tests and interceptor/live slice tests using existing test_dep/integration tags where required; scoped formatting, strict clippy and changed-Go lint. Full: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`, rebuilt-guest `make umpire-check-model-slice` and full `make umpire-check-model-gaps`. The live retry worker must use actual attempt observations, fail retryably once and complete the next attempt, without weakening old controls. Retain corpus identities, privacy checks, allocation/publication evidence and timing. No new dependency/version, panic or core ABI change.

## Acceptance
- [ ] World built from observed start options; out-of-domain options Unsupported with the field named
- [ ] Live retry-then-complete run passes with attempt advancing
- [ ] Existing slice controls pass; umbrella unchanged beyond the slice
- [ ] Owner-approved actual request/result transport, identity/idempotent replay, whole-Runtime replacement, exploration invalidation, admission-specific Unsupported freeze and Reset all pass native and actual Wasm publication/allocation rollback controls. Failed/crossed evidence changes nothing; private values never enter diagnostics. No envelope or cancellation amendment is inferred from this task.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
