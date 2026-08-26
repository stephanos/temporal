---
satisfies: [R3, R4, R8]
---
# fn-16-authored-variation-spaces-and.3 Expose canonical checked space metadata

## Description
Create the deterministic in-memory metadata projection that fn-5 can consume for R3/R4/R8. This task does not aggregate a registry, persist JSON, or implement list/explain behavior.

**Size:** M
**Files:** `model/Umpire/Space/Metadata.lean`, `model/Umpire/Space/Tests/Metadata.lean`
**Touches:** [model/Umpire/Space/Metadata.lean, model/Umpire/Space/Tests/Metadata.lean]

### Approach
- Project one checked space into typed rows for its space, axes, choices, faults, and coverage goals with exact references, source/version data, bounds, and base semantic digests.
- Compute one semantic digest from canonical meaning-bearing metadata only.
- Validate projection completeness/bijection against the checked space and expose no unchecked constructor.
- Test order independence, source-versus-semantic identity behavior, and stale base/metadata mismatches.
- Keep aliases, dispositions, graph aggregation, glossary/index encoding, and query UI outside this package.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:28-69` — shared metadata vocabulary
- `model/Umpire/Behavior/Language.lean:676-807` — canonical metadata projection conventions
- `model/Umpire/Query/Language.lean:298-406` — semantic digest and source-order boundary
- `.flow/tasks/fn-5-umpire-discovery-promotion-and-artifact.1.md` — downstream catalog ownership
- `model/Umpire/ARCHITECTURE.md:31-44` — checked-value lifecycle

### Acceptance
- [ ] Metadata has exactly one row per checked space/axis/choice/fault/goal identity and no semantic-body copies.
- [ ] Reordering authored inputs cannot change rows, references, or digest.
- [ ] Missing/extra/stale rows or base digest mismatch prevent a metadata value.
- [ ] Coverage goals remain declared intent without achieved-count fields.
- [ ] No persisted registry, JSON reader/writer, list/explain, disposition, alias, or Temporal import exists.

## Acceptance
- [ ] Canonical metadata is complete, bijective, and deterministic.
- [ ] Fn-5 receives a small typed input contract rather than a second registry.
- [ ] No persistence or discoverability surface is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
