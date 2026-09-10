---
satisfies: [R7]
---
# fn-83-author-a-live-case-from-a-model-file.7 Provisioning package and the umpire-run CLI

## Description
Extract namespace and Nexus endpoint provisioning from the live-test file into `common/testing/testpilot/temporal/provision` over gRPC clients only, then build `tools/umpire/cmd/umpire-run` on the Driver, `DeriveProfile`, and that package (R7). Independent of the Lean work; runs on the checked-in fixtures as they are, so it starts in the first wave and .4 builds on its live-test-file change.

**Size:** M
**Files:** `common/testing/testpilot/temporal/provision/provision.go` (new), `provision_test.go`, `common/testing/testpilot/temporal/README.md`, `tests/testpilot_live_case_test.go` (calls the package), `tools/umpire/cmd/umpire-run/main.go` (new), `run.go`, `run_test.go` (fake Driver, exit codes), `tests/testpilot_umpire_run_test.go` (new subprocess test), `common/testing/testpilot/README.md`, `Makefile` (build target and `.PHONY`)
**Touches:** [common/testing/testpilot/temporal/provision/**, common/testing/testpilot/temporal/README.md, common/testing/testpilot/README.md, tests/testpilot_live_case_test.go, tests/testpilot_umpire_run_test.go, tools/umpire/cmd/umpire-run/**, Makefile]

### Approach
- Provisioning: `Create(ctx, clients, Resources) (cleanup func(ctx) error, error)` using `WorkflowServiceClient.RegisterNamespace` followed by a bounded `DescribeNamespace` poll until the namespace cache serves it (the functional test base does the same), `OperatorServiceClient.CreateNexusEndpoint`/`DeleteNexusEndpoint`, and `OperatorServiceClient.DeleteNamespace` in cleanup; no `*testcore.TestEnv`, no `*testing.T`, no `common/namespace` import. The live-test file keeps its `t.Cleanup` wiring around the returned cleanup.
- CLI: follow the retired-vocabulary command's shape (`flag.String`, positional args rejected, errors to stderr). Flags per the spec's API Contracts. Build `Options{Profile, ServerEndpoints, SystemCallbackBaseURL, SDKClient, WorkerRoleID}` from the flags with the insecure transport credentials the live tests use; the composite Driver starts its own worker.
- Exit codes 0/1/2/3 and lifecycle per spec R7: `--timeout` default 5m; SIGINT cancels the context; teardown best-effort with one stderr line per leaked resource; `--create` collision is an error; without `--create` nothing is deleted.
- Output: Run status, cleanup status, Verdict status, one line per rule Verdict, all on stdout.
- Unit tests: a fake `testpilot.Driver` producing satisfied, violated, inconclusive, and a Run error, asserting exit codes and stderr lines; a typed fixture rejecting at prepare with its `PreparationError` category.
- Subprocess test under `tests/` (`//go:build test_dep && integration`, name prefixed `TestTestpilot` so the live gate selects it): `go build` the binary into a temp dir, run it with the async-Nexus fixture and `--create` against `env.FrontendGRPCAddress()`/`env.HttpAPIAddress()`, assert exit 0 and that the namespace is gone afterwards.
- Verify no server linkage: `go list -deps ./tools/umpire/cmd/umpire-run | grep -E 'tests/testcore|/service/'` is empty; put that in the unit test or the Makefile target.

### Investigation targets
**Required:**
- `tests/testpilot_live_case_test.go:26-119` — resources, namespace registration, endpoint create/delete, Profile freeze, Driver construction; the seam to break
- `tests/testcore/functional_test_base.go:550-564` — the namespace readiness poll to reproduce
- `common/testing/testpilot/temporal/driver.go:27-59` — `Options` and `New`
- `common/testing/testpilot/temporal/profile.go:26-56` — `DeriveProfile` and `Environment`
- `tools/umpire/cmd/umpire-check-retired-vocabulary/main.go` — CLI and exit-code conventions

**Optional:**
- `tests/testpilot_run_case_test.go:33-50` — `bindCase`, which the CLI mirrors without `testing.T`
- `common/testing/testpilot/preparation_error.go` — the categories printed on exit 3

### Key context
- The fn-70 deferral note in `UMPIRE4_ORDER.md` records that exporting `bindCase` pulled the whole server in; the pull-in is the provisioning's `TestEnv` coupling, not `DeriveProfile`.
- Memory: interface nil checks must cover every nil-capable kind; monitor closure must honor cancellation.
## Acceptance
- [ ] `go build ./tools/umpire/cmd/umpire-run` succeeds and `go list -deps` shows no `tests/testcore` or `service/` package
- [ ] The provisioning package has no `testing` or `testcore` import, polls namespace readiness, deletes the namespace on cleanup; the live tests use it and still pass
- [ ] Unit tests cover exit 0, 1, 2, 3, timeout, SIGINT, `--create` collision, and a typed fixture rejection
- [ ] The subprocess test passes under `make umpire-check-live-tests`; `make lint-code` is clean in touched files
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
