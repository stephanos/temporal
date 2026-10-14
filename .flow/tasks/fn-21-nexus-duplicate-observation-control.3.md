---
satisfies: [R3, R7]
---
# fn-21-nexus-duplicate-observation-control.3 Realize one participant-owned duplicate observation

## Description
Implement the exact Nexus participant negative-control realization for R3/R7 behind Task `.2`'s closed binding. It performs one real cancellation lifecycle and contributes one explicitly labeled duplicate observation, never a second request chain or synthetic history event.

**Size:** M
**Files:** `tools/umpire/temporal/nexus/participant.go`, `tools/umpire/temporal/nexus/workflow.go`, `tools/umpire/temporal/nexus/binding.go`, `tools/umpire/temporal/nexus/participant_test.go`, `tools/umpire/temporal/nexus/binding_test.go`
**Touches:** [tools/umpire/temporal/nexus/participant.go, tools/umpire/temporal/nexus/workflow.go, tools/umpire/temporal/nexus/binding.go, tools/umpire/temporal/nexus/participant_test.go, tools/umpire/temporal/nexus/binding_test.go]

### Approach
- Reuse fn-19's four-command participant and exact force-close binding; select the closed negative-control branch only from the already checked program/request, never from a CLI flag, environment value, or arbitrary argument.
- Pin the workflow Nexus cancellation mode to `WaitRequested`; retain exactly one normal lifecycle chain containing `NEXUS_OPERATION_CANCEL_REQUESTED` followed by `NEXUS_OPERATION_CANCEL_REQUEST_COMPLETED` and intercept the corresponding handler receipt/correlation.
- After the completed receipt, contribute exactly one synthetic observation before participant-output closure while preserving one force-close, one cancellation request chain, mechanical callback count one, and single-attempt semantics.
- Make activation idempotence structural: one run-scoped closed state admits one transition from completed real receipt to one injected contribution; retries/timing cannot activate it.
- Test normal/faulted branches and every participant/runtime row in the spec mutation table, including no readiness, rejection/failure/timeout/cancel, duplicate activation, second request chain, synthetic history mutation, wrong correlation/program/fault, and cleanup after every acquired resource.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.6.md:13-31` — current Nexus participant/binding contract
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md:55-59` — four commands and one force-close lifecycle
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md:65-86` — operational precedence, history, and source closure
- pinned Go SDK prototype.44.0 workflow Nexus cancellation-mode and history-event APIs
- pinned Nexus Go SDK v0.6.0 idempotent asynchronous `Operation.Cancel` contract

### Key context
`WaitRequested` completes through the ordinary requested/completed cancellation event pair. Keep the injected contribution at the participant observation edge and never mutate/count that history chain as two requests.

### Acceptance
- [ ] The faulted branch proves one completed real cancellation lifecycle before one and only one injected observation.
- [ ] Normal mode emits no marker/contribution and preserves existing behavior.
- [ ] No path issues a second force-close/request chain or synthetic history event; the normal requested/completed pair remains intact.
- [ ] Missing/failed/unbound real cancellation follows the exact operational/tooling status table and cannot emit a successful synthetic claim.
- [ ] Focused participant/binding tests prove deterministic activation and cleanup without a general injection surface.
## Acceptance
- [ ] R3 exact real-cancellation/injected-observation lifecycle is implemented and bounded.
- [ ] R7 no-framework/no-new-authority boundaries hold.

## Done summary
Implemented the exact checked duplicate-delivery Nexus branch: it retains one real WaitRequested cancellation lifecycle and emits one labeled synthetic participant observation only after successful completion, while normal, incomplete, failed, canceled, and repeated paths emit none. Activation is closed over the exact program/configuration identity, and duplicate delivery never issues a second force-close, cancellation request, or history mutation.

baseline: inherited red — the Lean Quick target and local-run Make targets are not present yet, while the untagged Go Quick commands fail on the Darwin /var-to-/private/var temporary-directory identity; these conditions existed before the task edit
verification: green — tagged Nexus aggregate suite, repeated focused lifecycle/package diagnostics, make lint-model (200/200), and git diff --check; make GOLANGCI_LINT_FIX=false lint-code remains inherited red with 1390 repository findings and no introduced diff finding, with all unrelated auto-fixes inverse-patched
review: SHIP with 0 introduced findings and R3/R7 covered; one capacity-only no-verdict transport attempt was refunded before the successful Codex review
stage: impl-review - ran [2026-08-31T07:23:52Z..2026-08-31T07:33:47Z]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7ff10a26973bbef797b89149cb97499bd5af1ecd
- Tests: baseline: red (cd model && mise exec -- lake build Temporal.Feature.Nexus.CallerClosureFaultTests failed pre-edit: target absent), baseline: red (go test -count=1 ./tools/umpire/temporal/nexus/... failed pre-edit: inherited Darwin /var symlink containment), baseline: red (go test -count=1 ./tools/umpire/runevaluation/... failed pre-edit: inherited Darwin /var symlink containment), baseline: red (make umpire-run-local SET=tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set OUTPUT_ROOT=/tmp/umpire-local-runs RUN_ID=caller-closure-duplicate-delivery failed pre-edit: Make target absent), baseline: red (make umpire-check-local-run-evaluation SET=/tmp/umpire-local-runs/caller-closure-duplicate-delivery OUTPUT_ROOT=/tmp/umpire-local-results failed pre-edit: prerequisite output/target unavailable), baseline: red (make umpire-check-regression failed pre-edit: inherited Darwin /var symlink containment), TMPDIR=$(cd -- "${TMPDIR:-/tmp}" && pwd -P) go test -tags test_dep -count=1 ./tools/umpire/temporal/nexus/..., TMPDIR=$(cd -- "${TMPDIR:-/tmp}" && pwd -P) go test -tags test_dep -count=10 -run TestRunCallerClosureDuplicateDelivery ./tools/umpire/temporal/nexus, TMPDIR=$(cd -- "${TMPDIR:-/tmp}" && pwd -P) go test -tags test_dep -count=5 ./tools/umpire/temporal/nexus, make lint-model, INHERITED_RED:make GOLANGCI_LINT_FIX=false lint-code - 1390 repository findings; no introduced diff finding; unrelated auto-edits inverse-patched, git diff --check
- PRs:
