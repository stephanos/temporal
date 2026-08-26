---
satisfies: [R1, R3, R8]
---
# fn-18-versioned-umpire-artifact-boundary.5 Define the bounded raw-evidence transport

## Description
Implement R3's exact bounded RawEvidence schema/canonical projection and strict codec without adding evidence interpretation.

**Size:** M
**Files:** `model/Umpire/Artifact/Evidence.lean`, `model/Umpire/Artifact/Tests/Evidence.lean`, `tools/umpire/artifact/evidence.go`, `tools/umpire/artifact/evidence_test.go`
**Touches:** [model/Umpire/Artifact/Evidence.lean, model/Umpire/Artifact/Tests/Evidence.lean, tools/umpire/artifact/evidence.go, tools/umpire/artifact/evidence_test.go]

### Approach
- Define every exact RawEvidence source/fact/field/payload/gap field, enum, order, bound, and identity view from the parent normative schema.
- Implement canonical Lean encoding and strict Go decode/validate/re-encode; treat fields and payloads as inert typed transport data.
- Validate source-local monotonic ordinals, source closure/gaps, causal reference closure and acyclicity, correlation references, field disposition/value shapes, payload digest/size, and experiment/configuration/run binding consistency.
- Add cross-language closed and partial fixtures plus one-at-a-time source, ordinal, cycle, gap, closure, payload, disposition, count, identity, and binding mutations.
- Keep mapping, qualification, trace construction, and Property evaluation absent.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact/Runtime.lean` after Task `.4` — exact run/config binding types
- fn-4 raw typed evidence input contract without its interpretation implementation
- parent spec `Normative v1 wire contract` RawEvidence schema

### Acceptance
- [ ] RawEvidence cannot encode a semantic observation, qualification, trace, or Property verdict.
- [ ] Every source/fact/field/payload/closure invariant has a focused positive and negative case.
- [ ] The 16-MiB family, 1-MiB payload, 4096-fact, 128-field, and causal/reference limits are enforced before admission.
- [ ] Cross-artifact digest/reference drift fails deterministically.

## Acceptance
- [ ] R3 RawEvidence is exact, bounded, and transport-only.
- [ ] No interpretation or semantic duplicate exists.
- [ ] Focused Lean/Go tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
