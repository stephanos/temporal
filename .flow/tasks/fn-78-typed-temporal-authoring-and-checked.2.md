---
satisfies: [R1, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.2 Make Query validity and endpoint semantics explicit

## Description
Implement D1 and R1 by extending the existing checked Query and Planning owners with explicit endpoint, trigger-coverage, answer, and search-completeness dimensions. Keep witness selection separate from universal verification and preserve deterministic finite selection and Exact Replay.

**Size:** L
**Files:** `model/Umpire/Target/{Language,Authoring,FiniteMachine,FiniteTable,Tests/**}.lean`, `model/Umpire/Query/{Language,Authoring,Tests/**}.lean`, `model/Umpire/Property/{Evaluation,Tests/**}.lean`, `model/Umpire/Planning/{Types,Engine,CaseAnalysis,Tests/**}.lean`, `model/Umpire/Variations/Compiler.lean`, `model/Umpire/Inventory/Tests/PlanningRuntime.lean`, `model/INVENTORY.md`
**Touches:** [model/Umpire/Target/**, model/Umpire/Query/**, model/Umpire/Property/Evaluation.lean, model/Umpire/Property/Tests/**, model/Umpire/Planning/**, model/Umpire/Variations/Compiler.lean, model/Umpire/Inventory/Tests/PlanningRuntime.lean, model/INVENTORY.md]

### Approach
- Add explicit checked Target terminal declarations and carry them through finite admission, composed Targets, canonical identity, and compatibility. A composed state is terminal only when every constituent's declared terminal condition is met; absent declarations do not silently infer terminality from deadlock or one finished operation.
- Extend the checked Query declaration and receipt path at the current policy/completeness seams instead of adding a second planner result. Add the minimal checked Property endpoint seam needed to distinguish closed answers from unresolved prefixes; operation-scoped transition logic remains task `.6`.
- Represent scenario satisfiability, requested exercise coverage, Property answer, endpoint interpretation, and search completeness independently; derive named high-level outcomes only from their valid combinations.
- Use the Target's declared terminal/composition semantics for terminal-model endpoints. Keep deliberately closed traces and runtime prefixes distinct.
- Include all semantic/work limits and assurance method in canonical receipts and preserve state needed for sound search merging.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Language.lean:94-139` — current Query form and claim vocabulary
- `model/Umpire/Query/Language.lean:528-589` — checked admission and canonical identity
- `model/Umpire/Planning/Engine.lean:495-590` — current outcome finalization
- `model/Umpire/Planning/Engine.lean:648-671` — receipt construction
- `model/Umpire/Property/Evaluation.lean` — checked closed-trace authority and prefix endpoint seam
- `model/Umpire/Target/Language.lean` — checked Target and composition payload
- `model/Umpire/Target/FiniteMachine.lean:420-496` — finite checked Target construction

### Key context
- An incomplete search cannot establish absence, unsatisfiability, or verification.
- A replay-valid counterexample remains evidence even if broader search exhausts its work budget.
- Preserve realized trigger identity in coverage findings; see `.flow/memory/bug/integration/coverage-findings-must-retain-2026-09-05.md`.
- Default-empty terminal metadata must preserve unchanged Target canonical bytes/fingerprints; an explicit terminal declaration participates in semantic identity.
## Acceptance
- [ ] Checked Query inputs explicitly select deliberately closed, runtime-prefix, or terminal-model endpoint interpretation and an exercise/nonvacuity policy.
- [ ] Checked Targets carry explicit terminal declarations through finite admission and composition; composed terminal eligibility requires every constituent declaration and never infers closure from deadlock or one completed operation.
- [ ] Receipts report satisfiability, trigger coverage, Property answer, and completeness independently, including impossible, nonempty-unexercised, witness, verified, counterexample, unresolved-prefix, and exhausted cases.
- [ ] Universal success requires a nonempty admissible behavior, requested trigger coverage, complete search, and no counterexample or unresolved obligation; incomplete search produces no negative or green claim.
- [ ] Terminal-model closure follows declared composed Target terminal semantics, and valid terminal states are not treated as backend deadlocks.
- [ ] Exact work-budget boundaries, counterexample-before-exhaustion, sound search merging, deterministic selection, and Exact Replay have focused positive and negative tests.
- [ ] Existing Query/Target forms with default-empty terminal metadata and unchanged canonical receipts retain their IDs/fingerprints; explicit terminal semantics have focused compatibility fixtures.
- [ ] `make umpire-build-model` and focused Target/Query/Property/Planning tests pass; final architecture documentation is owned by the qualification task.
## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: