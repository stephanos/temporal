---
satisfies: [R4, R7]
---
# fn-2-agentworkflow-configuration-and-cli.7 Restore legacy single-dash CLI flag compatibility

## Description
Resolve the spec-wide implementation review P1 by preserving the former Go flag parser single-dash long-form syntax across the Cobra command tree. Normalize legacy `-name value` and `-name=value` forms before Cobra parses arguments while retaining current double-dash flags, help, positional validation, stream isolation, and usage-error classification. **Size:** S **Touches:** [tools/agentworkflow/internal/cli/cli.go, tools/agentworkflow/internal/cli/cli_test.go]

## Acceptance
- [ ] Compatibility tests first fail for legacy single-dash long flags and equals forms.
- [ ] Every command accepts its former single-dash long flag spellings without introducing global Cobra state.
- [ ] Double-dash flags, unknown-flag usage errors, positionals, output streams, and exit codes remain compatible.
- [ ] Full tagged module tests/build and task-scoped configured lint pass.
## Done summary
Restored the Go flag parser's registered single-dash long forms across the fresh Cobra command tree. The per-run normalizer supports `-name value` and `-name=value`, copies caller arguments, skips flag values and `--` positionals, and preserves double-dash flags, `-h`/help, unknown-flag usage classification, streams, and exit codes.

TDD captured the expected unknown-shorthand RED before the focused compatibility suite turned GREEN. Complete tagged tests, build, vet, race, gofmt/gci, and task-scoped case-sensitive lint all pass; the lint retry only followed generated-cache ENOSPC and then ran all 13 analyzers with zero issues.

stage: impl-review - ran | verdict: SHIP | session: 01a03560-b87e-7b12-bb5d-60e9dbadbe0d
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 296339b8c957f9b19bd24e0bfabbd5f4076be938, 8294d1af93e749b741087c7ad39db19bf9edb501
- Tests: baseline: green (cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./...; cd tools/agentworkflow && GOWORK=off go build ./cmd/agentworkflow; make fmt-imports), INHERITED_RED: make lint-code - case-insensitive Temporal/environment versus temporal/environment checkout collision; user-approved task-scoped case-sensitive lint is authoritative, RED: cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./internal/cli -run 'TestCLI(AcceptsLegacySingleDashLongFlags|LegacyFlagNormalizationPreservesValuesAndSeparator|UsageErrorsStayOnStderr)$' -count=1 - legacy single-dash flags failed as unknown Cobra shorthands, RED: cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./internal/cli -run 'TestCLIHelpAliasesExposeCommandTree/legacy_long_help_flag$' -count=1 - legacy -help failed with usage status, GREEN: cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./internal/cli -run 'TestCLI(AcceptsLegacySingleDashLongFlags|LegacyFlagNormalizationPreservesValuesAndSeparator|UsageErrorsStayOnStderr|HelpAliasesExposeCommandTree|ClassifiesOutputFailure)$' -count=1, cd tools/agentworkflow && GOWORK=off go test -count=1 -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go build -o /tmp/agentworkflow-fn2-task7 ./cmd/agentworkflow, cd tools/agentworkflow && GOWORK=off go vet -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go test -count=1 -race -tags test_dep ./..., find tools/agentworkflow -type f -name '*.go' -print0 | xargs -0 gofmt -d, .bin/gci-v0.13.6 diff --skip-generated -s standard -s default tools/agentworkflow, GOLANGCI_LINT_BASE_REV=a0603e78bc84ed77c65927cbeca18b11ca7ab7a1 GOLANGCI_LINT_FIX=false make LOCALBIN=/tmp/fn2-5-lint.VBkjDB/tools lint-code (case-sensitive clone at 296339b8c; 13 analyzers; 0 issues; one identical retry after clearing generated caches following ENOSPC), FINAL: cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./..., FINAL: cd tools/agentworkflow && GOWORK=off go build -o /tmp/agentworkflow-fn2-task7-final ./cmd/agentworkflow, FINAL: make fmt-imports, NO_RECEIPT: unittest gate receipt not warrantable because unrelated config/development.yaml is dirty, CONCURRENT_HEAD: 8294d1af93e749b741087c7ad39db19bf9edb501 is conductor-owned Flow metadata plus unrelated simulation-contract work committed after the reviewed task head; Codex review receipt head is 296339b8c957f9b19bd24e0bfabbd5f4076be938
- PRs:
