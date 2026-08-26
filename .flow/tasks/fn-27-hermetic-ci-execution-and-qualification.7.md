---
satisfies: [R2, R3, R4, R5, R6, R7, R8]
---
# fn-27-hermetic-ci-execution-and-qualification.7 Run the bounded CI qualification proof

## Description
Prove R2-R8 through one real loopback CI-profile run and focused cross-layer failure controls.

**Size:** M
**Files:** `tools/umpire/ci/integration_test.go`, `tools/umpire/ci/isolation_test.go`, `tools/umpire/qualification/ci_integration_test.go`, `tools/umpire/cmd/umpire-qualify-ci/integration_test.go`, `model/Temporal/Tool/ConformanceTests.lean`
**Touches:** [tools/umpire/ci/integration_test.go, tools/umpire/ci/isolation_test.go, tools/umpire/qualification/ci_integration_test.go, tools/umpire/cmd/umpire-qualify-ci/integration_test.go, model/Temporal/Tool/ConformanceTests.lean]

### Approach

- Run one bounded real loopback CI-profile execution through the actual checker/profile siblings, publish/reopen v3, and independently assert the same experiment identity, distinct configuration/run identity, Property verdict, trust/omissions, source closure, and cleanup.
- Accept the checked-in read-only input below the workspace, then inject post-preflight tracked-tree change, resource leak/unknown, non-loopback/unknown authority, cancellation at each stage, and every output/workspace symlink, alias, ancestor, and component-replacement race; assert the exact reason/tooling boundary and no unsafe publication.
- Exercise statuses 0/1/2, status-2 evidence retention, reporting-after-publication, idempotent rerun, and changed GitHub run/attempt identity without a live GitHub dependency.
- Reopen the final set through the strict v3 reader and compare the six source bytes with their in-memory stage outputs.

### Investigation targets

**Required** (read before coding):
- `.flow/tasks/fn-20-local-execution-semantic-conformance.7.md` — bounded live proof pattern
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.7.md` — actual loopback runtime harness
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.4.md` — offline decision seam
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.5.md` — command/publication ambiguity contract
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.10.md` — strict reopen and publication recovery

### Acceptance

- [ ] The actual CI-profile loopback proof publishes/reopens one deterministic v3 result with successful cleanup and the exact self-reported non-release claim.
- [ ] Every postflight isolation row, containment race, cancellation stage, output status, and reporting ambiguity lands at the specified boundary without leaked authority or unsafe writes.
- [ ] The same input/run is idempotent, while a changed CI invocation identity yields a distinct correctly bound result.
- [ ] Focused live/integration tests pass without a live GitHub dependency.

## Acceptance
- [ ] R2-R8 bounded cross-layer proof is complete.
- [ ] All focused live-loopback, isolation, command, and reopen checks pass.
- [ ] Existing comments remain preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
