---
satisfies: [R5, R7]
---
# fn-140-one-sentence-witness-queries-with.6 Migrate remaining eligible Models, lint triples and close equivalence gates

Touches: [model/temporal/features/nexus/**, model/temporal/features/activity/standalone/system/Record.scala, model/temporal/features/activity/standalone/system/WithTaskQueue.scala, model/temporal/shared/taskqueue/**, model/irgen/Order.scala, model/irgen/test/**, model/irgen/testdata/**, model/ir/**, model/cases/**, tools/umpire/check/*test.go, tools/umpire/lower/*test.go, tools/umpire/conformance/*test.go, tests/testcore/testpilot/testdata/generated/**, tools/canary/casebinding/testdata/**]

## Description
Complete R5's remaining Model conversion, source-aware eligibility lint and the integrated R7 template checks. Own the final regeneration and existing fixture-consumer checks after the .4 proof and .5 documentation join.

**Size:** M
**Files:** eligible Nexus, Activity Record/composition and task-queue level files; source admission/lint and fixtures; affected existing checker/lowerer/assessment tests and managed artifact mirrors.

### Approach
- Re-anchor to .4's passing pin. Inventory every remaining pinned find and its Property/Scenario readers before migrating. Preserve verify Queries, shared declarations, functions returning claim bundles/query lists, monitor-only claims and repeated-tail paths that require the core form. Actual custom starts in current Models feed verify, so add no witness starts surface.
- Add source-aware lint at the existing typed source/order pass (`Order.scala`) rather than the Go IR lint. Determine eligibility from the same collected references and witness admissibility the migration uses, and report the Query at its source line. A core and sugar triple are indistinguishable in emitted IR.
- Add positive/negative lint fixtures for an eligible private pinned triple, a shared Property, a shared Scenario, verify, a monitor-only claim and a repeated exact tail class. Preserve existing order/structure errors and their independent attribution.
- Run the existing final regeneration once after integrating .5, then regenerate functional/canary Case pins only where their source Case changed. Review the complete IR/Case/golden diff under the parent spec; update exact affected literal goldens without broad normalization. Do not rewrite historical recorded Run companions or replace expected assessments.
- Save complete before/after Query receipts, tables, expectation inventory and canonical Case filenames. Prove rewritten Property functions have equal truth values on every applicable model row and map edited source positions to specific declaration spans. Require the same Query answers and expected/live-replay assessments under R5, excluding only R6's named deletion. If conversion changes a protected assessment, retain the required ends predicate or stop for the owner.
- Serialize heavy checks with the machine's shared lock. Reuse applicable passing evidence, run the final Go suite with test_dep, -p 2 and -timeout 30m once, and capture JSON output plus elapsed wall time in task scratch. Use the model gate's skip-go-checks option to avoid duplicate Go work.

### Investigation targets
**Required:**
- `model/irgen/Order.scala` - typed source admission seam.
- `model/irgen/test/Fixtures.test.scala:544` - exact golden comparison.
- `model/temporal/features/nexus/workflow/system/System.scala` and `TrustingCaller.scala` - Nexus path claims and negative control.
- `model/temporal/features/activity/standalone/system/Record.scala` and `WithTaskQueue.scala` - shared/monitor paths.
- `model/temporal/shared/taskqueue/system/System.scala` - shared claims/query-list forms.
**Optional:**
- `MILESTONES.md` verification instructions and `model/README.md:1464` - integrated gates and managed fixture ownership.

### Quick commands
```bash
make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-cases umpire-check-fixtures canary-check-case
make lint-model
go test -tags test_dep -p 2 -timeout 30m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...
```

## Acceptance
- [ ] Every remaining eligible Model triple is converted and each retained form has a concrete R5/Boundaries justification; source lint diagnoses only eligible triples.
- [ ] Refusal fixtures pin source ownership, eligible/shared/monitor/verify cases and repeated input-bearing tail identities without changing existing refusal classes.
- [ ] Full Query/table/expectation/Case inventory comparison, per-row Property truth equivalence and an explicit edited-source position map prove R5 equivalence and only the R6 named deletion; final template lifting proves R7.
- [ ] Regenerated IR, Cases, lift goldens and changed functional/canary pins pass their existing byte checks, with any unsupported difference reported to the owner.
- [ ] Required model/lint/Go gates pass with evidence paths and measured full-suite output; relevant existing live/replay assessment checks pass under unchanged expectations.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
