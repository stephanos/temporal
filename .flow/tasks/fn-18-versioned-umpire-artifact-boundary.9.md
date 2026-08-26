---
satisfies: [R1, R6, R8]
---
# fn-18-versioned-umpire-artifact-boundary.9 Implement deterministic named complete-set migrations

## Description
Implement R6's closed migration engine independently of publication.

**Size:** M
**Files:** `tools/umpire/artifact/migration.go`, `tools/umpire/artifact/migration_test.go`
**Touches:** [tools/umpire/artifact/migration.go, tools/umpire/artifact/migration_test.go]

### Approach
- Implement a closed registry keyed by exact source format, target format, and unique migration name.
- Strictly admit the complete source set, deterministically transform every affected member/reference/manifest in memory, and strictly admit the complete target set after every edge.
- Reject downgrades, skipped or unknown versions, aliases, guesses, ambiguous routes, semantic reinterpretation, partial-set output, and source mutation.
- Keep the production registry empty because no superseded production format exists.
- Prove one-way multi-step determinism, ambiguity rejection, before/after validation, atomic failure, and source immutability with private fixture-only formats.

### Investigation targets
**Required** (read before coding):
- Task `.8` admitted-set API and parent migration contract
- existing repository migration registry patterns, if any
- parent spec API and `Artifact Sets, Migrations, and Publication` sections

### Acceptance
- [ ] Every production v1 set returns stable `no-migration-route`; no fake product predecessor exists.
- [ ] Private fixture routes prove deterministic named multi-step migration and exact ambiguity/downgrade/invalid-intermediate rejection.
- [ ] Failed migration returns no target and never mutates source bytes.
- [ ] Every edge performs strict complete-set admission before and after transformation.

## Acceptance
- [ ] R6 deterministic complete-set migration semantics are implemented.
- [ ] The production registry remains honestly empty.
- [ ] Focused Go migration tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
