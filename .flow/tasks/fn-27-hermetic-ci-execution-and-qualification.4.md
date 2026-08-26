---
satisfies: [R4, R5, R8]
---
# fn-27-hermetic-ci-execution-and-qualification.4 Add CI provenance and QualificationReceipt v2 codecs

## Description
Implement the exact cross-language CI provenance and receipt codecs from R4/R5 without changing local schemas.

**Size:** M
**Files:** `model/Umpire/Qualification/**`, `model/Umpire/Artifact/Qualification.lean`, `model/Umpire/Artifact/Tests/Qualification.lean`, `tools/umpire/artifact/qualification.go`, `tools/umpire/artifact/qualification_test.go`
**Touches:** [model/Umpire/Qualification/**, model/Umpire/Artifact/Qualification.lean, model/Umpire/Artifact/Tests/Qualification.lean, tools/umpire/artifact/qualification.go, tools/umpire/artifact/qualification_test.go]

### Approach

- Add exact Lean/Go nested records, strict codecs, validators, canonical encoders, two-stage isolation outcomes, semantic digest projection, receipt/artifact identities, and limits for `CIProvenance/v1` and QualificationReceipt v2.
- Preserve QualificationReceipt v1 byte/token/cardinality behavior exactly and add explicit cross-version rejection rather than a permissive common decoder.
- Reuse the existing pathless Result reference, pilot/status/evidence/cleanup fields, omission bounds, and ArtifactProvenance; exclude nested/outer provenance from the receipt semantic projection exactly as specified.
- Implement the complete CI-isolation reason table, compound accumulation, rejected-over-incomplete precedence, and exact expected omission set without changing fn-26's reason table.
- Pin cross-language equality fixtures and mutate every nested field, order, nullability, enum, identity projection, omission, limit, and version boundary.

### Investigation targets

**Required** (read before coding):
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — canonical codecs, set validation, and publication
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — exact v1 receipt and v2 set contract
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.3.md` — local receipt/set implementation seam
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.4.md` — decision construction and precedence

### Acceptance

- [ ] CI provenance and v2 receipt round-trip byte-for-byte across Lean/Go at exact limits and reject every nested N+1/crossed/stale/projection mutation.
- [ ] Preflight cannot stand in for postflight; every isolation row and compound status has one deterministic reason/decision result.
- [ ] V1 receipt fixtures/readers remain byte-identical and reject v2.
- [ ] No set/publication, migration, repair, or permissive reader is introduced in this task.

## Acceptance
- [ ] R4/R5 provenance/receipt schemas, bindings, identity projections, limits, and reasons are complete.
- [ ] Cross-language equality and full nested/version/status mutation matrices pass.
- [ ] Existing artifact comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
