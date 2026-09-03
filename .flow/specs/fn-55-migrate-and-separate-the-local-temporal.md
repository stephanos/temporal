# Migrate and separate the local Temporal authority

## Overview

Retire Umpire's deprecated invocation-owned Temporal test server and make authority attachment explicit. Live Umpire execution tests run only from `tests/` against `testcore.TestEnv`; production Umpire packages receive the existing `local.NewAttachedFactory` result and never construct or import a test server.

This is an intentional ownership migration followed by a physical decomposition. `TestEnv` owns the cluster and SDK client. Umpire owns the per-run environment wrapper, SDK worker, Nexus endpoints, workflows, receipts, and cleanup of those run resources. The runner, participant, isolation, evidence, and semantic contracts remain unchanged apart from removing the zero-option authority path and correcting receipts that previously implied ownership of borrowed infrastructure.

## Goal & Context
<!-- scope: business -->

The local environment currently mixes reusable orchestration with a deprecated concrete test-server implementation. It also lets a zero-value Nexus binding create hidden infrastructure. The repository's supported live harness is `tests/testcore.TestEnv`, whose lifecycle is tied to `*testing.T`; it cannot legitimately be created from `local.NewFactory()` or `runner.Adapter` without a global registry, a fabricated `testing.T`, or a new test-harness lifecycle API.

The cleanup therefore makes the real boundary explicit: a test creates `TestEnv`, adapts its client, namespace, and frontend address to `local.AttachedAuthority`, and supplies the resulting factory to the Nexus adapter. Umpire remains unaware of `TestEnv`. Developers gain one supported live harness and a smaller authority module; users and operators gain no production endpoint, credential, persistence, deployment, retry, or configuration surface.

## Architecture & Data Models
<!-- scope: technical -->

- `tests/` owns the only live Umpire harness. A shared test-only helper creates `testcore.TestEnv`, constructs the small `local.AttachedAuthority` value, and calls `local.NewAttachedFactory`.
- `tools/umpire/temporal/nexus` keeps model/program behavior but requires an explicit `umpireruntime.EnvironmentFactory`; its zero value no longer starts infrastructure. The constructor validates a nil factory before runtime I/O.
- `tools/umpire/temporal/local/environment.go` keeps preparation, lifecycle orchestration, synchronization, isolation delegation after fn-53, receipt construction, and runtime-resource translation.
- A private authority module owns the existing starter/authority contracts and Umpire-owned resource algebra. `attached.go` is the sole real authority implementation and owns only per-run workers; its environment marker never represents ownership of the borrowed cluster or client.
- Unit tests below `tools/umpire` use deterministic fake factories, authorities, workers, and clients. Live Nexus, generated-portability, and Run Evaluation proofs move to `tests/` and share the TestEnv helper.

No ambient authority registry is introduced. One attached factory remains sequential. Parallel live cases create independent TestEnv/factory pairs; deliberately sequential portability cases may reuse one TestEnv while retaining distinct run, task-queue, endpoint, workflow, artifact, and evidence identities.

## API Contracts
<!-- scope: technical -->

- `Environment`, `WorkerRegistration`, `AsEnvironment`, `AttachedAuthority`, and `NewAttachedFactory` retain their exact exported shapes and behavior.
- `local.NewFactory` is removed; there is no compatibility wrapper or fallback test-server path.
- Nexus adapter construction gains one required environment-factory input. A zero-value or nil-factory adapter is incomplete and fails with the existing runner adapter-incomplete classification before participant or environment construction.
- The private `authorityStarter` and `temporalAuthority` contracts remain the orchestration seam. No second authority interface, global registry, endpoint option, credential option, or production `TestEnv` dependency is added.
- Generated caller-closure test source remains deterministic and checked in, but its live destination is under `tests/` so it can consume the shared TestEnv harness. It retains the exact admitted Artifact bindings, preflight order, bounded execution, and Run Evaluation comparisons.

## Edge Cases & Constraints
<!-- scope: technical -->

- `testcore.NewEnv` startup failures remain test-harness failures reported through `testing.T`; they are never converted into Umpire lifecycle receipts.
- TestEnv owns and closes its cluster and SDK client after Umpire cleanup. Umpire receipts must never report either resource as acquired or released.
- Nil authority, nil client, blank namespace/address, authority binding drift, worker-construction failure, canceled worker start, canceled stop, failed stop, repeated cleanup, and residual Umpire ownership retain deterministic coverage through fakes and the attached adapter.
- Request validation, nil/canceled context precedence, command validation, worker options and canonical timeout, isolation outcomes, diagnostic sanitization, cleanup retry, resource ordering, Artifact identities, Evidence closure, and semantic results remain exact.
- Live tests use no sleeps, share no attached factory concurrently, leak no worker or Nexus endpoint, and verify Umpire cleanup before TestEnv cleanup.
- Existing comments move with the declarations they describe and otherwise remain unchanged; only comments that falsely describe invocation-owned test infrastructure are rewritten.
- The migration uses the existing TestEnv persistence choice established by the portable executor proof. Reusing one TestEnv sequentially limits startup cost; tests requiring concurrency or authority isolation use separate TestEnv instances.
- Under 10x live-test load, cluster startup is the dominant cost. The plan avoids multiplying TestEnv instances for sequential cases and introduces no runtime concurrency or production scalability promise.

## Approach

1. Make Nexus environment-factory composition explicit while preserving preflight and semantic behavior.
2. Move generated and hand-written live Nexus execution proofs into `tests/`, backed by one shared TestEnv authority helper; keep pure package tests local and fake-backed.
3. Move live Run Evaluation proofs into `tests/` and preserve their exact normal, negative-control, publication, and failure classifications.
4. Remove the legacy zero-option factory and concrete test-server authority, then extract the remaining private authority seam and honest Umpire-owned resource algebra.
5. Reconcile ownership documentation and CI commands, add no-legacy dependency gates, and run focused, race, integration, generator, aggregate regression, formatting, and lint verification.

## Quick commands

