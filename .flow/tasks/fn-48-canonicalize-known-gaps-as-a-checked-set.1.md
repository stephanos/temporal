---
satisfies: [R1, R2, R3]
---
# fn-48-canonicalize-known-gaps-as-a-checked-set.1 Introduce the checked KnownGapSet boundary

## Description
Create the Planning-owned checked collection and its focused contract tests (R1-R3).

**Size:** M
**Files:** `model/Umpire/Planning/Types.lean`, `model/Umpire/Planning/Tests/KnownGaps.lean`
**Touches:** [model/Umpire/Planning/Types.lean, model/Umpire/Planning/Tests/KnownGaps.lean]

### Approach
- Reuse the existing kind rank, semantic key, validation order, error vocabulary, and row JSON in `model/Umpire/Planning/Types.lean:56-113`.
- Keep raw rows constructible while making checked collection construction opaque.
- Cover strict canonical admission, unordered producer admission, empty projection, union, exact cross-input deduplication, and conflicts.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Types.lean:28-113` — existing row, validation, ordering, and JSON authority
- `model/Umpire/Planning/Tests/KnownGaps.lean:5-58` — current typed negative cases and JSON example

**Optional** (reference as needed):
- `model/Temporal/Tool/RunEvaluation.lean:338-377` — duplicated canonicalization and strict parse behavior

## Acceptance
- [ ] Checked values can be created only through the documented strict, producer, empty, and union operations.
- [ ] Existing Known Gap error kinds and deterministic offending identities are preserved for invalid rows.
- [ ] Strict input order, duplicate, conflict, empty, and cross-input union cases have executable tests.
- [ ] `cd model && mise exec -- lake build Umpire.Planning.Tests.KnownGaps` passes.

## Done summary
Introduced the Planning-owned private-constructor `KnownGapSet` with strict admission, producer normalization, empty projection, and deterministic checked union. Focused R1-R3 tests cover invalid code and subject identities, strict ordering, duplicate and conflict failures, producer failures, empty inputs, exact cross-input deduplication, union conflicts, and both empty union identities.

Baseline waiver: `make lint-code` failed pre-edit with 1,379 inherited unrelated Go findings (errcheck 220, exhaustive 5, forbidigo 211, govet 5, revive 798, staticcheck 136, testifylint 4); its one auto-fix side effect was fully restored, and the conductor approved a Lean-only baseline waiver.

stage: impl-review - ran (Codex SHIP; completed 2026-09-03T05:56:11.855336Z; 0 findings)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 34cabcc9cc2238e4669602c8fd91aa99542d255a
- Tests: baseline: red (make lint-code failed pre-edit with 1,379 inherited unrelated Go findings: errcheck 220, exhaustive 5, forbidigo 211, govet 5, revive 798, staticcheck 136, testifylint 4; auto-fix fully restored; conductor-approved Lean-only waiver), cd model && mise exec -- lake build Umpire.Planning.Tests.KnownGaps Umpire.Artifact.Tests.Codecs Umpire.Artifact.Tests.Runtime Umpire.Artifact.Tests.Evidence Umpire.Artifact.Tests.Result Temporal.Tool.RunEvaluationTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests, go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/runevaluation, make umpire-check-regression, make lint-model
- PRs:
