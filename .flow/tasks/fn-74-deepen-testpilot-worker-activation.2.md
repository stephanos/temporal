---
satisfies: [R2, R6, R8]
---
# fn-74-deepen-testpilot-worker-activation.2 Integrate activation ownership into both SDK interpreters

## Description
Use task 1's activation owner in workflow and Nexus-handler interpretation (R2, R6, R8), with regression and ownership documentation updates.

**Size:** M
**Files:** `common/testing/testpilot/temporal/worker/interpreter.go`, `values.go`, `sdk_test.go`, relevant worker/session tests and README; `common/testing/testpilot/internal/execution/README.md`.
**Touches:** [common/testing/testpilot/temporal/worker/**, common/testing/testpilot/internal/execution/README.md, common/testing/testpilot/temporal/internal/activation/activation_test.go]

### Approach
- Instantiate state only for an actual interpretation. Replace manual maps, budget subtraction and separate validation/storage calls with task 1's operations. Retire superseded worker helpers only once all consumers migrate, preserving relevant comments.
- Keep prepared DAG traversal, SDK dispatch/futures/timeouts, terminal outcomes, reservation/redelivery cache and completion capability authority in their current owners.
- Preserve workflow SDK-context checks and bounded pure evaluation with the existing context strategy; canceled SDK outcomes must still follow existing outcome-validation paths. Add no wall-clock workflow deadlines or operation cancellation capability.
- Adapt direct interpreter test construction to evaluate before admission, especially Await timeout setup; preserve every existing outcome/timeout assertion. Retain real WorkflowReplayer and Stop-race coverage.
- Move the former private Slot-reference rejection assertion into the activation owner's existing ownership test; do not expose a lookup method merely to retain a worker-package assertion.
- Update worker and execution documentation to explain private state composition while retaining public low-level work-charge contracts.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/worker/interpreter.go:33` — workflow traversal and SDK boundaries.
- `common/testing/testpilot/temporal/worker/interpreter.go:138` — Nexus redelivery and fresh interpretation.
- `common/testing/testpilot/temporal/worker/sdk_test.go:75` — replay; Stop race and Await timeout later in the same file.
- `common/testing/testpilot/temporal/worker/session_test.go:123` — opaque authority and publication.
- `common/testing/testpilot/temporal/worker/README.md:3` — adapter ownership.
- `common/testing/testpilot/internal/execution/README.md:131` — returned work versus composition ownership.

### Quick commands
`go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot`
`go test -race -tags test_dep ./common/testing/testpilot/temporal/...`

## Acceptance
- [ ] Both production interpreters use the private module for all R2 responsibilities; SDK and delivery authorities remain separate.
- [ ] R6 replay, Start/Await/Finish, Nexus Respond, Stop/timeout/late-publication and session regressions pass with existing assertions retained.
- [ ] No public signature, Case/protocol byte, evaluator, controller store, retry or cleanup behavior changes under R8.
- [ ] Worker/execution documentation describes the final composition and unchanged work units; relevant existing comments are preserved.
- [ ] Focused unit/race checks pass. Whichever task completes last also runs the combined facade/Driver suites and `make lint-code GOLANGCI_LINT_FIX=false`, reporting exact inherited failures separately.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: