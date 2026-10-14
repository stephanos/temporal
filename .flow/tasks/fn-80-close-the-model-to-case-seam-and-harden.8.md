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
A Producer-neutral functional Case now asks the Driver for one real outage and requires the work to
survive it. Its controller stops the SDK worker of its own activation queue, starts the workflow
while nothing is polling that queue, resumes the worker, and long-polls for the closing history
event. Two rules sit on that Run: a bounded-liveness rule over the recorded `FAULT_INJECTED` events
that reaches its satisfied state only on a stop followed by a resume on this Case's task-queue role,
carrying the checked-in `rule_events` horizon; and a safety rule over the completed workflow, which
can only have been dispatched after the resume. Both are satisfied live.

`FaultIntentDeclaration.lower` resolves the task's open signature question. The declaration decides
the *outage*: a closed version-one vocabulary maps the capability a fault intent targets onto a
`FaultKind`, and a capability outside it rejects by name. It cannot decide the *placement* -- the
Space language has no Program to name an instruction, a role or a bound in -- so placement arrives as
a separate `FaultRealization` argument. The shipped Case's stop instruction is the lowered intent
itself, pinned by a `#guard` on the consumer's side of the Umpire/Temporal import boundary.

Constraints the earlier tasks discovered, applied:
- The fault Case owns its own queue resource binding, and the "concurrent plain Run unaffected"
  assertion puts the plain Nexus Case on a *different* queue. A pooled peer worker on the same
  physical queue keeps polling through the outage, so a same-queue assertion would be asserting the
  outage is not real. The task's acceptance text says "same queue"; this is the deviation.
- The functional manifest's exact count moves from five to six, with its stale-file test.

Deferred, with a trace beside the tests: the live resume-timeout case. This Case always resumes and
the Driver exposes no seam for making a live resume fail, so reaching it live would need a second
fixture that stops and never resumes -- a Case whose own Contract could not be satisfied. The two
halves are pinned where they are reachable: `TestSessionCloseResumesAndAlwaysReleasesTheHold` (task
.3) for the Driver surfacing a failed resume, and the `cleanup-failure-after-proved-violation`
conformance class for a failed cleanup leaving the Verdict alone. A Driver test seam is the follow-up.

`PreparedContract.Evaluate` is internal, so no test under `tests/` can call it. The online/offline
agreement for this Case is instead pinned in `internal/verification` over the shipped Contract and a
recorded outage Run, including the outage that never ends and expires on the count.

stage: impl-review - ran [8a251dbf..966bf310] SHIP
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c43161b503c3baee028c9b6b8c404671228faa12, 1197d567ac7e29b9423fa03524b92755f81bb269, c26d67874216c7249505f06322d6d0745eccd0ae, 966bf3101fe8b6e24973a53b9342ce08829f1c23
- Tests: cd model && lake build Temporal TemporalModelTests UmpireTests TestpilotTests, make lint-model (169 errors, unchanged baseline, all in generated Temporal/API; import-graph clean), make umpire-check-case-runtime-conformance, CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/..., go test -tags 'test_dep integration' ./tests -run '^TestTestpilotWorkerOutage' (both live outage tests pass), make umpire-check-live-tests (empty failure set across 6 passing identities, up from 4), make umpire-check-regression, make lint-code GOLANGCI_LINT_FIX=false (128: errcheck 1, govet 4, revive 106, staticcheck 17 - unchanged baseline), go vet -tags test_dep ./... (15 pre-existing diagnostics, unchanged)
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
