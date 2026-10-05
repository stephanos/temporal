---
satisfies: [R2, R3, R4]
---
# fn-127-simplify-the-dsls-words.2 Read composition law parameters through a member with through

## Description
Implements R2 and closes the spec (R3, R4). A composition's capability fields read a member's status set as `through(_.activity)(Admission.paused)`, and the forwarding objects `OverQueue` and `OverMatching` (status-set defs only) are deleted.

**Cross-spec entry gate:**
- Task 1 is done.
- Before fn-124.7 retires the golden harness. If it has already landed, record the function deltas in a before/after projection of the reader's outputs under `.flow/tmp/`.
- Not concurrently with fn-124.8.

**Size:** M
**Files:**
- `model/umpire/Compose.scala` (or `Capabilities.scala`, wherever capability fields are typed) for `through`;
- `model/irgen/Capabilities.scala` (or wherever capability fields bind defs; see fn-122 R3), plus a lifting fixture and a refusal fixture;
- `model/temporal/features/standaloneactivity/compositions/{Model,Capabilities}.scala`;
- `tools/umpire/internal/golden/config.json`;
- `model/README.md`, `.plans/DSL_OPERATORS.md`, `.plans/DSL_SIMPLIFICATION.md`.

**Touches:** [model/umpire/**, model/irgen/**, model/temporal/features/standaloneactivity/compositions/**, tools/umpire/internal/golden/config.json, model/README.md, .plans/DSL_OPERATORS.md, .plans/DSL_SIMPLIFICATION.md]

### Approach
- `through[S, M](select: S => M)(p: M => Boolean): S => Boolean` is framework core with no Temporal word, documented as core (not sugar: it is not a respelling of another form).
- The lifter binds a capability field written `through(path)(def)`. Fn-122 R3 binds a direct def reference; here the lifter lifts one function `s => def(s.<path>)` under a derived name and binds that. It refuses:
  - a selector that is not a field path;
  - a predicate that is not a named def of the lifted sources.

  Composition stays refused anywhere a lambda is refused today (`model/README.md` 618-620).
- Rewrite `overQueueCapabilities` and `overMatchingCapabilities` with `through`, and delete the forwarders. `phase` reads `through(_.activity)(Admission.phase)`.
- Record each removed forwarder's function name against its replacement in `function_name_substitutions`. The law tables, verdicts and Cases must not change.
- Close: run the full gates once, update `model/README.md`'s composition section and `.plans/DSL_OPERATORS.md`, and mark ranks 1 and 6 done in `.plans/DSL_SIMPLIFICATION.md`.

### Investigation targets
**Required:**
- `model/temporal/features/standaloneactivity/compositions/Model.scala:25-75`, `compositions/Capabilities.scala`
- `model/irgen/Capabilities.scala` (field binding), `model/irgen/Claims.scala:540-600` (folding)
- `model/README.md` around lines 600-630 (refused lambdas)
**Optional:**
- `.flow/specs/fn-122-capabilities-and-their-laws.md` R3

### Quick commands
```bash
make umpire-check-model && make lint-model
go test -count=1 -tags test_dep -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower
```

### Execution constraints
- The only IR delta is the replaced functions' names and positions. A law-table, verdict or Case difference stops the task.

## Acceptance
- [ ] `through(selector)(predicate)` exists in `model/umpire` with no Temporal word. The lifter lifts it as one function and refuses a non-path selector and a non-def predicate, each at its line, with one lifting fixture and one refusal fixture.
- [ ] The forwarding objects in `compositions/` are gone, their state case classes stay, and both composition capability defs use `through`.
- [ ] Law tables, verdicts, Query answers and Cases are identical. The replaced function names are recorded in the golden configuration (or in the `.flow/tmp/` projection if fn-124.7 has landed).
- [ ] The docs of R3 are updated, and `.plans/DSL_SIMPLIFICATION.md` marks ranks 1 and 6 done.
- [ ] All R4 gates pass. The done summary lists the IR deltas.


## Done summary
# fn-127.2 summary: `through` for composition law parameters (closes fn-127)

### Commits (branch umpire-fn127-2, on 88d83cde12)
- c5d458ddae feat(model): read a member's def through a field path with through
- 0bae7336d0 refactor(model): read the record's status sets through the activity member
- 16bafd1722 docs: describe through, and mark the DSL study's ranks 1 and 6 done
- c3a9d3d4e5 docs(model): name no Temporal concept in through's framework doc
- 29829e337a fix(model): name through in an overriding refusal, and lift it as a shared claim's argument (review notes)

### What changed
- `model/umpire/Compose.scala`: core `def through[S, M, A](select: S => M, read: M => A): S => A`, documented as core with no Temporal word.
- Lifter (`model/irgen/Expressions.scala`, `Context.scala`, plus one-line `functionName` uses in `Claims.scala` and `Capabilities.scala`): `forwardedDef` recognizes a `through(...)` call. It checks the call (the selector must be a non-empty field path, and `read` must be a def of the lifted sources or a bound one). Then it returns a synthetic symbol that stands for the function wherever a def is bound or named. `callee` lifts that function lazily, on its first call, as `s => read(s.<path>)` under `<state class>.through.<path>.<def full name>`. Both refusals are reported at the offending argument's line.
- Fixtures:
  - `lifts/Capabilities.scala`: `pair` now reads through one-field and two-field paths, and the `Pairs.paused`/`Pairs.running` forwarders are deleted. Added in review: `rightNeverHeld`, a shared claim `neverHeld(pair)(through(_.right, Jobs.paused))`. It covers the declaring-function-argument path (`Claims.boundDef`), the form `compositions/Properties.scala` uses.
  - `lifts/CapabilityRejects.scala`: adds `throughComputed` and `throughLambda`. Added in review: `overridingThrough`. Its refusal now names the `through` read ("overriding pausedIsNotDispatched names a def that takes the law's parameters, not a member's def read with through: …") instead of the placeholder symbol's internal name (`Capabilities.scala`, `sameSignature`).
  - `crossed/Capabilities.scala`: adds `throughForeign`, refused by the compiler at 30:80.
  - Expected `capabilities.json`, `capabilities.laws.json` and `rejects.txt` were regenerated, and `Fixtures.test.scala` is updated.
- Models: `compositions/Capabilities.scala` uses `through(_.activity, Admission.{phase,paused,running})`. `compositions/Properties.scala` passes `atMostOneActive(c)(through(_.activity, Admission.twoActive))`. The `object OverQueue` and `object OverMatching` forwarders are deleted; their case classes stay.
- Docs:
  - `model/README.md`: the composition section, the refused-lambda paragraph, how a law is lifted, the framework list, and core/sugar.
  - `.plans/DSL_OPERATORS.md`: new entry 7, `through` as a word.
  - `.plans/DSL_SIMPLIFICATION.md`: ranks 1 and 6 marked done in both tables.

### IR deltas (model/ir only; model/cases and *.lint.json unchanged)
Only `activity-system.json` and `activity-system.laws.json` changed. Old → new, with P = `temporal.features.standaloneactivity.`:

| Old | New | New position |
| --- | --- | --- |
| P`compositions.OverQueue$.paused` | P`compositions.OverQueue.through.activity.`P`admission.Admission$.paused` | compositions/Capabilities.scala:40 |
| P`compositions.OverQueue$.running` | …`OverQueue.through.activity.…Admission$.running` | Capabilities.scala:44 |
| P`compositions.OverQueue$.phase` | …`OverQueue.through.activity.…Admission$.phase` | Capabilities.scala:33 |
| P`compositions.OverQueue$.twoActive` | …`OverQueue.through.activity.…Admission$.twoActive` | compositions/Properties.scala:30 |
| P`compositions.OverMatching$.paused` | …`OverMatching.through.activity.…Admission$.paused` | Capabilities.scala:61 |
| P`compositions.OverMatching$.running` | …`OverMatching.through.activity.…Admission$.running` | Capabilities.scala:65 |
| P`compositions.OverMatching$.phase` | …`OverMatching.through.activity.…Admission$.phase` | Capabilities.scala:54 |
| P`compositions.OverMatching$.twoActive` | …`OverMatching.through.activity.…Admission$.twoActive` | Properties.scala:43 |

Each old function was at compositions/Model.scala:32-35 or 54-57.

- Every reference was renamed, including the laws sidecar's `bindings`.
- Parameter types are identical.
- The two `phase` bodies now call the existing `Admission$.phase(s.activity)` where the forwarder read the field `s.activity.phase`. The meaning is the same.
- The other position changes are confined to `compositions/{Model,Capabilities,Properties}.scala`: declarations below the deleted objects moved up, and the capability declarations were reformatted.
- Projection: `python3 .flow/tmp/fn127-2/project.py 88d83cde12` compares the task's base revision with the working tree (or a second revision) and produces `.flow/tmp/fn127-2/ir-deltas.json`. It was rerun after the review fixes. After the rename, with line numbers ignored, both files are equal apart from the 8 replaced functions. No function was added or removed beyond these.

### Decisions
1. **One parameter list, `through(select, read)`, instead of the curried `through(select)(read)`.** Tested with Scala 3.9.0: Scala does not carry the expected type `S => Boolean` into the first parameter list, so `through(_.activity)(…)` fails with "value activity is not a member of Any" unless the state type is written out. With one parameter list, Scala infers the state in capability fields and in declaring-function arguments. The order stays selector first. This is documented in the README, DSL_OPERATORS and DSL_SIMPLIFICATION.
2. **The result type is generic (`A`), not `Boolean`.** The task requires `status = through(_.activity, Admission.phase)`, and `Admission.phase` returns the phase, not a Boolean.
3. **`through` lives in `model/umpire/Compose.scala`.** It reads a member of a composed state, and it serves capability fields and declaring-function arguments alike. Because it goes through `forwardedDef`, it is accepted exactly where a def reference is. It is refused where a def is not enough: `keeps` still needs a field-path def, and `overriding` needs the law's signature.
4. **The IR function is lifted lazily, on first call.** An unread binding therefore adds no function, as with a def. The derived name `<state class>.through.<path>.<def full name>` is unambiguous per state type, path and def.
5. **`twoActive` also moved to `through`.** It was one of the forwarders' four status-set defs, and it is passed to `atMostOneActive`, a declaring-function argument.
6. **The renames are not recorded in `function_name_substitutions` (`tools/umpire/internal/golden/config.json` is unchanged).** `Config.FunctionsRenamed` (golden.go:827) rejects any substitution whose old name no frozen input declares. The frozen goldens predate fn-122, so none declares the `OverQueue$`/`OverMatching$` forwarders (checked in the archive and in the migration testdata). The golden comparison already reads functions by reference (`functions_by_reference`), and the original-baseline and migration tests pass unchanged. Following the spec's own fallback form, the deltas are recorded in the before/after projection under `.flow/tmp/fn127-2/` (`project.py`, `ir-deltas.json`).
7. **`model/SEMANTICS.md` is unchanged.** `through` lowers to an ordinary IR function and has no IR meaning of its own. The R3 grep for old words over `model/`, the README, SEMANTICS and DSL_OPERATORS finds nothing.

### Gates (logs in .flow/tmp/fn127-2/; status lines in gates.status)

| Gate | Result | Log |
| --- | --- | --- |
| `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` | ok | gen2.log |
| Original-baseline check (internal/golden, model, lower) | ok | original-baseline.log |
| `make lint-model` | ok after one fix: `s""` → `""` | lint-model.log |
| `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` | ok | model-gate.log; rerun after the doc fix: model-gate2.log |
| Full Go suite, `-json -p 2` (wall 337s) | see below | go-suite.log |
| `go test ./tools/umpire/model` (rerun) | ok | go-model.log |
| `go test -p 1 ./tools/umpire/export` | ok | go-export.log |
| `make umpire-check-cases` | ok | check-cases.log |
| `make umpire-check-fixtures` | ok | check-fixtures.log |
| `make canary-check-case` | ok | canary-check-case.log |
| `make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast` | ok | lint-code-fast.log |

After the review fixes (29829e337a):
- `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` was ok (`gen3.log`). Only the lifter fixture outputs changed: `capabilities.json` and `rejects.txt`. `model/ir` and `model/cases` are unchanged.
- `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` was ok (`model-gate3.log`, 104s).
- The original-baseline check was ok (`original-baseline2.log`, 131s).
- The projection was rerun and is still OK.

In the full Go suite, every package passed except two:
- `TestFrameworkNamesNoTemporal`: the first doc example of `through` named `activity` and `Admission`. Fixed in c3a9d3d4e5, and `./tools/umpire/model` passed on the rerun.
- `tools/umpire/export`: OOM-killed again (`signal: killed`) at -p 2. It passed when rerun alone at -p 1.

### Blocked / open
- None. `flowctl done` was not run, as instructed. MILESTONES.md is not updated; the conductor does that after the merge.
- Subagents: 0.

Review: claude-opus-5-5 at high, fresh context (host-dispatched subagent). Writer and reviewer are the same family (Opus). Round 1: SHIP, no P1/P2. The reviewer compiled the curried alternatives with Scala 3.9.0 and confirmed the one-list spelling: the curried form fails in capability fields, declaring-function arguments and typed vals. It also re-ran the projection against the base (OK). Three P3s were applied in 29829e337a: the `overriding` refusal names `through`; `project.py` takes the base revision; and a fixture for `through` as a declaring-function argument. The fn-126 spec's sketch now uses the one-list spelling (host edit with this receipt).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c5d458ddae, 0bae7336d0, 16bafd1722, c3a9d3d4e5, 29829e337a
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks, go test -tags test_dep -count=1 -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower, make lint-model, make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks, go test -count=1 -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/... (model rerun green after doc fix; export OOM-killed, rerun -p 1 green), make umpire-check-cases, make umpire-check-fixtures, make canary-check-case, make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast, after review fixes: make umpire-gen-model / umpire-check-model MODEL_GATE_ARGS=--skip-go-checks; OriginalBaseline check; project.py 88d83cde12
- PRs: