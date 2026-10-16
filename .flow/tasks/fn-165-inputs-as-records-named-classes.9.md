---
satisfies: [R1, R3, R6, R8, R9]
---
# fn-165-inputs-as-records-named-classes.9 Migrate native record callers and retire token and arity APIs

## Description
Finish native caller/API migration using .7/.8. Retire the public token and positional arity ladder independently of held exporter adaptations.

**Size:** M
**Files:** framework native DSL tests, realize helpers, six samples and framework migration guidance.
**Touches:** [model/framework/dsl/**, model/framework/realize/**, model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md]

### Approach
- Consume .7 inventory and migrate native callers, examples and tests. Keep intentional compile-negative and independent core-oracle fixtures explicit.
- Remove public token declarations/input(token), Assigned/class :=, Slot/InputAt overloads and the old effect arity ladder with no compatibility layer. Preserve unrelated request-scope :=.
- Do not retain a legacy public API for disabled Temporal or held IR fixtures; original .3/.6 own exporter reader/capture retirement later.
- Document records, declared defaults, scalar/no-input forms and structural selectors in framework-local guidance and Core forms. Record native step/default/class-order pins and unrun integration obligations.

### Investigation targets
**Required:** model/framework/dsl/Action.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Inputs.test.scala; model/framework/dsl/InputDefaults.test.scala; model/framework/samples; model/framework/realize.

### Verification
Run touched native core/check selections plus external-author old-spelling compile refusals. Compare migrated native fixtures to .7/.8 independent behavior pins. No full exporter/IR gate or Temporal sources.

## Acceptance
- [ ] Native inventory has no unexplained old callers; public token/class :=/arity APIs are inaccessible from an external author package.
- [ ] Native class ordering, defaults, step behavior and custom domain witnesses retain the recorded independent pins.
- [ ] Framework-local docs and Core forms describe the final API; request-scope := remains unchanged.
- [ ] Native checks pass; full R4/R6/R7 IR retirement and Temporal migration remain open.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
