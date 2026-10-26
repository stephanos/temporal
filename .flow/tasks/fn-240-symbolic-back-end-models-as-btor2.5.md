---
satisfies: [R6]
---
# fn-240-symbolic-back-end-models-as-btor2.5 Symbolic Worlds: World fields free within declared finite domains, computed starts, decoded World replay, parameterized_world golden

## Description
Fixture opt-in to symbolic mode (an expected/symbolic.txt header line or the fixture manifest, matching how TLC fixtures select options). World fields become free states with next = self constrained to their finite domains; each property is checked once across the whole domain. Computed starts: choose between the front end admitting #[starts] bodies into the tree and a native start table over the World domain; record the choice and why in the spec's Decision Context. Values is export-only. Provide a checked internal World-construction adapter, implemented by generated code in the defining module or a typed fixture callback, taking decoded typed fields to Result<W, Refusal>. Validate exact fields, types and domains and require export-after-construction equality, supporting private World fields without a new authoring surface. Use this adapter for both decoded witnesses and native enumeration of the declared finite domain, preflighting product overflow and the catalog limit before allocating. A violation decodes the World values, constructs the World through this adapter, and replays under it; the golden records the domain and, for violations, the decoded World. Keep the engine verdict Safe or Bounded { depth } separate from Agreement::{Agrees, Disagrees { reason }, Unconfirmed { reason }}. For Safe or Bounded absence, Agrees requires exhaustive native confirmation for every World at the verdict's scope, with no exploration cut; an over-limit domain or cut yields Unconfirmed with a reason. Unsafe agrees through a confirming native replay under its decoded World. Bounded agreement stays bounded. Required fixture goldens cannot accept an unconfirmed symbolic-only result as passing. parameterized_world is the first fixture. Errors: a World field without a finite domain refuses symbolic mode naming the field; an unconstructible decoded World is witness-rejected.

## Acceptance
- parameterized_world has a symbolic golden covering its whole World domain in one run per property.
- A seeded violation on one World decodes that World and replays natively under it.
- A fixture with an unbounded World field refuses symbolic mode naming the field.
- Private World fields round-trip through checked native construction; missing, extra, wrong-type and out-of-domain assignments refuse, and product overflow/over-limit enumeration refuses before allocation.
- Tests distinguish Safe/Bounded engine results from Agrees/Disagrees/Unconfirmed, including cut and over-limit native checks.
- Before implementing starts, record the computed-starts choice and construction-adapter shape in the spec's Decision Context. A native start table must cover the entire selected World domain under its admitted limit; otherwise refuse symbolic mode rather than silently omit Worlds or starts.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
