---
satisfies: [R3, R4, R6]
---
# fn-4-umpire-observation-and-semantic-verdicts.3 Produce semantic property verdicts and strict query summaries

## Description
Add the qualification preflight, structured per-property verdicts, and strict query aggregation for R3/R4/R6 while leaving the Property evaluator unchanged.

**Size:** M
**Files:** `model/Umpire/Observation/Verdict.lean`, `model/Umpire/Observation.lean`, `model/Umpire/Observation/Tests/Fixtures.lean`, `model/Umpire/Observation/Tests/Verdict.lean`, `model/Umpire/Observation/Tests/Aggregation.lean`
**Touches:** [model/Umpire/Observation/Verdict.lean, model/Umpire/Observation.lean, model/Umpire/Observation/Tests/Fixtures.lean, model/Umpire/Observation/Tests/Verdict.lean, model/Umpire/Observation/Tests/Aggregation.lean]

### Approach

- Validate qualification, coordinate/derivation bijection, vocabulary, applied evidence bound, and logical time before evaluation.
- Wrap existing clause results with relevant coordinate-keyed derivations, applied bound, and provenance.
- Aggregate independent property results deterministically without discarding unresolved entries.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Language.lean:1133-1218` — denotation and structured evaluation.
- `model/Umpire/Property/Language.lean:732-740` — mandatory trace view.
- `model/Umpire/Query/Language.lean:41-61` — checked query property forms.
- `model/Umpire/Query/Language.lean:157-172` — query identity/context.
- `model/Umpire/Query/Language.lean:225-233` — property-set validation.

## Acceptance
- [ ] Supported qualified traces preserve existing clause results as satisfied/violated.
- [ ] Unknown/conflict/unsupported qualification never invokes property evaluation.
- [ ] Missing required logical time is unknown, not violated.
- [ ] Bound-exhausted and non-bijective qualified inputs never invoke the Property evaluator.
- [ ] Aggregation tests cover satisfied, violation, missing, duplicate, unknown, conflict, unsupported, mixed, and divergent inputs.
- [ ] Verdicts link clauses, coordinate spans, query/evidence bounds, provenance, and relevant derivations even when equal values repeat.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
