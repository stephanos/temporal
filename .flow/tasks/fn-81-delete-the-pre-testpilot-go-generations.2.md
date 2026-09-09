---
satisfies: [R2, R3, R6]
---
# fn-81-delete-the-pre-testpilot-go-generations.2 Remove the legacy white-box seam from the history service and test harness

## Description
Implements R6 plus the test, genmodels, and umpire-tree parts of R2 and R3 (spec §Commit order 1 to 3). Three commits: first the legacy tests and the one retained monitor reader, then `cmd/umpire-genmodels` with its hooks, then the seam and the three umpire trees together, because the history service imports umpire1, the harness imports umpire2, and both trees import `common/testing/umpire`. Every commit compiles under the tagged build.

**Size:** M
**Files:** `tests/umpire2_*.go`, `tests/umpire3_*.go`, `tests/lost_task_test.go`, `tests/probe/`, `tests/gomadfunctional/`, `cmd/umpire-genmodels/`, `mise.toml`, `develop/umpire/install-tools.sh`, `service/history/workflow/cache/cache.go` (the only seam importer under `service`), comment-only edits in `service/history/api/respondworkflowtaskcompleted/workflow_task_completed_handler.go`, `service/history/api/startworkflow/api.go`, `service/history/api/updateworkflow/api.go`, `service/history/ndc/workflow_resetter.go`, `service/history/workflow/retry.go`, `service/history/workflow/update/util.go`, `tests/testcore/functional_test_base.go`, `tests/testcore/test_env.go`, `tests/testcore/test_env_test.go`, `tests/testcore/monitor/**` (delete), `common/testing/umpire/**` (delete), `tools/umpire1/`, `tools/umpire2/`, `tools/umpire3/`, `tools/umpire/vocabulary/retired_vocabulary_test.go`, `.github/CODEOWNERS` (line 98)
**Touches:** [tests/umpire2_*.go, tests/umpire3_*.go, tests/lost_task_test.go, tests/probe/**, tests/gomadfunctional/**, cmd/umpire-genmodels/**, mise.toml, develop/umpire/**, service/history/**, tests/testcore/*.go, tests/testcore/monitor/**, common/testing/umpire/**, tools/umpire1/**, tools/umpire2/**, tools/umpire3/**, tools/umpire/vocabulary/retired_vocabulary_test.go, .github/CODEOWNERS]

### Approach
- Commit 1: `git rm` the nine `tests/umpire[23]_*.go` files, `tests/lost_task_test.go` (branch-only, uses `AllowMonitorViolations` and `GetMonitor().CheckNamespace` at `:45,102,132,155,200,239`; record its lost-task property in the ledger as a future Testpilot candidate), `tests/probe`, `tests/gomadfunctional`. Build with tags.
- Commit 2: `git rm -r cmd/umpire-genmodels`; remove `mise.toml:6-13` tasks and `develop/umpire/install-tools.sh` (its only consumer). Build.
- Commit 3, one commit:
  - `service/history/workflow/cache/cache.go`: remove the `umpireotel` and `tools/umpire1/model` imports (`:27,32`) and the `umpireotel.Instrument`, `RecordError`, `RecordFact`, and `EntityTag` calls (`:342-432`). It is the only importer under `service`. The six other history-service files carry only comments that mention the umpire observer and `.plans/UMPIRE.md` (a file that does not exist); rewrite or drop those comment lines and change no code. Keep every other branch change; do not copy upstream files over.
  - `tests/testcore/functional_test_base.go`: remove the `testmonitor` and `umpire2` imports (`:50-51`), the `monitor` and `monitorViolationsExpected` fields (`:72-76`), the `UmpireMonitorFactory` param and `withUmpireMonitorFactory` (`:117,213-216`), `GetMonitor` (`:279-284`), `RequireRulePassed` and `PassedKeys` use (`:286-292`), the factory selection and interceptor install (`:340-352`), `defaultUmpireMonitorFactory` (`:407-408`), the teardown purge (`:536`), `CheckAndPurgeMonitor` (`:544-562`), and `AllowMonitorViolations` (`:566-570`). `tests/testcore/test_env.go`: remove `WithUmpireMonitorFactory` (`:118-127`) and the purge call (`:345-347`). `tests/testcore/test_env_test.go`: remove the factory test block (`:71-97`).
  - Delete `common/testing/umpire`, `tests/testcore/monitor`, `tools/umpire1`, `tools/umpire2`, `tools/umpire3`; repoint the fixture path at `tools/umpire/vocabulary/retired_vocabulary_test.go:91` from `tools/umpire3/history.go` to a neutral temp-dir name; drop CODEOWNERS line 98.
  - Re-run `git grep -n 'GetMonitor\|CheckAndPurgeMonitor\|WithUmpireMonitorFactory\|AllowMonitorViolations\|RequireRulePassed\|umpireotel' -- service tests tools common` and require no hits.
- Verify after each commit: `go build -tags 'test_dep integration' ./...`. After commit 3: `go vet -tags test_dep ./service/history/... ./tests/... ./tools/umpire/...`, `CGO_ENABLED=0 go test -tags test_dep ./tests/testcore/... ./tools/umpire/vocabulary/...`, then the live command `go test -v -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilot'` against a cluster.

### Investigation targets
**Required** (read before coding):
- `service/history/workflow/cache/cache.go:330-440` — instrumentation to strip
- `tests/testcore/functional_test_base.go:270-290,345-360,400-410,530-570` — monitor wiring
- `tests/testcore/test_env.go:110-130,340-350` — factory and purge helpers

**Optional** (reference as needed):
- `git diff main -- service/history` — to distinguish instrumentation lines from unrelated branch divergence
- `.flow/memory/bug/integration/full-integration-gates-must-select-the-2026-09-04.md` — why the gate compares whole failure sets

### Key context
- The interceptor removal changes the gRPC chain for every functional test; the live command above is the pin.
- Makefile targets still reference deleted dirs after this task; that is task .4. Run the live command directly rather than through `make`.
- `go test ./tests` compiles the whole package, which is why `tests/lost_task_test.go` must go in commit 1 and every monitor API in commit 3.

## Acceptance
- [ ] Three commits in the stated order, each passing `go build -tags 'test_dep integration' ./...`
- [ ] `git grep -n 'common/testing/umpire\|tools/umpire[123]\|GetMonitor\|AllowMonitorViolations\|RequireRulePassed\|umpireotel' -- service tests tools common cmd` returns nothing
- [ ] `common/testing/umpire`, `tests/testcore/monitor`, `tools/umpire1`, `tools/umpire2`, `tools/umpire3`, `cmd/umpire-genmodels`, the legacy tests, and `tests/lost_task_test.go` are deleted; CODEOWNERS line for `common/testing/umpire/verify/` removed; mise tasks and install script removed
- [ ] `cache.go` differs from its pre-task state only by removed imports and adapter calls; the six comment-only files differ only in comment lines (reviewed via `git diff`); `git grep -n -i 'umpire' -- service` returns nothing
- [ ] `go vet -tags test_dep ./service/history/... ./tests/... ./tools/umpire/...` passes; `CGO_ENABLED=0 go test -tags test_dep ./tests/testcore/... ./tools/umpire/vocabulary/...` passes
- [ ] `go test -v -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilot'` passes against a cluster
- [ ] Ledger records the lost-task property as a future Testpilot candidate

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
