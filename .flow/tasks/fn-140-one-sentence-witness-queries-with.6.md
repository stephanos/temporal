---
satisfies: [R5, R7]
---
# fn-140-one-sentence-witness-queries-with.6 Migrate remaining eligible Models, lint triples and close equivalence gates

Touches: [model/temporal/features/nexus/**, model/temporal/features/activity/standalone/system/Dispatch.scala, model/temporal/features/activity/standalone/system/DispatchRaces.scala, model/temporal/features/activity/standalone/system/DispatchWithTaskQueue.scala, model/temporal/features/activity/standalone/system/DispatchWithWorker.scala, model/temporal/features/activity/standalone/system/RetryTimeouts.scala, model/temporal/features/activity/standalone/system/Heartbeat.scala, model/temporal/features/activity/standalone/system/ResponseByID.scala, model/temporal/features/activity/standalone/system/Reset.scala, model/temporal/features/activity/standalone/ActivitySubjectRegression.test.scala, model/temporal/features/activity/standalone/ActivityHeartbeatRegression.test.scala, model/temporal/features/activity/standalone/ActivityByIDRegression.test.scala, model/temporal/features/activity/standalone/ActivityResetRegression.test.scala, model/temporal/foundations/taskqueue/**, model/irgen/Order.scala, model/irgen/test/**, model/irgen/testdata/**, model/ir/**, model/cases/**, tools/umpire/check/*test.go, tools/umpire/lower/*test.go, tools/umpire/conformance/*test.go, tests/testcore/testpilot/testdata/generated/**, tools/canary/casebinding/testdata/**]

## Description
Complete R5's remaining Model conversion, source-aware eligibility lint and the integrated R7 template checks. Own the sealed equivalence comparison and existing fixture-consumer checks after the .4 proof and .5 documentation join. Commit the independent witness seal and hand its frozen scratch IR, Cases, lift goldens and explicit identity/provenance mapping to fn-149.4's grouping migration and fn-149.5's final source/docs/layout grouping seal, before fn-123.1 starts fault changes. The selected fn-140/fn-149/fn-123 batch's single production regeneration, full gates, independent review and live run are shared at fn-123.8; this task remains pending that close and links its evidence without dropping any acceptance obligation.

**Size:** M
**Files:** eligible Nexus, remaining Activity subject/Dispatch/composition files and task-queue level files; source admission/lint and fixtures; affected existing checker/lowerer/assessment tests and managed artifact mirrors.

### Approach
- Re-anchor to .4's passing equivalence pin on the committed fn-156 baseline, consuming fn-155's actual mapping. The full gate may retain the separately recorded inherited completion, fatal-failure and pause/resume red results; those results are neither passing evidence nor relaxed expectations. Inventory every remaining pinned find and its Property/Scenario readers before migrating. Preserve verify Queries, shared declarations, functions returning claim bundles/query lists, monitor-only claims and repeated-tail paths that require the core form. Actual custom starts in current Models feed verify, so add no witness starts surface.
- Add source-aware lint at the existing typed source/order pass (`Order.scala`) rather than the Go IR lint. Determine eligibility from the same collected references and witness admissibility the migration uses, and report the Query at its source line. A core and sugar triple are indistinguishable in emitted IR.
- Add positive/negative lint fixtures for an eligible private pinned triple, a shared Property, a shared Scenario, verify, a monitor-only claim and a repeated exact tail class. Preserve existing order/structure errors and their independent attribution.
- Run a scratch regeneration to seal equivalence after integrating .5, including scratch functional/canary Case pins only where their source Case changed. Publish the managed artifacts and changed pins once at the shared fn-123.8 boundary. Review the complete IR/Case/golden diff under the parent spec; update exact affected literal goldens without broad normalization. Do not rewrite historical recorded Run companions or replace expected assessments.
- Save complete before/after Query receipts, tables, expectation inventory and canonical Case filenames. Prove rewritten Property functions have equal truth values on every applicable model row and map edited source positions to specific declaration spans. Require the same Query answers and expected/live-replay assessments under R5, excluding only R6's named deletion. If conversion changes a protected assessment, retain the required ends predicate or stop for the owner.
- Serialize heavy checks with the machine's shared lock. Reuse applicable passing evidence, run the shared final Go suite at fn-123.8 with test_dep, -p 2 and -timeout 30m once, and capture JSON output plus elapsed wall time in task scratch. Use the model gate's skip-go-checks option to avoid duplicate Go work.

### Investigation targets
**Required:**
- `model/irgen/Order.scala` - typed source admission seam.
- `model/irgen/test/Fixtures.test.scala:544` - exact golden comparison.
- `model/temporal/features/nexus/workflow/system/System.scala` and `TrustingCaller.scala` - Nexus path claims and negative control.
- `model/temporal/features/activity/standalone/system/Dispatch.scala`, `DispatchRaces.scala`, `DispatchWithTaskQueue.scala` and `DispatchWithWorker.scala` - shared/monitor paths.
- The remaining Activity subject and regression files listed in Touches - eligible triples and their independent source assertions.
- `model/temporal/foundations/taskqueue/system/System.scala` - shared claims/query-list forms.
**Optional:**
- `MILESTONES.md` verification instructions and `model/README.md:1590` - integrated gates and managed fixture ownership.

### Quick commands
These shared close commands run at fn-123.8. The earlier .6 equivalence seal uses isolated scratch outputs and retains all comparison and live/replay proof obligations.
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
- [ ] Required model/lint/Go gates and relevant existing live/replay assessment checks are run with unchanged expectations, evidence paths and measured full-suite output, linking the shared fn-123.8 close evidence. Witness equivalence must pass. The strict inherited completion, fatal-failure and pause/resume assertions remain unchanged; their red results are recorded separately from migration regressions and remain Batch 5 obligations, never passing evidence. All other required gate and live/replay checks must pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
