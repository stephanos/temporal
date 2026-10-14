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
Two seams, both of which the canary specs assume and neither of which existed.

- `common/testing/testpilot/temporal/provision` (new): `Create(ctx, Clients, Resources) (Cleanup,
  error)` over `WorkflowServiceClient.RegisterNamespace` + a bounded `DescribeNamespace` readiness
  poll, `OperatorServiceClient.CreateNexusEndpoint`/`DeleteNexusEndpoint`, and
  `OperatorServiceClient.DeleteNamespace`. No `testing`, no `testcore`, no `common/namespace`.
  It rolls back what it created if a later step fails, and its cleanup releases in reverse order
  reporting every resource it could not remove.
- `tests/testpilot_live_case_test.go` binds through it; `newTestpilotLiveCase` lost its
  `MetadataManager`-backed namespace registration and its hand-written endpoint cleanup.
- `tools/umpire/cmd/umpire-run` (new): `--case --grpc --http --namespace --task-queue
  [--nexus-endpoint] [--create] [--timeout]`, exit 0/1/2/3, SIGINT and timeout cancel the Run while
  teardown runs on its own context with a per-resource budget.
- `Makefile`: `umpire-run` build target (+ `.PHONY`); `common/testing/testpilot/README.md` and
  `common/testing/testpilot/temporal/README.md` describe both.

Deliberate design addition the spec did not name: `Resources.RetainNamespace`. Namespace deletion is
a system-worker workflow; the functional cluster this repo runs live tests on does not start the
worker service (the same reason `tests/namespace_test.go` asks for it explicitly), so waiting for it
in every live test would cost tens of seconds per namespace and fail. The live tests set it; the CLI
does not, so `--create` still deletes what it made against a real deployment.

Spec acceptance on `go list -deps`: `tests/testcore` is absent from the transitive closure entirely.
`service/` is NOT absent -- three packages (`service/history/consts`, `service/history/tasks`,
`service/matching/counter`) are reached transitively through `common/dynamicconfig`,
`common/persistence` and `chasm`, which the Driver's own Nexus support already pulled in before this
task. The test pins that exact set, so a fourth is a failing test; the CLI itself imports neither.

Tests: exit 0/1/2/3, a real `testpilot.Driver` fake failing through the real prepared-Case path,
a typed fixture rejecting with its `PreparationError` category, missing flags, positional arguments,
a non-positive timeout, an unreadable fixture, the binding timeout, a real SIGINT delivered to the
test process, a `--create` collision, per-resource leak lines, and the transitive-closure gate.
Provisioning: the readiness poll, endpoint skip, `RetainNamespace`, an existing namespace, rollback
after an endpoint failure, a namespace the cache never serves, an unrelated describe failure,
cleanup reporting both leaks, and three invalid requests.

Live: `tests/testpilot_umpire_run_test.go` builds the binary and runs it as a subprocess with
`--create` against the functional cluster -- exit 0, `run Completed`, `verdict Satisfied`, and the
Nexus endpoint gone afterwards -- plus an unreachable-address case pinning exit 3.
`make umpire-check-live-tests` is green across 8 passing identities (up from 6).

`make lint-code GOLANGCI_LINT_FIX=false` reports 161 findings, none in a file this task touched.
That is the tree's actual pre-existing count: the 128 recorded in this run's brief came from a pass
that aborted early ("Issues before processing: 11800, after processing: 1" with a `no space left on
device` typecheck failure), which is also what happened on my first attempt until I reclaimed the
Go build cache.

Review: SHIP after one round. The P2 (the import test read only declared imports, not the
transitive closure) was valid and fixed by walking `go list -deps`.
Pinned reviewer `claude:claude-fable-5-1:high` is account-limited for this session, so both rounds
ran on `claude:claude-sonnet-4-5:high` -- a same-family fallback, not an equivalent cross-family
review.

stage: impl-review - ran, 2 rounds (model: claude-sonnet-4-5, high; fable pinned but account-limited)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e752240daa, HEAD
- Tests: CC=/usr/bin/cc go test -count=1 ./tools/umpire/cmd/umpire-run ./common/testing/testpilot/temporal/provision, make umpire-check-live-tests (8 passing identities), make lint-code GOLANGCI_LINT_FIX=false (161 findings, 0 in touched files), go build ./tools/umpire/cmd/umpire-run
- PRs: