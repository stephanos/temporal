---
satisfies: [R6]
---
# fn-169-a-temporal-free-framework-the-lifter.3 Held sweep: a Temporal-free Umpire IR (Umpire namespace, generic realization layer, Temporal vocabulary as an extension schema)

## Description
Owner decision, 2026-10-13: the Umpire IR has no Temporal specifics. Held for the sweep (schema, Go, regeneration).

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
3. Move the Temporal vocabulary (entrypoint and instruction kinds, cause kinds, timeout bases, settlements, history evidence) to a Temporal-owned extension schema. The IR carries it in typed extension slots (`google.protobuf.Any` with a registered message type, or an equivalent the schema review picks). The Scala exporter's plug-in seam (fn-169.2) writes them, and the Go `tools/umpire/temporal` adapter (fn-170.2) and Testpilot read them.
4. A schema check fails if any IR file outside the extension schema imports `temporal/api` or names a Temporal concept.
5. Regenerate the IR, Cases and fixtures; the step-table equivalence and Case outcomes are unchanged under a recorded rename map.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
