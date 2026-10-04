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
| 1 | fn-112 foundations | Settle declaration, step, composition and author-computed Query-total contracts | — |
| 2 | fn-120 choices | Build named choices against the settled step surface, before branching Models are rewritten | fn-112 foundations |
| 3 | fn-112 showcase | Rewrite the standalone activity Model, extract the shared task queue and build the realization kit | fn-120 choices |
| 4 | fn-114 | Roll the final showcase constructs, including choices, out to every other Model | fn-112 showcase |
| alongside 3 | fn-118 interface | Settle hint-aware realization helpers with the shared kit; no waiting-behavior changes yet | done in fn-118.1 |
| after 4 | fn-118 behavior | Derive generated-test waits from API hints, as a separate behavioral change | structural Case-byte freeze verified |
| after 4 | fn-120 tools | Add model lint (including specification holes and a coverage summary) and the IR explorer | final reader contracts and fn-114 roots |
| alongside 4 | fn-122 | Capabilities and their laws: shared Temporal promises stated once, adopted per entity | fn-112.10 |
| after 3 | fn-121 | Shard generated Cases per Case in CI, with HSM/CHASM per Nexus Case | fn-112.10 |
| last | fn-119 | Example: one Go SDK workflow driven end to end from the IRs, with no hand-written Go | fn-118, fn-120 |

These are execution phases, not new specs. Build choices before the Model conversions so each
branching step uses its final syntax once. Coordinate settled schema additions and compatibility
handling; do not guess API-hint fields before their inventory. The structural migrations still freeze
Case bytes, while fn-118 separately permits specified Program changes and freezes Contracts. After
shared schema/reader changes settle, hint behavior and Go-only tools can run in parallel. fn-119's
generic Driver primitives are done (fn-119.1-.2); the rest waits for fn-114, fn-118 and fn-120.
Flow records dependencies only within a spec, so the conductor holds the cross-spec gates: fn-112.6
waits for fn-120.1, fn-112.9 builds against the interface in `.plans/API_BEHAVIOR_HINTS.md`, and
fn-121 and fn-122 wait for fn-112.10.

Design decisions taken on 2026-10-03 and folded into the specs: operator policy
(`.plans/DSL_OPERATORS.md`: words for logic, symbols only where every programmer knows them);
syntactic sugar lives in separate `Syntax.scala` files in the framework, the Temporal kit and the
lifter, each form lifting to the same IR as its core spelling; three-level temporal claims
(`.plans/TEMPORAL_PATTERNS.md`: named patterns now, `always`/`eventually` deferred, raw IR as the
backend form); reusable behavioral protocols (`.plans/SEMANTIC_PROTOCOLS.md`, vision #PROTOCOLS);
MUST/MAY/MUST NOT as generated views with a specification-hole lint rather than author keywords
(`.plans/MODALITIES.md`).

### fn-112: Make the standalone activity Scala Model a DSL showcase

Rewrites the standalone activity Model to read as the best Scala the DSL allows, without changing
its behavior: machine derivation (`rebind`, `extend`, `refining`, `assuming`, `unmonitored`) in place of copied machines,
compositions keyed by fields in place of strings, names taken from `val`s, and a shared Temporal kit
for what the activity and Nexus realizations both use. Files are split by kind (`Model.scala`,
`Properties.scala`, `Queries.scala`) and the system contract by subject into folders. The current
feature has 2,830 lines; the targets are at most 1,600 lines and 60 string literals. Each new
construct is built once in the lifter, and realization helpers use the typed Temporal API.
Two additional tasks require author-computed Query totals for capacity review and extract the
reusable task-queue entity, providers and shared properties into `temporal/taskqueue/`. Shared
claims are written with named patterns (`once(...).keeps(...)`, `never(...).from(...)`,
`stays(...).unless(...)`) as parameterized definitions fn-122 turns into laws.

Tasks 1 and 2 are done: the original-baseline archive and equivalence check, the DefinitionScope probe,
the starting metrics (2,830 lines, 462 string literals), and captured declaration names, evidence defaults
and refinement reads in the DSL and lifter (production Models not yet migrated). Task 3 is next. Task 6 also makes the 168
state/action pairs disabled only by a default arm explicit; two Properties that are false on 120
rows each and seven pause/unpause rows the server rejects are recorded follow-ups, because the
behavior freeze forbids changing them here.

### fn-114: State every Scala Model declaration once

Split out of fn-113 so the specs run in a line. Rolls fn-112's constructs out to the Nexus caller, its close policy, the worker and the lifter
fixtures, and removes the string-named forms from the framework. Realizations refer to their own
ids by value, identity evidence lines go, and the contents of each IR file are declared in Scala
with one lifter run writing all of them. Every Model folder gets the same file names as the
activity, and no `Claims.scala` remains. Lifter fixtures that copy live Model text shrink to minimal fixture-local Models (fn-114.10), and a final rename (fn-114.9, owner request 2026-10-04) gives the tool folders
names that say what they hold: `model/lifter` becomes `model/irgen`, `model/gate` becomes `model/check`
(absorbing `model/metrics`), and the `model/gen` build cache becomes `model/build`.

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
