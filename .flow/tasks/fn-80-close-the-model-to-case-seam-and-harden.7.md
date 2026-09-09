---
satisfies: [R5]
---
# fn-80-close-the-model-to-case-seam-and-harden.7 Derive ProfileSpec from a Case and add the RunCase helper

## Description
Implements R5 (spec §R5). Adds `temporal.DeriveProfile` in the composite Driver package and a two-layer runner in the fixture package: `BindCase` takes explicit bindings, endpoint-creation mode, and identity, and `RunCase` is the happy-path wrapper. Keeps the hand-written async-nexus Profile as the derivation oracle and migrates the live tests to the helpers. Runs after task .4 because that task changes the live test's Verdict assertions.

**Size:** M
**Files:** `common/testing/testpilot/temporal/profile.go` (new), `common/testing/testpilot/temporal/profile_test.go` (new), `tests/testcore/testpilot/run_case.go` (new), `tests/testcore/testpilot/derive_profile_test.go` (new, equality with `AsyncNexusProfile`), `tests/testpilot_async_nexus_case_test.go`, `tests/testcore/testpilot/README.md`, `common/testing/testpilot/temporal/README.md`
**Touches:** [common/testing/testpilot/temporal/profile.go, common/testing/testpilot/temporal/profile_test.go, common/testing/testpilot/temporal/README.md, tests/testcore/testpilot/**, tests/testpilot_async_nexus_case_test.go]

### Approach
- `DeriveProfile(source, catalog, env)` walks `Program.roles`, `environment`, every `InvokeRPC` method, every `ActivationReservationDefinition`, and every opcode present; carrier `MaximumCount` = number of reserving nodes per entrypoint context; `Identity` from `env.Identity`; limits cloned from the Case. Unknown method or role kind is an error. Never widen.
- Oracle: `AsyncNexusProfile` at `tests/testcore/testpilot/async_nexus_fixture.go:21-56` stays; add an equality test in the fixture package (the facade boundary test at `common/testing/testpilot/facade_external_test.go:128-137` forbids the facade package from importing `tests/testcore`, and `DeriveProfile` lives in `temporal`, which may import the SDK but must not import tests).
- Two-layer runner replacing `newTestpilotLiveBinding` (`tests/testpilot_async_nexus_case_test.go:132-193`), which today varies the concrete bindings per environment (`:54-60`) and the endpoint-creation flag (`:117-123`):
  - `BindCase(t, env, source, CaseBinding{Identity, Namespace, TaskQueue, NexusEndpoint, CreateEndpoint bool}) LiveBinding` derives the Profile from the explicit binding, provisions namespace and (when `CreateEndpoint`) the Nexus endpoint, builds `temporaldriver.New`, and prepares. `LiveBinding` exposes `Prepared`, `Driver`, `Client`, and `Run(ctx) (*Run, *Verdict, error)`.
  - `RunCase(t, env, name, binding CaseBinding) (*Run, *Verdict)` loads the fixture by name via `loadTestpilotCase`, calls `BindCase` with `CreateEndpoint: true`, runs once, and `require.NoError`s the Run error (R6).
  - The two-environment concurrent test and the missing-endpoint test use `BindCase` directly; single-shot tests use `RunCase`.
- `Identity` comes from `CaseBinding.Identity`; the equality test passes the oracle's literal identity so `DeriveProfile` output compares equal to `AsyncNexusProfile` byte for byte. `AsyncNexusEnvironment` gains an `Identity` field or is replaced by `CaseBinding`; keep `artifact_test.go:181`'s use of the hand-written Profile working either way.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/profile.go:21-135` — `ProfileSpec`, `RolePolicy`, carrier shapes, `Snapshot`
- `tests/testcore/testpilot/async_nexus_fixture.go:15-56` — the exact oracle output
- `tests/testpilot_async_nexus_case_test.go:47-193` — the sequence being replaced

**Optional** (reference as needed):
- `common/testing/testpilot/temporal/catalog.go:11-21` — `NewWorkflowServiceCatalog`
- `tests/testpilot_testenv_test.go` — environment provisioning

## Acceptance
- [ ] `DeriveProfile` output equals `AsyncNexusProfile` for the async-nexus Case (equality test in `tests/testcore/testpilot`)
- [ ] `DeriveProfile` errors on an unknown method or role kind; a Case with no worker roles yields no worker policy and no carriers; derived capabilities equal the set of opcodes present
- [ ] `TestTestpilotAsyncNexusCase` uses `BindCase` for both environments with unchanged asserted Verdicts and isolation checks; the missing-endpoint test uses `BindCase` with `CreateEndpoint: false` and still asserts incomplete plus inconclusive; at least one single-shot test uses `RunCase`
- [ ] The equality test supplies the oracle's identity through `CaseBinding.Identity` and compares the full `ProfileSpec` including `Identity`
- [ ] `TestPublicPackageDependencyBoundary` still passes; `go list -deps` of `common/testing/testpilot/temporal` excludes `tests/testcore`
- [ ] READMEs for the composite Driver and the fixture package describe `DeriveProfile` and `RunCase` and state that MOD-12's `Prepare` then `Run` sequence is unchanged

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
