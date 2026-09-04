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
- [ ] Lean emits a reproducible Case whose identities, typed values, paths, limits, and monitor
  obligations survive Go preparation exactly.
- [ ] The `GetSystemInfo` Case compiles and prepares with zero Host I/O, and its different method,
  projection, and Contract shape require no new instruction or special execution/verification/
  Temporal branch.
- [ ] Unsupported lowering fails deterministically; crossed-kind or descriptor mutations fail
  preparation before I/O.
- [ ] The composite Host preserves server/worker package boundaries and uses opaque completion
  authority without exposing its URL, headers, or token.
- [ ] Integration performs start workflow → SDK Nexus start/await → async response → controller
  completion → bounded history retrieval → workflow finish.
- [ ] Verdict evidence is only declared server-history Observations, and one PreparedCase drives
  isolated sequential/concurrent Runs including non-success and timeout cases.
- [ ] `make umpire-build-model` and `go test -count=1 -tags 'test_dep integration' ./tests -run
  TestUmpireAsyncNexusCase` pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
