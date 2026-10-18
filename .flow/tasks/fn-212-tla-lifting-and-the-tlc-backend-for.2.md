---
satisfies: [R2, R3]
---
# fn-212-tla-lifting-and-the-tlc-backend-for.2 Lift, emit TLA+, run TLC; agreement goldens on Kettle and the shared fixtures

## Description
On yes: `umpire-lift` (internal tree from the chosen source, liftable-subset warnings), TLA+ emitter (author's Rust names, stated integer bounds, liveness with weak fairness on internal steps), `umpire/backends/tlc/` runner on a pinned `tla2tools.jar` fetched to `model/build/`, dump decoded through `Spell`, and a TLC consumer in the shared runner owning `<Model>.tla`, `<Model>.cfg`, `tlc.txt`. Keep `make model-time` within fn-210 R30 (batch TLC or gate it behind `UMPIRE_BACKENDS=require`).

**Files:** model/umpire/lift/, model/umpire/backends/tlc/, model/umpire/testing/src/ (consumer), model/umpire/tests/fixtures/*/expected/, Makefile.

**Quick commands:** `make model-check`; `UMPIRE_BACKENDS=require make model-check`; `make model-time`

## Acceptance
- Kettle: TLC reaches 283 states, verdicts agree; `Kettle.tla`, `Kettle.cfg`, `tlc.txt` goldens.
- An out-of-subset expression is warned on and refuses emission naming it (golden).
- Gate within R30.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
