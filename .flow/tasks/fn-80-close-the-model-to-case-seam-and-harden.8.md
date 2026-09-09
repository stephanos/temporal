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
- `Makefile:1067-1088,1148-1171` — fixture gates and the live expected-failure list

### Key context
- The stop precedes `start-workflow` so no workflow task is in flight when the worker stops; the reservation ledger mints handles independently of SDK polling (verify; adjust ordering if not).
- EVD-18's six conformance classes stay exact; this fixture is functional, like `get-system-info`.

## Acceptance
- [ ] `worker-outage-case.json` is generated deterministically and `make umpire-check-case-runtime-conformance` passes on both trees
- [ ] Live test (integration tag): Run completed, cleanup succeeded, Verdict satisfied, two `FAULT_INJECTED` events in stop-then-resume order referenced by the Contract's supporting sequences, correlated history evidence present
- [ ] Concurrent plain async-nexus Run on the same queue is unaffected (satisfied) while the fault Run stops its dedicated worker
- [ ] Negative live case: resume timeout yields cleanup `failed` with the Verdict unchanged
- [ ] `FaultIntentDeclaration.lower` `#guard` equals the fixture's stop instruction; `Umpire/Exploration/Coverage.lean` wording preserved
- [ ] The classic rule declares a `rule_events` horizon and the offline `PreparedContract.Evaluate` over the recorded Run agrees with the live Verdict
- [ ] `umpire-check-live-tests` expected-failure list unchanged

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
