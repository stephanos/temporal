---
satisfies: [R3, R5]
---
# fn-217-annotation-catalog-over-the-temporal.3 Mini activity catalog with expected errors; fill and validate in the slice test

## Description
`model/temporal/features/activity/api.rs`: the complete catalog for every message fn-216 touches (`StartActivityExecutionRequest`, `DescribeActivityExecutionResponse`, `PollActivityTaskQueueResponse`, `RespondActivityTaskCompletedRequest`, `RespondActivityTaskFailedRequest`) and the `WorkflowService` method classification; every request-side rule and constraint with its `Expectation` restated from the server's validator (`covers:` names the Go function); Temporal tags (`Visibility { lag_polls }`) defined here if used. Wire `fill` (request_id, inputs) and `validate` (every request sent, every response observed) into fn-216.3's slice test; `judge` honours `Visibility` if declared. After .2. **Touches:** `model/temporal/features/activity/**`, `model/temporal/guest/**`, `model/temporal/realization/**`, `model/temporal/api/**`, `model/temporal/proto/**`, `model/temporal/hosts/go/**`, `model/umpire/wasm/**`, `model/umpire/proto/**`, `model/umpire/hosts/go/**`, `tests/testpilot_model_slice_test.go`, `model/Cargo.toml`, `model/Cargo.lock`.

The live slice actually lives in `tests/testpilot_model_slice_test.go`. The sole catalog and validator/filler implementation must remain in Rust; provide a thin typed guest/host route if needed to apply it before Go SDK transport and to validate returned messages. Do not duplicate rules in Go or claim Rust-only fixture validation covers live requests. Any protocol addition must preserve existing wire contracts and reserve method 6 for fn-228's Propose. Request expectations come from the actual `chasm/lib/activity` and frontend validators, never from assumed semantic kinds.

**Quick:** `cd model && mise exec -- cargo test -p umpire-temporal -p umpire-temporal-tests`; `make umpire-check-model-slice`.
**Full gates:** `make model-check`; `make model-check UMPIRE_BACKENDS=require`.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
