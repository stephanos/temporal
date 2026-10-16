---
satisfies: [R2, R3, R8]
---
# fn-173-reachable-exploration-drop-full-tables.12 Native reach/open/checker authoring and Query total retirement

## Description
Finish native authoring portions currently mixed into .2/.3/.8. Backend execution, export coverage admission and reached-graph semantics stay held.

**Size:** M
**Files:** framework DSL Claims/Machine/Syntax, native Query tests/examples and local guidance.
**Touches:** [model/framework/dsl/**, model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md]

### Approach
- Implement per-machine reach configuration, explicit query open(because) acceptance and typed checkedBy/confirm declarations using the settled design. Use framework-owned defaults, never Temporal Bounds. Preserve query subject/limits/expectation and deterministic checker configuration.
- Retire public Query.total authoring and migrate native callers/docs with no compatibility alias. Internal IR/carrier reader retirement remains .3's later obligation; do not force a schema migration now.
- Refuse invalid reach bounds, empty acceptance reasons, invalid checker values/configuration and repeated/conflicting configuration rather than silently overwrite it. Prove default behavior and copying/builder order preserve native declaration meaning.
- Keep configuration distinct from a checked answer. No backend invocation, export, replay, coverage admission or apparent successful result is introduced by this task.

### Investigation targets
**Required:** model/framework/dsl/Claims.scala; model/framework/dsl/Machine.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/IrFile.test.scala; model/design/reachable-exploration.md.

### Verification
Run native query/machine/syntax suites and external author compile tests with nonzero counts. No schema/IR/exporter/Go/backends or Temporal inputs.

## Acceptance
- [ ] Native reach/open/checker/confirmation authoring retains exact typed configuration, defaults, subject and limits; malformed configuration rejects.
- [ ] Public total authoring is inaccessible externally and native callers/docs use the final surface.
- [ ] Independent ordinary Query declaration pins and native tests pass; configuration is never reported as backend execution or a checked answer.
- [ ] Full R2/R3 engine/carrier retirement and R8 coverage/replay/backend proofs remain open.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
