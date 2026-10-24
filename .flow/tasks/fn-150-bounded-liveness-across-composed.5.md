---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-150-bounded-liveness-across-composed.5 Demonstrate Temporal composition progress and close gates

## Description
**Recut 2026-10-18 (one-language series):** Gates run on the Scala checker; the composed progress example verifies natively (fn-199).

Demonstrate Temporal composition progress and close gates. Advances R1, R2, R3, R4, R5 of the parent spec.

**Size:** M
**Files:** `model/temporal/features/activity/standalone/system/**`, `model/ir/**`, `model/cases/**`, `model/README.md`, `model/framework/SEMANTICS.md`, `tools/umpire/README.md`, `tools/umpire/check/*test.go`
**Touches:** [model/temporal/features/activity/standalone/system/**, model/ir/**, model/cases/**, model/README.md, model/framework/SEMANTICS.md, tools/umpire/README.md, tools/umpire/check/*test.go]

### Approach
- Use an existing Temporal composition such as StandaloneActivity as the base; choose a small explicitly restricted design with a reachable two-member obligation and a structural bound. The positive result must exercise from and must not be from-already-to.
- Show a safety promise and bounded progress promise, plus a faulty or missing-prerequisite variant. Explain the restricted actions/start conditions rather than asserting that weak fairness bounds response time. Preserve existing product Models; a disagreement with implementation goes to the owner.
- Document composed step counting, typed fairness selection, inherited/replacement assumptions, supported/unsupported cases and model-only scope. Group claims per fn-149 only when that authoring contract has landed.
- Regenerate once after integrating task 3 and task 4. Review new model receipts and any generated changes; do not add live composition Cases or alter historical Run evidence.
- Run the existing model/Case/lint/Go gates once under the shared lock; retain focused evidence and full-suite JSON timing. No new CI or generated-API drift gate.

### Investigation targets
**Required:**
- `model/temporal/features/activity/standalone/system/System.scala:589` - existing composition.
- `model/framework/SEMANTICS.md:403` - composed state/start semantics; Progress section.
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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: