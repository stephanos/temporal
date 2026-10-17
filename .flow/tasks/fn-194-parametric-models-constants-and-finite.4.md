---
satisfies: [R7]
---
# fn-194-parametric-models-constants-and-finite.4 DSL completion: document parameters in SEMANTICS and the README

## Description
**DSL completion phase (owner, 2026-10-17).** Describe the two kinds of parameter (a domain the Model is generic over as a `Finite`-bounded type parameter, a constant as a value parameter of a liftable kind), instantiation in the suite, naming, and the unchanged object form, with Voting as the worked example. Add the refusal texts to the API contracts list. State that lifting an instance is the IR phase.

**Size:** S

**Files:** `model/framework/SEMANTICS.md` (the Machines section and a new Parameters subsection), `model/framework/README.md`.

### Verification
- Docs match the DSL as landed in .2 and .3; the SEMANTICS source links point at Voting.

## Acceptance
- [ ] SEMANTICS describes parameters, kinds, instantiation and naming with Voting as the example
- [ ] README test guidance updated

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
