---
satisfies: [R4, R5, R14]
---
# fn-133-lean-typed-realizations.3 One instruction convention, own names, named evidence arguments, ids from facts

## Description
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
Part B, R4, R5, R14 (ids).

- **Lower-case instruction forms.** Feature files write only `fault`, `hold`, `release`, `finish` and `command`. The upper-case case classes stay core forms. Lint: an upper-case instruction form in a feature file.
- **No borrowed names.** `Command("release-dispatch", …)` (activity lost race) and `Command("start-nexus-operation", …)` (Nexus caller schedule) are the two today. Decide between evidence naming both commands and an explicit kit alias that the evidence line shows, and record the decision in the spec. The lifter refuses two commands of one script with one name unless the alias declares it.
- **`withFields` matching documented.** The kit and README say that evidence naming a call matches all its `withFields` variants.
- **Named arguments.** `delivered(…, attempt = 1, after = startActivity, …)` and `answeredAs`. No evidence kind is a free string; ids come from facts or declared constants (`evidenceId("scheduled")`, `sourceId("history")` go).

## Acceptance
- [ ] The lint finds no upper-case instruction form in a feature file, and no feature file builds a `Command` with an explicit id outside the alias.
- [ ] The alias decision is recorded in the spec's Decision Context.
- [ ] No feature file passes a string literal to `evidenceId`/`sourceId`/`answeredAs`.
- [ ] A before/after projection is identical apart from command ids the alias decision changes, which are listed.
- [ ] The spec's Verification gates pass.

## Done summary
Feature files now use lower-case instruction forms only. A command takes another's name only through `aliasOf`. `delivered` and `answeredAs` are called with named arguments, and no feature file passes a string literal to `evidenceId`, `sourceId` or `answeredAs`. A scratch lift of model/ir shows every `realizations` section equal to the baseline with positions stripped: the alias keeps both shared names.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: IMPLEMENTER claude-opus-5-5 at high

### What changed
- `model/temporal/realize/Kit.scala` (appended at the end, so no earlier kit line moved) adds the helpers `fault`, `hold`, `release`, `awaitLearned(learned)`, `awaitCommand(command)`, `attemptFailure(failure)`, `attemptCanceled`, `workflowCommand`, `nexusReply` and `nexusCompletion`. `finish` and `command` already existed.
- `delivered` renames its parameters `number` and `call` to `attempt` and `after`.
- Core widening:
  - `Instruction.AwaitCommand(command: String | Command | Instruction)` and `AwaitLearned(learned: String | Learned)` (model/umpire/realize/Realize.scala).
  - `NexusReply(binds: String | Learned)` and `NexusCompletion(handle: String | Learned)` (model/temporal/realize/Realize.scala, same lines).
  - A value is lifted as its id or name, as before.
- `aliasOf(target)(instruction)` in umpire/realize/Scripts.scala: a command named as `target` is.
- `withFields`'s doc, and the README, now state that evidence naming a call matches every `withFields` variant of it.
- Lifter (Realizations.scala):
  - `lowerCaseForm` refuses an Instruction case class written in a file under a `features` directory.
  - A written-out `Command(id, …)` in such a file is refused.
  - `commandOrigins` refuses two commands of one realization with one name, unless one is a `withFields` variant of the other or aliases it with `aliasOf`.
  - `commandName` handles `aliasOf`, including through a helper's call.
- Feature files:
  - activity: `loseAdmissionResponse = aliasOf(releaseDispatch)(fault(…))`; the local `attemptFailure` def is renamed `failed(retryable)`.
  - nexus caller: the scheduling variants are `aliasOf(startNexusOperation)(schedule(…))`; `awaitCommand(startNexusOperation)` and `awaitLearned(completionAuthority)` take values.
  - The evidence kinds are the declared constants `scheduledAgain`, `scheduledKind` and `describeSource`.
- README: the realization bullet now describes `extended`, `aliasOf`, `withFields` matching, the lower-case rule and the evidence modules; the sugar bullet describes `proto`, `read` and message literals for request fields. This covers fn-133.1–.3.

### Decisions (owner unavailable; please record in the spec's Decision Context, R4 alias)
- **Explicit alias rather than evidence naming both commands.** Renaming `release-dispatch`/`start-nexus-operation` would change the command ids that existing generated Cases carry (tests/testcore/testpilot/testdata/generated/nexus-workflow-*.json; Go lowering tests), and the edge case "a shared race command stays one command" forbids splitting them. `aliasOf(releaseDispatch)(fault(…))` keeps one id and makes the sharing visible at the declaration that borrows the name.
- **What counts as a feature file for the lint:** a file under a directory named `features`. This covers model/temporal/features and lets the refusal fixture live at model/irgen/testdata/features.
- **The lint is a lifter refusal,** not an umpire-lint finding.
- **Kinds that differ from their fact are declared constants (vals), as R5 allows.** IDs are unchanged.

### Declared IR delta (batch regeneration)
- Source positions only. No command ids change, because the alias keeps both shared names. No Cases are added, and no carrier metadata changes.
- Lifter fixture `rejects.txt` gains one line (ScriptRejects.scala:168, `namedTwice`). The new fixture dir `model/irgen/testdata/features` has no expected file; its test asserts the refusals inline.

### Tests
- `mise exec -- scala-cli test model/irgen`: 96 passed, 0 failed. New: "a feature file's upper-case instruction and written-out command are refused" (Features.scala:20 and :25), and the refusal `namedTwice`.
- `--check-syntax` and `--check-comments`: clean. scalafmt check: clean.
- A scratch lift of model IR: every `realizations` section is equal with positions stripped.

### Line counts
After .3: activity 299, nexus workflow 361, nexus standalone 88.

### For later tasks
- An inline helper call that returns `aliasOf(...)` is named by its target, and its origin is the target.
- The class-pattern rule, read for fn-133.4: the lifter's `named` lowers `start(scheduleToStart := expires)` to the positional class with every omitted input at its domain's first value (`unset`), and Go compares class keys (action plus every input value, check/claims.go `classKey`, interp.ClassKey). So a class pattern is **exact**.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ca2a888acc
- Tests: mise exec -- scala-cli test model/irgen, scala-cli run model/check -- --check-syntax, scala-cli run model/check -- --check-comments, scala-cli fmt --check model sources, scratch lift --ir of model IR, realizations equal with positions stripped
- PRs: