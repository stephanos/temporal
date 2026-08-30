---
satisfies: [R2, R5, R6]
---
# fn-51-shorten-ordinary-model-authoring.3 Add and migrate QueryLimits.bounded

## Description
Introduce the fixed-unit bounded constructor and migrate ordinary Query declarations (R2, R5, R6).

**Size:** M
**Files:** `model/Umpire/Query/Language.lean`, `model/Umpire/Query/Tests/Identity.lean`, `model/Umpire/Query/Tests/Fixtures.lean`, `model/Umpire/Planning/Tests/Fixtures.lean`, `model/Umpire/Examples/Switch.lean`, `model/Temporal/Feature/Nexus/Lifecycle/Target.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean`, `model/Temporal/Feature/Nexus/Experimental/VariationSpace.lean`
**Touches:** [model/Umpire/Query/Language.lean, model/Umpire/Query/Tests/Identity.lean, model/Umpire/Query/Tests/Fixtures.lean, model/Umpire/Planning/Tests/Fixtures.lean, model/Umpire/Examples/Switch.lean, model/Temporal/Feature/Nexus/Lifecycle/Target.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean, model/Temporal/Feature/Nexus/Experimental/VariationSpace.lean]

### Approach
- Fix only the established semantic-transition, selected-action, and candidate-evaluation units.
- Inventory all three-unit Query limit records across `model/Umpire` and `model/Temporal`; migrate ordinary production and shared positive fixture triples while retaining unit-mutation and negative records raw.
- Prove output equality and retain Query checking for zero/insufficient/custom cases.
- Preserve values, parameterized budgets, and planner policies.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Language.lean:24-39` — Limit structures and ownership
- `model/Umpire/Examples/Switch.lean:539-547` — ordinary 1/1/8 limits
- `model/Temporal/Feature/Nexus/Lifecycle/Target.lean:448-456` — lifecycle limits
- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean:646-654` — experimental limits
- `model/Temporal/Feature/Nexus/Experimental/VariationSpace.lean:90-102` — 2/2/32 Space query limits
- `model/Umpire/Query/Tests/Fixtures.lean:255-261` — positive Query fixture limits
- `model/Umpire/Planning/Tests/Fixtures.lean:267-273` — parameterized positive planner limits
- `model/Umpire/Query/Tests/Identity.lean:110-155` — identity sensitivity and deliberate mutations
## Acceptance
- [ ] Constructor output equals the current nested Limit record with exact units and values.
- [ ] A repository-wide inventory covers every ordinary fixed-unit production and shared-positive-fixture declaration, including Query and Planning fixtures; all use the constructor.
- [ ] Custom-unit, negative, and deliberate mutation records remain possible and raw.
- [ ] Query diagnostics, identities, planner results, and artifact outputs are unchanged.
- [ ] Query, Planning, Switch, lifecycle, and experimental focused suites pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
