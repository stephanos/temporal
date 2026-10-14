---
satisfies: [R4]
---
# fn-94-simplify-the-testpilot-go-runtime.9 Validate once in the worker Driver and delivery carrier

## Description
Lane C for the worker Driver and `delivery`: the worker's construction-time ceiling check calls `ir`'s exported check, `validateSymbolicRoles` shrinks to what is not re-validation, and the carrier's plan-shape re-checks go while runtime checks stay.

**Size:** M
**Files:** `common/testing/testpilot/temporal/worker/driver.go`, `temporal/worker/carrier.go`, `temporal/internal/delivery/ledger.go`, their tests, `temporal/README.md`
**Touches:** [common/testing/testpilot/temporal/worker/driver.go, common/testing/testpilot/temporal/worker/carrier.go, common/testing/testpilot/temporal/worker/*_test.go, common/testing/testpilot/temporal/internal/delivery/ledger.go, common/testing/testpilot/temporal/internal/delivery/*_test.go, common/testing/testpilot/temporal/README.md]

### Approach
- `validWorkerProfile` (`worker/driver.go:80-108`): keep the construction-time check, but call `ir.CheckCeilings` (from fn-94.7) instead of its reflection loop. The Drivers may import `internal/ir` only if Go's internal-package rule allows it from `temporal/`; otherwise route through a facade/contract export.
- `validateSymbolicRoles` (`:388-418`): keep only the worker-role-ID check and `profileRoleHasMethods`.
- `delivery` (`ledger.go:285-360`): remove the plan-shape re-checks in `validateTopology`/`validateRoutes`; for each, cite the `execution/carrier.go` admission check that already rejects that shape (in the commit message or a test name). Keep building the expected-handle map `validateHandles` consumes, keep `validateHandles`, and keep the runtime `MaxRoutes` limit, each pinned by a rejection test.
- Worker carrier `validCarrierBinding` (`carrier.go:58-73`): drop the count and duplicate checks; keep the workflow-entrypoint lookup.
- Update `temporal/README.md:23-25`.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/internal/delivery/ledger.go:280-370`
- `common/testing/testpilot/internal/execution/carrier.go` — the admission checks that replace them
- `common/testing/testpilot/temporal/worker/driver.go:75-110,380-425`
- `common/testing/testpilot/temporal/worker/carrier.go:40-75`

### Quick commands
```sh
go test -race -tags test_dep ./common/testing/testpilot/temporal/...
make umpire-check-case-runtime-conformance
make lint-code-fast
```

## Acceptance
- [ ] No plan-shape re-check remains in `delivery` or the worker carrier; each removed check has a named admission counterpart.
- [ ] `validateHandles`, the expected-handle map and `MaxRoutes` remain, each pinned by a rejection test.
- [ ] The worker ceiling check calls the shared export; `-race` tests and lint pass; goldens unchanged.


## Done summary
The worker Driver's construction-time ceiling check now calls `ir.CheckCeilings`. `validateSymbolicRoles` keeps only the worker-role check and `profileRoleHasMethods`. Delivery's `validateTopology`/`validateRoutes` plan-shape re-checks, the ledger's `EndpointRoleID` check, and the worker carrier's count and duplicate checks are gone. The commit message names the admission counterpart of each removed check, and `temporal/README.md` now describes what the Carrier still checks.

The expected-handle map, `validateHandles`, the runtime `MaxRoutes` limit and the workflow-entrypoint lookup remain. `TestCreateBundleUsesExactIdentityAndRetainsRejectedHandles` (now asserting specific errors) and the new `TestCreateBundleRejectsPlanBeyondMaxRoutes` pin them; each `MaxRoutes` case was mutation-checked. The new `TestWorkerProfileRejectsLimitsOutsideTheCeiling` pins the shared ceiling check. The single-workflow guarantee's counterpart is the worker Driver's `reservedWorkflowQueueRole` at Validate/Open rather than execution admission. Baseline: green (race tests). No goldens moved.

stage: impl-review - ran [fanout rid 37962e326fb6449cbfb4c7a0e375f3ce, 3 draws SHIP]
## Evidence
- Commits: 32991e77376031a9333f737d291141f6dbf13f80
- Tests: go test -race -tags test_dep ./common/testing/testpilot/temporal/... (run with -overlay pinning concurrent fn-94.10 server edits to HEAD), make umpire-check-case-runtime-conformance, make lint-code-fast, go test -tags test_dep ./common/testing/testpilot -run '^TestCaseRuntimePublicFacadeConformance$|Golden'
- PRs: