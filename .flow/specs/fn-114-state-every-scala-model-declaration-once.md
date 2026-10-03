# State every Scala Model declaration once

## Goal & Context
<!-- scope: business -->

A feature developer writes and reviews the Temporal Models. A review on 2026-10-01 found that the Models state many things twice, mostly as string literals.

- 76 of 93 named declarations repeat their `val` name as a string.
- The two realizations hold 46 private string constants that other strings refer to by spelling.
- `nexusProduct` lists six evidence lines that map each fact to its own name.
- The gate (`run.sh` on that date, `model/gate` now) lists 35 fully qualified Scala declarations to say what goes into which IR file, and starts one lifter JVM per IR file.

`fn-112-make-the-standalone-activity-scala` introduces the constructs that remove this repetition and applies them to the standalone activity. The early named-choice mechanism of fn-120 is available before this rollout. This spec applies the settled syntax to every other Model and fixture, removes the old forms from the framework so that one way to write a declaration remains, and moves the IR roots into Scala.

This spec was Part E of `fn-113-clean-up-the-scala-model-layer-around`. It is a spec of its own so that the specs run in a line: fn-115, fn-113, fn-117, fn-112, then this one. fn-115 has by then moved the model and replaced the shell scripts with one gate program, so where this spec says `model/temporal` or `model/umpire` it means the Models and the DSL at the places fn-115 gives them.

## Architecture & Data Models
<!-- scope: technical -->

No new mechanism beyond what fn-112 defines, with one exception. Today the gate decides which declarations each IR file holds, in lists kept apart from the Models. After this spec a Scala declaration decides it, and the lifter reads that declaration and writes every Model IR file in one run.

| Concern | Today | After |
| --- | --- | --- |
| Name of a declaration | string argument, usually equal to the `val` name | the `val` name, read by the lifter |
| References inside a realization | a string equal to another declaration's id | the declaration itself, by value |
| Evidence for a fact | one line per fact, including identity lines | only the exceptions |
| Contents of an IR file | root lists in the gate | one declaration per IR file, beside the Models |
| Lift | one JVM per IR file | one run writes every Model IR file |

The lifter fixtures keep their own lift step, because some of them must fail.

## API Contracts
<!-- scope: technical -->

The author surface is fn-112's. This spec adds no construct except the IR-file declaration, whose shape the first task settles and records here, and the monitor pattern `sticky`, which fn-112's claim patterns left to this spec because both of its uses are in the close policy (`.plans/TEMPORAL_PATTERNS.md` section 2.4). The lifter's `lift: <file>:<line>: <message>` refusal format and the IR schema stay as they are.

**The `sticky` monitor pattern**, beside `monitor` in `umpire`, a promise that once broken stays broken:

```scala
val retainedOutcome = sticky(outcomePreserved)             // same-step predicate; name from the val
val ownerAcknowledgment = stickyAcross(ackOnlyWhenKept)     // two-state predicate (before, after)
```

Each is sugar for the `monitor(…)(next)(violated)` lambda, so by fn-112's core/sugar rule it lives in `model/umpire/Syntax.scala` beside the claim patterns, documented with the `monitor` form it stands for, with its matching in `model/lifter/Syntax.scala` and a fixture proving IR equality with that form; `Monitor.scala` imports nothing from it. Each is a plain `def` the lifter matches by name and lowers to the existing `Monitor` record with a Boolean state, `initial = false`, `next = or(m, not(call(p, after)))` (or `call(p, before, after)`), `violated` the identity and the verdict read after every step: the tree the two hand-written lambdas lift to today, so the monitor entries of `nexus-close.json`, the monitor-agreement exports and `MonitorExpectation` names are byte-identical. A predicate of another state type is a compiler refusal. Monitors that need history (`singleOutcome`, `cancelPrincipal`, `terminalFinality`, `atMostOneActiveAttempt`) stay hand-written.

**Lifted meaning is frozen.** For every IR file, the Go reader derives the same tables, Definition IDs, refinement rows, fingerprints and Query answers before and after each task, and the lowering produces the same Case bytes. fn-112 R1's function projection applies: function bodies, the `functions` inventory and function references are outside the IR text check and are proven on those derived outputs, so a conversion that changes how a step function or Property is written is allowed exactly when every derived output and Case byte is exact. The rollout introduces no sugar of its own: every convenience form it uses is one fn-112 placed in a `Syntax.scala`, and `sticky` (R12) follows the same rule. A name taken from a `val` equals the string the declaration wrote before. Existing branch alternatives use fn-120's settled named-choice syntax during this conversion; only their inert IR choice-name metadata may be added. The exact allowed metadata deltas inherited from fn-112 remain narrow. They are: inert choice names; source file paths, lines and the IR `source` root string of a declaration whose file moves; and the symbol of a top-level function whose file moves (the lifter names it after its file's `X$package$` object), each recorded as an entry in the migration golden configuration. No new branch, reordered result, or changed Case is authorized.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Names that differ from their `val`.** 17 declarations do today, for example `storageLossAssumed` named `storageLoss`. Each one either renames its `val` to the IR name or states the name through the one explicit form fn-112 defines. Neither changes the IR name.
- **Computed names.** A Query whose name is computed from its machine's name keeps its spelling.
- **Realization ids the IR needs as text.** An id such as `temporal.nexus.caller.evidence.started` is written once, on the declaration it identifies. Every other mention is a reference to that declaration.
- **Removing an old form.** A string-named form leaves `model/umpire` only when no Model, fixture or test uses it. The lifter's case for it is removed in the same task, and a fixture proves the lifter now refuses it or that it no longer compiles.
- **Fixtures that must not compile.** The lifter's own tests name a fixture file and a line. They stay, and are the only place outside the Models that names anything inside a Scala source.
- **Gates.** Each task runs the scoped parts of the model gate, `make lint-model` and the Go tests of the Umpire tooling. The closing task runs all three in full and `make lint-code-fast`.
- **Order.** The fn-120 choice mechanism lands before model conversion. This spec finishes all consumer migration, including fixtures, before fn-120 refuses unnamed branching. Its Scala-owned root inventory is stable before fn-120 snapshots lint findings. No fn-120 full-spec dependency blocks this rollout.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The golden set of fn-115 R2 (`tools/umpire/model/testdata/migration`, `tools/umpire/lower/testdata/migration`) passes before and after every task. Errors: a task that needs a difference states it, amends the allowed list in this spec, and gets the owner's agreement before landing.
- **R2:** Across every Model under `model/temporal` and every lifter fixture, a machine, action, timer, Property, Scenario, Query, Limits, assumption, hole, monitor and channel takes its name from its `val`. Errors: a declaration whose IR name must differ from its `val` states the name through the one explicit form; a computed name keeps its spelling (R1).
- **R3:** The string-named forms of the declarations in R2 are removed from `model/umpire` and from the lifter. Errors: a form still used by a test stays until the test is rewritten in the same task; none remains at the closing task.
- **R4:** A realization refers to its own roles, learned values, observations, evidence kinds, controls and commands by value. Errors: a reference to a declaration that does not exist fails to compile; an id the IR needs as text is written once, on its declaration.
- **R5:** Evidence that is the fact's own name is not written in any Model. Only exceptions are listed (no error surface beyond R1).
- **R6:** The other constructs fn-112 defines (the family as a `given`, the step helpers, machine derivation, compositions keyed by field, a refinement read without `Reads.through`, named action inputs, the realization script helpers) are the only form in `temporal/nexuscaller`, `temporal/nexuscaller/closepolicy` and `temporal/worker`. Errors: a construct that does not fit a Nexus declaration is listed in the done summary with the reason, and the old form stays for that declaration only.
- **R7:** Each IR file's contents are declared once in Scala, and one lifter run writes every Model IR file (or, under R8's fallback, one run per IR file that selects its Scala declaration by name). The gate program contains no fully qualified name of a Model declaration. Errors: a Scala-owned root that names nothing fails to compile, and a fixture root that names nothing fails the lift as today; a declaration listed for two IR files is lifted into both, as today.
- **R8:** The done summary of R7 states the wall-clock time of the gate's lift stage before and after. Errors: if the single run is slower, the summary says so and the task keeps the faster arrangement: parallel per-file lifter runs, each selecting its Scala IR-file declaration by name, so the roots stay in Scala.
- **R9:** The model gate, `make lint-model`, the Go tests of the Umpire tooling and `make lint-code-fast` pass at the closing task, and its done summary states the line counts of the Models and the gate program before and after (no error surface beyond the gates).
- **R10:** Every Model folder under `model/temporal` uses the file names fn-112 gives the standalone activity: `Model.scala` for domains, actions, step functions, machines, compositions and monitors, `Properties.scala` for Properties and progress claims, `Queries.scala` for Scenarios, Queries and Limits, and `Realization.scala`. No file is named `Claims.scala`. This applies to `nexuscaller`, `nexuscaller/closepolicy` and `worker`. Errors: a folder with nothing of a kind omits that file; a move that would change an IR name outside the allowed list keeps the declaration in its file and the done summary says so.
- **R11:** Each Model's string literals are counted before and after, by the method fn-112 R18 records. After this spec a string literal in any Model is one of the three kinds fn-112 R18 allows: prose a view shows, an id the IR needs as text written once on its declaration, or a name that differs from its `val`. Errors: a literal outside those three is listed with its line and the reason it stays.
- **R12:** `umpire` provides `sticky(predicate)` and `stickyAcross(predicate)` as plain `def`s in `model/umpire/Syntax.scala` (sugar for the `monitor` lambda, with its matching in `model/lifter/Syntax.scala` and an IR-equality fixture), the lifter lowers each to the existing Monitor IR (Boolean state, `initial = false`, `next = or(m, not(p))`, `violated` the identity, read after every step) with the Definition ID from the `val` as today, and `retainedOutcome` and `ownerAcknowledgment` use them with their `nexus-close.json` entries byte-identical. Errors: a predicate of another state type is a compiler refusal, recorded as such; one lifting fixture covers both arities and one refusal fixture is recorded; a monitor that needs history is not rewritten.

## Boundaries
<!-- scope: business -->

- No change to what any Model says.
- No IR schema change.
- No new construct beyond the IR-file declaration and the monitor pattern `sticky` (R12). A missing construct is a finding for fn-112 or a later spec.
- No change to what the Go reader, the lowering or the exports compute.
- No new library for the Models in this spec.
- The standalone activity Model is fn-112's. This spec touches it only to remove a form R3 retires and to add the IR-file declarations R7 requires for its IR files.

## Decision Context
<!-- scope: both — conditionally substructured -->

This work started as Part E of fn-113. fn-112 needs fn-113's evaluator removal first, and this rollout needs fn-112's constructs first, so keeping it inside fn-113 made the two specs depend on each other. As its own spec the order is a line.

Roots move beside the Models because the list in the gate is a second place that must change whenever a Model adds a Query set, and because it is what forces one JVM per IR file.

**Rejected:** letting the lifter lift every declaration it finds with no root declaration. The checked-in IR files group declarations on purpose, and a faulty control design must be able to stay out of a file.

**Plan amendment (2026-10-03, task breakdown).** Boundaries said this spec touches the standalone activity Model only to remove a retired form, while R7 requires every IR file's contents to be declared beside its Models, including the three activity IR files. The Boundaries bullet now also allows adding those IR-file declarations; no requirement changed. The former Parked unknown (the shape of the IR-file declaration) is scheduled work: task .1 settles it and records it under API Contracts.

**Plan-review amendments (2026-10-03).** Three contradictions found by plan review were fixed without changing intent. (1) R7 required one lifter run while R8's errors clause kept the faster arrangement, which can be per-file runs; R7 now admits R8's fallback, which still keeps the roots in Scala. R7's errors clause now says a Scala-owned root that names nothing fails to compile, because the lifter never sees it. (2) R10's errors clause said to keep a declaration's package to avoid an IR-name change, but the lifter names top-level functions after their file, not their package; it now says to keep the declaration in its file. (3) API Contracts now lists the allowed metadata deltas explicitly, including moved-function symbols that fn-112's harness already allows and that the file moves in R10 need.

**Plan amendment (2026-10-03, temporal patterns).** fn-112 adopted three level-1 claim patterns for Properties (`once/keeps`, `never/from`, `stays/unless`; `.plans/TEMPORAL_PATTERNS.md`). The fourth, the monitor pattern `sticky`, has both of its uses in the close policy (`retainedOutcome`, `ownerAcknowledgment`), so it lands here in task 4 rather than in fn-112, as the one construct this spec adds beside the IR-file declaration (R12). It lowers tree-identically, so R1's frozen rule is untouched. `fn-122-capabilities-and-their-laws` runs alongside this spec and adds files only (`umpire/laws`, `temporal/laws`, `temporal/nexusoperation`, `lifter/Capabilities.scala`); where both touch the lifter's claim fold, the gate's root list or the golden config, whichever lands second rebases.

## Quick commands

```bash
make umpire-check-model
make lint-model
go test -count=1 -tags test_dep ./tools/umpire/...
make lint-code-fast
```

## Early proof point

