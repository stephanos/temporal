---
satisfies: [R1, R4, R6]
---
# fn-38-consolidate-layered-model-helpers.5 Enforce shared-module import policy

## Description
Extend the executable import policy so the new production and test-support seams are enforced rather than documented only. This task owns the direct and transitive rejection contracts for R1, R4, and R6.

**Size:** M
**Files:** `model/ModelLint/ImportGraph.lean`, `model/ModelLint/ImportGraphTests.lean`
**Touches:** [model/ModelLint/ImportGraph.lean, model/ModelLint/ImportGraphTests.lean]

### Approach
- Add explicit classification/predicates for `Temporal.Shared*` and test-support modules; do not rely on the current broad `.temporal`/`.umpire` categories to imply narrower rules.
- Permit `Temporal.Shared*` to reach only allowed lower Shared/Umpire layers and reject reachability to Feature, System, verification, and test-support modules.
- Reject any production-module reachability to `Umpire.Shared.Test` or another test-support module while allowing the intended test consumers.
- Cover direct and transitive violations for every new prefix, plus legal lower-layer/test-consumer examples.
- Keep source inventory reconciliation and deterministic violation ordering intact.

### Investigation targets
**Required** (read before coding):
- `model/ModelLint/ImportGraph.lean:69-106` — current prefix classification and module classes.
- `model/ModelLint/ImportGraph.lean:171-199` — current forbidden reachability rules.
- `model/ModelLint/ImportGraphTests.lean` — synthetic direct/transitive policy test pattern.
- `model/ARCHITECTURE.md:109-121` — normative MOD-01/MOD-09/MOD-11 boundary contract.

**Optional** (reference as needed):
- `model/ModelLint.lean` — live metadata/source-inventory integration.

## Acceptance
- [ ] `Temporal.Shared*` direct and transitive reachability to Feature/System/test-support modules is rejected while allowed lower-layer imports pass.
- [ ] Production reachability to `Umpire.Shared.Test` is rejected, and intended test-module imports pass.
- [ ] Existing MOD-01, MOD-03, MOD-05, MOD-09, MOD-10, source-inventory, and deterministic-order tests remain green.
- [ ] `cd model && mise exec -- lake build ModelLint.ImportGraphTests` and `make lint-model` pass.

## Done summary
Classified `Temporal.Shared*` explicitly and added fail-closed direct/transitive policies that keep it on lower Shared/Umpire layers while preventing production reachability to `Shared.Test`, `Umpire.Shared.Test`, and `Temporal.Shared.Test`. Synthetic legal-consumer coverage, the live source inventory/import graph, aggregate builds, and regression gates all pass without changing existing traversal, ordering, comments, or wire fixtures.

stage: impl-review - ran (SHIP)
## Evidence
- Commits: 4c8abf02b27f8620d549ef4b5b959711d17f3621
- Tests: baseline: green via handoff (green verified at 254da06df by fn-38-consolidate-layered-model-helpers.4); aggregate build, make lint-model, and regression independently revalidated before edits, git diff --check, cd model && mise exec -- lake build ModelLint.ImportGraphTests, cd model && mise exec -- lake exe modelLintTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect, make lint-model, make umpire-check-regression
- PRs: