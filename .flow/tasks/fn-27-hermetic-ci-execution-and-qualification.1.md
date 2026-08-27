---
satisfies: [R1, R6]
---
# fn-27-hermetic-ci-execution-and-qualification.1 Pin the byte-identical v2 Artifact for ordinary CI tests

## Description
Freeze the exact canonical v2 `ExperimentSpec` already used by the local Nexus path as the sole CI semantic input. Generate an ordinary Go test from that admitted Artifact without recompiling or reconstructing its definitions. Check the exact bytes, format version, Artifact Checksum, Definition IDs, Behavior Fingerprints, Limits, Known Gaps, query, Properties, Observation program, and Implementation Link before runtime IO.

## Acceptance
- [ ] CI and local tests consume byte-identical canonical v2 Artifact bytes.
- [ ] One-byte, version, checksum, fingerprint, closure, or generated-output drift fails before runtime IO.
- [ ] No CI Evaluation Profile, provenance schema, or semantic copy is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
