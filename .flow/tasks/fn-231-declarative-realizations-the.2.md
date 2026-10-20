---
satisfies: [R2, R4]
---
# fn-231-declarative-realizations-the.2 After .1: coverage golden realization.snap.txt against the fn-217 catalog; unknown paths refused; DESIGN.md section

## Description
Check reads and ignores against the fn-217 catalog at build time (unknown path is a compile error, empty ignore reason refused) and emit expected/realization.snap.txt for the activity Model: per handled RPC every catalog field as mapped, ignored with reason, or unsupported, followed by the Model-action RPCs with no handler. Negative control: a mutated catalog with one added field fails the golden. Write the DESIGN.md section Declarative Realizations with the surface, return-type rules, golden format and before/after line counts.

## Acceptance
- [ ] realization.snap.txt present with the three classifications and unhandled RPCs
- [ ] Unknown field path and empty reason are compile errors; added-field control fails the golden
- [ ] DESIGN.md section written

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
