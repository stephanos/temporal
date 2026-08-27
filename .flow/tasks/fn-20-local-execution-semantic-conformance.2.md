---
satisfies: [R2, R3, R4]
---

# fn-20-local-execution-semantic-conformance.2 Build the exact Temporal Nexus Implementation Link evaluator

## Description
### Umpire4 reconciliation (normative)

The Temporal checker composes `Temporal.System.Nexus.Observation` with `Temporal.System.Nexus.ImplementationLink`; it must never map raw evidence directly to Feature facts. Unknown, unsupported, unaccepted, or failed Implementation Link remains distinct from a Property result.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement the fixed Lean side of the private checker bridge and the sole live-evidence adapter (R2/R3/R4/R5), using Task `.1` as the semantic entry point.

**Size:** M
**Files:** `model/Temporal/Tool/RunEvaluation/Protocol.lean`, `model/Temporal/Tool/RunEvaluation.lean`, `model/Temporal/Tool/RunEvaluationTests.lean`, `model/Temporal/Feature/Nexus/Observation.lean`, `model/lakefile.toml`
**Touches:** [model/Temporal/Tool/RunEvaluation/Protocol.lean, model/Temporal/Tool/RunEvaluation.lean, model/Temporal/Tool/RunEvaluationTests.lean, model/Temporal/Feature/Nexus/Observation.lean, model/lakefile.toml]

### Approach
- Register the closed checker identity/version/digest and exactly one caller-closure declaration closure; resolve every request identity against compiled checked values.
- Decode only the private direct Generated View with the four exact non-path admitted artifact-binding tuples, separate Run/RawEvidence Known Gaps, exact canonical request shape, and Limits; never read a file, manifest, artifact member, environment option, or arbitrary extension.
- Freeze the fn-19 source schema/version/digest table after that dependency lands and translate its four source kinds into fn-4's typed EvidenceBundle while preserving order, causality, gaps, closure, correlations, and dispositions.
- Call Task `.1`; then compose its Observation Evaluation/verdict Generated View with the exact compiled ExperimentSpec plan and checked program/mapping/query/Property values to compute fn-18's `evaluationOutcomeChecksum` in the Lean authority.
- Emit mapping/Observation Evaluation-only `observationKnownGaps` and the canonical exact-value union `resultKnownGaps` from request Run Known Gaps, RawEvidence Known Gaps, and semantic Known Gaps; keep unknown/conflict/unsupported distinct from protocol failure.
- Register `temporal-run-evaluation-checker` and prove stdin/stdout/stderr bytes, exit behavior, request/response N/N+1 Limits, and deterministic repeated checking.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-4-umpire-observation-and-semantic-verdicts.4.md` — Temporal-owned mapping/profile seam
- `model/Temporal/Feature/Nexus/CallerClosure.lean:441-471` — exact checked Property subject
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean` — current scenario fixture/testing pattern
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.7.md` — exact four-source producer
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.6.md` — plan-sensitive outcome identity and Known Gap projections
- `model/lakefile.toml:1-20` — executable registration pattern
## Acceptance
- [ ] The checker accepts only the exact compiled caller-closure experiment/program/mapping/query/Property/source closure and exact echoed artifact bindings; every drift rejects deterministically.
- [ ] Four-source mapping preserves source-local and causal facts; incomplete/ambiguous/conflicting/unsupported/disposition cases receive the correct fn-4 outcome without guessed order.
- [ ] Accepted-outcome identity includes the compiled ExperimentSpec plan plus every fn-18 stable semantic input and excludes only the specified transport/run fields.
- [ ] Semantic Known Gaps and the canonical Result Known Gap union follow the parent contract byte-for-byte and preserve upstream auditability through bound artifacts.
- [ ] The executable performs no filesystem, network, environment-authority, artifact admission/publication, or Temporal runtime operation.
- [ ] Canonical protocol, 32-MiB N/N+1, and no-stderr deterministic success tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
