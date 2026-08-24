---
satisfies: [R7]
---
# fn-2-agentworkflow-configuration-and-cli.6 Restore branch Go 1.27 lint compatibility

## Description
Remove the tooling blocker before Agentworkflow implementation by advancing the repository-pinned golangci-lint to the stable release with Go 1.27 generic-method support and repairing inherited merge losses exposed by current compilation: Nexus Operation TaskInvocation handlers, pointer-only UnprocessableTaskError identity, Umpire2 payload-domain size validation, workflow-resetter CHASM/HSM reapply/context plumbing, retained test-cluster call shapes and SQLite persistence option, stale matching migration assignments, and XDC call sites for absent cluster options. Preserve the existing lint configuration, enabled analyzers, comments, branch-intended behavior, and current logger/routing behavior.

**Size:** L
**Touches:** [Makefile, chasm/lib/nexusoperation/operation_tasks.go, chasm/lib/nexusoperation/cancellation_tasks.go, chasm/lib/nexusoperation/operation_tasks_test.go, chasm/lib/nexusoperation/cancellation_tasks_test.go, service/history/queues/errors/errors.go, service/history/queues/errors/errors_test.go, tools/umpire2/internal/action/reject.go, tools/umpire2/internal/action/payload_domain.go, tools/umpire2/internal/action/reject_domain_test.go, service/history/ndc/workflow_resetter.go, tests/testcore/onebox.go, tests/testcore/test_env.go, tests/testcore/test_env_test.go, tests/testcore/functional_test_base.go, service/matching/matching_engine_test.go, tests/xdc/base.go, tests/xdc/stream_based_replication_test.go]
## Acceptance
- [ ] The pinned golangci-lint version is v2.13.1 with documented Go 1.27 support.
- [ ] The former `buildir` panic on generic methods no longer occurs.
- [ ] Nexus Operation handlers satisfy the current `TaskInvocation` interface and focused tests pass with required build tags.
- [ ] `UnprocessableTaskError` has pointer-only error identity and its focused tests/vet pass.
- [ ] Umpire2 rejects payloads whose full encoded size exceeds the action limit, including metadata-heavy payloads, without restoring the removed server helper.
- [ ] Workflow resetter restores the latest intended CHASM/HSM reapply and seven-argument workflow-context behavior covered by retained focused tests.
- [ ] Test-cluster callers match the retained client and worker-service request contracts and use the retained lazy router getter.
- [ ] The retained in-memory SQLite functional-test caller is backed by a propagated persistence option and require-style helper test.
- [ ] Matching tests no longer configure the removed migration flag and their focused cases pass.
- [ ] XDC tests no longer configure absent cluster fields/options and type-check with the retained API.
- [ ] Task-scoped `make lint-code` completes with all configured linters and zero findings on a case-sensitive source filesystem; inherited default-baseline findings are documented separately.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
