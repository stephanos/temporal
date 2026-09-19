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
Preparation now derives what a Case no longer writes, and the wire no longer carries it. `Program.environment`, `InstructionNode.activation_reservations` and `InstructionNode.outcome` are removed, with field numbers kept dense. Instruction limits the Case omits take the new `ProfileSpec.InstructionDefaults`.

**Derivations** (Go preparation, `internal/execution`)
- **Environment:** `EnvironmentBindingIDs` lists each binding a role names (namespace, then resource, in role order), then each other binding an expression references. Preparation resolves exactly that set. A binding the Profile lacks rejects `unknown` at `environment`, naming the binding. `temporal.DeriveProfile` reads the same set through the facade.
- **Reservations:** `deriveReservations` covers an ordinary controller instruction that invokes a Profile carrier method. It reserves one activation of each workflow and Nexus-handler entrypoint whose kind the carrier's shapes admit, in declaration order. If two instructions could carry the same entrypoint, preparation rejects `unsupported` naming both. The existing count, attempt-weight and topology checks run unchanged afterwards. `DeriveProfile` treats every ordinary controller `StartWorkflowExecution` as a carrier, with shapes that count the Program's workflow and handler entrypoints.
- **Outcome fields** follow from the instruction:
  - every instruction: status and detail
  - `InvokeRpc` and `CompleteNexusOperation`: protocol code
  - worker instructions: SDK failure code
  - `AwaitInstruction`: a text VALUE
- **Limits:** `InstructionLimits` keeps presence through single-arm oneofs. `InstructionDefaults.Resolve` is the one resolution rule, shared by preparation and the server Driver. The worker reads `InstructionPlan.TimeoutMilliseconds`, `MaxAttempts` and `Reservations`. An omitted limit the Profile has no default for rejects `malformed`. `temporal.DefaultInstructionLimits` is 10000 ms and one attempt.

**Lean**
- `Program.node` drops its outcome and reservation arguments and takes an optional `limits`.
- `Program.instructionLimits` takes an optional timeout and optional attempts.
- `Program.environment`, `outcome`, `outcomeField`, `reservation` and `CaseSupport.statusOutcome` are removed.
- Every Producer writes only a non-default limit: 5000 or 20000 ms.
- `FaultRealization` loses its outcome, and its limits default to empty.

**Fixtures and oracle**
- All fixtures were regenerated. `expected.json` is unchanged.
- The new R10 oracle step (`protocolmigration/derived.go`) spells each rule a second time, independently. It checks each dropped environment list, reservation list, outcome declaration and default limit against its rule. Confirmed red with a wrong default and with a missing await VALUE.
- Test Profiles: synthetic and correlated take 1000 ms and one attempt; the facade conformance Profile spells the Temporal defaults.

**Decisions** (recorded in the fn-87 Planning decisions, "decided in .12")
1. "Rejects at preparation" is read as strict decode, which runs before `Prepare`. `TestCaseProtoJSONRejectsDerivedDeclarations` has one case per removed field and asserts `unknown field "<name>"`.
2. Outcome fields are what each instruction produces, not exactly the set each fixture wrote. No kind-only rule matches every fixture: the synthetic Finish declared a message VALUE and the async Finishes did not.
   - A Finish or RespondNexus result ends its activation, so neither derives VALUE. The worker no longer copies that result into its outcome.
   - The oracle drops such a declaration only when no expression reads it.
   - The awaited Nexus result is typed text.
   - The spec's "method response descriptor" hint is not used, because RPC payloads are read only through response reads.
3. Every reservation counts one. Multi-activation reservations, partial reservations and handler-only carriers can no longer be expressed. Carrier and scheduler tests were rewritten accordingly, and the combined two-carrier bound test was removed.
4. Instruction defaults are a Go value, and a zero field supplies nothing. They are part of the Profile snapshot, and the Profile identity doc now names them. ART-14's binding fingerprint is unchanged. Existing Driver identity strings need not change, because every checked-in Case resolves to the limits it wrote.
5. The ART-13 restatement is drafted under GOV-02.

**Tests added**
- `TestPrepareDerivesTheEnvironmentBindingGraph` (a missing binding, with the exact diagnostic)
- `TestPrepareRejectsAnAmbiguousReservationCarrier` (confirmed red without the check)
- `TestPrepareDerivesReservationsFromTheProfileCarriers`
- `TestInstructionLimitsTakeTheProfileDefaults` (the default and the explicit override; an absent default, a negative default, and defaults above the ceilings)
- `TestPrepareDerivesOutcomeFieldsFromTheInstruction`
- `TestDeclaredDerivedDeclarationsStepDropsOnlyWhatPreparationDerives`
- `TestCaseProtoJSONRejectsDerivedDeclarations`

**Retired names:** `EnvironmentDefinition`, `ActivationReservationDefinition`, `ActivationReservations`, `activation_reservations`, `InstructionOutcomeDefinition`, `OutcomeFieldDefinition`.

**Outside the declared Touches** (forced by the removals): `model/Umpire/Variations/{Lowering,Tests/Lowering}.lean`, `model/Umpire/Case/Tests/{CorrelatedFixtures,FieldLowering}.lean`, `model/Umpire/ARCHITECTURE.md`, `tools/umpire/internal/retiredvocabulary/check.go`, and the `.flow` spec decision and review state.

**Gates**
- Baseline was green: regression receipt 8954093a was honored and the oracle passed.
- After the change:
  - oracle, testpilot/umpire/testcore go tests, Lean protocol and authoring checks, the retired-vocabulary gate, and the lake build of Umpire, Temporal and the test libraries all pass
  - `make lint-model`: 163, the baseline
  - `make umpire-check-regression`: exit 0 with 9 passing live identities at 6c9de5df, first run with no flakes. The async and typed Nexus live tests passed.
  - `go clean -cache && make lint-code`: 163. The 2 new `enforce-switch-style` issues were fixed in 58853057, after which lint-code-fast reported 161.
  - 58853057 only adds switch default clauses, re-verified by go test.

stage: impl-review - ran (claude backend, SHIP on the first round; P3 notes applied: a parenthesized condition and an oracle ordering comment. The duplicated StartWorkflowExecution constant across the temporal and worker packages was left as is.)
## Evidence
- Commits: 5c0d7faf3a3ae56b44b017fe2ff9fcb3902a9fc0, 6c9de5df35f035ec492cd8acad6a699c6408b36b, 58853057f3a3178063ac01b10895a8a085877e2e
- Tests: baseline: green via receipt 8954093a (regression) + oracle go test green pre-edit, go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/, make umpire-check-testpilot-protocol, make umpire-check-testpilot-authoring, make umpire-check-retired-vocabulary, lake build Umpire Temporal UmpireTests TemporalModelTests TemporalExperimentalTests, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (exit 0, 9 passing live identities, at 6c9de5df; 58853057 adds only switch default clauses, re-verified by go test ./common/testing/testpilot/... ./tools/umpire/... ./tests/testcore/testpilot/...), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (163 at 6c9de5df: 2 new enforce-switch-style fixed in 58853057; make lint-code-fast then 161), make lint-model (163)
- PRs: