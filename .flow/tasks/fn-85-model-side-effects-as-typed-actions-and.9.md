---
satisfies: [R10]
---
# fn-85-model-side-effects-as-typed-actions-and.9 One observation declaration per Case; the pendingAttempts read source

## Description
A Case declares each observation once with its source (a history event kind, a Run Event kind, or a read such as `DescribeWorkflowExecution`), its correlation key path and the fields it exposes; Program waits and Contract rules refer to it by name (R10, second half). The Producer emits the declarations from the Model's evidence and the realization's catalog, which gives `pendingAttempts` its read source: a bounded poll of one RPC projected by the pending operation's `scheduled_event_id`.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/program.proto` (observation declaration with a source oneof: history event, Run Event, read), `contract.proto` and `correlated.proto` (rules reference observations by name; no repeated key paths), `api/testpilot/v1/*`, `model/Testpilot/Authoring.lean`, `model/Umpire/Case/Producer.lean` and `Projection/Declaration.lean` (emit one declaration per observation), `model/Temporal/Case/Realization/Nexus.lean` (catalog: history events keyed by `scheduled_event_id`; `pendingAttempts` read binding), `model/Temporal/Case/EventKind.lean` (a second catalog kind for reads), `common/testing/testpilot/internal/execution/{prepare,dataflow,projection}.go` (waits resolve a declaration; a read source polls with a bound), `common/testing/testpilot/internal/verification/prepare.go` (rules resolve declarations; duplicate or undeclared rejects), `common/testing/testpilot/temporal/server/session.go` (the read RPC), conformance corpus (one case per observation source; undeclared and duplicate rejections)
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/Testpilot/**, model/Umpire/Case/**, model/Temporal/Case/**, common/testing/testpilot/**]

### Approach
- Extension checklist again: a new Run Event payload is not needed; the read source is a new observation source kind, so the checklist's "expression reference" and "conformance class" rows apply.
- A read observation is an instruction the Program performs (poll `DescribeWorkflowExecution` until the projected field satisfies the wait or the bound expires) whose result feeds the declared observation; the Contract reads it by name. No wait-for-duration instruction; timeouts in Queries 6 and 7 are observed through history events (task .11).
- Rejections: a reference to an undeclared observation; the same source and key path declared twice.
- The response read (fn-87's `ResponseRead`) stays for slot-bound reads; an observation declaration is the Contract-facing form.

### Investigation targets
**Required:**
- `model/Temporal/Case/EventKind.lean:16-73` — the history catalog to generalize
- `model/Umpire/Case/Projection/Declaration.lean` and `Producer.lean:296-345` — `projectionDeclaration`, `resolveEvidence`
- `common/testing/testpilot/internal/execution/projection.go` and `program.go:121-176` — reads and sinks
- `common/testing/testpilot/internal/verification/prepare.go` — how rules resolve observations today
- `tests/nexus_workflow_test.go:579-700` — `TestNexusOperationRetriesAfterHTTPFault` reads `pending_nexus_operations[].attempt`

**Optional:**
- `common/testing/testpilot/temporal/server/session.go:68` — `InvokeRPC`

### Key context
- Spec Decision Context: "Program and Contract declaring observations separately" was rejected because the same event kind, key path and fields were written twice and could drift.

## Acceptance
- [ ] every fixture declares each observation once; Program waits and Contract rules refer to declarations by name; a Case that declares the same source and key path twice, or references an undeclared observation, rejects at preparation with an existing category
- [ ] `pendingAttempts` is a read observation whose source is `DescribeWorkflowExecution`'s `pending_nexus_operations.attempt` keyed by `scheduled_event_id`, declared once; a conformance case reads it through the Temporal Driver
- [ ] a Driver conformance case exists per observation source (history event, Run Event, read)
- [ ] fixtures regenerate with the diff listed; `make umpire-check-regression` exit 0


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