Task fn-114-state-every-scala-model-declaration-once.1 validates the core approach (Scala-owned IR-file declarations that one lifter run reads, byte-identical with today's six IR files). If it fails, re-evaluate the declaration shape (annotation or registry instead of a value per file) before continuing with .2+.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | The golden set of fn-115 R2 (`tools/umpire/model/testdata/migration`, `tools/umpire/lower/testdata/migration`) passes before and after every task. Errors: a task that needs a difference states it, amends the allowed list in this spec, and gets the owner's agreement before landing. | fn-114-state-every-scala-model-declaration-once.1, fn-114-state-every-scala-model-declaration-once.2, fn-114-state-every-scala-model-declaration-once.3, fn-114-state-every-scala-model-declaration-once.4, fn-114-state-every-scala-model-declaration-once.5, fn-114-state-every-scala-model-declaration-once.6, fn-114-state-every-scala-model-declaration-once.7, fn-114-state-every-scala-model-declaration-once.8 | — |
| R2 | Across every Model under `model/temporal` and every lifter fixture, a machine, action, timer, Property, Scenario, Query, Limits, assumption, hole, monitor and channel takes its name from its `val`. Errors: a declaration whose IR name must differ from its `val` states the name through the one explicit form; a computed name keeps its spelling (R1). | fn-114-state-every-scala-model-declaration-once.2, fn-114-state-every-scala-model-declaration-once.4, fn-114-state-every-scala-model-declaration-once.6 | — |
| R3 | The string-named forms of the declarations in R2 are removed from `model/umpire` and from the lifter. Errors: a form still used by a test stays until the test is rewritten in the same task; none remains at the closing task. | fn-114-state-every-scala-model-declaration-once.7 | — |
| R4 | A realization refers to its own roles, learned values, observations, evidence kinds, controls and commands by value. Errors: a reference to a declaration that does not exist fails to compile; an id the IR needs as text is written once, on its declaration. | fn-114-state-every-scala-model-declaration-once.3 | — |
| R5 | Evidence that is the fact's own name is not written in any Model. Only exceptions are listed (no error surface beyond R1). | fn-114-state-every-scala-model-declaration-once.2, fn-114-state-every-scala-model-declaration-once.4 | — |
| R6 | The other constructs fn-112 defines (the family as a `given`, the step helpers, machine derivation, compositions keyed by field, a refinement read without `Reads.through`, named action inputs, the realization script helpers) are the only form in `temporal/nexuscaller`, `temporal/nexuscaller/closepolicy` and `temporal/worker`. Errors: a construct that does not fit a Nexus declaration is listed in the done summary with the reason, and the old form stays for that declaration only. | fn-114-state-every-scala-model-declaration-once.2, fn-114-state-every-scala-model-declaration-once.3, fn-114-state-every-scala-model-declaration-once.4, fn-114-state-every-scala-model-declaration-once.6 | — |
| R7 | Each IR file's contents are declared once in Scala, and one lifter run writes every Model IR file (or, under R8's fallback, one run per IR file that selects its Scala declaration by name). The gate program contains no fully qualified name of a Model declaration. Errors: a Scala-owned root that names nothing fails to compile, and a fixture root that names nothing fails the lift as today; a declaration listed for two IR files is lifted into both, as today. | fn-114-state-every-scala-model-declaration-once.1 | — |
| R8 | The done summary of R7 states the wall-clock time of the gate's lift stage before and after. Errors: if the single run is slower, the summary says so and the task keeps the faster arrangement: parallel per-file lifter runs, each selecting its Scala IR-file declaration by name, so the roots stay in Scala. | fn-114-state-every-scala-model-declaration-once.1, fn-114-state-every-scala-model-declaration-once.8 | — |
| R9 | The model gate, `make lint-model`, the Go tests of the Umpire tooling and `make lint-code-fast` pass at the closing task, and its done summary states the line counts of the Models and the gate program before and after (no error surface beyond the gates). | fn-114-state-every-scala-model-declaration-once.8 | — |
| R10 | Every Model folder under `model/temporal` uses the file names fn-112 gives the standalone activity: `Model.scala` for domains, actions, step functions, machines, compositions and monitors, `Properties.scala` for Properties and progress claims, `Queries.scala` for Scenarios, Queries and Limits, and `Realization.scala`. No file is named `Claims.scala`. This applies to `nexuscaller`, `nexuscaller/closepolicy` and `worker`. Errors: a folder with nothing of a kind omits that file; a move that would change an IR name outside the allowed list keeps the declaration in its file and the done summary says so. | fn-114-state-every-scala-model-declaration-once.2, fn-114-state-every-scala-model-declaration-once.5, fn-114-state-every-scala-model-declaration-once.6 | — |
| R11 | Each Model's string literals are counted before and after, by the method fn-112 R18 records. After this spec a string literal in any Model is one of the three kinds fn-112 R18 allows: prose a view shows, an id the IR needs as text written once on its declaration, or a name that differs from its `val`. Errors: a literal outside those three is listed with its line and the reason it stays. | fn-114-state-every-scala-model-declaration-once.2, fn-114-state-every-scala-model-declaration-once.3, fn-114-state-every-scala-model-declaration-once.5, fn-114-state-every-scala-model-declaration-once.8 | — |
| R12 | `umpire` provides `sticky(predicate)` and `stickyAcross(predicate)`; the lifter lowers each to the existing Monitor IR with the Definition ID from the `val`; `retainedOutcome` and `ownerAcknowledgment` use them with byte-identical `nexus-close.json` entries. | fn-114-state-every-scala-model-declaration-once.4 | — |


