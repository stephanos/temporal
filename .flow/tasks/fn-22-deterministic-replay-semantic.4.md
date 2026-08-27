---
satisfies: [R4, R5]
---
# fn-22-deterministic-replay-semantic.4 Run deterministic canonical replay and semantic minimization

## Description
### Umpire4 reconciliation (normative)

Minimization operates on checked semantic candidates and an accepted witness, with canonical semantic replay as the acceptance oracle. It does not minimize logs in place, invoke live side effects implicitly, or use SDK history replay as semantic authority.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement the bounded monotonic reducer over compiled candidates. Enumerate action, ordering, and fault edits in the spec's literal order and carry the candidate authority's explicit non-applicable configuration class into the report; reject a candidate after one conclusive mismatch and confirm an accepted candidate with a second isolated execution. Enforce at most eight semantic edit trials, twelve live executions, and 25 minutes including cleanup; record every proposed edit, compilation rejection, execution classification, acceptance, and untried suffix. Separately compute the smallest closed diagnostic EvidenceCore by pruning responsible clause/Observation/derivation/fact/receipt references in fixed order against the admitted dependency graph, never by rewriting RawEvidence or feeding a reduced capture to Claim Assessment.

**Size:** M
**Files:** `tools/umpire/replay/reducer.go`, `tools/umpire/replay/evidence_core.go`, `tools/umpire/replay/reducer_test.go`, `tools/umpire/replay/evidence_core_test.go`
**Touches:** [tools/umpire/replay/reducer.go, tools/umpire/replay/evidence_core.go, tools/umpire/replay/reducer_test.go, tools/umpire/replay/evidence_core_test.go]
## Acceptance
Accepted reductions preserve the original ViolationSignature on two isolated accepted replays and are never reintroduced. Configuration is emitted exactly once as non-applicable and consumes neither edit nor execution budget. Zero accepted edits after all applicable conclusive trials returns complete irreducible success. An indeterminate trial or exhausted bound stops with exact untried work and blocks promotion. EvidenceCore is deterministic, dependency-closed, refers only to retained admitted artifacts, omits every removable nonessential reference, and leaves all six input artifacts byte-identical.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
