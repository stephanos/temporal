---
satisfies: [R3, R5]
---
# fn-55-migrate-and-separate-the-local-temporal.2 Migrate live Run Evaluation proofs to TestEnv

## Description
Move the remaining live Run Evaluation normal and negative-control proofs from `tools/umpire/runevaluation` into `tests/`, composing the Nexus binding with the shared TestEnv-backed attached factory. Retain fast package-local evaluation and command tests with admitted fixtures/fakes only.

**Size:** M
**Files:** `tools/umpire/runevaluation/live_test.go`, `tools/umpire/runevaluation/live_negative_control_test.go`, `tools/umpire/runevaluation/integration_test.go`, `tools/umpire/runevaluation/command_test.go`, `tests/umpire4_testenv_test.go`, `tests/umpire4_run_evaluation_test.go`
**Touches:** [tools/umpire/runevaluation/live_test.go, tools/umpire/runevaluation/live_negative_control_test.go, tools/umpire/runevaluation/integration_test.go, tools/umpire/runevaluation/command_test.go, tests/umpire4_testenv_test.go, tests/umpire4_run_evaluation_test.go]

### Approach
- Move only cases that execute a real Temporal path; retain pure admission, subject, mutation, publication, CLI, and classification tests locally with fake operational output.
- Build each live adapter from the shared tests-only TestEnv authority helper and the explicit Nexus factory input from task `.1`; use one factory sequentially and a fresh TestEnv/factory for any parallel case.
- Preserve the normal satisfied result and duplicate-delivery uniqueness-only violation, exact operational/evidence closure, Run Evaluation publication, exit status, stdout/stderr discipline, and retry/determinism assertions.
- Keep crossed input, invalid subject, and malformed artifact paths ahead of TestEnv construction whenever the contract says execution must not occur.
- Recreate any private test helper at the tests boundary using exported behavior; do not export production declarations solely for tests.
- Delete or reduce the old live files so no test under `tools/umpire/runevaluation` starts a server.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/runevaluation/live_test.go:54-120,184-225` — normal live run and command publication proof
- `tools/umpire/runevaluation/live_negative_control_test.go:153-305` — duplicate-delivery table, evaluation, and CLI proof
- `tools/umpire/runevaluation/integration_test.go:180-240,300-345` — fake/admitted integration coverage that should remain local
- `tools/umpire/runevaluation/command_test.go:20-165` — fast CLI error and output contract
- `tests/umpire4_portable_executor_test.go:32-130` — existing sequential normal/faulted TestEnv proof
- `tools/umpire/runner/runner.go:93-161` — no-execution versus execution-occurred classification

### Key context
- TestEnv startup errors belong to `testing.T`; deterministic Umpire lifecycle error receipts stay covered by fake authorities in the local package.
- Preserve the existing distinction between stable semantic meaning and transport-scoped run/evidence/publication identities.

## Acceptance
- [ ] Every real Run Evaluation execution case lives under `tests/` and uses the shared TestEnv-backed `NewAttachedFactory`; no Run Evaluation package test starts a live server.
- [ ] Normal evaluation remains satisfied and duplicate delivery remains a uniqueness-only violation with exact clause statuses, Evidence closure, cleanup status, and publication contents.
- [ ] CLI success/failure, exit code, stdout/stderr, destination, retry, and deterministic stable-meaning behavior remain exact.
- [ ] Crossed or malformed input fails before TestEnv/factory access wherever execution is prohibited.
- [ ] Package-local tests retain fake-backed operational, error-precedence, mutation, publication, and command coverage without new exported test hooks.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/runevaluation/...` passes without starting a live server.
- [ ] `go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpire.*RunEvaluation'` passes.


## Done summary
Moved the live Run Evaluation normal and duplicate-delivery proofs under `tests/` on the shared TestEnv-backed attached factory, while retaining fake-backed package-local operational and command coverage. Focused unit, race, exact tagged integration, aggregate regression, formatting, and scoped lint gates pass; global lint matches the inherited 1,378-finding baseline, and the parent-wide `^TestUmpire` gate remains red only in unchanged Umpire2/Umpire3 tests that execute before this task's new tests.

stage: impl-review - ran [2026-09-04T03:11:12Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9521431bad43606e14d4d4669d7d6e54d4c82247, a9d06446f7e38bebb070713fc72f24037f6e90bb
- Tests: go test -count=1 -tags test_dep ./tools/umpire/temporal/local/... ./tools/umpire/runner/... ./tools/umpire/temporal/nexus/... ./tools/umpire/runevaluation/... (pass, post-review), go test -race -count=1 -tags test_dep ./tools/umpire/temporal/local/... (pass, post-review), TMPDIR=<physical> CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/runevaluation/... (pass), TMPDIR=<physical> CC=<Apple clang> SDKROOT=<macOS SDK> go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpire.*RunEvaluation' (pass), TMPDIR=<physical> CC=<Apple clang> SDKROOT=<macOS SDK> go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpire' (inherited red: unchanged earlier-ordered Umpire2/Umpire3 failures; exact task subset passes), TMPDIR=<physical> CC=<Apple clang> SDKROOT=<macOS SDK> make umpire-check-regression (pass: 270 jobs), CC=<Apple clang> SDKROOT=<macOS SDK> make fmt-imports (pass), CC=<Apple clang> SDKROOT=<macOS SDK> make lint-code (inherited red: exact 1378-issue baseline), CC=<Apple clang> SDKROOT=<macOS SDK> .bin/golangci-lint-v2.13.1 run --build-tags 'disable_grpc_modules,test_dep,integration' --timeout 10m --fix=false --new-from-rev=718fb9d365f0f873ff468189af2e8d98b047f83c --config=.github/.golangci.yml ./tools/umpire/runevaluation/... ./tests/... (pass: 0 issues), git diff --check (pass)
- PRs: