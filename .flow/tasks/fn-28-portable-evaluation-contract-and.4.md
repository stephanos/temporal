---
satisfies: [R3, R4, R8]
---

# fn-28-portable-evaluation-contract-and.4 Interpret portable Observation, link, and Property clauses in Go
## Description

Implement the generic bounded Go interpreter that consumes one admitted contract plus closed Raw Evidence and produces the existing detailed Run Evaluation dimensions without invoking Lean.

**Size:** L
**Files:** `tools/umpire/portableevaluation/**`
**Touches:** [`tools/umpire/portableevaluation/**`]

### Approach
- Normalize only contract-declared fields, validate source/correlation/causal and source-local order, enforce closure, construct the exact System trace, apply the bundled link, and evaluate the bundled Property clauses.
- Preserve `satisfied`, `violated`, `unknown`, `conflict`, and `unsupported`; never turn missing closure, ambiguity, unsupported data, or deadline into success or a false violation.
- Implement only the parent spec's exact version-one operator table, tagged types, diagnostic mapping, canonical evaluation order, and precharged work accounting; keep every operator switch exhaustive and fail closed on unknown values.

### Investigation targets

**Required** (read before coding):
- Parent spec, contract proto/admission, and existing `runevaluation` protocol/result validation.
- `Umpire.Observation.Evaluation`, `Umpire.Observation.Check`, and caller-closure Run Evaluation tests.
- Existing Evidence Link, disposition, causal-order, source-closure, and Limit representations.

## Acceptance
- [ ] Focused fixtures cover every version-one operator's success/failure/type/missing/N/N+1 branches plus accepted/satisfied, accepted/violated, unknown, conflict, unsupported, and incomplete closure without Lean.
- [ ] Every accepted Model Fact and clause retains auditable Evidence Links and exact contract bindings.
- [ ] Mutation, N/N+1, cancellation, race, fuzz, and lint checks pass without adding a second model registry.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
