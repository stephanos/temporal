---
satisfies: [R8]
---
# fn-124-shrink-and-simplify-the-umpire-go.8 Split tools/umpire/model into ir, interp, check and realization packages

## Description
Implements R8. Cross-spec entry gate: fn-118 and fn-120 landed (both edit model and lower). Pure moves only: tools/umpire/{ir,interp,check,realization} with the checker engine at check/internal/engine; update the 59 importers (no alias needed where the new name stands alone); encode the new boundaries in the ownership test; behaviour, Case bytes and IR unchanged.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
