---
satisfies: [R5, R6, R7]
---
# fn-89-one-contract-rule-per-entity.4 Producer fold into one instanced Rule, regenerated fixtures and pair tests

## Description
Make the Lean Producer emit one Rule per relation with one Rule instance per placement over N > 1 instances (R5), keeping per-placement lowering and its `DerivedRule` certificates unchanged and adding a fold with a shape-agreement check; then regenerate the fixtures (R6) and move the Pair model tests and the Go pair fixture tests to one Rule with two instances (R7). Regeneration lives here, not in a later task, so no commit leaves the Producer disagreeing with the committed pair fixture (a red `umpire-check-case-runtime-conformance` on a shared branch). It depends on task 3 because the regenerated pair Case must pass Go preparation and evaluation; the Lean fold can be drafted as soon as task 1 lands, but it lands after task 3.

**Size:** M (Lean fold is the substance; regeneration and test updates are mechanical)
**Files:** `model/Umpire/Case/Projection/Lowering.lean` (literal path retained on the safety shape, shape erasure/agreement, render with an instance value, folded certificate), `model/Umpire/Case/Producer.lean` (fold over the placements loop), `model/Umpire/Case/LocalNames.lean` (visit instance rule IDs), `model/Umpire/Case/Tests/Producer.lean`, `model/Temporal/Feature/Nexus/Pair/Tests.lean`, `tests/testcore/testpilot/testdata/nexusPairTests-bothComplete-case.json` (regenerated), `tests/testcore/testpilot/nexus_pair_artifact_test.go`, `tests/testcore/testpilot/nexus_pair_fixture.go`
**Touches:** [model/Umpire/Case/Projection/**, model/Umpire/Case/Producer.lean, model/Umpire/Case/LocalNames.lean, model/Umpire/Case/Tests/**, model/Temporal/Feature/Nexus/Pair/Tests.lean, tests/testcore/testpilot/**]

## Approach
- Keep `lowerRelation` (`Producer.lean:887-940`) per placement. Today the loop at `:1098-1103` flattens N lowerings; over `input.instances > 1` group each relation's N results and fold them; over one instance, or when the compared literal is boolean, emit exactly today's rules.
- Instance value naming: the ID is the last segment of the compared literal's field path. The capture shape has it in `selector.path`; `Shape.safety` keeps only `Literal` today (`Lowering.lean:163-172`), so retain the literal's path (or its field name) on the shape. The pair Case yields `operation`. Type: the field's schema singular type (the same schema `Literal.of` consults through `sideSchema`).
- Fold (in `Projection`, beside `DerivedRule` `Lowering.lean:265-274`): a new structure holding the N per-placement `DerivedRule`s (each keeps its `literals_assigned`) plus a decidable proof that their shapes agree once the literal value is erased (kind, negation, reads, capture state names, literal field and wire type). Disagreement rejects `relation.instance-shape` through the existing `productionError` path. Render once with the unsuffixed rule suffix (`FieldRelation.ruleSuffix`, not `++ placement.suffix`), the literal replaced by `Expr.instanceValue <id>`, one instance value declaration, and one Rule instance per placement (`ruleId ++ placement.suffix`, value = that placement's `Literal.wire`).
- The Rule ID stays `property.id ++ "." ++ ruleSuffix` before localization; instance IDs add the placement suffix, so after `LocalNames` they read `relation`, `relation-1`, `relation-2`. `LocalNames.visitCase` (`LocalNames.lean:323-327`) renames only `rule_id`; extend it to the instances' rule IDs, visiting in an order that keeps every other provenance row byte-stable (the pair fixture's rows at `:136-141` keep `relation-1`/`relation-2` and gain `relation`).
- Coverage (`coverage.inputs`) stays per placement; `Compiler.ContractLowering.monitor` (`Compiler.lean:54`) carries the instanced `ContractRule` unchanged in type.
- Proofs stay structural (`decideMem` at `Lowering.lean:241-255`, `cases`, `subst`); no `grind`, no `native_decide`, no 4.33-only lemmas (spec Dependencies: fn-88.12 may move to Lean 4.32.0, where `simp` rewrites `some a = some b` and the looping-simp lint is stricter).
- Lean tests: Pair tests (`Pair/Tests.lean:137-178`) assert one Rule with instance value `operation` and instances `relation-1` = `complete-1`, `relation-2` = `complete-2`, capture `nexusOperationScheduled-relation`, transitions `capture-nexusOperationScheduled-relation` and `match-nexusOperationCompleted-relation`. Producer tests add a `relation.instance-shape` rejection and a one-instance case rendering today's plain rule.
- Regenerate only through `make umpire-gen-case-runtime-conformance` (it owns the corpus and the functional fixtures). `git diff --stat` must show only the pair fixture; inside it only `contract.rules` and the Rule's `localNames` rows change (spec R6). Expand the regenerated Rule by hand (or a throwaway, uncommitted script) and diff against the old two rules: the only differences are the IDs the spec names and the literal becoming an instance value reference. The persisted-form/declaration-order check (`generate_test.go:418`) must pass (new fields in field-number order).
- Go tests: `nexus_pair_artifact_test.go:25` asserts one Rule `relation` with instances `relation-1`/`relation-2`, unsuffixed capture/transition IDs and the local-name rows; `nexus_pair_fixture.go:19-20` keeps the instance rule IDs the live test reads. `TestTestpilotNexusPairCase` (`tests/testpilot_nexus_pair_case_test.go`) passes unchanged in its Verdict assertions (it reads `verdict.GetRules()` by position and correlated evidence by rule ID and scheduled event ID, per fn-90.5).

## Investigation targets
**Required**:
- `model/Umpire/Case/Projection/Lowering.lean:160-377`
- `model/Umpire/Case/Producer.lean:220-240,870-940,1090-1135`
- `model/Umpire/Case/LocalNames.lean:300-340`
- `model/Temporal/Feature/Nexus/Pair/Tests.lean:137-178`
- `tests/testcore/testpilot/nexus_pair_artifact_test.go`, `nexus_pair_fixture.go:19-20`

**Optional**:
- `.plans/LEAN_GUIDELINES.md` — required reading before Lean work
- `model/Umpire/ARCHITECTURE.md:281-296` — typed field lowering (doc updated in task 6)

## Key context
- fn-93 A6 edits `Producer.lean:1062-1077`; keep the fold in its own declarations.
- Fixture bytes come from the Lean encoder; Go `protojson.Marshal` never writes fixtures.
- Build the way the Makefile does (`mise exec -- lake`); `make lint-model` needs `LEAN_NUM_THREADS=1`. Live tests need a physical `TMPDIR`.

## Acceptance
- [ ] over N > 1 instances a non-boolean relation lowers to one Rule with one instance value and N Rule instances; over one instance the rule is byte-identical to today's
- [ ] the folded certificate carries each placement's `DerivedRule`; no `sorry`, no new axioms; `relation.instance-shape` rejects disagreeing shapes, with a test
- [ ] only the pair fixture changes, and only in `contract.rules` and the Rule's local-name rows; `make umpire-check-case-runtime-conformance` and `make canary-check-case` pass
- [ ] Pair model tests and the pair fixture test assert one Rule with two instances; `TestTestpilotNexusPairCase` passes with unchanged Verdict assertions
- [ ] `LEAN_NUM_THREADS=1 make lint-model` shows no new findings over the baseline

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
