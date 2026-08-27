---
satisfies: [R1, R2, R6, R8]
---
# fn-18-versioned-umpire-artifact-boundary.3 Prove canonical v2 admission and hard rejection

## Description
Make fn-37's v2 DrivePlan and ExperimentSpec the sole persisted baseline and prove Lean/Go agreement before adding later families.


**Size:** M
**Files:** `tools/umpire/artifact/experiment.go`, focused tests and testdata, `model/Umpire/Artifact/Tests/Codecs.lean`
**Touches:** [tools/umpire/artifact/experiment.go, tools/umpire/artifact/experiment_test.go, tools/umpire/artifact/testdata/**, model/Umpire/Artifact/Tests/Codecs.lean]

### Approach
- Strictly decode every exact v2 field, re-encode canonical bytes, and independently recompute nested and outer Artifact Checksums.
- Validate Definition IDs, Behavior Fingerprints, Limits, Known Gaps, occurrences, checkpoints, Properties, requirements, and provenance.
- Reject earlier formats before field validation, then reject legacy keys, unknown keys, malformed values, checksum drift, noncanonical bytes, and missing or extra LF.
- Keep migrations reserved for a future reviewed post-v2 successor; implement no current migration registry.

### Investigation targets
**Required:** fn-37 v2 codecs, fixtures, and the parent early proof point.

## Acceptance
- [ ] The caller-closure and Switch v2 fixtures round-trip byte-for-byte with independently verified checksums.
- [ ] One-at-a-time mutations cover every required hard-rejection class.
- [ ] Any earlier current-Artifact format reports unsupported format before field-level validation.
- [ ] No migration, best-effort normalization, alias, or fallback is present.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestExperimentV2`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
