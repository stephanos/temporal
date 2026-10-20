---
satisfies: [R2, R3]
---
# fn-229-realization-ergonomics-shared-temporal.2 After .1: support module with Temporal helpers; activity Realization rewritten; Phase TryFrom beside the Model

## Description
Create the realization crate's shared support module with Observed (method plus typed
response or gRPC code), rejection(code), service-based RPC method naming,
plain_text(&Payload), Failure::application, reply(&impl Name), and callback::<T>.
Reuse existing Any::pack/unpack/required and prost::Name, without new dependencies.
Use an extension trait for Failure::application because the generated Failure type
belongs to another crate. Service-based method naming may use a typed Service marker;
do not infer RPC/service identity from response message names.

Rewrite features/activity/temporal.rs against these helpers and .1's constructors.
Put TryFrom<ActivityExecutionInfo> for Phase in a realization-only sibling phase.rs,
included by temporal.rs, never inside model.rs: the independent fixture compilation
and source lifting must retain their existing imports and bytes. Import actor enums,
clone the echo Payload, and keep the existing model.rs path inclusion with its short
reason because the fixture runner compiles the Model separately.

Preserve refusal precedence: missing outer type URL is Malformed, wrong type Unsupported;
unknown method with missing result stays Unsupported; Describe missing/error result is
Malformed and wrong response type Unsupported; Cancel wrong response type stays Malformed.
plain_text checks missing encoding, then extra metadata, then unsupported encoding.
Unknown gRPC codes remain Unexplained (no candidate), never unconstrained Cancel.
Keep all six valid phase pairs and every existing invalid pair/unknown-enum refusal.
Unit controls must actually run: the library has cfg(not(test)), so integration tests
under realization/tests/support.rs are the straightforward route.

The activity Realization body is at most eighty lines excluding imports/docs. Preserve
every activity golden byte, wire bytes, host parity and all six live-slice scenarios,
including CancelAfterCompletion and CancelBeforeWorkerStart. Record measured before/after
counts and model-time. No Model behavior or protocol changes.

**Touches:** `model/temporal/realization/**`, `model/temporal/features/activity/temporal.rs`, `model/temporal/features/activity/phase.rs`
**Quick:** `cd model && mise exec -- cargo test -p umpire-temporal`; `cd model && mise exec -- cargo clippy -p umpire-temporal --all-targets -- -D warnings`
**Full gates:** `make model-check`; `make umpire-check-model-slice`; `make model-time`.
Use a lane-specific Cargo target to avoid embedding another worktree's manifest paths.
## Acceptance
- [ ] support module with the seven helpers and refusal controls; no activity names
- [ ] Activity Realization rewritten, at most eighty lines of body
- [ ] Activity goldens byte-identical; slice, parity and cover tests pass; model-time pasted

## Done summary
Shared Temporal helpers and the activity binding rewrite are integrated, with behavior-preserving payload normalization restored after review.

Task: fn-229-realization-ergonomics-shared-temporal.2
Worker source: 103209f5528a74444bf57090d0ae387669e5f40f.
Review regression: aeec53b3f771c93f2b67b9d630556741f2bec8cb (red); fix: 351191e5d811b469bfe4c56243aed69d663396b2 (green).
Integrated HEAD: 0a30d59b1df98dafda200153520c98e5cd22fce8.

The seven shared helpers contain no activity-specific names. Phase conversion is realization-only, outside the independent Model. The binding body is 80 physical lines excluding documentation and imports. Complete replies clone and explicitly clear external payload references, retaining the previous metadata/data-only reply behavior. All 19 focused controls and clippy passed in the isolated worker. Default model-time after the fix: gate 15.443s, inner loop 1.883s; the earlier 44.704s soft-budget overrun remains documented in the worker evidence.

Conductor ran CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check and make umpire-check-model-slice at the integrated HEAD: both exit 0; all six live scenarios passed. The activity Model, goldens/generated output and protocol trees are unchanged against the actual integration base 0da21c3a639887dc86da3e80b82dc438bef66e36.

stage: wave-join - ran (isolated worker commits merged; integrated gates green)
stage: impl-review - ran (first fanout NEEDS_WORK; accepted external_payloads regression fixed; resumed single review SHIP; receipt /tmp/impl-review-receipt-8f37faba39e2-fn-229-realization-ergonomics-shared-temporal.2.json; model: gpt-6.1-sol high, same family as writer)
stage: plan-sync - skipped(config: disabled)

Per-task review is SHIP, not a spec-completion claim. Whole-spec quality audits and completion review remain conductor-owned.
## Evidence
- Commits: 103209f5528a74444bf57090d0ae387669e5f40f, aeec53b3f771c93f2b67b9d630556741f2bec8cb, 351191e5d811b469bfe4c56243aed69d663396b2, 0a30d59b1df98dafda200153520c98e5cd22fce8
- Tests: worker baseline: green; focused test and clippy, worker regression aeec53b3f7: complete_reply_drops_external_payload_references red (exit 101), worker fix 351191e5d8: all 19 focused controls and clippy green, worker fix 351191e5d8: make model-check and all six live Model slice scenarios passed, worker default make model-time: gate 15.443s; inner 1.883s, integrated 0a30d59b1d: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check (exit 0), integrated 0a30d59b1d: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make umpire-check-model-slice (exit 0; six scenarios), integrated git diff --exit-code 0da21c3a639887dc86da3e80b82dc438bef66e36 HEAD -- activity Model, expected/generated trees and both protocol trees (exit 0), codex resumed implementation review: SHIP; prior finding fixed; receipt /tmp/impl-review-receipt-8f37faba39e2-fn-229-realization-ergonomics-shared-temporal.2.json
- PRs: