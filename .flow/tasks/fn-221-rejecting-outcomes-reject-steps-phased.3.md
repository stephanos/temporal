---
satisfies: [R3]
---
# fn-221-rejecting-outcomes-reject-steps-phased.3 Protocol and realization: Explanation.outcomes, matches with outcome, error observations and the cancel projection

## Description
Extend `umpire.model.v1.Explanation` with `repeated string outcomes = 2`; pass the replayed outcome to `Realize::matches`; make `umpire.temporal.v1.Observation.result` a `oneof` of `response` (tag 2) and `error` (`google.rpc.Status` if in the closure, else a two-field `Error`; record which). In `temporal.rs`, recognize `RequestCancelActivityExecution`: success matches `Accepted`, NotFound matches `Rejected(NotFound)`, other codes are Unexplained; one gRPC code table. Native realization tests cover both, the freeze on a contradicting code, Unsupported on an unknown method, and byte-identical old messages.

**Size:** M
**Touches:** [model/umpire/proto/**, model/umpire/wasm/src/**, model/umpire/wasm/tests/**, model/temporal/proto/**, model/temporal/api/build.rs, model/temporal/features/activity_mini/temporal.rs, model/temporal/realization/**, model/temporal/hosts/go/**]

### Approach
- Build on the fn-216.4 cleanup; do not reintroduce a delegating world.
- Regenerate the Go envelopes; the generic Go host only carries `Any`.
- Keep Describe projection and `Answer` unchanged.

## Acceptance
- [ ] Judge explains NotFound-after-close with one step and `Rejected(NotFound)`, phase unchanged; successful cancel in Scheduled reaches Canceled.
- [ ] Old wire controls and Go host parity pass; pinned bytes identical.
- [ ] Boundary gate passes with the code table in `model/temporal`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
