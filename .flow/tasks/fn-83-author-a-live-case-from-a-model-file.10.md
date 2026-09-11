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
The five Model commands and the authoring core behind them are now `Umpire.Command`.

- `model/Umpire/Command/Authoring.lean` (moved from `Temporal/Feature/Nexus/Success/Authoring.lean`),
  `Syntax.lean` (the five commands, moved from the same directory), `Registry.lean` (new: the
  Scenario and Query entries plus one project's `Conventions`), `Command.lean` (facade),
  `Command/Tests/Authoring.lean` (new: the test-only mutation helpers).
- `model/Temporal/Case/Syntax.lean` (new): the `case` command and the `caseTemplate` grammar, which
  stay Temporal-owned. `Temporal/Case/Conventions.lean` (new) declares the project's conventions
  once. `Temporal/Case/Registry.lean` keeps only the Case entry.
- `Temporal/Feature/Nexus/Success/{Syntax,Authoring}.lean` are deleted; `Model.lean` imports
  `Temporal.Case.Syntax` and nothing else from the old pair.

Four Temporal couplings replaced rather than moved:
- The `temporal.` Definition ID root and the `Temporal.Feature` namespace prefix are a project's
  declared conventions (`model_conventions root "temporal" under Temporal.Feature gaps ...`), held
  in an environment extension. A file that declares none gets an empty root and its whole namespace
  as the semantic family.
- `packageRelativePath` split the file path on `/model/`. The source now derives from the module
  name (`Temporal.Feature.Nexus.Success.Model` -> `Temporal/Feature/Nexus/Success/Model.lean`), so
  it depends on neither the checkout nor the package directory's name -- and the recorded values are
  unchanged, which is why Provenance is byte-identical.
- The hard-coded Known Gaps were Nexus constants inside the authoring core. They ride the same
  conventions declaration (`Temporal.Case.completionKnownGaps`) until .11 lets a Model file author
  its own. Deviation from the task's "move unchanged": they could not move into `Umpire.Command`
  without carrying `temporal.nexus.success.*` literals, which the SCP-02 acceptance forbids.
- `Umpire.Case.Producer.Identity.ofFixture` hard-coded `"temporal.case."`. It takes the root now,
  and `Temporal.Case.caseIdRoot` owns the value.

Also: `ModelVocabulary` is now an abbreviation of `Umpire.Case.Producer.Vocabulary` rather than a
second copy, so `producerInput` no longer converts it. `producerInput` itself stays, and the reason
is recorded in its docstring: the Producer needs the operation role and the declaring file's source,
which belong to the declaration rather than to the check, and the Query flattened to the three
fields it reads. Diagnostics lost their "Nexus model" prefix; the setup-domain, sorted-Action and
sorted-start-state requirements each say what they require, with `#guard_msgs` pins (the setup one
is new). `Origin.occurrence` and `outcomeIdAt`/`factIdAt`/`relationIdAt` are deleted.
`tools/umpire/internal/retiredvocabulary/check.go` moves the retired-keyword exemption to the new
path.

Byte pin: `make umpire-check-case-runtime-conformance` and `make umpire-check-goldens` are clean
with no regeneration, so every fixture, golden, `CheckedQuery.id` and fingerprint is unchanged and
the Nexus success family is still `temporal.nexus.success`.

`make umpire-check-regression` is exit 0 end to end (571 Lean jobs, all eight conformance checks,
9 passing live identities). `make lint-model` reports 0 findings outside generated
`Temporal/API/Proto.lean`; `git grep` for a Temporal namespace, import or semantic prefix under
`model/Umpire` is empty, which is what `umpire-check-regression`'s own scan enforces.

AUT-07a is drafted in `.plans/UMPIRE4_SPEC.md` marked "*(drafted by fn-83; awaiting GOV-02
approval.)*". It is NOT approved.

Swept in, not mine: `model/Temporal/Feature/Workflow/Start/DESIGN.md` (271 lines), a design specimen
the parallel session left untracked in the shared checkout. `git add -A` staged it; per this run's
instructions it was not reverted. The impl-review flagged it as the one P2, correctly, as
out-of-scope for this commit.

Review: SHIP, 1 finding, which is that swept file.
Pinned reviewer `claude:claude-fable-5-1:high` is account-limited for this session, so the review
ran on `claude:claude-sonnet-4-5:high` -- a same-family fallback, not an equivalent cross-family
review.

stage: impl-review - ran (model: claude-sonnet-4-5, high; fable pinned but account-limited)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7c42dec82c
- Tests: make umpire-check-regression (exit 0), make umpire-check-case-runtime-conformance (no regeneration needed), make umpire-check-goldens, make lint-model (0 findings outside generated Temporal/API/Proto.lean), make umpire-check-retired-vocabulary
- PRs: