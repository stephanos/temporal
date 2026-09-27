---
satisfies: [R2]
---
# fn-94-simplify-the-testpilot-go-runtime.5 Remove dead Driver code, the untyped Nexus residue and the live runCase trio

## Description
Lane A for the Temporal Drivers and live tests: dead and test-only Driver declarations, the retired untyped Nexus path, dead test helpers and the live `runCase` trio. Disjoint from fn-94.4.

**Size:** M
**Files:** `common/testing/testpilot/temporal/worker/{driver,routing,carrier,registry,interpreter,typed,sdk,callback,outage,api}.go` and their tests, `temporal/internal/delivery/carrier.go` (Nexus parameters only), `temporal/internal/delivery/ledger.go` (`Activation.Handle` only), `temporal/driver.go` (`newCompositeSession`), `temporal/internal/activation/activation_test.go`, `tests/testpilot_run_case_test.go`, `tests/testpilot_workflow_start_case_test.go`, `temporal/worker/README.md`, `tests/testcore/testpilot/README.md`
**Touches:** [common/testing/testpilot/temporal/worker/**, common/testing/testpilot/temporal/internal/delivery/carrier.go, common/testing/testpilot/temporal/internal/delivery/ledger.go, common/testing/testpilot/temporal/internal/delivery/*_test.go, common/testing/testpilot/temporal/driver.go, common/testing/testpilot/temporal/driver_test.go, common/testing/testpilot/temporal/internal/activation/activation_test.go, tests/testpilot_run_case_test.go, tests/testpilot_workflow_start_case_test.go, tests/testcore/testpilot/README.md]

### Approach
- Unreachable: `Driver.prepareDefinitionPlans` (`worker/driver.go:254`), worker `validCoordinate` (`routing.go:481`), `Carrier.Handles`/`Quarantine` (`carrier.go:76,141`).
- Test-only, tests re-pointed at production: `Carrier.ParentTerminal` → `Session.parentTerminal`; `Session.preparedNexusDispatch`; `workerLease.release` → `releaseLocked`; `Outage.Stopped`; `delivery.Activation.Handle`; `Options.SessionOptions`/`Driver.Open` (`api.go:33`, `driver.go:180-188`); `newCompositeSession` (`temporal/driver.go:158`).
- Untyped Nexus residue: first add a test pinning the typed sync reply with no payload (`typed.go:117`); then remove `nexusResult.value` (`interpreter.go:36,194,337`), the `typed` map and `future.Get(&value)` fallback (`interpreter.go:57-59,125-130`, `typed.go:86`), the `*testpilotspb.Value` input case (`sdk.go:103-104`), `Value` payload encoding (`callback.go:118-124`), `PrepareNexus`'s `header`/`value` parameters and `NexusDispatch.value`/`Value()` (`delivery/carrier.go:38,47,189`), and two of the three reserved-header merges (keep the one the production path needs).
- Dead test helpers: `runtimeText`/`runtimeStatusType`/`runtimeTextType` in `activation_test.go:200-226`; `runtimeStatusType`/`runtimeTextType` in `worker/runtime_fixture_test.go:248-256` (keep its used `runtimeText`); `openCountingDriver.Open`; `workflowAdmissionForTest`.
- Live: delete `runCase`, `runBoundCase`, `runCaseWithBinding` (`tests/testpilot_run_case_test.go:94-110`) and move the one caller (`tests/testpilot_workflow_start_case_test.go:29`) to `runCapturedCaseWithBinding` without renaming its test.
- Docs: `temporal/worker/README.md:53-55` (carrier delegation), `tests/testcore/testpilot/README.md:30-36` (the single-shot wrapper paragraph).

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/worker/interpreter.go:30-140,180-200,330-340`
- `common/testing/testpilot/temporal/worker/typed.go:80-120`
- `common/testing/testpilot/temporal/internal/delivery/carrier.go:30-50,185-240`
- `common/testing/testpilot/temporal/worker/routing.go:190-260,460-490`
- `tests/testpilot_run_case_test.go`
**Optional:**
- `tests/testpilot_signature_test.go:120-130` — `runCapturedCaseWithBinding`

### Quick commands
```sh
go test -race -tags test_dep ./common/testing/testpilot/temporal/...
go vet -tags test_dep ./tests/...
make umpire-check-live-tests
make lint-code-fast
```

## Acceptance
- [ ] Every Driver-side declaration and test helper lane A lists is gone; tests use production paths.
- [ ] A test pins the empty typed sync reply before the untyped fallback goes, and passes after.
- [ ] No live test renamed or removed; the moved caller compiles and `make umpire-check-live-tests` passes (or is reported not run with the reason).
- [ ] fn-94.2 goldens unchanged; `-race` tests and `make lint-code-fast` pass.


## Done summary
Removed lane A's Driver-side dead and test-only code, the untyped Nexus residue and the live runCase trio; tests now use the production paths (Session.parentTerminal, preparedNexusHeader, Outage.Restore, stoppedLocked, the bundle's handle, newPreparedCompositeSession over a captured conformance Program). The empty typed sync reply was pinned first (744223b49a, "synchronous without payload" in TestSessionAnswersTypedReplies: the caller receives the SDK's binary/null payload) and still passes after the fallback went; a mutation to an empty RawValue fails it. The three reserved-header merges are now the one in preparedNexusHeader (case header, SDK header and route, with any collision refused). PrepareNexus takes no header or value.

Tests: go test -race ./common/testing/testpilot/temporal/... green. The fn-94.2 goldens are untouched and green. Live Go step of umpire-check-live-tests at 5bc236723d, with other workers' dirty files pinned to HEAD by a go -overlay: 45 passing, 0 failing, equal to fn-94.2's count. TestTestpilotWorkflowStartCase keeps its name and exercised both namespaces. An unpinned run failed 12 identities: the canary harness build hit another worker's mid-edit execution/prepare.go, and NexusCallerScheduleToStartTimeout/hsm failed once. The pinned rerun was clean. deadcode -test over Testpilot, tests and tools reports no Testpilot function. lint-code-fast: my packages are clean. Its one issue is gci in common/testing/testpilot/prepare.go, from fn-94.4's commit 1d94ab30ff.

baseline: green (focused Quick commands, pre-edit)
Follow-up: tests/testpilot_signature_test.go:116,123 comments still say "runCase" and "runCaseWithBinding". That file is outside this task's Touches, so fn-94.15 or fn-94.17 should reword them. The commit range 735caa401c..HEAD also holds fn-94.3/fn-94.4 commits; only 744223b49a and 5bc236723d belong to this task.

stage: impl-review - ran [codex fan-out rid 312bca414082493f80f9f232333a7f55: correctness/contracts/integration all SHIP..SHIP]
## Evidence
- Commits: 744223b49af4ec3f1986e94725264af5b609657f, 5bc236723d3a27a7c8e456ad758b964f54618dfd
- Tests: go test -race -tags test_dep ./common/testing/testpilot/temporal/..., go vet -tags test_dep ./tests/..., go test -v -count=1 -timeout 30m -tags 'test_dep integration' ./tests -run '^TestTestpilot' (Go step of make umpire-check-live-tests, physical TMPDIR; 45 passing, 0 failing), deadcode -test -tags 'test_dep integration' ./common/testing/testpilot/... ./tests/... ./tools/... (no Testpilot entries), make lint-code-fast (only issue: gci in common/testing/testpilot/prepare.go from fn-94.4, outside this task)
- PRs: