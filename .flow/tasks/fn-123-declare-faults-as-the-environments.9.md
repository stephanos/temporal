---
satisfies: [R1, R5, R9]
---
# fn-123-declare-faults-as-the-environments.9 Native fault declarations, budgets and realization choices

## Description
Implement the framework-native authoring portions of .1/.5/.6 after the settled input/domain APIs. Go exploration, IR/schema carriers, lowering and Temporal providers stay held.

**Size:** M
**Files:** model/framework/dsl/Action.scala, native binding/syntax seams, realize and focused tests.
**Touches:** [model/framework/dsl/**, model/framework/realize/**, model/framework/README.md, model/framework/samples/ReliableBroadcast.scala]

### Approach
- Re-anchor fault-party actions to current Action/Declares; add crash/responseLoss/storageLoss metadata and nonempty modelOnly reasons without alternate actors or compatibility spellings.
- Implement typed budgetedBy field binding on Boolean or Int domains, including bounded numeric carriers still supported by the final API. The approved fn173 amendment permits plain Int; do not require a finite state catalog or introduce a new bounded wrapper. Reject duplicate/wrong-field/wrong-type selectors and preserve the budget as ordinary native state. Global reached-state Go budget checks remain later; native declaration acceptance does not prove them.
- Retain native realization choosing metadata and check performed/model-only action/choice contracts at the native boundary that owns them; do not claim witness-lowering or manifest enforcement.
- Follow the parent's Native declaration seams: couple exact ordered choice tokens and deferred typed arms in one native program accepted by existing binding adapters. Preserve ordinary eager choose and callback behavior; do not introduce detached inventory assertions, execute effects to inspect choices, or retain stale choice metadata after rebind. Use identity rather than token names/Step equality.
- Enforce explicit selected-token membership at machine-aware completed realization/kit admission, with direct Performance/modelOnly bypass controls. Omitted selection means all, not an empty subset; opaque evidence cannot justify explicit selection. Add poison-arm controls for metadata construction and admission, and record/scalar/class-filter controls for program execution.
- Reuse ActionDecl copy/boundTo metadata and the native Choice/Performance seams. Retain choice identity explicitly rather than executing arbitrary effects to rediscover flattened choices. Migrate the framework ReliableBroadcast fault declaration if missing-kind enforcement reaches it; no disabled Temporal caller migration.
- Add framework-only fixtures and local author guidance. No task-queue/mini migration or new schemas.

### Investigation targets
**Required:** model/framework/dsl/Action.scala; model/framework/dsl/RuleBindings.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Machine.scala; model/framework/realize.

### Verification
Run touched native action/rule/realization/check selections, with external-author type/refusal probes and nonzero executed tests. Preserve independent no-fault native rule pins.

## Acceptance
- [ ] Fault kinds and nonempty modelOnly declarations survive native construction; missing/duplicate/invalid metadata rejects.
- [ ] Native budget selector/choice authoring is typed, rejects invalid bindings and preserves ordinary state/default/step behavior.
- [ ] Choice inventory and executable arms are structurally coupled; exact-token selection, duplicates, empty selection, wrong machine/class and opaque evidence refusals have independent controls without evaluating callbacks during admission.
- [ ] Framework-only examples and native positive/negative tests pass; global budget, witness lowering, schema and Temporal proofs remain open.

## Done summary
Blocked:
Owner-ordered native DSL queue. Release only after fn-173.11's native finite-domain/input-guard checkpoint is committed on umpire. Cross-spec task edges are unsupported; use this admission hold, never whole IR-bearing spec closure. Old Temporal migration dependencies do not gate the appended framework-native tasks. IR/Go/backend work stays held and model/temporal stays ignored.

Blocked:
Owner-ordered native DSL queue. Release only after fn-173.12's native reach/open/checker authoring and total-retirement checkpoint is committed on umpire. Cross-spec task edges are unsupported; use this admission hold, never whole IR-bearing spec closure. Old Temporal migration dependencies do not gate the appended framework-native tasks. IR/Go/backend work stays held and model/temporal stays ignored.
## Evidence
- Commits:
- Tests:
- PRs:
