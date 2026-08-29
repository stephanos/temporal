---
satisfies: [R4, R5, R6]
---

# fn-20-local-execution-semantic-conformance.1 Compose Observation Evaluation, Implementation Link, and Property results behind one Run Evaluation API

## Description
### Umpire4 reconciliation (normative)

The reusable Lean check API must implement the full altitude chain: checked System Observation mapping -> Evidence-backed System Model Trace -> checked Implementation Link -> Feature Model Trace -> unchanged Feature Property evaluation. Observation, Implementation Link, and Property outcomes and Evidence Links remain independently represented.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Add the small domain-neutral semantic-check deep module consumed by the private checker (R4/R5/R6). It composes fn-4 Observation Evaluation and verdict APIs without adding Temporal, transport, or plan-identity meaning.

**Size:** M
**Files:** `model/Umpire/Observation/Check.lean`, `model/Umpire/Observation/Tests/Check.lean`, `model/Umpire/Observation.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Observation/Check.lean, model/Umpire/Observation/Tests/Check.lean, model/Umpire/Observation.lean, model/UmpireTests.lean]

### Approach
- Accept only checked mapping/query/Property values plus one bounded typed EvidenceBundle and return a complete Observation Evaluation/verdict Generated View.
- Reuse fn-4 Observation Evaluation, coordinate bijection checks, `CheckedProperty.traceView`, `evaluateProperty`, and strict aggregation without changing their denotation.
- Keep non-accepted outcomes total: emit the complete matching non-resolved Property partition without invoking evaluation.
- Stop at the domain-neutral Observation Evaluation/verdict Generated View. Do not accept an ExperimentSpec plan, compute `evaluationOutcomeChecksum`, or mirror fn-18 transport; Task `.2` owns the plan-sensitive composition.
- Pin deterministic ordering, exactly-at-bound behavior, independent operational absence, and proof that Property inputs are immutable/reusable.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-4-umpire-observation-and-semantic-verdicts.2.md` — Observation Evaluation/Evidence Link contract that must be reused
- `.flow/tasks/fn-4-umpire-observation-and-semantic-verdicts.3.md` — verdict/aggregation contract that must be reused
- `model/Umpire/Property/Language.lean:687-740` — capability-limited trace Generated View
- `model/Umpire/Property/Language.lean:1133-1218` — existing evaluator structures
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.6.md` — downstream identity inputs deliberately excluded here
## Acceptance
- [ ] One call returns a complete accepted/non-accepted semantic result with canonical Property ordering and no partial trace or verdict set.
- [ ] Non-accepted inputs provably skip Property evaluation; accepted inputs preserve the existing evaluator's clause results.
- [ ] The API contains no ExperimentSpec plan, transport binding, operational status, or accepted-outcome identity computation.
- [ ] Reusable modules import no Temporal, artifact IO, process, or command package.
- [ ] Focused Lean tests cover satisfied, violated, Observation Evaluation non-success, incomplete verdicts, repeated values, and N/N+1 evidence.
## Done summary
Implemented the domain-neutral Run Evaluation seam across checked Observation Evaluation, Evidence-backed System Model Trace, checked Implementation Link, Feature Model Trace, and unchanged Property evaluation. The result retains Observation, Implementation Link, and canonical Property/Query outcomes independently, fails closed on every non-accepted altitude, and contains no Temporal, Artifact, process, plan, operational, or outcome-identity meaning.

Focused tests cover satisfied and violated Properties, Observation and Implementation Link non-success, incomplete verdict partitions, repeated values, deterministic canonical ordering, exact N/N+1 evidence behavior, logical-time prerequisites, and destination/query target binding. Memory capture was attempted after review fixes but the repository memory store is not initialized.

baseline: red (cd model && mise exec -- lake build Umpire.Observation.Tests.Check failed pre-edit: task target absent)

GATE_RECEIPT_NOT_WRITTEN:unittest:inherited protected config/development.yaml dirty state made receipt non-warrantable

stage: impl-review - ran [2026-08-29T22:29:27Z..2026-08-29T22:40:02Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 12a1acd2d9c22f839215273930a4199e9756619f, e9a28ac0b0b52dca194c0643783b984ec1f7b81e
- Tests: baseline: red (cd model && mise exec -- lake build Umpire.Observation.Tests.Check failed pre-edit: task target absent), cd model && mise exec -- lake build Umpire.Observation.Tests.Check, cd model && mise exec -- lake build Umpire.Observation.ImportTests Umpire.Observation.Tests, make umpire-check-regression, make lint-model, GATE_RECEIPT_NOT_WRITTEN:unittest:inherited protected config/development.yaml dirty state made receipt non-warrantable
- PRs: