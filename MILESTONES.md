# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-04.

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
- Commands that hold: run the full Go tooling suite with `-tags test_dep -p 2 -timeout 30m` (the
  lower, model and export test binaries take 3.5-5 GB each, so never more than two at once, and
  `-p 1` doubles wall time); run the model gate with `MODEL_GATE_ARGS=--skip-go-checks` when the Go
  suite runs separately, since its Go phase repeats it. Agents sharing one machine serialize heavy
  suites with one `flock` lock file. After any Model or lifter change, run `make umpire-gen-model` and
  the original-baseline check from fn-112.1: `go test -tags test_dep -count=1 -p 2 -run
  OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower`. Its
  allowed differences are listed in `tools/umpire/internal/golden/original.json`, which only the
  task introducing a difference extends.
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
| 1 | fn-114 | Roll the fn-112 showcase constructs, including choices, out to every other Model; then shrink copied fixture text and rename the model/ tool folders | — (in progress) |
| alongside 1 | fn-122 | Capabilities and their laws: shared Temporal promises stated once, adopted per entity | — (in progress) |
| alongside 1 | fn-121 | Shard generated Cases per Case in CI, with HSM/CHASM per Nexus Case | tasks 1-2 done and merged (each Case is its own shard unit, names pinned by a golden); task 3 needs GitHub CI |
| after 1 | fn-120 rollout and tools | Refuse unnamed branches; add model lint (specification holes, coverage summary) and the IR explorer | fn-114 |
| after 1 | fn-118 behavior | Derive generated-test waits from API hints, as a separate behavioral change | fn-114 |
| last | fn-119 | Example: one Go SDK workflow driven end to end from the IRs, with no hand-written Go | fn-118, fn-120 |

fn-112 (the standalone activity DSL showcase) is closed: the feature is 1,567 lines and 54 string literals
(targets 1,600 and 60), on captured names, machine derivation, typed compositions, claim patterns, input tokens,
`UpTo` counters, author-computed Query totals, a shared `model/temporal/taskqueue` entity and a shared Temporal
realization kit, with Case bytes unchanged. fn-120.1 (named choices) is done. These rows are execution phases,
not new specs. The structural migrations freeze Case bytes, while fn-118 separately permits specified Program
changes and freezes Contracts. fn-119's generic Driver primitives are done (fn-119.1-.2). Flow records
dependencies only within a spec, so the conductor holds the cross-spec gates: fn-120.2 and fn-118.2-.5 wait for
fn-114 to close.

