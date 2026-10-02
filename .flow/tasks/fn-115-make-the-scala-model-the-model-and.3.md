---
satisfies: [R2, R3]
---
# fn-115-make-the-scala-model-the-model-and.3 Transfer parity claims to goldens and remove legacy oracle imports

## Description
Transfer parity claims to goldens and remove legacy oracle imports. Implements R2, R3 using the reviewed parent contracts.

**Size:** M
**Files:** model/scalav2/goir parity tests and golden tests; parity claim coverage inventory
**Touches:** [model/scalav2/goir/*test.go, model/scalav2/goir/testpilot/**, model/scalav2/goir/testdata/**, .plans/umpire-migration-*.json]

### Approach
- The lowerer may expose its existing local Query/realization conversion through a narrow bridge in `*_test.go` only for its external golden test package. Reuse the frozen original Nexus IR with exactly the five existing comparative history Exhaustive adjustments and corresponding asyncNexus/controller/history closing-read removals, plus captured original Source/Identity (both kind vectors must equal temporal.nexus.caller.evidence.{started,completed,failed,canceled,timedOut} in that order); full original-oracle equality must remain strict. No production API or reader Testpilot edge is added.
- Retire the temporary refined-row oracle comparison in migration_golden_test.go while keeping immutable refined-properties verification; its activityModel/goPropertyRows/rowKeyOf/rowSide dependencies must disappear with the legacy oracle. Preserve the complete capture entrypoint and transfer original Nexus typed/keyed snapshots through equivalent admitted-IR inputs. Keep the generic job-only checker builders and immutable job comparison here; their admitted-IR construction and exact row permutation belong to task 5 with the producer copy.
- Retain the shared `tableSide`/`sideOf` helper now used by the immutable reader goldens; move it to a dedicated test-support file when retiring `activity_parity_test.go`. Remove only superseded oracle comparisons and imports, not this live golden dependency.
- Transfer the seven typed/keyed Nexus producer parity claims into live lowerer admitted-IR fixture comparisons against task 2’s frozen original evidence. Add those replacement tests under the current live lowerer; leave original `model/go/caseproducer` tests byte-identical for the archive. Task 5 omits superseded oracle imports only from its new producer copy.
- List each test comparing handwritten activity, Nexus or worker models and each optional legacy dump comparison. Map its actual claim to an established golden assertion before removing it.
- Preserve disabled pairs, every-row Property evaluation and mutation sensitivity, intentional old-model differences, refinements and witnesses. Add missing golden coverage instead of merely deleting comparisons.
- Remove superseded comparison imports and dump-path skips only once the claim table is complete. Keep focused generic semantic tests and native Scala evaluator tests; the latter are fn-113 scope.
- Run the golden and reader suites, then inspect test dependency graphs to prove the old oracle packages are no longer required by live reader tests.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `model/scalav2/goir/activity_parity_test.go`
- `model/scalav2/goir/activity_properties_test.go:278`
- `model/scalav2/goir/nexus_close_baseline_test.go:117`
- `model/scalav2/goir/nexus_close_test.go`
- `model/scalav2/goir/isolation_test.go`

### Quick commands
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir/...; mise exec -- go list -tags test_dep -deps -test ./model/scalav2/goir/...; make lint-code-fast

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.
## Acceptance
- [ ] Every removed oracle comparison has a specific golden carrying its complete claim; mutation checks still detect the relevant changes.
- [ ] Live reader tests no longer import handwritten oracle packages or read legacy dumps.
- [ ] Native Scala evaluator and focused semantic tests remain, and the complete golden verification passes.

## Done summary
Transferred retired handwritten Model and optional dump comparisons to immutable evidence with explicit surviving-claim mappings. Reader test closure has zero Testpilot or handwritten Model dependencies; complete reader/lowerer closure has zero handwritten Model dependencies. The generic checker/producer stay until their task-5 copy. All original typed/keyed Nexus Case/table/Query values now reproduce through admitted IR and a narrow test-only lowerer bridge. Exact paired five-kind comparative realization adjustments are documented; no broad normalization. Full original/mapped/oracle verification, all five off-path activity mutants, source/declaration/specimen checks and complete capture entrypoints remain.

The durable .plans/umpire-migration-claims.json preserves claim coverage and 33 retired/revised commentary groups verbatim with original source attribution. All1411 compressed goldens,843 protected originals and index are unchanged. No production code, Scala, runtime fixture or source location changed. Generic job admitted-IR conversion remains task5 scope.

Focused goir/... tests passed (reader150.657s lowerer213.675s), required dependency listing and lint0issues passed; targeted red/green and preservation evidence is in .flow/tmp/fn115-3-evidence.json. Independent read-only Codex gpt-6.1-sol high returned SHIP in round1, verifying snapshots/originals/comments/index; receipt .flow/tmp/fn115-3-review/receipt.json. Exact uncommitted scope adaptation preserves user no-commit constraints; independent context, same family.

stage: implementation - ran (model: gpt-6-astra); existing agent reused under host thread limit with disk re-anchor.
stage: verification - ran (focused Quick gates, dependency boundaries and preservation).
stage: impl-review - ran (model: gpt-6.1-sol); SHIP round1.
stage: plan-sync - skipped(config: planSync.enabled != true); bounded fixture decisions recorded in map and task brief.
stage: tracker-sync - skipped(bridge inactive).
stage: commit - skipped(user reserves commits and staging).
## Evidence
- Commits:
- Tests: CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir -run '^TestReaderAndLowererDoNotImportHandwrittenOracles$' -count=1, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir/testpilot -run '^TestOriginalNexusFixturesThroughAdmittedIR$' -count=1, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir/testpilot -run '^TestOriginalNexusFixturesThroughAdmittedIR$' -count=1, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir -run '^(TestActivityEveryClaimDeclarationIsLifted|TestActivityPropertiesAgreeOnEveryRow)$' -count=1, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir/testpilot -run '^(TestOriginalNexusFixturesThroughAdmittedIR|TestALoweredCaseIsTheComparativeGoModelsCase)$' -count=1, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir -run '^(TestReaderAndLowererDoNotImportHandwrittenOracles|TestActivityEveryClaimDeclarationIsLifted|TestActivityTablesAccountForEveryPair|TestMigrationRefinedPropertiesCoverEveryProductPropertyRow)$' -count=1, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir/..., CC=/usr/bin/clang mise exec -- go list -tags test_dep -deps -test ./model/scalav2/goir/..., CC=/usr/bin/clang mise exec -- go list -tags test_dep -deps -test ./model/scalav2/goir, make lint-code-fast
- PRs: