---
satisfies: [R1, R7, R8]
---
# fn-15-standalone-api-and-config-input-catalogs.1 Build the generic catalog query deep module

## Description
Implement the pure shared query, validation, JSON-envelope, and dispatch boundary required by R1/R7/R8 before either production adapter lands.

**Size:** M
**Files:** `model/Temporal/Tool/Catalog/Core.lean`, `model/Temporal/Tool/Catalog/Query.lean`, `model/Temporal/Tool/Catalog/Json.lean`, `model/Temporal/Tool/Catalog/QueryTests.lean`
**Touches:** [model/Temporal/Tool/Catalog/Core.lean, model/Temporal/Tool/Catalog/Query.lean, model/Temporal/Tool/Catalog/Json.lean, model/Temporal/Tool/Catalog/QueryTests.lean]

### Approach
- Define the small public catalog definition, item, query, error, checked-catalog, and result interface.
- Enforce exact `<kind>:<identity>` selectors, closed kinds/views, canonical `(kind, identity)` ordering, uniqueness, recomputed identity, and domain validation.
- Keep response/error JSON and pure command dispatch behind the public facade.
- Exercise two unrelated synthetic payloads, valid empty views, reordering, duplicates, malformed selectors, unknown values, and delegated domain failures.
- Keep all API/config/semantic vocabulary and every effect outside the core.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Tool/Inspect.lean:17-77` — current pure result and effect-thin command shell.
- `model/Temporal/Tool/InspectTests.lean:12-71` — exact JSON/status testing.
- `model/Temporal/System/Configuration/Core.lean:78-101` — exact lookup/identity validation pattern.
- `.flow/tasks/fn-5-umpire-discovery-promotion-and-artifact.2.md` — downstream semantic-catalog reuse contract.

### Quick command
`cd model && lake env lean Temporal/Tool/Catalog/QueryTests.lean`

## Acceptance
- [ ] The same engine checks and queries two fixtures with unrelated domain payloads.
- [ ] Reordering source items cannot change checked identity, item order, or JSON bytes.
- [ ] Valid empty views succeed; malformed selector, unknown kind/view/subject, duplicate subject, invalid item, and failed domain validation retain distinct stable errors.
- [ ] Success/error stdout/stderr/status envelopes are exact and have one final LF.
- [ ] No API, config, semantic registry, IO, file, runtime, or Temporal-independent Umpire dependency enters the core.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
