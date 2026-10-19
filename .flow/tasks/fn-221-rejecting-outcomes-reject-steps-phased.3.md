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
Judge now explains successful activity cancellation with Accepted and closed cancellation with a one-step Rejected(NotFound) self-loop. Explanations carry every actual replayed outcome in parallel with steps, and final-outcome matching preserves non-Clone outcomes and facts.

R3 uses the permitted local umpire.temporal.v1.Error with code and message because descriptor-choice.log confirms google.rpc.Status is absent from proto/api.binpb. Both Go envelopes were regenerated with the pinned scripts. Existing handcrafted protocol byte vectors remain unchanged; additive wire controls pin outcomes at tag 2 and response/error at tags 2/3. The numeric status table remains in the Temporal binding. Projection, Answer, method numbers and ABI exports retain their contracts. The existing live helper only migrated to the oneof response wrapper; new live cancel scenarios remain in .4.

Native realization controls cover successful cancellation, repeated rejected self-loops after Completed/Failed/Canceled, actual World replay and outcome matching for all six phases including TimedOut, contradictory and unknown codes, unknown methods, missing/malformed/wrong response envelopes, freezing, reset and publication rollback. Generic runtime controls cover multi-step non-Clone outcomes, final-edge matching, zero-step None and Debug-formatted unit outcomes. The .1 invalid-Model refusal and transaction controls still pass. Go/WASM parity exercises these protocol and cancellation results through the transport-only generic host. The existing Judge method 4 now admits cancellation result evidence; no new ABI entrypoint was added.

baseline: green for make model-check, make umpire-check-model-slice, the configured direct Go selector and nonfixing Go lint. Logs are baseline-model.log, baseline-slice.log, baseline-live.log and baseline-lint.log. An initial nested-shell invocation failed before Make started because its cwd reset; the corrected invocation used an explicit absolute cd. The cancel reproduction failed before implementation because the method was unsupported (red-cancel.log); the outcome API control failed on the absent field (red-outcomes.log). The closed-state table addition caught a temporary any-Rejected mutation (red-closed-rejection-mutation.log), then passed after restoring exact matching (green-closed-rejection.log). The temporary mutation is absent from the committed diff.

Final make model-check, make umpire-check-model-slice, configured direct Go selector and nonfixing Go lint all passed. Logs are verify-model-final.log, verify-slice-final.log, verify-live-final.log and final-lint.log. The model gate covers formatting, clippy, docs, native fixtures, boundary checks, generated Go freshness and both WASM/Go hosts. Kettle and all existing fixture goldens remain unchanged. Final cumulative classification required full gates; the three green receipts at 31dc2c08 were then honored. All logs are under .flow/tmp/fn2213.

The first formal review produced three SHIP draws with no findings, followed by a same-receipt single SHIP re-review of the closed-state coverage addition. Every dispatch explicitly ran codex:gpt-6.1-sol:high, from the writer's model family. Reviewer Cargo execution was blocked by the read-only build lock; the worker's gate logs supply execution evidence. The receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-221-rejecting-outcomes-reject-steps-phased.3.json and fanout evidence is .flow/review-fanout/ed1c820997c84c5997c1c932526c2d2a/. The conductor's fresh correctness and standards audits each found zero issues at ce4fda11; .flow/tmp/fn2213/quality-audits.md includes its separate review of the test-only 31dc2c08 addition.

GATE_SKIPPED:unittest:green-receipt 31dc2c08 - verified post-change model gate reused at final verification
GATE_SKIPPED:smoke:green-receipt 31dc2c08 - verified post-change slice gate reused at final verification
GATE_SKIPPED:live:green-receipt 31dc2c08 - verified configured direct Go selector reused at final verification

stage: impl-review - ran [2026-10-19T21:34:00Z..2026-10-19T21:38:12Z] (three SHIP draws and one single SHIP re-review; model: gpt-6.1-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tier: session (jev-unavailable(no_key))
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: ce4fda1129325a2de37a14ca692b977650474402, 31dc2c08ff17ac0cb28ac969409131f4eae7e3b7
- Tests: make model-check, make umpire-check-model-slice, UMPIRE_TEMPORAL_WASM=/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_temporal_guest.wasm CGO_ENABLED=0 mise exec -- go test -tags 'test_dep integration' ./tests -run '^TestTestpilotModelSlice$' -count=1, GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=c7ef6d8a62 make lint-code-fast, GATE_SKIPPED:unittest:green-receipt 31dc2c08 - verified post-change model gate reused at final verification, GATE_SKIPPED:smoke:green-receipt 31dc2c08 - verified post-change slice gate reused at final verification, GATE_SKIPPED:live:green-receipt 31dc2c08 - verified configured direct Go selector reused at final verification
- PRs: