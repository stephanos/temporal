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
`Temporal.Case.Template` now owns the Temporal-specific Programs as plain `Realization` values.

- `model/Temporal/Case/Template/NexusOperation.lean` (new): `nexusOperation service operation
  responds sync|async`. The async form is the previous success Program verbatim. The sync form
  drops the completion-authority slot and the controller's await/complete pair, answers the
  operation synchronously in the handler, and orders the full history read behind a close-event
  read -- nothing in that controller observes the workflow, and `wait_new_event` returns the first
  page as soon as any event exists.
- `model/Temporal/Case/Template/Workflow.lean` (new): `workflow type`, the worker-outage Program
  without its two fault instructions, keyed on the single close event's `event_id` because a
  completed-workflow event names no operation. `faultRuleId` stays `worker-outage-order`.
- `model/Temporal/Case/EventKind.lean` (new): the admitted evidence kinds read off the generated
  `HistoryEvent.Attributes` oneof (60 kinds), with `resolve` producing the located diagnostic .3
  reports. No kind is listed by hand.
- `model/Temporal/Case/Support.lean` (new): role IDs, the two method paths, observation IDs, the
  correlated evidence type, and the close-event filter (read off the generated
  `HistoryEventFilterType` enum) shared by both templates.
- `model/Temporal/Case/Evidence.lean` (new): resolved `EvidenceRule`s to the Program's lift rules.
- `model/Temporal/Case/Template.lean`, `model/Temporal.lean`: facade and aggregator.
- `model/Temporal/Case/Tests/Template.lean` (new) + `model/TemporalModelTests.lean`: hooks,
  evidence sources, the sync ordering and its close filter, the async instruction list, and event
  kind resolution incl. the rejection listing.
- `model/Temporal/Feature/Nexus/Success/Producer.lean` is now three decisions: the template, the
  fixture name, and the evidence mapping. `model/Temporal/TestpilotTests.lean` reads the
  coordinates off the template.

Blocker check: `NEXUS_RESPONSE_KIND_SYNCHRONOUS` exists in `instruction.proto` and is exercised
throughout `common/testing/testpilot/temporal/worker` -- no blocker recorded.

Behaviour pin: `async-nexus-case.json` unchanged from .1's bytes
(`make umpire-check-case-runtime-conformance` green after every step).

Review: SHIP after two NEEDS_WORK rounds on `claude:claude-sonnet-4-5:high`.
- Round 1 P2/P3 (magic close-filter number, constants duplicated between templates) were valid and
  fixed by `Temporal.Case.Support` plus the generated-enum read.
- Round 1+2 P0 claimed the template's `faultRuleId` changes the async fixture bytes. That premise
  is false and was verified empirically: `faultRuleId` reaches a Case only through the ordering
  rule the Producer adds when a Scenario carries `fault` lines, the async Case carries none, and
  the byte-exact conformance diff was green on every round. The suggestion itself was harmless, so
  the per-response naming was adopted and the field's doc comment now states when it is read.
- The pinned reviewer `claude:claude-fable-5-1:high` returned an account limit on this task too, so
  every round ran on `claude:claude-sonnet-4-5:high`. Same-family fallback, not an equivalent
  cross-family review. The failed fable dispatch had already consumed the review artifact, so the
  first sonnet round needed `--force` to get past the unchanged-artifact guard.

stage: impl-review - ran, 3 rounds (model: claude-sonnet-4-5, high; fable pinned but account-limited)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3410350846, 0c88a1ff00, HEAD
- Tests: cd model && mise exec -- lake build, make umpire-check-case-runtime-conformance, make lint-model (0 findings outside generated Temporal/API/Proto.lean)
- PRs: