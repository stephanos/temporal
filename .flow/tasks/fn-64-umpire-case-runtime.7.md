---
satisfies: [R8, R9]
---
# fn-64-umpire-case-runtime.7 Compile and execute the async Nexus Case

## Description
Prove the Producer/runtime seam with two unrelated Cases (R8, R9): compile and prepare a small
`GetSystemInfo` Case without target I/O, then compose the server/worker Host, run async Nexus success
through one prepared Case, and verify bounded server history.

**Size:** M
**Files:** generic Case compiler modules under `model/Umpire/`, Temporal Nexus Case lowering/fixtures
under `model/Temporal/`, composite `tools/umpire/temporal` wiring/documentation, new Umpire
async-Nexus integration tests and model documentation
**Touches:** [model/Umpire/**, model/Temporal/**, tools/umpire/temporal/*.go,
tools/umpire/temporal/README.md, tests/umpire4_async_nexus_case_test.go,
tests/umpire4_testenv_test.go, model/README.md, model/ARCHITECTURE.md,
model/Umpire/ARCHITECTURE.md]

## Approach
- Compose the task18 carrier/ledger with task6 SDK admission and task5 server transport. Configure
  the declarative StartWorkflowExecution carrier policy, inject only reserved delivery headers,
  and retain the standard full Umpire Value payload convention.
- Lower the selected checked Lean model into the public Case schema, including
  controller/workflow/handler entrypoints and deterministic Contract machines.
- First compile an orthogonal `GetSystemInfo` Case with an empty request, a typed `server_version`
  projection, a different Contract topology, and the exact authorized WorkflowService descriptor;
  prepare it through the public Go API without calling `Run`.
- Fail unsupported checked constructs explicitly; never emit a weakened Contract or reconstruct
  identity from runtime limits.
- Compose server and worker packages at the root Temporal Host without weakening their import or
  authority boundary.
- Start through `InvokeRPC`, run generic SDK/Nexus entrypoints, await opaque completion readiness,
  complete through the Host capability, and retrieve bounded history with execution-owned
  `EmitEach` projection.
- Make Contract observations only from declared authoritative history fields and correlations.
  Lower typed rule captures to retain the scheduled event ID and compare later scheduled-event
  references against it; never hardcode runtime IDs or correlate by event kind alone.

## Investigation targets
**Required** (read before coding):
- `model/Temporal/Tool/PortableEvaluationContract.lean:993-1042` — scenario-specific lowering to
  replace
- `model/Temporal/Tool/PortableEvaluationContract.lean:1167-1181` — legacy plans
- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean` — retired slice, not a template
- `tests/umpire4_testenv_test.go:1-69` — current environment and fixture setup
- `tests/umpire4_portable_executor_test.go:27-120` — current Nexus setup/cleanup behavior

**Optional** (reference as needed):
- `.flow/memory/bug/integration/portable-model-plans-need-exact-2026-09-03.md` — exact compiler
  obligation lesson
- `.plans/LEAN_GUIDELINES.md` — mandatory Lean authoring rules

## Key context
Lean owns scenario/property semantics. Go may interpret only the general IR; it must not add an
async-Nexus lifecycle checker.

## Acceptance
- [ ] The real start/header/Nexus-delivery path matches exact reservation identities across repeated
  and concurrent Runs; reversed arrivals, failed starts and cancellation never consume another Run's
  route. The transmitted request obeys limits after Host metadata injection.
- [ ] Lean emits a reproducible Case whose identities, typed values, paths, limits, and monitor
  obligations survive Go preparation exactly.
- [ ] The `GetSystemInfo` Case compiles and prepares with zero Host I/O, and its different method,
  projection, and Contract shape require no new instruction or special execution/verification/
  Temporal branch.
- [ ] Unsupported lowering fails deterministically; crossed-kind or descriptor mutations fail
  preparation before I/O.
- [ ] The composite Host preserves server/worker package boundaries and uses opaque completion
  authority without exposing its private Slot contents. Authorized RPC response fields retain
  ordinary projection semantics.
- [ ] Integration performs start workflow → SDK Nexus start/await → async response → controller
  completion → bounded history retrieval → workflow finish.
- [ ] Matching history IDs satisfy the intended correlation; a completed event pointing to a
  different scheduled event cannot satisfy it. Captures/support references and deadline boundary
  results agree live and offline without scenario-specific Go logic.
- [ ] Verdict evidence is only declared server-history Observations, and one PreparedCase drives
  isolated sequential/concurrent Runs including non-success and timeout cases.
- [ ] `make umpire-build-model` and `go test -count=1 -tags 'test_dep integration' ./tests -run
  TestUmpireAsyncNexusCase` pass.

## Done summary
Implemented generic Lean Case compilation/ProtoJSON plus reproducible GetSystemInfo and async Nexus artifacts, and composed the Temporal server/worker Host through the generic Umpire runtime. The async Case now authors per-Run workflow/request identity through a closed Program-input intrinsic, preserves scheduled/completed field pairing with an exact-descriptor bounded HistoryEvent capture, and keeps verdict support limited to declared history Observations.

Fixed all four first-round review findings: unique request identity, aligned scheduled-event correlation, fresh bounded retryable carrier finalization including Drain, and post-expansion callback metadata sizing. The additive schema scope includes authoritative protobuf/generated Go, the exact Lean mirror/generated API, restricted admission/runtime resolution, and descriptor/byte/ownership tests.

Validation passed: protobuf and Lean API generation, full Lean model build, byte-identical artifact regeneration, focused normal and race Go suites, sequential/concurrent exact-artifact reuse plus non-success/timeout/cross-correlation cases, the real SDK integration, import formatting, scoped no-fix golangci-lint, and scoped errortype. The repo-wide regression vocabulary scan remains inherited red at the pre-existing literal validation message in tools/umpire/internal/ir/catalog.go:214; the first broad lint retry also surfaced inherited baseline findings, while the task-owned patch lint reported zero issues.

Official codex:gpt-5.6-sol:high review round 2 returned SHIP with every prior finding fixed and no new blocking findings. Receipt: /tmp/impl-review-receipt-fn-64-umpire-case-runtime.7.json.

stage: impl-review - ran [round 1 NEEDS_WORK..round 2 SHIP]
## Evidence
- Commits:
- Tests: make proto, make umpire-gen-lean-api, make umpire-build-model, cd model && lake env lean --run Temporal/Tool/CaseRuntime.lean async-nexus && lake env lean --run Temporal/Tool/CaseRuntime.lean get-system-info (byte-identical fixture comparison), CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/internal/ir ./tools/umpire/internal/execution ./tools/umpire/verification ./tools/umpire/temporal/..., TMPDIR=$PWD/.flow/tmp/fn64-task7-race-tmp CGO_ENABLED=1 CC=/usr/bin/clang go test -race -count=1 -tags test_dep ./tools/umpire/internal/execution ./tools/umpire/verification ./tools/umpire/temporal ./tools/umpire/temporal/server, CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/temporal -run '^TestLean(AsyncNexusPreparedCaseReuseAndCorrelation|CasesDecodeAndGetSystemInfoPreparesWithoutHostIO)$', CGO_ENABLED=0 go test -count=1 -tags 'test_dep integration' ./tests -run TestUmpireAsyncNexusCase, make fmt-imports, TMPDIR=$PWD/.flow/tmp/fn64-task7-lint-tmp CGO_ENABLED=0 .bin/golangci-lint-v2.13.1 run --verbose --build-tags 'disable_grpc_modules,test_dep' --timeout 10m --fix=false --new-from-patch=.flow/tmp/fn64-task7-owned.patch --config=.github/.golangci.yml ./tools/umpire/..., TMPDIR=$PWD/.flow/tmp/fn64-task7-lint-tmp CGO_ENABLED=0 go vet -tags 'disable_grpc_modules,test_dep' -vettool=.bin/errortype -style-check=false ./tools/umpire/internal/execution ./tools/umpire/internal/ir ./tools/umpire/verification ./tools/umpire/temporal/...
- PRs: