---
satisfies: [R5]
---
# fn-254-gomad-deeper-rust-host-testing-mutation.4 proptest codec and canonical-encoding properties

## Description
R5: proptest properties for codecs and canonical encodings.

Per surveyed codec, write a property (each test names its invariant):
- decode(encode(x)) == x
- canonical encoding is unique
- refused inputs stay refused after re-encoding attempts
- digests over a canonical form are stable

The default case count is small enough for the fn-253 inner loop (target: the whole property set adds ≤ 5 s to `check-fast`). `deep:proptest` sets `PROPTEST_CASES` high. Failure seeds are persisted under `proptest-regressions/` and committed.

## Acceptance
- The surveyed codecs have round-trip, canonical-uniqueness and refusal properties that name their invariants.
- `check-fast` overhead from proptest is recorded at ≤ 5 s; `deep:proptest` runs the large count.
- A seeded encoder bug is caught, and its seed is kept in `proptest-regressions/`.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
