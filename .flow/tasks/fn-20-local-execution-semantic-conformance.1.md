---
satisfies: [R4, R5, R6]
---
# fn-20-local-execution-semantic-conformance.1 Compose Observation refinement and verdicts behind one Lean check API

## Description
### Umpire4 reconciliation (normative)

The reusable Lean check API must implement the full altitude chain: checked System Observation mapping -> qualified System trace -> checked Refinement -> Feature trace -> unchanged Feature Property evaluation. Observation, Refinement, and Property outcomes and derivations remain independently represented.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Add the small domain-neutral semantic-check deep module consumed by the private checker (R4/R5/R6). It composes fn-4 qualification and verdict APIs without adding Temporal, transport, or plan-identity meaning.

**Size:** M
**Files:** `model/Umpire/Observation/Check.lean`, `model/Umpire/Observation/Tests/Check.lean`, `model/Umpire/Observation.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Observation/Check.lean, model/Umpire/Observation/Tests/Check.lean, model/Umpire/Observation.lean, model/UmpireTests.lean]

### Approach
- Accept only checked mapping/query/Property values plus one bounded typed EvidenceBundle and return a complete qualification/verdict projection.
- Reuse fn-4 qualification, coordinate bijection checks, `CheckedProperty.traceView`, `evaluateProperty`, and strict aggregation without changing their denotation.
- Keep non-qualified outcomes total: emit the complete matching non-resolved Property partition without invoking evaluation.
- Stop at the domain-neutral qualification/verdict projection. Do not accept an ExperimentSpec plan, compute `qualifiedOutcomeIdentity`, or mirror fn-18 transport; Task `.2` owns the plan-sensitive composition.
- Pin deterministic ordering, exactly-at-bound behavior, independent operational absence, and proof that Property inputs are immutable/reusable.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-4-umpire-observation-and-semantic-verdicts.2.md` — qualification/derivation contract that must be reused
- `.flow/tasks/fn-4-umpire-observation-and-semantic-verdicts.3.md` — verdict/aggregation contract that must be reused
- `model/Umpire/Property/Language.lean:687-740` — capability-limited trace projection
- `model/Umpire/Property/Language.lean:1133-1218` — existing evaluator structures
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.6.md` — downstream identity inputs deliberately excluded here
## Acceptance
- [ ] One call returns a complete qualified/non-qualified semantic result with canonical Property ordering and no partial trace or verdict set.
- [ ] Non-qualified inputs provably skip Property evaluation; qualified inputs preserve the existing evaluator's clause results.
- [ ] The API contains no ExperimentSpec plan, transport binding, operational status, or qualified-outcome identity computation.
- [ ] Reusable modules import no Temporal, artifact IO, process, or command package.
- [ ] Focused Lean tests cover satisfied, violated, qualification non-success, incomplete verdicts, repeated values, and N/N+1 evidence.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
