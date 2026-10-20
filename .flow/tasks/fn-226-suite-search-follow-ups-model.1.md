---
satisfies: [R1, R2, R3]
---
# fn-226-suite-search-follow-ups-model.1 Suite search: predicate panics, Model-state budget, panic relevance

## Description
Apply the three follow-ups and the cosmetic fixes from the re-review of df992fd8e8 as listed in the spec, in model/umpire/suite (search.rs, space.rs, run.rs) with fixtures; README and Space::budget doc aligned. Verify make model-check and make model-time.

## Acceptance
- [ ] Predicate panics refused as such, naming the waypoint
- [ ] find/along budget counts Model states; large bounded pattern fixture
- [ ] Irrelevant deeper panics do not refuse an answer
- [ ] make model-check passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
