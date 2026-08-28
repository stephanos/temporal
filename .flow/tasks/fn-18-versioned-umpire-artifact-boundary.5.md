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
- Implement exactly the parent `umpire-raw-evidence/v2` field/nested-record order,
  ArtifactBinding/provenance/checksum rules, three capture/source statuses, and closed fact/field
  value grammar.
- Preserve source identity, source-local order, causal links, typed fields, dispositions, closure, and exact Artifact bindings.
- Enforce 64 sources, 4,096 facts, 128 fields/fact, 1 MiB/fact, and 16 MiB aggregate decoded
  payload before allocation/append and reject ordinal gaps, forward/dangling references, or cycles.
- Keep Observation Evaluation, Model Trace construction, Property evaluation, and Claim Assessment absent.

### Investigation targets
**Required:** the parent RawEvidence contract and fn-4 Evidence input boundary.

## Acceptance
- [ ] Canonical cross-language fixtures preserve exact raw facts and bindings.
- [ ] Malformed types, ordering, closure, causality, disposition, Limit, and checksum mutations reject.
- [ ] RawEvidence cannot encode an accepted Model Fact or Property result.
- [ ] N/N+1 fixtures cover every evidence ceiling without truncation, and field disposition/value
  combinations reject prohibited raw values.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestRawEvidence`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
