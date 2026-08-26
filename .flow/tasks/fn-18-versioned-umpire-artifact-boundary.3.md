---
satisfies: [R1, R2, R8]
---
# fn-18-versioned-umpire-artifact-boundary.3 Prove legacy v1 compatibility and the complete current schema

## Description
### Umpire4 reconciliation (normative)

Treat `umpire-experiment/v1` as a strict legacy compatibility input and prove the complete current schema separately. The current schema uses an explicit Umpire canonical JSON profile; protobuf deterministic serialization or ProtoJSON must not define portable identity.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement the R2/R1 compatibility proof gate: one strict Go decoder/encoder and invariant checker for the existing Lean wire contract.

**Size:** M
**Files:** `tools/umpire/artifact/experiment.go`, `tools/umpire/artifact/experiment_test.go`, `tools/umpire/artifact/testdata/**`, `model/Umpire/Artifact/Tests/Codecs.lean`
**Touches:** [tools/umpire/artifact/experiment.go, tools/umpire/artifact/experiment_test.go, tools/umpire/artifact/testdata/**, model/Umpire/Artifact/Tests/Codecs.lean]

### Approach
- Mirror every exact v1 field and nested shape without a partial envelope or unknown-field tolerance.
- Recompute DrivePlan and ExperimentSpec semantic views/identities independently, content/provenance digests, canonical order, bounds, positions, occurrence/action links, checkpoints, properties, requirements, omissions, and bindings.
- Admit the current 164-KiB caller-closure fixture and a Lean-generated non-empty choice/variant/fault fixture; re-encode exact persisted bytes.
- Generate one-at-a-time mutation cases for every identity, intent, bound, occurrence, property, observation, omission, provenance, key, token, and size relation.
- Treat any Go/Lean discrepancy as a contract bug; do not normalize, weaken, or create an alternate representation.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact/Experiment.lean` after Task `.1`
- `model/Temporal/Feature/Nexus/testdata/nexus-caller-closure-experiment-spec.json`
- `model/Umpire/Planning/Tests/Artifacts.lean:22-68`
- fn-16 R6 and artifact-intent task

### Acceptance
- [ ] Both golden families round-trip byte-for-byte and recomputed identities exactly match Lean.
- [ ] Every independent mutation fails with the intended structured classification.
- [ ] The 1-MiB boundary accepts N and rejects N+1 without truncation.
- [ ] Tasks `.4`–`.8` remain blocked until this proof passes.
## Acceptance
- [ ] The early compatibility proof implements R1/R2 with no permissive path.
- [ ] Existing v1 remains unchanged and non-empty intent is fully validated.
- [ ] Lean and Go focused suites pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
