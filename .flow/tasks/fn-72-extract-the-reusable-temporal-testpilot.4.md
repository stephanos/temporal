---
satisfies: [R8, R9]
---
# fn-72-extract-the-reusable-temporal-testpilot.4 Align ownership docs, package gates, and final verification

## Description
Update current ownership records and every exact package selector, then run the complete extraction verification (R8, R9). Keep historical migration evidence unchanged.

**Size:** M
**Files:** `.plans/UMPIRE4_{SPEC,COMPONENTS,ORDER}.md`, `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md`, `model/{README,ARCHITECTURE}.md`, `common/testing/{testpilot,temporal}/README.md`, `tests/testcore/testpilot/README.md`, `.flow/specs/fn-70-*.md`, `Makefile`, `.github/workflows/umpire.yml`, `tools/umpire/{regression/ci_workflow_test.go,internal/retiredvocabulary/check.go}`
**Touches:** [.plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_COMPONENTS.md, .plans/UMPIRE4_ORDER.md, .plans/UMPIRE_CASE_RUNTIME_DESIGN.md, model/README.md, model/ARCHITECTURE.md, common/testing/testpilot/README.md, common/testing/temporaltestpilot/README.md, tests/testcore/testpilot/README.md, .flow/specs/fn-70-scheduled-canary-proof-of-concept-as-a.md, Makefile, .github/workflows/umpire.yml, tools/umpire/regression/ci_workflow_test.go, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- Correct MOD-13 and active architecture/readme ownership to the shared composite/server/worker tree while preserving the stable rule and authority split.
- Recast the old testcore README around fixture admission/generation and functional provisioning; keep current Driver prose with the new package.
- Update the exact Make, CI, regression, and vocabulary selectors to include `./common/testing/temporaltestpilot/...` while retaining `./tests/testcore/testpilot/...` for fixture tests and the full inherited live selector.
- Update the downstream fn-70 README reference through Flow state, and move fn-72 into completed cutovers in the delivery order only after all gates pass.
- Leave the architecture review, fn-68/fn-69/fn-71 records, migration ledger, and cleanup inventory as historical evidence.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md:148-158` — normative MOD-12 through MOD-14 rules.
- `.plans/UMPIRE4_COMPONENTS.md:25-52` — active owner map and historical boundary.
- `tools/umpire/regression/ci_workflow_test.go:17-182` — exact CI/package/docs assertions.
- `Makefile:1065-1170` — fixture, package-local, live, and aggregate gates.
- `.github/workflows/umpire.yml:30-38` — package-local and live CI selection.

**Optional** (reference as needed):
- `.flow/memory/bug/integration/full-integration-gates-must-select-the-2026-09-04.md` — complete selector constraint.
- `tools/umpire/CLEANUP_INVENTORY.md` — historical record to leave untouched.

## Acceptance
- [ ] MOD-13 and all active package/architecture docs name the shared Driver owners without changing the server/worker/internal-execution authority contract.
- [ ] Driver docs live with the shared package; the retained testcore README accurately owns fixtures, admission/reuse tests, provisioning, and generator destination.
- [ ] Make, CI, regression dry-run assertions, and vocabulary scans include the shared Driver packages and retain generic Testpilot, fixture tests, the complete live selector, and exact inherited failure baseline.
- [ ] Focused shared Driver, generic Testpilot, fixture, dependency, and architecture suites pass with `-tags test_dep`; the complete Umpire Go gate and `make lint-code` are recorded.
- [ ] The established live functional check runs with `-tags 'test_dep integration'` and repository persistence configuration, with any missing service/compiler/resource reported as an explicit failed or blocked verification step.
- [ ] Historical review, migration, task, and cleanup evidence remains unchanged.

## Done summary
Aligned MOD-13 and active ownership documentation with the shared `common/testing/temporaltestpilot` Driver tree, restored fixture-focused documentation at the retained test path, and extended Make, CI, regression, and retired-vocabulary selectors without weakening the full live selector or inherited failure baseline. Updated the fn-70 reference through Flow state and reconciled the completed fn-71/fn-72 cutovers in the delivery order while preserving historical evidence files.

The focused package and architecture suites, complete serial Umpire Go package gate, exact live Make target, diff check, and Go formatting check passed. `make lint-code` was invoked once and stopped when free disk fell to 121 MiB; this is recorded as an explicit resource-blocked verification step. The task-scoped implementation review reached SHIP after the recorded verification evidence was made visible. No staging or commit was performed by user instruction.

stage: impl-review - ran (model: gpt-5.6-sol medium, SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./common/testing/temporaltestpilot/... ./tests/testcore/testpilot), baseline: green (TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./tools/umpire/regression -run 'Test(TestpilotOwnsCaseProtocolAndRuntime|UmpireCIWorkflowRunsSeparatedUnitAndLiveProofs)'), TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/... ./common/testing/temporaltestpilot/... ./tests/testcore/testpilot, TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./tools/umpire/regression -run 'Test(TestpilotOwnsCaseProtocolAndRuntime|TestpilotDependencyBoundaryRejectsForbiddenEdges|UmpireCIWorkflowRunsSeparatedUnitAndLiveProofs|UmpireDocumentationStatesAttachedOwnershipAndBoundedClaim)', TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./common/testing/temporaltestpilot/... ./tests/testcore/testpilot/..., TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 make umpire-check-live-tests, git diff --check, gofmt -d tools/umpire/regression/ci_workflow_test.go tools/umpire/internal/retiredvocabulary/check.go, BLOCKED: TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 make lint-code (stopped after one invocation when free disk reached 121 MiB), impl-review: SHIP (codex:gpt-5.6-sol:medium; /tmp/impl-review-receipt-fn-72.4.json)
- PRs:
