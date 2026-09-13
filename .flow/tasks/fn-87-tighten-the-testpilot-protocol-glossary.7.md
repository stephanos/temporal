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
A Run Event's kind-specific data now lives in one `payload` oneof (`outcome`, `fault_injected`). Contracts read it through a path from a new payload reference. `RunEventField` keeps only the common coordinates, and `RUN_EVENT_FIELD_FAULT_ROLE_ID`/`FAULT_KIND` are retired. The worker-outage Verdicts did not change: the unit Verdict tests and the live test pass, and the oracle passes with one declared step.

**Protocol**
- `RunEvent` fields are `observations = 7`, `execution_incomplete = 8`, then `oneof payload { outcome = 9; fault_injected = 10 }`. `RUN_EVENT_KIND_FAULT_INJECTED` is unchanged.
- `RunEventReference` is now a `selection` oneof: a `RunEventField` coordinate, or an empty `RunEventPayloadReference`.
- A payload is read only as `path(run_event.payload, <arm>.<field>)`. The first path segment names the arm. So a future payload adds no reference arm and no enum value.
- Decision: reading a declared protocol message by path counts as an event-field read under EVD-13.
- Decision: no diagnostic-reference arm, because nothing records one.
- `InstructionOutcome` moved to `run.proto` and joins `FaultInjected` in the R11 Run-only list. No Case message names it; outcome typing uses only the enums.

**Go**
- `ir.RunEventPayloadOf` is the single kind-to-payload table:
  - `FAULT_INJECTED` requires `fault_injected`.
  - `INSTRUCTION_COMPLETED`, `INSTRUCTION_TIMED_OUT` and `DIAGNOSTIC` may carry `outcome`.
  - A test pins the table against the oneof members.
- `ir.CheckRunEventPayload` checks each event before the recorder stages it. A mismatch is recorded as an `INVARIANT` diagnostic with code `payload_kind_mismatch`. Test: `TestRecorderRejectsPayloadKindMismatchAsInvariant`.
- `ir` binds a payload path through the arm named in its first segment:
  - The arm must be declared in scope; the rest of the path is typed by the arm's descriptor.
  - Located rejections: an unknown arm, an undeclared arm, and an empty path.
  - A bare payload operand is also rejected.
  - Test: `TestRunEventPayloadPathsBindThroughTheArmTheyName`.
- Contract preparation declares the arms a transition's filter kinds may carry:
  - A path into an arm none of them carries rejects `unknown` at `...path.path.segments[0].field`.
  - Per evaluated kind, only a required arm counts as available. A read of an arm the event may lack is absent and needs a presence guard (fail-closed until .16).
  - Tests: `TestPrepareLocatesPayloadPathsTheFilterCannotCarry` and `TestEvaluatorReadsAnUncarriedPayloadAsAbsent`.
  - Both new tests went red when their checks were loosened.

**Lean and fixtures**
- `Expr.runEventPayload` is added, and `Run.event` takes a `payload`.
- The worker-outage Producer reads `fault_injected.role_id` and `fault_injected.kind` by path, unguarded, because both transitions filter only on `FAULT_INJECTED`.
- `worker-outage-case.json` was regenerated through its generator. Its only diff is the four coordinate reads turned into paths.

**Oracle, vocabulary and docs**
- The oracle step rewrites the fault coordinates, matched by name or number, and refuses a reference that carries extra keys.
- The retired-vocabulary gate gains a `RUN_EVENT_FIELD_FAULT_*` rule, with a positive test line added after review.
- `protocol_test` forbids the retired enum names.
- The READMEs and the fn-87 Planning decisions ("Run Event payloads (decided in .7)") are updated.

**Gates**
- Baseline was green via the receipt at a1c1ec7d.
- Oracle and unit tests: green.
- `lint-code`: 161 issues after `go clean -cache`, which is the baseline.
- `lint-model`: 163, the baseline.
- `umpire-check-regression`:
  - At e2bbfff445: exit 0, 9 live identities.
  - At HEAD, run 1: failed on `TestTestpilotAsyncNexusCaseRunsFromItsFixtureNameAlone`. The async-nexus Run was INCOMPLETE, which is known flake (c).
  - At HEAD, run 2: exit 0, 9 identities. A green receipt was written.
- Worker-outage live tests: 8 of 8 passed across 4 separate processes.
- Under `-count=4` in one process, the peer async-nexus Run sometimes ends INCOMPLETE. The base commit shows the same failure, so the flake predates this change and my change does not cause it.

**Follow-ups (not built)**
- `RunEventPayloadValue` treats a marshal failure as absent (reviewer FYI).
- Renumbering the `RunEvent` fields breaks binary compatibility; the wire has no compatibility promise.

stage: impl-review - ran (claude backend, SHIP on first round, one P3 applied as a test-only follow-up commit)
## Evidence
- Commits: e2bbfff44535880a2a139e00712e24c0a4cb9405, 02d09609d092a1942771d05c6ca1caf44a5dc6e3
- Tests: baseline: green via receipt a1c1ec7d (regression) and a pre-edit oracle run, go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/, go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/internal/retiredvocabulary/ ./tests/testcore/testpilot/..., make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance, make umpire-check-retired-vocabulary, go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161 issues, baseline), make lint-model (Found 163 errors, baseline), CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression: run 1 at e2bbfff445 exit 0 (9 live identities); run 2 at 02d09609d0 exit 2 (TestTestpilotAsyncNexusCaseRunsFromItsFixtureNameAlone: async-nexus Run INCOMPLETE, known flake c); run 3 at 02d09609d0 exit 0 (9 live identities), go test -count=1 -tags "test_dep integration" ./tests -run ^TestTestpilotWorkerOutage: 4 separate runs, 8/8 pass, go test -count=4 ... -run ^TestTestpilot(WorkerOutage|AsyncNexusCase$): async-nexus Run INCOMPLETE on repeated in-process iterations at HEAD and at the base commit alike (base: 1 of 3 runs failed, 2 fails)
- PRs: