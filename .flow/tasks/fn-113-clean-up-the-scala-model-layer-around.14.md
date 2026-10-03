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
# fn-113.14 handover: stable names for compiler-synthesized parameters (R26)

Nothing is staged or committed. Changed files are all in Touches: `model/lifter/Expressions.scala`
(+61), `model/lifter/Context.scala` (+3), 6 files in `model/ir/` and 5 in
`model/lifter/testdata/lifts/expected/`, all regenerated. The lifter grows from 1,942 to 2,006 lines.
`model/cases`, `rejects.txt` and the lifter tests are unchanged.

### The naming rule, and where it is recorded

The rule is in the docstring of `parameters` in `model/lifter/Expressions.scala`:

- It covers every Function and Lambda parameter whose compiler name contains `$`. Two kinds occur:
  `_$N` (placeholder `_`) and `x$1` (the parameter of a `{ case ... }` lambda, e.g. machine
  `evidence`).
- **Scope widened beyond `_$N`.** `x$1` is also a compiler-synthesized name, and R26 says "no
  compiler-synthesized name". It appeared 24 times (in nexus-caller, nexus-control, activity and
  admission.json), and is renamed too. After the change, no Param or `var` name in `model/ir` or
  `expected/` contains `$`.
- **The name comes from the parameter's type:** the simple name with its first letter in lower case
  (`step`, `state`, `admissionState`, `productFact`, `boolean`).
  - The type name is the written one: `widen.typeSymbol.name`, with no dealias.
  - It falls back to `it` when the type name is not a plain `[A-Za-z][A-Za-z0-9]*`, or when its
    lower-case form is a Scala keyword (e.g. `Type`).
- A lifted reference reads the name through `Context.nameOf(symbol)`, backed by a new
  `Context.renamed` map. References are lifted by symbol, so no body text changes.
- A parameter that already has a user-written name is never renamed.

### Disambiguation

`inScope(fn, body)` collects the names the new name must avoid:

- **Names around the parameter.** The names bound by the parameter's own function or lambda, and by
  every enclosing local DefDef, lambda or local val. It walks the owner chain while the owner is a
  DefDef or a local ValDef, and takes every ValDef or Bind owned by that chain.
- **Names in the body.** Every name bound in the body (vals, pattern binds, nested lambda
  parameters) and every local name the body reads. This covers both directions: the new name hides
  no other name, and no other name hides it.

A taken name gets the first free numeric suffix, starting from 2 (`disk2`, `disk3`). Parameters of a
single list are named left to right, and each name is added to the taken set.

No checked-in Model or fixture triggers a suffix today. I proved the disambiguation with a scratch
probe outside the repo:

- Inputs: the lifts fixture plus three monitors, packaged and lifted with the real lifter.
- Fixture diff: `.flow/tmp/fn113-14/probe-fixture.diff`. Result:
  `.flow/tmp/fn113-14/probe-disambiguation.log`.
- `(_, _, after)` where both `_` are `Disk` gives `disk, disk2`.
- `(disk, _, after)` gives `disk, disk2`.
- `(_, _, after) => { val disk = ...; ... }` gives `disk2, disk3`, so the local val is not hidden.

The Touches list does not include the fixture sources, so no fixture was added to the repository.
A permanent fixture would need `model/lifter/testdata/lifts/*.scala`. This is optional, and the
conductor decides.

### The proof that only renamed parameters and their references changed

- **Before state.** Copies of `model/ir`, `expected/` and `model/cases` taken before any edit are in
  `.flow/tmp/fn113-14/before/`.
- **Structural check, both trees.** The script `.flow/tmp/fn113-14/prove_renames.py` decodes before
  and after JSON and walks them in parallel. Every key and value must be equal, with these
  exceptions:
  - A Function or Lambda param whose old name contains `$` may change name. The new name must not
    contain `$`, and names within one list must be distinct.
  - A user-named param must keep its name.
  - Each `var` must equal the scope-resolved new name of its binder. Scopes are params, `let` and
    pattern `bind`.
  - Logs: `proof-ir.log` (rc 0: 27 parameters and 16 references renamed over the 6 IR files) and
    `proof-expected.log` (rc 0: 18 parameters and 12 references over 5 fixtures; `presence.json`
    unchanged). Both list every old-to-new pair.
  - Negative check: one `var` mutated to `stepX` makes the script fail (`proof-negative.log`, rc 1).
- **Text checks** (`text-checks.log`):
  - The `git diff` lines of `model/ir` and `expected/` that are not `"name":`/`"var":` lines: 0.
  - `grep -rn '_\$' model/ir model/lifter/testdata/lifts/expected`: empty.
  - No Param or var name containing `$`.
  - `model/cases` is identical to the copy taken before the change.
  - `rejects.txt` is unchanged.
- **R8.** `rejects.txt` and `model/lifter/test/` are unmodified. The identity test ("a declaration's
  identity does not move with its line") and every refusal test pass (`lifter-test.log`).
- **R25.** No library was weighed. The change is about 60 lines over the compiler's reflection API.

### Checks (logs under .flow/tmp/fn113-14/)

| Command | Result | Log |
| --- | --- | --- |
| `mise exec -- scala-cli run model/gate -- --generate-ir` (first run in this worktree) | ok | generate-ir.log |
| `mise exec -- scala-cli compile model/lifter` | ok | (console) |
| `mise exec -- make fmt-model` | rc 0 | fmt-model.log |
| `mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` | ok | gen-model.log |
| `mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` | ok | check-model.log |
| `mise exec -- scala-cli test model/lifter` | 14/14 passed, identity and refusal tests included | lifter-test.log |
| `mise exec -- make lint-model` | rc 0, "All files are formatted with scalafmt"; the existing fixture warnings are unchanged | lint-model.log |
| `make umpire-check-cases` | rc 0 | umpire-check-cases.log |
| `make umpire-check-fixtures` | rc 0 | umpire-check-fixtures.log |
| `make canary-check-case` | rc 0 | canary-check-case.log |
| `GOMEMLIMIT=4500MiB GOFLAGS=-p=1 go test -tags test_dep -count=1 -timeout 30m -parallel 1 -run 'Migration\|Golden' ./tools/umpire/model` | ok, 238 s | model-migration.log |
| the same command for `./tools/umpire/lower` | ok, 140 s | lower-migration.log |

Notes:

- The goldens needed no `config.json` entry: task 5's alpha-normalization covers Function and Lambda
  parameters.
- **Memory.** The first two runs of the model goldens were OOM-killed (`signal: killed`). Each time,
  another agent's `model.test` or `lower.test` was running at the same time. The kernel log shows
  the killed `model.test` at 8 GB anon RSS. The run passed alone with `GOMEMLIMIT=4500MiB -parallel 1`.
  These settings only bound GC and subtest parallelism; test semantics are unchanged. I recommend
  them to anyone running these packages in this sandbox.
- **Not run:** `make umpire-check-backends` (deferred) and the gate's Go step (`--skip-go-checks`).

### Shared-doc changes for the conductor

- Spec R26 counted 47 `_$1` occurrences. The measured count is 49 `_$N` name/var lines, plus 24 `x$1`
  lines. R26 also covers `x$1` now, since it is a compiler-synthesized name too. Consider rewording
  R26 to cover `{ case ... }` lambda parameters as well.
- MILESTONES.md: task 14 is done pending review. Exploration Case IDs of nexus-caller,
  nexus-control and realizations may change downstream, per the R13 amendment. No `model/cases` byte
  changed in this lift.
- `model/README.md` / `model/SEMANTICS.md` (not in Touches): optionally add one line saying that
  lifted parameter names are never compiler-made, and that a placeholder parameter is named after
  its type.

### Review (independent review: SHIP; conductor decisions and fixes applied)

Conductor decisions:

- Renaming `x$1` is accepted under R26.
- No permanent disambiguation fixture is added: it would change the frozen inputs of the lifter
  golden. The scratch probe stays as the evidence.

Fixes:

1. **Capture check in `prove_renames.py` (should-fix).** For each renamed parameter, the script now
   fails in three cases:
   - the new name is already bound by an enclosing binder (it would hide it);
   - the body before the rename reads a free `var` of that name (it would be captured);
   - a parameter, `let` or pattern `bind` inside the body binds that name (it would hide the
     parameter).

   Re-runs:
   - Both trees pass, with the same counts: `proof-ir.log` reports 27 parameters and 16 references,
     and `proof-expected.log` reports 18 and 12.
   - The earlier negative still fails (`proof-negative.log`, rc 1).
   - New capture negatives (`proof-negative-capture.log`, cases under `neg-capture/`):
     - `free` (`_$1` to the body's free `x`): rc 1.
     - `enclosing` (a lambda's `_$1` to the enclosing Function's `state`): rc 1.
   - `inner` (`_$1` to `state` under a `let state` in the body): rc 1.
   - A capture-free control passes (rc 0).
2. **`Flags.Synthetic` probed. The `$` test is kept.** I added a temporary print of each lifted
   parameter's flags, lifted the scratch probe (Declarations and Admission roots), then restored the
   file. Logs: `flags-probe-declarations.log` and `flags-probe-admission.log`.
   - `_$1`, `_$2`, `_$6` and `x$1` carry `Synthetic`.
   - `_$1` to `_$5` and `_$7` to `_$11` of the admission fixture's lambdas carry only `Param`.
   - The user-named `f` and `s` carry `Param | Synthetic`.

   So the flag both misses placeholders and marks user names. Detection stays on `$`, with a
   one-line comment that says why. Neither a backticked user name with `$` nor the IR is affected.
3. **Docstring.** The name is that of the outer type constructor (`tuple2`, `function1`, `list`),
   and `it` for a type with no plain name or whose lower-case name is a keyword.
4. **`inScope` docstring.** A lambda inlined from a top-level val avoids the names of its definition
   site, not of its use site. This is harmless, because its body cannot read the use site's locals.

Re-runs after the fixes:

| Command | Result | Log |
| --- | --- | --- |
| `mise exec -- scala-cli test model/lifter` | 14/14 passed | lifter-test.log |
| `mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` | ok, so `model/ir`, `expected/` and `model/cases` equal a fresh lift | check-model.log |
| `mise exec -- make lint-model` | rc 0 | lint-model.log |

Only comments changed, and the IR is byte-identical, so the goldens were not re-run. The lifter
total is still 2,006 lines: Expressions.scala +66/-5 and Context.scala +3.

**For the conductor.** R26 as implemented renames parameters only: Function and Lambda params and
their references. A compiler-synthesized local `val` or pattern-bind name would pass through
unchanged. None occurs today:
- Named-argument `x$N` vals are substituted away by `arguments`.
- No `let` or `bind` name in `model/ir` or `expected/` contains `$`.

If a future construct produces such a name, extend the rule or add a gate check.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2b46c7a0672cf5276a431d8ebe1bd562073bdc29
- Tests: mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks -> ok, mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks -> ok, mise exec -- scala-cli test model/lifter -> 14 passed (identity and refusal tests unchanged), mise exec -- make lint-model -> rc 0, all files formatted, python3 .flow/tmp/fn113-14/prove_renames.py before/ir model/ir -> equal modulo 27 renamed params, 16 refs, python3 .flow/tmp/fn113-14/prove_renames.py before/expected expected -> equal modulo 18 renamed params, 12 refs, grep -rn '_\$' model/ir model/lifter/testdata/lifts/expected -> empty, make umpire-check-cases -> rc 0, make umpire-check-fixtures -> rc 0, make canary-check-case -> rc 0, GOMEMLIMIT=4500MiB GOFLAGS=-p=1 go test -tags test_dep -count=1 -parallel 1 -run 'Migration|Golden' ./tools/umpire/model -> ok 238s, GOMEMLIMIT=4500MiB GOFLAGS=-p=1 go test -tags test_dep -count=1 -parallel 1 -run 'Migration|Golden' ./tools/umpire/lower -> ok 140s, prove_renames.py with capture check: model/ir rc 0 (27 params, 16 refs); expected rc 0 (18 params, 12 refs), prove_renames.py negatives: mutated var rc 1; capture free/enclosing/inner rc 1; control rc 0, Flags.Synthetic probe: does not mark all _$N and marks user names f/s -> kept '$' detection, post-review: mise exec -- scala-cli test model/lifter -> 14 passed, post-review: mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks -> ok (IR byte-identical), post-review: mise exec -- make lint-model -> rc 0, independent review (claude-opus-5-5, fresh context): SHIP; capture check added to the proof, docstrings completed, Synthetic flag probed and rejected
- PRs: