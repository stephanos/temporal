---
satisfies: [R1, R2]
---
# fn-224-accepted-self-loops-in-the-attribute.1 Self-loop marker in #[machine] handlers, lifter admission, and fixtures moved back to TLC

## Description
Add the accepted self-loop marker beside reject! in umpire-dsl and #[machine] (name chosen consistently with reject! and recorded in DESIGN.md; provisional stay!()). Unchanged handler arm with the marker = enabled self-loop, unit outcome or Accepted in modules using reject!; reject! takes precedence; misuse outside #[on(..)] handlers refused, pinned by trybuild. Admit it in umpire-lift through the existing authored-self-loop emission. Then rewrite the checker-only fixtures whose only gap was the self-loop (progress_self_loop and siblings, edge_checks/vacuity where their remaining reasons allow) back into shared attribute fixtures with byte-identical behaviour goldens up to fn-223's listed spellings, and update fn-223's disposition table. Verify make model-check, UMPIRE_BACKENDS=require, make model-time. Starts after fn-213 is closed in umpire's history.

## Acceptance
- [ ] Marker makes an unchanged arm an enabled self-loop; misuse refused by trybuild
- [ ] All pre-existing goldens byte-identical
- [ ] Self-loop fixtures back in the shared corpus agree with TLC; disposition table updated
- [ ] make model-check and UMPIRE_BACKENDS=require pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
