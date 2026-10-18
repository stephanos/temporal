---
satisfies: [R4, R5]
---
# fn-211-apalache-backend-for-umpire-in-rust.4 Apalache consumer in the fixture runner; Kettle agreement golden

## Description
Add an Apalache consumer to the fixture runner, selected per fixture by the presence of `expected/apalache.txt` and run only when backends are requested. It writes per invariant: name, bound, verdict, and agreement with the native checker within the bound. Add `apalache.txt` to Kettle and to any other fixture TLC already covers where the bound is meaningful.

**Files:** the consumer registry in `umpire-testing` and the shared fixtures in `umpire/tests/fixtures/` (fn-210 R29), Kettle's `expected/apalache.txt`, Makefile.

**Quick commands:** `UMPIRE_BACKENDS=require make model2-check`; `make model2-check` (unchanged default)
## Acceptance
- Kettle's `apalache.txt` shows agreement for every invariant and postcondition within the bound.
- Default gate unchanged; a disagreement is listed and fails.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
