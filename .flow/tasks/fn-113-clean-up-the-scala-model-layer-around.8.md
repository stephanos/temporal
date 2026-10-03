---
satisfies: [R15, R19, R24, R25]
---
# fn-113-clean-up-the-scala-model-layer-around.8 Retire the native evaluator from model/umpire

## Description
Retire the native evaluator from model/umpire. Implements R15, the code half of R19 for the framework, and R25 for what stays generic; moves `model/umpire` toward R24's 1,300 lines.

**Size:** M
**Files:** model/umpire/Table.scala and Search.scala (deleted); Refine.scala, Compose.scala, Machine.scala, Claims.scala, Domain.scala, Errors.scala, Channel.scala, Monitor.scala, Assume.scala (runtime halves removed); model/umpire/test/Declarations.test.scala, model/temporal/test/NexusCallerPins.test.scala, model/temporal/test/NexusKernel.test.scala (deleted per the audit), model/temporal/test/StandaloneActivityPins.test.scala (evaluator tests removed; the step-function test stays)
**Touches:** [model/umpire/**, model/umpire/test/**, model/temporal/test/**]

### Approach
- The rule: a declaration stays when the lifter reads it by name or when its type makes a wrong Model fail to compile; code that builds a table, searches, checks a refinement, composes machines, answers a Query, or spells a key or a Definition ID at runtime goes. Names the lifter reads (keep name and type, R15's error clause): `umpire.Step` (`Context.scala:83`), `action`, `machine`, `starts`, `ends`, `evidence`, `refines`, `visible`, `steps`, `channel`, `monitor`, `assume` (`Declarations.scala:38,126-183,283,423,454`), `query`, `Limits` with `steps`/`actions`/`search` (`Claims.scala:93-99,207-218`), and the types `umpire.Action`, `Channel`, `Class`, `Composition`, `Finite`, `Hole`, `Inbox`, `Limits`, `Machine`, `Progress`, `Query`, `Reads`, `Step`, `realize.*`; `Verdict` if the IR's expected verdicts come through it.
- Goes: `Table`, `Row`, `RowResult`, `Atom`, `ClaimEntry`, `Family.id` if only tables used it, `Search`, `Keys` (and `Keyed`: no Model or fixture uses it), `Model.table` and `Machine.build/bind/enumerate/run`, `Composition.build` and its helpers, `Refinement`, `Refinement.of`, `productStep`, `RefinementRow`, `Query.answer`, `Answer`, `Trace`, `TraceStep`, `check`/`checkQuery`/`checkModel`, and `Errors.scala`'s `Checked`/`Fails`/`checked`/`fail`/`ModelError` once nothing calls them.
- `Finite` keeps `derived` (the compile-time rejection of a non-finite field, `Domain.scala:67-79`), `upTo` (the lifter reads the bound from the `given` block, `Types.scala:167-184`) and `of`; it may stop carrying `values` and `product`. `Channel.contents` keeps its name and type `Finite[Inbox[M]]` (the lifter reads it, `Types.scala:65-70`).
- Step functions stay executable (`Step`, the typed `~>`), so `StandaloneActivityPins` "one lost admission response..." stays as a native test; the other tests go exactly as the audit covered them; a test whose audit outcome is missing blocks the task.
- The Models do not change (no Model reads `Keys`, `table` or `answer` outside tests; checked at planning), so the lifted IR must be byte-identical: run the check-mode gate, `git status --short model/ir model/cases model/lifter/testdata` stays empty, the goldens pass strictly.
- Comments: a comment explaining a rule stays; one whose code goes, goes; the search-order and key-spelling comments that named the Go consumer go with their code.
- R25: record the library weighed for what remains generic (`Finite.derived`, about 40 lines, against shapeless-3 or magnolia; Iron for integer bounds) with line counts both ways; the spec expects none to pay for its wiring.
- Report the line count of `model/umpire` (non-test) in the summary; if it is above 1,300, name what stayed and why rather than cutting a compile-time guarantee (R24).

### Investigation targets
**Required**:
- `model/umpire/Machine.scala:9-17,131-271`, `model/umpire/Claims.scala:30-50,150-257`, `model/umpire/Refine.scala:4-30,49-163`, `model/umpire/Compose.scala:19-60,86-242`, `model/umpire/Domain.scala:22-128`, `model/umpire/Errors.scala:1-33`, `model/umpire/Table.scala`, `model/umpire/Search.scala`, `model/umpire/Channel.scala`, `model/umpire/Monitor.scala`, `model/umpire/Assume.scala`
- `model/lifter/Context.scala:83`, `model/lifter/Declarations.scala:38,126-183,283,423,454`, `model/lifter/Claims.scala:93-99,207-218`, `model/lifter/Types.scala:65-70,167-184`, `model/lifter/Realizations.scala:19-22` (`namedByIR`)
- `model/umpire/test/Declarations.test.scala`, `model/temporal/test/*.scala`, `.plans/umpire-scala-evaluator-audit.md`
- `.flow/tmp/fn115-7-summary.md` (borderline kept list)

### Quick commands
mise exec -- scala-cli compile model/project.scala model/umpire; mise exec -- scala-cli test model/project.scala model/umpire model/temporal; CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (check mode; IR must not change); git status --short model/ir model/cases model/lifter/testdata; CC=/usr/bin/clang mise exec -- go test -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/model ./tools/umpire/lower; mise exec -- make lint-model; find model/umpire -name '*.scala' -not -path '*/test/*' | xargs wc -l

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] `Table`, `Search`, `Keys`, `Model.table`, `Query.answer`, `check` and the runtime halves of `Refine`, `Compose` and `Errors` are gone; nothing in `model/umpire` builds a table, answers a Query, checks a refinement, composes machines or spells a key or a Definition ID at runtime; the DSL compiles alone.
- [ ] Every declaration the lifter reads keeps its name and type: the check-mode gate lifts byte-identical IR and expected fixtures, and the goldens pass strictly.
- [ ] The munit tests removed are exactly those the audit covered; the remaining native tests call step functions only.
- [ ] The summary records the line count of `model/umpire` (non-test), each borderline declaration kept and why, and the R25 weighing; `lint-model` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
