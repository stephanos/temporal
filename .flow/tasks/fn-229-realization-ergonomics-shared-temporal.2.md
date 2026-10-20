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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
