---
satisfies: [R3, R4]
---
# fn-80-close-the-model-to-case-seam-and-harden.8 Author the worker-outage fault Case and its live test

## Description
Implements the acceptance Case for R4 and the checked-in `rule_events` horizon for R3. A Lean-authored, Producer-neutral functional fixture stops the task-queue worker before start-workflow, resumes after a bounded wait, and asserts the ordered `FAULT_INJECTED` events plus correlated success. Also lowers a Space fault intent to the same instruction.

**Size:** M
**Files:** `model/Temporal/Testpilot/WorkerOutage.lean` (new), `model/Temporal/Testpilot.lean`, `model/Temporal/Tool/Testpilot.lean`, `model/Umpire/Space/Language.lean` or new `model/Umpire/Space/Lowering.lean`, `model/Umpire/Space/Tests/*.lean`, `tests/testcore/testpilot/testdata/worker-outage-case.json` (generated), `tools/umpire/cmd/umpire-gen-case-runtime-conformance` manifest, `tests/testpilot_worker_outage_case_test.go` (new), `tests/testcore/testpilot/README.md`
**Touches:** [model/Temporal/Testpilot/**, model/Temporal/Testpilot.lean, model/Temporal/Tool/Testpilot.lean, model/Umpire/Space/**, tests/testcore/testpilot/**, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tests/testpilot_worker_outage_case_test.go]

### Approach
- Author the Program as a copy of the async-nexus shape (workflow + Nexus handler entrypoints) with controller nodes: `injectFault WORKER_STOP` on the task-queue role, `start-workflow`, `await` bounded by the instruction timeout, `injectFault WORKER_RESUME`, then the existing completion and history nodes. Follow `model/Temporal/Testpilot/GetSystemInfo.lean` and `Conformance.lean` for the Producer-neutral shape.
- Contract: one classic rule with a `rule_events` horizon (`Monitor.horizonEvents`) whose transitions filter on `RUN_EVENT_KIND_FAULT_INJECTED` and compare `FAULT_KIND` fields for stop-then-resume order, plus the correlated history success rule.
- Register the fixture name in `model/Temporal/Tool/Testpilot.lean:21-24` and the generator manifest; regenerate with `make umpire-gen-case-runtime-conformance`; the functional tree is under `tests/testcore/testpilot/testdata`, not the six-class conformance tree.
- `FaultIntentDeclaration.lower` (`Umpire/Space/Language.lean:68-88`) returns the `InjectFault` instruction definition for a worker-stop intent; `#guard` it equals the fixture's stop node. Keep `Umpire/Exploration/Coverage.lean:13,46` wording.
- Live test uses `RunCase` and the derived Profile; run two Runs concurrently, one with and one without the fault Case on the same queue, asserting the plain Run is unaffected.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Testpilot/GetSystemInfo.lean` and `Conformance.lean` — Producer-neutral Case shape
- `model/Temporal/Feature/Nexus3/Testpilot.lean` (post task .4) — Program realization to copy
- `model/Umpire/Space/Language.lean:66-88,767-830` — fault intents and their checking
- `tests/testpilot_async_nexus_case_test.go` (post task .7) — live test pattern with `RunCase`

**Optional** (reference as needed):
- `model/Umpire/Artifact/Planning.lean:32-40,112-156` — fault intent validation
- `Makefile` — the fixture gates and the `umpire-check-live-tests` target (fn-81 retired its pinned expected-failure list; the gate now selects `^TestTestpilot`, compares against an empty baseline, and requires at least one passing identity)

### Key context
- The stop precedes `start-workflow` so no workflow task is in flight when the worker stops; the reservation ledger mints handles independently of SDK polling (verify; adjust ordering if not).
- EVD-18's six conformance classes stay exact; this fixture is functional, like `get-system-info`.

## Deferred acceptance

**"Negative live case: resume timeout yields cleanup `failed` with the Verdict unchanged" is
deferred, and this is the record of that decision.** The shipped outage Case always resumes, and the
live environment supplies the real SDK worker factory to the Driver's registry, so nothing a test
can reach makes a live resume fail. Reaching it live would need either a worker-factory seam on the
live test environment, or a second functional fixture that stops and never resumes -- a Case whose
own Contract could not be satisfied.

Both halves are pinned where they are reachable today:
- the Driver surfacing a failed resume from `Session.Close`, and releasing the hold either way:
  `common/testing/testpilot/temporal/worker/fault_test.go`
  (`TestSessionCloseResumesAndAlwaysReleasesTheHold`, task .3);
- a failed cleanup leaving an already-reached Verdict alone: the
  `cleanup-failure-after-proved-violation` conformance class (EVD-18).

Follow-up: a worker-factory override on the live test environment would close it live.

Also deferred, with the same reasoning recorded in the done summary: the acceptance's "concurrent
plain async-nexus Run on the same queue" is run on a *different* queue, because a pooled peer worker
on the same physical queue keeps polling through the outage.

## Acceptance
- [ ] `worker-outage-case.json` is generated deterministically and `make umpire-check-case-runtime-conformance` passes on both trees
- [ ] Live test (integration tag): Run completed, cleanup succeeded, Verdict satisfied, two `FAULT_INJECTED` events in stop-then-resume order referenced by the Contract's supporting sequences, correlated history evidence present
- [ ] Concurrent plain async-nexus Run on the same queue is unaffected (satisfied) while the fault Run stops its dedicated worker
- [ ] Negative live case: resume timeout yields cleanup `failed` with the Verdict unchanged
- [ ] `FaultIntentDeclaration.lower` `#guard` equals the fixture's stop instruction; `Umpire/Exploration/Coverage.lean` wording preserved
- [ ] The classic rule declares a `rule_events` horizon and the offline `PreparedContract.Evaluate` over the recorded Run agrees with the live Verdict
- [ ] `make umpire-check-live-tests` passes

## Done summary
Blocked:
BLOCKED: EXTERNAL_BLOCKED — four of this task's seven acceptance bullets need a live cluster this
session cannot stand up, and no code was written for it.

Started and stood down without edits: the tree is exactly as task .7 left it. Nothing is
half-implemented.

Why it stopped here rather than landing the offline half: the acceptance is live-weighted.
  - live Run completed, cleanup succeeded, Verdict satisfied, two ordered `FAULT_INJECTED` events
  - a concurrent plain async-nexus Run on the same queue unaffected
  - the negative resume-timeout case setting cleanup `failed`
  - `make umpire-check-live-tests` passing
all require `go test -tags 'test_dep integration' ./tests`, which needs a running test cluster.
This session has ~5 GiB of disk, which the whole-server build plus a cluster does not fit. Landing
only the fixture bytes would put a generated Case in the tree that nothing has ever executed —
exactly the vacuous-evidence failure task .4 was blocked to avoid.

What is ready for whoever picks it up, from the work that did land:

1. **The Driver half is done and unit-tested** (task .3). `Session.InjectFault` realizes
   `WORKER_STOP`/`WORKER_RESUME` on a dedicated worker group keyed by Run ID, suppresses the SDK
   fatal path for the outage window, resumes before release, and reports an unrealized transition
   as a `fault_not_realized` outcome plus a Driver invariant diagnostic.

2. **Two constraints the Case must respect**, both discovered while landing .2 and .3:
   - `FAULT_INJECTED` is recorded **only on a succeeded outcome** (`scheduler.go`), so the Case's
     Contract may treat the event as evidence that the outage actually happened.
   - A dedicated group isolates the stop from peer Runs, but **a pooled peer worker on the same
     physical task queue keeps polling it**. The Case's fault queue therefore needs its own
     resource binding, and .8's "concurrent plain Run unaffected" assertion should put the plain
     Run on a *different* queue, or the outage is not real for either Run.

3. **The Profile is derived, not hand-written** (task .7). The live test should use `bindCase` /
   `runCase` from `tests/testpilot_run_case_test.go`; `temporal.DeriveProfile` will produce the
   fault Case's Profile, including the `InjectFault` capability, from the Case itself. Note
   `temporal.Environment` is a fixed three-resource shape, so a second endpoint role carrying a
   resource binding would collide — keep the fault Case to one.

4. **Open design decision for `FaultIntentDeclaration.lower`.** The spec's signature is
   `FaultIntentDeclaration → Except LoweringError InstructionDefinition`, but the declaration
   (`Umpire/Space/Language.lean:68-78`) carries only `occurrence`, `action` and `capability`: it
   has no role id, no fault kind, no instruction bounds and no outcome schema, so it cannot on its
   own produce a node that `#guard`-equals the fixture's stop instruction. It needs a realization
   argument (instruction id, task-queue role id, limits, outcome) or a closed capability-id to
   `FaultKind` vocabulary. Decide that before writing the Case, since the Case's stop node is what
   the guard compares against. `model/Umpire/Space/Lowering.lean` is the right home: it has to
   import `Testpilot.Authoring` for `InstructionDefinition`, which `Space/Language.lean` does not.

5. **The functional manifest asserts an exact count.**
   `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:282` reads
   `want exactly 5`; a sixth fixture changes that literal and its test.

Suggested resolution: run this task where `make umpire-check-live-tests` can run. The offline half
(Case authoring, `FaultIntentDeclaration.lower`, manifest, regenerated bytes) is perhaps half the
work and is safe to do anywhere, but it should not land without the live run that gives it meaning.
## Evidence
- Commits:
- Tests:
- PRs:
