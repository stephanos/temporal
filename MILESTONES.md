# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-08.

## Keeping this page current

- This page describes the present. Rewrite a status in place; do not append dated entries.
- List each open spec's tasks with ID, status (✅ done, 🔄 in progress, ⬜ todo, ⏸️ deferred) and a brief
  description; set a task's status in place
  when it changes. Keep completed tasks listed until the whole spec is complete, then remove the spec.
  Flow and git keep the history.
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
- Optimize measured bottlenecks, starting with duplicated model construction if the timings
  implicate it. Preserve independent assertions and immutable fixtures; shared
  test setup must not leak mutable state. Compare timings on equivalent inputs before claiming a
  speedup. Do not add a profiling or caching framework without evidence it is needed.
- Commands that hold: run the full Go tooling suite with `-tags test_dep -p 2 -timeout 30m` (the
  lower, model and export test binaries take up to about 6.5 GB each (export), so never more than
  two at once, and `-p 1` doubles wall time); run the model gate with `MODEL_GATE_ARGS=--skip-go-checks` when the Go
  suite runs separately, since its Go phase repeats it. Agents sharing one machine serialize heavy
  suites with one `flock` lock file. After any Model or lifter change, run `make umpire-gen-model` (inside a batch, only at the batch's regeneration; see Batches),
  review the diff of `model/ir` and `model/cases`, and run `make umpire-check-cases`: the reader's
  tests over `model/ir` pin what a Model means, and the managed Case trees what its Cases contain.
- Keep handovers concise and link existing evidence. Add audits, inventories or verification gates
  only for an explicit requirement or a concrete uncovered risk. Move to the next implementation
  task once the required checks and review pass.

## Direction

The Scala model (`model/`) is the model. Lean is the past, and nothing has to look like it.
Scala declares, a lifter reads the declarations into the Umpire IR, and generic Go consumers check
that IR, lower it to Testpilot Cases, and run those Cases as functional tests and canary checks. The
Umpire IR and the Testpilot IR are what connect the parts. See [SCALA.md](.plans/SCALA.md) and
[UMPIRE4_SPEC.md](.plans/UMPIRE4_SPEC.md).

The DSL framework (`model/umpire`) stays Temporal-agnostic as far as is realistic: Temporal's capability
vocabulary, properties, realization vocabulary and kit live under `model/temporal/`, and the lifter and Testpilot IR
are the parts that are Temporal's driver tooling by design (fn-114.12, fn-122.8).

The planned work below continues that direction: the Scala layer first, then the Models.

## Specs

Listed in delivery order. Flow records spec and task dependencies; the conductor also holds
batch-close gates and serializes work that shares a regeneration baseline.

### Activity model batch: fn-128 → fn-138 → fn-129

Ready 2026-10-07. These specs run as one batch: one regeneration, one gate run, and one live run serving
fn-128.6 and fn-129.5. Source: `.plans/ACTIVITY_MODEL_COMPARISON.md`. Each task declares its IR change,
which is checked at the batch regeneration.

Approved source gates: fn-138 implementation starts after fn-128.5 is done; fn-129 implementation
starts after fn-138.3 is done. Flow cannot express these cross-spec task edges, so the conductor
enforces them without spec-close dependencies inside the batch. fn-128.6 and fn-129.5 share the
regeneration, review and live-run evidence; no activity spec closes before that boundary.
The fn-142 then fn-143 preparation may run alongside this batch in isolated worktrees. The fn-142
join dry-run found a milestones-file conflict; preserve its isolated work and serially rerun the
move after this batch closes, with a separate regeneration baseline. Re-anchor later task paths
after those moves; fn-143 still waits for fn-142 to close.

### fn-128: Close the activity's precision gaps

Tasks run in order.

| Task | Status | What |
| --- | --- | --- |
| fn-128.1 | ✅ done | Dispatch as a field replacing the `backingOff` phase; start delay; unpause-after-backoff and schedule-to-start-in-backoff fixed |
| fn-128.2 | ✅ done | Explicit rejection rows and repeated RequestCancel refusal; owned lint subjects removed; focused tests pass; artifacts/full gates remain batch-deferred |
| fn-128.3 | ✅ done | 2026-10-08: finite retry policy and retryable start-to-close timeout; exhaustion/timeout-retry Queries and bounded withholding bridge; focused source/lowering/runtime checks pass; artifacts/full gates/review/live remain batch-deferred |
| fn-128.4 | 🔄 in progress | Stutter facts checked: `visible` on `ActivitySystem`'s refinement |
| fn-128.5 | ⬜ todo | `cancelIsNotUndone` Property; attempt count in every Case; time-window `because` |
| fn-128.6 | ⬜ todo | Evidence map, live Cases run once (the batch's live run); close |

### fn-138: Retries and Deadline capabilities

Runs after fn-128.5: fn-128.1 replaces the `backingOff` phase and fn-128.3 adds the retry policy,
both of which Retries reads. Open questions for the owner are in the spec; tasks .1 and .2 settle them
before writing Properties.

Retries (a retryable failure of a `Held` attempt lands in `Waiting`, a non-retryable one in `Failed`,
attempts never exceed the bound) and Deadline capabilities are declared by the activity and Nexus
workflow systems; their timer windows become role tests.

| Task | Status | What |
| --- | --- | --- |
| fn-138.1 | ⬜ todo | Retries capability: kit Properties and lifter refusals (owner question first) |
| fn-138.2 | ⬜ todo | Deadline capability: kit Properties, per-declaration names and lifter refusals |
| fn-138.3 | ⬜ todo | Activity and Nexus workflow systems declare Retries and Deadlines; timer windows as role tests; docs |

### fn-129: Activity coverage

Runs after fn-138. Tasks run in order.

| Task | Status | What |
| --- | --- | --- |
| fn-129.1 | ⬜ todo | Heartbeat action and retryable heartbeat timeout; realized |
| fn-129.2 | ⬜ todo | Respond by ID as a `service` actor; realized |
| fn-129.3 | ⬜ todo | Reset with `keepPaused` and deferred apply; precedence Property extended; realized |
| fn-129.4 | ⬜ todo | Exploration on the activity's `find` Queries |
| fn-129.5 | ⬜ todo | New Cases listed, live run (the batch's live run); close |

### fn-140: One-sentence witness Queries with explicit live expectations

Runs next. It rewrites the Models' `properties` and `queries` sections and its R5 renames Definition
IDs. Ready, with six M-sized tasks. The three foundations run in order; task 4 and the documentation
task 5 are disjoint parallel candidates, and task 6 joins them for final regeneration and gates.

`witness(<classes>).records(<fact>)` states a path-and-outcome claim in one declaration and lifts to the existing Scenario, Property and `find` Query, each named after it. `.live(<expectation>)` replaces `.expect` as the one word that generates a Case, and every non-satisfied expectation carries a reason. Pinned `find` Queries in every Model migrate, and the hand-written `terminate`, which repeats the capability-generated `terminateSettles`, is deleted. `query verify` and its triple are unchanged.

| Task | Status | What |
| --- | --- | --- |
| fn-140.1 | ⬜ todo | Typed witness builders and existing core Query values |
| fn-140.2 | ⬜ todo | Witness lifting, names, bounds, composed facts and capability duplicate refusals |
| fn-140.3 | ⬜ todo | Query `.live`; source-only non-satisfied reasons; all caller and fixture spellings |
| fn-140.4 | ⬜ todo | First Activity conversion and assessment equivalence proof; duplicate terminate removed |
| fn-140.5 | ⬜ todo | Author docs, core form and layout template witness |
| fn-140.6 | ⬜ todo | Other eligible Models, source-aware triple lint, final regeneration and equivalence gates |

### fn-123: Declare faults as the environment's actions

Ready 2026-10-07. Starts after fn-140 and is written against the names it leaves (`when` blocks,
`.live`, the shared `Outcome`, the fault instruction and typed `perform`). The approved delivery order
then runs fn-145 through fn-148, with fn-141 last against the settled schema. Tasks run in order; 6 and 7 need
only 5. Task 3 is the proof: Go derives crash rows equal to `crashDetail`'s from a durability
classification, or the work stops for the owner. Planning took defaults for five owner questions,
listed in the spec's Open Questions.

A fault is declared once with its kind, budget and whether it can be realized; a machine classifies
its fields as durable or in-memory, and Go derives the crash. The explorer half of R7 and the P clause
of R8 were dropped, since neither tool exists.

| Task | Status | What |
| --- | --- | --- |
| fn-123.1 | ⬜ todo | Fault kinds and `modelOnly` on the `fault` actor's actions; IR `Fault` record; spellings settled |
| fn-123.2 | ⬜ todo | Durability classification and `crashes(…)` in the DSL and lifter, on fixtures |
| fn-123.3 | ⬜ todo | Go derives the crash row; `SEMANTICS.md` Faults section (proof) |
| fn-123.4 | ⬜ todo | Task-queue providers converted; `fault-overridden` lint; derived storage-loss assumption |
| fn-123.5 | ⬜ todo | Budgets: `budgetedBy`, four table rules in Go, `LostStartAnswer` bound to `lossAvailable` |
| fn-123.6 | ⬜ todo | Choice-level fault performance; lowering refuses model-only faults and unperformed choices |
| fn-123.7 | ⬜ todo | Trace output and Quint export of derived crashes and budgets |
| fn-123.8 | ⬜ todo | `umpire-faults` report, docs; close |

### fn-142: Split `model/temporal/shared` into `foundations` and `actors`

Mechanical move: the task queue goes to `foundations/taskqueue`, the worker to `actors/worker`,
`Client.scala` to `actors/client/`, and `Bounds.scala` to `model/temporal/`; the IR differs only in
paths and positions.

| Task | Status | What |
| --- | --- | --- |
| fn-142.1 | 🔄 in progress | Isolated move and equivalence checks passed; preserve worktree and serially rerun from the closed activity batch after the join dry-run found a milestones-file conflict; integrated review/gates pending |

### fn-143: Rename `model/umpire` to `model/framework`

Gate: after fn-142. Folder and package both become `framework`; product names (`tools/umpire`, `umpire-*` targets, `umpire.v1`) stay. The IR differs only in paths and positions.

| Task | Status | What |
| --- | --- | --- |
| fn-143.1 | ⬜ todo | Move, rename the package, regenerate, prove the diff is the umpire.→framework. mapping, docs |

### IR schema research: fn-145 → fn-146 → fn-147 → fn-148

Planned 2026-10-06 from [Umpire IR schema research](.plans/UMPIRE_IR_SCHEMA_RESEARCH.md) and
[Testpilot schema research](.plans/TESTPILOT_SCHEMA_RESEARCH.md). Gate: fn-142/fn-143 and fn-123 close
before fn-145; each following spec waits for its predecessor, and fn-141 executes last after fn-148.
These are separate migrations, each with its own baseline, regeneration and full gates, not another
DSL batch. Serialize fn-140, fn-131 and any revived fn-144 or Model batch against these migrations; re-anchor their tasks
to the schema and vocabulary left by completed work. Deferred specs do not block this chain.

Owner decision: breaking IR changes are allowed. Producers, consumers, generated artifacts and
recorded Case/Run companions migrate together. No compatibility decoders, legacy evaluators or
parallel old field spellings are required; retired formats reject explicitly. The declaration-only
split still proves semantic and artifact equivalence, not a historical compatibility promise.

### fn-145: Modularize the Umpire IR schema

Nine files, one existing protobuf package and an acyclic import graph. This owns fn-131's realization
extraction slice; fn-131 retains metadata, canonicalization, level checks and producer provenance.
No declaration is removed merely because generated production Models omit it. Shared settings and
disposition/cleanup leaves receive a maintenance evaluation, not an assumed shared schema.

| Task | Status | What |
| --- | --- | --- |
| fn-145.1 | ⬜ todo | Full schema closure, linked descriptors and Scala jar exclusions; multi-file proof |
| fn-145.2 | ⬜ todo | Extract the nine responsibility files without changing declarations or Model meaning |
| fn-145.3 | ⬜ todo | Local empty marker → `google.protobuf.Empty`; preserve oneof meanings |
| fn-145.4 | ⬜ todo | Equivalence gates, shared-leaf evaluation, schema ownership docs; close |

### fn-146: Adopt CEL for runtime predicates and values

Gate: after fn-145. Testpilot owns the restricted CEL environment and descriptor-aware value adapter;
Umpire lowers symbolic realization operands into it. Finite Model expressions, `ModelValue` and
descriptor-exact `ValueType` remain separate. Formats and identities move together, without legacy
runtime paths. Tasks run in order; admission and value adaptation share edit surfaces.

| Task | Status | What |
| --- | --- | --- |
| fn-146.1 | ⬜ todo | Breaking format, deterministic canonical identity and companion migration contract |
| fn-146.2 | ⬜ todo | Canonical CEL AST, restricted admission, pinned engine bridge and budgets |
| fn-146.3 | ⬜ todo | Standard CEL values with authoritative descriptors, exact numbers and opaque `Any` |
| fn-146.4 | ⬜ todo | Native execution and Driver/worker values; descriptor/capture and online/offline proof |
| fn-146.5 | ⬜ todo | Contract and correlated verification, rule expansion and reference walkers |
| fn-146.6 | ⬜ todo | Umpire operand lowering, Run Event guards and conformance agreement |
| fn-146.7 | ⬜ todo | Retire custom machinery, regenerate companions, full gates and docs; close |

### fn-147: Migrate elapsed-time fields to protobuf Duration

Gate: after fn-146. Seven Testpilot elapsed-time fields and corresponding Umpire hints/defaults
migrate through checked whole-millisecond conversions. Absent polling interval means one read;
present positive interval means polling. Counts, logical bounds, percentages and timestamps stay.
Scalar-only singleton-oneof presence cleanup belongs here. Tasks run in order.

| Task | Status | What |
| --- | --- | --- |
| fn-147.1 | ⬜ todo | Inventory, exact conversion bounds, defaults, presence and monotonicity; proof |
| fn-147.2 | ⬜ todo | Replace schemas and migrate Scala authoring, lifting, realization admission and producers |
| fn-147.3 | ⬜ todo | Runtime consumers, polling policy and scalar presence |
| fn-147.4 | ⬜ todo | Regenerate artifacts and Run companions, categorize identities, full gates and docs; close |

### fn-148: Consolidate Testpilot evidence and correlated state schemas

Gate: after fn-147. One generalized evidence declaration replaces inline extraction. Response lifts
keep ordered first-match behavior; Run Event overlaps still fail. Contract evidence policies remain
independent. Complete states include both atom and fields; results are separate from authorized
prior-state transitions, and projection result order stays explicit. Tasks run in order.

| Task | Status | What |
| --- | --- | --- |
| fn-148.1 | ⬜ todo | Generalized evidence declarations in a leaf schema; full lift-shape proof |
| fn-148.2 | ⬜ todo | One binder and lowering path; source selection and independent Contract policies |
| fn-148.3 | ⬜ todo | Complete-state and result tables in schema/lowering; compact-size measurement |
| fn-148.4 | ⬜ todo | Normalized admission/verification, causal authorization and expanded-work ceilings |
| fn-148.5 | ⬜ todo | Derived Contract kind, fixed correlated clock and explicit support Boolean |
| fn-148.6 | ⬜ todo | Local references, cardinality, Empty markers and coordinated format 4.0 activation |
| fn-148.7 | ⬜ todo | Artifact/companion migration, measurements, full gates and ownership docs; close |

### fn-141: Shrink the IR generator: one description of each DSL construct

Ready 2026-10-08. The approved delivery order puts this spec last, after fn-148 closes, against the
settled schema and the vocabulary left by fn-140, fn-123 and fn-145 through fn-148.
Because its tasks were planned against the earlier tree, each re-reads its files and recounts first.
Tasks run in order; 1 and 2 depend on nothing, and 6 and 7 need only them. Task 5 is the proof for the
export part and task 8 its size proof; either can stop tasks 9 to 13, and tasks 1 to 4 stand without
them. Source: a full read of `model/irgen` on 2026-10-06 (8,242 lines, of which about 10% lifts
function bodies and types and 1,479 are lints that emit no IR).

Each DSL construct is described once. Four parts, in this order: the declaration-order lint, the
structure lint and the marker checks leave `model/irgen` for their own gate step; spellings no Model
or kit uses are retired; a sugar's definition becomes its only lowering, so the lifter knows core
constructs and types alone; and declarations are exported from the constructed Models, one kind at
a time with realizations first, so the lifter lifts function bodies and types only. Declaration-level
Scala becomes free and function bodies stay in the liftable subset, so the IR schema, the Go consumers
and the Quint export do not change, and every step leaves `model/ir` and `model/cases` byte-identical.
It revises `.plans/DSL_OPERATORS.md` rule 5 and fn-113 R15: the framework uses `inline` and macros at
its capture points only. Proof point: the realization step stops the work unless it removes at least
half of the realization lifter's lines net of what it adds.

| Task | Status | What |
| --- | --- | --- |
| fn-141.1 | ⬜ todo | Order lint, structure lint and marker checks out of `model/irgen` into their own gate step |
| fn-141.2 | ⬜ todo | Spellings no Model or kit uses retired, after a recount the owner confirms |
| fn-141.3 | ⬜ todo | Generic sugar expansion; `enter`, `stay`, `reject`, `disabled`, `in`, `implies`, `records` lifted from their definitions (proof for the sugar part) |
| fn-141.4 | ⬜ todo | Claim patterns and sticky monitors by definition; lifter's sugar file gone; syntax lint holds the lifter to core |
| fn-141.5 | ⬜ todo | Capture points (name, ID, position; function span and captured values) and lifting a function by span (proof for the export part) |
| fn-141.6 | ⬜ todo | Refusal ledger: one outcome per refusal kind (deleted, kept and where, left to Go) |
| fn-141.7 | ⬜ todo | Realization factories keep everything they are given |
| fn-141.8 | ⬜ todo | Exporter; realizations exported; two-run determinism; init-order refusal; size proof with stop |
| fn-141.9 | ⬜ todo | Signature exported: actions, inputs, channels, assumptions, holes, Limits; `codeOf` gone |
| fn-141.10 | ⬜ todo | Machines exported: header, rules, monitors, refinement, derivations |
| fn-141.11 | ⬜ todo | Compositions and syncs exported; `sync` and `replaces` record what they pair |
| fn-141.12 | ⬜ todo | Properties, Scenarios, Queries and progress exported; the lifter's fold deleted; comprehension fixture |
| fn-141.13 | ⬜ todo | Capability expansions exported, in the shape the current framework leaves |
| fn-141.14 | ⬜ todo | Dead lifter code, rules of record, docs, backends check, size report; close |

## Captured, not yet scheduled

### fn-149: Safety and liveness groups for object properties

[Spec](.flow/specs/fn-149-safety-and-liveness-groups-for-object.md) captured with six acceptance
criteria; no tasks yet and not marked ready. Split authored claims into `properties.safety` and
`properties.liveness`, enforce declaration kinds, and carry the distinction into existing diagnostics
and reports. Liveness retains explicit bounds and assumptions. Migrate Models, shared laws and docs
while preserving behavior and check results.

Coordinate with fn-140's property/Query authoring changes and fn-141's declaration lifting changes.
Scheduling remains open; the approved delivery order above is unchanged.

Cross-machine safety uses compositions today. New composition progress support is tracked separately
in fn-150; it is not a prerequisite for fn-149's grouping.

### fn-150: Bounded liveness across composed machines

[Spec](.flow/specs/fn-150-bounded-liveness-across-composed.md) captured with five acceptance criteria;
no tasks yet and not marked ready. Let a composition own bounded progress claims over multiple
member states. Count composed steps, resolve fairness against synchronized and member-only actions,
and preserve deadlock, cycle, deadline and incomplete-check distinctions. Include one concrete
Temporal composition with passing and negative examples.

Related to fn-149's safety/liveness groups. Coordinate with fn-141 and the queued schema changes;
execution remains unscheduled. Live Case generation for compositions is outside this spec.

## Deferred

Specs the owner deferred keep their tasks here so they can be revived as planned.

### fn-119: Show one Go SDK workflow driven end to end from the IRs

Deferred 2026-10-04.

| Task | Status | What |
| --- | --- | --- |
| fn-119.1 | ✅ done | Driver's workflow schedules an activity, awaits it, completes with its result |
| fn-119.2 | ✅ done | Workflow-scheduled activity attempts routed to the Driver's interpreter |
| fn-119.3 | ⏸️ deferred | Workflow-scheduled activities and awaited outcomes in the realization DSL, lifter and lowering |
| fn-119.4 | ⏸️ deferred | Activity workflow example modeled; its Queries run live from the gate |
| fn-119.5 | ⏸️ deferred | Faulty variant and the zero-Go check |
| fn-119.6 | ⏸️ deferred | Walkthrough, one-command entry point; close |

### fn-125: Represent dynamic configuration in the Models

Deferred 2026-10-05. Evidence: `.plans/DYNAMIC_CONFIG.md`.

| Task | Status | What |
| --- | --- | --- |
| fn-125.1 | ✅ done | HSM/CHASM switch fixed; schedule-to-close no longer from the Profile |
| fn-125.2 | ⏸️ deferred | `setting[T]` over finite domains in the framework, lifted |
| fn-125.3 | ⏸️ deferred | Query `under`: one Query per valuation |
| fn-125.4 | ⏸️ deferred | Settings in the Quint/P exports |
| fn-125.5 | ⏸️ deferred | Dynamic-config keys declared once in the kit, pinned to the server registry |
| fn-125.6 | ⏸️ deferred | API preconditions; derived required settings; ShutdownWorker precondition |
| fn-125.7 | ⏸️ deferred | Nexus implementation encoded; one Case per valuation; switch retired |
| fn-125.8 | ⏸️ deferred | Caller attempt semantics as the owner chooses |
| fn-125.9 | ⏸️ deferred | Bound assumptions on server durations checked at preparation |
| fn-125.10 | ⏸️ deferred | Disposition for every implicit assumption |
| fn-125.11 | ⏸️ deferred | Docs; close |

### fn-130: Model views

Deferred 2026-10-05 before task planning; the spec has no tasks yet. When revived, starts after fn-126 closes. Evidence: `.plans/MODEL_VISUALIZATION.md`.

Rendered views per Model (signature, phase diagram, refinement, compositions, derived-design diff, witness paths), checked in as `.d2` plus `.svg` under `model/views/` and gated; D2 as a Go library with ELK; no DSL declaration.

### Other deferred items

- fn-122.7 (the Pausable capability Property on fn-119's example) waits for fn-119.
- `make umpire-check-backends` in CI; it runs locally after `make umpire-install-backends`.
- The IR explorer (fn-120.4) was removed. fn-112, fn-114, fn-118, fn-120, fn-121, fn-122 and fn-127 are closed.

## Open for the owner

- 691 lint findings accepted with reasons in `model/ir/*.lint.json` (fn-120.3); review the H2 reasons first.
- Behavior-freeze follow-ups from fn-112: the witness-only Properties `terminated` and
  `cancelRequestedWhileStarted`, and seven pause/unpause rows the server rejects. fn-140 deletes
  `terminated` with its Query (R6) and turns `cancelRequestedWhileStarted` into a witness (R5).
- Whether HSM and CHASM may count Nexus `attempt` differently; no current Query shows a difference (fn-125, deferred).
- Whether upstream's Go conformance harness (`tests/activity_driver.go`) should run our IR through the Go interpreter instead of its hand-written model, making one Model drive both (`.plans/ACTIVITY_MODEL_COMPARISON.md` P3-12); needs the owning team. A workflow-scheduled activity realization (P3-11) overlaps the deferred fn-119.
- The canary policy's `workflowPath` names the deleted production-canary workflow, so production dispatch
  fails closed.
