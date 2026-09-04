---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-65-design-and-prototype-approachable.2 Compare interfaces and draft the concrete Nexus2 design

## Description
Compare ordinary typed constructors, focused syntax, and finite-model authoring. Record concrete author-facing examples, checked lowering, failure semantics, trust constraints, module ownership, and prototype experiments in model/Temporal/Feature/Nexus2/DESIGN.md.

## Acceptance
- [ ] TBD

## Done summary
Compared three independent interface designs and recorded the proposed Nexus2 design in model/Temporal/Feature/Nexus2/DESIGN.md. The design covers typed finite tables, separate existing declaration languages, alternative admission approaches, explicit race scope, diagnostic and trust constraints, and prototype evaluation. Candidate syntax is illustrative and pending user review.
## Evidence
- Commits:
- Tests: git diff --check, Python local-link and code-fence validation of model/Temporal/Feature/Nexus2/DESIGN.md
- PRs: