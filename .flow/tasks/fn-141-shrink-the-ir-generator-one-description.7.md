---
satisfies: [R5]
---
# fn-141-shrink-the-ir-generator-one-description.7 Realization factories build complete values

## Description
The realization DSL discards part of what an author writes, because the lifter reads it from source. Make every factory keep everything, so the values can be exported.

**Size:** M
**Files:** `model/framework/realize/{Realize,Typed,Scripts,Exploration,Syntax}.scala`, `model/temporal/realize/*.scala`, focused munit tests beside their owners, and the existing migration README for a short recipe if author syntax changes.
**Touches:** model/framework/realize/**, model/temporal/realize/**, model/temporal/migrate/README.md
**Scope:** framework and shared Temporal kit values, exercised by `activity_mini` and focused fixtures. Real features are not edited, lifted or regenerated. This is the prerequisite for fn-174.2, whose DSL types construct their own messages; fn-141.5 and .8–.14 are superseded. The current lifter remains unchanged in this task.

### Approach
- A request scope (`rpc { … }`, `readUntil { … }`, `withFields { … }`) keeps its assignments; today the block runs and its result is dropped.
- An evidence value keeps its operation and fields. `Condition.equal`, `greater`, `not` and `all` keep their operator; today two pairs build identical values.
- `Field(_.a.b)` records its path, and a factory that takes a protobuf type argument records the message name, as `Action.schema` already does through the companion.
- Retain alias targets and action-valued `onPath` inputs without unsafe casts; preserve all factory arguments needed by constructed export. Names/family may remain deferred to fn-174.2's reflection context, but identify those deferred identity fields explicitly. Do not invent authored identities or a span-lookup seam from superseded fn-141.5.
- The lifter still reads realizations from source in this task. Nothing it reads may change.

### Investigation targets
**Required:**
- `model/framework/realize/Scripts.scala` (the scope helpers), `Typed.scala` (`Field`, `Condition`), `Realize.scala` (`Evidence`, `Observed`, `Proto`, `Command`, `family`), `Syntax.scala` and `Exploration.scala`
- `model/framework/export/Realizations.scala`: for each construct, what it reads from the tree that the value lacks; inspect only, no exporter rewrite here
- `model/temporal/realize/` and `model/temporal/features/activity_mini/realization/MiniActivityRun.scala`

### Verification
- Focused construction tests for scoped assignments, nested protobuf scopes, field paths/message descriptors, evidence operands, distinct condition operators, aliases and action-valued path selectors. Start with the affected suites and fixture tests, not the broad exporter suite.
- At task end run `make model-check-changed BASE=<committed predecessor>` once; cite the preceding task's record and mini step-table equivalence. Existing lifter outputs remain unchanged by retained runtime metadata; run only affected exporter fixtures while iterating.
- Formatting of edited Scala files. Full broad Scala gate and independent review remain deferred to the source-batch boundary; Go, regeneration, real-feature migration and lint/manual acceptance stay held.

## Acceptance
- [ ] A test constructs one value per factory and reads every argument back; it fails naming the factory and the argument.
- [ ] No factory of `model/framework/realize` or the shared kit drops an argument; deferred reflection identity fields are explicitly accounted for.
- [ ] Focused tests, formatting and the scoped gate with mini step-table equivalence pass. The current lifter's affected fixture IR remains unchanged; fn-174's oracle and broad batch gates are not claimed by this task.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
