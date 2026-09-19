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
- [x] a machine `setup:` parameter is bound by the Profile through the realization's dynamic config key; an unbindable one yields a Known Gap in the Case, pinned; an unknown switch in `repeat` rejects at the set
- [x] the async-Nexus live test runs once per switch value under two environments with identical Case bytes and reports both Verdicts; an injected divergence (a unit test over the harness) fails naming both values and Verdicts
- [x] `DeriveProfile` records the bound setup values and the switch value; `derive_profile_test.go` pins it
- [x] `make umpire-check-live-tests` green with the identity count recorded (CC=/usr/bin/cc, physical TMPDIR)


## Done summary

A machine's `setup:` parameters now travel with the declared Model -- `DeclaredNames.setupParameters`
by name, `DeclaredModel.setupParameters` with each parameter's definition (`<family>.setup.<machine>.<name>`),
carried through the instances product and into the Producer's `Input` -- and the realization binds
each one to a dynamic-config key (`Realization.setup : List SetupBinding`). A parameter the
realization binds to no key is an `input` Known Gap of the Case, coded `<parameter>.unbound` with the
parameter as its subject, because the Profile cannot set it and the Case would otherwise run under
whatever value the environment has; a bound one adds nothing to the Case bytes, since its value is
the Profile's to record. Pinned on the success slice with `configuredLifecycle` (`setup: probe: Bool`):
the template's realization leaves it unbound and the Case carries the gap; a realization binding it
to `history.enablechasm` carries none; and `lifecycle` itself declares no parameter, so the
async-Nexus fixture did not move.

### Switches

A switch is declared by the realization, not the Model: `Umpire.Command.Switch` is the record a
`set` resolves `repeat:` against, `Umpire.Case.Producer.SwitchBinding` carries each value with the
configuration it sets, and `Realization.switch?` answers a name or `none`, which is what rejects
the `set` `.7` builds. The Nexus realization declares `implementation` with `hsm` and `chasm`, each
the three settings the upstream Nexus suites set at environment construction
(`history.enablechasm`, `history.enablechasmcallbacks`,
`nexusoperation.enablechasmworkflowoperations`), named through the generated catalog so a key that
leaves the registry fails at elaboration; `switch? "rollout"` is `none`, pinned.

### The Profile

`temporal.Environment` gains `DynamicConfig`, and `DeriveProfile` records it as
`ProfileSpec.Configuration`: lower-case keys (the catalog's spelling, so two spellings of one key are
one Profile and two are a rejection), sorted, an empty key or value rejected. The configuration is
part of `BindingFingerprint`, appended only when present so a Profile that sets none fingerprints as
it did, and so the prepared Case's identity differs per switch value over the same bytes; a
configuration with no environment bindings is an identity too. `derive_profile_test.go` and
`prepare_test.go` pin all of it, including the preparation-error category of an invalid value.

### The live harness

`tests/testcore/testpilot/switch.go` is the harness: `SwitchValue` (a name and the settings it
sets, `Configuration()` as the Profile records them), `NexusImplementationSwitch()`, and
`CheckSwitchAgreement`, which fails naming the switch, both values and both Verdicts rule by rule,
with a unit test injecting a divergence. `TestTestpilotAsyncNexusCase` runs the one Case once per
value, each in a subtest with its own dedicated environment constructed with the value's settings
-- the testpilot environment is a dedicated cluster over in-memory SQLite, so the settings apply
cluster wide and reach the namespace the Case provisions -- with identical Case bytes, Program,
Contract and provenance under both, two Profile identities, and both Verdicts checked for
agreement. `CaseBinding.DynamicConfig` carries the value into `bindCase`.

### What is not here

`DESIGN.md`'s `atConcurrencyLimit` is not bound. This receipt first said no dynamic-config setting
bounds pending Nexus operations; the research spike of 2026-09-19
(`.plans/UMPIRE4_RESEARCH_NEXUS_MODEL.md`) corrected that: there is one key per implementation
(`component.nexusoperations.limit.operation.concurrency` for HSM,
`nexusoperation.limit.operation.concurrencyPerWorkflow.max` for CHASM), both rejecting with
`WORKFLOW_TASK_FAILED_CAUSE_PENDING_NEXUS_OPERATIONS_LIMIT_EXCEEDED` and writing no scheduled event.
Binding it needs a table that varies with `setup:`, a value beside the key, a key per switch value
and an evidence source not keyed by the scheduled event, none of which this task delivers, so the
spike recommends `.10` drop the parameter from the protocol machine rather than carry a meaningless
gap. The `set` command and `repeat:` themselves are `.7`'s; this task delivers the records and the
check it resolves against.

### Gates

`lake build` green (622 jobs); `make umpire-check-testpilot-authoring`,
`umpire-check-case-runtime-conformance` and `umpire-check-goldens` exit 0 with the async-Nexus
fixture byte-identical; `go test -tags test_dep ./common/testing/testpilot/...
./tests/testcore/testpilot/...` green; `GOLANGCI_LINT_BASE_REV=fd1d4bd make lint-code-fast` 0
issues; `LEAN_NUM_THREADS=1 make lint-model` at the baseline; `CC=/usr/bin/cc make
umpire-check-live-tests` green: "failure identities match the empty expected set across 11 passing
identities", `TestTestpilotAsyncNexusCase/hsm` and `/chasm` among them.

Self-review: no second backend is installed in this cloud session, so this owes a cross-model
re-review before the completion review, as the tasks before it do.

### Re-review fix, 2026-09-19

The cross-model re-review found two things. The rewritten live test had dropped the cross-Run
isolation check the earlier test carried; it is restored per switch value: the Case is bound to
two namespaces under each value's cluster, run twice concurrently against each, and every Run's
workflow is described through its own namespace's client and `NotFound` through the other's. And
`Umpire.Command.Switch` was a record nothing used beside `Registry.SwitchEntry` and
`Producer.SwitchBinding`; it is deleted.

## Evidence
- Commits: a6d98db
- Tests: cd model && lake build; make umpire-check-testpilot-authoring; make umpire-check-case-runtime-conformance; make umpire-check-goldens; go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/...; GOLANGCI_LINT_BASE_REV=fd1d4bd make lint-code-fast; LEAN_NUM_THREADS=1 make lint-model; CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-live-tests
- PRs:
