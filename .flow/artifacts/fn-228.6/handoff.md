# fn-228.6 handoff (lane A): Testpilot legacy packages retired; Runner coverage mapped

Stopped at the coordinator's wrap-up request. All the work is committed and nothing is staged.
Some gates were not run; see "Left open".

## Commits (branch umpire)

| Commit | Subject |
| --- | --- |
| 96d635bc17 | test(testpilot): retire the live Case tests |
| 7a2eb2458b | feat(canary): run the activity Model through the Runner and publish Runs |
| 166d864a6f | refactor(umpire): retire Case lowering, conformance and the IR bridge |
| e689481277 | refactor(testpilot): retire the Case runtime and cel-go |

## Acceptance

- [x] **Listed packages are gone, and cel-go is out of go.mod.**
  - Deleted: verification, evaluation, cel, predicate, contract, casefile, campaign, replay, publish and recordedrun.
  - Also deleted: the root facade, internal/*, and the Case Temporal Driver (`temporal`, `temporal/{server,worker,binding,internal}`).
  - The IR CEL bridge went in 166d864a6f: `tools/umpire/internal/runtimecel` and `cmd/umpire-ir-bridge`.
  - `go mod tidy` removed `cel.dev/cel-go` and `antlr4-go`, and moved genproto/api to indirect. No dependencies were added.
- [x] **ir and execution are deleted entirely, not shrunk (decision 1).** Act-and-observe already lives in `temporal/act` from fn-228.4, and instruction decoding had no remaining consumer.
- [x] **Case trees are gone.**
  - Deleted: `tests/testcore/testpilot` (generated Nexus Case tree and fixtures), `tools/canary/casebinding/testdata` and `tools/canary/assessment/testdata`.
  - The 11 live Case test files are retired in 96d635bc17.
- [x] **The seven legacy failures and the four pre-existing live failures are retired and mapped** in `.flow/tmp/fn-228.6/coverage-map.md`, section 1. Gaps that have no Runner equivalent are stated explicitly: the Nexus and Workflow Models and Drivers, IR-bridge minimization, workflow-scheduled activities, heartbeats, and attempt timeout.
- [x] **Umbrella, Testpilot and canary parts are green.** Every `common/testing/testpilot/...` and `tools/canary/...` package passes (`gate-umbrella.log`). The `tools/umpire` reader failures are pre-existing (see Gates).
- [x] **Per-package counts are reported; the targets are met** (below).
- [x] **Carried items:**
  - **Canary.** It is reduced to lease, fence and policy, and publishes with `runner.Publish`.
  - **`testpilot/publish`.** Retired, together with its importers.
  - **`UMPIRE_REPEAT_RUN_DIR` and `UMPIRE_CONTROL_RECORD`.** Both are gone, along with the pinned control record (`replay/testdata`).
  - **Stale prose.** It was removed with the deleted packages. The READMEs are updated: Testpilot, tools/umpire, the canary, model/README and the module map.
  - **`TestTestpilotModelSlice`.** Kept, not retired (decision 2).
  - **Folding `act` into its parent.** Not done (decision 3).

## Line counts (non-test Go, `.flow/tmp/fn-228.6/count.sh`)

Generated `v1` packages are excluded from both counts. They total 5,734 lines in `model/umpire/hosts/go`.

| Scope | Before | After |
| --- | --- | --- |
| Testpilot generic (`common/testing/testpilot`, non-`temporal`) | 21,506 | 36 (duration) |
| Testpilot Temporal (`temporal/*`) | 10,734 | 1,670 (act 1,353, provision 198, control 119) |
| Runner (`model/umpire/hosts/go/umpire/runner`) | 1,611 | 1,611 |
| **Generic runtime total (Testpilot generic + runner)** | 23,117 | **1,647**, against a target of at most 5,000 |
| **Total (Testpilot + runner)** | 33,851 | **3,317**, against a target of at most 11,000 |
| Including the Runner's adapters, the canary (3,010) and cmd/umpire (60) | — | 6,387, still at most 11,000 |
| tools/umpire, all packages | 29,726 | 17,088 (the Scala-era IR reader: check, export, engine, interp, ir, realization, plus cmd/umpire) |

Per-package detail is in `count-before.txt` and `count-after.txt`.

## Gates (logs in `.flow/tmp/fn-228.6/`)

**Build and vet** (`gate-build.log`): rc 0. This covers:
- `go build ./...`;
- `go vet -tags test_dep` on testpilot, tools/umpire, tools/canary and chasm/lib/activity;
- `go vet -tags test_dep,integration` on `./tests/`;
- the same with `canary_harness`.

Unrelated `tools/tdbg` vet findings exist repository-wide.

**Umbrella unit tests** (`gate-umbrella.log`): `go test -tags test_dep ./common/testing/testpilot/... ./tools/canary/... ./tools/umpire/...` ran with rc 1.
- **Passing:** every testpilot and canary package, `tools/umpire/cmd/umpire`, `internal/engine` and `realization`.
- **Failing:** `tools/umpire/{check,export,interp,ir}`. They fail on the retired `model/irgen` and `model/temporal/features` fixtures, and they also failed at the base (`baseline-unit.log`, `baseline-packages.txt`). These are pre-existing failures, not regressions.
- **Two test fixes in ir.** `TestEveryToolingPackageHasALiveCaller` now passes. `TestReaderAndLowererDoNotImportHandwrittenOracles` was adjusted for the deleted `lower` package.

**Live canary harness** (`canary-live.log`): `go test -tags 'test_dep integration' ./tests -run '^TestTestpilotCanary'` ran with rc 0. It covers:
- EndToEnd;
- RecoversALostProcess at all four phases;
- RefusesAProductionPolicy.

This ran on the canary commit's tree, before the last two deletion commits.

**Other focused checks:**
- `go test -race -count=3 ./tools/canary/...`: rc 0.
- `act -count=3`: rc 0.
- The remaining testpilot packages and chasm/lib/activity: rc 0.

**Not run (usage ran out):**
- `make umpire-check-live-tests`
- `make umpire-check-model-slice` alone
- golangci-lint on the changed packages
- default `make model-check`

The next session should run each alone. `model-check` touches no deleted Go: its go-host step tests `model/umpire/hosts/go` only.

## Decisions on the owner's behalf

1. **ir and execution are deleted, not shrunk.** The spec predates fn-228.4, whose Temporal Driver depends on neither package.
2. **`TestTestpilotModelSlice` is kept.** It is the only live coverage of CancelAfterCompletion, CancelBeforeWorkerStart and the Annotations controls, and it imports no retired package.
3. **`act` stays its own package.** Folding it into `temporal` would only churn import paths.
4. **Canary design.**
   - **Policy v2.** It pins `modelHash`, the SHA-256 of the committed guest, so a `make umpire-guest` rebuild needs a policy update. Preflight refuses a mismatch as `model-mismatch`. Three coordinates remain (grpc, namespace, taskQueue); handlerQueue and nexusEndpoint are dropped.
   - **Recovery record v2** names activity IDs.
   - **Iterations.** Each iteration fences a fresh `umpire-canary-<uuid>` standalone activity on the lease before its Run. The Driver is wrapped so it refuses any other start.
   - **World and seed.** The World is a Complete activity; the seed is 1.
   - **Cleanup and reconcile** describe and terminate activities.
   - **Statuses.** They are explained/unexplained/refused (exit 0/1/1) and failed (exit 3).
   - **Published Runs** keep their World: namespace and task queue, but never the credential or the gRPC target. Task tokens are not redacted, so Runs replay offline.
5. **The Temporal Driver gained `Options.Transport`** (dial options, client TLS and credentials) for the canary's TLS and API key. The zero value keeps the old plaintext behaviour.
6. **`tools/umpire/lint` is retired.** Its only caller, `umpire-lint`, had to go with `lower`.
7. **The Testpilot root is now test-only.**
   - It keeps the protocol schema tests that `make umpire-check-testpilot-protocol` runs.
   - It has a rewritten `TestTestpilotDependencyBoundary`: only `act` imports `umpire.local`, and Testpilot imports nothing from tests/, tools/umpire or tools/canary.
   - The Testpilot protos (`api/testpilot/v1`) are kept, out of scope.
8. **Two live harness tests were not rewritten.** LeavesALiveInvocationAlone and RecoversATimedOutLease remain covered by in-process unit tests (coverage map, section 2). RunsARepointedEndpointIncomplete has no Runner equivalent, since there is no Nexus Model or Driver.

## Left open

- **Gates not yet run:** live-tests umbrella, slice alone, golangci-lint, model-check.
- **AGENTS.md** still describes Testpilot, tools/canary and tools/umpire as Case tooling (lines 45, 57, 58). I did not edit the instruction file; the owner or conductor should update it.
- **Retired Make targets** (umpire-ir-bridge, -gen-cases, -check-cases, -check-lint, -gen/check-fixtures, canary-gen/check-case, exploration/replay bridge) are gone. The module map's historical command table (around lines 476–489) still lists them as history.
- **The Scala-era IR reader** (tools/umpire check, export, engine, interp, ir, realization; 17k lines) has no live IR. Its tests are red at the base. A follow-up spec should retire or re-home it.
- **The Testpilot protocol protos and `api/testpilot/v1`** have no runtime consumer left. Their only users are `cmd/tools/getproto` and the protocol schema test.
- **DESIGN.md "Runner and Run" counts (R7)** belong to fn-228.7. The numbers above are its input.
