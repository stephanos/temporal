---
satisfies: [R6]
---
# fn-169-a-temporal-free-framework-the-lifter.3 Held sweep: a Temporal-free Umpire IR (Umpire namespace; Temporal vocabulary as data validated against descriptors and Driver capabilities)

## Description
Owner decision, 2026-10-13: the Umpire IR has no Temporal specifics, not even as an extension schema. Held for the sweep (schema, Go, regeneration).

Today the Model core (`machine`, `claim`, `expression`, `operand`, `value`, `common`; about 900 lines) is Temporal-free. Temporal sits in:
- the package `temporal.server.api.umpire.v1` inside Temporal's internal proto tree;
- `script.proto`:
  - activation kinds `WorkflowActivation`, `NexusHandlerActivation` and `ActivityActivation`;
  - instructions `WorkflowCommand`, `NexusReply`, `NexusCompletion`, `AttemptHeartbeat`/`AttemptWithheld` and `AttemptFailure` (a `temporal.api.failure.v1.Failure`);
- `realization.proto`:
  - `ActivityExternalSettlement` and `ActivityResetSettlement`;
  - `CauseKind` (`RespondActivityTask…`, `RespondWorkflowTaskCompleted`, `RespondNexusTask…`);
  - `TimeoutBasis` (`schedule_to_close`…);
  - `Evidence.history` (`temporal.api.history.v1.HistoryEvent` attributes).

Do:
1. Move the IR to an Umpire-owned namespace (for example `umpire.ir.v1`), out of Temporal's proto tree.
2. Keep a generic realization layer in the IR: roles, requests and response reads by typed message name (`Rpc`, `Assignment`, `ResponseRead`, `Poll`), evidence by message path, `Correlation`, `Learned`/`Observed`, `Control`/`HoldDispatched`, `Visibility`, `WaitBound`, gRPC rejection codes, and worker scripts as entrypoints plus instructions. Neutral IR type names for the framework's DSL types can come with this (the fn-169.1 follow-up).
3. **No extension schema (owner, 2026-10-13).** Express the Temporal vocabulary as data in the generic layer, resolved against the system's descriptor set (fn-169.2's generic proto support):
   - typed-message instructions (a message type name plus its value) replace `WorkflowCommand`, `NexusReply`, `NexusCompletion` and `AttemptFailure`;
   - an entrypoint has a kind name plus its delivering and answering methods, replacing the three activation messages;
   - the causing method, named as data, replaces `CauseKind`;
   - the request field path that supplies a timeout replaces `TimeoutBasis`;
   - evidence read at a path in a recorded message stream replaces `Evidence.history`;
   - kind names with typed message arguments replace heartbeats and settlements.
   Safety: the exporter validates every message type, method and field path against the descriptor set, and each Driver declares the entrypoint and instruction kinds it realizes, so preparation refuses the rest (as Testpilot's Driver-reach table does for message fields). Temporal knowledge stays in the Scala Temporal kit (which produces the names) and the Temporal Driver (which interprets them). A construct that cannot be expressed generically is listed as an explicit exception for the owner.
4. A schema check fails if any IR file outside the extension schema imports `temporal/api` or names a Temporal concept.
5. Regenerate the IR, Cases and fixtures; the step-table equivalence and Case outcomes are unchanged under a recorded rename map.
## Acceptance
- [ ] TBD

## Done summary
Superseded by fn-201/fn-202 (one-language series, 2026-10-18): ir.proto and Testpilot protos move under model/framework/proto

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: