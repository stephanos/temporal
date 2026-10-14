---
satisfies: [R9, R10]
---
# fn-93-simplify-the-lean-model.14 Delete the observed-trace Implementation Link path and draft the SEM-08 restatement (B2)

## Description
Lane B2, part two. It runs after the artifacts are deleted (.13), because `Artifact/Result` imports `ImplementationLink.Application`; if D3 was declined, keep the parts of `Application` that `Result` uses and list them (§Decision interactions). It runs before the Evidence chain deletion (.15): `ImplementationLink/Application` and the Nexus evaluate path use `Evidence/Evaluate/Admission`'s `EvidenceBackedTrace` and `ObservationResult`. Whatever survives in `Application` must use none of the Raw, Structure, Admission or Reading.Check declarations; if it does, stop (R9). Delete the observed-trace sections of `ImplementationLink/Application` and the Nexus evaluate path (`evaluateFeatureProperty`, `applyImplementationLink`) with their tests; keep the SEM-08 link declaration, `Language`, `Refinement` and whatever Cancellation needs from `Temporal/System/Nexus/Evidence`. Draft the SEM-08 restatement. D2 was recorded in task 11.

**Size:** M
**Files:** `model/Umpire/ImplementationLink/Application.lean` (1,049; observed-trace sections), `model/Umpire/ImplementationLink/Tests/Application.lean` (756), `model/Temporal/System/Nexus/ImplementationLink.lean` (evaluate path ~466-486), `model/Temporal/System/Nexus/Evidence.lean` (137; only if nothing but the evaluate path uses it — Cancellation ~489-578 does), `model/TemporalModelTests/Nexus/ImplementationLink.lean` (619), `model/TemporalModelTests.lean`, `model/Temporal/Tool/Inventory.lean` + `model/INVENTORY.md` (Implementation Link application family), `.plans/UMPIRE4_SPEC.md` (SEM-08 restatement; Implementation Link glossary ~48-50), vocabulary gate, E2 entries
**Touches:** [model/Umpire/ImplementationLink/**, model/Temporal/System/Nexus/**, model/TemporalModelTests/**, model/TemporalModelTests.lean, model/Temporal/Tool/Inventory.lean, model/INVENTORY.md, .plans/UMPIRE4_SPEC.md, tools/umpire/internal/retiredvocabulary/**, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md]

### Approach
- Map `Application.lean` sections: what `Refinement`/`Language` and production need stays; observed-trace application goes. If `constructorClassifiers_exactlyOne` for its status survives, keep it; its two name functions then go to the reuse lane.
- `mapOutcome` lemmas (`Nexus/ImplementationLink.lean:154,182-186`) stay for A8.
- SEM-08: append `*Restatement (drafted by fn-93; awaiting GOV-02 approval.)*` stating the link is checked by refinement, not by replaying an observed trace; cite only surviving names. Do not edit the rule sentence itself.
- INVENTORY.md diff limited to the deleted family's rows.

### Investigation targets
**Required:**
- `model/Umpire/ImplementationLink/Application.lean` section headers
- `model/Temporal/System/Nexus/ImplementationLink.lean:440-580`
- `.plans/UMPIRE4_SPEC.md:48-60,150-160` — glossary and SEM-08

### Quick commands
```sh
cd model && lake build
make umpire-check-inventory umpire-check-goldens umpire-check-retired-vocabulary umpire-check-plan-index
go test ./tools/umpire/vocabulary/...
```
## Acceptance
- [ ] Observed-trace path and its tests gone; link declaration, Language, Refinement and Cancellation intact
- [ ] SEM-08 restatement drafted, awaiting approval; `spec_names_test` green
- [ ] INVENTORY.md changed only by the deleted family; everything else byte-identical
## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
