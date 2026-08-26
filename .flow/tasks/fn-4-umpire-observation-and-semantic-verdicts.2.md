---
satisfies: [R2, R3, R4, R5]
---
# fn-4-umpire-observation-and-semantic-verdicts.2 Qualify synthetic evidence with derivations and dispositions

## Description
Implement the pure checked-plan plus typed-bundle qualification boundary for R2-R5. Verdict evaluation remains in task 3.

**Size:** M
**Files:** `model/Umpire/Observation/Qualification.lean`, `model/Umpire/Observation/Tests/Fixtures.lean`, `model/Umpire/Observation/Tests/Qualification.lean`, `model/Umpire/Observation/Tests/Derivation.lean`, `model/Umpire/Observation/Tests/Disposition.lean`
**Touches:** [model/Umpire/Observation/Qualification.lean, model/Umpire/Observation/Tests/Fixtures.lean, model/Umpire/Observation/Tests/Qualification.lean, model/Umpire/Observation/Tests/Derivation.lean, model/Umpire/Observation/Tests/Disposition.lean]

### Approach

- Debit one `evidence-records` unit per input before normalization, returning canonical bound-exhausted `unknown` without a partial trace or further evaluation.
- Normalize and bind evidence through the checked closed expressions, establish deterministic ordering/closure, and construct a wrapper around `SemanticTrace`.
- Define stable one-based semantic coordinates for initial state and every step action, outcome, resulting state, and observation; attach exactly one complete derivation to every coordinate.
- Validate a bijection between trace slots and derivations so repeated equal semantic values cannot collapse.
- Enforce retain/redact/hash/reject and raw-value non-retention.
- Preserve compatible alternatives and classify unknown/conflict/unsupported without a partial trace.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:94-105` — immutable pure trace boundary.
- `model/Umpire/Property/Language.lean:687-740` — capability-admitted projection.
- `model/Umpire/Property/Language.lean:840-846` — missing-coordinate handling.
- `model/Umpire/Property/Tests/Evaluation.lean:53-84` — hidden-value non-interference.
- `model/Umpire/Property/Tests/LogicalTime.lean:33-78` — logical-time edge cases.

## Acceptance
- [ ] Complete closed synthetic evidence produces the independently authored qualified trace.
- [ ] Exactly-at-limit input qualifies normally; limit-plus-one input is bound-exhausted unknown and exposes no partial trace.
- [ ] Every R2 status and R3 derivation failure has an exact test.
- [ ] Repeated equal values retain distinct coordinates, while missing, duplicate, and extra coordinate derivations fail.
- [ ] Compatible alternatives remain visible and ordering never selects one.
- [ ] Disposition tests prove forbidden raw values never appear.
- [ ] No failure exposes a partial `SemanticTrace`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
