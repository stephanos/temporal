---
satisfies: [R1, R7, R8]
---
# fn-114-state-every-scala-model-declaration-once.1 Declare IR-file roots in Scala and lift every Model IR file in one run

## Description
Settle the IR-file declaration's shape (a `val` per file, an annotation, or one registry; formerly the spec's Parked unknown) and move the six root lists out of the gate, before any Model file is renamed. Roots that name declarations by value survive the later `Claims.scala` -> `Properties.scala`/`Queries.scala` moves, so this goes first. Also record the before-metrics every later task and the closing task compare against, and the exact metadata allowance file moves need.

**Cross-spec entry gate (flowctl tracks only same-spec deps):** start only after fn-112 is closed (fn-112.10 done: showcase constructs, DefinitionScope and the structural Case-byte freeze exist) and fn-120.1 (named-choice syntax) is done. Verify both with `flowctl show` before claiming. fn-120.3 snapshots lint findings against the roots this task makes Scala-owned, so finish this before fn-120.3 starts.

**Size:** M
**Files:** `model/gate/Roots.scala` (deleted or reduced to non-Model wiring), `model/gate/Gate.scala` lift stage, `model/lifter/Lift.scala`, the lifter's root resolution, one IR-file declaration per IR file beside its Models (including `standaloneactivity`, per the spec's amended Boundaries), `model/umpire` declaration for it, lifter fixtures, `.flow/tmp/fn114-1/**`.
**Touches:** [model/gate/**, model/lifter/Lift.scala, model/lifter/Lifting.scala, model/lifter/Context.scala, model/lifter/Declarations.scala, model/lifter/test/**, model/lifter/testdata/**, model/umpire/**, model/temporal/nexuscaller/**, model/temporal/standaloneactivity/**, model/temporal/taskqueue/**, tools/umpire/internal/golden/**, model/README.md, .plans/UMPIRE_MODULES.md, .flow/specs/fn-114-state-every-scala-model-declaration-once.md, .flow/tmp/fn114-1/**]

### Approach
- Record before-metrics first, with one reproducible command: lines per Model folder and of `model/gate/**`, string literals per Model file by fn-112 R18's syntax-aware method (reuse the fn-112.1 counting command; do not write a second counter), and wall-clock time of the gate lift stage (currently one JVM per IR file in a `Future`). Store under `.flow/tmp/fn114-1/`.
- Choose the declaration shape against what the lifter reads from TASTy: prefer one typed value per IR file naming its declarations by reference (`irFile("nexus-caller")(nexusProduct, ...)`-like) over an annotation or a global registry. Record the settled shape in the spec's API Contracts (`flowctl spec set-plan`).
- Teach `lift` (`@main def lift` at `model/lifter/Lift.scala:90-139`; `class Lifter(roots, prefixes)` at :37, built at :128) to discover every IR-file declaration and write every Model IR file in one JVM. Each IR file gets fresh `Context` accumulators and caches (`Context.scala:59-84`: `renamed`, `folded`, `lifting`) over one shared symbol index, and every refusal still names its IR file, as the gate's per-file message does today (`Gate.scala:376`). Keep each IR file's `source` string identical by deriving the same root strings from the referenced symbols. A Scala-owned root naming nothing fails to compile (fixture roots keep the lifter refusal); a declaration in two files is lifted into both.
- Gate: replace the per-file lift loop (`Gate.scala:354-378`) with the single run; keep `Gate.settle` orphan/stale checks (:429-448). Lifter fixtures keep their own per-fixture lift (`Fixtures.test.scala:76-150`) because some must fail.
- File-move allowance for tasks 2-6: the lifter names top-level functions after their file's package object (`X$package$`), so moving a file changes source paths/lines, the `source` root string and moved-function symbols. Confirm the fn-112.1 harness permits all three through `tools/umpire/internal/golden/config.json` (source-path renames, source labels, `function_name_substitutions`); where it does not, add exactly that support. The spec's R1 list (amended at planning) already names these deltas; tasks 2-6 add per-move entries only.
- Measure the lift stage again. If the single run is slower, use R8's fallback: the gate starts one lifter JVM per IR file in parallel, each selecting its Scala IR-file declaration by name, so roots stay in Scala and the gate names no Model declaration; say so in the done summary.

### Investigation targets
**Required:**
- `model/gate/Roots.scala:1-101` - the six root lists (57 FQNs after prefix expansion)
- `model/gate/Gate.scala:340-450` - fixture stage, lift stage, settle
- `model/lifter/Lift.scala:30-144` - Lifter class, CLI, single-model output
- `.flow/tasks/fn-112-make-the-standalone-activity-scala.1.md` - baseline harness and literal-count command to reuse
**Optional:**
- `model/lifter/test/Fixtures.test.scala:76-150` - fixture lift and FQN fixture roots
- `tools/umpire/internal/golden/golden.go`

### Quick commands
```bash
make umpire-check-model
go test -count=1 -tags test_dep ./tools/umpire/model/... ./tools/umpire/lower/...
make lint-model
```

### Execution constraints
- No IR schema change; in this task every `model/ir/*.json` and `model/cases/*` byte stays identical in this task (R1 golden set plus the fn-112.1 equivalence command).
- Evidence goes to `.flow/tmp/fn114-1/`.
## Acceptance
- [ ] One IR-file declaration per IR file lives beside its Models; the gate program contains no fully qualified Model declaration name.
- [ ] The lifter writes all six Model IR files byte-identically from the Scala declarations, in one run or under R8's per-file fallback; a declaration listed for two files lifts into both (fixture). A Scala-owned root that names nothing fails to compile (fixture); the existing `lift: root <fqn>: names no declaration` refusal stays tested for the fixture lift (`Lifting.scala:22`, `expected/rejects.txt`).
- [ ] Each IR file is lifted with fresh per-file state, and a refusal names its IR file.
- [ ] Before-metrics (lines, literals per Model file, lift-stage wall time) and the after lift-stage time are recorded with their command; the faster arrangement is kept under R8.
- [ ] The settled declaration shape is recorded in the spec's API Contracts; the harness supports the file-move deltas named in R1's allowed list.
- [ ] R1 goldens, fn-112.1 equivalence, model gate, lint-model and Umpire Go tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
