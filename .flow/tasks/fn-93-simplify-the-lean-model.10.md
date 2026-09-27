---
satisfies: [R9, R10]
---
# fn-93-simplify-the-lean-model.10 Retire Umpire.Variations (B1, decision D1)

## Description
Lane B1. Delete the Variations package and everything that only it feeds.

### Owner decision
- **D1 — delete `Umpire.Variations`. Recommended default: taken.** Also **D9 — rewrite the `UMPIRE4_SPEC.md` descriptions that cite deleted names in this commit. Recommended default: taken.**
- Record both as the first line of the Done summary before the deletion commit (`D1: taken by <who>, default followed`). If D1 is declined, close the task with the decision recorded and no change.

**Size:** M
**Files:** delete `model/Umpire/Variations.lean`, `model/Umpire/Variations/**` (production ~1,970, tests ~1,430); edit `model/Umpire.lean:12` (facade), `model/UmpireTests.lean:28-33`, `model/Umpire/Examples/SwitchTests.lean:2-6` (drop Variations uses, incl. `checkVariationSpace_baseQuery`), `model/Umpire/ImportTests.lean:26-28`, `model/ModelLint/ModuleIndex.lean:58-88` (hard-coded list), `model/HANDWRITTEN_INVENTORY.md` rows, `model/Umpire/Search/Admission.lean:18` (prose), `tools/umpire/internal/retiredvocabulary/check.go` (+ tests), `.plans/UMPIRE4_SPEC.md` (owner list ~113, Variations glossary ~489-493, Exploration glossary ~674), `model/README.md:78-80`, `model/ARCHITECTURE.md:31,96`, `model/Umpire/ARCHITECTURE.md:29,105`, `model/Umpire/Property/COMPATIBILITY.md` row, `common/testing/testpilot/README.md:253` mention, E2 checker entries naming deleted declarations
**Touches:** [model/Umpire/Variations/**, model/Umpire/Variations.lean, model/Umpire.lean, model/UmpireTests.lean, model/Umpire/Examples/**, model/Umpire/ImportTests.lean, model/ModelLint/ModuleIndex.lean, model/HANDWRITTEN_INVENTORY.md, model/Umpire/Search/Admission.lean, tools/umpire/internal/retiredvocabulary/**, tools/umpire/vocabulary/**, .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_*.md, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, model/Umpire/Property/COMPATIBILITY.md, common/testing/testpilot/README.md]

### Approach
1. Re-verify importers (`grep -rn 'import Umpire.Variations'`): only the facade, test aggregator, SwitchTests, ImportTests and ModuleIndex are expected; any other production importer blocks the task (R9).
2. Delete, fix every reader listed above in the same commit so every gate stays green.
3. Vocabulary gate: add `Umpire.Variations` in dotted and slash forms, `VariationSpace`, and the package's compound identifiers (not bare words), following the split-literal style of `check.go:330-746`. Resolve gate hits in `.plans/UMPIRE4_*.md` prose that uses the terms as current; leave historical mentions to the gate's exemption mechanism.
4. `UMPIRE4_SPEC.md`: rewrite the descriptive text (D9) so no backticked name is unresolved; append a `*Restatement (drafted by fn-93; awaiting GOV-02 approval.)*` wherever rule text itself changes meaning.
5. `model/Umpire/Examples/Generated/Switch.md` must stay byte-identical (`make umpire-check-regression-views`).

### Investigation targets
**Required:**
- `model/Umpire/Examples/SwitchTests.lean`
- `model/ModelLint/ModuleIndex.lean:58-88`
- `tools/umpire/internal/retiredvocabulary/check.go:72-83,330-400,745-760` — scan roots, token list, matching rules
- `tools/umpire/vocabulary/spec_names_test.go`

### Quick commands
```sh
cd model && lake build
make umpire-check-retired-vocabulary umpire-check-plan-index umpire-check-goldens umpire-check-regression-views
go test ./tools/umpire/vocabulary/... ./tools/umpire/internal/retiredvocabulary/...
LEAN_NUM_THREADS=1 make lint-model
```

## Acceptance
- [ ] Decision D1 (and D9) recorded before the deletion commit
- [ ] No module imports Variations; package, tests and uses gone; module index and ledger updated
- [ ] Retired names in the gate; `spec_names_test` green; restatement drafted where rule text changed
- [ ] Surviving goldens, regression views, Case fixtures byte-identical; every gate in Quick commands green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
