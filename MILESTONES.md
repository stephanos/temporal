# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-06.

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
vocabulary, laws, realization vocabulary and kit live under `model/temporal/`, and the lifter and Testpilot IR
are the parts that are Temporal's driver tooling by design (fn-114.12, fn-122.8).

The planned work below continues that direction: the Scala layer first, then the Models.

## Specs

Listed in delivery order. Flow records dependencies only within a spec, so each spec names the
cross-spec gates the conductor holds.

### DSL batch

fn-132 closed 2026-10-06 (commit 96de1fd92d is its tree). The DSL work of fn-133, fn-134, fn-135, fn-136, fn-137 and fn-139 runs as one batch
instead of one regeneration per task. Tasks write and commit their Scala in the order below, with no
`make umpire-gen-model` and no full gates in between. The tree may be red between tasks. The batch then
regenerates once, checks the diff, and runs the full Go suite, the model gate and reviews once. Batched
2026-10-06 (owner decision) to cut repeated regenerations and full runs; the Model work that changes what
the activity says (fn-128, fn-138, fn-129) is deferred until the batch closes.

- **Within the batch.** A framework or lifter task still runs its own fixtures and munit tests, which are
  cheap and need no regeneration. A Model task needs no regeneration and no gate. Each task is its own
  commit, so an unexplained difference can be bisected with `make umpire-gen-model`; prefer leaving the
  tree compiling so bisection can test each commit. No regeneration of any kind runs during the batch,
  so each spec's single-regeneration rule (fn-134.3, fn-135.4) becomes the batch's regeneration.
- **The diff check.** Baseline: the tree at fn-132's close, `96de1fd92d`. The expected IR change is the union of the
  deltas the tasks declare: source positions; fn-133.3, .5 and .6 renames and ID map, fn-133.4's listed new
  Cases and fn-133.8's carrier metadata; fn-134's `origin` and the deleted `*.laws.json`; fn-135's `status`;
  fn-136.5's inlined Functions; fn-137.6's removed `states.terminal` Functions and fn-137.7's
  owner-approved change; fn-139's renamed actions, classes, inputs and outcomes and the other differences
  its R7 lists. Classify every changed line against that list; any line it does not explain stops the
  batch until it is traced to its task. Per-task "equal but for positions" claims are checked against the
  baseline, never absorbed. Every Query answer and receipt is otherwise unchanged.
- **Order.**
  1. Framework and lifter, beside the old forms: fn-134.1, fn-134.2, fn-135.1–.3, fn-135.5, fn-136.1,
     fn-136.4 (spelled `when[R]`, since fn-139 renames the rule-case `in`), fn-137.1–.2, fn-139.1–.3.
  2. Realizations: fn-133.1–.6, then fn-133.8 (IR carrier metadata and its Go consumers).
  3. Models, in this order because they share `Product.scala`, `System.scala`, `Record.scala`,
     `WithTaskQueue.scala` and the Nexus System and Product: fn-135.4 → fn-136.2, .3 → fn-134.3 →
     fn-137.3, .4 → fn-136.5 → fn-137.5, .6 → fn-137.7 (ask the owner about `pausedWhileHeld` before
     writing it) → fn-139.4–.7.
  4. Removals and Go: fn-134.4, then the batch regeneration, then fn-134.5 (Go refuses law sidecars, so
     it follows the regeneration that deletes them) and fn-139.8 (rejection-to-RPC-code table and
     conformance).
  5. Gates: the full Go suite, the model gate and reviews once, then one live run for fn-133.4's new Cases
     and fn-139.8's conformance check. Close fn-133 (task 7), fn-134, fn-135, fn-136, fn-137 and fn-139
     after it.

### fn-133: Lean, typed realizations

Gate: the DSL batch. Source: review of the realizations, 2026-10-05. Tasks run in order.

| Task | Status | What |
| --- | --- | --- |
| fn-133.1 | 🔄 in progress | `proto[T] { … }` literal scope, response reads in the call scope, literal helpers |
| fn-133.2 | 🔄 in progress | Kit evidence modules: described status, history evidence, request base |
| fn-133.3 | 🔄 in progress | Lower-case instruction forms, no borrowed command names, named evidence arguments, ids from facts |
| fn-133.4 | 🔄 in progress | Coverage report, class-pattern rule, `deadlines(…)` binding, action-level `onPath` |
| fn-133.5 | 🔄 in progress | Name collisions removed; activity realization local fixes |
| fn-133.6 | 🔄 in progress | Typed realization objects; derived operation, roles, server steps; derived realizations |
| fn-133.8 | 🔄 in progress | Per-class carrier schemas derived from typed realizations |
| fn-133.7 | ⬜ todo | Line counts, docs; close |

### fn-134: Capabilities own their properties

Gate: the DSL batch; closes before fn-131 starts. Source: owner conversation, 2026-10-06. Tasks run in the batch order. Task 3's equivalence harness takes the batch baseline, and its regeneration is the batch's.

| Task | Status | What |
| --- | --- | --- |
| fn-134.1 | ✅ done | Inert `Property.origin` in the IR schema and Go reader; identity test |
| fn-134.2 | ✅ done | `capabilities` section, capability Properties, bounds in `queries`, waiver reasons from the model gate, added beside the old path (early proof) |
| fn-134.3 | ⬜ todo | Kit and every Model migrated; equivalence diff at the batch regeneration |
| fn-134.4 | ⬜ todo | `Law`, `Catalog`, `Implements`, `cited` and the law sidecar removed from Scala |
| fn-134.5 | ⬜ todo | Law sidecar reader, law table and law lint kinds removed from Go (after the batch regeneration) |
| fn-134.6 | ⬜ todo | Docs, vocabulary check; close |

### fn-135: `effect { }` and `is { }` blocks for effects and predicates

Gate: the DSL batch. Tasks run in order. Task 4's comparison takes the batch baseline. Amended 2026-10-06: `ActivityProduct`'s phase cases declare their status facts (R8–R10). Parked: renaming `Phase` to `Status` across every machine.

| Task | Status | What |
| --- | --- | --- |
| fn-135.1 | ✅ done | `effect`, `is`, `record` and `reject(outcome)` sugar in umpire; run-time equivalence tests |
| fn-135.2 | ✅ done | Lifter resolves `val` section members and lifts `is { }`; `def` paths unchanged (proof point) |
| fn-135.3 | ✅ done | Lifter lifts `effect { }` with its statement refusals |
| fn-135.5 | 🔄 in progress | Status facts declared on phase cases: derived in `effect { }` at run time and in lifted IR, with refusals and a fixture machine |
| fn-135.4 | 🔄 in progress | `ActivityProduct` converted, projection renamed `status`; IR equal but for positions and that name; docs |

### fn-136: Phase roles on lifecycle enums

Gate: the DSL batch. Tasks run in the batch order. R7 (positions only, for tasks 1 to 3) and R9 (task 5's bounded change: retired predicate Functions removed and their case sets inlined, Query answers unchanged) are checked at the batch regeneration.

| Task | Status | What |
| --- | --- | --- |
| fn-136.1 | 🔄 in progress | Role traits; lifter lowers role tests (`isInstanceOf`, type patterns) to case-set membership and refuses conflicting roles |
| fn-136.2 | 🔄 in progress | Activity product, system and record carry roles; `states` bodies become role tests; refinement closedness check |
| fn-136.3 | 🔄 in progress | Nexus workflow, standalone and product carry roles; docs |
| fn-136.4 | 🔄 in progress | `in[R]` and `p.is[R]` role-test spellings; lifter lowers them like `in(...)` and `isInstanceOf` |
| fn-136.5 | ⬜ todo | Role-set `states` predicates retire, callers read roles directly; bounded IR change (R9); docs; close |

### fn-137: Capabilities read phase roles

Gate: the DSL batch. Task 7 asks the owner about `pausedWhileHeld` before it is written. Where a capability's predicate differs from the role it now reads, the batch diff shows it and the owner resolves it; it is never absorbed.

| Task | Status | What |
| --- | --- | --- |
| fn-137.1 | 🔄 in progress | `Phased` mixin; argument-less `Rules` reads it beside the old form (early proof) |
| fn-137.2 | 🔄 in progress | Lifter reads the projection from the `Phased` parent |
| fn-137.3 | ⬜ todo | Every Model migrated to `Phased` and argument-less `Rules` |
| fn-137.4 | ⬜ todo | `Rules(projection)` retired: framework, lifter fallback, fixtures, docs |
| fn-137.5 | ⬜ todo | Default `end` for `Phased` objects (needs fn-136's `Closed` role) |
| fn-137.6 | ⬜ todo | Closable reads the `Closed` role through `Phased` (needs fn-134, fn-136.5) |
| fn-137.7 | ⬜ todo | Pausable reads `Suspended` and `Held`; `pausedWhileHeld` settled; capability docs |

### fn-139: Actor-grouped rules, per-RPC actions, shared rejections

Gate: the DSL batch; task 8 follows the batch regeneration. Captured 2026-10-06. fn-136.4 spells the role form `when[R]` directly.

`from(actor) { on(action) { when(phases) ~> effect } }` rules, `when` replacing the rule-case `in`, one action per RPC for the standalone activity with a `Failure` enum, and a shared `Outcome`/`Rejection` with `rejects(r)` rows. R7: the IR differs only in positions, the renamed actions, classes and inputs, and the other differences R7 lists.

| Task | Status | What |
| --- | --- | --- |
| fn-139.1 | 🔄 in progress | Shared `Outcome`/`Rejection` and `rejects(r).because(text)` in framework and lifter; parameterized outcome admitted by the Go reader (early proof) |
| fn-139.2 | 🔄 in progress | Framework: `from(declarer)` with leading import, `when` case forms, multi-action `on`, overlap across blocks, beside the old forms |
| fn-139.3 | 🔄 in progress | Lifter reads `from`/`when`/multi-action `on`; block-form rule in the model gate |
| fn-139.4 | ⬜ todo | Standalone activity: one action per RPC with a `Failure` enum; every consumer on the new actions |
| fn-139.5 | ⬜ todo | Activity product and System rules in `from` blocks, grouped by meaning; activity on the shared `Outcome` |
| fn-139.6 | ⬜ todo | Nexus and the shared worker on the shared `Outcome`; `alreadyCompleted` becomes `rejected(failedPrecondition)` |
| fn-139.7 | ⬜ todo | `in` → `when` and block form across every Model and fixture; rule-case `in` retired; block-form lint on; docs |
| fn-139.8 | ⬜ todo | Rejection-to-RPC-code table and conformance check |

### fn-142: Split `model/temporal/shared` into `foundations` and `actors`

Gate: after the DSL batch closes. Mechanical move: the task queue goes to `foundations/taskqueue`, the worker to `actors/worker`, `Client.scala` to `actors/client/`, and `Bounds.scala` to `model/temporal/`; the IR differs only in paths and positions.

| Task | Status | What |
| --- | --- | --- |
| fn-142.1 | ⬜ todo | Move, regenerate, paths-only proof, docs |

### fn-143: Rename `model/umpire` to `model/framework`

Gate: after fn-142. Folder and package both become `framework`; product names (`tools/umpire`, `umpire-*` targets, `umpire.v1`) stay. The IR differs only in paths and positions.

| Task | Status | What |
| --- | --- | --- |
| fn-143.1 | ⬜ todo | Move, rename the package, regenerate, paths-only proof, docs |

## Deferred

Specs the owner deferred as not needed for the current code deliverable (the DSL and its execution).
They keep their tasks so they can be revived as planned.

### fn-128: Close the activity's precision gaps

Deferred 2026-10-06 so the DSL batch runs first. When revived, fn-128, fn-138 and fn-129 run as one batch after the DSL batch closes, by the same rules: one regeneration, one gate run, and one live run serving fn-128.6 and fn-129.5. Source: `.plans/ACTIVITY_MODEL_COMPARISON.md`. Tasks run in order; each declares its IR change, checked at the batch regeneration.

| Task | Status | What |
| --- | --- | --- |
| fn-128.1 | ⏸️ deferred | Dispatch as a field replacing the `backingOff` phase; start delay; unpause-after-backoff and schedule-to-start-in-backoff fixed |
| fn-128.2 | ⏸️ deferred | Rejections as rows (`failedPrecondition`, `invalidArgument`); repeated RequestCancel; `silent-rejection` acceptances removed |
| fn-128.3 | ⏸️ deferred | Retry policy: `maxAttempts` input, `retriesRemaining`, retryable start-to-close timeout |
| fn-128.4 | ⏸️ deferred | Stutter facts checked: `visible` on `ActivitySystem`'s refinement |
| fn-128.5 | ⏸️ deferred | `cancelIsNotUndone` Property; attempt count in every Case; time-window `because` |
| fn-128.6 | ⏸️ deferred | Evidence map, live Cases run once (the deferred batch's live run); close |

### fn-138: Retries and Deadline capabilities

Deferred 2026-10-06 with fn-128. When revived, runs after fn-128.5 (fn-128.1 replaces the `backingOff` phase and fn-128.3 adds the retry policy, both of which Retries reads). Open questions for the owner are in the spec; tasks .1 and .2 ask them before writing Properties.

Retries (a retryable failure of a `Held` attempt lands in `Waiting`, a non-retryable one in `Failed`, attempts never exceed the bound) and Deadline capabilities, declared by the activity and Nexus workflow systems; their timer windows become role tests.

| Task | Status | What |
| --- | --- | --- |
| fn-138.1 | ⏸️ deferred | Retries capability: kit Properties and lifter refusals (owner question first) |
| fn-138.2 | ⏸️ deferred | Deadline capability: kit Properties, per-declaration names and lifter refusals |
| fn-138.3 | ⏸️ deferred | Activity and Nexus workflow systems declare Retries and Deadlines; timer windows as role tests; docs |

### fn-129: Activity coverage

Deferred 2026-10-06 with fn-128. When revived, runs after fn-138. Tasks run in order; each declares its IR change, checked at the batch regeneration.

| Task | Status | What |
| --- | --- | --- |
| fn-129.1 | ⏸️ deferred | Heartbeat action and retryable heartbeat timeout; realized |
| fn-129.2 | ⏸️ deferred | Respond by ID as a `service` actor; realized |
| fn-129.3 | ⏸️ deferred | Reset with `keepPaused` and deferred apply; precedence Property extended; realized |
| fn-129.4 | ⏸️ deferred | Exploration on the activity's `find` Queries |
| fn-129.5 | ⏸️ deferred | New Cases listed, live run (the deferred batch's live run); close |

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

### fn-123: Declare faults as the environment's actions

Deferred 2026-10-04 before task planning; the spec has no tasks yet.

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

- fn-122.7 (Pausable on fn-119's example) waits for fn-119; fn-122 itself is closed.
- `make umpire-check-backends` in CI; it runs locally after `make umpire-install-backends`.
- The IR explorer (fn-120.4) was removed. fn-112, fn-114, fn-118, fn-120, fn-121, fn-122 and fn-127 are closed.

## Open for the owner

- The matching ShutdownWorker race (`service/matching/matching_engine.go`, upstream #9424) makes the Cases
  that stop a worker INCONCLUSIVE under both Nexus implementations; with
  `frontend.enableMatchingFanOutForPollCancellation=false` they pass. The planned `workerStop` precondition
  (fn-125.6) is deferred with fn-125, so these Cases stay INCONCLUSIVE until the server is fixed or fn-125 resumes.
- 691 lint findings accepted with reasons in `model/ir/*.lint.json` (fn-120.3); review the H2 reasons first.
- Behavior-freeze follow-ups from fn-112: the witness-only Properties `terminated` and
  `cancelRequestedWhileStarted`, and seven pause/unpause rows the server rejects.
- Whether HSM and CHASM may count Nexus `attempt` differently; no current Query shows a difference (fn-125, deferred).
- Whether upstream's Go conformance harness (`tests/activity_driver.go`) should run our IR through the Go interpreter instead of its hand-written model, making one Model drive both (`.plans/ACTIVITY_MODEL_COMPARISON.md` P3-12); needs the owning team. A workflow-scheduled activity realization (P3-11) overlaps the deferred fn-119.
- The canary policy's `workflowPath` names the deleted production-canary workflow, so production dispatch
  fails closed.
