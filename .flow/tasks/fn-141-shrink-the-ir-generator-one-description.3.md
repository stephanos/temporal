---
satisfies: [R3, R4]
---
# fn-141-shrink-the-ir-generator-one-description.3 Generic definition expansion for function-body sugar

## Description
Part A, first half: the lifter expands a function-level sugar from its Scala definition, with its arguments bound, then lifts core constructs and types. Remove the hand-written expression lowerings for enter, stay, reject, disabled, membership/role in, implies and records, covering all their current function-body overloads. Declaration-level patterns, witness builders and block/rule forms remain explicitly open in .4 and the constructed-value export tasks; they are not permanent exceptions to R3/R4. Current paths and mini-only execution follow MILESTONES.md and the activated parent spec.

**Size:** L
**Touches:** [model/framework/dsl/**, model/framework/export/**, model/framework/inspect/**, model/framework/lint/**, model/framework/check/**, model/temporal/check/**, model/temporal/migrate/README.md, model/README.md, model/SEMANTICS.md]

### Approach
- Re-read export/{Syntax,Expressions,Claims,Context,Lift}.scala, inspect/Index.scala, dsl/Syntax.scala and the existing Sugar.scala fixture. Index the framework and fixture kit definitions as well as Model TASTy while excluding framework definitions from author roots and source lint. Reuse one shared index and existing parameter/type/function bindings.
- Select a generic sugar-definition marker (Syntax.scala membership or a dedicated annotation) from the established syntax contract and record the choice. The expansion and reducers must name no particular sugar. A new fixture-kit sugar expands without changing exporter code or registering its name.
- Preserve the existing IR, including node positions: generated core nodes use the author call's position, while explicit argument expressions retain the positions the existing lowering gives. Reuse the same fixture/golden assertions instead of recapturing a bespoke baseline. Split sugar tests from Fixtures.test.scala beside expansion code.
- Implement generic reductions only as needed: known constructor field access, supplied/default arguments, type/value/function binding, repeated arguments and core collection/boolean/type-test operations. Preserve implication's short circuit and holes, ordered facts, method arity and finite role membership. Do not replace named sugar cases with a renamed whitelist or another hand-written per-sugar dispatcher.
- Composition records must honor the selected member's exact composed fact key. Its current native suffix-only body ignores the selector while its IR uses the member key; reconcile through a core definition/capture mechanism and prove the native answers match the existing IR contract, including same-named facts in different members. If recording selector data needs a capture point, the parent architecture already allows inline/macros there only; keep sugar plain and report the design. No macro may lift a function body, and no arbitrary compiler-name/reflection heuristic is an accepted selector substitute.
- Refuse unliftable sugar definitions with their name and definition position, and direct/indirect recursion with a named cycle. Keep the rejection at the author-facing call where appropriate. Add meaningful focused red/green tests for extension/context/default/type/vararg binding, kit sugar, cycles, errors and source positions; reuse existing controls for forms they already cover.
- Preserve live mini/shared/sample sources and runtime answers, and fixture core oracles including the reserved Bindings mechanism. Real-feature sources/generated outputs remain untouched. A concise migration recipe is required only for an author-visible change; no new Scalafix rule or bespoke proof/capture/hash manifest.
- Baseline: fn-141.2's green .flow/tmp/checks/20261013-031649/record.md (597 tests), with the recorded PropertyGroupsNative classpath race separate. Iterate affected suites, then run make model-check-changed BASE=<task-base> once at end, with a bound matching its documented 8–13 minute range and mini step-table equivalence. A genuinely failed run licenses diagnosis and a warranted rerun; an observation timeout never licenses restarting a live command. Strict generated-IR/full gates, broad lint, independent review, Go and regeneration remain held until the batch boundary.

## Acceptance
- R3/R4: every current function-body overload of the seven named sugar families expands generically from its definition. The expression pipeline contains no named lowering arm, intrinsic masquerading as a sugar-specific fallback, or name-registration requirement for a new kit sugar. Remaining declaration work is recorded for .4/.10-.12 and the final audit.
- Sugar/core fixtures retain the same IR and source positions under their existing comparison contract; a new fixture-kit sugar lifts without an exporter change. Runtime controls include short circuit, ordered facts, role membership and exact composition member identity.
- Unliftable sugar and direct/indirect sugar cycles are refused with the required names and positions. Surviving refusal/golden contracts and mini/shared semantics are preserved.
- The task's model-check-changed record passes with mini step-table equivalence. Before/after source counts, marker/position/binding design, focused results, any migration recipe and deferred gates are in the receipt. No generated payload or real-feature source changes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
