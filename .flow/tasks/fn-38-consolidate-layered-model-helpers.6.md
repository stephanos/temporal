---
satisfies: [R6]
---
# fn-38-consolidate-layered-model-helpers.6 Document helper ownership and run final model gates

## Description
Document the final helper ownership and complete the repository-level verification for R6. Keep documentation focused on the enforced architecture and existing consumer surface.

**Size:** M
**Files:** `model/ARCHITECTURE.md`, `model/README.md` (conditional), `model/Umpire/ARCHITECTURE.md` (minimal/conditional)
**Touches:** [model/ARCHITECTURE.md, model/README.md, model/Umpire/ARCHITECTURE.md]

### Approach
- Update the normative model dependency map and test-root guidance with the final `Umpire.Shared*` and `Temporal.Shared` ownership rules.
- State that shared helpers are implementation/test-support seams and do not replace existing public imports or declarations.
- Update the model README only if the documented commands or boundary semantics need clarification; do not invent a new command surface.
- Add at most a minimal Umpire architecture note if the internal helper seam would otherwise be misleading next to the public focused-import list.
- Run the full aggregate build, transitive lint, and regression gate after all migrations land.

### Investigation targets
**Required** (read before coding):
- `model/ARCHITECTURE.md:69-121,218-265` — normative dependency and test-root guidance.
- `model/README.md:156-192` — documented build, lint, and regression commands.
- `model/Umpire/ARCHITECTURE.md:7-31,330-353` — public focused imports and Switch reference.
- `Makefile:1269-1282` — model lint and controlled-boundary violation gate.
- `Makefile:1019-1040,1058-1082` — complete regression gate composition.

## Acceptance
- [ ] Architecture documentation states the final ownership and one-way dependency rules without presenting test helpers as public facades.
- [ ] README and Umpire architecture changes are limited to text made necessary by the final module surface.
- [ ] `cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect` passes.
- [ ] `make lint-model` and `make umpire-check-regression` pass.

## Done summary
Documented the final one-way `Umpire.Shared`, `Umpire.Shared.Test`, and `Temporal.Shared` ownership rules as internal implementation/test-support seams while leaving README commands and public consumer facades unchanged. The aggregate build, model lint, and regression gate pass; the pre-edit lint needed one sequential cache warm after the known Lake/virtiofs missing-`.ilean` race.

stage: impl-review - ran [2026-08-28T19:00:17Z..2026-08-28T19:00:17Z] (SHIP via deterministic docs-only triage)
## Evidence
- Commits: 5a46d3de3812703f35be3be0b6899ed9238ceb54
- Tests: baseline: green via handoff (verified at f79eefa8c by fn-38-consolidate-layered-model-helpers.7); pre-edit make lint-model passed after sequential recovery from an inherited Lake/virtiofs missing-.ilean race, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect, make lint-model, make umpire-check-regression
- PRs: