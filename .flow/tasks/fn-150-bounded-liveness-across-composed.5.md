---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-150-bounded-liveness-across-composed.5 Demonstrate Temporal composition progress and close gates

## Description
Demonstrate Temporal composition progress and close gates. Advances R1, R2, R3, R4, R5 of the parent spec.

**Size:** M
**Files:** `model/temporal/features/activity/standalone/system/**`, `model/ir/**`, `model/cases/**`, `model/README.md`, `model/SEMANTICS.md`, `tools/umpire/README.md`, `tools/umpire/check/*test.go`
**Touches:** [model/temporal/features/activity/standalone/system/**, model/ir/**, model/cases/**, model/README.md, model/SEMANTICS.md, tools/umpire/README.md, tools/umpire/check/*test.go]

### Approach
- Use an existing Temporal composition such as StandaloneActivity as the base; choose a small explicitly restricted design with a reachable two-member obligation and a structural bound. The positive result must exercise from and must not be from-already-to.
- Show a safety promise and bounded progress promise, plus a faulty or missing-prerequisite variant. Explain the restricted actions/start conditions rather than asserting that weak fairness bounds response time. Preserve existing product Models; a disagreement with implementation goes to the owner.
- Document composed step counting, typed fairness selection, inherited/replacement assumptions, supported/unsupported cases and model-only scope. Group claims per fn-149 only when that authoring contract has landed.
- Regenerate once after integrating task 3 and task 4. Review new model receipts and any generated changes; do not add live composition Cases or alter historical Run evidence.
- Run the existing model/Case/lint/Go gates once under the shared lock; retain focused evidence and full-suite JSON timing. No new CI or generated-API drift gate.

### Investigation targets
**Required:**
- `model/temporal/features/activity/standalone/system/System.scala:589` - existing composition.
- `model/SEMANTICS.md:403` - composed state/start semantics; Progress section.
- `model/README.md` - authoring and known limitations.
- `tools/umpire/check/composed_test.go` - model composition checks.
- `MILESTONES.md` - gate serialization.

### Key context
Re-anchor paths and interfaces against completed fn-140/fn-141 and the approved schema/package moves before editing. Keep this work outside the activity batch and serialize shared regeneration with it and the schema chain; no new spec-close dependency is implied.

### Quick commands
```bash
make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-cases umpire-check-fixtures canary-check-case
make lint-model
make lint-code-fast
go test -tags test_dep -p 2 -timeout 30m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...
```

## Acceptance
- [ ] A concrete Temporal example exercises both member predicates, passes under an explained structural bound and fails under its deliberate negative variant.
- [ ] Docs and receipts explain exact assumptions, composed steps, unsupported monitor/runtime cases and vacuity.
- [ ] Existing behavior pins, generated artifacts and required integrated gates pass; record evidence for all five requirements.

## Done summary
`model/temporal/features/activity/standalone/system/DispatchProgress.scala`: ServedDispatch composes
the activity restricted to start/poll/respondCompleted with a worker restricted to serve; claim
`startsWithinOneStep` (from: scheduled, dispatch now, worker polling -- both members; to: started;
within 1; fair dispatch sync) passes all three parts, exercised. UnservedDispatch (worker may stop,
NegativeControl) fails it: deadlock and deadline counterexamples whose witness ends in `worker_stop`,
with the activity's poll locally enabled and the sync disabled. Safety Query (only a polling worker
starts an attempt) verifies over both. Rules use qualified `on(actor.action)`/`when(Phase.x)`; roots
added to `activity-standalone`'s exports. Docs: SEMANTICS.md Progress (composed counting, fairness
selection, inheritance/replacement, refusals, unsupported monitored members), README authoring
paragraph, tools/umpire/README. model/ir not regenerated (Batch 5b); verified by lifting
`activity-standalone` into `.flow/tmp/fn150/ir/` and running TestDispatchProgress... with
UMPIRE_MODEL_IR pointing there. No Case or realization; composition progress stays model-only.
Deferred to Batch 5b (needs regenerated IR): `make umpire-gen-model`, `umpire-check-model`,
`umpire-check-cases/fixtures`, `canary-check-case`, `lint-model`, full Go suite, and
TestDispatchProgressHoldsWithAServingWorkerAndNotWithAStoppingOne against the checked-in model/ir
(it fails on today's stale IR).

Blocked:
Waits for fn-173 Phase 1 (fn-173-reachable-exploration-drop-full-tables.2), owner-approved 2026-10-13. This task's Go work is built on eager full tables, catalog-wide checks or identities that Phase 1 replaces (on-demand rows from roots, reach bound, IR structural fingerprint). Write it against the reachable graph once Phase 1 lands, then reset this task to todo. A cross-spec task dependency isn't possible in Flow, so this block stands in for it. See the "Redesign impact" note in this spec and `model/design/reachable-exploration.md`.
## Evidence
- Commits:
- Tests:
- PRs:
