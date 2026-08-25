---
satisfies: [R2, R5, R7]
---
# fn-10-temporal-semantic-model-layout-and.6 Move inspector tooling and assemble Temporal model tests

## Description
Move registry/CLI behavior into `Temporal.Tool.Inspect`, colocate its tests, and assemble the import-only `TemporalModelTests` root from Feature, System, and Tool tests (R2, R5, R7). Keep old roots as temporary delegating build bridges until task 7 performs the clean target cutover.

**Size:** M
**Files:** `model/Temporal/Tool/Inspect.lean`, `model/Temporal/Tool/InspectTests.lean`, `model/TemporalModelTests.lean`, `model/Temporal/Umpire/Inspect.lean`, `model/TemporalUmpireTests.lean`
**Touches:** [model/Temporal/Tool/Inspect.lean, model/Temporal/Tool/InspectTests.lean, model/TemporalModelTests.lean, model/Temporal/Umpire/Inspect.lean, model/TemporalUmpireTests.lean]

### Approach
- Move the generic registry/result/diagnostic and production scenario composition without changing CLI behavior.
- Update the registry to import the Feature scenario and reusable switch directly.
- Extract inspector success and failure assertions into Tool tests.
- Make `TemporalModelTests` an import-only aggregate; do not import `UmpireTests` or consume reusable test fixtures.
- Keep any temporary old entry root declaration-free except for executable delegation required by the still-old Lake target.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Umpire/Inspect.lean:1-82` — registry, canonical diagnostics, and executable entry point
- `model/TemporalUmpireTests.lean:300-351` — inspector behavior assertions
- `model/UmpireTests.lean:1-10` — import-only aggregate convention
- `model/Umpire/Examples/Switch.lean` — reusable registered scenario

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/CallerClosure.lean` — Temporal registered scenario from task 4

### Acceptance
- [ ] Tool registry contains both unchanged scenario identities and owns no feature semantics.
- [ ] Valid, failed, unknown, and invalid-arity results preserve status/stdout/stderr contracts.
- [ ] `TemporalModelTests` imports colocated Feature/System/Tool tests and owns no assertions.
- [ ] Temporal tests do not import `UmpireTests` or reusable test fixtures.
- [ ] Existing comments remain attached to moved tool declarations and tests.

## Acceptance
- [ ] New Tool inspector and tests compile with unchanged behavior.
- [ ] TemporalModelTests is an import-only aggregate independent of generic test internals.
- [ ] Temporary old roots contain no semantic declarations.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
