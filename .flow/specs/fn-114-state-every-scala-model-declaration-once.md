# State every Scala Model declaration once

## Goal & Context
<!-- scope: business -->

A feature developer writes and reviews Models under `model/scalav2/scala/temporal`. A review on 2026-10-01 found that the Models state many things twice, mostly as string literals.

- 76 of 93 named declarations repeat their `val` name as a string.
- The two realizations hold 46 private string constants that other strings refer to by spelling.
- `nexusProduct` lists six evidence lines that map each fact to its own name.
- `run.sh` lists 35 fully qualified Scala declarations to say what goes into which IR file, and starts one lifter JVM per IR file.

`fn-112-make-the-standalone-activity-scala` introduces the constructs that remove this repetition and applies them to the standalone activity. This spec applies them to every other Model and fixture, removes the old forms from the framework so that one way to write a declaration remains, and moves the IR roots into Scala.

This spec was Part E of `fn-113-clean-up-the-scala-model-layer-around`. It is a spec of its own so that the three specs run in a line: fn-113, then fn-112, then this one.

## Architecture & Data Models
<!-- scope: technical -->

No new mechanism beyond what fn-112 defines, with one exception. Today `run.sh` decides which declarations each IR file holds. After this spec a Scala declaration decides it, and the lifter reads that declaration and writes every Model IR file in one run.

| Concern | Today | After |
| --- | --- | --- |
| Name of a declaration | string argument, usually equal to the `val` name | the `val` name, read by the lifter |
| References inside a realization | a string equal to another declaration's id | the declaration itself, by value |
| Evidence for a fact | one line per fact, including identity lines | only the exceptions |
| Contents of an IR file | root lists in `run.sh` | one declaration per IR file in Scala |
| Lift | one JVM per IR file | one run writes every Model IR file |

The lifter fixtures keep their own lift step, because some of them must fail.

## API Contracts
<!-- scope: technical -->

The author surface is fn-112's. This spec adds no construct except the IR-file declaration, whose shape the first task settles and records here. The lifter's `lift: <file>:<line>: <message>` refusal format and the IR schema stay as they are.

**Lifted meaning is frozen.** For every IR file, `goir` derives the same tables, Definition IDs, refinement rows, fingerprints and Query answers before and after each task, and `goir/testpilot` lowers the same Case bytes. A name taken from a `val` equals the string the declaration wrote before.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Names that differ from their `val`.** 17 declarations do today, for example `storageLossAssumed` named `storageLoss`. Each one either renames its `val` to the IR name or states the name through the one explicit form fn-112 defines. Neither changes the IR name.
- **Computed names.** A Query whose name is computed from its machine's name keeps its spelling.
- **Realization ids the IR needs as text.** An id such as `temporal.nexus.caller.evidence.started` is written once, on the declaration it identifies. Every other mention is a reference to that declaration.
- **Removing an old form.** A string-named form leaves `scala/umpire` only when no Model, fixture or test uses it. The lifter's case for it is removed in the same task, and a fixture proves the lifter now refuses it or that it no longer compiles.
- **Fixtures that must not compile.** The `refuses` checks in `run.sh` name a fixture file and a line. They stay, and are the only place `run.sh` names anything inside a Scala source.
- **fn-107.** Tasks 10, 11 and 22 of `fn-107-scala-umpire-prototype-for-standalone` edit the activity realization and the lifter. This spec's tasks that touch those files start after they land.
- **Gates.** Each task runs the scoped parts of `model/scalav2/run.sh`, `make lint-scala` and `go test -tags test_dep ./model/scalav2/...`. The closing task runs all three in full and `make lint-code-fast`.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The baseline script of fn-113 R13 passes before and after every task. Errors: a task that needs a difference states it, amends the allowed list in this spec, and gets the owner's agreement before landing.
- **R2:** Across every Model under `scala/temporal` and every lifter fixture, a machine, action, timer, Property, Scenario, Query, Limits, assumption, hole, monitor and channel takes its name from its `val`. Errors: a declaration whose IR name must differ from its `val` states the name through the one explicit form; a computed name keeps its spelling (R1).
- **R3:** The string-named forms of the declarations in R2 are removed from `scala/umpire` and from the lifter. Errors: a form still used by a test stays until the test is rewritten in the same task; none remains at the closing task.
- **R4:** A realization refers to its own roles, learned values, observations, evidence kinds, controls and commands by value. Errors: a reference to a declaration that does not exist fails to compile; an id the IR needs as text is written once, on its declaration.
- **R5:** Evidence that is the fact's own name is not written in any Model. Only exceptions are listed (no error surface beyond R1).
- **R6:** The other constructs fn-112 defines (the family as a `given`, the step helpers, a refinement read without `Reads.through`, named action inputs) are the only form in `temporal/nexuscaller`, `temporal/nexuscaller/closepolicy` and `temporal/worker`. Errors: a construct that does not fit a Nexus declaration is listed in the done summary with the reason, and the old form stays for that declaration only.
- **R7:** Each IR file's contents are declared once in Scala, and one lifter run writes every Model IR file. `run.sh` contains no fully qualified Scala declaration name outside the lifter fixtures' checks. Errors: a root that names nothing fails the lift, as today; a declaration listed for two IR files is lifted into both, as today.
- **R8:** The done summary of R7 states the wall-clock time of `run.sh`'s lift stage before and after. Errors: if the single run is slower, the summary says so and the task keeps the faster arrangement.
- **R9:** `model/scalav2/run.sh`, `make lint-scala`, `go test -tags test_dep ./model/scalav2/...` and `make lint-code-fast` pass at the closing task, and its done summary states the line counts of `scala/temporal` and `run.sh` before and after (no error surface beyond the gates).

## Boundaries
<!-- scope: business -->

- No change to what any Model says.
- No IR schema change.
- No new construct beyond the IR-file declaration. A missing construct is a finding for fn-112 or a later spec.
- No change to `goir`, `goir/testpilot` or `backends` semantics.
- No library in `scala/temporal`.
- The standalone activity Model is fn-112's. This spec touches it only to remove a form R3 retires.

## Decision Context
<!-- scope: both — conditionally substructured -->

This work started as Part E of fn-113. fn-112 needs fn-113's evaluator removal first, and this rollout needs fn-112's constructs first, so keeping it inside fn-113 made the two specs depend on each other. As its own spec the order is a line.

Roots move into Scala because the list in `run.sh` is a second place that must change whenever a Model adds a Query set, and because it is what forces one JVM per IR file.

**Rejected:** letting the lifter lift every declaration it finds with no root declaration. The checked-in IR files group declarations on purpose, and a faulty control design must be able to stay out of a file.

## Parked unknowns

- The shape of the IR-file declaration (a `val` per file, an annotation, or one registry). R7's first task settles it against what the lifter can read from TASTy.
