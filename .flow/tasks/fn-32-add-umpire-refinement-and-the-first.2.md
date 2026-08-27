---
satisfies: [R2, R3]
---

# fn-32-add-umpire-refinement-and-the-first.2 Apply checked Implementation Links with complete Evidence Links

## Description
Implement total checked trace correspondence and Evidence Links for R2 and R3.

### Review reconciliation (normative)

`applyImplementationLink checked sourceSetup evidenceBackedTrace` must replay the exact initial state and each step through the linked source kernel before translation. It uses positional coordinates only. The failure mapping is exhaustive: stale target/Behavior Fingerprint/setup, non-authoritative source transitions, and invalid coordinates are `invalid`; absent coordinates and Limit Reached are `unknown`; duplicate/contradictory coordinates, multiple mappings, and Evidence Link mismatch are `conflict`; explicit Known Gaps or out-of-partition vocabulary are `unsupported`. Every diagnostic has a canonical identity over Implementation Link, kind/status, coordinate, related identities, Limits/counts, and Known Gap provenance.

**Size:** M
**Files:** `model/Umpire/ImplementationLink/**`, `model/Umpire/ImplementationLink/Tests/**`
**Touches:** [model/Umpire/ImplementationLink/**]

### Approach
- Preserve `initialState`, step-index, and observation-position coordinates across repeated equal values.
- Return no partial Feature Model Trace on any non-success.
- Keep Implementation Link outcomes separate from Observation and Property outcomes.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Evaluation.lean` — Observation Evaluation and no-partial-trace precedent
- `model/Umpire/Observation/Tests/EvidenceLink.lean` — coordinate Evidence Link pattern
- `model/Umpire/Property/Language.lean` — downstream pure trace consumer

### Acceptance
- [ ] Every destination coordinate has one complete Implementation Link Evidence Link.
- [ ] Every named failure kind maps to exactly one invalid/unknown/conflict/unsupported outcome and canonical diagnostic identity.
- [ ] Failed Implementation Link cannot invoke Feature Property evaluation.
## Acceptance
- [ ] Every destination coordinate has one complete Implementation Link Evidence Link.
- [ ] Every named failure kind maps to exactly one invalid/unknown/conflict/unsupported outcome and canonical diagnostic identity.
- [ ] Failed Implementation Link cannot invoke Feature Property evaluation.
### Acceptance
- [ ] R2/R3 source-kernel admission, application, Limit, exhaustive status, and Evidence Link matrices pass.
- [ ] Repeated-value coordinates remain auditable.
- [ ] No partial destination trace is observable.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
