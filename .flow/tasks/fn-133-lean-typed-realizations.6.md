---
satisfies: [R9, R15, R16]
---
# fn-133-lean-typed-realizations.6 Typed realization objects, derived defaults, derived realizations

## Description
Part C, R9, R15, R16.

The activity's typed realization objects remain in `model/temporal/features/activity/standalone/system/Realization.scala`, preserving fn-126.11's placement through fn-132. All three existing activity System realizations stay in that file; replacing wrapper objects does not move them to the root or introduce a Product realization file.

- **Typed objects.** `object Standalone extends Realization(ActivityProtocol)`, typed by the machine's state, outcome and fact types. A header holds what is not derived, and named sections in a fixed order: controller, worker scripts, evidence, server steps, controls. Evidence facts, status-table keys and `perform`/`onPath` classes are checked against the machine, with one negative compile fixture each. The wrapper objects (`ActivityRealization`, `NexusRealization`, `OperationRealization`) go.
- **Lint.** The structure lint (fn-126 R20) checks realization sections and their order, and one realization per object.
- **Derived defaults.** The operation is the machine's entity. Roles are those named by the scripts' calls and activations. Deadline and backoff timers get the kit's default server steps. An explicit value equal to the derived one is a lint finding.
- **Derived realizations.** A realization declared from another with steps added or replaced; `forgedCompletion` derives from `asyncNexus`, and the local `realization(machine, steps*)` factory and its concatenated controller go.

Realization and script IDs follow the new objects' fully qualified names (fn-126 decision 23): prove the projection with an ID map.

## Acceptance
- [ ] Every realization is its own typed object; the negative compile fixtures fail to compile; the lint refuses an out-of-order section and an object holding two realizations.
- [ ] No realization states its operation, roles or default server steps.
- [ ] `forgedCompletion` is derived, and replacing a step its base lacks is refused at its line.
- [ ] A before/after projection with the ID map applied is identical.
- [ ] The spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
