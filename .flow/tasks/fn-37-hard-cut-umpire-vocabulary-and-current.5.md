---
satisfies: [R2, R5, R6]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.5 Emit v2-only Lean artifacts with exact checksums

## Description
Implement the Lean half of R5 and the source fixtures for R6. Replace the current identity/digest wire contract with canonical v2 DrivePlan and ExperimentSpec artifacts using the approved names and exact Artifact Checksum semantics.

**Size:** M
**Files:** `model/Umpire/Artifact.lean`, `model/Temporal/Tool/Inspect.lean`, inspector tests, Switch/Nexus artifact fixtures and producers
**Touches:** [model/Umpire/Artifact.lean, model/Umpire/Examples/**, model/Temporal/Tool/Inspect*.lean, model/Temporal/Feature/Nexus/**/*.lean, model/Temporal/Feature/Nexus/**/testdata/*.json]

### Approach
- Rename artifact record fields by meaning, including Definition ID references, Behavior Fingerprints, model preconditions, expanded Limits, Known Gaps, and Artifact Checksums.
- Emit only `umpire-drive-plan/v2` and `umpire-experiment/v2`; remove v1 constants, keys, expectations, and compatibility comments.
- Compute each Artifact Checksum over its complete canonical object with only its own checksum field absent; ExperimentSpec includes the complete nested DrivePlan representation and checksum.
- Keep deterministic field/set ordering and current modeled trace content; source locations and Known Gaps now participate in the Artifact Checksum.
- Replace checked-in Switch and Nexus fixtures through their Lean producers and update inspector tests.
- Add one-at-a-time mutation tests for every checksum-bearing content category and domain separation between the two artifact kinds.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact.lean:30-110` — current structures and omissions.
- `model/Umpire/Artifact.lean:185-245` — current canonical JSON boundary.
- `model/Umpire/Artifact.lean:285-355` — current lowering and identity derivation.
- `model/Temporal/Tool/Inspect.lean` — authoritative fixture emitter.
- `model/Temporal/Tool/InspectTests.lean` — inspector contract tests.
- `model/Umpire/Examples/Fixtures/SwitchCompiledArtifact.json` — small golden artifact.
- `model/Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json` — Nexus vertical-slice artifact.

### Key context
There is no Lean v1 reader today and this task must not add one. The checksum covers provenance and the complete nested plan because it identifies exact Artifact content, unlike a Behavior Fingerprint.

## Acceptance
- [ ] Lean emits only the two exact v2 format versions and replacement JSON keys.
- [ ] DrivePlan and ExperimentSpec checksums are reproducible, domain-separated, and cover all canonical content except their own field.
- [ ] Switch and Nexus fixtures are replaced with valid v2 bytes while retaining the same selected model traces and Properties.
- [ ] Mutation tests prove every content category changes the owning checksum.
- [ ] No v1 constant, serializer branch, reader, or migration exists in Lean.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
