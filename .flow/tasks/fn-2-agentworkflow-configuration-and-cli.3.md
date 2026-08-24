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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
