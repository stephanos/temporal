---
satisfies: [R4]
---
# fn-82-unify-the-umpire-and-testpilot.7 Correlated protocol, Provenance, Opcode, and generator filter

## Description
Testpilot protocol, Lean authoring, and Go facade (R4, spec §R4): `Scoped` becomes
`Correlated` in proto, Lean, and Go in one pass, the horizon becomes a deadline, `outcome.proto`
folds away, the `Umpire.Case` alias family dies and its provenance half becomes
`Umpire.Provenance`, `Testpilot.Authoring.Monitor` becomes `Contract`, Go exposes `Opcode`, and
the duplicate testpilot mirror leaves `Temporal.API`.

**Size:** M (mechanical sweep across proto, Lean, and Go)
**Files:** `proto/internal/temporal/server/api/testpilot/v1/{contract,run,instruction,outcome}.proto`, `api/testpilot/v1/*.pb.go` (generated), `model/Testpilot/{Authoring,Correlated,Protocol}.lean`, `model/Shared/{CorrelatedProjection,CorrelatedObligation}.lean`, `model/Umpire/Provenance.lean`, `model/Umpire/Case.lean`, `model/Umpire/Case/{Compiler,Correlated,CorrelatedProofs}.lean`, `model/Umpire/Property/Correlated/**`, `model/Temporal/Testpilot/*.lean`, `model/Temporal/Feature/Nexus3/Testpilot.lean`, `model/Temporal/API/Types.lean` (regenerated), `common/testing/testpilot/**` (public names), `tools/umpire/cmd/umpire-gen-lean-api/config.go`, `Makefile`, fixtures, gate
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/Testpilot/**, model/Shared/**, model/Umpire/Case*, model/Umpire/Provenance.lean, model/Umpire/Property/**, model/Temporal/**, common/testing/testpilot/**, tests/testcore/testpilot/**, tests/*testpilot*_test.go, tools/umpire/cmd/umpire-gen-lean-api/**, tools/umpire/internal/retiredvocabulary/check.go, Makefile, .flow/specs/*.md]

### Approach
- Proto (renames only, field numbers unchanged): every `Scoped*` message and enum to `Correlated*`, `ScopedClause` to `CorrelatedRule`, `ScopedEndpoint` to `TraceEnding{PARTIAL,FINAL}`, `Contract.scoped` to `correlated`, `ContractHorizonDefinition` to `ContractDeadline` (fn-80's `rule_events` field rides along), `Instruction.await_outcome` to `await_instruction`, and move `InstructionOutcome*` from `outcome.proto` into `instruction.proto`. Update `TESTPILOT_PROTOCOL_PROTOS` (`Makefile:127-135`), run `make proto`, then rebuild `Testpilot.Protocol` from a clean Lake target (it loads the protos at elaboration and Lake does not track them).
- Lean: `Testpilot/Scoped.lean` to `Correlated.lean` with `Run` to `Monitor`; `Shared/Scoped*` to `Correlated*` with `Coordinate` to `Match` and `Answer` to `Verdict`; `Umpire/Property/Scoped/` to `Correlated/` and `PropertyScopedClause` to `PropertyCorrelatedClause`; `Umpire/Case/Scoped*.lean` to `Correlated*.lean`; `Testpilot.Authoring.Monitor.*` to `Contract.*` plus new `correlated`/`correlatedRule` constructors replacing the hand-built values in `Umpire/Case/Scoped.lean`; `Umpire.Case.Compiler.LoweringError` to `Compiler.Error`; provenance types from `Umpire/Case.lean:15-80` to `Umpire/Provenance.lean` as `Provenance.{Metadata,DefinitionBinding,DefinitionKind,KnownGap}`; delete `Umpire/Case/{Program,Contract,Run,Value,ProtoJSON}.lean`, the `abbrev Case`, and `Testpilot/Tests/Compatibility.lean`; `Umpire/Case.lean` becomes the facade over `Compiler`, `Coverage`, `Correlated`, `Projection`.
- Generator: add `--skip-package` to `tools/umpire/cmd/umpire-gen-lean-api/config.go:52-93`, pass `temporal.server.api.testpilot.v1` in `UMPIRE_GEN_LEAN_API_ARGS`, regenerate `Temporal/API*.lean`, and cover the flag in the generator's golden tests.
- Go: `testpilot.Capability` to `Opcode` and `ProfileSpec.Capabilities` to `Opcodes`; `execution.SlotBridge` to `CapabilityBridge`; `execution.Policy` to `Profile`; the `Context` fields and `EntrypointPlan.Context()` to `Kind`; `recorder.terminalDisposition` and `delivery.TriggerDisposition` to `terminalStatus`/`TriggerStatus`. Add the twenty-four `Scoped*` names, `ContractHorizonDefinition`, and `EntrypointContext` to the retired loop in `protocol_test.go:49-56`.
- Regenerate the sixteen Case fixtures (`make umpire-gen-case-runtime-conformance` twice per `Makefile:1067-1070`); run the conformance, protocol, authoring, and live checks; add compound old names to the gate; respell scanned docs and open specs (fn-26, fn-30, fn-60, fn-78 use "scoped").

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/contract.proto:49-224` and `run.proto:127-140` — the Scoped family and the horizon
- `model/Testpilot/Authoring.lean:361-481` — `Monitor`, `Run`, `Verdict` namespaces
- `model/Umpire/Case.lean`, `model/Umpire/Case/{Scoped,Compiler,Provenance}.lean` — aliases, hand-built correlated values, provenance
- `common/testing/testpilot/{case.go:40-50,driver.go:76-140,profile.go:52-90,protocol_test.go:34-58}` — opcode enum, plans, profile, retired-name test
- `tools/umpire/cmd/umpire-gen-lean-api/config.go:52-93` — flag parsing to extend

**Optional** (reference as needed):
- `common/testing/testpilot/internal/verification/scoped.go`, `scoped_prepare.go` — the Go evaluator to rename
- `model/Temporal/API/Types.lean:12500` — the unused mirror this task removes

### Key context
- Task .1 already added the buf `ignore` entry, so `make lint-protos` stays green.
- Strict ProtoJSON decoding already rejects unknown fields, so a fixture still carrying `scoped` fails to decode; no new error path is needed.
- The activity entrypoint, `RunRef`, `AnyType`, and `ROLE_KIND_PARTICIPANT` stay (spec §Boundaries).
- Retire `ScopedContract`, `ScopedClause`, `ScopedEvidence`, `ScopedEndpoint`, `ContractHorizonDefinition`, `Testpilot.Scoped`, `Shared.ScopedProjection`, `Shared.ScopedObligation`, `Umpire.Case.Scoped`, `PropertyScopedClause`, `LoweringError`, `CaseMetadata`, `CaseDefinitionKind`, `Umpire.Case.ProtoJSON`, `SlotBridge`, `terminalDisposition`.

## Acceptance
- [ ] No `Scoped*` name remains in the proto package, Lean, or Go; `ContractDeadline`, `CorrelatedRule`, `TraceEnding`, and `await_instruction` are in place; `outcome.proto` is gone and `TestProtocolUsesCohesivePublicVocabulary` rejects every retired proto name
- [ ] `Umpire.Provenance` holds the producer-owned types; the `Umpire.Case` alias modules, `abbrev Case`, and `Umpire.Case.ProtoJSON` are deleted; `Testpilot.Authoring.Contract` builds both rule kinds and the Nexus3 Producer uses it
- [ ] `Temporal.API` contains no `Temporal.Server.Api.Testpilot` declarations and the generator's `--skip-package` flag is covered by its tests
- [ ] Go exposes `Opcode`/`Opcodes`, `Kind`, `CapabilityBridge` inside `execution`, and `Profile`; `go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/...` passes
- [ ] The sixteen regenerated Case fixtures pass `umpire-check-case-runtime-conformance`, `umpire-check-testpilot-protocol`, `umpire-check-testpilot-authoring`, `lint-protos`, and the tagged live selector with unchanged Verdicts; the gate passes


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
