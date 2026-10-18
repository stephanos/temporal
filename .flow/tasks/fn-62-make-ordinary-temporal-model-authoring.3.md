---
satisfies: [R4]
---
# fn-62-make-ordinary-temporal-model-authoring.3 Specialize Temporal identities and source contracts

## Description
Add only the Temporal-owned specialization left uncovered by fn-65. Reuse generic identities, sources, and named Limit constructors rather than rebuilding them.

**Size:** M
**Files:** `model/Temporal/Shared.lean`, `model/Temporal/Feature/Nexus/Operations/Internal.lean`, Temporal shared public tests and aggregate root as needed
**Touches:** [model/Temporal/Shared.lean, model/Temporal/SharedTests.lean, model/Temporal/Feature/Nexus/Operations/Internal.lean, model/TemporalModelTests.lean]

## Approach
- Add a narrow constructor fixing the `temporal` root and accepting explicit semantic family, kind, and suffix, returning existing `DefinitionFamily`/`DefinitionId` values.
- Reuse `Temporal.Shared.sourceLocation` and `QueryLimitSpec`; keep source locations explicit and stable for migrated declarations. Do not introduce a macro or implicit identity selection.
- Test equivalent raw/helper identities, declaration/source-order independence, and the inability to replace the owned root. Raw ID syntax remains unchanged; checked reference errors belong to languages.
- Prepare the ordinary Nexus shared identity inputs without migrating the three operation bodies (owned by `.4`).

## Investigation targets
**Required:**
- `model/Temporal/Shared.lean:7` — existing ID and source helpers.
- `model/Umpire/Target/Authoring.lean:7` — generic family implementation.
- `model/Umpire/Query/Authoring.lean:9` — existing `QueryLimitSpec`.
- `model/Temporal/Feature/Nexus/Operations/Internal.lean` — existing shared operation identity/source inputs.

## Acceptance
- [ ] Helper-generated IDs equal existing raw IDs and cannot change the fixed root; suffixes/kinds remain explicit.
- [ ] Source-location and ordering tests preserve identity/fingerprint meaning; malformed raw IDs remain rejected by existing syntax checking.
- [ ] Compiled specimens exercise duplicate/crossed-reference and invalid/zero/wrong-unit Limits through existing language checkers, preserving IDs and source diagnostics.
- [ ] Public Temporal tests are wired into `TemporalModelTests`; focused new root and `Temporal.Feature.Nexus.OperationsTests` build, with applicable lint checks.
- [ ] Structural cost inventory shows only explicit ID string construction and source/Limit assembly, with no registry or repeated declaration scan; quantify added work for 1×/10× independent declarations separately from unchanged checker work.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: