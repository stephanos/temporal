---
satisfies: [R1, R3, R8]
---
# fn-18-versioned-umpire-artifact-boundary.5 Define the bounded RawEvidence transport

## Description
Persist bounded typed raw records without interpreting them as Model Facts.


**Size:** M
**Files:** `model/Umpire/Artifact/Evidence.lean`, tests, and `tools/umpire/artifact/evidence.go`
**Touches:** [model/Umpire/Artifact/Evidence.lean, model/Umpire/Artifact/Tests/Evidence.lean, tools/umpire/artifact/evidence.go, tools/umpire/artifact/evidence_test.go]

### Approach
- Preserve source identity, source-local order, causal links, typed fields, dispositions, closure, and exact Artifact bindings.
- Enforce collection and payload Limits before allocation and reject dangling or cyclic causal references.
- Keep Observation Evaluation, Model Trace construction, Property evaluation, and Claim Assessment absent.

### Investigation targets
**Required:** the parent RawEvidence contract and fn-4 Evidence input boundary.

## Acceptance
- [ ] Canonical cross-language fixtures preserve exact raw facts and bindings.
- [ ] Malformed types, ordering, closure, causality, disposition, Limit, and checksum mutations reject.
- [ ] RawEvidence cannot encode an accepted Model Fact or Property result.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestRawEvidence`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
