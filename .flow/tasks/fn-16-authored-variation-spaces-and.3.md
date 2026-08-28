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
Added canonical source-backed Space metadata rows with exact references, bounds, Behavior Fingerprints, deterministic ordering, and fail-closed projection validation; sealed checked Space construction and covered source/semantic identity plus missing, extra, stale, and fingerprint mismatch cases.

Baseline was red only for absent cumulative pre-feature targets: Compilation (.4), Metadata (.3), and Temporal Variation Space (.5). Verification now passes Validation, Metadata, Switch, aggregate model suites, and regression; Compilation and Temporal remain expected future-task absences with no scope-violating stubs. Review fixed forgeable checked Space construction and returned SHIP; memory capture and gate receipts were non-blockingly unavailable because memory is uninitialized and the preserved plan diff keeps the worktree dirty.

stage: impl-review - ran [2026-08-28T01:18:46.244425Z..2026-08-28T01:24:53.596179Z]
## Evidence
- Commits: 41faee380c58936472b7eac9bcd6b24bdca5073a, abb9355fd51f43e1b3e9e313a9d56d2bc05dfe52, 39f0e659df834ea66d0b2bdb3934b61062af4680
- Tests: cd model && mise exec -- lake build Umpire.Space.Tests.Validation, cd model && mise exec -- lake build Umpire.Space.Tests.Metadata, cd model && mise exec -- lake build Umpire.Examples.SwitchTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, EXPECTED_FUTURE_TARGET: cd model && mise exec -- lake build Umpire.Space.Tests.Compilation - absent for task .4, EXPECTED_FUTURE_TARGET: cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.VariationSpaceTests - absent for task .5
- PRs: