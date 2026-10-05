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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
