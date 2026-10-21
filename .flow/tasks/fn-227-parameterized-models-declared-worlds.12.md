---
satisfies: [R6]
---
# fn-227-parameterized-models-declared-worlds.12 Actual-request Observation transport and owned same-invocation capture

## Description
Implement only the approved request-evidence transport and capture slice of R6. Expose an owned actual after-fill request/raw-result pair to the existing Temporal observation consumer; live World admission remains .5. This is transport checkpoint T, allocated as fn-227-parameterized-models-declared-worlds.12. Owner approval at `bbf0835fbf` authorizes optional Observation.request at tag 4 and no other envelope/core ABI change.

**Size:** M. One existing-envelope/interceptor adaptation with wire equivalence and capture ownership controls. Generated protocol output is mechanical. No admission state machine, pairing cache or new endpoint.
**Dependencies:** No new dependency on .4 or .5. Existing parent prerequisites supply catalog/annotations; use completed fn217/fn231. Shared generated/guest ownership requires serial scheduling, not fabricated model dependencies.
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
