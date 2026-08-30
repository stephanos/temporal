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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
