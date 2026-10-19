---
satisfies: [R2]
---
# fn-223-one-lifted-source-form-the-lifter-reads.2 Delete the manual lowering from umpire-lift; one located refusal for hand-written Machine source

## Description
With no trait-form shared fixture left, delete model/umpire/lift/src/manual.rs and lift/src/manual/ and the dispatch to them. Source selected for lifting that implements Machine by hand is refused by location with one message naming the file and the impl Machine item, pinned by a refusal golden. Update model/README.md (one admitted source form) and model/DESIGN.md (the lifter's new line count against the old 5,007 source lines). Verify make model-check, the required TLC run (UMPIRE_BACKENDS=require) and make model-time.

## Acceptance
- [ ] lift/src/manual* removed and the crate builds
- [ ] Hand-written Machine source is refused by location with a golden-pinned message
- [ ] README and DESIGN describe one source form and the size change
- [ ] make model-check and UMPIRE_BACKENDS=require runs pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
