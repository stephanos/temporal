---
satisfies: [R5]
---
# fn-85-model-side-effects-as-typed-actions-and.5 Setup parameters bound by the Profile; switches and per-value live runs

## Description
Bind machine setup parameters through the Profile and let a functional set's `repeat` run each Query's Case once per switch value with a divergence failure (R5). The realization binds each setup parameter and switch value to a dynamic config key; the Go `Environment` gains the config values `DeriveProfile` records; the live harness runs a fixture under each switch value the way the upstream Nexus suites run under HSM and CHASM.

**Size:** M
**Files:** `model/Temporal/Case/Realization/Nexus.lean` (setup and switch bindings to `Temporal.DynamicConfig` keys), `model/Umpire/Command/Records.lean` (`Switch` declared by the realization, not the Model), `model/Umpire/Case/Producer.lean` (an unbindable setup parameter becomes a Known Gap), `common/testing/testpilot/temporal/profile.go` (`Environment.DynamicConfig`; `DeriveProfile` records setup values in the Profile), `common/testing/testpilot/profile.go` (Profile carries bound setup values; fingerprint includes them), `tests/testpilot_run_case_test.go` (`CaseBinding` carries switch values; `runCase` runs once per value and fails naming both values and Verdicts on divergence), `tests/testpilot_async_nexus_case_test.go` (runs under both switch values), `tests/testcore/testpilot/derive_profile_test.go`, `tests/testcore/testpilot/profile.go`
**Touches:** [model/Temporal/Case/**, model/Umpire/Command/Records.lean, model/Umpire/Case/Producer.lean, common/testing/testpilot/profile.go, common/testing/testpilot/temporal/**, tests/testpilot_*_test.go, tests/testcore/testpilot/**]

### Approach
- The async-Nexus live test is the temporary carrier for per-switch-value runs; `.10` replaces it with
  Query 2's test, so its removal there is not a lost gate.
- The upstream suites set `EnableChasm`, `EnableCHASMCallbacks` and `nexusoperation.enableChasmWorkflowOperations` at test-environment construction; a Case cannot set them, so the live harness constructs one environment per switch value and runs the same fixture bytes under each (two environments over one Case, the fn-73 pattern), and the Profile records which value it ran under.
- Setup parameters that the environment can set per Run go through dynamic config overrides in the environment; one it cannot set yields a Known Gap in the Case at production (`Umpire.KnownGap` kind `input`).
- Divergence: the live test fails with both switch values and both Verdicts in one message; the spec treats it as a finding.

### Investigation targets
**Required:**
- `tests/nexus_workflow_test.go:68-87` — the two suites and `newTestEnv`
- `common/testing/testpilot/temporal/profile.go:11-56,207-234` — `Environment`, `DeriveProfile`, `deriveBindings`
- `tests/testpilot_run_case_test.go:18-102` — `CaseBinding`, `bindCase`, `runCase`
- `chasm/lib/nexusoperation/config.go:38-43` and `service/history/hsm/nexusoperations/config.go:148` — the keys
- `model/Temporal/DynamicConfig.lean` — the generated catalog the realization names keys from

**Optional:**
- `tests/testpilot_async_nexus_case_test.go:33-104` — the two-environment byte-identity test to extend

### Key context
- The Case bytes do not depend on switch values; the Profile does (spec API Contracts).

## Acceptance
- [ ] a machine `setup:` parameter is bound by the Profile through the realization's dynamic config key; an unbindable one yields a Known Gap in the Case, pinned; an unknown switch in `repeat` rejects at the set
- [ ] the async-Nexus live test runs once per switch value under two environments with identical Case bytes and reports both Verdicts; an injected divergence (a unit test over the harness) fails naming both values and Verdicts
- [ ] `DeriveProfile` records the bound setup values and the switch value; `derive_profile_test.go` pins it
- [ ] `make umpire-check-live-tests` green with the identity count recorded (CC=/usr/bin/cc, physical TMPDIR)


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
