# Simplify the DSL's words

## Goal

Rename the DSL words that collide with Temporal's vocabulary or with other DSL words, and give composition law parameters one word in place of hand-written forwarding objects. The Models must mean exactly what they mean today. Owner decisions of 2026-10-05 on the DSL-simplification study (`.plans/DSL_SIMPLIFICATION.md`, ranks 1 and 6).

There is one exception: the study's rank 2 guard sugar is not here. It became the rule headings of fn-126 (`when(g) { action ~> effects.x }`, `in(phases) { … }`). There `when` is a rule heading and nothing else, so one design holds across both specs.

## Why

- **`accept(...)` and `given Accepted[O]`** read as Temporal's own "accepted": Update accepted (`WorkflowExecutionUpdateAccepted`) and Nexus operation accepted. They are used 101 times in the Models (`model/umpire/Syntax.scala:14-27`).
- **`poll(...) { }`** in realizations (`model/umpire/realize/Scripts.scala:89`, `Realize.scala:480`) reads as the worker's long poll (`PollActivityTaskQueue`), which the Models also model.
- **`.setting { field := … }`** (`Scripts.scala:104`) collides for certain with fn-125's `setting[T]` (dynamic configuration) as soon as fn-125 resumes.
- **`always(command)`** (`Scripts.scala:48`) takes the temporal operator `.plans/DSL_OPERATORS.md` reserves (`always`/`eventually`, Do-not-do 7).
- **Two `Outcome`s.** `umpire.realize.Outcome` (satisfied/violated) shares its name with each Model's `enum Outcome`. `…/admission/Queries.scala:6` already imports it renamed (`Outcome as RunOutcome`).
- **Forwarding objects.** Each composition restates the record's status sets through its `activity` member so its laws can read them (`standaloneactivity/compositions/Model.scala:31-35, 53-57`: 8 forwarding defs). Every future composition adds more.

## Requirements

- **R1 Words renamed.**

  | Today | After | Where |
  | --- | --- | --- |
  | `accept(state, facts*)` | `enter(state, facts*)` | framework sugar `model/umpire/Syntax.scala` |
  | `given Accepted[O] = Accepted(o)` | `given Ok[O] = Ok(o)` | same |
  | `poll(evidence, role)(until)` (both forms) | `readUntil(…)` | `model/umpire/realize/{Scripts,Realize}.scala`, the kit |
  | `.setting { field := … }` | `.withFields { … }` | `Scripts.scala` |
  | `always(command)` | `everyCase(command)` | `Scripts.scala` |
  | `umpire.realize.Outcome` | `umpire.realize.PropertyOutcome` | `Realize.scala:599` and its users |

  The renames cover every Model, lifter fixture and test that uses these words, and the lifter's name matches (`model/irgen/Syntax.scala`, `Realizations.scala`). Each sugar definition keeps its `Core form:` doc, and `umpire.check.SyntaxRule.sugarNames` lists the new names.

  **`Outcome.accepted` stays.** It is Model vocabulary, not a DSL word. It is an enum case whose name is in the IR type catalogs, the fingerprints and the Case bytes (`"definitionId":"accepted"` in `model/cases/activity-completion-case.json`). Renaming it would change Contracts. `given Ok[Outcome] = Ok(Outcome.accepted)` reads as "this machine's ok outcome is `accepted`".

  Errors:
  - the old words no longer exist, so a Model that uses one does not compile;
  - the IR, the Cases, the Contracts and the lint findings are byte-identical before and after (`make umpire-check-model --update` leaves `model/ir/**` and `model/cases/**` unchanged);
  - any difference stops the task.

- **R2 `through(selector)(predicate)` for composition law parameters.**
  - A capability field of a composition may read a member's status set as `through(_.activity)(Admission.paused)`. This is framework core in `model/umpire`, with no Temporal word.
  - The lifter folds the selector and the named predicate into one function that it lifts. Composition stays refused where a lambda is refused today (`model/README.md` 618-620). Only a field path selector and a reference to a named def are accepted.
  - The forwarding objects `OverQueue` and `OverMatching` in `compositions/` (their status-set defs) are deleted. Their state case classes stay.

  Errors:
  - refused at its line: a selector that is not a field path; a predicate that is not a named def of the lifted sources; a predicate over another state type (compiler);
  - one lifting fixture and one refusal fixture;
  - the law tables, verdicts, Query answers and Cases are identical;
  - the only IR deltas are the names and positions of the lifted functions that replace the forwarders, recorded as `function_name_substitutions` in `tools/umpire/internal/golden/config.json`.

- **R3 Docs.** The following describe the new words and name no old one:
  - `model/README.md` and `model/SEMANTICS.md`;
  - `.plans/DSL_OPERATORS.md`: one entry per renamed word with its reason, and `through` as a word with a guessable meaning, not a symbol;
  - `.plans/DSL_SIMPLIFICATION.md`: rank 1 and rank 6 marked done.

  Errors: a grep of the live docs and of `model/` (not archives, not `.flow/`) for `accept(`, `Accepted[`, `.setting {`, `always(` and realize's `poll(` finds nothing.

- **R4 Gates.** The model gate, `make lint-model`, the Umpire Go tests, `make umpire-check-cases`, `make umpire-check-fixtures`, `make canary-check-case` and `make lint-code-fast` pass at each task.

## Ordering

- **Entry:** fn-114, fn-118 and fn-122 are closed. fn-118.5 rewrites the waits in every `Realization.scala`; these renames follow it, so neither rebases the other.
- **Not concurrently with fn-124.8**, which forbids other edits to `model/` and `tools/umpire/model` while it runs.
- **Before fn-126 starts.** fn-126 task 1 waits for this spec to close. Both specs touch every Model file, so the order keeps fn-126's layout move from rebasing onto 101 renamed calls. fn-126's sketches already use `enter` and `Ok`.
- **Before fn-125 resumes.** `.withFields` frees `setting` for fn-125's `setting[T]`.
- **Before fn-124.7**, which retires the golden harness that R2 records into. If fn-124.7 lands first, R2 records its function deltas in a before/after projection of the reader's outputs under `.flow/tmp/`.
- **fn-124.3** also edits the realization surfaces. Whichever lands second rebases.
- **Inside the spec:** task 1, then task 2.

## Boundaries / Non-goals

- **No guard sugar.** Rank 2 (`when(g) { steps }`) is folded into fn-126 as rule headings. A step-level guard helper is not added, so `when` has one meaning.
- **No meaning change, no IR schema change, no Case change.** R2's recorded function names are the only IR delta.
- **Not renamed here:** `results("Delivery")` (the study proposed `resultDomain`; not approved) and `starts`/`ends` (replaced by fn-126's machine-object members `start` and `end`).
- **Later, not planned** (see `.plans/DSL_SIMPLIFICATION.md` section 6):
  - rank 7: a feature-local request helper (`activityCall(method)`) and `perform(… , then = await)`. Both need a realization fold in the lifter. Revisit after fn-118.5 and fn-124.3.
  - rank 8: the deadline step helper, `UpTo.succ` and enum status methods. Revisit only if still felt after fn-126.
- **Not recommended at all:** annotations, `inline` helpers, macro annotations, by-name bindings, named-tuple bundles and a shared protocol template (study section 2).

## Verification

```bash
make umpire-check-model && make lint-model && make lint-code-fast
go test -count=1 -tags test_dep -p 2 ./tools/umpire/...
make umpire-check-cases && make umpire-check-fixtures && make canary-check-case
```

Task 1: `git diff --stat model/ir model/cases` is empty after regeneration. Task 2: that diff holds only the recorded function names and positions.

## Requirement coverage

| Task | Requirements | Gate |
| --- | --- | --- |
| .1 rename the DSL words | R1, R3 (words), R4 | fn-114, fn-118, fn-122 closed; not concurrent with fn-124.8 |
| .2 `through` for composition law parameters; close | R2, R3, R4 | after .1; before fn-124.7 |
