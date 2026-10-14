# fn-27-hermetic-ci-execution-and-qualification.3 Reuse the canonical Run Evaluation authority in CI

## Description
Feed the CI run's admitted Evidence to the same fixed Run Evaluation boundary used locally. Preserve Execution, Observation Evaluation, Implementation Link, Property, cleanup, and tooling outcomes independently. CI code and workflow YAML must not interpret Evidence, translate System facts to Feature facts, or evaluate Properties.

## Acceptance
- [ ] CI uses the shared runner and Run Evaluation API without a CI-specific mapper or evaluator.
- [ ] Equivalent local and CI Evidence produces the same stable meaning and Behavior Fingerprints.
- [ ] Non-success and malformed Evidence remain distinct and fail closed at their owning boundaries.

## Done summary
Generated ordinary CI portability test now materializes the runner's admitted execution set and hands it opaquely to the existing `umpire-check-local-run-evaluation` target, preserving the verified installed-checker digest boundary without a CI semantic mapper or evaluator. The strict direct-`Check` seam was replanned after proving that an ordinary `go test` executable cannot satisfy the fixed sibling/digest invariant; all parent Quick commands pass, while repository `make lint-code` remains inherited red only at unchanged `tools/umpire/runtime/errors.go:60` (`et:unw+`) after reporting zero task-diff issues.

stage: impl-review - ran [2026-08-31T17:55:23Z..2026-08-31T18:02:07Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 62eb6a0374125997fbab2f515be6556b5633bcb8
- Tests: baseline: green - cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests, baseline: red inherited - mise exec -- go test -count=1 ./tools/umpire/runtime/... ./tools/umpire/runevaluation/... ./tools/umpire/temporal/nexus/... (mise-selected Lean clang lacked the macOS SDK; green with repository-established Xcode CC/CXX/SDKROOT and physical TMPDIR), baseline: red inherited - mise exec -- go test -count=1 ./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$' (same Darwin toolchain cause; green with repository-established Xcode CC/CXX/SDKROOT and physical TMPDIR), baseline: green - mise exec -- make umpire-check-regression, STRICT_RED: mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-tests-go -run '^TestRenderGeneratedRunnerTestPinsHermeticSubjectBeforeRuntimeIO$' (generated private runCallerClosureEvaluation handoff absent), mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-tests-go, mise exec -- go test -count=1 -tags test_dep ./tools/umpire/runevaluation -run '^(TestCheckWithCheckerRejectsResponseDriftWithoutASet|TestCheckWithCheckerAdmitsTheCompleteIndependentStatusMatrix|TestRunFixedCheckerRequiresInstalledDigest)$', mise exec -- go test -count=1 -tags test_dep ./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$', cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests, mise exec -- go test -count=1 ./tools/umpire/runtime/... ./tools/umpire/runevaluation/... ./tools/umpire/temporal/nexus/..., mise exec -- go test -count=1 ./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$', mise exec -- make umpire-check-regression, GOLANGCI_LINT_BASE_REV=7f545326687bfc3c2ffbbb304ea65a82e93582f6 GOLANGCI_LINT_FIX=false mise exec -- make lint-code (golangci: 0 task-diff issues; INHERITED_RED: tools/umpire/runtime/errors.go:60:9 et:unw+), git diff --check
- PRs:
