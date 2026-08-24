---
satisfies: [R5]
---
# fn-2-agentworkflow-configuration-and-cli.3 Internalize the Agentworkflow engine and test helper

## Description
Move the unconsumed root Go library and backend test helper under internal package boundaries for R5. Keep package behavior and all existing comments intact while updating module-local imports.

**Size:** M
**Files:** root `tools/agentworkflow/*.go` implementation/tests/examples, `tools/agentworkflow/backendtest/**`, `tools/agentworkflow/internal/agentworkflow/**`, `tools/agentworkflow/internal/backendtest/**`, module-local importers
**Touches:** [tools/agentworkflow/*.go, tools/agentworkflow/backendtest/**, tools/agentworkflow/internal/agentworkflow/**, tools/agentworkflow/internal/backendtest/**, tools/agentworkflow/internal/backend/**, tools/agentworkflow/internal/quality/**, tools/agentworkflow/cmd/agentworkflow/**]

### Approach
- Move the root engine, workflow contracts/results, implementation, tests, examples, and package documentation into `internal/agentworkflow`.
- Move `backendtest` into `internal/backendtest`.
- Update provider, quality, CLI, integration-test, and test-helper imports without creating compatibility aliases.
- Preserve package names and existing comments unless a moved path makes a package comment inaccurate.

### Investigation targets
**Required** (read before coding):
- `tools/agentworkflow/doc.go` — current package declaration
- `tools/agentworkflow/engine.go` — engine package dependencies
- `tools/agentworkflow/backend_example_test.go` — public example to internalize
- `tools/agentworkflow/backendtest/backendtest.go:151-157` — shared invocation fixture
- `tools/agentworkflow/cmd/agentworkflow/main.go:15-18` — command imports

**Optional** (reference as needed):
- `tools/agentworkflow/integration_test.go` — external-package integration coverage

### Acceptance
- [ ] No Go package outside the module can import the internalized implementation.
- [ ] All module-local imports resolve without a root compatibility package.
- [ ] Existing comments and test coverage survive the move.
- [ ] The complete nested module compiles and focused tests pass with `-tags test_dep`.

## Acceptance
- [ ] R5 internal package boundary is enforced.
- [ ] No stale root-package imports or compatibility shims remain.


## Done summary
Moved the Agentworkflow engine, workflow contracts/results, tests, examples, and backend conformance helper behind `internal/` boundaries while preserving their contents and updating only module-local imports. Structural characterization changed from an externally importable root package to an enforced Go `internal` boundary with no root compatibility surface.

The complete tagged module tests, command build, focused formatting/import checks, tagged vet and race suites, and the task-scoped 13-analyzer lint pass. The canonical branch-wide formatter/lint comparison remains the explicitly approved inherited baseline exception.

stage: impl-review - ran | verdict: SHIP | session: 01a03516-77b5-79a0-a383-4bd290323e0d
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b743906171cba9b3344b1c4e23d2fc4e35ca8b4e
- Tests: baseline: green (cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./...), structural RED: root package externally importable and internal package directories absent, structural GREEN: no root Go surface or stale imports; external import rejected with use of internal package not allowed, cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go build ./cmd/agentworkflow, make fmt-imports (Agentworkflow paths clean; inherited unrelated formatter diff approved), cd tools/agentworkflow && GOWORK=off go vet -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go test -race -tags test_dep ./..., GOLANGCI_LINT_BASE_REV=74f03cc8eee8b8d25f1507fa9e65a8c97398f82e GOLANGCI_LINT_FIX=false make lint-code (13 analyzers, 0 issues, disposable case-sensitive clone), NO_RECEIPT: unittest gate receipt not warrantable because shared checkout has pre-existing config/development.yaml changes
- PRs:
