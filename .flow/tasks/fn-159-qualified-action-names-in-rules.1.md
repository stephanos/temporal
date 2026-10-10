---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R12]
---
# fn-159-qualified-action-names-in-rules.1 Require qualified on targets; from(X) groups without importing; rename the bound worker to activityWorker

## Description
Require every `on` target in a machine's `rules` section to name its action through its declarer (`on(worker.poll)`). `from(X)` stays as a grouping header with its ownership check, but no longer imports `X`'s members. Enforce this in the IR generator's lifter and in the framework's rule builder, then migrate every model, test and doc to the qualified form. In the same pass, rename the standalone activity's bound worker object from `worker` to `activityWorker`, so its rules read `on(activityWorker.poll)` and its signature no longer needs the full path to reach the kind's `worker`.

**Size:** L
**Files:** `model/irgen/Declarations.scala`, `model/framework/Syntax.scala`, `model/README.md`, `model/framework/Rules.test.scala`, `model/irgen/testdata/lifts/{Grouped,Rejects,RuleRejects}.scala` and their tests, every standalone activity model and test file that names the bound `worker` object (about 260 references in some 28 files under `model/temporal/features/activity/standalone/`, plus `model/temporal/capabilities/Pausable.test.scala`)
**Touches:** [model/irgen/Declarations.scala, model/irgen/*.test.scala, model/irgen/testdata/lifts/*.scala, model/framework/Syntax.scala, model/framework/Rules.test.scala, model/README.md, model/temporal/features/activity/standalone/**/*.scala, model/temporal/capabilities/Pausable.test.scala]

### Approach
- **irgen:** `fromBlock` (`Declarations.scala`, around line 625) currently *requires* `import <declarer>.*` as a `from`'s first statement. Invert this so an import of the declarer's members is refused, with a message that says to qualify the actions instead. In the `on` handling (around lines 540-580), require each target's action reference to be a `Select` on its declaring object, not a bare `Ident`, for the whole-action, class (`respondFailed(Failure.fatal)`: check the function part, not its arguments) and two-to-four-target forms. Apply this both inside and outside `from`. Keep the existing `declaredBy` ownership check.
- **framework:** `Rules.block` (`Syntax.scala`, around line 505) gets each target's source text through `codeOf`. Refuse a target whose action reference isn't qualified. Take care with class forms, whose arguments contain dots of their own: check the text before the first `(`. Keep the `within` ownership check. Update the `from` doc comment (around line 592), which currently documents `import clerk.*` and bare names.
- **Rename the bound worker (owner's choice, folded in here so each rule line changes once):** in the standalone signature, rename `object worker` to `object activityWorker`, with a comment saying it holds the activity kind's worker actions bound to this form's `activity`. Its body then reads `val poll = worker.poll.on(activity)`: `worker` now resolves to the kind's object, so the full `temporal.features.activity.worker` path goes. Update every reference to the bound object in the standalone product, system, derived models, realizations and tests (`activityWorker.poll`, `from(activityWorker)`), and the comments that call it "the worker". Don't confuse it with the worker process (`process`) or the kind's `worker` object. The IR is unaffected, because action ids come from the kind's declaration (`temporal.features.activity.worker.poll`). Confirm this with the IR check.
- **Docs:** update the README rules spec, plus every doc comment and error message that shows `from(X) { import X.*; on(a) ... }` or a bare `on(...)`, to the qualified form.
- **Migrate:** the 4 `from` blocks in the standalone product machine and the 6 in the system machine, plus the `from` fixtures in `framework/Rules.test.scala` and in the irgen lift testdata. Drop each `import X.*` and qualify each `on` target.
- **Tests:** in the irgen testdata, add refusal fixtures for a bare name inside `from`, a bare name outside `from` (reached through an import), and an import inside `from`. Keep accepted fixtures for the qualified whole-action, class and multi-target forms. Add matching framework-level tests in `Rules.test.scala`.
- **IR stays the same:** the change is purely syntactic. `make umpire-check-model` must pass with no checked-in IR file regenerated differently.

### Investigation targets
**Required:**
- `model/irgen/Declarations.scala`: the rules-section lift (`from`/`on` handling, `fromBlock`, `declaredBy`)
- `model/framework/Syntax.scala`: `Rules.block`, `on` overloads, `each`, `from`, `codeOf`/`writtenAction`
- `model/README.md`: the rules spec section
**Optional:**
- `model/irgen/testdata/lifts/RuleRejects.scala`, the existing refusal fixtures and how their expected messages are asserted

### Open question
- Whether `disabled(...)` targets should follow the same rule (spec Parked unknowns). Leave them unenforced unless the user decides otherwise.

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): land the Scala changes now. Checked-in IR, Cases and Go consumers may go stale. Verify with the Scala build, framework and model tests, and fn-160.5's step-table comparison under this task's rename map. Declare the IR change this task will cause; the IR is regenerated once, in Batch 5b, whose shared close supplies the gate evidence for the `make umpire-check-*` criteria. The framework's runtime check for `on` works immediately; the lifter rule and its fixtures run once the lifter builds again (fn-147).

### Quick commands
```sh
make umpire-check-model
make umpire-check-lint
```
## Acceptance
- [ ] The IR generator refuses a bare `on` target inside or outside `from`, naming the action and its position and spelling out the qualified form; class-form arguments are not treated as action references.
- [ ] The framework's rule builder refuses the same bare targets when the machine initializes.
- [ ] `from(X)` still refuses an `on` whose action `X` does not declare, and refuses `import X.*` inside it.
- [ ] Every `from` block and `on` target in the temporal models, framework tests and irgen testdata uses the qualified form; checked-in IR files are unchanged.
- [ ] The standalone activity's bound worker object is `activityWorker`, its signature reaches the kind's actions as `worker.<action>` without the full path, and no reference to the old bound `worker` name remains; the IR is unchanged by the rename.
- [ ] The README rules spec and framework doc comments show only the qualified form.
- [ ] Refusal and acceptance fixtures exist for each form above; `make umpire-check-model` and `make umpire-check-lint` pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
