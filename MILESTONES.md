# Umpire4 milestones

The current state of Umpire work: what is being built, what is left, and what is not being done.
Flow (`.flow/`, `flowctl`) is the record for specs and tasks; this page is the overview across them.

As of 2026-10-05.

## Keeping this page current

- This page describes the present. Rewrite a status in place; do not append dated entries.
- List each open spec's tasks with ID, status and a brief description; set a task's status in place
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

### fn-114: State every Scala Model declaration once

Gate: task 9 runs after fn-118.2 and fn-120.3 have merged (both done); task 8 closes the spec.

| Task | Status | What |
| --- | --- | --- |
| fn-114.1 | done | IR-file roots declared in Scala, one lifter run |
| fn-114.2 | done | Nexus caller Model restated (captured names, derivation, four files) |
| fn-114.3 | done | Nexus caller realization by value with the shared kit |
| fn-114.4 | done | Close-policy declarations in the final DSL (`sticky`) |
| fn-114.5 | done | Close-policy Model split into the four-file layout |
| fn-114.6 | done | Worker Model and lifter fixtures restated |
| fn-114.7 | done | String-named declaration forms retired |
| fn-114.10 | done | Lifter fixtures reduced to fixture-local Models |
| fn-114.11 | done | Redundant type annotations dropped |
| fn-114.12 | done | `model/umpire` Temporal-agnostic, guarded by a test |
| fn-114.9 | todo | Rename tool folders (`irgen`, `check`, `build`); group `model/temporal` into `features/` and `shared/` |
| fn-114.8 | todo | Close: counts and full gates |

### fn-118: Declare how Temporal APIs behave once

Gate: started before fn-114 closed (its remaining tasks are cleanup).

| Task | Status | What |
| --- | --- | --- |
| fn-118.1 | done | Wait inventory and hint-aware helper interface |
| fn-118.2 | done | Hints declared in the kit, lifted to the IR, validated in Go |
| fn-118.3 | done | Testpilot waits by condition within declared bounds |
| fn-118.4 | todo | Lowering derives Case waits from hints; undeclared visibility refused |
| fn-118.5 | todo | Realizations migrated to derived waits; close |

### fn-120: Adopt what Quint does well

| Task | Status | What |
| --- | --- | --- |
| fn-120.1 | done | Named choices, inert IR names, Quint export |
| fn-120.2 | done | Unnamed branching refused; `choose` accepts helper calls |
| fn-120.3 | done | Model lint, specification holes, coverage summary, accepted findings |
| fn-120.4 | deferred | IR explorer |
| fn-120.5 | todo | Close |

### fn-122: Capabilities and their laws

| Task | Status | What |
| --- | --- | --- |
| fn-122.1 | done | Law bodies and catalog with server citations |
| fn-122.2 | done | `capabilities` declaration, `except`/`overriding`, lifting |
| fn-122.3 | done | Activity capabilities; authored twins retired |
| fn-122.4 | done | Standalone Nexus operation Model; required settings |
| fn-122.8 | done | Capability vocabulary moved to `model/temporal/capabilities` |
| fn-122.5 | in progress | Law lint kinds; waiver reasons in the accepted-findings file |
| fn-122.6 | todo | Docs, authored vs generated counts; close |
| fn-122.7 | deferred | Pausable on fn-119's example |

### fn-124: Shrink and simplify the Umpire Go tooling

Gates: task 3 after fn-118; task 7 after fn-114, fn-120 and fn-122 close; task 8 last.

| Task | Status | What |
| --- | --- | --- |
| fn-124.1 | done | `tools/umpire0`, `model0` and the empty command deleted |
| fn-124.2 | done | Duplicate refinement and test-only APIs removed |
| fn-124.3 | todo | Temporal facts the judge hard-codes declared in the realization |
| fn-124.4 | todo | Verdict aggregation defined once; judge rules documented |
| fn-124.5 | todo | Generated-Case outcomes compared by declared ids |
| fn-124.6 | todo | Model assessment as the command-line judge; Evaluation Profile derived or retired |
| fn-124.7 | todo | Migration harness and frozen snapshots retired |
| fn-124.8 | todo | `tools/umpire/model` split into `ir`, `interp`, `check`, `realization` |

### fn-125: Represent dynamic configuration in the Models

Gates: tasks 2-3 after fn-114 closes, not alongside fn-124.8; tasks 5-6 after fn-118.2 (done); task 7 after
fn-118.5; task 9 after fn-118.4. Evidence: `.plans/DYNAMIC_CONFIG.md`.

| Task | Status | What |
| --- | --- | --- |
| fn-125.1 | done | HSM/CHASM switch fixed; schedule-to-close no longer from the Profile |
| fn-125.2 | todo | `setting[T]` over finite domains in the framework, lifted |
| fn-125.3 | todo | Query `under`: one Query per valuation |
| fn-125.4 | deferred | Settings in the Quint/P exports |
| fn-125.5 | todo | Dynamic-config keys declared once in the kit, pinned to the server registry |
| fn-125.6 | todo | API preconditions; derived required settings; ShutdownWorker precondition |
| fn-125.7 | todo | Nexus implementation encoded; one Case per valuation; switch retired |
| fn-125.8 | todo | Caller attempt semantics as the owner chooses |
| fn-125.9 | todo | Bound assumptions on server durations checked at preparation |
| fn-125.10 | todo | Disposition for every implicit assumption |
| fn-125.11 | todo | Docs; close |

## Deferred

Deferred by the owner on 2026-10-04 as not needed for the code deliverable (the DSL and its execution):
fn-119 (Go SDK workflow showcase; tasks 1-2 done, 3-6 blocked), fn-122.7, fn-123 (faults as environment
actions, not planned), fn-120.4 and fn-125.4. Also deferred: `make umpire-check-backends` in CI (it runs
locally after `make umpire-install-backends`). fn-112 and fn-121 are closed.

## Open for the owner

- The matching ShutdownWorker race (`service/matching/matching_engine.go`, upstream #9424) makes the Cases
  that stop a worker INCONCLUSIVE under both Nexus implementations; with
  `frontend.enableMatchingFanOutForPollCancellation=false` they pass. fn-125.6 declares that as a `workerStop`
  precondition and drafts the upstream report.
- 691 lint findings accepted with reasons in `model/ir/*.lint.json` (fn-120.3); review the H2 reasons first.
- Behavior-freeze follow-ups from fn-112: the witness-only Properties `terminated` and
  `cancelRequestedWhileStarted`, and seven pause/unpause rows the server rejects.
- Whether HSM and CHASM may count Nexus `attempt` differently (fn-125.8); no current Query shows a difference.
- The canary policy's `workflowPath` names the deleted production-canary workflow, so production dispatch
  fails closed.
