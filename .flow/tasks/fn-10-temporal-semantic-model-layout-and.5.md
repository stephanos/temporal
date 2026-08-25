---
satisfies: [R1]
---
# fn-10-temporal-semantic-model-layout-and.5 Replace Umpire test fixtures with synthetic vocabulary

## Description
Rewrite the reusable Core, Property, and Behavior test fixtures so they exercise the same DSL contracts using synthetic `test.*` identities and sources (R1). This task changes fixture vocabulary only, not production DSL APIs or test strength.

**Size:** M
**Files:** `model/Umpire/CoreTests.lean`, `model/Umpire/Property/Tests.lean`, `model/Umpire/Behavior/Tests.lean`
**Touches:** [model/Umpire/CoreTests.lean, model/Umpire/Property/Tests.lean, model/Umpire/Behavior/Tests.lean]

### Approach
- Replace Temporal-owned provider, capability, relation, target, state, action, outcome, observation, bound, and source identifiers with explicit synthetic equivalents.
- Preserve every positive, negative, connector, composition, deterministic, law-witness, and digest assertion.
- Keep the switch example unchanged because it is already domain-neutral.
- Avoid broad word replacement; ordinary temporal-logic vocabulary remains valid.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/CoreTests.lean` — connector, declaration, source, and digest fixtures
- `model/Umpire/Property/Tests.lean` — property construction, validation, and law witnesses
- `model/Umpire/Behavior/Tests.lean` — behavior composition and bounded action fixtures
- `model/Umpire/Examples/Switch.lean:7-613` — accepted domain-neutral vocabulary pattern

**Optional** (reference as needed):
- `model/UmpireTests.lean:1-10` — reusable test aggregate

### Acceptance
- [ ] The three suites contain no Temporal-owned namespace, source, or semantic identity prefixes.
- [ ] Test counts and positive/negative assertion categories remain equivalent.
- [ ] Connector laws, deterministic normalization, composition, and digest checks retain their original intent.
- [ ] `UmpireTests` builds without importing Temporal modules.

## Acceptance
- [ ] Reusable test fixtures use only synthetic domain-neutral vocabulary.
- [ ] Existing test categories and strength are preserved.
- [ ] UmpireTests builds independently of Temporal.

## Done summary
Replaced product-owned identities and sources in the reusable Core, Property, and Behavior fixtures with domain-neutral `test.*`/`Test/*` vocabulary while preserving comments and every assertion category. `UmpireTests` and the full regression command pass; baseline: green via handoff (verified at `b89e1e2e` by task 4).

stage: impl-review - ran [2026-08-25T23:24:52Z..2026-08-25T23:26:45Z] | SHIP
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0c87730a90413a60b6636ca1d15cb4304ce4a9ab
- Tests: baseline: green via handoff (verified at b89e1e2e by fn-10-temporal-semantic-model-layout-and.4), cd model && mise exec -- lake build UmpireTests, make umpire-check-regression
- PRs: