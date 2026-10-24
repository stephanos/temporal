---
satisfies: [R4, R5, R8]
---
# fn-240-symbolic-back-end-models-as-btor2.4 Fixture consumer: expected/symbolic.txt and symbolic/paths, UMPIRE_BACKENDS=require gating, ownership registration, Kettle goldens, DESIGN and MILESTONES

## Description
Add a symbolic consumer to umpire-testing beside the TLC one: selected by expected/symbolic.txt, run only under UMPIRE_BACKENDS=require, ownership of symbolic.txt and symbolic/paths/*.txt registered even when skipped, default mode preserves committed payloads without launching a tool. Golden lines per property in declaration order: name, kind, engine, verdict (safe, unsafe, bounded N, timeout, not run), agrees or the disagreement; violations write symbolic/paths/<property>.txt. Strategy: ic3 first; on timeout, bmc to native depth plus one. Produce goldens for sample_kettle and kettle_cut; wire make model-check (require mode) and make model-update; the xtask rIC3 admission command runs from the Makefile only in require mode. Write the DESIGN.md section (verdict kinds, evidence rule: a proof counts only with native agreement, Apalache deferred until set-valued state) and update MILESTONES and the status line of model/design/native-symbolic-backend.md.

## Acceptance
- sample_kettle and kettle_cut have expected/symbolic.txt with every property agreeing; never_beeps has a paths file.
- Default make model-check reports the skipped consumer explicitly and touches no tool; require mode runs it and a disagreement fails the fixture.
- Ownership tests reject duplicate or stale symbolic output in both modes.
- DESIGN.md, MILESTONES.md and the design write-up status line are updated; timings absent from goldens.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
