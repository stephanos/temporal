---
satisfies: [R6]
---
# fn-82-unify-the-umpire-and-testpilot.8 Nexus tree, command keywords, and tool names

## Description
Nexus tree, command keywords, and tool names (R6, spec §R6): generation numbers leave the module
tree, identity roots follow, the five commands take the spec's keyword set and reject the old
spellings with located errors, the Implementation Link test module merges, per-feature
`EVIDENCE.md` becomes `COVERAGE.md`, and the Lake executables and Make targets take the `umpire-`
convention.

**Size:** M (mechanical sweep across the Temporal tree)
**Files:** `model/Temporal/Feature/Nexus/Race/**` (from `Nexus2/`), `model/Temporal/Feature/Nexus/Success/**` (from `Nexus3/`), `model/Temporal/Feature/Nexus/Race/Terminal.lean` (from `Nexus3/Cancellation.lean`), `model/Temporal/Feature.lean`, `model/Temporal/System/Nexus/ImplementationLink.lean`, `model/TemporalModelTests/Nexus/ImplementationLink.lean` (from `model/Temporal/ImplementationLinkTests/Nexus.lean`), deletion of `model/Temporal/ImplementationLinkTests/`, `model/TemporalModelTests.lean`, `model/TemporalExperimentalTests.lean`, `model/Temporal/Feature/Nexus/{,Race/}COVERAGE.md`, `model/lakefile.lean`, `Makefile`, `tools/umpire/cmd/umpire-gen-regression-views/generate.go` (hardcodes `temporal-model-inspect` at `:20`), `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go` (hardcodes `temporal-testpilot` at `:23`), `model/ModelLint/ImportGraph.lean` + tests, `tests/*` live test names, docs, gate
**Touches:** [model/Temporal/**, model/TemporalModelTests.lean, model/TemporalModelTests/**, model/TemporalExperimentalTests.lean, model/lakefile.lean, model/ModelLint/**, model/README.md, Makefile, tests/**, tools/umpire/cmd/umpire-gen-regression-views/**, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tools/umpire/regression/ci_workflow_test.go, tools/umpire/internal/retiredvocabulary/check.go, .flow/specs/*.md]

### Approach
- Moves: `Nexus2/*` to `Nexus/Race/*` keeping file names; `Nexus3/Cancellation.lean` to `Nexus/Race/Terminal.lean`; `Nexus3/Nexus.lean` to `Nexus/Success/Model.lean`, `Nexus3/Testpilot.lean` to `Nexus/Success/Producer.lean`, the rest of `Nexus3/` to `Nexus/Success/` unchanged. Identity roots `temporal.nexus2.*` to `temporal.nexus.race.*`, `temporal.nexus3.*` to `temporal.nexus.success.*`, the terminal table to `temporal.nexus.race.terminal.*`; `caseId` values in the Nexus3 Producer follow and the Case fixtures regenerate.
- Keywords in `Nexus/Success/Syntax.lean` (fn-80 R2 generalizes it first): `starts`, `ends`, `steps`, `when X`, `state`, `scenario`, `steps N`, `actions N`, `search N`, `find`, `verify`; keep a macro arm per retired keyword that throws a located error naming the replacement, with one `#guard_msgs` block per keyword in `Nexus/Success/Tests.lean`. Re-baseline the 31 + 2 + 5 `#guard_msgs` blocks in the moved test modules by running them.
- Implementation Link tests: move `model/Temporal/ImplementationLinkTests/Nexus.lean` to `model/TemporalModelTests/Nexus/ImplementationLink.lean` (module `TemporalModelTests.Nexus.ImplementationLink`, class `.modelTests` by the `TemporalModelTests` prefix at `ImportGraph.lean:130`, which may import both Feature and System modules; a `Temporal.System.*` home would trip `systemIsolation` at `ImportGraph.lean:298-300`). Import it from `model/TemporalModelTests.lean`. Remove the `temporalImplementationLinkTest` class, the exact classifier, and the `closedClassifierNamespaces` entry in `ImportGraph.lean:30,124-128,138-139,214,237,305` and the test at `ImportGraphTests.lean:514`.
- Docs: `EVIDENCE.md` to `COVERAGE.md` in both trees, retarget links in `Race/README.md`; respell `Success/Nexus.md` and `Integration.md` to the new keywords (fn-67 closed before this spec started).
- Tools: Lake exes `temporal-model-inspect` to `umpire-inspect`, `temporal-testpilot` to `umpire-case`, `modelLint`/`modelLintTests` to `umpire-lint`/`umpire-lint-tests`, `testpilotProtoJSONFixture` to `umpire-protojson-fixture` (`umpire-correlated-fixtures` was task .7's); Makefile variables at `Makefile:120-126` and targets `umpire-list-nexus`/`umpire-explain-nexus` to `umpire-list`/`umpire-explain`; `lint-model` keeps its name. The Go generators hardcode the exe names (`umpire-gen-regression-views/generate.go:20`, `umpire-gen-case-runtime-conformance/generate.go:23`) and must change before the fixture regeneration in this task runs. Update `tools/umpire/regression/ci_workflow_test.go` where it pins target names.
- Regenerate fixtures and goldens; add compound old names to the gate; respell scanned docs and open specs (fn-70, fn-74, fn-79 name `Nexus3` or `temporal-testpilot`).

### Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Nexus/ImplementationLink.lean:1-6` and `model/Temporal/Feature/Nexus3/Cancellation.lean:1-34` — the import chain into Nexus2 via Nexus3
- `model/Temporal/Feature/Nexus3/Syntax.lean` — the five macros (post fn-80 shape)
- `model/ModelLint/ImportGraph.lean:101-140,205-240,296-310` — root classifiers and the Implementation Link exception
- `model/lakefile.lean:59-113` and `Makefile:120-140,1000-1017` — executables, variables, inspect targets
- `model/Temporal/Feature/Nexus2/AuthoringTests.lean:565,767` — expected messages that embed form names and JSON keys

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/Lifecycle/Target.lean` — the established model that stays in place
- `tests/testcore/testpilot/testdata/async-nexus-case.json` — `caseId` bytes that change with the identity root

### Key context
- `Nexus2.Lifecycle` (basic lifecycle) and `Nexus.Lifecycle` (established) are different models; move, never merge.
- The `#guard_msgs` re-baseline must come from running the modules, not from editing text.
- `lint-model` stays because MOD-11 and the `lint` aggregate cite it; the `Makefile:1421` diagnostic string names `Umpire.Core` and is unchanged.
- Retire `Temporal.Feature.Nexus2`, `Temporal.Feature.Nexus3`, `temporal.nexus2`, `temporal.nexus3`, `temporal-model-inspect`, `temporal-testpilot`, `umpire-list-nexus`, `umpire-explain-nexus`, `selected_actions`, `candidate_evaluations`, `resultingState`; never the bare keywords.
- Carried from .2: the retired gate still has no `resultingState` rule because `resultingState` is a live Nexus3 `require` keyword. Add it to the gate in the same commit that respells the keyword to `state`.
## Acceptance
- [ ] No module, directory, or identity root named `Nexus2`/`Nexus3` remains; `Nexus/Race/**` and `Nexus/Success/**` build; the Implementation Link imports `Nexus.Race.Terminal`
- [ ] The five commands accept the R6 keyword set and each retired keyword produces a located macro error naming its replacement, asserted by one `#guard_msgs` block per keyword
- [ ] `Temporal.ImplementationLinkTests` is gone; the tests build as `TemporalModelTests.Nexus.ImplementationLink` and `make lint-model` passes with the `temporalImplementationLinkTest` class, exact classifier, and closed-namespace entry removed; `EVIDENCE.md` files are `COVERAGE.md` with links retargeted; `Nexus.md` and `Integration.md` use the new keywords
- [ ] Lake executables and Make targets use the `umpire-` convention and the two Go generators invoke the renamed exes; `make umpire-inspect`, `umpire-list`, `umpire-explain`, `umpire-gen-regression-views`, `umpire-check-regression` pass
- [ ] Regenerated Case fixtures pass the conformance check and the tagged live selector with the same satisfied Contract; `make lint-model` and the gate pass
## Done summary
Generation numbers left the module tree, the five commands took the R6 keyword set, and the Lake executables took one naming convention.

- Moves: `Temporal/Feature/Nexus2/` -> `Nexus/Race/` keeping file names; `Nexus3/Cancellation.lean` -> `Nexus/Race/Terminal.lean`; the rest of `Nexus3/` -> `Nexus/Success/` with `Nexus.lean` -> `Model.lean` and `Testpilot.lean` -> `Producer.lean`. The terminal table's checks followed it into `Race/Tests.lean` rather than staying behind in the Success tree. Identity roots became `temporal.nexus.race.*`, `temporal.nexus.success.*` and `temporal.nexus.race.terminal.*`; Case fixtures and regression views regenerated through their owning targets.
- Keywords: `initial`/`terminal`/`transitions` -> `starts`/`ends`/`steps`, `when action X` -> `when X`, `resultingState` -> `state`, `behavior` -> `scenario`, `transitions N`/`selected_actions`/`candidate_evaluations` -> `steps N`/`actions`/`search`, `witness` -> `find`, `all` -> `verify`. Every retired spelling still parses: a keyword position accepts both and the elaborator rejects the retired one in place naming its replacement, with one `#guard_msgs` block per keyword. Only `scenario` had to become a reserved token, because a command's leading position cannot be a non-reserved symbol.
- ModelLint: `Temporal.ImplementationLinkTests.Nexus` became `TemporalModelTests.Nexus.ImplementationLink`, where the model-test class already permits importing both a Feature and a System module. That retired the `temporalImplementationLinkTest` class, its exact classifier, and the `closedClassifierNamespaces` mechanism, whose only entry was that namespace; the unclassified-module test now uses `TemporalVeilTests.UnclassifiedBridge`, unclassified for a still-live reason.
- Names: `umpire-inspect`, `umpire-case`, `umpire-lint`, `umpire-lint-tests`, `umpire-protojson-fixture`; `umpire-list` and `umpire-explain` replaced the `-nexus` suffixed targets. `EVIDENCE.md` became `COVERAGE.md` in both trees with links retargeted.
- Gate: added the compound names for both this task and the executables, plus three rules that had to be narrower than the shared compound pattern and are now each pinned by a rejecting and an accepting row in the vocabulary table test: `Nexus2`/`Nexus3` exclude a leading hyphen (immutable Flow spec slugs), `temporal-testpilot` excludes hyphens on both sides (three reservation-carrier wire headers), and the Nexus `require <label>: resultingState` spelling is held instead of the bare token, which stays live as an `Umpire.Property` field constructor. Every rule was proved to fire by planting it and reading the gate's report.

Review reached SHIP on round 3. Round 1 found that the lint driver still spawned the retired `modelLintTests` target, so `make lint-model` failed at its import-graph step rather than at its known baseline. Round 2 found that the `temporal-testpilot` sweep had eaten six references to the Flow spec `fn-72-extract-the-reusable-temporal-testpilot`, a record on disk -- the same "sweep rewrites a name that must not move" hazard the spec warns about -- and asked for the carve-out tests. Round 3's P3 and both FYIs were applied.

Carried debt: `Temporal.Feature.Nexus.Race.Race` stutters, because the spec's R6 text fixes "keeping its file names". `resultingState` cannot be held as a compound while `PropertyTraceField` and `PropertyClauseField` both carry that constructor. `.plans/UMPIRE_DSL_*.md` and `UMPIRE_ARCHITECTURE_REVIEW.md` still name the generations; they sit outside the gate's `UMPIRE4_*.md` glob by an existing scope decision.
## Evidence
- Commits: 2aa8937c32, 15ebd0d100, 6c47065765, 9942497b40, ca5ebaa6fb, 06def2bf0f
- Tests: make umpire-check-regression, make lint-model, make lint-code GOLANGCI_LINT_FIX=false, CC=/usr/bin/cc go vet -tags test_dep ./..., CC=/usr/bin/cc go test -tags test_dep -count=1 ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/..., make umpire-list, make umpire-explain QUERY=temporal.nexus.basic-lifecycle.query.async-start, make umpire-inspect SCENARIO=switch.query.exact-action, go run ./tools/planindex
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
