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
- Plan review round 1 (F3): check the Definition IDs first. `Fixtures/SwitchCompiledArtifact.json`
  carries `switch.state.power`, `switch.setup.subject-is-off` and `switch.role.subject` plus three
  behavior fingerprints, and the commands derive Definition IDs from `Origin`. If the command path
  cannot reproduce those ids the artifact identity moves, so
  `model/Umpire/Tests/MigrationCompatibility.lean` is part of this task, not a discovery inside it.
  Report the ids the command path produces before touching any golden.
- Export contract: the importers read about 25 raw symbols (`target`, `LawStatement`, `compiledArtifact`, `exactActionQuery`, `authoredProperty`, `modelSpec`, `switchCapabilityId`, ...); keep every exported name and type stable (define them from the command-elaborated declarations); a rename is a failure of this task, not a follow-up.
- Definition root: the commands derive Definition IDs through `model_conventions root "temporal" under Temporal.Feature`; add an `Umpire.Examples` convention with root `umpire` so the Switch IDs (`switch.*`) stay byte-identical; if an ID must move, the goldens' diff is listed with the reason.
- MOD-01: `Umpire.Examples` imports only `Umpire.Command` (the R7 rule applies to it).
- Adjusted 2026-09-19 after fn-85 .14, .16, .15, .4 and .7 landed. The command surface the
  re-authoring must meet: `machine` needs `for:` an `entity` (so the example declares one), a
  `state:` structure of finite fields, `starts:` and `ends:` (both required since fn-85 .16; a step
  out of an end state is admitted, so `ends:` lists the terminal state without forbidding steps
  from it), `steps:` as one Lean function per action, and `evidence:` names resolved through the
  platform's catalog hook -- which `Umpire.Examples` has none of, so the example writes no
  `evidence:`; `property` takes `machine:`, optional `when:` and `holds:` a predicate (fn-85 .15),
  and its fingerprint reads through the machine's state fields (fn-85 .4), so the three
  fingerprints in `SwitchCompiledArtifact.json` move even if the ids do not; Definition IDs come
  from `model_conventions` (`Umpire/Command/Registry.lean:349`; the only declaration is
  `Temporal/Case/Conventions.lean:11`) as `<root>.<family>.<kind>.<name>`, and today's
  `switch.state.power` carries no root segment, so a byte-identical golden is unlikely and the
  listed diff with its reason is the expected outcome of the second acceptance line. `Umpire.Command`
  also carries `set`, `register_switch` and `classClaims` (fn-85 .7); the example needs none of
  them.

### Investigation targets
**Required:**
- `model/Umpire/Examples/Switch.lean:11-30` (Definition IDs) and the exported declarations the importers read
- `model/Umpire/Tests/MigrationCompatibility.lean` (487 lines) — the compatibility family pins over Switch
- `model/Temporal/Case/Conventions.lean:11` and `model/Umpire/Command/Registry.lean:18,349` — `model_conventions`
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
