---
satisfies: [R3, R8]
---
# fn-72-extract-the-reusable-temporal-testpilot.3 Enforce the shared Driver dependency boundary

## Description
Replace the old path-based architecture check with direct and transitive dependency enforcement for the shared Driver tree (R3, R8). This task is independent of consumer migration once task 1 establishes the final location.

**Size:** M
**Files:** `tools/umpire/regression/ci_workflow_test.go` and focused test fixtures/helpers owned by that package
**Touches:** [tools/umpire/regression/ci_workflow_test.go, tools/umpire/regression/testdata/**]

### Approach
- Extend the established AST/import scan at `TestTestpilotOwnsCaseProtocolAndRuntime` to cover the new composite, server, worker, and delivery paths while continuing to scan generic Testpilot independently.
- Add a transitive production/test dependency closure check using the repository's Go tooling so the shared Driver cannot reach `tests/`, Umpire generators, or canary orchestration.
- Reject direct Driver imports of generic Testpilot private IR/execution/verification while allowing their expected transitive presence through the public facade.
- Preserve server/worker peer isolation and prohibit private delivery from importing either adapter.
- Add negative inputs for every forbidden edge plus unreadable or syntactically invalid inspected source; checker failures must fail closed.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/regression/ci_workflow_test.go:282-346` — current direct import and owner-path gate.
- `common/testing/testpilot/conformance_test.go` — generic conformance dependency precedent.
- `.flow/memory/bug/integration/moved-conformance-tests-must-not-import-2026-09-06.md` — required transitive test closure.

**Optional** (reference as needed):
- `tools/umpire/regression/ci_workflow_test.go:140-182` — executable documentation assertion style.
- `go.mod` — module boundary used by `go list`.
## Acceptance
- [ ] Executable checks scan `common/testing/temporaltestpilot` and reject direct or transitive production/test dependencies on repository `tests/`, Umpire generators, and canary orchestration.
- [ ] Direct Driver imports of generic Testpilot private packages are rejected, while their expected transitive presence through the public facade is admitted.
- [ ] The generic Testpilot package remains independently checked against Temporal Driver imports.
- [ ] Server-to-worker, worker-to-server, and delivery-to-either-adapter edges are rejected at the new paths.
- [ ] Table-driven negative inputs prove each forbidden edge, malformed Go input, and inspection failure is detected rather than omitted.
- [ ] The real shared and generic package trees pass the focused regression gate.
## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: