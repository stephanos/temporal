---
satisfies: [R6]
---
# fn-227-parameterized-models-declared-worlds.12 Actual-request Observation transport and owned same-invocation capture

## Description
Implement only the approved request-evidence transport and capture slice of R6. Expose an owned actual after-fill request/raw-result pair to the existing Temporal observation consumer; live World admission remains .5. This is transport checkpoint T, allocated as fn-227-parameterized-models-declared-worlds.12. Owner approval at `bbf0835fbf` authorizes optional Observation.request at tag 4 and no other envelope/core ABI change.

**Size:** M. One existing-envelope/interceptor adaptation with wire equivalence and capture ownership controls. Generated protocol output is mechanical. No admission state machine, pairing cache or new endpoint.
**Dependencies:** No new dependency on .5 (or on the former .4, now fn-237.3). Existing parent prerequisites supply catalog/annotations; use completed fn217/fn231. Shared generated/guest ownership requires serial scheduling, not fabricated model dependencies.
**Files:** `model/temporal/proto/umpire/temporal/v1/temporal.proto`; its existing generated Rust/Go consumers through current build/generation targets; `model/temporal/realization/src/support.rs` and `annotations.rs` only for request transport projection; `model/temporal/realization/tests/realization.rs`; `model/temporal/hosts/go/parity_test.go`; `tests/testpilot_model_annotations_test.go`; existing annotation authoring docs and `model/README.md`; committed Temporal guest when its linked bytes change.
**Touches:** [model/temporal/proto/**, model/temporal/hosts/go/**, model/temporal/realization/src/support.rs, model/temporal/realization/src/annotations.rs, model/temporal/realization/tests/realization.rs, model/temporal/realization/README.md, tests/testpilot_model_annotations_test.go, model/README.md]

### Approach

1. Pin existing request-absent wire payloads and field numbers. Add optional request Any at tag 4, regenerate through existing Temporal targets and preserve method/result tags 1/2/3, Callback, Reply, the three-envelope count and core oracle. Keep absent-request consumers behavior-compatible until .5 installs live admission.
2. At the existing SDK after-fill interceptor, clone the actual completed request immediately before invoke and the successful raw result immediately after that same invoke. Preserve raw RPC error separately from later local response-validation failure. Verify the innermost invoker sees the exact captured message; later interceptors/caller aliases cannot mutate the capture.
3. Associate captures through the invocation's lexical ownership. Test overlapping calls whose responses complete in reverse order. Preserve filled request ID across retries. No global last-request slot, successive Annotate pairing cache, intended-options reconstruction or opaque extra envelope.
4. Carry actual requests for subsequent observed RPCs through the same approved slot. .5 owns namespace/activity/run admission and the immutable execution receipt. Same-invocation association is a trusted transport-owner assertion; Start result does not echo/authenticate request ID. Preserve raw evidence and redact only report copies.

### Investigation targets

Required:

- `.flow/artifacts/fn-227-parameterized-models-declared-worlds/start-admission-contract.md` for capture trust and wire restrictions, interpreting its former owner-pending language under the approved parent.
- `model/temporal/proto/umpire/temporal/v1/temporal.proto` for the current envelope.
- `model/temporal/realization/src/support.rs` for owned projection.
- `tests/testpilot_model_annotations_test.go` for after-fill invocation and retry controls.
- `model/temporal/hosts/go/parity_test.go` and `model/temporal/realization/tests/realization.rs` for exact wire/typed roundtrip pins.

### Verification

Quick: `mise exec -- cargo test --manifest-path temporal/realization/Cargo.toml` from model/; `go test -tags test_dep ./...` from `model/temporal/hosts/go`; from repository root run `CGO_ENABLED=0 mise exec -- go test -v -count=1 -timeout 2m -tags 'test_dep integration' ./tests -run '^TestTestpilotModelSlice/Annotations$'`. The existing Annotations subtest invokes `modelSliceAnnotationControls`; extend that control harness instead of inventing a separate cluster runner. Run scoped Rust fmt/strict clippy and changed-Go lint.

Full from root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`; regenerate the Temporal Go closure using `bash model/scripts/proto-temporal-go.sh`, refresh the embedded guest through `make umpire-guest`, then run `make umpire-check-model-slice` and `make umpire-check-model-gaps`. Retain exact wire baselines, capture/privacy controls and actual timings. No new dependency/version or panic/ABI policy.
## Acceptance

- [ ] The approved tag roundtrips typed actual requests, old absent-request bytes and field numbers are exact, and Callback/Reply/core oracle remain unchanged. Generated protocol freshness passes.
- [ ] Capture equals the after-fill request seen by the innermost invoker and that invocation's raw result. Caller/later-interceptor mutations cannot alter the owned pair; raw RPC error remains distinct from a local validation error.
- [ ] Overlapping reversed responses and retry/request-ID controls preserve association and ownership without hidden caches. Later RPC observations carry their actual requests.
- [ ] Existing annotation/parity/live controls and privacy checks pass using refreshed guest bytes. This task installs no World or live receipt and claims no .5 admission result.
## Done summary
Observation carries the optional actual request `Any` at fresh tag 4; tags 1-3, request-absent bytes, Callback, Reply, the three-envelope count and the core oracle are unchanged (Go and prost field orders both pinned as decodable). `Observed` keeps the request with typed `request<T>()` decoding; current projections ignore it, so judgments are byte-identical with or without one. A caller-owned context sink records one capture per attempt: request cloned after fill and before invoke, raw result cloned after invoke, RPC errors kept apart from local validation errors, post-capture mutation refused while keeping the raw result (review L1). Controls cover overlap, retries and mutation. The live slice transports the Start pair and checks it against the SDK handle; no World admission (that is .5).

Built in worktree `work/fn22712-request` and merged as ccacbf4b37 with the guest rebuilt on the merged tree (3,332,311 bytes). Review: SHIP (.flow/tmp/fn22712-review.md). Receipt: the filtered `Annotations` command cannot pass because its parent cleanup needs coverage only scenario subtests produce (pre-existing since fn-217.3); the full live slice is the receipt. Full gates on the merged tree: `UMPIRE_BACKENDS=require make model-check` 349s, `make model-time` 53.993s gate / 7.088s inner loop, slice 7s, gaps 364s, full live slice green.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 156b02d165, c108121fe1, f25b1e9a38, ccacbf4b37
- Tests: UMPIRE_BACKENDS=require make model-check, make model-time, make umpire-check-model-slice, make umpire-check-model-gaps, go test -tags test_dep,integration ./tests -run ^TestTestpilotModelSlice$
- PRs: