---
satisfies: [R2, R3, R7]
---
# fn-10-temporal-semantic-model-layout-and.2 Extract callback configuration and colocate system tests

## Description
Move callback-owned classifications, interpretations, contexts, typed uses, routing, admission, dispatch, and projection semantics into the Callback System module, then split the combined tests by ownership (R2, R3, R7). Shared and matching tests live under the Configuration test module; callback tests live beside Callback.

**Size:** M
**Files:** `model/Temporal/System/Callback/Configuration.lean`, `model/Temporal/System/Callback/ConfigurationTests.lean`, `model/Temporal/System/Configuration/Tests.lean`, `model/Temporal/Umpire/Config.lean`, `model/Temporal/Umpire/ConfigTests.lean`
**Touches:** [model/Temporal/System/Callback/Configuration.lean, model/Temporal/System/Callback/ConfigurationTests.lean, model/Temporal/System/Configuration/Tests.lean, model/Temporal/Umpire/Config.lean, model/Temporal/Umpire/ConfigTests.lean]

### Approach
- Move callback entries out of the mixed authored classification list together with callback interpretations, contexts, and typed use constructors; leave only shared catalog/checking concerns in Configuration.
- Move callback address policy, captured projection, route selection, admission, dispatch, and concrete trace semantics as one cohesive module.
- Split current tests by semantic owner without weakening assertions or duplicating fixtures, including coverage that exercises callback and matching uses in a shared resolved view.
- Convert the former combined production module to an import-only transition root until the final clean cutover; remove the former combined test module in this task.
- Preserve existing checked error kinds and deterministic fixture behavior.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Umpire/Config.lean:861-1066` — callback address and interpretation semantics
- `model/Temporal/Umpire/Config.lean:1089-1221` — callback classifications, interpretations, contexts, and typed uses in the mixed authored section
- `model/Temporal/Umpire/Config.lean:1245-1335` — callback projection and concrete trace behavior
- `model/Temporal/Umpire/ConfigTests.lean:9-370` — core, mixed-use, matching, and resolution coverage
- `model/Temporal/Umpire/ConfigTests.lean:372-606` — callback decoding, projection, and trace coverage

**Optional** (reference as needed):
- `model/Temporal/System/Configuration.lean` — shared facade created by task 1

### Acceptance
- [ ] Callback-specific classifications, interpretations, contexts, and typed uses exist only under Callback ownership.
- [ ] Callback depends on shared Configuration in one direction and exposes no Feature dependency.
- [ ] Shared, Matching, and Callback tests build through their owning System modules.
- [ ] Invalid values, overrides, settings, fixture identities, callback addresses, routes, admission, and dispatch retain existing checked outcomes.
- [ ] The former combined production module is import-only, the former combined test module is absent, and no assertions were weakened.
- [ ] Existing comments remain attached to the moved declarations and assertions.
## Acceptance
- [ ] Callback production and tests compile in the new System namespace.
- [ ] Split system test suites preserve positive, negative, and deterministic coverage.
- [ ] The mixed legacy test module is removed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
