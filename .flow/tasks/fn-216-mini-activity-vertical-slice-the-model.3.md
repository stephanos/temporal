---
satisfies: [R3]
---
# fn-216-mini-activity-vertical-slice-the-model.3 Go test: in-process server, SDK worker answering from the Model, observations judged

## Description
Go test in `common/testing/testpilot/temporal` (name indicative `model_slice_test.go`): in-process server from the existing test infrastructure; the existing SDK worker path registering one activity whose body builds the `Callback` from `activity.GetInfo(ctx)`, calls the host's `Answer`, and translates the `Reply` (a `RespondActivityTaskCompletedRequest` becomes the return value, a `RespondActivityTaskFailedRequest` a non-retryable `ApplicationError`), the SDK performing the real RPC; `StartActivityExecution` via the client with `maximumAttempts = 1`; `DescribeActivityExecution` polled until a closed status, each response passed to `Judge` as `Observation { method, response }`; scenarios completed and failed; assert all observations `Explained` and the final phase; a mutated observation is `Unexplained` naming the expected actions. Runs with `CGO_ENABLED=0` and the Temporal guest built from `model/temporal/wasm` in the Testpilot gate (not in `make model-check`). After .2. **Touches:** `common/testing/testpilot/temporal/**` (test only), Makefile Testpilot section (exact minimal replacement).
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
