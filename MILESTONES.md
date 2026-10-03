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
Umpire IR and the Testpilot IR are what connect the parts. See [SCALA.md](.plans/SCALA.md) and
[UMPIRE4_SPEC.md](.plans/UMPIRE4_SPEC.md).

The planned work below continues that direction: the Scala layer first, then the Models.

## Planned

Deferred by the owner: `make umpire-check-backends`, which needs P and .NET installed. Open for the
owner: the canary policy's `workflowPath` names the deleted production-canary workflow, so
production dispatch fails closed.

| Order | Spec | In one line | Waits for |
| --- | --- | --- | --- |
| 1 | fn-112 foundations | Settle declaration, step, composition and author-computed Query-total contracts | — |
| 2 | fn-120 choices | Build named choices against the settled step surface, before branching Models are rewritten | fn-112 foundations |
| 3 | fn-112 showcase | Rewrite the standalone activity Model, extract the shared task queue and build the realization kit | fn-120 choices |
| 4 | fn-114 | Roll the final showcase constructs, including choices, out to every other Model | fn-112 showcase |
| alongside 3 | fn-118 interface | Settle hint-aware realization helpers with the shared kit; no waiting-behavior changes yet | shared-kit design |
| after 4 | fn-118 behavior | Derive generated-test waits from API hints, as a separate behavioral change | structural Case-byte freeze verified |
| after 4 | fn-120 tools | Add model lint, the IR explorer and ITF interchange using settled metadata and Model inventory | final reader contracts and fn-114 roots |
| last | fn-119 | Example: one Go SDK workflow driven end to end from the IRs, with no hand-written Go | fn-118, fn-120 |

These are execution phases, not new specs. Build choices before the Model conversions so each
branching step uses its final syntax once. Coordinate settled schema additions and compatibility
handling; do not guess API-hint fields before their inventory. The structural migrations still freeze
Case bytes, while fn-118 separately permits specified Program changes and freezes Contracts. After
shared schema/reader changes settle, hint behavior and Go-only tools can run in parallel; fn-119's
generic Driver primitives can also start independently before the final example integration.

### fn-112: Make the standalone activity Scala Model a DSL showcase

Rewrites the standalone activity Model to read as the best Scala the DSL allows, without changing
its behavior: machine derivation (`rebind`, `extend`, `refining`, `assuming`, `unmonitored`) in place of copied machines,
compositions keyed by fields in place of strings, names taken from `val`s, and a shared Temporal kit
for what the activity and Nexus realizations both use. Files are split by kind (`Model.scala`,
`Properties.scala`, `Queries.scala`) and the system contract by subject into folders. The current
feature has 2,830 lines; the targets are at most 1,600 lines and 60 string literals. Each new
construct is built once in the lifter, and realization helpers use the typed Temporal API.
Two additional tasks require author-computed Query totals for capacity review and extract the
reusable task-queue entity, providers and shared properties into `temporal/taskqueue/`.

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
