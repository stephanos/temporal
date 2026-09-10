---
satisfies: [R2]
---
# fn-83-author-a-live-case-from-a-model-file.2 Realization templates and evidence kind resolution

## Description
Create `Temporal.Case.Template` with `nexusOperation service operation responds sync|async` and `workflow type` as `Realization` values (R2), each declaring hooks `start` and `completion` and its own `EvidenceSource` list, plus a resolver from a history event kind name to the generated attribute oneof field. The success Producer's hand-built `Realization` from .1 becomes `nexusOperation ... .async`. Split from .3 so the templates are proved as plain Lean values before the `case` command reads them.

**Size:** M
**Files:** `model/Temporal/Case/Template.lean` (new), `model/Temporal/Case/Template/NexusOperation.lean` (new), `model/Temporal/Case/Template/Workflow.lean` (new), `model/Temporal/Case/EventKind.lean` (new), `model/Temporal/Case/Tests/Template.lean` (new), `model/Temporal/Feature/Nexus/Success/Producer.lean`, `model/Temporal.lean` or the aggregator that imports Temporal modules, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/Case/**, model/Temporal/Feature/Nexus/Success/Producer.lean, model/Temporal.lean, model/TemporalModelTests.lean]

### Approach
- `nexusOperation` async form is the current success Program verbatim, as a function of `Identity` and the resolved evidence rules. The sync form replaces the asynchronous handler response plus the controller's await/complete pair with a synchronous handler response and drops the completion-authority slot; confirm the synchronous response kind exists in the instruction proto and the worker interpreter before designing, and record a blocker if it does not.
- Sync history read: with the await/complete pair gone nothing sequences the history read after completion, and `GetWorkflowExecutionHistory` with `wait_new_event` returns the first page as soon as any event exists. Follow the worker-outage Program: a first read with the close-event filter, then the full read the Projection consumes, with the second depending on the first.
- `workflow` is the worker-outage Program minus its two fault instructions; keep `faultRuleId := "worker-outage-order"`. Its one `EvidenceSource` is `workflow_execution_completed_event_attributes`; a completed-workflow event names no operation and `ScopedEvidenceRule.operation` is always a field path, so the operation key is a stable path on the single close event such as `event_id` (one workflow per Run). The Nexus sources keep `scheduled_event_id`.
- Hooks map `start` to the `start-workflow` controller instruction and `completion` to the history read; placement rewriting is the Producer's job in .5, the template only names instructions.
- Event kind resolution: read the `HistoryEvent.Attributes` oneof from the generated schema node (all `(.oneof "attributes")` fields) and map a camelCase kind name to the snake_case field; an unknown kind returns the admitted list for the diagnostic in .3. Do not hand-list the kinds.
- Role IDs, binding IDs, and RPC method strings move from the two Producer files into the templates; `Temporal.Testpilot.CaseSupport` stays the home of the shared builders.
- Behavior pin: async-Nexus fixture unchanged from .1's bytes.

### Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus3/Testpilot.lean:37-48, 100-175` — role IDs, the history node, and the Program that becomes the async template
- `model/Temporal/Testpilot/WorkerOutage.lean:39-45, 107-146` — the Program that becomes the workflow template, including its close-event read
- `model/Temporal/API.lean:3466` and `model/Temporal/API/Types.lean:4151-4214` — the history attributes oneof and its typed mirror
- `proto/internal/temporal/server/api/testpilot/v1/instruction.proto:70-136` — `RespondNexus` kinds and the instruction vocabulary
- `common/testing/testpilot/temporal/worker/interpreter.go:33-45` — what the SDK interpreter supports for a synchronous handler

**Optional:**
- `service/history/api/getworkflowexecutionhistory/api.go:260-278` — why a `wait_new_event` read without a close filter returns early
## Acceptance
- [ ] Both templates compile as `Realization` values; a Lean test checks each declares exactly hooks `start` and `completion` and its evidence sources
- [ ] The sync template's Program has a close-event read before the full history read; a Lean test asserts the dependency
- [ ] Event kind resolution accepts `nexusOperationStarted`, `nexusOperationCompleted`, `workflowExecutionCompleted` and rejects an unknown kind with the admitted list
- [ ] The success Producer uses `nexusOperation "umpire.case.service" "complete" .async`; async-Nexus fixture bytes unchanged
- [ ] `make lint-model` passes with the new modules classified
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