Open for the owner: generated Cases that stop a worker (activity-terminate and activity-pauseResume always,
the two scheduleToStartTimeout Cases sometimes) come back INCONCLUSIVE because of a matching race, not Model or
test-shape changes. `ShutdownWorker` returns early when the task queue's root partition is not loaded yet
(`service/matching/matching_engine.go`, upstream #9424), never records the worker as shut down, and later polls
hang until `stop-worker` runs out its 10 s limit. With `frontend.enableMatchingFanOutForPollCancellation=false`
the same Cases are satisfied in 20 of 20 Runs. The choice is a server fix or that setting in the generated
test's Profile; fn-121.3's sharded CI run stays red on these Cases until then.

Open for the owner (deferred by fn-112's behavior freeze): the witness-only Properties `terminated` and
`cancelRequestedWhileStarted` (false on 120 rows each) and seven pause/unpause rows the server rejects (pause in
paused, pauseRequested and cancelRequested; unpause in scheduled, backingOff, started and cancelRequested).

Design decisions taken on 2026-10-03 and folded into the specs: operator policy
(`.plans/DSL_OPERATORS.md`: words for logic, symbols only where every programmer knows them);
syntactic sugar lives in separate `Syntax.scala` files in the framework, the Temporal kit and the
lifter, each form lifting to the same IR as its core spelling; three-level temporal claims
(`.plans/TEMPORAL_PATTERNS.md`: named patterns now, `always`/`eventually` deferred, raw IR as the
backend form); reusable behavioral protocols (`.plans/SEMANTIC_PROTOCOLS.md`, vision #PROTOCOLS);
MUST/MAY/MUST NOT as generated views with a specification-hole lint rather than author keywords
(`.plans/MODALITIES.md`).

### fn-114: State every Scala Model declaration once

Split out of fn-113 so the specs run in a line. Rolls fn-112's constructs out to the Nexus caller, its close policy, the worker and the lifter
fixtures, and removes the string-named forms from the framework. Realizations refer to their own
ids by value, identity evidence lines go, and the contents of each IR file are declared in Scala
with one lifter run writing all of them. Every Model folder gets the same file names as the
activity, and no `Claims.scala` remains. Type annotations the compiler and lifter do not need are dropped from the Models (fn-114.11), lifter fixtures that copy live Model text shrink to minimal fixture-local Models (fn-114.10), and a final rename (fn-114.9, owner request 2026-10-04) gives the tool folders
names that say what they hold: `model/lifter` becomes `model/irgen`, `model/gate` becomes `model/check`
(absorbing `model/metrics`), and the `model/gen` build cache becomes `model/build`.


Task 1 is done: each Model folder declares its IR files in Scala (`val x = irFile("name")(roots...)` in
`IrFiles.scala`), one lifter run writes all six, and the gate's root lists are gone; the lift step went from
8-16 s to 5 s. The new files put standaloneactivity at 1,614 lines, above fn-112's 1,600; fn-114.8's counts
report it. Task 2 is done: the Nexus caller Model uses captured names, derivation and the four-file layout
(its four files went from 898 lines and 114 literals to 780 and 21); the source-path change moved the canary
Case identity, so its pinned Run was re-recorded. Task 3 is done: the caller realization refers to
its declarations by value through the shared kit (525 lines and 30 literals, from 645 and 60). Tasks 4 and 5
are done: the close policy uses the final DSL with a new `sticky` monitor form and the four-file layout (58
literals, from 148). Task 6 is done: the worker Model is `Model.scala`
with captured names and pinned IDs, and the lifter fixtures use captured names and named choices. Task 7
(retire the string-named forms) is next, then 11, 10, 9 and 8. `leadsTo` has no captured-name form yet,
which leaves two literals in the close policy.
### fn-118: Declare how Temporal APIs behave once, and let the generated tests use it

Task 1 is done: the inventory (16 Cases, 17 polls, 52 waits, 440 s declared wait budget) and the
hint-aware kit interface in `.plans/API_BEHAVIOR_HINTS.md`. The rest waits for the Case freeze.

Realizations hand-write their waiting today: three polls with a literal 250 ms interval and two
literal 5,000 ms timeouts. This spec declares how an API behaves once, as metadata beside the typed
API: whether a write is visible to a read at once or eventually, how long a condition may take, and
which error means "not yet". The hints travel through the IR, which is a schema change. The lowering
derives each Case's waiting from them, refuses a read after a write with no declared visibility, and
the Go framework waits by condition within the declared bound. Further hints (repeatable calls,
blocking reads, call cost) are candidates until an existing Case needs one.

### fn-120: Adopt what Quint does well

Decides on ten suggestions from a review of Quint and adopts three; ITF interchange was withdrawn by the
owner on 2026-10-04. A `choose` construct names the
alternatives of a nondeterministic step and the IR records the names, which is a schema change. A
lint command reports model-quality findings from the IR (an unreachable case, an action never
enabled, a Property no Query names, a fact with no evidence) and the gate fails on a new one. An
explorer steps through a machine from the IR and says why a class is disabled, at its Scala line.
The lint also reports specification holes (a pair disabled only by a
default arm, an enabled class no Property constrains, a Property only pinned Queries ask), with a
per-operation rules table and a per-state view of MAY, MUST and MUST NOT. Temporal operators, Scenario combinators and Queries answered
by Quint, Apalache or TLC are recorded as later work with their own specs.

### fn-119: Show one Go SDK workflow driven end to end from the IRs

A showcase for newcomers: a workflow that runs one activity (completion, retry, timeout) is modeled
in Scala, and a Go SDK worker executes it against a real server with no hand-written Go for the
feature. The workflow, the activity answers, the test and the verdict all follow from the two IRs,
and a check fails if any Go file names the example. A walkthrough follows one Query from its Scala
declaration to the Verdict. The Testpilot Driver realizes only one workflow command today
(scheduling a Nexus operation), so the spec adds the general primitives an activity workflow needs.
The workflow is the Driver's interpreter executing the Case; testing a hand-written workflow
function is out of scope. Tasks 1 and 2 are done: the Driver schedules an activity, routes its
retries to the Case's script and can withhold an attempt so the server times it out.

### fn-121: Shard generated Cases per Case in CI

CI shards functional tests by test name, and the salt optimizer balances by depth-2 names. Today
every generated Case runs inside `TestTestpilotGeneratedCases/<hsm|chasm>`, so two blocks hold all
Cases, standalone activity Cases run twice, and the Testpilot tests run only in the unsharded
`umpire-check-live-tests`. The spec names each Case at depth 2, runs HSM and CHASM only for Nexus
Cases, pins the names to the manifest and runs the generated Cases in the sharded functional job.

### fn-122: Capabilities and their laws

An entity declares capabilities (`Closable`, `Terminable`, `Pausable`, `Pollable`, `Describable`)
and their bindings; each capability brings laws, and pairs bring interaction laws without being
listed. The pilots lift `terminalIsFinal` into `terminalStatesAreFinal` and `pausedIsNotDispatched`
into a `Pausable × Pollable` law, on standalone activity and a minimal Nexus operation Model. A law
joins only once two entities adopt it; an entity that differs overrides it with a recorded reason.

Task 1 is done (branch `umpire-fn122`, merging): law bodies and the catalog as plain defs with server
citations; the inventory classifies 76 claims as 8 law instances and 68 feature-specific. Three bodies wait for
task 2's binding of plain value arguments. Task 2 is done on the same branch: `capabilities(m, limits)(...)`
with `except` and `overriding` (each with a reason) lifts each law into a Property, Scenario and Query named
`<machine>.<law>`, a `<file>.laws.json` sidecar sits beside each IR file that declares capabilities, and all five
laws now lift. Task 3 is done on the branch: the activity declares Closable, Pausable, Pollable, Terminable,
Cancelable and Describable, its law claims are generated and every generated twin answers as the claim it
retired, with two new generated Cases. `closedIsRejectedUniformly` is false for the admission record (a delivery
to a timed-out record is accepted and recorded as `admissionRejected`), so the designs waive it with that reason.
The new live Cases and the authored terminate Case time out inconclusive under shared load (10 s Contract
window); fn-118's derived waits take that up. Task 4 (standalone Nexus operation Model) is next.