```bash
go test -count=1 -tags test_dep ./tools/umpire/temporal/local/... ./tools/umpire/runner/... ./tools/umpire/temporal/nexus/... ./tools/umpire/runevaluation/...
go test -race -count=1 -tags test_dep ./tools/umpire/temporal/local/...
go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpire'
make umpire-check-regression
make fmt-imports
make lint-code
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The deprecated concrete test-server authority, `local.NewFactory`, and every hidden zero-option/fallback authority path are removed. No Umpire production or test source imports the deprecated package, and production `tools/umpire` code does not import `tests/testcore`. Errors: any remaining fallback, global authority registry, fabricated `testing.T`, or second live harness blocks completion.
- **R2:** Nexus execution requires an explicitly supplied non-nil environment factory while retaining the existing checked request, participant, output validation, and runner failure classifications. The zero value performs no infrastructure I/O. Errors: nil factory fails deterministically before participant/environment construction; crossed input still fails before factory access.
- **R3:** Every live Umpire Nexus, generated portability, portable executor, and Run Evaluation proof resides under `tests/`, creates or reuses `testcore.TestEnv`, and binds it through `local.NewAttachedFactory`. Unit tests under `tools/umpire` use fakes and start no live server. Errors: TestEnv startup remains a test failure, while Umpire lifecycle failures remain receipts.
- **R4:** Ownership is truthful: TestEnv owns cluster/client lifecycle; Umpire owns its environment wrapper, per-run worker, endpoints, workflows, and other run resources. Preparation, worker, and cleanup receipts list only Umpire-owned resources, remain deterministically ordered, and end with zero open Umpire handles. Errors: borrowed cluster/client acquisition or release appearing in a receipt blocks completion.
- **R5:** Existing behavior remains exact for validation precedence, bounded phase limits, isolation, participant actions, Evidence closure, normal and duplicate-delivery meaning, generated Artifact bindings, publication, sanitized diagnostics, and transport-scoped identity differences. Errors: semantic drift, changed first diagnostic, stale generated bytes, or preflight runtime I/O blocks completion.
- **R6:** The existing private authority seam is extracted into one focused module without widening it. Attached-authority fakes cover nil/drift/cancellation/partial-worker/stop/retry cases; live cases prove worker and Nexus endpoint cleanup against TestEnv without sleeps, races, or leaked goroutines. Errors: one attached factory used concurrently, TestEnv cleanup preceding Umpire cleanup, or residual owned resources blocks completion.
- **R7:** Runtime, Run Evaluation, architecture, generator, Makefile, and CI documentation/commands describe the TestEnv-attached ownership model and execute the relocated integration tests. Focused unit, race, TestEnv integration, generator determinism, aggregate regression, formatting, and lint gates pass.

## Early proof point

First prove that an explicitly supplied attached factory can run the normal and duplicate-delivery paths against one TestEnv, with crossed input rejected before factory access and cleanup leaving no Nexus endpoint or Umpire-owned handle. If that cannot be achieved through `NewAttachedFactory` without an ambient registry or a production dependency on `tests/testcore`, stop before removing the legacy authority and reconsider the adapter composition seam.

## Boundaries
<!-- scope: business -->

- No new production test-server constructor, authority registry, implicit default, compatibility wrapper, or fallback.
- No production import of `tests/testcore`; TestEnv adaptation exists only under `tests/`.
- No new endpoint, namespace, credential, persistence, deployment, scheduler, fleet, recovery, retry, authorization, metrics, or logging surface.
- No change to model meaning, Artifact schemas, protobufs, Lean sources, portable-plan semantics, observation/property evaluation, or production canary policy.
- No rewrite of the repository-wide TestEnv lifecycle and no deletion or modification of the deprecated package outside Umpire's dependency surface.

## Decision Context
<!-- scope: both — conditionally substructured -->

`TestEnv` requires `*testing.T`, reports startup failure with `t.Fatalf`, and registers cleanup with `t.Cleanup`; the zero-argument `local.NewFactory` cannot create it honestly. Retiring the zero-option API is therefore required, not an optional cleanup. A global TestEnv registry would make test order and concurrency ambient and is rejected.

The existing fn-28 portable executor already establishes the intended seam: `tests/` constructs TestEnv, adapts client/namespace/address, and gives `NewAttachedFactory` to an adapter. This spec generalizes that pattern and reuses a TestEnv sequentially where isolation is provided by fresh per-run identities. Independent TestEnvs are reserved for genuinely parallel or authority-isolation cases because each cluster has meaningful startup and memory cost.

The generated live test moves to `tests/` rather than teaching production Umpire code about the test harness. Its generator continues to own deterministic checked source and exact bindings; the canonical Artifact fixtures remain single-source and are loaded through a tests-only helper instead of copied.

Crash behavior remains test-process scoped: a crash may leave TestEnv resources until process exit, while normal cleanup must close every Umpire-owned resource before `t.Cleanup` closes TestEnv. No durable recovery is introduced.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Remove deprecated authority and all hidden fallback paths | `.3`, `.4` | — |
| R2 | Require explicit Nexus environment factory | `.1`, `.3` | — |
| R3 | Consolidate all live Umpire proofs under tests/TestEnv | `.1`, `.2`, `.4` | — |
| R4 | Report only truthful Umpire-owned resources | `.3`, `.4` | — |
| R5 | Preserve runtime, generated, evidence, and semantic behavior | `.1`, `.2`, `.4` | — |
| R6 | Extract and failure-test the private attached authority seam | `.3` | — |
| R7 | Reconcile documentation, CI, and full gates | `.4` | — |

## References

- Umpire 4 rules MOD-06 through MOD-08 and EVD-01 require small cohesive interfaces, explicit ownership, isolated testability, and a thin runtime.
- Fn-28 establishes the supported TestEnv-backed `NewAttachedFactory` pattern and truthful borrowed-authority ownership.
- Fn-53 owns isolation collection and must land first because it changes the same environment implementation.
- Fn-52 should consume the final explicit TestEnv attachment for its remaining disposable gRPC proof.
