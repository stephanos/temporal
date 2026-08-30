---
satisfies: [R3, R4, R5]
---
# fn-48-canonicalize-known-gaps-as-a-checked-set.4 Use checked Known Gap unions in Result and Run Evaluation

## Description
Finish the Lean consumer cutover in interpreted Evidence, Result artifacts, and composed Run Evaluation (R3-R5).

**Size:** M
**Files:** `model/Umpire/Artifact/Result.lean`, `model/Umpire/Artifact/Tests/Result.lean`, `model/Temporal/Tool/RunEvaluation.lean`, `model/Temporal/Tool/RunEvaluationTests.lean`, `model/Temporal/Tool/RunEvaluationMutationTests.lean`, `tools/umpire/runevaluation/protocol.go`, `tools/umpire/runevaluation/result_test.go`
**Touches:** [model/Umpire/Artifact/Result.lean, model/Umpire/Artifact/Tests/Result.lean, model/Temporal/Tool/RunEvaluation.lean, model/Temporal/Tool/RunEvaluationTests.lean, model/Temporal/Tool/RunEvaluationMutationTests.lean, tools/umpire/runevaluation/protocol.go, tools/umpire/runevaluation/result_test.go]

### Approach
- Remove only the Lean Run Evaluation private rank/key/sort/dedup implementation and return checked collections from Lean parsing.
- Compose run, raw-evidence, Observation, interpreted Evidence, and Result gaps through checked union, preserving phase-owned status mapping.
- Retain Go protocol validation/canonical response verification unchanged except for tests that pin the independent boundary.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact/Result.lean:134-160,250-265,420-445,560-585,727-790` — interpreted Evidence/Result fields, rendering, validation
- `model/Temporal/Tool/RunEvaluation.lean:338-377` — duplicate Lean canonicalization and parsing
- `model/Temporal/Tool/RunEvaluation.lean:718-762,897-960` — phase unions and response projection
- `model/Temporal/Tool/RunEvaluationTests.lean:398-403` — strict Lean protocol coverage
- `tools/umpire/runevaluation/protocol.go:370-410` — independent Go union/verification boundary
- `model/Temporal/Tool/RunEvaluationMutationTests.lean` — independent phase mutation oracles
## Acceptance
- [ ] Interpreted Evidence, Result, and Lean Run Evaluation contain no private Lean Known Gap ranking, sorting, deduplication, or raw-list validation body.
- [ ] Exact duplicates across checked Lean phases collapse; cross-phase conflicts fail closed with no partial Result.
- [ ] Go admission/response verification remains independent and fully tested.
- [ ] Existing protocol errors, result statuses, generated views, canonical bytes, checksums, Result, Run Evaluation, mutation, and Go tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
