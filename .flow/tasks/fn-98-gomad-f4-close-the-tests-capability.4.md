---
satisfies: [R5]
---
# fn-98-gomad-f4-close-the-tests-capability.4 Verify server build and stock suite, record F4

## Description
`go build ./...` with and without the gomad tag; run the named stock suite; update F4 status.

## Acceptance
- builds pass, stock suite passes, F4 status updated

## Done summary
Ran the R5 checks on darwin/arm64 and recorded F4's darwin outcome in the F4 Status of `MILESTONES.md`. The server builds with and without `-tags gomad,test_dep,disable_grpc_modules`, and the seam packages' unit tests pass with the seams at their defaults. Under the tag, five tests fail by design: three need ringpop, two need the refused password command. The stock `TestActivityAPIBatchCancelClientTestSuite` passes natively, and lint over the seam packages is clean. The Work tracking row now reads F4 done on darwin/arm64. The F8 section is untouched.

`make lint-code-fast` cannot run as written on this branch: its target set includes the nested `tools/gomad3` and `tests/mixedbrain` modules and the integration-only `tools/gomad3integration`. The local `.bin/errortype` was stale (built with go1.26) and was rebuilt with go1.27. A branch-wide lint run's `--fix` default rewrote `tools/flakereport/report.go`; that change was dropped from the commit. Follow-up: exclude nested modules from `lint-code-fast` targets.

stage: impl-review - ran [2026-09-27] triage_skip (docs-only) SHIP

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b0496ee8b6da0aea8126c950b91872cfc4ed23a8
- Tests: baseline: none (spec defines no Quick commands), CGO_ENABLED=0 go build ./... (pass), CGO_ENABLED=0 go build -tags gomad,test_dep,disable_grpc_modules ./... (pass), CGO_ENABLED=0 go test -tags test_dep -count=1 ./temporal/... ./common/config/... ./common/persistence/sql/sqlplugin ./common/archiver/provider/... ./common/persistence/visibility/store/elasticsearch/client/... ./service/worker/... ./tests/testcore/... ./service/history/workflow/update/... (pass), same packages with -tags test_dep,gomad,disable_grpc_modules: 5 tests fail by seam design (TestNewServer{,WithOTEL,WithJSONEncoding} need ringpop; TestSQLResolvePassword_Command* need the refused password command), CGO_ENABLED=0 go test -tags test_dep[,gomad,disable_grpc_modules] -count=1 -run SQLite ./common/persistence/tests (82 tests, pass both), make lint-code-fast: cannot run as written (targets nested tools/gomad3 + tests/mixedbrain modules and integration-only tools/gomad3integration -> typecheck error), make lint-code LINT_CODE_TARGETS=<seam packages> GOLANGCI_LINT_BASE_REV=$(git merge-base HEAD main): 0 issues, go vet errortype clean after rebuilding stale go1.26 .bin/errortype, CGO_ENABLED=0 go test -tags test_dep -count=1 -run ^TestActivityAPIBatchCancelClientTestSuite$ ./tests (pass, 4 subtests), flowctl gate classify: FULL (unmatched MILESTONES.md); no Quick-command gates defined
- PRs: