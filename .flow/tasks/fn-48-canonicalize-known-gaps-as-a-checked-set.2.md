---
satisfies: [R2, R4]
---
# fn-48-canonicalize-known-gaps-as-a-checked-set.2 Admit checked Known Gaps at artifact boundaries

## Description
Move shared Lean Artifact types, encoders, and set checks onto the checked boundary while retaining raw negative cases at `KnownGapSet.checkCanonical` and the Go persisted decoder (R2, R4).

**Size:** M
**Files:** `model/Umpire/Artifact/Types.lean`, `model/Umpire/Artifact/Codecs.lean`, `model/Umpire/Artifact/Set.lean`, `model/Umpire/Planning/Tests/Artifacts.lean`, `tools/umpire/internal/artifactv2/artifact_test.go`
**Touches:** [model/Umpire/Artifact/Types.lean, model/Umpire/Artifact/Codecs.lean, model/Umpire/Artifact/Set.lean, model/Umpire/Planning/Tests/Artifacts.lean, tools/umpire/internal/artifactv2/artifact_test.go]

### Approach
- Store only checked sets in Lean semantic artifacts and render them through canonical projection; do not invent a Lean JSON decoder.
- Move malformed-list Lean assertions to strict set-admission tests where opacity makes invalid semantic Artifacts unconstructible.
- Retain the Go decoder's persisted malformed/order/duplicate/conflict/checksum matrix as the independent wire boundary.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact/Types.lean:60-75` — planning Artifact Known Gap field
- `model/Umpire/Artifact/Codecs.lean:120-150` — canonical encoder/checksum path
- `model/Umpire/Artifact/Set.lean:104-115` — repeated Lean set validation
- `model/Umpire/Planning/Tests/Artifacts.lean:139-182` — semantic Artifact mutations that opacity changes
- `tools/umpire/internal/artifactv2/artifact.go:580-610` — actual persisted-array admission
- `tools/umpire/internal/artifactv2/artifact_test.go` — Go negative-wire coverage
## Acceptance
- [ ] Lean semantic artifact models cannot carry an unchecked Known Gap list and no Lean decoder subsystem is added.
- [ ] Invalid Lean rows/order/duplicates/conflicts are covered at strict set admission; Go still rejects malformed, noncanonical, duplicate, conflicting, stale, and checksum-invalid persisted input.
- [ ] Valid planning Artifact bytes and checksums are unchanged.
- [ ] Focused Lean planning/set tests and Go artifactv2 tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
