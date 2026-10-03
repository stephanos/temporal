# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-03.

## Keeping this page current

- This page describes the present. Rewrite a status in place; do not append dated entries.
- Remove a spec or task from this page when it is done. Flow and git keep the history.
- Close a cancelled or abandoned spec in Flow (tasks blocked, a "Closed: won't do" note in the
  spec) and remove it from this page.
- Update the "As of" date with every edit.

## Verification instructions for agents

Apply these instructions when implementing the milestones:

- Reuse the previous task's passing baseline when its commands, source scope, fixtures and
  environment still apply. Inspect its recorded evidence; a new task or agent is not a reason to
  rerun it. Changes to relevant inputs invalidate the affected results.
- During implementation, run the smallest tests that exercise the changed behavior and its failure
  modes. Once ready for review, run the task's required full tests, goldens, lint and dependency
  checks once. After a fix, repeat affected checks; repeat broader gates only when the change or
  failure invalidates their results. Preserve all required coverage and acceptance criteria.
- Give reviewers the source scope, commands, results and log paths from that run. Reviewers inspect
  this evidence and request an additional check only for a concrete unresolved concern. Resume the
  same review after fixes, with the changed code and relevant new results.
- Measure the next already-required full Go test run with `-json`, retaining its exit status and
  output in the task's `.flow/tmp/` directory. Use test completion events to identify slow tests;
  package times overlap and must not be summed as wall time. Record elapsed wall time separately.
  Keep a running process running; add instrumentation to the next run instead of restarting it.
- Optimize measured bottlenecks, starting with duplicated model construction or golden decoding
  if the timings implicate them. Preserve independent assertions and immutable fixtures; shared
  test setup must not leak mutable state. Compare timings on equivalent inputs before claiming a
  speedup. Do not add a profiling or caching framework without evidence it is needed.
- Keep handovers concise and link existing evidence. Add audits, inventories or verification gates
  only for an explicit requirement or a concrete uncovered risk. Move to the next implementation
  task once the required checks and review pass.

## Direction

The Scala model (`model/`) is the model. Lean is the past, and nothing has to look like it.
Scala declares, a lifter reads the declarations into the Umpire IR, and generic Go consumers check
that IR, lower it to Testpilot Cases, and run those Cases as functional tests and canary checks. The
Umpire IR and the Testpilot IR are what connect the parts. The Lean toolchain is removed from this
checkout, and the Lean-only specs are closed as won't-do. See [SCALA.md](.plans/SCALA.md) and
[UMPIRE4_SPEC.md](.plans/UMPIRE4_SPEC.md).

The planned work below continues that direction: the Scala layer first, then the Models.

## Planned

fn-115 is closed, with an independent completion review. `model/` is the Scala model and holds no
Go or shell script; `tools/umpire/` is its Go tooling; the earlier trees are archived in `model0/`
and `tools/umpire0/`, outside the build. One Scala gate at `model/gate` builds, lifts and checks the
model, the IR namespace is `umpire/v1`, the functional fixtures and the canary's pinned Case are
lowered from the Scala model, and tests enforce the import rules. The 1,411 semantic and artifact
goldens are the baseline for the specs below. `model/README.md` describes the system for a newcomer.

Deferred by the owner: `make umpire-check-backends`, which needs P and .NET installed. Open for the
owner: the canary policy's `workflowPath` names the deleted production-canary workflow, so
production dispatch fails closed.

The CEL spike (fn-116) is closed. Its [report](.plans/UMPIRE_CEL_SPIKE.md) recommends keeping the
IR's own expressions: CEL matched the current evaluator on every Model, but adoption grows the IR
1.8-3.7x, slows table derivation up to about 6x and adds more code than it removes. Adopting CEL
would be a new spec the owner opens after reading the report.

| Order | Spec | In one line | Waits for |
| --- | --- | --- | --- |
| 1 | fn-117 | Typed Temporal API in the Models, in place of proto names as strings | — |
| 2 | fn-112 | Rewrite the standalone activity Model as the DSL showcase | fn-117 |
| 3 | fn-114 | Roll the showcase's constructs out to every other Model | fn-112 |
| after 2 | fn-118 | API behavior hints (eventual consistency, wait bounds) that the generated tests use | fn-112 |
| after 3 | fn-120 | Named choices, a model linter, an explorer over the IR and ITF trace interchange | fn-114 |
| last | fn-119 | Example: one Go SDK workflow driven end to end from the IRs, with no hand-written Go | fn-118, fn-120 |

### fn-117: Type the Temporal API in the Models

Models name Temporal API messages, methods, field paths and enum values as string literals today,
about 100 of them, and the compiler checks none. This spec puts ScalaPB classes for the Temporal API
on the Models' classpath so that a Model writes `schema[StartActivityExecutionRequest]` and
selects a field such as `_.getTaskQueue.name`, and a wrong field or type does not compile. The IR
keeps names as text, so Go is unaffected. The [spike](.plans/UMPIRE_TYPED_API_SPIKE.md) passed
independent review and selected the complete ScalaPB API: a warm Model edit took 0.77 seconds
against a 0.87-second baseline, within the two-times limit. Selective generation and the typed
catalog fallback were not triggered.

The eight-task plan passed independent review. The feasibility spike and stamped build
integration passed their implementation reviews; complete API classes are available and Model
edits reuse the generated jar. Typed declarations, field paths, symbolic operands, constant
messages, the standalone activity and Nexus migrations, and positive authoring fixtures passed
review. Retirement of the string forms and closing checks remain.

### fn-112: Make the standalone activity Scala Model a DSL showcase

Rewrites the standalone activity Model to read as the best Scala the DSL allows, without changing
its behavior: machine derivation (`rebind`, `extend`, `unmonitored`) in place of copied machines,
compositions keyed by fields in place of strings, names taken from `val`s, and a shared Temporal kit
for what the activity and Nexus realizations both use. Files are split by kind (`Model.scala`,
`Properties.scala`, `Queries.scala`) and the 1,108-line `System.scala` by subject into folders. Its
string literals drop from 514 to a target of at most 60. Each new construct is built once, in the
lifter, which is why it follows fn-113, and its realization helpers are written against the typed
Temporal API of fn-117. It no longer carries three rules that fn-113
reverses: agreement with the native evaluator, the ban on libraries, and the freeze on Lean
comments.

### fn-114: State every Scala Model declaration once

Split out of fn-113 so the specs run in a line. Rolls fn-112's constructs out to the Nexus caller, its close policy, the worker and the lifter
fixtures, and removes the string-named forms from the framework. Realizations refer to their own
ids by value, identity evidence lines go, and the contents of each IR file are declared in Scala
with one lifter run writing all of them. Every Model folder gets the same file names as the
activity, and no `Claims.scala` remains.

### fn-118: Declare how Temporal APIs behave once, and let the generated tests use it

Realizations hand-write their waiting today: three polls with a literal 250 ms interval and two
literal 5,000 ms timeouts. This spec declares how an API behaves once, as metadata beside the typed
API: whether a write is visible to a read at once or eventually, how long a condition may take, and
which error means "not yet". The hints travel through the IR, which is a schema change. The lowering
derives each Case's waiting from them, refuses a read after a write with no declared visibility, and
the Go framework waits by condition within the declared bound. Further hints (repeatable calls,
blocking reads, call cost) are candidates until an existing Case needs one.

### fn-120: Adopt what Quint does well

Decides on ten suggestions from a review of Quint and adopts four. A `choose` construct names the
alternatives of a nondeterministic step and the IR records the names, which is a schema change. A
lint command reports model-quality findings from the IR (an unreachable case, an action never
enabled, a Property no Query names, a fact with no evidence) and the gate fails on a new one. An
explorer steps through a machine from the IR and says why a class is disabled, at its Scala line.
Umpire traces convert to and from ITF. Temporal operators, Scenario combinators and Queries answered
by Quint, Apalache or TLC are recorded as later work with their own specs.

### fn-119: Show one Go SDK workflow driven end to end from the IRs

A showcase for newcomers: a workflow that runs one activity (completion, retry, timeout) is modeled
in Scala, and a Go SDK worker executes it against a real server with no hand-written Go for the
feature. The workflow, the activity answers, the test and the verdict all follow from the two IRs,
and a check fails if any Go file names the example. A walkthrough follows one Query from its Scala
declaration to the Verdict. The Testpilot Driver realizes only one workflow command today
(scheduling a Nexus operation), so the spec adds the general primitives an activity workflow needs.
The workflow is the Driver's interpreter executing the Case; testing a hand-written workflow
function is out of scope.
