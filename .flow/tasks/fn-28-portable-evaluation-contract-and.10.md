---
satisfies: [R4, R8]
---

# fn-28-portable-evaluation-contract-and.10 Prove fail-closed closure, mutation, and resident reuse
## Description

Complete the cross-module negative matrix for portable contracts, eventual Evidence closure, HTTP transport, executor reuse, and disposable-cluster cleanup.

**Size:** L
**Files:** `tools/umpire/evaluationcontract/*_test.go`, `tools/umpire/portableevaluation/*_test.go`, `tools/umpire/executor/*_test.go`, `tools/umpire/executorhttp/*_test.go`, `tests/umpire4_portable_executor_test.go`
**Touches:** [`tools/umpire/evaluationcontract/*_test.go`, `tools/umpire/portableevaluation/*_test.go`, `tools/umpire/executor/*_test.go`, `tools/umpire/executorhttp/*_test.go`, `tests/umpire4_portable_executor_test.go`]

### Approach
- Mutate every binding/operator/Limit/closure/status seam independently and require the responsible stage to reject it without partial success.
- Exercise delayed-but-closed Evidence, deadline-before-closure, post-closure records, duplicate/source-crossed facts, stale run correlations, overlapping requests, cancellation, cleanup uncertainty, and poisoned-executor reuse.
- Keep global/model claims outside the matrix; these tests prove only one exact contract's local evaluation behavior.

### Investigation targets

**Required** (read before coding):
- Parent tasks `.2`, `.4`, `.6`, `.8`, and `.9` test matrices.
- Existing Umpire mutation tests and source-closure semantics.
- Repository race/fuzz and eventual-consistency test patterns; use `require.Eventually`, never sleep.

## Acceptance
- [ ] Exact N succeeds and N+1 fails for contract, body, Evidence, operator, and time/work Limits at the responsible seam.
- [ ] Missing/late/ambiguous/conflicting/unsupported Evidence and uncertain cleanup produce `inconclusive`, never pass or an invented violation.
- [ ] Crossed bindings, stale correlations, unknown operators, overlapping admission, cancellation leaks, post-closure Evidence, and reuse after poisoning fail closed under unit, race, fuzz, and tagged integration tests.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
