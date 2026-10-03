---
satisfies: [R26]
---
# fn-113-clean-up-the-scala-model-layer-around.14 Give placeholder lambda parameters stable names

## Description
Give placeholder lambda parameters stable names. Implements R26 on the ported lifter.

**Size:** S
**Files:** model/lifter/Expressions.scala (lambda and function parameter lifting), possibly model/lifter/Context.scala; model/ir/*.json and model/lifter/testdata/lifts/expected/*.json (regenerated: only parameter names and their references change)
**Touches:** [model/lifter/Expressions.scala, model/lifter/Context.scala, model/ir/**, model/lifter/testdata/lifts/expected/**]

### Approach
- At planning, `_$1` occurs in 11 files (22 lines by grep; the spec counted 47 occurrences): the six IR files and five expected fixtures. A placeholder lambda (`_.facts.contains(x)`, `_ => true`) lifts its `$anonfun` parameter with the compiler's name (`Expressions.scala:340-347` after task 3's port; `function` at 88-91 for named functions).
- When a parameter's name is compiler-synthesized (`_$N`, or the symbol's synthetic flag), give it a stable readable name: derived from the parameter type's simple name in lower case (`state`, `step`, `fact`) or a fixed `it`; disambiguate against names in scope (enclosing lambda and function parameters, local vals in `env`) with a numeric suffix. References are lifted by symbol, so a name map suffices; the lambda's body text is otherwise unchanged. Record the rule in a comment.
- Prove: after `umpire-gen-model`, `git diff` of `model/ir` and `expected/` shows changes only in `params[].name` and the matching `var` references; `grep -rn '_\$' model/ir model/lifter/testdata` is empty; the lifter's identity test ("a declaration's identity does not move with its line") and refusals are unchanged (R8).
- R13: the goldens pass through task 5's parameter alpha-normalization; Case bytes carry no parameter names (verified at planning), so `umpire-check-cases`, `umpire-check-fixtures` and `canary-check-case` report no change.

### Investigation targets
**Required**:
- `model/lifter/Expressions.scala:85-100,340-350` (after task 3)
- `model/lifter/Context.scala` (`defs`, `lifting`, parameter environments)
- `model/ir/activity.json:1823,1858` (a `_$1` binder and its reference)
- `tools/umpire/internal/golden/golden.go` (task 5's alpha-normalization)
- `.flow/tmp/fn113-5-summary.md`

### Quick commands
mise exec -- scala-cli compile model/lifter; CC=/usr/bin/clang mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks; git diff model/ir model/lifter/testdata/lifts/expected | grep '^[-+] ' | grep -v 'name\|var' (expected empty); CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks; CC=/usr/bin/clang mise exec -- go test -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/model ./tools/umpire/lower; make umpire-check-cases umpire-check-fixtures canary-check-case; mise exec -- make lint-model

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] No compiler-synthesized name remains in `model/ir` or the expected fixture outputs; placeholder lambda parameters carry a stable, readable name, disambiguated against names in scope, and the rule is recorded in the lifter.
- [ ] The regenerated IR and fixtures differ only in the renamed parameters and their references; refusals and the identity test are unchanged; the goldens, `umpire-check-cases`, `umpire-check-fixtures`, `canary-check-case` and `lint-model` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
