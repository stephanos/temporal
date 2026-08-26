---
satisfies: [R1, R4, R8]
---
# fn-18-versioned-umpire-artifact-boundary.6 Persist semantic evidence and Result without merging their semantics

## Description
Implement R4's exact persisted projection of fn-4 qualification/derivations and separate property Result, including the stable qualified-outcome seam.

**Size:** M
**Files:** `model/Umpire/Artifact/Result.lean`, `model/Umpire/Artifact/Tests/Result.lean`, `tools/umpire/artifact/result.go`, `tools/umpire/artifact/result_test.go`
**Touches:** [model/Umpire/Artifact/Result.lean, model/Umpire/Artifact/Tests/Result.lean, tools/umpire/artifact/result.go, tools/umpire/artifact/result_test.go]

### Approach
- Project fn-4 QualificationOutcome/QualifiedTrace/derivations/dispositions into the exact semantic-evidence records without copying mapping/evaluator logic.
- Define Result with artifact bindings and embedded semantic references resolved from RuntimeConfiguration/ExperimentSpec; no phantom query, Property, or program artifacts.
- Encode fn-4 verdict/clause/span/bound projections and enforce every row of the parent status matrix, query partition, trace/diagnostic rule, and derivation/disposition completeness invariant.
- Independently recompute the exact qualified-outcome and artifact identity views, proving transport exclusions and semantic sensitivity.
- Add cross-language satisfied, violated, qualified-incomplete, unknown, conflict, and unsupported fixtures plus single-field corruption/reference/status mutations.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/**` — qualification and derivation authority
- `model/Umpire/Property/Language.lean:1162-1228` — Property evaluation structure
- fn-4 R3/R6 qualification, verdict, and strict aggregation contracts
- parent spec `Normative v1 wire contract` SemanticEvidence, Result, status matrix, and identities

### Acceptance
- [ ] SemanticEvidence and Result remain distinct and neither codec interprets raw evidence or evaluates Property.
- [ ] Every status-matrix row and invalid combination is pinned, including query partitions and outcome nullability.
- [ ] Qualified outcome identity is stable across transport/time/path changes and sensitive to every allowed semantic field.
- [ ] Embedded semantic references and artifact bindings reject every drift.

## Acceptance
- [ ] R4 exact schemas, status separation, and outcome identity are implemented.
- [ ] Fn-4 semantic types remain the only qualification/verdict authority.
- [ ] Focused Lean/Go tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
