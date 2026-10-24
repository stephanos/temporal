---
satisfies: [R5, R6]
---
# fn-164-expect-blocks-for-properties.4 Migrate capability and foundation predicates to expectation blocks

## Description
**Note 2026-10-18 (one-language series):** IR parity proofs use the in-process internal IR (fn-201); no checked-in IR.

**Size:** M
**Files:** product/system capability constructors, their focused laws/tests and foundation predicate users.
**Touches:** [model/temporal/capabilities/**, model/temporal/foundations/**]

Migrate reusable capability/foundation Property bodies after the API/lifting proof. R6's feature callers migrate in the next task; old finishers are not removed here.

### Approach
- Freeze scratch lifts and an independent declaration/function inventory before conversion. Convert Closable's two constructors, Terminable's one, Retryable's five, Expirable's four, Fenced's one and Idempotent's one, plus any remaining foundation lambda bodies discovered in the current tree. Inventory is grounded, not limited to these counts if predecessors changed it.
- Use generic Declares expectation predicates rather than acquiring a machine Owner; preserve law names, facet/group placement, restrictions, assumptions, limits, generated Query/Scenario identities and exact function reference/binding shapes.
- Convert conditional implications and true alternatives while keeping old association/eager locals; forward existing helpers without wrappers. Use explicitly named contextual bindings where exact old parameter names require them.
- Compare every scratch-lift declaration/function with the frozen predecessor using the explicit source-map seam from the lifting task; explain any unexpected difference before acceptance, not through field stripping. Run independent framework/capability tests and step-table equivalence.
- Do not rewrite pattern-only Pausable/Exclusive laws unnecessarily; inspect Pollable/Describable and foundations for missed callers instead of assuming the old scan remains complete.

### Investigation targets
**Required:** `model/temporal/capabilities/product/{Closable,Terminable}.scala`; `model/temporal/capabilities/system/{Retryable,Expirable,Fenced,Idempotent}.scala`; `model/temporal/capabilities/Properties.test.scala`; their adjacent focused suites; the preceding expectation/core comparison fixtures.
**Optional:** `model/temporal/foundations/`, `model/irgen/Capabilities.scala` (read-only generated metadata).

### Quick commands
```sh
mise exec -- scala-cli compile model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/project.scala model/framework model/temporal --test-only '*capabilities*'
make lint-model-syntax
make umpire-check-equivalence BASE=.flow/tmp/fn160/baseline-ae3f36ee51 MAP=model/testdata/equivalence/batch-1d.map STEPS_ONLY=1
```

Confirm the selected test filter actually executes capability suites; use actual suite names when package filtering selects none. Serialize heavy commands; retain scratch evidence, do not update checked-in IR/Cases.
## Acceptance
- [ ] Every reusable capability/foundation lambda Property is migrated with an independent before/after inventory; existing pattern-only laws remain correct (R6).
- [ ] Frozen scratch lifts prove complete declaration/function parity except mapped source positions, including generated claims/bounds/groups and explicit binding names (R5).
- [ ] Framework/capability regression tests and step-table equivalence pass on the migrated generic Declares path; selected suites are nonempty.
- [ ] No old-finisher removal, managed regeneration, Go, full review/live closure or unlisted preservation exemption is claimed.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: