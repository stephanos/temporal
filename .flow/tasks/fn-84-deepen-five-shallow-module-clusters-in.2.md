---
satisfies: [R2, R6, R7]
---
# fn-84-deepen-five-shallow-module-clusters-in.2 Driver-contract leaf package replacing the facade mirror

## Description
Move the Driver-facing vocabulary (R2) into one leaf package both the facade and the execution package import, re-export it from the facade by alias, and delete the four pass-through adapters, the two coordinate converters and the profile policy copy loop. The probe on 2026-09-09 confirmed the mirrored ranges reference no `ir` symbol.

**Size:** M
**Files:** `common/testing/testpilot/contract/*.go` (new package; name it `contract` unless the retired gate or an existing import path objects), `common/testing/testpilot/driver.go`, `profile.go`, `prepare.go`, `prepared_case.go`, `prepare_test.go` (deletes the adapter-instantiating test), `internal/execution/contracts.go`, `internal/execution/program.go`, `internal/execution/*_test.go`, `common/testing/testpilot/README.md`, `internal/execution/README.md`, `.plans/UMPIRE4_SPEC.md` (MOD-14 draft restatement), `.plans/UMPIRE4_COMPONENTS.md`, `model/README.md`, `tools/umpire/internal/retiredvocabulary/check.go` (scanned-path registry if the leaf path needs an entry)
**Touches:** [common/testing/testpilot/*.go, common/testing/testpilot/contract/**, common/testing/testpilot/internal/execution/**, common/testing/testpilot/README.md, .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_COMPONENTS.md, model/README.md, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- Follow the existing leaf-package shape: `temporal/internal/activation/activation.go` (imports only the proto package, the facade and `proto`) and `temporal/internal/delivery/ledger.go` (package-owned sentinels and value types).
- Move `Coordinate`, `DriverIdentity`, `ReservationIdentity`, `ReservationRequest`, `EffectResult`, `OpaqueCapability`, `PreparedRole`, `OutcomeSnapshot`, `ReservationCarrierPlan/Topology/Route`, and the `EffectHandle`, `ReservationHandle`, `CapabilityBridge` and `Session` interfaces, plus the profile mirror pairs (`RolePolicy`, `ReservationCarrierPolicy`, `ReservationCarrierShape`, `EnvironmentBinding`, the opcode enum and ceiling) into the leaf. Keep the `MaxOpcode` justification comment once, in the leaf.
- In the facade declare `type X = contract.X` for every moved type; keep `Driver`, `PreparedProgram`, `EntrypointPlan`, `InstructionPlan`, `Expression` in the facade; keep one thin `driverAdapter` for the two methods that take `PreparedProgram`; delete `sessionAdapter`, `effectAdapter`, `reservationAdapter`, `bridgeAdapter`, `publicCoordinate`, `internalCoordinate` and the 22-line `ProfileSpec.policy` copy.
- Delete `TestSessionAdapterReturnsReservationEffectsToTheirDriver` in `prepare_test.go` with the adapters it instantiates. Foreign-handle refusal (quarantining a handle a Session did not issue) is each Driver Session's decision and already exists in the delivery ledger tests and the server Driver tests; the leaf holds only types, so do not move the refusal, pin it through the existing conformance suite.
- Verify MOD-14 with `go list -deps` over every production package outside Testpilot (the memory entry on moved conformance tests names this check).
- Draft the MOD-14 restatement naming the leaf as the admissible Driver dependency and mark it pending GOV-02, following fn-80's EVD-20/EVD-21 pattern; do not reword MOD-12's internals list.
- Docs: the execution README's `ReservationCarrier`/`OutcomeSnapshot` names must resolve to the leaf; add a sentence to the Testpilot README telling Driver authors what to import; the fragment "`common/testing/testpilot` owns the Profile/Driver contract" in `model/ARCHITECTURE.md` is pinned by the documentation gate, so edit the surrounding sentence rather than that one.

### Investigation targets
**Required** (line refs at HEAD ebb94a44e):
- `common/testing/testpilot/driver.go:14-138, 140-239, 262-397` — the mirror, the adapters, the converters
- `common/testing/testpilot/profile.go:40-56, 77-92, 109-136` — profile mirror pairs and the policy copy loop
- `common/testing/testpilot/internal/execution/contracts.go:53-122` and `program.go:15-64, 106-113, 130-149, 305-308` — the internal originals
- `common/testing/testpilot/prepare_test.go:19, 369` — the comment naming the three-place mirror; the adapter-instantiating test to delete
- `common/testing/testpilot/facade_external_test.go`, `conformance_test.go` — must compile unedited

**Optional:**
- `common/testing/testpilot/temporal/internal/delivery/ledger_test.go:475`, `common/testing/testpilot/temporal/server/driver_test.go:249` — where foreign-handle refusal already lives
- `common/testing/testpilot/temporal/internal/activation/activation.go:1-30`, `temporal/internal/delivery/ledger.go:1-52` — leaf patterns
- `tools/umpire/regression/ci_workflow_test.go:152-210` — documentation gate fragments
- `.plans/UMPIRE4_SPEC.md` MOD-12, MOD-14, GOV-02

### Key context
- fn-82 .7 renames `Capability` to `Opcode`, `ProfileSpec.Capabilities` to `Opcodes`, `execution.SlotBridge` to `CapabilityBridge`, `execution.Policy` to `Profile`, `Context()` to `Kind()`; use the landed names.
- The retired-vocabulary scan fails closed on a moved path; a leaf under `common/testing/testpilot/` is inside an existing scan root, so verify rather than add.
- Preserve existing comments when moving code.
## Acceptance
- [ ] the leaf package imports neither `internal/execution` nor `internal/ir`; `internal/execution` and the facade import it; every moved type is re-exported by alias in the facade
- [ ] `driver.go` keeps only `Driver`, `PreparedProgram`, `EntrypointPlan`, `InstructionPlan`, `Expression` and one thin adapter; the four adapters, both converters, `ProfileSpec.policy`'s copy loop and `TestSessionAdapterReturnsReservationEffectsToTheirDriver` are deleted; the `MaxOpcode` comment exists once
- [ ] `conformance_test.go` and `facade_external_test.go` compile unedited; `go test -count=1 -tags test_dep ./common/testing/testpilot/...` and the `^TestCaseRuntimePublicFacadeConformance$` run are green
- [ ] each Driver Session still refuses to quarantine a handle it did not issue, pinned through the existing conformance suite (no refusal logic is added to the leaf)
- [ ] `go list -deps` over production packages outside Testpilot shows no import of `internal/execution`
- [ ] `.plans/UMPIRE4_SPEC.md` MOD-14 restatement drafted and marked pending GOV-02; execution README, Testpilot README, components plan and model README updated; documentation gate passes
- [ ] `make lint-code` clean; `make umpire-check-regression` green
## Done summary
The Driver-facing Testpilot vocabulary (coordinates, reservation identity/request, effect result, opaque capability and CapabilityEffect, EffectHandle, ReservationHandle, CapabilityBridge, Session, PreparedRole, ValueReference/ReferenceKind, OutcomeSnapshot, reservation carrier plan/topology/route, RolePolicy, carrier policy/shape, EnvironmentBinding, Opcode and MaxOpcode) now lives once in the new leaf `common/testing/testpilot/contract`, which imports only the proto package. Internal execution uses it directly (type-checked rewrite to `contract.X`); the facade re-exports every type and constant by alias in `contract.go`. `driver.go` keeps only Driver, PreparedProgram, EntrypointPlan, InstructionPlan, Expression and one thin `driverAdapter`; the session/effect/reservation/bridge adapters, both coordinate converters and `ProfileSpec.policy` are deleted. MOD-14 restatement drafted pending GOV-02; Testpilot, execution, components and model docs point at the leaf.

Equivalence pins (R6): `conformance_test.go` and `facade_external_test.go` unedited and green; `TestCaseRuntimePublicFacadeConformance` green; foreign-handle refusal stays in each Driver Session, pinned by `TestParallelSessionsAndQuarantineCapacity` (server) and `TestQuarantineKeepsReservationOwnershipAndUnwrapsExactHandle` (delivery ledger). `go list -deps ./...`: no package outside Testpilot imports internal/execution. `make lint-code`: 161 (inherited baseline, healthy 14505 -> 161, no touched-file finding). `make umpire-check-regression`: exit 0 with nine live identities.

Decisions taken autonomously:
- Also moved `CapabilityEffect`, `ReferenceKind`/`ValueReference` into the leaf so driver.go holds only the five named types; aliases live in a new facade file `contract.go`.
- Deleted `TestCapabilitiesMatchExecutionOpcodes` alongside `TestSessionAdapterReturnsReservationEffectsToTheirDriver`: with one Opcode declaration it compared a type to itself. The oneof-to-opcode pin in `internal/execution/fault_test.go` remains; its comment was corrected.
- Nil Session/handle/bridge returns are now rejected by execution's existing checks (runtime.go, scheduler.go) instead of the deleted adapters; messages differ but no test or doc pins the old strings, and partial reservation handles are now retained rather than dropped.
- `model/ARCHITECTURE.md` left untouched: it is outside the task's Touches and its sentence stays true.
- Baseline taken from the handoff at the same HEAD (f3f8c625) rather than re-running full gates pre-edit.
- Applied the review's two P3 doc findings (rewrap, "Testpilot's private execution package") after SHIP.

stage: impl-review - ran [2026-09-12T20:05Z..2026-09-12T20:11:53Z] SHIP (claude backend, receipt /tmp/impl-review-receipt-3684356e61f5-fn-84-deepen-five-shallow-module-clusters-in.2.json)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9ddddf5469760005dc8e42df4d773e7efbd1e0f7, eabc8bfc2488e513f7914dea84a0e652966ec47e
- Tests: go test -count=1 -tags test_dep ./common/testing/testpilot/..., go test -count=1 -tags test_dep ./common/testing/testpilot -run '^TestCaseRuntimePublicFacadeConformance$', go test -count=1 -tags test_dep ./common/testing/testpilot/temporal/worker/ -run '^TestFault|^TestSession|^TestPreparedDefinition|^TestWorkerProfile', go test -count=1 -tags test_dep ./common/testing/testpilot/temporal/internal/delivery/ -run '^TestQuarantineKeepsReservationOwnershipAndUnwrapsExactHandle$', go list -deps ./... (no package outside common/testing/testpilot imports internal/execution; leaf imports only api/testpilot/v1), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161 issues, exit 2, inherited baseline; 14505 -> 161; none in touched files), make umpire-check-regression (exit 0; 571 Lean jobs; nine live TestTestpilot identities pass), baseline: green via handoff (verified at f3f8c625 by fn-84-deepen-five-shallow-module-clusters-in.1)
- PRs: