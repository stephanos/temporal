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
Deleted the manual-trait lowering: `lift/src/manual.rs`, `lift/src/manual/` and the dispatch in `Lift::lift`. Source without a `#[machine]` module is refused by location: at its `impl Machine` item when it has one ("the lifter reads #[machine] Models; `impl Machine for X` implements Machine by hand"), otherwise at 1:1 ("... this source declares none"). The shared fixture `progress_dead_end` (a hand-written `Machine` wrapper that is selected for lifting) pins the hand-written message in its lift golden, which names the file and the impl item. `kettle_cut`/`kettle_progress` lift goldens changed from the old manual message to "declares none".

Removed tests: `lift/tests/manual.rs`, `manual_admission.rs`, `manual_overflow.rs`, their fixtures and `expected/manual_admission.txt`, and emission.rs's escaped-fact TLC control (facts exist only in trait form). The control-character test in source.rs now uses attribute Kettle with a `#[doc]` literal in place of the checker `choices` source.

README: one admitted source form, hand-written `Machine` refused at its impl item. DESIGN: lifter source 5,007 -> 3,018 lines (5,048 just before this task); shared-fixture Models row 25/980/862 -> 12/534/458.

Gates: `make model-check` pass; `UMPIRE_BACKENDS=require make model-check` pass.

model-time:
```
tests                             2.172s
go-host                           1.116s
temporal-go-host                  5.754s
inner loop (edit+test)            2.879s
gate total                       10.231s  (soft target 30s)
inner loop                        2.879s  (soft target 10s)
```

Deviation: the refusal golden is the existing shared fixture `progress_dead_end`'s lift golden rather than a new fixture, because it is already hand-written source selected for lifting.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2f1b3b415e
- Tests: make model-check, UMPIRE_BACKENDS=require make model-check, make model-time
- PRs: