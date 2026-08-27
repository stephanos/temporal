---
satisfies: [R2, R5, R6]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.5 Emit v2-only Lean artifacts with exact checksums

## Description
Implement the Lean half of R5 and every checked-in Lean fixture affected by R2/R5/R6. Replace the identity/digest wire contract with canonical v2 DrivePlan and ExperimentSpec Artifacts using the approved names and exact Artifact Checksum semantics.

**Size:** L
**Files:** `model/Umpire/Artifact.lean`, `model/Temporal/Tool/Inspect.lean`, inspector tests, and every current Switch/Nexus Artifact fixture and producer
**Touches:** [model/Umpire/Artifact.lean, model/Umpire/Examples/Switch*.lean, model/Umpire/Examples/Fixtures/SwitchCompiledArtifact.json, model/Umpire/Examples/testdata/switch-experiment-spec.json, model/Temporal/Tool/Inspect*.lean, model/Temporal/Feature/Nexus/Operations*.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosure*.lean, model/Temporal/Feature/Nexus/Fixtures/OperationsAsyncStartArtifact.json, model/Temporal/Feature/Nexus/Fixtures/OperationsCancellationArtifact.json, model/Temporal/Feature/Nexus/Fixtures/OperationsSuccessfulCompletionArtifact.json, model/Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json]

### Approach
- Rename Artifact record fields by meaning, including Definition ID references, Behavior Fingerprints, Model preconditions, expanded Limits, exact Known Gap rows, and Artifact Checksums.
- Emit only `umpire-drive-plan/v2` and `umpire-experiment/v2`; remove v1 constants, keys, expectations, and compatibility comments.
- Define canonical bytes as the canonical Lean JSON object followed by exactly one LF. Use a fixed field order, no insignificant whitespace, canonical string escaping, and canonical base-10 natural numbers.
- Compute each Artifact Checksum over its complete canonical object with only its own checksum field absent; ExperimentSpec includes the complete nested DrivePlan representation and its checksum.
- Keep deterministic field/set ordering and current modeled trace content; Source Locations and complete Known Gap rows participate in the Artifact Checksum.
- Regenerate all six Artifact goldens from their existing authoritative Lean producers: `SwitchCompiledArtifact.json`, `switch-experiment-spec.json`, three `Operations*Artifact.json` files, and `nexus-caller-closure-experiment-spec.json`.
- Add one-at-a-time mutation tests for every checksum-bearing content category and domain separation between the two Artifact kinds.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact.lean:30-110,185-245,285-355` — current structures, JSON boundary, lowering, and identity derivation.
- `model/Temporal/Tool/Inspect.lean` and `InspectTests.lean` — authoritative fixture emitter and contract tests.
- `model/Umpire/Examples/SwitchTests.lean` — Switch compiled and ExperimentSpec fixture producers.
- `model/Temporal/Feature/Nexus/OperationsTests.lean` — three Nexus Operations Artifact producers.
- `model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean` — Nexus caller-closure Artifact producer.
- All six JSON output paths listed in **Touches** — complete golden set that must be replaced, never hand-edited.

### Key context
There is no Lean v1 reader today and this task must not add one. The checksum covers provenance and the complete nested plan because it identifies exact Artifact content, unlike a Behavior Fingerprint.
## Acceptance
- [ ] Lean emits only the two exact v2 format versions, replacement JSON keys, and canonical bytes ending in exactly one LF.
- [ ] DrivePlan and ExperimentSpec checksums are reproducible, domain-separated, and cover all canonical content except their own checksum field.
- [ ] `SwitchCompiledArtifact.json`, `switch-experiment-spec.json`, all three `Operations*Artifact.json` files, and `nexus-caller-closure-experiment-spec.json` are regenerated from authoritative Lean producers and retain the same selected Model Traces and Properties.
- [ ] Mutation tests prove every content category—including complete Known Gap rows, provenance, nested plan content, and format version—changes the owning checksum.
- [ ] No v1 constant, serializer branch, reader, or migration exists in Lean.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
