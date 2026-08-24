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
Pinned golangci-lint v2.13.1 for Go 1.27 and restored the branch-intended Nexus, queue-error, payload, workflow-resetter, test-cluster, matching, and XDC compatibility lost across merges. Focused tests and task-scoped full-analyzer lint pass with zero findings on a case-sensitive source copy.

Inherited baseline: default `make lint-code` reports 1,811 pre-existing findings because the configured `main` baseline is six months and 1,384 commits behind this branch; the user explicitly approved task-scoped full-analyzer lint as this task's gate.

stage: impl-review - skipped(policy: conductor-authorized finalize preserving existing user implementation commit)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3abf20e5d1e47808df013b0318bbf4661abee76a
- Tests: cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go build ./cmd/agentworkflow, make fmt-imports, go test -tags 'disable_grpc_modules,test_dep' ./chasm/lib/nexusoperation ./service/history/queues/errors, go test -tags 'disable_grpc_modules,test_dep' ./tools/umpire2/internal/action, go test -tags 'disable_grpc_modules,test_dep' ./service/history/ndc -run '^TestWorkflowResetterSuite/(TestCherryPickChasmEvent|TestReapplyEventsHSMToChasmFallback|TestReapplyEventsHSMNotFoundDoesNotConsultChasm|TestCherryPickHSMEvent)$' -count=1, go test -tags 'disable_grpc_modules,test_dep' ./service/matching -run '^(TestAutoEnableV2ConfigChange|TestAutoEnableV2ConfigChange_NoUnloadWhenEffectiveConfigUnchanged)$' -count=1, go test -tags 'disable_grpc_modules,test_dep' ./tests/testcore -run '^(TestWithInMemorySQLitePersistence|TestClusterPool_)' -count=1, go test -tags 'disable_grpc_modules,test_dep' ./tests/xdc -run '^$', GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT_FIX=false make LOCALBIN=<host-built-tools> lint-code (case-sensitive clone; 13 analyzers; 0 issues), BASELINE_RED: make GOLANGCI_LINT_FIX=false lint-code - golangci-lint v2.12.2 buildir panic on Go 1.27 generic methods, INHERITED_RED: make lint-code - 1811 pre-existing branch-wide findings against stale main baseline 6875191ef
- PRs:
