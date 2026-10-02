---
satisfies: [R2]
---
# fn-112-gomad-determinism-assurance-and-test.2 Run the orphaned simulation, overlay, and choice-replay tests in a gate

## Description
Make tests that exist but no target runs execute in `make -C tools/gomad3 test` and CI (R2). Depends on task 1 so newly failing tests are attributable.

**Size:** M
**Files:** `tools/gomad3/Makefile`, `tools/gomad3/architecture_test.go`, `tools/gomad3/internal/gomadtool/conformance/runtime_campaign.go` and the `runtime_*.go` file that owns replay behavior, `.github/workflows/gomad3.yml`
**Touches:** [tools/gomad3/Makefile, tools/gomad3/architecture_test.go, tools/gomad3/internal/gomadtool/conformance/runtime_campaign.go, tools/gomad3/internal/gomadtool/conformance/runtime_choice.go, .github/workflows/gomad3.yml, tools/gomad3sim/**]

### Approach
- Add a Make target that runs `tools/gomad3sim` with the `gomad3_toolchain` tag under the patched toolchain, and add it to `test`. Decide whether the `integration`-tagged `simulation_root_integration_test.go` joins it or is superseded by it.
- Extend `overlay-test` with `internal/gomadsim`, `internal/gomadmodelwire`, `internal/gomadio`, `os`, and `cmd/internal/gomadcap`.
- Execute the `choice-replay` fixture that `runtime_campaign.go` builds, in the conformance file that owns replay behavior.
- Run `./toolchain` in `test-toolchain` (patched) and `test-builder` (stock) only; drop it from `test-host`.
- `TestMakeTargetsMatchTheirOwnership` constrains new Make targets; update its table.
- A test that fails once it runs is fixed or recorded as a finding; deletion needs a stated reason.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/Makefile:127-133` — `test-host` and `overlay-test` package lists
- `tools/gomad3/internal/gomadtool/conformance/runtime_campaign.go:299` — `choice-replay` build with no executor
- `tools/gomad3/architecture_test.go:138` — Make target ownership check
- `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go:1-19` — the only current executor of the simulation tests

**Optional** (reference as needed):
- `Makefile:189` — root `gomad3-integration-test`
- `tools/gomad3/architecture_test.go:290` — conformance grouping rule

### Key context
- `tools/gomad3sim` is on the list of features whose fate is undecided (spec Open Questions 4). Running its tests does not settle that.
## Acceptance
- [ ] The six `tools/gomad3sim/*_toolchain_test.go` files run in a Make target included in `test` and in CI
- [ ] `overlay-test` covers the five previously omitted overlay packages
- [ ] The `choice-replay` fixture is executed and asserted
- [ ] `./toolchain` runs once per toolchain kind
- [ ] Every newly running test passes, or is recorded as a finding with its failure
- [ ] The new gate's name and scope are in the done summary for task 10 to document
- [ ] `make -C tools/gomad3 test` passes on darwin/arm64; linux status recorded
## Done summary
Task 2 gate repair, 2026-10-02, uncommitted working tree over HEAD 1d7272e65.

The test-simulation gate runs all six tagged gomad3sim test files under the seeded patched runtime. Its Runner integration invocation now executes ten of eleven transport cases: the three scenario/exploration cases and all seven process-node cases other than TestProcessBackendSynchronizesNodeClockWithModelDelay. The six unsupported exclusions identified in review-before-repair.json are restored. The remaining clock-delay exclusion is tied to finding-process-backend-watchdog.log; twenty repeats passed after the stop fix, but that does not establish a root cause or close the prior watchdog finding.

The graceful-stop failure was traced to canceling node model/time services before cleanup finished. Node services now keep a separate context until supervisor completion, then cancel explicitly. The deterministic TestRunKeepsNodeModelTransportAliveDuringGracefulStop sends a model request only after the stop control byte: regression-graceful-stop-before.log fails with SIGTERM, regression-graceful-stop-after.log passes ten repeats. The test also requires the transport context to be canceled after Run returns. Existing watchdog, hard-crash, stale-incarnation, and process-group behavior is preserved. process-backend-repeated-after.log records twenty passing repetitions of both previously failing cases; the stop case is restored to the gate.

Overlay-test now covers all five formerly omitted packages, including internal/gomadsim. process_time_test.go uses an external test package and a testing-free export_test.go alias. The model-response test starts a fresh subprocess with inherited request/response descriptors because runtime control environment is captured at startup; correlation and host-error assertions remain. Its host-error fixture now clears the request payload when constructing the error response. The version descriptor allowlist includes the new test export. overlay-simulation-repaired.log records successful overlay and simulation gates on darwin/arm64 key 6b775117cc6b13d04c2d00926818e102edb540f8c74ece2794b5e6f3cd2c19ee; that invocation had loaded the earlier selector before the stop case was restored. execution-tests-repaired.log records the complete execution package pass; focused-vet.log records a clean focused vet run. Full make test with the final selector passed with exit code 0; make-test-repaired.log retains every completed tier, including runtime and upstream compatibility. All seven source hashes and the active toolchain build key were checked against repair-sources.json after exit and match.

The existing choice-replay executor and stock/patched toolchain ownership changes remain as delivered in a275a3713. Linux CI wiring includes test-simulation and overlay-test, but current uncommitted changes have no Linux execution evidence. The all-green baseline run 36968858553 tested 8789deab0 before these gate additions. Root make lint-code-fast GOLANGCI_LINT_BASE_REV=8789deab0 exits 2 because the root module does not contain nested gomad3/internal/gomadtool/conformance, choice/internal/wire, and target/internal/livecap packages; root-lint-repaired.log retains the typecheck errors. Focused vet and formatting checks pass; root lint is not clean.

Task 10 handover: document test-simulation as all six directly seeded toolchain test files plus ten Runner transport cases, and overlay-test as including internal/gomadsim, internal/gomadmodelwire, internal/gomadio, os, and cmd/internal/gomadcap. Keep the specific clock-delay watchdog finding open and do not claim Linux qualification. No files were staged and no new commits were created.

The uncommitted repair is retained in repair.patch with SHA-256 19c72cb19928eddb108e25f0f22e8d29de787d245de1828c6770f96fbc1378ba. repair-sources.json binds its seven source files, HEAD 1d7272e654f268f9a45f3fe965918fe2522827c6, and toolchain key 6b775117cc6b13d04c2d00926818e102edb540f8c74ece2794b5e6f3cd2c19ee. review-repaired.json records SHIP against the reviewed working-tree artifact.

Impl-review: ran (codex:gpt-6-astra:high, SHIP).
Plan-sync: skipped (planSync.enabled=false).
Tracker sync: n/a (bridge inactive).
## Evidence
- Commits:
- Tests: make -C tools/gomad3 test (exit 0; darwin/arm64; make-test-repaired.log; toolchain key 6b775117cc6b13d04c2d00926818e102edb540f8c74ece2794b5e6f3cd2c19ee), make -C tools/gomad3 overlay-test test-simulation (exit 0; overlay-simulation-repaired.log), tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep ./runner/internal/execution (exit 0; execution-tests-repaired.log; cwd tools/gomad3), tools/gomad3/.toolchain/bin/go test -count=10 -tags test_dep -run ^TestRunKeepsNodeModelTransportAliveDuringGracefulStop$ -v ./runner/internal/execution (exit 0 after fix; regression-graceful-stop-after.log; cwd tools/gomad3), tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep -run ^TestRunKeepsNodeModelTransportAliveDuringGracefulStop$ -v ./runner/internal/execution (exit 1 before fix with SIGTERM, expected regression; regression-graceful-stop-before.log; cwd tools/gomad3), tools/gomad3/.toolchain/bin/go test -count=20 -tags test_dep,integration -run ^TestRootProcessSimulationUsesRunnerTransport$/^TestProcessBackend(SynchronizesNodeClockWithModelDelay|ResetsGlobalsDescriptorsAndGoroutines)$ -v ./runner/internal/execution (exit 0; process-backend-repeated-after.log; cwd tools/gomad3), tools/gomad3/.toolchain/bin/go vet -tags test_dep . ./runner/internal/execution ./toolchain/version internal/gomadsim (exit 0; focused-vet.log; cwd tools/gomad3), gofmt -l changed Go files and git diff --check (exit 0; no output), make lint-code-fast GOLANGCI_LINT_BASE_REV=8789deab0 (exit 2; nested-module package discovery; root-lint-repaired.log; executed by parent), Seven source SHA-256 bindings and active toolchain build key match repair-sources.json after full gate exit
- PRs: