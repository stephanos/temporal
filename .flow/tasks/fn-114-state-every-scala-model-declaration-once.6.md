---
satisfies: [R1, R2, R6, R10]
---
# fn-114-state-every-scala-model-declaration-once.6 Restate the worker Model and the lifter fixtures with captured names and named choices

## Description
Convert the remaining consumers of string-named forms and unnamed branching outside the Nexus folders: the worker entity Model and every lifter fixture. fn-120.2 turns on the unnamed-branch refusal only after this spec has migrated all fixtures, so this task owns that fixture migration.

**Size:** M
**Files:** `model/temporal/worker/Worker.scala` (76 lines; `Family("temporal.worker")` :17, `action("workerStop", party)` :52-54, `machine(..., "polling")` :71) -> `worker/Model.scala` (plus Properties/Queries only if it has such declarations); `model/lifter/testdata/**` and `lifts/expected` IR; `model/lifter/test/Fixtures.test.scala`.
**Touches:** [tools/umpire/internal/golden/config.json, model/temporal/worker/**, model/lifter/testdata/**, model/lifter/test/Fixtures.test.scala, model/ir/nexus-caller.json]

### Approach
- Worker: captured names, `given` family, DefinitionScope so `worker.polling` keeps its ID (it is lifted into `nexus-caller.json`). Rename `Worker.scala` to `Model.scala`; its functions `stopStep`, `resumeStep`, `serveStep` change from `Worker$package$` to `Model$package$` in `nexus-caller.json`, recorded as `function_name_substitutions` entries.
- Positive fixtures: rewrite every string-named declaration to the captured form and every multi-result branch to fn-120's named choices, keeping expected IR identical except for inert choice names. Refusal fixtures that deliberately exercise a form task 7 retires stay for task 7.
- Fixture roots stay fully qualified in the test (spec: fixtures are the only place outside Models naming Scala sources); update any root a rename changed. Depends on task 2 (both edit fixture roots in `Fixtures.test.scala`) and task 5 (both add entries to the golden `config.json`).

### Investigation targets
**Required:**
- `model/temporal/worker/Worker.scala`
- `model/lifter/test/Fixtures.test.scala:76-245`
- `model/lifter/testdata/lifts/`
- `.flow/tasks/fn-120-adopt-what-quint-does-well-named.2.md` - what fn-120.2 expects to find migrated
**Optional:**
- `.flow/memory/bug/integration/feature-modules-must-import-temporal-2026-09-28.md` - pin worker IDs after a move

### Quick commands
```bash
scala-cli test model/lifter
make umpire-check-model
```

### Execution constraints
- Expected fixture IR and `nexus-caller.json` change only within the R1 allowed-difference list recorded in the spec (fn-120 inert choice names, file-move source paths/lines and `source` root strings, and moved-function symbols recorded as `function_name_substitutions` in `tools/umpire/internal/golden/config.json`, as fn-112's harness allows).
## Acceptance
- [ ] Worker declarations take names from `val`s; the folder uses `Model.scala`; `worker.polling` ID unchanged.
- [ ] Every positive lifter fixture uses captured names and names every multi-result branch; expected IR otherwise unchanged.
- [ ] Lifter tests, model gate and R1 goldens pass.
## Done summary
Restated the worker Model and the positive lifter fixtures with captured names and named choices. Commit: b68537d54a.

**Worker (R2, R6, R10)**
- `worker/Worker.scala` is now `worker/Model.scala`. There are no Properties, Queries or realization, so those files are omitted.
- The family is a `given`. `workerStop`, `workerResume`, `serve` and `polling` take their names from their vals.
- Step functions use `accept`/`stay`/`disabled` with `given Accepted[Outcome]`.
- `given DefinitionScope("temporal.worker.Worker$package$")` keeps every worker Definition ID, including `worker.polling`.
- Kept explicit: `Party("worker")` and `Entity("worker", key = "taskQueue")`. Their vals (`party`, `entity`) differ from the names, and features read `worker.party`. Neither is an R2 kind.
- Literals went from 8 to 5: the family root, the scope pin, the two names that differ from their vals, and the entity key. Lines went from 81 to 85.
- Two Model comments that said "from Worker.scala" now say worker/Model.scala, with the line counts unchanged (nexuscaller and standaloneactivity `Queries.scala`).

**Fixtures (R2, fn-120.2 readiness)**
- Strings that repeated their val, or the default Query/sync name, were dropped in Declarations, Presence, Channels, Typed, Admission, CloseReset, TaskQueue, Members and Realizations. Realizations also lost its redundant `Realization(name = …)` lines, and Patterns' composition is now keyed by field.
- No string remained to drop in Sugar, Totals, Captured, Choices, Inputs, Derived or Scripts.
- Named branches:
  - Admission `admitted`: `committed`/`redelivered`, the README and `Choices.scala` names for this function. The live admission's `admitted` is a different function.
  - CloseReset `committed` and `keptAtOperation`: `taken`/`rejectedForNow`/`ackLost`. `deliverStep`: `refused`/`rejectedForNow`. These are the live close policy's tokens.
  - No `choose` needed a helper call as an alternative.
- An IR scan of every lifted fixture finds no unnamed multi-result list except the deliberate `Choices.scala` `*Unnamed`/`unchosen` twins, which fn-120.2 retires.
- Kept on purpose:
  - Explicit names that differ from their val, or that have no val: Query names in lists, the `putBoth` and `tap_both-ways` syncs, the design machines in CloseReset and Realizations, `flushEventuallyRuns`, and Properties declared under a predicate's name.
  - Computed names.
  - `leadsTo("…")`, which has no captured form.
  - `.input[A]("…")` names and inline `Party("…")`, which are not R2 kinds.
  - `Spelled.scala` and the core/spelled twins of the pair tests.
  - Refusal fixtures (Rejects, ScriptRejects, MovedRejects and the refusal directories), left for task 7 because they pin lines.
- Copied Model text got names and choices only; nothing was polished.

**IR (R1)**
- Expected fixture IR differs only in lines and inert choice names.
- `model/ir` differs only in the worker's path, lines and the `Worker$package$` → `Model$package$` function symbols.
- No Case changed, so the canary identity is unchanged and nothing needed re-pinning.
- Golden config, appended only:
  - one `source_path_merges` entry: `model/temporal/worker/`
  - one `source_root_moves` entry: `Worker$package$.polling`
  - three `function_name_substitutions`: `stopStep`, `resumeStep`, `serveStep`

**Go tests**
- Re-pinned to the new fixture lines: admission, totals, interpret and declarations_pins.
- Expect the new choice names: interpret and checking. `choices_test.go` clears the source-given names in its "plain" model.

**Decisions taken autonomously**
- Used one directory merge for the worker, mirroring closepolicy, rather than per-file entries.
- Dropped the `name =` on Realizations and keyed the Patterns composition by field (R6). Both lifts compare equal.
- Fanned the work out to four Opus subagents (fixture groups) while doing the worker myself. Every lifted output was compared against a HEAD baseline, ignoring lines and choice names.

**Verification (`.flow/tmp/fn114-6/gates.status`, under the heavy lock)**
- These passed: gen-model, OriginalBaseline+Migration goldens, gen-fixtures+canary-gen-case (no change), check-model (includes the lifter tests), lint-model and lint-code-fast.
- Full Go suite: the first run failed only in tools/umpire/model, on six stale Channels/Presence line pins. After the fix, the package rerun exited 0. The other packages had passed in the full run.

**Review.** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with no findings.
- FYI: the Makefile `umpire-clean-scratch` glob is not this task's.
- P3, pre-existing: `model0/go/measure.sh` names an old worker path, in an archived tool.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b68537d54a
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|Migration' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), make umpire-gen-fixtures && make canary-gen-case (exit 0, no change), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, includes scala-cli test model/lifter), make lint-model (exit 0), go test -json -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 1: 6 stale line pins in tools/umpire/model only; fixed), go test -json -tags test_dep -count=1 ./tools/umpire/model (exit 0 after the fix), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: