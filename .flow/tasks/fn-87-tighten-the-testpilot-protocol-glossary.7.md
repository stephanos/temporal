---
satisfies: [R4, R11]
---
# fn-87-tighten-the-testpilot-protocol-glossary.7 Run Event payload oneof and Contract paths into payloads

## Description
Give `RunEvent` one `oneof payload` (instruction outcome, fault injected, diagnostic reference) instead of per-kind optional fields, trim `RunEventField` to the common coordinates (drop `FAULT_ROLE_ID`, `FAULT_KIND`), and let Contract expressions read payload fields through a `path` from a `run_event` reference (R4, spec "Run Event payloads"). Keep the R11 closure test green with the payload messages added to its forbidden list.

**Size:** M
**Files:** `proto/.../v1/{run,event,expression}.proto`, `api/testpilot/v1/*`, `common/testing/testpilot/internal/execution/{scheduler.go,recorder.go,recorder_test.go,fault_test.go}`, `common/testing/testpilot/internal/verification/{evaluator.go,prepare.go,fault_test.go,worker_outage_test.go}`, `common/testing/testpilot/internal/ir/{expression.go,expression_test.go}`, `common/testing/testpilot/temporal/worker/{session.go,fault_test.go}`, `tests/testpilot_worker_outage_case_test.go`, `model/Testpilot/Authoring.lean:495-540`, `model/Temporal/Testpilot/WorkerOutage.lean`, `common/testing/testpilot/protocol_test.go`, fixtures (worker outage), mapping
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, common/testing/testpilot/**, model/Testpilot/**, model/Temporal/Testpilot/**, tests/testcore/testpilot/**, tests/testpilot_worker_outage_case_test.go]

### Approach
- Proto: `RunEvent { ...common coordinates...; oneof payload { InstructionOutcome outcome; FaultInjected fault_injected; ... } }`. Kind to payload is not 1:1 today: `DIAGNOSTIC` events carry an `Outcome` (`execution/scheduler.go:765`) and projection-emitted `INSTRUCTION_COMPLETED` events carry Observations and no outcome (`scheduler.go:797`). So the arms are `outcome` and `fault_injected`; `DIAGNOSTIC` keeps its outcome arm; a diagnostic-reference arm is added only if it costs nothing (record the decision). `RUN_EVENT_KIND_FAULT_INJECTED` keeps its name (EVD-20). `observations`, `execution_incomplete`, `causal_source_ids` are common, not payload.
- Kind/payload table in one Go place (`execution` or `ir`): which arms each kind may carry (`INSTRUCTION_COMPLETED`, `INSTRUCTION_TIMED_OUT`, `DIAGNOSTIC` ↔ outcome or none; `FAULT_INJECTED` ↔ fault_injected, required; every other kind ↔ none). Used by the recorder and by preparation.
- Runtime error surface: a staged event whose payload does not match its kind is recorded as a Run diagnostic of kind `RUN_DIAGNOSTIC_KIND_INVARIANT` (the spec says "Driver invariant diagnostic", but these events are built by the scheduler, not the Driver, and EVD-20's wording is about refused faults; do not cite EVD-20; the recorder's generic failure path at `recorder.go:76-78` records `RECORDER`, so add an explicit invariant check before staging, beside `recorder.go:107-116`). Unit test with a mismatched event.
- Preparation error surface: a Contract `path` from `reference.run_event` into a payload arm rejects with a located path when every kind in the transition's `RunEventFilter` lacks that arm, or when the filter is empty (for example a `fault_injected.kind` path on a transition filtered to `INSTRUCTION_COMPLETED`); otherwise an arm the event does not carry is absent at runtime. Today fault fields are bound at `verification/prepare.go:209-241` with `Available: true` and read at `evaluator.go:354-357` (returns `""`/0, never absent); the new path read is typed from the payload descriptor, and a path into a payload the event does not carry is absent (fail-closed per EVD-12 until .16's rule). This is a semantic shift for non-fault events (`""` today, absent after); it is safe only because the worker-outage Contract filters on `FAULT_INJECTED`; state that in the mapping step.
- `RunEventReference` selects the event itself (common coordinates via `RunEventField`, payload via path). Decide whether `run_event` stays `RunEventField` plus a separate path, or becomes a reference to the whole event read by path for every field; pick the one that keeps EVD-13 (Contracts inspect declared Observations and event fields, never arbitrary raw payloads: the payload messages are declared protocol messages, so reading them by path is an event field read) and record it.
- Lean: `Testpilot.Authoring.RunEvent` constructor takes the payload; the worker-outage Producer's Contract reads `fault_injected.role_id`/`kind` by path.
- R11: forbid the Run-only payload messages (`FaultInjected`, `DiagnosticReference`) in the closure test. `InstructionOutcome` may have to stay in the Case closure because Program outcome typing names it (`temporal.server.api.testpilot.v1.InstructionOutcomeStatus` as an enum type); if so, record why.
- Mapping: move `outcome`/`faultInjected` keys under the payload arm (JSON name unchanged if the oneof arm keeps the field names; then the step only rewrites Contract `runEvent: {field: RUN_EVENT_FIELD_FAULT_*}` into the path form). Retire `RUN_EVENT_FIELD_FAULT_ROLE_ID`, `RUN_EVENT_FIELD_FAULT_KIND`.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/execution/scheduler.go:770-800`, `recorder.go:60-140`
- `common/testing/testpilot/internal/verification/prepare.go:200-245`, `evaluator.go:330-370`
- `common/testing/testpilot/internal/ir/expression.go:265-293` — run event field range check at `:284`
- `model/Temporal/Testpilot/WorkerOutage.lean` — the one Contract reading fault fields
- `common/testing/testpilot/temporal/worker/session.go:240-260` — `diagnoseFault`

**Optional:**
- `.plans/UMPIRE4_SPEC.md` EVD-13, EVD-20

### Key context
- The worker-outage live test and `verification/worker_outage_test.go` are the Verdict pins for the fault payload.

## Acceptance
- [ ] `RunEvent` carries kind-specific data in one payload oneof; `RunEventField` has only the common coordinates; `RUN_EVENT_KIND_FAULT_INJECTED` unchanged
- [ ] a payload/kind mismatch becomes an `INVARIANT` Run diagnostic (unit test); a Contract path into a payload its filtered kinds cannot carry rejects at preparation with a located path (unit test)
- [ ] the worker-outage Contract reads the fault through a path; its live test and unit Verdicts unchanged
- [ ] R11 closure test forbids the Run-only payload messages and passes; equivalence test passes with declared steps; retired tokens added
- [ ] `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
