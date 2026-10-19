---
satisfies: [R3]
---
# fn-221-rejecting-outcomes-reject-steps-phased.3 Protocol and realization: Explanation.outcomes, matches with outcome, error observations and the cancel projection

## Description
Extend `umpire.model.v1.Explanation` with `repeated string outcomes = 2`; pass the replayed edge outcome to `Realize::matches`; make `umpire.temporal.v1.Observation.result` a `oneof` of `response` (tag 2) and `error` (tag 3). Use `google.rpc.Status` only if already in the upstream descriptor closure, otherwise the spec's local two-field `Error { int32 code = 1; string message = 2; }`; revalidate the closure and record the choice. In the current activity `temporal.rs`, recognize `RequestCancelActivityExecution`: success matches `Accepted`, NotFound matches `Rejected(NotFound)`, and other status codes are Unexplained. Keep the numeric gRPC code table in the Temporal binding only.

**Size:** M
**Touches:** [model/umpire/proto/**, model/umpire/wasm/src/**, model/umpire/wasm/tests/**, model/umpire/hosts/go/**, model/temporal/proto/**, model/temporal/api/build.rs, model/temporal/features/activity/temporal.rs, model/temporal/realization/**, model/temporal/hosts/go/**, tests/testpilot_model_slice_test.go]

### Approach
- Build on fn-216.4's direct generated `activity::World`; do not restore a delegating adapter. Current path is `features/activity`, not archived `activity_mini`.
- Use the current standard formatting contract: outcomes are `Debug`-formatted, including `()` for unit outcomes; states/actions use `Display`. No removed `Spell` trait or compatibility alias.
- `Realize::matches` receives `None` only for a zero-step current-state check; candidate replay supplies its actual final edge outcome. Collect all replayed outcomes in parallel with steps; `Explanation.outcomes.len() == steps.len()`, including multi-step paths. Do not synthesize outcomes from expected system status.
- Preserve the existing `Machine::Outcome: Debug` contract; carrying the replayed outcome must not add a `Clone`/`Copy` bound to arbitrary manual outcomes or facts. Move the chosen edge outcome as needed and retain a non-Clone outcome replay control.
- Preserve checked-step Model refusal propagation and transaction/publication rollback from .1. A rejecting self-loop is a real explained step despite unchanged state. Contradictory recognized observations become Unexplained and freeze the run; unsupported methods refuse Unsupported without publishing state changes. Missing or malformed envelopes refuse MalformedRequest.
- Native realization controls cover successful cancel from Scheduled, NotFound cancel after each relevant closure, incorrect success/rejection combinations, unknown error code, unknown method, missing result, wrong response Any, and frozen-run behavior. Preserve Describe behavior, callback delivery/reply behavior, `Answer`, Projection and ABI method numbers.
- Keep existing handcrafted pinned wire expectations unchanged; additive fields remain absent when unset and old response tag 2 decodes through the new oneof. Add new wire cases rather than replacing old expected bytes. Runtime explanations intentionally now carry outcomes; test that contract separately.
- Regenerate both core and Temporal Go envelopes through the existing pinned scripts `model/scripts/proto-go.sh` and `proto-temporal-go.sh`; do not hand-edit generated files. Generic host only carries `Any` and must not import Temporal types. Verify native/WASM/Go parity and boundary checks.
- Mechanically migrate the existing live-slice observation constructor to the generated oneof response wrapper in this task, so the unchanged existing scenarios keep compiling and remain the R3 gate. New live cancel scenarios still belong to .4; do not implement them here.
- Preflight notes and descriptor inspection: `.flow/tmp/fn2213-preflight.md`. Revalidate before relying on them after .2 integration.
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
