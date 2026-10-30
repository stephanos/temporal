---
satisfies: [R4, R5, R8]
---
# fn-240-symbolic-back-end-models-as-btor2.4 Fixture consumer: expected/symbolic.txt and symbolic/paths, UMPIRE_BACKENDS=require gating, ownership registration, Kettle goldens, DESIGN and MILESTONES

## Description
Add a symbolic consumer to umpire-testing beside the TLC one: selected by expected/symbolic.txt, run only under UMPIRE_BACKENDS=require, ownership of symbolic.txt and symbolic/paths/*.txt registered even when skipped, default mode preserves committed payloads without launching a tool. Golden lines per property in declaration order: name, kind, engine, verdict (safe, unsafe, bounded N, timeout, not run), and agreement (Agrees, Disagrees { reason }, Unconfirmed { reason }); violations write symbolic/paths/<property>.txt. Strategy: ic3 first; on timeout, bmc to native depth plus one. Produce goldens for sample_kettle and kettle_cut; wire the framework dev/xtask gate/update/validate routes; only require mode provisions rIC3 before fixtures through the framework admission command. Root make model-check/model-update remain delegating entry points. Standalone framework-copy checks must work without the system workspace or root Makefile. Write the DESIGN.md section (verdict kinds, evidence rule: a proof counts only with native agreement, Apalache deferred until set-valued state) and update MILESTONES and the status line of model/design/native-symbolic-backend.md.

## Acceptance
- sample_kettle and kettle_cut have expected/symbolic.txt with every property agreeing; never_beeps has a paths file.
- Default make model-check reports the skipped consumer explicitly and touches no tool; require mode runs it and a disagreement fails the fixture.
- Ownership tests reject duplicate or stale symbolic output in both modes.
- Framework xtask gate/update/validate own provisioning and output updates in an isolated standalone framework copy; root Make only delegates.
- Agrees requires native confirmation at the claimed scope. For Safe or Bounded absence, cut/over-limit exploration and symbolic-only results are Unconfirmed with a reason; Unsafe requires a confirming native replay; required fixtures cannot bless Unconfirmed, Timeout, NotRun or completion-budget exhaustion as agreement. Bounded results retain their bound and never become unbounded proof.
- DESIGN.md, MILESTONES.md and the design write-up status line are updated; timings absent from goldens.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
