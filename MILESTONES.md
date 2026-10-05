# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-05.

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

The DSL framework (`model/umpire`) stays Temporal-agnostic as far as is realistic: Temporal's capability
vocabulary, laws, realization vocabulary and kit live under `model/temporal/`, and the lifter and Testpilot IR
are the parts that are Temporal's driver tooling by design (fn-114.12, fn-122.8).

The planned work below continues that direction: the Scala layer first, then the Models.

## Specs

Listed in delivery order. Flow records dependencies only within a spec, so each spec names the
cross-spec gates the conductor holds.



### fn-126: Read each feature top to bottom: one object per machine

Gate: never alongside fn-124.8; closes before fn-124.7 and before fn-125 resumes.

| Task | Status | What |
| --- | --- | --- |
| fn-126.1 | ✅ done | Standalone activity as one feature file per folder; `record/`, `withTaskQueue/`; declaration-order lint |
| fn-126.2 | ⬜ todo | Nexus folders and shared Models as feature files; shared bounds; per-kind file names retired |
| fn-126.3 | ⬜ todo | Actions grouped by actor in section objects that keep Definition IDs |
| fn-126.4 | ⬜ todo | Machine objects with effects, rules and sections; lifter reshaped; standalone activity converted |
| fn-126.5 | ⬜ todo | Remaining Models as machine objects; builder forms retired |
| fn-126.6 | ⬜ todo | One rename batch: Product and System, history record, actions and designs; close |

### fn-124: Shrink and simplify the Umpire Go tooling

Gates: task 7 after fn-126 closes (fn-114, fn-120 and fn-122 are closed); task 8 last, never alongside fn-126.

| Task | Status | What |
| --- | --- | --- |
| fn-124.1 | ✅ done | `tools/umpire0`, `model0` and the empty command deleted |
| fn-124.2 | ✅ done | Duplicate refinement and test-only APIs removed |
| fn-124.3 | ✅ done | Temporal facts the judge hard-codes declared in the realization |
| fn-124.4 | ✅ done | Verdict aggregation defined once; judge rules documented |
| fn-124.5 | ✅ done | Generated-Case outcomes compared by declared ids |
| fn-124.6 | 🔄 in progress | Model assessment as the command-line judge; Evaluation Profile derived or retired |
| fn-124.7 | ⬜ todo | Migration harness and frozen snapshots retired |
| fn-124.8 | ⬜ todo | `tools/umpire/model` split into `ir`, `interp`, `check`, `realization` |


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
- The canary policy's `workflowPath` names the deleted production-canary workflow, so production dispatch
  fails closed.
