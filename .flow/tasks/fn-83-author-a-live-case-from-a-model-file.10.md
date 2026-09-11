---
satisfies: [R10]
---
# fn-83-author-a-live-case-from-a-model-file.10 Move the Model commands and their authoring core into Umpire

## Description
Move the five Model commands (`model`, `property`, `scenario`, `limits`, `query`) and the authoring core behind them out of `Temporal.Feature.Nexus.Success` into `Umpire` (R10). The logic is already feature-neutral (`RaceSyntaxTests.lean` authors an unrelated lifecycle through it), but it lives under a Nexus namespace, so the worker-outage Model (.5) and the sync-Nexus Model (.6) would import `Temporal.Feature.Nexus.Success.Syntax`. The `case` command and the realization templates stay Temporal-owned. Draft an AUT-07 amendment under GOV-02 that names the commands as the Umpire command surface over `Umpire.Property`, `Umpire.Scenario` and `Umpire.Query`.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Success/Syntax.lean` and `Authoring.lean` (split: the Model-command half and its core move to a new `Umpire` module tree, final name a task decision, for example `Umpire.Command`; the `case` elaborator moves to a Temporal module such as `Temporal.Case.Syntax`), `model/Temporal/Case/Registry.lean` (Scenario and Query entries move to Umpire; the Case entry stays), `model/Umpire/Case/Producer.lean` (`Identity.ofFixture` stops hard-coding `temporal.case.`), `Model.lean`, `RaceSyntaxTests.lean`, `Tests.lean` and every other importer, `model/Umpire.lean` / `model/UmpireTests.lean` roots, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_SPEC.md` (drafted AUT-07 amendment only)
**Touches:** [model/Umpire/**, model/Temporal/Feature/Nexus/Success/**, model/Temporal/Case/**, model/Temporal/Tool/Testpilot.lean, model/Umpire.lean, model/UmpireTests.lean, model/ARCHITECTURE.md, .plans/UMPIRE4_SPEC.md]

### Approach
- Read `.plans/LEAN_GUIDELINES.md` and `.plans/UMPIRE4_SPEC.md` (MOD-01, SCP-02, AUT-07, AUT-09) first.
- Temporal couplings to remove from the moved code, each with its replacement:
  - `Temporal.Shared.definitionFamily` prepends `temporal.`, and `semanticFamilyOf` strips `Temporal.Feature` / `Temporal` from the namespace. Replace with an injected definition root and namespace prefix that a Temporal module declares once (for example a small environment extension or option set by a Temporal prelude the Model files import). Build on `Umpire.Shared`, never `Temporal.Shared`.
  - `packageRelativePath` splits on `/model/`. Make the package-root convention explicit rather than a string split on a repository directory name.
  - Diagnostics say "Nexus model …". Reword to feature-neutral text and update every `#guard_msgs` pin.
  - `Identity.ofFixture` in `Umpire.Case.Producer` hard-codes `"temporal.case." ++ fixture`, an existing SCP-02 leak. The Case ID prefix comes from the `case` command.
- Delete the Temporal-to-Umpire conversion `producerInput` once the checked bundle is Umpire-owned, if the bundle and `Umpire.Case.Producer.Input` can then be the same record; otherwise record why not.
- Out of the production module: the test-only mutation helpers (`withStates`, `withTransitions`, `transitionRow`, `withOccurrences`, `withClauses`, `reorderedAndDocumented`) move to the test tree; `Origin.occurrence`, `outcomeIdAt`, `factIdAt`, `relationIdAt` have no caller and are deleted.
- Make the hidden constraints visible in the moved grammar or its diagnostics: the `model` command resolves a type literally named `Setup` in scope with exactly one constructor, admits one role, and requires Action constructors in sorted order. Keep the behavior; name the requirement in the diagnostic.
- The hard-coded Known Gaps (`completionKnownGaps`) move unchanged in this task; .11 replaces them. Keep the diff a move plus the listed couplings.
- Retired-keyword arms keep their located errors unless `make umpire-check-retired-vocabulary` requires otherwise.
- The byte pin: every checked-in fixture, `CheckedQuery.id`, fingerprint and golden is byte-identical before and after (run `make umpire-check-case-runtime-conformance` and `make umpire-check-goldens` unchanged).

### Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus/Success/Syntax.lean` — `originTerm`, `semanticFamilyOf`, `packageRelativePath`, `domainConstructors`, the five command elaborators, the `case` elaborator and `templateTerm`
- `model/Temporal/Feature/Nexus/Success/Authoring.lean` — `Origin`, `successModel`, `authoredProperty`, `authoredScenario`, `check`, `completionKnownGaps`, `producerInput`
- `model/Temporal/Case/Registry.lean` — the three environment extensions
- `model/Umpire/Case/Producer.lean` — `Identity`, `Input`, `ofFixture`
- `model/Temporal/Shared.lean` and `model/Umpire/Shared.lean` — family, source and metadata helpers
- `Makefile` `lint-model` and `model/ModelLint` — how MOD-01 and SCP-02 are enforced

### Key context
- fn-84.3 also edits `Success/Authoring.lean`'s `check` chain (`Umpire.Search.admit`). Whichever lands second rebases onto the moved path; record the new path in the receipt so fn-84.3's file list can be updated.
- .5 changes the `scenario` grammar and .6 and .8 write new importers and the tutorial; they depend on this task so they target the Umpire path from the start.
- A parallel session may own neighbouring fn-83 tasks; check `git log` before reporting.

## Acceptance
- [ ] The five Model commands, their authoring core, and the Scenario and Query registry entries live under `Umpire`; `make lint-model` passes with no MOD-01 or SCP-02 finding and no `temporal` literal in the moved modules or `Umpire.Case.Producer`
- [ ] The `case` command and templates are Temporal-owned; no Model file imports a `Temporal.Feature.Nexus.Success` authoring module
- [ ] Every checked-in fixture, golden, `CheckedQuery.id` and fingerprint is byte-identical; the Nexus success family is still `temporal.nexus.success`
- [ ] Test-only helpers live in the test tree; the uncalled accessors are deleted; the `Setup`, single-role and sorted-Action requirements are named in their diagnostics, pinned by `#guard_msgs`
- [ ] An AUT-07 amendment is drafted in `UMPIRE4_SPEC.md` under GOV-02 and the architecture documents name the new owner
- [ ] `cd model && lake build`, `make umpire-check-regression` pass


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
