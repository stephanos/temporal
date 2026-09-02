---
satisfies: [R1, R2, R4]
---
# fn-40-centralize-plannerpolicy-constructors.1 Add canonical PlannerPolicy constructors and query coverage

## Description
Add the canonical constructor interface after the fn-17.1 strategy rename, then prove and document its exact policy and identity semantics (R1, R2, R4).

**Size:** M
**Files:** post-fn-17 owner of `PlannerPolicy` (currently `model/Umpire/Query/Language.lean`), `model/Umpire/Query/Tests/Fixtures.lean`, `model/Umpire/Query/Tests/Identity.lean`, `model/Umpire/ARCHITECTURE.md`
**Touches:** [model/Umpire/Search.lean, model/Umpire/Query/Language.lean, model/Umpire/Query/Tests/**, model/Umpire/ARCHITECTURE.md]

### Approach
- Re-anchor after `fn-17-bounded-semantic-exploration-and.1` and add the three constructors in the namespace of the post-rename `PlannerPolicy` owner.
- Give the public type and constructors Lean docstrings that distinguish identity-bearing seeds from seeded traversal.
- Replace the ordinary Query fixture policies with constructor aliases while keeping the deliberate seed-18 record update.
- Add checked examples for exact strategy/seed/tie-break fields, the default seeded policy, explicit seed zero, and identity sensitivity.
- Extend the architecture Query section with the canonical authoring interface; preserve existing comments.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Language.lean:41-45` — current policy representation and likely pre-rename owner
- `model/Umpire/Planning/Engine.lean:300-318` — seed consumption semantics
- `model/Umpire/Query/Tests/Fixtures.lean:270-299` — repeated default policies
- `model/Umpire/Query/Tests/Identity.lean:126-150` — deliberate strategy/seed identity mutations
- `model/Umpire/ARCHITECTURE.md:273-306` — public Query documentation

### Key context
- `fn-17.1` removes the false coverage-guided Query name without a compatibility alias; expose `PlannerPolicy.seeded`, not `PlannerPolicy.coverageGuided`.
- The facade already re-exports the policy owner, so no parallel public interface is needed.

### Acceptance
- [ ] Constructors produce the exact R1 fields for default seed 17 and explicit seeds including zero.
- [ ] Query fixtures use the constructors while the seed-18 identity regression remains explicit and passing.
- [ ] Public docstrings and architecture prose state the R4 semantics without contradicting canonical serialization.
- [ ] Existing comments remain intact.
- [ ] `cd model && mise exec -- lake build UmpireTests` passes.

## Acceptance
- [ ] Constructor values and explicit/default seed checks pass.
- [ ] Query identity coverage and public documentation satisfy R1/R2/R4.
- [ ] Focused Umpire tests pass with existing comments preserved.

## Done summary
Added documented `PlannerPolicy.shortest`, `PlannerPolicy.exhaustive`, and defaultable `PlannerPolicy.seeded` constructors; Query fixtures now use the ordinary aliases, exact default/zero fields are checked, and the deliberate seed-18 identity mutation remains explicit. Focused, aggregate, full model, and model-lint checks pass; the pre-existing regression-vocabulary failure and 1,385-item Go lint baseline are unchanged and explicitly waived by the owner.

stage: impl-review - ran | SHIP (codex)
## Evidence
- Commits: 91f561cb273c3e1323a3a491862ce883996b664e
- Tests: cd model && mise exec -- lake build Umpire.Query.Tests.Identity (TDD red before constructors; green after implementation), cd model && mise exec -- lake build Umpire.Query.Tests.Identity Umpire.Query.Tests, cd model && mise exec -- lake build UmpireTests, cd model && mise exec -- lake build Umpire.Examples.Switch Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, make umpire-build-model, make lint-model, make umpire-check-regression (inherited baseline and post-change red only: retired umpire-experiment/v1 test literals in fn-17 files; owner-authorized waiver), GOLANGCI_LINT_FIX=false make lint-code (inherited baseline: 1385 issues)
- PRs: