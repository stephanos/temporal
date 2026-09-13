---
satisfies: [R5]
---
# fn-86-retire-hand-written-models-one.7 Umpire.Examples.Switch re-authored with the commands inside Umpire

## Description
Re-author Umpire's Temporal-free worked example with the commands (R5) so it no longer builds `Umpire.Search` records by hand: the same two-state Model, Property, exploratory and exact-trace Queries and compiled artifact, declared through `enum`, `entity`, `action`, `machine`, `property`, `scenario`, `limits` and `query` under a Temporal-free definition root. Every Umpire test and tool importing it builds against the same exported names; its two goldens, regression view and experiment fixture regenerate byte-identical or with the diff listed and explained.

**Size:** M
**Files:** `model/Umpire/Examples/Switch.lean` (rewritten), `model/Umpire/Examples/SwitchTests.lean`, `model/Umpire/Command/Registry.lean` or `Conventions` (a Temporal-free `model_conventions` root for `Umpire.Examples`; today the only `model_conventions` lives in `Temporal.Case.Conventions`), the 11 importers if a name must move (`Umpire/Artifact/Tests/{Codecs,RunRecord}.lean`, `Umpire/Evidence/Tests/Fixtures.lean`, `Umpire/ImplementationLink/Tests/Application.lean`, `Umpire/PromotionTests.lean`, `Umpire/Search/Tests/Admission.lean`, `Umpire/Tests/MigrationCompatibility.lean`, `Umpire/Variations/Tests/Fixtures.lean`, `Temporal/Tool/Goldens.lean`), `model/Umpire/Examples/Fixtures/Switch*.json`, `model/Umpire/Examples/Generated/Switch.md`, `model/Umpire/Examples/testdata/switch-experiment-spec.json`, `model/Umpire/Artifact/Tests/Fixtures/SwitchPlanV2.json`
**Touches:** [model/Umpire/Examples/**, model/Umpire/Command/**, model/Umpire/Artifact/Tests/**, model/Umpire/Evidence/Tests/Fixtures.lean, model/Umpire/ImplementationLink/Tests/Application.lean, model/Umpire/PromotionTests.lean, model/Umpire/Search/Tests/Admission.lean, model/Umpire/Tests/MigrationCompatibility.lean, model/Umpire/Variations/Tests/Fixtures.lean, model/Temporal/Tool/Goldens.lean]

### Approach
- Export contract: the importers read about 25 raw symbols (`target`, `LawStatement`, `compiledArtifact`, `exactActionQuery`, `authoredProperty`, `modelSpec`, `switchCapabilityId`, ...); keep every exported name and type stable (define them from the command-elaborated declarations); a rename is a failure of this task, not a follow-up.
- Definition root: the commands derive Definition IDs through `model_conventions root "temporal" under Temporal.Feature`; add an `Umpire.Examples` convention with root `umpire` so the Switch IDs (`switch.*`) stay byte-identical; if an ID must move, the goldens' diff is listed with the reason.
- MOD-01: `Umpire.Examples` imports only `Umpire.Command` (the R7 rule applies to it).

### Investigation targets
**Required:**
- `model/Umpire/Examples/Switch.lean:11-30` (Definition IDs) and the exported declarations the importers read
- `model/Umpire/Tests/MigrationCompatibility.lean` (487 lines) — the compatibility family pins over Switch
- `model/Temporal/Case/Conventions.lean` and `model/Umpire/Command/Registry.lean:143-147` — `model_conventions`
- `model/Temporal/Tool/Goldens.lean:62-74` — the Switch goldens and `SwitchPlanV2.json`

**Optional:**
- `tools/umpire/cmd/umpire-gen-regression-views/catalog.go:26` — the regression view

### Key context
- Memory: behavior-neutral refactors must not strengthen validation; the example's admitted and rejected inputs stay the same (the `SwitchTests` `#guard_msgs` are the pin).

## Acceptance
- [ ] `Umpire.Examples.Switch` declares its Model through the commands and imports Umpire's authoring owners only through `Umpire.Command`; every importer builds unchanged
- [ ] `SwitchExactActionQuery.json`, `SwitchCompiledArtifact.json`, `SwitchPlanV2.json`, `Generated/Switch.md` and `switch-experiment-spec.json` regenerate byte-identical, or each diff is listed with its reason
- [ ] `SwitchTests` specimens pass unchanged; `lake build UmpireTests` green; `make umpire-check-goldens umpire-check-regression-views` exit 0; the inventory row is marked re-authored


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
