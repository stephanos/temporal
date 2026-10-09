---
satisfies: [R2, R4]
---
# fn-150-compile-temporal-server-to-wasi.7 Fix CI failures in the WASI draft PR stack

## Description
Diagnose current PR9 failed jobs, fix stack regressions, verify focused checks and publish corrections without rewriting open PR history. Preserve research files and deferred Gomad qualification boundaries.

## Acceptance
Retain failed-job causes and verification evidence; correct implementation regressions with coverage; document unrelated or remaining native qualification failures.

## Done summary
Diagnosed PR9 CI failures and corrected source selection: root unit tests now exclude archived probes, nested Gomad module packages and opt-in integration fixtures while retaining stock Gomad simulation tests. Import/YAML formatters and shellcheck preserve archival evidence and embedded runtime bytes. Registered all SQL drivers in existing cross-store visibility converter tests after reproducing the empty-registry failure. Applied mandatory inherited Go1.27 go-fix changes and verified idempotence, focused stock tests, changed-package lint, root package resolution, nested generated validation and external Runner fixture coverage. Fresh expanded review found no actionable issues; reviewers/writer same Codex family. Driver implementation unchanged. DockerHub unauthenticated pull quotas and stale modernc Gomad compatibility packs remain unresolved; full native qualification stays deferred. All seven SQLite functional jobs passed on published PR head601f before the source correction push; test-step seconds shard0=520, shard1=479, shard2=509, shard3=632, shard4=546, xdc=666, ndc=67. This does not establish a driver-only speedup. Research artifacts preserved and uncommitted.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d93b12f0f7, 10fade50d4
- Tests: Reproduced TestSQLQueryConverter empty provider registry before fix, CGO_ENABLED=0 go test -mod=readonly -tags test_dep ./common/persistence/visibility/store/tests -count=1, CGO_ENABLED=0 go test -mod=readonly -tags test_dep ./tools/gomad3sim -count=1, TMPDIR=<canonical workspace scratch> CGO_ENABLED=0 go test -mod=readonly -tags test_dep ./cmd/tools/lintcode -count=1, make shell-check, Go/import/YAML formatter idempotence verification, Root246package read-only Go package resolution with test_dep, make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=HEAD lint-code-fast, make -C tools/gomad3 validate, GOTOOLCHAIN=go1.27.1 go -C tools/gomad3 test -tags test_dep -run ^TestRunnerRequestsCompileInExternalModule$ -count=1 ., Both-source-set Go package listing for gomad3sim
- PRs: https://github.com/stephanos/temporal/pull/9, https://github.com/stephanos/temporal/pull/10