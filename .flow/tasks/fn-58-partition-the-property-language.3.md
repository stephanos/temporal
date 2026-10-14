---
satisfies: [R1, R4]
---
# fn-58-partition-the-property-language.3 Lock the Property facade, documentation, and compatibility

## Description
Expand facade checks across Language, Check, Trace, and Evaluation; document the internal ownership while retaining public author guidance; audit moved comments and theorem trust; and run aggregate compatibility gates.

**Size:** S
**Files:** `model/Umpire/Property/ImportTests.lean`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `Makefile`
**Touches:** [model/Umpire/Property/ImportTests.lean, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md]

### Approach
- Expand the facade regression to check representative Language, Check, Trace, and Evaluation declarations while retaining the Behavior and Query negative guards.
- Preserve the package gate and its exact direct Language import requirement.
- Document the four internal modules as implementation modules normally reached through `Umpire.Property`.
- Preserve the existing Property lifecycle, Limit, and raw-checker guidance; refresh documentation anchors after concurrent Observation documentation work.
- Audit moved comments, declaration names, warnings, imports, theorem statements, and axiom inventories.
- Run focused, aggregate, artifact regression, model lint, and repository lint gates.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property.lean:1` — stable facade imports
- `model/Umpire/Property/ImportTests.lean:1-13` — facade visibility and negative guards
- `Makefile:1185-1203` — package facade and import enforcement
- `model/Umpire/ARCHITECTURE.md:34-42` — public versus internal imports
- `model/Umpire/ARCHITECTURE.md:163-192` — Property lifecycle and checking guidance
- `model/ARCHITECTURE.md:431-434` — facade implementation navigation

**Optional** (reference as needed):
- `model/UmpireTests.lean:1-24` — aggregate reusable test imports

### Key context
- Do not rewrite unchanged public semantics or edit the Makefile unless the exact existing gate cannot remain intact.
- This documentation task may follow Observation documentation edits without coupling the Property implementation to those specs.

## Acceptance
- [ ] R1 and R4 are satisfied by a stable `Umpire.Property` facade that exposes representative Language, Check, Trace, and Evaluation declarations with unchanged types.
- [ ] Behavior and Query authoring declarations remain absent from the narrow facade, and the package gate retains the exact direct Language import contract.
- [ ] Architecture docs describe internal ownership and continue directing ordinary authors to `Umpire.Property` without rewriting unchanged semantics.
- [ ] No consumer source change, generated output, artifact byte/checksum, warning, import-boundary, comment, theorem, or trust drift is introduced.
- [ ] `cd model && mise exec -- lake build Umpire.Property.ImportTests UmpireTests Temporal TemporalModelTests TemporalExperimentalTests` passes.
- [ ] `make umpire-build-model`, `make umpire-check-regression`, and `make lint-model` pass.
- [ ] `make lint-code GOLANGCI_LINT_FIX=false` is run without exceeding the approved inherited exact 1,381-finding baseline, task-diff-scoped golangci reports zero findings, and the unchanged `tools/umpire/runtime/errors.go:60` errortype finding remains isolated.

## Done summary
Locked the `Umpire.Property` facade with exact type checks across Language, Check, Trace, and Evaluation, retained the Behavior/Query negative guards and direct-Language package contract, and documented the four internal owners behind the unchanged public author surface. Commit `532963d564146f559232b4e9cf37572eea91f8e2` contains the three task-owned paths together with concurrent out-of-task fn-62/fn-63 Flow and roadmap changes; `95c975ca2e0399e7560683141b8280d66d29da1a` is an empty task-boundary marker created before conductor guidance arrived.

stage: impl-review - ran (model: gpt-5.6-sol, verdict: SHIP, receipt: /tmp/impl-review-receipt-fn-58-partition-the-property-language.3.json)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 532963d564146f559232b4e9cf37572eea91f8e2, 95c975ca2e0399e7560683141b8280d66d29da1a
- Tests: cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Property.ImportTests Umpire.Observation.Tests.Verdict, cd model && mise exec -- lake build UmpireTests Temporal TemporalModelTests TemporalExperimentalTests, make umpire-build-model, make umpire-check-regression, make lint-model, make lint-code GOLANGCI_LINT_FIX=false (approved inherited exact 1,381 findings), GOLANGCI_LINT_BASE_REV=d322473a8a26573feeb9dbf6c80e7258a030a3a5 make lint-code GOLANGCI_LINT_FIX=false (0 golangci findings; waived unchanged tools/umpire/runtime/errors.go:60 errortype finding), git diff --check, Property extraction audit against 8c69f221a9fc5b4bacec4f9a4fc61f3bec268fe6 (no declaration-name, theorem-statement, moved declaration-doc, or trust drift), #print axioms for ValueConstraint.evaluate_agrees, PropertyPattern.evaluate_agrees, and evaluatePropertyClause_agrees (unchanged propext, Classical.choice, Quot.sound inventory), CONCURRENT_COMMIT: 532963d564146f559232b4e9cf37572eea91f8e2 contains the three fn-58.3 task paths plus concurrent out-of-task fn-62/fn-63 Flow and .plans/UMPIRE4_ORDER.md changes
- PRs:
