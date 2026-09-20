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
- [x] `Umpire.Examples.Switch` declares its Model through the commands and imports Umpire's authoring owners only through `Umpire.Command`; every importer builds unchanged
- [x] `SwitchExactActionQuery.json`, `SwitchCompiledArtifact.json`, `SwitchPlanV2.json`, `Generated/Switch.md` and `switch-experiment-spec.json` regenerate byte-identical, or each diff is listed with its reason
- [x] `SwitchTests` specimens pass unchanged; `lake build UmpireTests` green; `make umpire-check-goldens umpire-check-regression-views` exit 0; the inventory row is marked re-authored


## Done summary

Done 2026-09-20; self-review. Commit 1136129.

### The example

`model/Umpire/Examples/Switch.lean` declares the switch through the commands and nothing else:
`entity subject`, `enum Position | off | on`, the one-field state `SwitchState { power }`, `enum
FlipOutcome | applied | deferred`, `enum Power | off | on` (the position a flip shows), `action flip`
(party `operator`, on the subject), one step function `flipStep` (the applied result first, so the
shortest witness is the flip that took), `machine twoState` (starts off, ends on), `property
flipTurnsOn` (`when: flip`, `holds: fun step => step.state.power == .on`), `scenario oneFlip` and
`scenario explore` (one flip from off), `limits one` (1/1/8) and `query exactAction` (`find:
flipTurnsOn in: oneFlip`). The file imports `Umpire.Command` and the new
`Umpire.Examples.Conventions`, which declares `model_conventions root "umpire" under
Umpire.Examples`, so the family is `switch` under the `umpire` root. Every exported name the eleven
importers read is kept and defined from the command's declarations: the ids from `twoState`
(`targetId`, `kernelId`, `switchCapabilityId`, `switchProviderId`, `flipLawId`, `switchRoleId`,
`relationIds`), the values from the admitted Query's vocabulary (`offState`, `onState`,
`flipAction`, `appliedOutcome`, `deferredOutcome`, `powerOffObservation`, `powerOnObservation`),
`LawStatement`/`flipLaw`/`flipLawProof` from the table law, `target` as the admitted Query's Model,
`machine`/`initialStates`/`stepResults`/`authoritativeInitial`/`authoritativeStep` as the target's
kernel, `modelSpec`/`modelProviders`/`switchProvider`/`finitePlanning`/`targetAuthoring` as the
records `DraftModel.make` recomposes the same kernel from, `authoredProperty`/`flipProperty`, the
three Scenario declarations and checked Scenarios, `limits`, `shortestPolicy`, `queryContext`, the
three Queries (`exactActionQuery` is the command's own Query re-addressed to `target`; the
exploratory `pick` Query and the exact-trace Query are checked here over the command's Model because
the `query` command declares neither form), `exactActionAdmitted`, the three runs, `artifact` and
`compiledArtifact`. Domain lemmas `target_setupDomain`, `target_stateDomain`, `target_actionDomain`,
`target_outcomeDomain`, `target_observationDomain`, `target_initial` and `target_step` are proved
from the kernel's complete vocabulary and the pinned rows (`target_initialStates`, `target_steps`),
so a proof over the target's domains reads the switch's members without evaluating the admission.
Two names are new (`flipOccurrenceId`, `offFlipRelationId`/`onFlipRelationId`); the hand-built
kernel's own obligations (`initialStates_sound`/`_complete`, `stepResults_sound`/`_complete`,
`stepResults_length_le_two`) went with the kernel they were about. `checked` is `@[irreducible]`,
which keeps the elaborator from unfolding the admission behind `Option.get` (the .4 lesson).

### The ids moved, and what moved with them

Reported before any golden was touched (task text): the command path derives
`umpire.switch.<kind>.twoState.<member>`, so nothing is byte-identical. `switch.target.two-state`
is `umpire.switch.target.twoState`; the kernel `umpire.switch.kernel.twoState.planner`; the
capability `…capability.twoState.transitions`; the provider `…provider.twoState.finite-table`; the
law `…law.twoState.canonical-table`; the role `…role.twoState.subject`; the action
`…action.twoState.flip`; the outcomes `…outcome.twoState.applied`/`deferred`; the Property
`umpire.switch.property.flipTurnsOn` with its one clause `…flipTurnsOn.state-on`; the Scenarios
`umpire.switch.behavior.oneFlip`/`explore`/`exactTrace` with occurrence `…occurrence.oneFlip.1` and
setup `…setup.oneFlip.subject`; the Queries `umpire.switch.query.exactAction`/`explore`/`exactTrace`.
Two things changed shape, not only spelling: each state and each fact is its own definition
(`…state.twoState.off`/`on`, `…fact.twoState.off`/`on`) where the hand-written example had one
`switch.state.power` and one `switch.observation.power` spanning both values, and the `power`
field has its own definition `…state-field.twoState.power`, which is what `powerStateId` now names
(a `.state`/`.priorState`/`.resultingState` pattern reads it through the state's fields);
`powerObservationId` is the off fact's definition, the one the evidence tests record. The
definitions list gained the field and two relation definitions (`…relation.twoState.off-flip`,
`on-flip`), which no provider means; the target fingerprint is
`sha256:4bd4815c…`, the artifact checksum `sha256:91c59681…` (plan `sha256:32183cb9…`).

Regenerated accordingly, none byte-identical: `Umpire/Examples/Fixtures/SwitchExactActionQuery.json`
and `SwitchCompiledArtifact.json`, `Umpire/Artifact/Tests/Fixtures/SwitchPlanV2.json` (same bytes as
the artifact), `RuntimeConfigurationV2.json`, `ExperimentRunV2.json`, `RawEvidenceV2.json`, `EvidenceV2.json`,
`ResultV2.json` and `ArtifactSetV2.json` (they bind the artifact's checksum, fingerprint,
capability and the flip's occurrence and action ids, and each other's checksums), `Umpire/Examples/testdata/switch-experiment-spec.json`,
`Umpire/Examples/Generated/Switch.md`, `tools/umpire/regression/switch_generated_view_test.go`, and
`Umpire/Promotion/Tests/Fixtures/CompiledSource.lean` (the rendered source carries the trace's
ids; its digest is `sha256:e71b0289…`). The regression catalog, the Makefile fixture list, the Go
regression and artifact tests and the inspector registry name `umpire.switch.query.exactAction`.

### The importers

Every importer builds; the commands' leading words (`property`, `scenario`, `limits`, `query`)
are now non-reserved like `entity`, `action` and `observation` already were
(`Umpire/Command/Syntax.lean`, `declarationKeyword`), so a module importing the commands keeps
binding `query`, `property` and the `limits` field. `Registry.conventionsFor` reads the declared
conventions whose namespace prefix covers the declaring namespace (longest wins), so Temporal's and
Umpire's declarations can share an import closure. Importer edits, each because a value's shape
moved rather than its name: `ImplementationLink/Tests/Application.lean` reads the domain lemmas
instead of `change`-ing the hand-built kernel's relations, carries the two relation definitions as
relation Known Gaps (no provider means them, so no semantic reference reaches them), reads the
state through one conditioned rule per position, and pins that two invalid setups both diagnose
canonically (a table kernel encodes an out-of-catalog setup as the empty key, so their identities
no longer differ: `CONSIDER(umpire)` left there); `Variations/Tests/{Fixtures,Validation,
Metadata,Intent}.lean` and `Artifact/Tests/RunRecord.lean` name `flipOccurrenceId`,
`flipActionId` and `switchCapabilityId` where they spelled the raw ids;
`Artifact/Tests/Codecs.lean` moved its guarded-Property ids under `umpire.switch` so canonical
order is unchanged; `Artifact/Tests/Evidence.lean` spells the receipt's action and occurrence ids
from the example; `Evidence/Tests/Check.lean` remaps the satisfied plan's observation rule to the
`on` fact the way it already remapped its outcome rule, expects the verdicts in the new canonical
order (`test.…initial-off` before `umpire.switch.…`), and pins that a clause naming the `on` state
examines no state coordinate of a step that leaves the switch off; `Tests/MigrationCompatibility.lean` keeps the `switch` family and the expert
route (`DraftModel.make` over `modelSpec`, `modelProviders`, `machine`, `finitePlanning`; relocated
occurrences; the wrong-kind, invalid-id, missing-provider, missing-law and incomplete-kernel
targets; the Query error kinds) and admits the relocated targets through `Search.admit` instead of
proving `SearchView.ofCheckedQuery?` by unfolding the hand-built kernel. `SwitchTests.lean` keeps
every specimen (the default-proof pins, the two `#guard_msgs` rejections, the Property and Scenario
error matrices, the goldens, the planner outcomes) and re-pins the ids, definitions, provider,
Scenario and Property shapes, the fingerprint, and that the command's Query, run and admission are
what the views expose. `HANDWRITTEN_INVENTORY.md` marks the row re-authored.

### Gates

`lake build` green; `make umpire-gen-goldens`/`umpire-check-goldens`,
`umpire-gen-regression-views`/`umpire-check-regression-views`, `umpire-gen-inventory`/
`umpire-check-inventory`, `umpire-check-retired-vocabulary`, `umpire-check-testpilot-protocol`,
`umpire-check-testpilot-authoring` exit 0; `LEAN_NUM_THREADS=1 make lint-model` at the .1
baseline; `make lint-code-fast` clean; `go test ./tools/umpire/... ./common/testing/testpilot/...
./tests/testcore/testpilot/...` green; `make umpire-check-regression` green.

## Evidence
- Commits: 1136129
- Tests: `cd model && lake build`; `make umpire-gen-goldens umpire-check-goldens umpire-gen-regression-views umpire-check-regression-views umpire-gen-inventory umpire-check-inventory umpire-check-retired-vocabulary umpire-check-testpilot-protocol umpire-check-testpilot-authoring`; `LEAN_NUM_THREADS=1 make lint-model`; `make lint-code-fast`; `go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...`; `make umpire-check-regression`
- PRs:
