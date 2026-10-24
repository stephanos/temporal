---
satisfies: [R3]
---
# fn-246-realization-as-a-checked-refinement-a.2 Shared Temporal layer: descriptor-generated WorkflowService catalog with claims and field kinds, entity lens machinery, reads, program envelope, probe engine over fillers

## Description
After .1. Create `model/temporal/shared/{catalog,lens,probes,support}`. Catalog: generate the WorkflowService mirrors from the descriptors, versioned with the API; Temporal-wide field kinds (`Namespace`, `Identity`, `ActivityId`, `RequestId`, `TaskQueueName`) carry limits and rejection patterns once; features claim reads and fills; unclaimed fields are Ignored by default; the opaque-field rule gives `Secret`/`Ignored` no inferred byte-length bound. Lens: `#[entity(bootstrap, identity(...))]` and `#[read(rpc, interval, timeout)]` drive the admission transaction, receipt, freeze, Reset, proposal correlation (including the reconciled acknowledgement), the read loop and the hold, moved from `realization/src/live.rs` and `feature/activity/temporal.rs` with every feature name removed; program envelope assembly from `ReplyPolicy` rows. Probes: the engine from `realization/src/probes*` takes catalogs and fillers, no hand baselines. A scoped check fails on any feature identifier in `shared`. Goldens for Models without a Temporal binding stay byte-identical.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
