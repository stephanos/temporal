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
**Files:** `model/Temporal/Feature/Nexus/Race/**` (from `Nexus2/`), `model/Temporal/Feature/Nexus/Success/**` (from `Nexus3/`), `model/Temporal/Feature/Nexus/Race/Terminal.lean` (from `Nexus3/Cancellation.lean`), `model/Temporal/Feature.lean`, `model/Temporal/System/Nexus/{ImplementationLink,ImplementationLinkTests}.lean`, deletion of `model/Temporal/ImplementationLinkTests/`, `model/TemporalModelTests.lean`, `model/TemporalExperimentalTests.lean`, `model/Temporal/Feature/Nexus/{,Race/}COVERAGE.md`, `model/lakefile.lean`, `Makefile`, `tools/umpire/cmd/umpire-gen-regression-views/generate.go` (hardcodes `temporal-model-inspect` at `:20`), `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go` (hardcodes `temporal-testpilot` at `:23`), `model/ModelLint/ImportGraph.lean` + tests, `tests/*` live test names, docs, gate
**Touches:** [model/Temporal/**, model/TemporalModelTests.lean, model/TemporalExperimentalTests.lean, model/lakefile.lean, model/ModelLint/**, model/README.md, Makefile, tests/**, tools/umpire/cmd/umpire-gen-regression-views/**, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tools/umpire/regression/ci_workflow_test.go, tools/umpire/internal/retiredvocabulary/check.go, .flow/specs/*.md]

### Approach
- Moves: `Nexus2/*` to `Nexus/Race/*` keeping file names; `Nexus3/Cancellation.lean` to `Nexus/Race/Terminal.lean`; `Nexus3/Nexus.lean` to `Nexus/Success/Model.lean`, `Nexus3/Testpilot.lean` to `Nexus/Success/Producer.lean`, the rest of `Nexus3/` to `Nexus/Success/` unchanged. Identity roots `temporal.nexus2.*` to `temporal.nexus.race.*`, `temporal.nexus3.*` to `temporal.nexus.success.*`, the terminal table to `temporal.nexus.race.terminal.*`; `caseId` values in the Nexus3 Producer follow and the Case fixtures regenerate.
- Keywords in `Nexus/Success/Syntax.lean` (fn-80 R2 generalizes it first): `starts`, `ends`, `steps`, `when X`, `state`, `scenario`, `steps N`, `actions N`, `search N`, `find`, `verify`; keep a macro arm per retired keyword that throws a located error naming the replacement, with one `#guard_msgs` block per keyword in `Nexus/Success/Tests.lean`. Re-baseline the 31 + 2 + 5 `#guard_msgs` blocks in the moved test modules by running them.
- Implementation Link tests: fold `model/Temporal/ImplementationLinkTests/Nexus.lean` into `model/Temporal/System/Nexus/ImplementationLinkTests.lean`; remove the `temporalImplementationLinkTest` class, the exact classifier, and `closedClassifierNamespaces` entry in `ImportGraph.lean:30,124-128,138-139,214,237,305` and the test at `ImportGraphTests.lean:514`.
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
## Acceptance
- [ ] No module, directory, or identity root named `Nexus2`/`Nexus3` remains; `Nexus/Race/**` and `Nexus/Success/**` build; the Implementation Link imports `Nexus.Race.Terminal`
- [ ] The five commands accept the R6 keyword set and each retired keyword produces a located macro error naming its replacement, asserted by one `#guard_msgs` block per keyword
- [ ] `Temporal.ImplementationLinkTests` is gone with its lint class and exception; `EVIDENCE.md` files are `COVERAGE.md` with links retargeted; `Nexus.md` and `Integration.md` use the new keywords
- [ ] Lake executables and Make targets use the `umpire-` convention and the two Go generators invoke the renamed exes; `make umpire-inspect`, `umpire-list`, `umpire-explain`, `umpire-gen-regression-views`, `umpire-check-regression` pass
- [ ] Regenerated Case fixtures pass the conformance check and the tagged live selector with the same satisfied Contract; `make lint-model` and the gate pass
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
