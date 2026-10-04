---
satisfies: [R1, R3]
---
# fn-114-state-every-scala-model-declaration-once.7 Retire the string-named declaration forms from the DSL and the lifter

## Description
Once no Model, fixture or test uses them, remove the string-named overloads of every R2 declaration kind from `model/umpire` and their cases from the lifter, keeping only fn-112's one explicit-name form. Touch the standalone activity Model only if it still uses a retired form (spec Boundaries).

**Size:** M
**Files:** `model/umpire/{Action,Assume,Monitor,Channel,Machine,Compose,Claims}.scala`; `model/lifter/{Declarations,Claims,Compositions,Constants}.scala`; refusal fixtures under `model/lifter/testdata/`; `model/README.md`, `model/SEMANTICS.md` author-surface sections.
**Touches:** [model/umpire/**, model/lifter/**, model/temporal/standaloneactivity/**, model/README.md, model/SEMANTICS.md]

### Approach
- Inventory callers per overload with grep before deleting (line refs as of 2026-10-03, re-locate after fn-112: `action`/`timer`/`internal` Action.scala:107-113, `assume`/`hole`/`leadsTo` Assume.scala:15-56, `monitor` Monitor.scala:37, `channel` Channel.scala:88, `machine`/`restrict` Machine.scala:59/:140, `compose`/`sync` Compose.scala:23/:47, `property`/`scenario`/`Limits`/`query` Claims.scala:81-159).
- Delete each overload and its lifter branch (Declarations.scala:35-429, Claims.scala:82-252, Compositions.scala:46) in the same change; for each, add a fixture proving the old form no longer compiles or is refused at its line.
- Rewrite any test still using a form in the same task (R3 errors clause). Update docs to show one way to name a declaration.

### Investigation targets
**Required:**
- `model/umpire/Claims.scala`, `model/umpire/Machine.scala`
- `model/lifter/Declarations.scala`, `model/lifter/Claims.scala`
- `model/lifter/test/Fixtures.test.scala:200-250` - refusal assertion format `File.scala:line:col`

### Quick commands
```bash
grep -rnE '(machine|property|scenario|query|action|timer|monitor|channel|assume|hole)\("' model/temporal model/lifter/testdata
scala-cli test model/lifter
make umpire-check-model
```

### Execution constraints
- No IR or Case byte changes; refusal fixtures are new files only.
## Acceptance
- [ ] No string-named overload of an R2 kind remains in `model/umpire` or the lifter, apart from fn-112's one explicit-name form.
- [ ] Each retired form has a fixture proving it no longer compiles or is refused at its line.
- [ ] README/SEMANTICS show one naming form; lifter tests, model gate and R1 goldens pass.
## Done summary
Retired the string-named and string-keyed declaration forms from the DSL and the lifter. Commits: aae7575aa3 (retirement), 68443151d3 (captured fixture made admissible).

**Removed (DSL overload and lifter case together)**
- `timer(name)`, `internal(name)`, `hole(name)`, and the `name` parameter of `channel`.
- `Machine.restrict(family, name)`, `compose[S](family, name)(…)`, and the string-keyed `compose[S]("field" -> m)`.
- The string-member `sync(name, "f" -> a, "g" -> b)`, `replaces(field: String, …)` and `ScenarioBuilder.actionKeys`.
- Composition's member, sync and replacement types are narrowed to selectors, and `@targetName("composeBySelectors")` is dropped.
- Lifter hints that advertised a retired form were updated (hole, channel, `arrow`, the `synced` refusal).

**Kept: each kind's one explicit-name form, still needed**
- `machine(family, name)`: design machines in CloseReset and Realizations.
- `action(name, party)`: Members `turn-on`.
- `monitor(name, initial)`: the archived golden refusal "two monitors are named twice" needs two monitors under one name.
- `assume(name)`: closepolicy and taskqueue names differ from their vals.
- `property` / `scenario` / `query(name)`: declarations in defs and lists.
- `Limits(name, …)`: inline Limits with no val.
- `leadsTo(name)`: Progress is not an R2 kind and has no captured form. Adding one is a construct the Boundaries exclude.
- `whenAction(String)`: a reference, not a name.

**Fixtures (R2/R3)**
- The refusal fixtures now state names through captured forms wherever the string repeated the val, and every retired form is gone. Files: Rejects, Crossed, ActionInput, NamedInput, SameState, werror/Steps, unsupported/Unsupported and `model/umpire/Inputs.test.scala`.
- `misplaced` now uses `Flags(left, middle, right)` with `.replaces(_.middle, first)`.
- rejects.txt: no text changed. One position moved, 165 → 166 for "middle replaces first…".
- All 16 archived refusals still appear verbatim apart from positions (OriginalInventory).
- Pins moved: werror `21:58` → `23:58`, unsupported `:18` → `:20`.
- **Spelled.scala is retired.** Its pair test proved Captured == Spelled at HEAD (`lifter-B-0-head.log`) before the deletion.
- **Captured.scala is now a golden fixture**, pinned as `lifts/expected/captured.json`.
  - Its DefinitionScope test still asserts the pinned IDs and type names.
  - `later_inventory` in `tools/umpire/internal/golden/config.json` gained one appended entry.
  - The Go tooling reads every expected json, so the fixture's `ledger` realization now declares its correlated observation. That is one line, and only captured.json changes.
  - `TestLiftedModelsAreAdmitted` lists `captured`.
- **New compile-refusal fixture** `model/lifter/testdata/retiredNames/Invalid.scala` writes each retired form beside its current one. Each retired spelling fails at its pinned `line:col`, and the current spellings compile. The test is in Fixtures.test.scala.

**IR/Cases.** `model/ir`, `model/cases` and the existing `expected/*.json` are byte-identical. original.json is untouched. Golden config: one append only.

**Docs.**
- The model/README.md naming paragraph now lists the one explicit form per kind. It also says which kinds have none and that compositions use selectors, never string keys.
- model/SEMANTICS.md has no author-surface text on naming (it specifies IR semantics), so it is unchanged.

**Decisions taken autonomously**
- Retire an explicit form only where no Model or positive fixture needs it.
- Keep `monitor(name, …)` because the R1 archived refusal needs it.
- Retire the string `replaces` by moving `misplaced` to a selector form that yields the archived message verbatim.
- Make Captured a golden rather than delete it, since it alone covers captured Party/Entity/Observation, timer, evidence exceptions and DefinitionScope pins.
- The standalone activity Model is not touched: it uses no retired form.
- Execution constraint "refusal fixtures are new files only": the new fixture is a new file. The existing refusal fixtures had to be rewritten anyway (R3 errors clause).

**Review.** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with one P3, deferred:
- **P3, deferred.** The `synced` refusal for an input-taking partner no longer suggests `actionKeys`. When both members' actions take inputs, a pinned Scenario cannot list that class any more; only `.free` remains. No Model or fixture uses this, and restoring it would be a new construct. This is a finding for fn-112 or a later spec.
- FYI: `model/gate/SourceMetrics.scala:36` and `model/gate/test/SourceMetrics.test.scala` still name `actionKeys` and retired spellings as metric strings. They are outside this task's Touches; fn-114.8/.9 should handle them.

**Shared with the fn-122 lane:**
- `tools/umpire/internal/golden/config.json` (later_inventory)
- `model/lifter/{Claims,Compositions,Declarations}.scala`
- `model/umpire/{Action,Assume,Channel,Claims,Compose,Machine}.scala`
- `model/lifter/test/Fixtures.test.scala`
- `tools/umpire/model/fixtures_test.go`
- `model/README.md`

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: aae7575aa3, 68443151d3
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, x2), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, includes scala-cli test model/lifter with the retiredNames fixture), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|Migration' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), scala-cli test model/project.scala model/umpire (exit 0), make lint-model (exit 0), go test -json -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 1, 652 s: only tools/umpire/model, captured.json realization not admissible; fixed in 68443151d3), go test -json -tags test_dep -count=1 -p 2 ./tools/umpire/model ./tools/umpire/internal/golden (exit 0 after the fix), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: