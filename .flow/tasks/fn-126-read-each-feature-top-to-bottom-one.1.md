---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8, R12]
---
# fn-126-read-each-feature-top-to-bottom-one.1 Write the standalone activity as one feature file per folder, rename record/ and withTaskQueue/, and lint declaration order

## Description
Early proof point for the layout. Convert `features/standaloneactivity` and its two subpackages into one feature file each (R1, R2, R3), using today's declaration forms: module objects hold vocabulary, step functions, the machine `val`s, Properties, capabilities, Scenarios and Queries. Rename the subpackages `admission/` → `record/` and `compositions/` → `withTaskQueue/` (R12). Land the declaration-order lint (R4 a-d). Meaning and identity stay frozen (R5), with only recorded deltas.

**Cross-spec entry gate:**
- fn-114, fn-118 and fn-122 are closed, and fn-127 (Simplify the DSL's words) is closed.
- Never alongside fn-124.8.
- fn-126 lands before fn-124.7 (this task records into the golden harness).
- fn-125 stays paused until fn-126 closes.

**Size:** L
**Files:**
- `model/temporal/features/standaloneactivity/**`: `StandaloneActivity.scala`, `record/Record.scala` and `withTaskQueue/WithTaskQueue.scala` replace the per-kind files; `Realization.scala` gets imports and qualified references only;
- `model/check/**` (the lint) and its refusal fixtures;
- `model/project.scala` (if Scala's safe-init checkers are chosen);
- `tools/umpire/internal/golden/config.json`;
- `model/ir/activity*.json`, `model/cases/**`, the generated fixtures.

**Touches:** [model/temporal/features/standaloneactivity/**, model/check/**, model/project.scala, tools/umpire/internal/golden/**, model/ir/**, model/cases/**, tests/testcore/testpilot/testdata/generated/**]

### Approach
- Record the R11 baseline first: files, lines, hops from state to Query, cross-file references, the `disabled` count and the inverted-guard count, all under `.flow/tmp/fn-126/`.
- Layout per R2:
  - header comment;
  - top-level types;
  - signature (actions stay top-level here; task 3 groups them);
  - module objects in dependency order (`Product`, `Protocol`; the record designs; the compositions) built from today's vocabulary objects, so status-set function symbols do not move;
  - irFile roots last.
- Machine `val`s keep their names inside their module objects (R7).
- The feature section holds only what has no module object yet: the composition with the worker, `protocolCapabilities` (it reads the realization) and the irFile roots. Task 4 moves each to its final home.
- Pins (R6): the file-level pin stays. A module object that holds a monitor, assumption, hole or channel pins its former owner (`…System$package$` in `record/`). Two owners may pin one former owner; add the lifter fixture that proves it.
- Lint (R4 a-d):
  - decide between `-Wsafe-init`/`-Ysafe-init-global` with `-Werror` and an irgen pass for (a) and (b), and record the decision;
  - (c) and (d) read the R2 order and R3 placement;
  - write one refusal fixture per kind, and confirm no passing lifter fixture is refused.
- Regenerate. Record function-symbol moves, `source_root_moves`, path moves and the package renames in the golden configuration. Confirm by the reader's table projection that tables, IDs, type names, answers, lint findings and Contracts are unchanged.

### Investigation targets
**Required:**
- `.plans/QUINT_MODULE_LAYOUT.md` sections 2-6
- `model/temporal/features/standaloneactivity/*.scala`, `admission/*.scala`, `compositions/*.scala`
- `model/irgen/Context.scala:225-335` (Definition IDs, pins, type names)
- `tools/umpire/internal/golden/config.json` (projection, substitutions)
**Optional:**
- `model/README.md:500-560` (today's layout section)

### Quick commands
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks && make lint-model
go test -count=1 -tags test_dep -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower
make umpire-check-cases && make umpire-check-fixtures && make canary-check-case
```

### Execution constraints
- Stop if R5 needs a delta beyond positions, root strings and function symbols. Report it before task 2.

## Acceptance
- [ ] `standaloneactivity`, `record/` and `withTaskQueue/` each hold one feature file in R2's order, plus `Realization.scala` and tests. No `Model.scala`, `Properties.scala`, `Queries.scala`, `Capabilities.scala` or `IrFiles.scala` remains there.
- [ ] Definition IDs, IR type names, machine/Property/Scenario/Query/Limits/law names, tables, answers, lint findings and Contracts are unchanged. The IR and Case diffs hold only R5's deltas, recorded in the golden configuration.
- [ ] The declaration-order lint runs in `make umpire-check-model` and refuses each R4 (a)-(d) kind at its line, with one fixture each. It passes on all Models and on every passing lifter fixture. The choice of checker for (a)/(b) is recorded.
- [ ] A fixture proves that two owners may pin one former owner.
- [ ] The R11 baseline is recorded under `.flow/tmp/fn-126/`.
- [ ] The model gate, `make lint-model`, the original-baseline check, the Umpire Go tests, the Case/fixture/canary checks and `make lint-code-fast` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
