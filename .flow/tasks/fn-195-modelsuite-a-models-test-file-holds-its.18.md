---
satisfies: [R4, R5, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.18 Add realize expectation conveniences and shared conformance validation

Touches: [model/framework/realize/Expectations.scala, model/framework/realize/Expectations.test.scala, model/framework/REFERENCE.md, .flow/tmp/fn195-expectations/**]

## Description
Add the accepted .10 convenience tuples and one shared validator for later conformance admission. Reuse the existing RunExpectation and MonitorExpectation hierarchy while preserving raw apply/copy compatibility. This leaf is independent of waited coverage, inventory and capability feasibility; .11 retains actual conformance-entry integration.

**Size:** M, three expected files and four acceptance groups.
**Files:** new `model/framework/realize/Expectations.scala`, new `Expectations.test.scala`, and the expectation/native-conformance sections of `model/framework/REFERENCE.md`. Existing .5 owns final README/global guidance synchronization.

### Approach
- Expose satisfied, conformant and the accepted property-only inconclusive(reason, because) conveniences from framework.realize. Preserve the existing nine-field order and independent conformance/property/Contract/disposition/cleanup/reason/conformanceReason/monitor/because meanings at `Realize.scala:736`.
- Use the frozen .10 tuples. Path satisfied is conformant/satisfied/satisfied/completed/succeeded with absent reasons, empty monitors and empty because. Sample conformant changes the singular Property to inconclusive with neverEvaluated and exactly `A sampled path selects no individual Property`; its Contract/disposition/cleanup remain satisfied/completed/succeeded.
- Restrict the property-only inconclusive shorthand to explanationsDisagree, neverEvaluated and unreadable with supplied nonblank because. Other reasons require an explicit full RunExpectation. Preserve independently mixed fields instead of deriving outcomes from one another.
- Provide private[framework] `validateConformanceExpectation(value: RunExpectation): Unit`, reporting invalid metadata with IllegalArgumentException detail. C later wraps that detail in its located diagnostic. Convenience outputs and later retained raw/copied values use this same rule set without checker-owned types. Preserve the existing raw constructors and restricted copy, including the deliberately accepted copied-empty raw value at `Expectation.test.scala:97`; reject such values only at conformance validation.
- Cover all nine independent Property/Contract pairs. Shared reason is empty exactly when both assessments are satisfied; because is empty in that pair and trim-nonempty in each other pair. ConformanceReason is empty exactly when conformance is conformant. Each monitor's reason is empty exactly when its own outcome is satisfied, with empty because for satisfied and trim-nonempty because otherwise. Map no semantic outcome from another field. Preserve monitor names/references unchanged. C still resolves monitor identity, duplicates and foreign-owner references when binding an actual declaration.
- Document the convenience tuples and shared admission boundary in REFERENCE without Scala-file links or a claim that ModelSuite conformance has shipped. Existing .5 consumes this interface for final guidance; .11 owns revalidation at its real entry point and .3 retains old vocabulary retirement.

### Investigation targets
**Required:**
- `.flow/artifacts/fn-195-modelsuite-a-models-test-file-holds-its/conformance-admission.md` - accepted complete tuple table and shorthand domain.
- `model/framework/realize/Realize.scala:736` - RunExpectation raw constructor, copy and independent fields.
- `model/framework/realize/Realize.scala:833` - MonitorExpectation construction/copy and field ownership.
- `model/framework/dsl/Expectation.test.scala:72` - existing copy-bypass controls and copied-empty compatibility neighbor.
- `.flow/tmp/fn1952-conformance-proof/CapabilityProbe.scala:76` - executed scratch tuple/validator controls.
- `model/framework/REFERENCE.md:1165` - task-owned expectation forms and refusals.
**Optional:**
- `model/framework/REFERENCE.md:1461` - native ModelSuite guidance and current integration limit.

### Verification and handoff
Run focused convenience/validator controls, the unchanged raw expectation compatibility neighbors and fresh outside-package source/separately packaged library clients under existing strict flags, with fresh complete-recipe workspaces. Retain exact selectors/nonzero counts, private-constructor/copy compile diagnostics, artifact hashes, commands and terminal exits under task-owned `.flow/tmp/fn195-expectations/`. Confirm declaration-only loading without checker initialization. The helpers express authored metadata and establish no observed Run/server verdict.

Keep .11 blocked on actual integrated inventory/capability prerequisites and its other leaves plus .9; this task does not register suites or make declarations eligible. Preserve .8/.14 and fn-196.2 holds, .3 retirement ownership and phase-batch review deferral. Temporal reads, actual repository Make, Go, export/lifting/IR/schema/backend/generated/regeneration, server/recorded-Run and Case operations remain held; export compile-only permission remains pending and ungranted.

## Acceptance
- [ ] Public framework.realize conveniences produce every accepted nine-field path/sample default, including sample Property inconclusive/neverEvaluated and its exact fixed because text, independent satisfied Contract/completed disposition/succeeded cleanup, absent conformanceReason and empty monitors. No default invents a selected singular Property or an observed result.
- [ ] Property-only inconclusive admits explanationsDisagree, neverEvaluated and unreadable with nonblank because; unsupported shorthand reasons refuse and remain expressible through the independent full RunExpectation. Mixed conformance/Property/Contract/disposition/cleanup/reason/monitor controls retain every supplied field.
- [ ] One reusable conformance validator checks full/raw/copied values and monitor values. It refuses omitted, empty, whitespace-only and copied-empty text for non-satisfied Property/Contract assessments, requires empty text for fully satisfied assessments, and checks all nine Property/Contract pairs with shared reason empty exactly when both are satisfied. ConformanceReason is empty exactly when conformant; each monitor has reason empty exactly when satisfied and its own matching text rule. Existing raw apply/copy behavior and private-construction/copy refusals remain unchanged; C retains actual monitor identity/owner/duplicate admission.
- [ ] Focused and fresh outside-package source/library controls execute nonzero counts under strict flags and load declarations without the checker. REFERENCE documents the forms and validation boundary without Scala-file links or shipped C claims. .11 explicitly consumes this validator at actual retained-value admission; .3 retains retirement and .5 retains final guide synchronization.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
