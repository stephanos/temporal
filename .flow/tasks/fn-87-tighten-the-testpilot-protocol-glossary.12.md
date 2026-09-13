---
satisfies: [R10]
---
# fn-87-tighten-the-testpilot-protocol-glossary.12 Derived environment bindings, activation reservations and outcome fields; default instruction limits

## Description
Stop writing what preparation can derive (R10): the environment binding list is the set of bindings roles and expressions reference; activation reservations follow from the instructions that start activations; an instruction's outcome fields follow from its kind; instruction limits equal to the Profile's defaults are omitted. The Profile gains instruction defaults. ART-13 requires a Program to "declare a complete closed graph of symbolic text bindings", so draft its restatement beside SEM-16's.

**Size:** M
**Files:** `proto/.../v1/{program,instruction}.proto`, `api/testpilot/v1/*`, `common/testing/testpilot/{profile.go,prepare.go,prepare_test.go,preparation_error_test.go}`, `common/testing/testpilot/contract/profile.go`, `common/testing/testpilot/internal/execution/{prepare.go,dataflow.go,carrier.go,prepare_test.go}`, `common/testing/testpilot/temporal/{profile.go,profile_test.go}`, `common/testing/testpilot/conformance_test.go`, `tests/testcore/testpilot/profile.go`, `tools/umpire/cmd/umpire-run/*`, `model/Testpilot/Authoring.lean` (`environment`, `reservation`, `node` outcome and limits), `model/Temporal/Testpilot/CaseSupport.lean` (`statusOutcome`, `bounds`), `model/Temporal/Case/Template/{Workflow,NexusOperation}.lean`, `model/Temporal/Testpilot/{WorkerOutage,GetSystemInfo,Conformance}.lean`, `model/Temporal/Feature/Nexus/Success/*.lean`, `model/Testpilot/Examples/Synthetic.lean`, fixtures, mapping, `.plans/UMPIRE4_SPEC.md` (ART-13 draft), docs (`common/testing/testpilot/README.md:7-9`, `internal/execution/README.md:116-122`, `temporal/README.md:42-51`, `model/README.md:49-50`, `model/ARCHITECTURE.md:155-157`, `tests/testcore/testpilot/README.md:13`)
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, common/testing/testpilot/**, tests/testcore/testpilot/**, tools/umpire/**, model/Testpilot/**, model/Temporal/**, model/README.md, model/ARCHITECTURE.md, .plans/UMPIRE4_SPEC.md]

### Approach
- Environment: remove `Program.environment`. Preparation derives the set from `Role.namespace_binding_id`, `Role.resource_binding_id` and every `environment_binding_id` reference, then resolves each against the Profile exactly as `resolveEnvironment` does (`execution/prepare.go:387`); a referenced binding the Profile lacks still rejects. The "declared but unused binding" check (`prepare.go:89-93`) goes, since an unused binding can no longer be declared. `temporal.DeriveProfile`'s `deriveBindings` reads the derived set.
- Reservations: remove `InstructionNode.activation_reservations`. Today every reservation sits on the one controller instruction that starts the workflow and reserves one activation of every non-controller entrypoint (async-nexus: workflow + handler; typed-nexus: workflow + two handlers; typed-unary and worker-outage: workflow). Derive per the Profile's reservation carriers (`contract/profile.go:27-38`): an instruction whose role and method is a carrier reserves one activation of each non-controller entrypoint whose kind its shapes admit, with the count rule `bindReservations`/`reservationCount` uses today (`execution/prepare.go:558-617`). If more than one instruction could carry the same entrypoint's reservation, reject at preparation as `unsupported` naming both instructions (no checked-in Case has two carriers; record the rule).
- Outcome fields: remove `InstructionNode.outcome` (`InstructionOutcomeDefinition`/`OutcomeFieldDefinition`). `bindOutcomes` (`dataflow.go:189`) derives the field set and types from the instruction kind and, for `InvokeRpc`, the method's response descriptor; prove the derived set equals every written set in today's fixtures before deleting (the mapping step validates it).
- Instruction defaults: `ProfileSpec` gains `InstructionDefaults { timeout_milliseconds, max_attempts }`, included in the Profile snapshot and identity (ART-14 fingerprint covers bindings only; state in the summary whether the Profile identity string must change for Drivers). `InstructionLimits` fields become presence-bearing (`optional`, or absent message); an absent value takes the Profile default. The Temporal default set (from .10) gains defaults valued at the most common value across today's Cases (timeout 10000 ms, one attempt); instructions with another value keep it explicit.
- Error surface: the derived fields are gone from the wire, so a Case that still writes one is rejected by strict ProtoJSON decode naming the unknown field, before `Prepare` (`testpilot.DecodeCaseProtoJSON`). `PreparationError` excludes decode failures (`preparation_error.go` doc comment), so the R10 error is the decode error: add a test per removed field asserting the error names it, and record this reading of "rejects at preparation" in the done summary.
- ART-13 draft: under ART-13 in `.plans/UMPIRE4_SPEC.md`, a restatement marked `*(drafted by fn-87; awaiting GOV-02 approval.)*`: a Program's symbolic binding graph is the closed set its roles and expressions reference, derived at preparation; the Profile still owns the physical values. Do not edit the approved text.
- Lean: Authoring drops `Program.environment`, `reservation`, outcome and limit arguments that are defaults; `CaseSupport.statusOutcome` and `bounds` go or shrink.
- Mapping: validated steps: dropped environment list equals the derived reference set; dropped reservations equal the derived ones; dropped outcome fields equal the derived ones; dropped instruction limits equal the Profile defaults. Retire `ActivationReservationDefinition`, `EnvironmentDefinition`, `InstructionOutcomeDefinition`, `OutcomeFieldDefinition`, `activationReservations`.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/execution/prepare.go:85-95,260-400,555-620`
- `common/testing/testpilot/internal/execution/dataflow.go:120-240` — bounds and outcomes
- `common/testing/testpilot/contract/profile.go` and `common/testing/testpilot/profile.go:60-90`
- `common/testing/testpilot/temporal/profile.go` — `deriveBindings`, reservation carriers
- `.plans/UMPIRE4_SPEC.md` ART-13, ART-14

**Optional:**
- `model/Temporal/Testpilot/CaseSupport.lean:20-40`
- `common/testing/testpilot/temporal/README.md:42-51`

### Key context
- Reservations gate worker activation delivery (EVD-14, EVD-16); the async and typed Nexus live tests are the pins that derived reservations equal today's.

## Acceptance
- [ ] `Program.environment`, `InstructionNode.activation_reservations` and `InstructionNode.outcome` are gone; preparation derives each and the derived values equal today's for every checked-in Case
- [ ] a Case writing a removed field fails strict decode naming the field (test per field); a referenced binding missing from the Profile and an ambiguous reservation carrier reject at preparation (unit tests)
- [ ] `ProfileSpec` carries instruction defaults; instruction limits equal to them are omitted from every fixture; other values stay explicit
- [ ] ART-13 restatement drafted under GOV-02; docs listed in Files updated
- [ ] equivalence test passes with the validated derivation steps; Verdict pins unchanged; `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
