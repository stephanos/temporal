---
satisfies: [R3, R5, R6, R7]
---
# fn-19-bounded-local-temporal-execution-and.7 Capture closed causal evidence and build one live output set

## Description
Complete R3/R5/R6's live operational proof by projecting allowlisted SDK/history/control/cleanup facts and returning the exact admitted four-member output set in memory.

**Size:** M
**Files:** `tools/umpire/temporal/nexus/evidence.go`, `tools/umpire/temporal/nexus/output.go`, `tools/umpire/temporal/nexus/evidence_test.go`, `tools/umpire/temporal/nexus/integration_test.go`, `tools/umpire/temporal/nexus/testdata/**`
**Touches:** [tools/umpire/temporal/nexus/evidence.go, tools/umpire/temporal/nexus/output.go, tools/umpire/temporal/nexus/evidence_test.go, tools/umpire/temporal/nexus/integration_test.go, tools/umpire/temporal/nexus/testdata/**]

### Approach
- Iterate complete terminal caller history with explicit context and project only allowlisted event/correlation fields; close four exact sources under parent limits.
- Capture participant cancellation count, control request/receipt, open-resource counts, sanitized codes, and digest/redact dispositions without semantic names or raw payload/header/error leakage.
- Build and fn-18-admit the exact ExperimentRun/RawEvidence/output manifest over the two immutable inputs and return it in memory. Production code in this task never publishes; the integration test may call fn-18 `PublishSet` directly only as its harness assertion.
- Exercise field/disposition/order/gap/causal/binding/terminal-history/control/cleanup mutations before the live control.
- Run one bounded real LiteServer caller-closure case and assert operational artifacts/closure, never Property truth.

### Investigation targets
**Required** (read before coding):
- Task `.3` accumulator and `.6` participant receipts
- Go SDK `GetWorkflowHistory` iterator contract
- fn-18 Run/RawEvidence/Set/Publish exact validators
- fn-4 synthetic evidence field schemas only for structural interoperability

### Acceptance
- [ ] Closed live evidence has exactly four sources, gapless source ordinals, complete causal/reference closure, terminal history, one control receipt, and zero open handles.
- [ ] Every corruption/capacity/partial-history case is explicit and cannot be mislabeled closed/succeeded.
- [ ] Headers/payloads/raw errors/authority values cannot enter retained fields.
- [ ] The admitted in-memory set contains exactly ExperimentSpec, RuntimeConfiguration, ExperimentRun, and RawEvidence; its integration harness can publish/reopen it through fn-18 exactly once.

## Acceptance
- [ ] R3/R5/R6 one-live-run evidence and admitted output set are complete.
- [ ] Negative mutations pass before the live test.
- [ ] No semantic evidence, Result, or Run Evaluation claim exists.

## Done summary
Implemented the bounded Nexus terminal-history projector and admitted four-member in-memory output, with exact four-source closure, allowlisted mechanical fields, mutation coverage, and one live publish/reopen harness. The review fix preserves every valid failure prefix and reports N+1 history as an explicit capacity gap with partial capture and incomplete operational status.

Verification is green for runtime, local, Nexus, temporaltest, and Lean LocalProfile. The missing `umpire-local-run` package, `Temporal.Feature.Nexus.ExecutionTests` target, and `umpire-run-local` Make target were red before this task and remain inherited later-task surfaces; memory capture was skipped because flow memory is not initialized.

baseline: green for implemented dependency surfaces; inherited red for the three later-task Quick entries listed above

stage: impl-review - ran [2026-08-29T18:42:35Z..2026-08-29T19:05:06Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 040a250cc30e2a6cdb56f61949badeed8605779d, 17f4bea7fb6ce24cd298d38f73f2e498f6a9cebe
- Tests: go test -count=1 ./tools/umpire/runtime/..., go test -count=1 ./tools/umpire/temporal/local/..., go test -count=1 ./tools/umpire/temporal/nexus/..., go test -count=1 ./temporaltest/..., cd model && mise exec -- lake build Temporal.System.Execution.LocalProfileTests, git diff --check -- tools/umpire/runtime tools/umpire/temporal/nexus, INHERITED_RED:go test -count=1 ./tools/umpire/cmd/umpire-local-run/... - package absent before edit and after verification, INHERITED_RED:cd model && mise exec -- lake build Temporal.Feature.Nexus.ExecutionTests - target absent before edit and after verification, INHERITED_RED:make umpire-run-local SET=tools/umpire/temporal/nexus/testdata/caller-closure-input-set OUTPUT_ROOT=/tmp/umpire-local-runs RUN_ID=umpire.local.caller-closure.run-1 - target absent before edit and after verification
- PRs:
