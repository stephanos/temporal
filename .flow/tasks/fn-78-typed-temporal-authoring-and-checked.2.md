---
satisfies: [R1, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.2 Make Query validity and endpoint semantics explicit

## Description
Implement D1 and R1 by extending the existing checked Query and Planning owners with explicit endpoint, trigger-coverage, answer, and search-completeness dimensions. Keep witness selection separate from universal verification and preserve deterministic finite selection and Exact Replay.

**Size:** L
**Files:** `model/Umpire/Target/{Language,Authoring,FiniteMachine,FiniteTable,Tests/**}.lean`, `model/Umpire/Query/{Language,Authoring,Tests/**}.lean`, `model/Umpire/Property/{Evaluation,Tests/**}.lean`, `model/Umpire/Planning/{Types,Engine,CaseAnalysis,Tests/**}.lean`, `model/Umpire/Space/Compiler.lean`, `model/Umpire/SemanticInventory/Tests/PlanningRuntime.lean`, `model/SEMANTIC_INVENTORY.md`
**Touches:** [model/Umpire/Target/**, model/Umpire/Query/**, model/Umpire/Property/Evaluation.lean, model/Umpire/Property/Tests/**, model/Umpire/Planning/**, model/Umpire/Space/Compiler.lean, model/Umpire/SemanticInventory/Tests/PlanningRuntime.lean, model/SEMANTIC_INVENTORY.md]

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
Implemented explicit checked Query endpoint/exercise policies, conjunctive finite Target terminal declarations, and independent Planning satisfiability, trigger coverage, answer, and search-completeness receipts. Runtime prefixes use a checked Property endpoint seam; complete universal claims require exercise and no unresolved obligations, and counterexamples survive later exhaustion. Exact budget completion, deterministic selection, and Exact Replay are covered.

Status: complete and ready for the user to commit; implementation remains uncommitted.
Baseline: green (`make umpire-build-model`, 480 jobs), before edits. The new exact-budget regression was observed failing before the planner change.

Changed files:
- model/SEMANTIC_INVENTORY.md
- model/Umpire/Planning/CaseAnalysis.lean
- model/Umpire/Planning/Engine.lean
- model/Umpire/Planning/Tests.lean
- model/Umpire/Planning/Types.lean
- model/Umpire/Property/Evaluation.lean
- model/Umpire/Property/Tests.lean
- model/Umpire/Query/Authoring.lean
- model/Umpire/Query/Language.lean
- model/Umpire/SemanticInventory/Tests/PlanningRuntime.lean
- model/Umpire/Space/Compiler.lean
- model/Umpire/Target/FiniteMachine.lean
- model/Umpire/Target/FiniteTable.lean
- model/Umpire/Target/Language.lean
- model/Umpire/Target/Tests/FiniteMachine.lean
- model/Umpire/Target/Tests/FiniteTable.lean
- model/Umpire/Planning/Tests/Endpoints.lean
- model/Umpire/Property/Tests/Endpoints.lean

Validation:
- `LEAN_NUM_THREADS=2 make umpire-build-model`: passed, 482 jobs (/tmp/fn78-task2-final-build.log).
- `cd model && mise exec -- lake build Umpire.Planning.Tests Umpire.Property.Tests Umpire.Query.Tests Umpire.Target.Tests.FiniteMachine Umpire.Target.Tests.FiniteTable Umpire.Target.Tests.Compatibility`: passed, 82 jobs (/tmp/fn78-task2-final-focused.log).
- `make umpire-gen-semantic-inventory`: passed; regenerated only model/SEMANTIC_INVENTORY.md, adding the two outcome names.
- `LEAN_NUM_THREADS=2 make lint-model`: passed, including builtin lint (346 jobs), exit 0 (/tmp/fn78-task2-final-lint-model.log). Reduced concurrency resolved an earlier resource-killed aggregate lint run.
- `make lint-code GOLANGCI_LINT_FIX=false`: failed with 1,284 pre-existing issues in untouched Go sources (/tmp/fn78-task2-lint-code.log). No Go files differ; unrelated fixes were deliberately excluded. The failure prevents the recipe's later go-vet step from running.
- `git diff --check`: passed.

Focused evidence includes impossible versus unexercised scenarios; exercised witness and universal success; unresolved runtime prefixes and inclusive deadlines (plain and guarded), missing state-invariant evidence versus actual violations; exact work boundaries and counterexample-before-exhaustion; conjunction of terminal declarations, empty/unfinished constituents, typed finite lowering, and invalid finite terminal states; distinct paths converging to one Target state; Exact Replay; realized trigger path/coordinate identity; and unchanged legacy Target canonical bytes/fingerprints plus frozen new terminal and default Query fingerprints.

The final complete model build includes the last missing-state prefix correction and all new tests.

Trust audit: evaluatePropertyEndpoint and evaluatePropertyEndpoint_closed retain the existing evaluateProperty transitive set [propext, Classical.choice, Quot.sound]; PlanningOutcome.constructorClassifiers_exactlyOne remains axiom-free. FiniteTable.validate retains its existing set; no new custom/compiler-trust axioms, toolchains, or dependencies were added.

Scope notes: Property/Evaluation and its focused tests are necessary additional task touches. Space/Compiler and SemanticInventory test/generated inventory are compatibility consumers of the two added outcomes. Runtime-prefix semantics cover the existing checked fragment; correlated operation-scoped transition semantics remain task .6. Terminal conditions explicitly list eligible global states per constituent and are conjunctive; empty metadata or an empty constituent never infers terminality from deadlock. Architecture documentation remains qualification-task work. Existing .flow and .plans/UMPIRE4_ORDER.md changes were preserved.

stage: impl-review - passed(gpt-6-astra at medium; two findings fixed; resumed verdict SHIP)

Independent review fixes (conductor-owned review):
- P1: guarded prefix evaluation now visits applicable trigger positions directly, preserving missing/malformed coordinates as unresolved instead of losing those obligations in the diagnostic observation list. Closed Boolean evaluation is unchanged. The regression checks both missing and malformed logical-time observations with an applicable guarded trigger and no response.
- P2: final Planning metadata comes from actual traversal termination; selecting a stored counterexample no longer resets completeness.established. Regressions assert both completeness fields for exhaustive and budget-exhausted counterexample runs.
- Review-fix files: model/Umpire/Property/Evaluation.lean; model/Umpire/Property/Tests/Endpoints.lean; model/Umpire/Planning/Engine.lean; model/Umpire/Planning/Tests/Endpoints.lean.
- Both regression modules failed before the fixes (/tmp/fn78-task2-review-red.log), then passed together (37 jobs; /tmp/fn78-task2-review-focused.log).
- Post-review gates: `LEAN_NUM_THREADS=2 make umpire-build-model` passed (482 jobs, exit 0; /tmp/fn78-task2-review-build.log); `LEAN_NUM_THREADS=2 make lint-model` passed (346 builtin-lint jobs, exit 0; /tmp/fn78-task2-review-lint-model.log). Final `git diff --check` passed. Existing Go lint debt was not rerun because no Go files changed.


stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (make umpire-build-model, 480 jobs), LEAN_NUM_THREADS=2 make umpire-build-model (passed, 482 jobs), cd model && mise exec -- lake build Umpire.Planning.Tests Umpire.Property.Tests Umpire.Query.Tests Umpire.Target.Tests.FiniteMachine Umpire.Target.Tests.FiniteTable Umpire.Target.Tests.Compatibility (passed, 82 jobs), make umpire-gen-semantic-inventory (passed), LEAN_NUM_THREADS=2 make lint-model (passed, builtin lint 346 jobs, exit 0), make lint-code GOLANGCI_LINT_FIX=false (inherited failure: 1284 issues in unchanged Go sources), git diff --check (passed), review regressions: both endpoint test modules failed before fixes (/tmp/fn78-task2-review-red.log), cd model && LEAN_NUM_THREADS=2 mise exec -- lake build Umpire.Property.Tests.Endpoints Umpire.Planning.Tests.Endpoints (passed, 37 jobs; /tmp/fn78-task2-review-focused.log), LEAN_NUM_THREADS=2 make umpire-build-model (post-review fixes passed, 482 jobs; /tmp/fn78-task2-review-build.log), LEAN_NUM_THREADS=2 make lint-model (post-review fixes passed, 346 builtin-lint jobs, exit 0; /tmp/fn78-task2-review-lint-model.log), git diff --check (post-review fixes passed), flowctl codex impl-review fn-78.2 (gpt-6-astra at medium; two findings fixed; resumed verdict SHIP)
- PRs: