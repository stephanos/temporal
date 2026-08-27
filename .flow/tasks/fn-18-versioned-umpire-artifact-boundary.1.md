---
satisfies: [R2, R8]
---
# fn-18-versioned-umpire-artifact-boundary.1 Adopt the v2 Artifact baseline and vertical package

## Description
Place fn-37's canonical v2 DrivePlan and ExperimentSpec behind the vertical Artifact facade without changing bytes or introducing a second format.


**Size:** M
**Files:** `model/Umpire/Artifact.lean`, `model/Umpire/Artifact/**`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Artifact.lean, model/Umpire/Artifact/**, model/UmpireTests.lean]

### Approach
- Preserve the existing declarations and comments while moving implementation behind focused modules.
- Reuse Definition IDs, Behavior Fingerprints, Artifact Checksums, Limits, and Known Gaps exactly as fn-37 defines them.
- Keep `umpire-drive-plan/v2` and `umpire-experiment/v2` as the sole supported current formats.
- Add no reader, migration, alias, or fallback for an earlier prototype format.

### Investigation targets
**Required:** fn-37 Artifact modules and the parent v2-baseline contract.

## Acceptance
- [ ] Existing v2 canonical bytes and Artifact Checksums remain byte-identical.
- [ ] Public imports expose one vertical Artifact package with comments preserved.
- [ ] No earlier-format reader, alternate writer, compatibility alias, or inferred missing intent exists.

### Quick command

`cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
