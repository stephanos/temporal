---
satisfies: [R18, R19]
---
# fn-113-clean-up-the-scala-model-layer-around.13 Fold the Nexus kernel into ordinary Scala and remove the Stainless residue

## Description
Fold the Nexus kernel into ordinary Scala and remove the Stainless residue. Implements R18 and R19 (Part D).

**Size:** M
**Files:** model/temporal/nexuscaller/kernel/Nexus.scala (moved to model/temporal/nexuscaller/Nexus.scala, package `temporal.nexuscaller`), kernel/NexusActions.scala (deleted), model/umpire/prelude/Prelude.scala (deleted), model/lifter/Expressions.scala (the `umpire.prelude` case and one comment), model/temporal/nexuscaller/Model.scala, Control.scala, Realization.scala, model/temporal/standaloneactivity/Model.scala (one comment), model/temporal/test/NexusKernel.test.scala (deleted if still present), model/specimens/README.md (kernel path citations), model/ir/nexus-caller.json and nexus-control.json (regenerated), tools/umpire/internal/golden/config.json (closed rename entries), optionally tools/umpire/model/isolation_test.go (vocabulary word)
**Touches:** [model/temporal/nexuscaller/**, model/umpire/prelude/**, model/lifter/Expressions.scala, model/temporal/standaloneactivity/Model.scala (comment), model/temporal/test/NexusKernel.test.scala, model/specimens/README.md, model/ir/**, tools/umpire/internal/golden/config.json, tools/umpire/model/isolation_test.go]

### Approach
- Step functions: `step(o, s, f)` becomes `Step(o, s, f)`, `one(x)` becomes `List(x)`, `none`/`facts0` become `Nil`, `facts1(a)` becomes `List(a)`; the `Steps`/`Facts` aliases become `List`. The lifter lifts both forms to the same IR (`Expressions.scala:195-215`), so the IR changes only in function names (`temporal.nexuscaller.kernel.X` to `temporal.nexuscaller.X`), positions and the file. `Control.scala:4,12` uses the prelude too; convert it.
- Contracts: Go evaluates `Function.requires` (`tools/umpire/model/eval.go:235`, admitted in `validate.go:160`), so `require(...)` preconditions stay (they are IR, not residue); `.ensuring(...)` postconditions exist only for the lemmas and go (four contract sites in `Nexus.scala`). The lifter's `stripContracts` stays for `require`; its comment loses the Stainless sentence.
- Domains: `derives Finite` where no Int bound is involved (the `given Finite[X] = Finite.derived` lines in `Model.scala:60-69` go); `ProtocolState` keeps the `given Finite[ProtocolState] = { given Finite[Int] = Finite.upTo(Protocol.attemptBound); Finite.derived }` block, because the lifter reads the bound from that form (`Types.scala:167-184`); say so in a comment that names the lifter.
- Delete `Actions` (`NexusActions.scala`) and `NexusKernel.test.scala` (they exist for the lemmas; task 8 may already have removed the test). Remove the `export kernel.{...}` and `import kernel.Protocol.terminalPhase` from `Model.scala`; `Realization.scala:478` loses the `kernel.` prefix. The lifter loses the `umpire.prelude` case (`Expressions.scala:195-202`).
- Comments: Stainless references at `kernel/Nexus.scala:1-9`, `NexusActions.scala:1-6`, `Model.scala:7,56-57`, `standaloneactivity/Model.scala:12`, `Prelude.scala:1-4`, `lifter/Expressions.scala:95-98,195`; a comment that explains a rule keeps the rule and loses the citation, one that only cites is deleted. `model/specimens/README.md:79-80,181` cite kernel paths: update the paths minimally. Afterwards `grep -rni stainless model/` is empty.
- R19's error clause: a rule Go reads (the Int bound, `requires`) stays with a comment naming the Go consumer.
- Optional, outside `model/`: add `stainless` to `retiredFrontEnd` in `tools/umpire/model/isolation_test.go:41-42` with a table case, so the gate holds R19; record whether done.
- Regenerate the IR (`umpire-gen-model`); add the function-name substitutions and the moved file's path substitution to the golden config as closed entries (task 5's mechanism); the goldens pass under the projection; Case bytes, functional fixtures and the canary Case are unchanged (`model/cases` carry no positions or function names; verified at planning). `model/gate/Roots.scala` names no kernel symbol (verified at planning); confirm after the move.

### Investigation targets
**Required**:
- `model/temporal/nexuscaller/kernel/Nexus.scala:1-30` and its four `require`/`ensuring` sites
- `model/temporal/nexuscaller/kernel/NexusActions.scala:1-72`
- `model/umpire/prelude/Prelude.scala:1-15`
- `model/temporal/nexuscaller/Model.scala:1-80`, `Control.scala:1-15`, `Realization.scala:478`
- `model/lifter/Expressions.scala:95-100,195-215`, `model/lifter/Types.scala:167-184`
- `tools/umpire/model/eval.go:235`, `tools/umpire/model/validate.go:160`
- `tools/umpire/internal/golden/config.json` (after task 5)
- `model/specimens/README.md:79-80,181`, `tools/umpire/model/isolation_test.go:37-55`
- `model/gate/Roots.scala:10-24`

### Quick commands
mise exec -- scala-cli test model/project.scala model/umpire model/temporal; CC=/usr/bin/clang mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks; git diff --stat model/ir; CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks; CC=/usr/bin/clang mise exec -- go test -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/model ./tools/umpire/lower; make umpire-check-cases umpire-check-fixtures canary-check-case; mise exec -- make lint-model; grep -rni stainless model/

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] No `kernel` package, no `umpire.prelude`, no `NexusActions.scala`, no `NexusKernel.test.scala`; the Nexus caller's step functions use `Step`, `List` and `Nil` like every other Model, its domains use `derives Finite` except where an Int bound keeps the `given` block the lifter reads, and the lifter has no `umpire.prelude` case.
- [ ] `grep -rni stainless model/` is empty and no comment under `model/` cites a proof file, line or command; `require` preconditions and the Int bound stay with a comment naming the Go consumer; `.ensuring` postconditions are gone.
- [ ] The IR is regenerated and differs only in function names, positions and the moved file; the goldens pass under the projection with the closed rename entries; `umpire-check-cases`, `umpire-check-fixtures`, `canary-check-case` and `lint-model` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
