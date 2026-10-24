---
satisfies: [R6]
---
# fn-240-symbolic-back-end-models-as-btor2.5 Symbolic Worlds: World fields free within declared finite domains, computed starts, decoded World replay, parameterized_world golden

## Description
Fixture opt-in to symbolic mode (an expected/symbolic.txt header line or the fixture manifest, matching how TLC fixtures select options). World fields become free states with next = self constrained to their finite domains; each property is checked once across the whole domain. Computed starts: choose between the front end admitting #[starts] bodies into the tree and a native start table over the World domain; record the choice and why in the spec's Decision Context. A violation decodes the World values, constructs the World natively (inverse of the Values export), and replays under it; the golden records the domain and, for violations, the decoded World. NoViolation is confirmed natively for every World when the domain product is within the catalog limit, otherwise reported as symbolic only. parameterized_world is the first fixture. Errors: a World field without a finite domain refuses symbolic mode naming the field; an unconstructible decoded World is witness-rejected.

## Acceptance
- parameterized_world has a symbolic golden covering its whole World domain in one run per property.
- A seeded violation on one World decodes that World and replays natively under it.
- A fixture with an unbounded World field refuses symbolic mode naming the field.
- The computed-starts choice is recorded in the spec's Decision Context.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
