---
satisfies: [R1]
---
# fn-87-tighten-the-testpilot-protocol-glossary.3 Glossary renames, part two: Program, instruction and value names

## Description
Apply the remaining rows of the spec's Renames table (R1) through every layer in one atomic change, the same way .2 did: proto, `make proto`, generated Lean, `Testpilot.Authoring`, Producers, Go runtime and tests, regenerated fixtures, equivalence mapping steps and retired-vocabulary tokens.

**Size:** M (mechanical, wide)
**Files:** `proto/internal/temporal/server/api/testpilot/v1/{program,instruction,value,expression,contract}.proto`, `api/testpilot/v1/*.pb.go`, `model/Testpilot/{Authoring,Correlated}.lean`, `model/Testpilot/Tests/*.lean`, `model/Testpilot/Examples/Synthetic.lean`, `model/Temporal/Testpilot/*.lean`, `model/Temporal/Case/**`, `model/Temporal/Feature/Nexus/Success/{TypedNexus,TypedUnary}.lean`, `model/Umpire/Case/**`, `model/Umpire/Variations/Lowering.lean`, `common/testing/testpilot/**`, `tests/testcore/testpilot/**`, `tests/testpilot_*_test.go`, `tools/umpire/**`, fixtures, `common/testing/testpilot/internal/protocolmigration/mapping.go`, Testpilot READMEs (`internal/execution/README.md`, `temporal/README.md`, `temporal/server/README.md`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`) where they say "response projection" or "opaque capability"
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/Testpilot/**, model/Umpire/Case/**, model/Umpire/Variations/**, model/Temporal/**, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, common/testing/testpilot/**, tests/testcore/testpilot/**, tests/testpilot_*_test.go, tools/umpire/**]

### Approach
- Renames in this task: `ResponseProjection` → `ResponseRead` (and `InvokeRPC.response_projections` → `response_reads`); `ProjectionTarget` → `ReadTarget`; `ProjectionKind` → `ReadCardinality` (`READ_CARDINALITY_ONE`, `READ_CARDINALITY_EMIT_EACH`); `OpaqueCapabilityType` → `OpaqueHandleType` and both `opaque_capability` fields → `opaque_handle`; `capability_slot_id` → `handle_slot_id`; `InvokeRPC` message → `InvokeRpc`; every `Value` oneof arm suffixed `_value` (`text_value`, `natural_value`, `signed_integer_value`, `unsigned_integer_value`, `floating_point_value`; the rest already are); drop `Definition` suffixes on Program declarations: `RoleDefinition` → `Role`, `SlotDefinition` → `Slot`, `ObservationDefinition` → `Observation`, `EntrypointDefinition` → `Entrypoint`, `CleanupDefinition` → `Cleanup`, `InstructionDefinition` → `InstructionNode` (its body stays `Instruction`); `InstructionRef` → `InstructionReference` (it survives .5 for dependencies and `AwaitInstruction`). Not renamed here because later tasks delete them: `ActivationReservationDefinition`, `EnvironmentDefinition`, `InstructionOutcomeDefinition`, `OutcomeFieldDefinition` (.12 derives them and retires their tokens) and the `Program*`/`Contract*` expression messages and other `*Ref` messages (.5 and .6 replace them); record this in the summary. `ResponseProjection.source` is a `FieldPath`, not an operand, so it becomes `path`.
- Go naming: generated Go follows the proto. Hand-written Go keeps Go initialism style where staticcheck ST1003 is on (`.github/.golangci.yml` enables staticcheck `all`): the Opcode `contract.InvokeRPC` and `Session.InvokeRPC` keep `RPC`, the same way hand-written Go writes `CaseID` beside generated `GetCaseId`. Record this in the done summary. The Lean constructor `Testpilot.Authoring.invokeRPC` → `invokeRpc`, `capabilitySlot` → `handleSlot`, `opaqueCapability` → `opaqueHandle`, `responseProjection` → `responseRead`.
- Suffix-free names can clash in Lean under `open temporal.server.api.testpilot.v1` (`Testpilot/Authoring.lean:19`) with Authoring namespaces (`Program.observation`, `Program.role`) and with `Umpire.Case` declarations; resolve each clash deliberately (qualify, or rename the Authoring helper) and list them in the summary.
- Same order and checks as .2 (proto → `make proto` → Lean build → Go build → regenerate → mapping → gates). Field numbers unchanged.
- Mapping steps: key renames (`responseProjections`→`responseReads`, `capabilitySlotId`→`handleSlotId`, `opaqueCapability`→`opaqueHandle`, `text`→`textValue` etc. only inside `Value`, `source`→`path` inside response reads) and enum literal renames (`PROJECTION_KIND_*`). Value-arm renames must be scoped to `Value` objects by descriptor type, not by key name, because `text` is not a Value arm elsewhere.
- Retired tokens: `ResponseProjection`, `ProjectionTarget`, `ProjectionKind`, `PROJECTION_KIND_`, `OpaqueCapabilityType`, `capability_slot_id`, `capabilitySlotId`, `CapabilitySlotId`, `invokeRPC` (Lean/JSON spelling only, see Go naming), `RoleDefinition`, `SlotDefinition`, `ObservationDefinition`, `EntrypointDefinition`, `CleanupDefinition`, `InstructionDefinition`, `InstructionRef`, `responseProjections`. Check that none collides with unrelated live identifiers (for example `Umpire.Case.Projection` must stay legal; `ResponseProjection` as a whole token does not match it).
- READMEs: "response projection" → "response read", "opaque capability"/"capability bridge" → "opaque handle"/"handle bridge" in the Testpilot and model docs the docs-gap scan listed; leave the Go Driver-contract `CapabilityFactory` names alone (not in the Renames table) and say so in the summary.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/{program,instruction,value}.proto`
- `model/Testpilot/Authoring.lean:20-120,223-380` — constructors spelling old names
- `model/Temporal/Testpilot/CaseSupport.lean` — shared Producer defaults
- `common/testing/testpilot/internal/execution/{dataflow.go,projection.go,program.go}` — response projection and handle slot users
- `common/testing/testpilot/contract/{driver.go,profile.go}` — Driver-facing `InvokeRPC` names

**Optional:**
- `common/testing/testpilot/temporal/server/driver.go:199` — `GetOpaqueCapability`
- `.github/.golangci.yml:23-26` — staticcheck `all`

### Key context
- `Umpire.Case.Projection` is the model Projection and keeps its name; only the RPC response read is renamed.
- `Value.natural` is renamed here (to `natural_value`) and removed or kept by .9; do not remove it early.

## Acceptance
- [ ] every rename listed in Approach is applied (names later tasks delete are left for them and listed in the summary) across proto, generated Go and Lean, Authoring, Producers, Go runtime, tests, fixtures and the named READMEs
- [ ] fixtures regenerated through their generators; the equivalence test passes with one declared step per rename (Value-arm steps scoped by descriptor type); every `expected.json` byte-identical
- [ ] retired tokens added; `make umpire-check-retired-vocabulary` green; hand-written Go initialism decision recorded in the summary
- [ ] `make umpire-check-testpilot-protocol`, `make umpire-check-testpilot-authoring`, `make umpire-check-case-runtime-conformance` green; `make lint-model` at 163
- [ ] `make umpire-check-regression` exit 0 with nine passing live identities; `make lint-code` no new issues


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
