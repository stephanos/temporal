---
satisfies: [R1, R2, R5]
---
# fn-149-safety-and-liveness-groups-for-object.4 Migrate Model claims and capability references with behavior pins

## Description
Migrate Model claims and capability references with behavior pins. Advances R1, R2, R5 of the parent spec.

**Size:** M
**Files:** `model/temporal/**`, `model/irgen/Capabilities.scala`, `model/irgen/Structure.scala`, `model/irgen/testdata/**`, `tools/umpire/check/*test.go`, `model/ir/**`, `model/cases/**`
**Touches:** [model/temporal/**, model/irgen/Capabilities.scala, model/irgen/Structure.scala, model/irgen/testdata/**, tools/umpire/check/*test.go, model/ir/**, model/cases/**]

### Approach
- Inventory declarations from the sealed baseline, including capabilities, inherited claims, helper bundles, derived designs and compositions. Classify by declaration kind, not names such as completes.
- Migrate all live authored properties to groups, update references/export selections and activate flat-layout rejection. Keep monitors in their existing declaration/attachment sections and group only their claim references.
- Consume fn-140's completed witness vocabulary; do not recreate extracted witness-only Properties. Re-anchor package paths after fn-142/fn-143.
- Use the existing nexus_close_baseline and declaration/Model pin helpers to compare tables, predicate truth on applicable rows, Query/progress receipts and canonical Case meaning. Preserve negative controls and account individually for changed qualified function names, identities and source spans.
- Regenerate under the existing shared lock, updating only artifacts caused by this migration. Task 5 owns integrated full gates.

### Investigation targets
**Required:**
- `model/temporal/features/nexus/workflow/system/ClosePolicy.scala:617` - mixed claims and per-design bundles.
- `model/temporal/features/activity/standalone/system/System.scala:589` - composition safety.
- `model/irgen/Capabilities.scala:410` - generated law registration.
- `tools/umpire/check/nexus_close_baseline_test.go:344` - behavior baseline.
- `model/irgen/Structure.scala` - final layout refusal.

### Key context
Re-anchor paths and interfaces against completed fn-140/fn-141 and the approved schema/package moves before editing. Keep this work outside the activity batch and serialize shared regeneration with it and the schema chain; no new spec-close dependency is implied.

### Quick commands
```bash
make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks
go test -tags test_dep ./tools/umpire/check -run 'Test.*(Close|Activity|Capabilities|Declarations|Composed)'
```

## Acceptance
- [ ] All authored Models and shared law references use the final grouping, with a refusal fixture for the retired flat form.
- [ ] Baseline comparison preserves tables, predicates, existing Query/progress verdicts and Case meaning, including negative controls.
- [ ] Every changed generated identity/provenance span has a declared cause; monitor attachment and shared-law counts are unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
