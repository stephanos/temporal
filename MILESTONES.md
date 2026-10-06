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
  suites with one `flock` lock file. After any Model or lifter change, run `make umpire-gen-model`,
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



### fn-132: Group the Nexus and activity Models by kind: workflow and standalone

Gate: starts after fn-126 and fn-124 close; closes before fn-128 starts. Source grouping first; tasks 1 and 2 never run at the same time. Focused checks per checkpoint, shared full validation after both Part A moves.

| Task | Status | What |
| --- | --- | --- |
| fn-132.8 | ✅ done | Kind/form source grouping admitted before either move; unchanged-tree proof |
| fn-132.1 | ✅ done | `features/nexus/workflow` and `features/nexus/standalone`; all four export stems moved, exact identity proof and focused checks passed; broad Part A gates at task 2 |
| fn-132.2 | ✅ done | `features/activity/standalone`; three form-owned export stems; exact move proof, closing Part A gates and SHIP; inherited matching worker-stop live exception retained |
| fn-132.3 | 🔄 in progress | Structure lint and docs learn the kind level |
| fn-132.4 | ⬜ todo | Spike: one action shared by two forms' entities; outcomes; `terminated` |
| fn-132.5 | ⬜ todo | One `NexusProduct` in `features/nexus/` that both Nexus forms refine |
| fn-132.6 | ⬜ todo | General activity declarations in `activity/Activity.scala` |
| fn-132.7 | ⬜ todo | Close |

### fn-133: Lean, typed realizations

Gate: starts after fn-126 and fn-132 close; closes before fn-128 starts. Source: review of the realizations, 2026-10-05. Tasks run in order (each regenerates `model/ir`).

| Task | Status | What |
| --- | --- | --- |
| fn-133.1 | ⬜ todo | `proto[T] { … }` literal scope, response reads in the call scope, literal helpers |
| fn-133.2 | ⬜ todo | Kit evidence modules: described status, history evidence, request base |
| fn-133.3 | ⬜ todo | Lower-case instruction forms, no borrowed command names, named evidence arguments, ids from facts |
| fn-133.4 | ⬜ todo | Coverage report, class-pattern rule, `deadlines(…)` binding, action-level `onPath` |
| fn-133.5 | ⬜ todo | Name collisions removed; activity realization local fixes |
| fn-133.6 | ⬜ todo | Typed realization objects; derived operation, roles, server steps; derived realizations |
| fn-133.8 | ⬜ todo | Per-class carrier schemas derived from typed realizations |
| fn-133.7 | ⬜ todo | Line counts, docs; close |

### fn-128: Close the activity's precision gaps

Gate: starts after fn-126, fn-132 and fn-133 close. Source: `.plans/ACTIVITY_MODEL_COMPARISON.md`. Tasks run in order (each regenerates `model/ir`).

| Task | Status | What |
| --- | --- | --- |
| fn-128.1 | ⬜ todo | Dispatch as a field replacing the `backingOff` phase; start delay; unpause-after-backoff and schedule-to-start-in-backoff fixed |
| fn-128.2 | ⬜ todo | Rejections as rows (`failedPrecondition`, `invalidArgument`); repeated RequestCancel; `silent-rejection` acceptances removed |
| fn-128.3 | ⬜ todo | Retry policy: `maxAttempts` input, `retriesRemaining`, retryable start-to-close timeout |
| fn-128.4 | ⬜ todo | Stutter facts checked: `visible` on `ActivitySystem`'s refinement |
| fn-128.5 | ⬜ todo | `cancelIsNotUndone` Property; attempt count in every Case; time-window `because` |
| fn-128.6 | ⬜ todo | Evidence map, live Cases run once; close |

### fn-129: Activity coverage

Gate: starts after fn-128 closes. Tasks run in order (each regenerates `model/ir`).

| Task | Status | What |
| --- | --- | --- |
| fn-129.1 | ⬜ todo | Heartbeat action and retryable heartbeat timeout; realized |
| fn-129.2 | ⬜ todo | Respond by ID as a `service` actor; realized |
| fn-129.3 | ⬜ todo | Reset with `keepPaused` and deferred apply; precedence Property extended; realized |
| fn-129.4 | ⬜ todo | Exploration on the activity's `find` Queries |
| fn-129.5 | ⬜ todo | New Cases listed, live run; close |

## Deferred

Specs the owner deferred as not needed for the current code deliverable (the DSL and its execution).
They keep their tasks so they can be revived as planned.

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
