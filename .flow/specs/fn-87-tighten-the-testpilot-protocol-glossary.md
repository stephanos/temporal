> HTML render lens: open local `.flow/artifacts/fn-87-tighten-the-testpilot-protocol-glossary/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

The Testpilot protocol (`proto/internal/temporal/server/api/testpilot/v1`, 855 lines in seven
files) is the wire every Lean Producer writes and every Go runtime component reads: Case, Program,
Contract and Run. It grew one capability at a time (instructions, captures, the correlated
capability, faults, environment bindings), and it shows:

- **Three expression languages.** `ProgramExpression` and `ContractExpression` are the same seven
  operators spelled twice as fourteen messages, and the correlated capability adds a third
  (`CorrelatedCorrelation`, `CorrelatedComparison`, `CorrelatedOperand`, `CorrelatedPredicate`).
- **Names that break SEM-19** ("one word per concept, spelled the same in Lean, protobuf, Go, command
  syntax and prose"): `RunStatus` for the glossary's Run disposition; `clauses` of type
  `CorrelatedRule` with a `clause_id`; "projection" for both Umpire's model Projection and reading
  fields out of an RPC response; "capability" for both `Umpire.Capability` and an opaque effect
  handle; `NONTERMINAL` and `PENDING` for the same unfinished rule.
- **Special cases instead of structure.** A Run Event carries the fault it records in a dedicated
  optional field, and Contract expressions address it through two dedicated enum values; the next
  event kind with a payload would add two more of each.
- **Duplicates and dead shapes.** Two encodings of an opaque handle, `natural` beside
  `unsigned_integer`, a capture type that repeats `SingularType`, three different binding messages,
  a nested `version` inside a versioned Case, an enum no wire message references, and a string
  equality guard bolted onto a path guard.
- **A layout that scatters concepts.** The correlated capability lives in three files;
  `FormatVersion` sits in `value.proto`; `FaultInjected` sits in `instruction.proto` "so the accessor
  stays in the file that declares the enum".

The wire has no compatibility promise: the `buf` breaking check ignores this package ("an internal
test-only wire with no compatibility promise; its messages are renamed wholesale"), every Case in the
repository is a generated fixture, and fn-85 is about to add Nexus instructions and fn-86 to migrate
every hand-written Case. Cleaning the protocol now means those specs author against the final shapes
instead of adding to the current ones.

The generated fixtures are also hard to review. `typed-nexus-case.json` is 2,025 lines and 316 KB:
21 parameterized model values of about 11,000 characters each make up 77% of its bytes; 34 distinct
dotted Definition IDs appear 104 times; guards, outcome declarations, instruction limits and
dependency lists take 660 lines, and every one of the 14 instruction guards across the six
checked-in Cases is "every dependency succeeded"; fields appear in alphabetical order, so an
instruction's id sits after its guard; field paths are nested objects; enum literals are numbers;
and each Case carries 24 to 83 limit fields, none of which takes more than four distinct values
across the six Cases.

This spec makes the protocol consistent with the glossary, structured by concept, smaller, and
readable as a reviewed artifact, and documents how it is extended. Breaking changes are allowed.
Verdicts do not change: every migrated Case reaches the same Verdict on the same Run as the fixture
it replaces.

## Architecture & Data Models
<!-- scope: technical -->

### Review findings

Each finding names the requirement that addresses it.

| # | Finding | Kind | R |
| --- | --- | --- | --- |
| 1 | `ProgramExpression` and `ContractExpression` duplicate path, present, equals, compare, not, all and any as fourteen messages | over-engineering | R3 |
| 2 | The correlated capability has its own predicate, comparison, operand and correlation messages for the same boolean and comparison operators | over-engineering | R3 |
| 3 | `equals` is a separate operator while `ComparisonOperator` has only ordering comparisons; the correlated comparison has only equal and not-equal | inconsistency | R3 |
| 4 | `RunStatus` names the glossary's Run disposition | naming (SEM-19) | R1 |
| 5 | `CorrelatedContract.clauses` holds `CorrelatedRule` values keyed by `clause_id`; the glossary calls them Correlated Rules | naming (SEM-19) | R1 |
| 6 | `ResponseProjection`, `ProjectionTarget` and `ProjectionKind` read RPC response fields into Slots; "Projection" is `Umpire.Case.Projection`, which the correlated projection correctly shares | naming (SEM-19) | R1 |
| 7 | `OpaqueCapabilityType` and `capability_slot_id` name an effect handle; "Capability" is `Umpire.Capability`, and the runtime concepts already say "effect handles" | naming (SEM-19) | R1 |
| 8 | `ContractStateStatus.NONTERMINAL` and `RuleVerdictStatus.PENDING` name one unfinished rule twice | naming (SEM-19) | R1 |
| 9 | `CorrelatedValue { definition_id, value }` is `Umpire.ModelValue` under another name | naming (SEM-19) | R1 |
| 10 | `InvokeRPC` generates `Instruction_InvokeRpc`, so one instruction has two spellings in Go | naming | R1 |
| 11 | `RespondNexus` beside `StartNexusOperation` and `CompleteNexusOperation`; `NexusResponseKind` answers an operation, not "Nexus" | naming | superseded: fn-85 R10 removes both messages, so this spec does not rename them |
| 12 | `Value` oneof fields mix `text`, `natural`, `signed_integer`, `floating_point` with `bool_value`, `bytes_value`, `enum_value`, `list_value` | naming | R1 |
| 13 | `Definition` suffix on some declarations (`RoleDefinition`, `SlotDefinition`, `ContractRuleDefinition`) and not others (`Program`, `CorrelatedRule`, `CorrelatedTransition`); `InstructionDefinition` holds an `Instruction` | naming | R1 |
| 14 | `source` means a path operand, a Run Event's origin, an evidence source name and an operand of an expression | naming | R1 |
| 15 | `INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS` beside `SDK_FAILURE` | naming | R1 |
| 16 | The correlated capability is spread over `instruction.proto` (evidence lift), `contract.proto` (contract) and `run.proto` (evidence); `FormatVersion`, `SlotDefinition` and `ObservationDefinition` live in `value.proto`; `FaultInjected` lives in `instruction.proto` | structure | R2 |
| 17 | Field numbers skip (`CorrelatedRule` starts at 7; `CorrelatedEvidence` skips 5; `Program.environment` is 8 after `limits` 7) without `reserved`, suggesting removed fields that never existed | structure | R2 |
| 18 | Most messages outside the correlated capability have no comment; `InvokeRPC`, `RequestAssignment`, `ActivationReservationDefinition`, `SlotDefinition` and every Limits field are undocumented | clarity | R2 |
| 19 | `RunEvent.fault_injected` is a per-kind optional field and `RunEventField` adds `FAULT_ROLE_ID` and `FAULT_KIND`; each future event payload repeats that | extensibility | R4 |
| 20 | Presence is expressed three ways: single-member oneofs (`RunDiagnostic.support`, `Run.evaluation_failure`), empty marker messages (`PresenceSelector`, `RunRef`), and a `bool present` whose `false` has no meaning (`CorrelatedPredicate`) | inconsistency | R5 |
| 21 | `ContractDeadline` states "exactly one bound is positive" in a comment over two fields, with `violation_state_id` between them | clarity | R5 |
| 22 | Two encodings of an opaque effect handle: `SlotDefinition.opaque_capability` (the one fixtures use) and `SingularType.opaque_capability` | duplicate | R6 |
| 23 | `Value.natural` beside `Value.unsigned_integer`, with no fixture using either | duplicate | R6 |
| 24 | `ContractCaptureType` repeats three arms of `SingularType` | duplicate | R6 |
| 25 | `CorrelatedBinding`, `CorrelatedEvidenceBinding` and `CorrelatedEvidenceField` are three shapes for a named value, one of them string-typed | duplicate | R6 |
| 26 | `CorrelatedContract.version` versions a message inside a Case that already carries `FormatVersion` | duplicate | R6 |
| 27 | `EntrypointKind` is referenced by no wire message; the Go runtime uses it as its own classification of the activation oneof | dead shape | R6 |
| 28 | `CorrelatedEvidenceRule.guard_equals_text` refines `guard` with a string equality instead of an expression | ad hoc | R3, R6 |
| 29 | `ScalarKind` distinguishes protobuf wire encodings (`SINT32`, `FIXED64`, `SFIXED64`) that `Value` does not represent, since every integer is canonical base-10 text | possible over-engineering | R6 |
| 30 | Adding an instruction, fault kind, Run Event payload or expression leaf touches the protocol, Lean generation, `Testpilot.Authoring`, the Go interpreter, Profile Opcodes and conformance classes, with no written checklist | extensibility | R7 |

### Target structure

One file per concept, each message documented, field numbers dense:

| File | Contents |
| --- | --- |
| `case.proto` | `Case`, `FormatVersion`, `CaseProvenance` |
| `value.proto` | `Value` and its list and map forms, `ValueType` and its arms, `FieldPath` |
| `expression.proto` | `Expression` and its references (R3) |
| `program.proto` | `Program`, roles, entrypoints and activations, Slots, Observations, environment bindings, cleanup, `ProgramLimits` |
| `instruction.proto` | the instruction graph node, `Instruction` and every instruction message, outcomes, response reads, `InstructionLimits`, fault kinds |
| `contract.proto` | `Contract`, rules, states, transitions, captures, `Deadline`, `ContractLimits` |
| `correlated.proto` | the correlated contract, its rules, projection, evidence lift and evidence (`CorrelatedEvidence`) |
| `event.proto` | Run Event kinds and filters, shared by Contract and Run |
| `run.proto` | `Run`, Run Events and their payloads (R4), diagnostics, cleanup, `Verdict` |

### One expression language

`Expression` replaces `ProgramExpression`, `ContractExpression` and the correlated predicate,
comparison, operand and correlation messages:

```proto
message Expression {
  oneof expression {
    Value literal = 1;
    Reference reference = 2;          // slot, outcome, run, environment, observation, run event,
                                      // capture, evidence field, correlated capture, model value
    PathExpression path = 3;          // operand + FieldPath
    PresentExpression present = 4;
    CompareExpression compare = 5;    // EQUAL, NOT_EQUAL, LESS_THAN, ... in one ComparisonOperator
    NotExpression not = 6;
    AllExpression all = 7;
    AnyExpression any = 8;
  }
}
```

Which references an expression may use is a property of where it appears, checked at preparation:

| Context | References admitted |
| --- | --- |
| instruction input and guard | slot, instruction outcome, run, environment |
| Contract transition predicate | observation, run event field, capture |
| correlated rule condition, trigger and response | evidence field, correlated capture, model value |
| evidence lift guard | path over the projected value |

The type-level separation between contexts becomes an admission check with a located error. Every
operator is defined once, so a new operator or reference is added in one place.

### Run Event payloads

`RunEvent` carries its kind-specific data in one `oneof payload` (instruction outcome, fault
injected, diagnostic reference) instead of per-kind optional fields. Contract expressions reach
payload fields through a `FieldPath` from a `run_event` reference, so `RunEventField` keeps only the
coordinates every event has (sequence, elapsed milliseconds, kind, entrypoint, activation,
instruction, attempt, source, run) and loses `FAULT_ROLE_ID` and `FAULT_KIND`.

### Defaults and derived fields

- **Order by default.** An instruction depends on the previous instruction of its entrypoint and
  runs only when every dependency succeeded. `after:` names any other dependency set within the
  entrypoint (a second root, or an instruction that is not its predecessor), and an explicit guard is
  written only for any other condition.
- **Derived declarations.** The environment binding list is the set of bindings roles and
  expressions reference; activation reservations follow from the instructions that start
  activations; an instruction's outcome fields follow from its kind. None is written in a Case.
- **Omitted defaults.** Instruction limits equal to the Profile's defaults are omitted.

### Resource ceilings in the Profile

A Case declares only bounds that carry meaning: instruction timeout and attempts, Contract deadlines,
and correlated windows. Node, edge, byte, work, capture and depth ceilings move to the Profile, which
admission already checks every Case against. This amends SEM-16, which today makes the Case
authoritative for all its bounds.

### Readable provenance and identity

- **Structured provenance.** `CaseProvenance` holds typed rows: Definition IDs with fingerprints and
  sources, and Known Gaps; fn-85 R8 adds its abstraction-claim row to this structure. The runtime
  still reads none of it. The glossary's "generic opaque provenance" is amended.
- **Case-local identifiers.** Program and Contract refer to states, actions, outcomes, facts, fields
  and rules by short Case-local names (`operation-identity`, not
  `temporal.nexus.success.typed-nexus.evidence.operation-identity`); provenance maps each local name
  to its Definition ID.
- **Short model values.** A model value is its declared spelling. A parameterized value's canonical
  encoding moves into provenance as a fingerprint, so a Case carries `completed`, not an
  11,000-character key.

### Presentation

- **Declaration order.** `Testpilot.ProtoJSON` emits fields in declaration order, and each message
  declares its identity first, then what it does, then when it runs.
- **Readable paths and enums.** A field path is a string in a documented grammar
  (`attributes<nexus_operation_completed_event_attributes>.scheduled_event_id`, `history.events[*]`),
  parsed at preparation; enum literals carry the value name.
- **Absent operands.** A comparison with an absent operand is false, so Producers stop emitting
  presence checks beside every comparison on the same path. This is a semantic change and applies to
  every expression context.

### Case closure

The Case's import closure excludes `Run`, `Verdict` and the Run-only messages: Run Event kinds and
filters move to a file both `contract.proto` and `run.proto` import.

### Renames

Names follow the glossary, and a word means one thing:

| Today | After | Reason |
| --- | --- | --- |
| `RunStatus` | `RunDisposition` | glossary Run disposition |
| `CorrelatedContract.clauses`, `CorrelatedRule.clause_id` | `rules`, `rule_id` | glossary Correlated Rule |
| `ResponseProjection`, `ProjectionTarget`, `ProjectionKind` | `ResponseRead`, `ReadTarget`, `ReadCardinality` | Projection is `Umpire.Case.Projection` |
| `OpaqueCapabilityType`, `capability_slot_id` | `OpaqueHandleType`, `handle_slot_id` | Capability is `Umpire.Capability`; the runtime concepts say effect handle |
| `ContractStateStatus.NONTERMINAL`, `RuleVerdictStatus.PENDING` | `PENDING` in both | one word for an unfinished rule |
| `CorrelatedValue` | `ModelValue` | the same word as `Umpire.ModelValue` |
| `InvokeRPC` | `InvokeRpc` | one spelling in generated Go |
| `Value` oneof fields | one convention, every arm suffixed `_value` | consistent, and avoids language keywords |
| `*Definition` declarations | no suffix; the graph node is `InstructionNode` and its body stays `Instruction` | one convention; the Opcode is the `Instruction` oneof case |
| `source` as a path operand | `operand`; the evidence source name becomes `evidence_source`; `RunEvent.source_id` keeps its meaning | one meaning per word |
| `PROTOCOL_NON_SUCCESS` | `PROTOCOL_FAILURE` | parallel to `SDK_FAILURE` |

Names the approved rules cite stay: `rule_events` and `elapsed_milliseconds` (EVD-21),
`RUN_EVENT_KIND_FAULT_INJECTED` (EVD-20), and the facade sequence (MOD-12). `RespondNexus`,
`NexusResponseKind` and `StartNexusOperation` are not renamed: fn-85 R10 replaces them and fn-86 R3
removes them with the last Producer that emits them, retiring their names in the vocabulary gate
then; this spec documents and renumbers them like every other message.

## API Contracts
<!-- scope: technical -->

The protocol package, Go package and `Case` root stay: `temporal.server.api.testpilot.v1`, rooted at
`case.proto`, generated into `go.temporal.io/server/api/testpilot/v1` and `Testpilot.Protocol`.
`FormatVersion` stays `{ major: 1 }`: the wire has no compatibility promise and no Case outside the
repository exists.

`Expression` is shown under Architecture. Message and field names in these sketches are the
contract, except where a generated language reserves a name (`not` in Lean), in which case the task
picks the nearest unreserved spelling and records it. The references `Expression` admits:

```proto
message Reference {
  oneof reference {
    string slot_id = 1;
    InstructionOutcomeReference outcome = 2;
    RunReference run = 3;
    string environment_binding_id = 4;
    string observation_id = 5;
    RunEventReference run_event = 6;
    string capture_id = 7;
    string evidence_field_id = 8;
    CorrelatedCaptureReference correlated_capture = 9;
    ModelValue model_value = 10;
  }
}
```

`ContractDeadline` after R5:

```proto
message Deadline {
  string violation_state_id = 1;
  oneof bound {
    int64 rule_events = 2;
    int64 elapsed_milliseconds = 3;
  }
}
```

## Edge Cases & Constraints
<!-- scope: technical -->

- **No Verdict change.** Every migrated Case, conformance corpus entry and expected Verdict reaches
  the same Verdict on the same Run; a Go test decodes each pre-migration fixture through a snapshot
  of today's descriptors, maps it to the new protocol under an explicit mapping (renames, derived
  fields, defaults, local identifiers, moved ceilings), and compares it with the regenerated
  fixture. A difference the mapping does not declare fails.
- **Absent operands.** Changing comparisons on absent operands to false is checked against every
  conformance class and live test; any Verdict that moves is a finding to explain before the change
  lands.
- **Ceilings.** A Case whose former ceiling was tighter than the Profile's keeps that bound only if
  it carried meaning (a timeout, attempts, a deadline, a window); otherwise the Profile's ceiling
  applies and the receipt lists the loosened bound.
- **Context checks replace types.** An expression reference outside its admitted context rejects at
  preparation with the existing static-preparation error category and the offending path; the
  conformance corpus gains one such rejection.
- **Presence.** A `oneof` with one arm becomes a proto3 `optional` field; an empty marker message
  stays where it selects a oneof arm (`RunReference`, the presence path selector).
- **Fixtures** are regenerated only through their generators (ART-11, ART-12); every fixture's diff
  is listed in the receipt.
- **Scalar kinds.** Folding the wire-encoding scalar kinds happens only if descriptor admission does
  not need them to match a field's declared type; otherwise the finding is recorded as kept, with
  the reason.
- **Concurrent protocol work.** fn-84 .2 moves the Driver contract into a leaf package and fn-84 .5
  changes projection lowering; this spec starts after fn-84. fn-85's Nexus instruction additions
  should land on the new shapes.
- **Umpire and Temporal independence.** Renames in `Testpilot.Protocol` reach `Testpilot.Authoring`,
  `Umpire.Case.Compiler`, `Umpire.Case.Producer` and `Temporal.Case`; `lint-model` stays green under
  MOD-01, SCP-02 and the retired-vocabulary gate, which gains the retired compound names.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Every rename in the Renames table is applied across the protocol, `Testpilot.Protocol`,
  `Testpilot.Authoring`, the Lean Producers, the Go runtime and tests, and the fixtures; each retired
  compound name is added to the retired-vocabulary gate. Errors: a retired compound name anywhere in
  the scanned trees fails `make umpire-check-retired-vocabulary` naming the file.
- **R2:** The protocol files follow the Target structure table; every message and every field whose
  meaning is not its name carries a leading comment; field numbers are dense from 1 within each
  message. Errors: a message without a leading comment fails a Go check over the descriptor set
  naming the message.
- **R3:** `Expression` and `Reference` replace `ProgramExpression`, `ContractExpression`,
  `CorrelatedPredicate`, `CorrelatedComparison`, `CorrelatedOperand`, `CorrelatedCorrelation` and
  `CorrelatedCorrelationGroup`; `ComparisonOperator` includes equal and not-equal; evidence lift
  guards are expressions and `guard_equals_text` is gone; preparation checks each expression's
  references against its context. Errors: a reference outside its context rejects at preparation
  with the static-preparation category and the expression's path, pinned by a conformance
  rejection case and Lean and Go unit tests.
- **R4:** `RunEvent` carries kind-specific data in one payload oneof; `RunEventField` holds only the
  common coordinates; Contract expressions read payload fields through a path. Errors: a Run Event
  whose payload does not match its kind is a Driver invariant diagnostic, and a Contract path into a
  payload its event kind cannot carry rejects at preparation.
- **R5:** Single-arm presence oneofs are proto3 `optional` fields; the correlated `present`
  constraint is a presence marker; `Deadline` holds `violation_state_id` and a `bound` oneof over
  `rule_events` and `elapsed_milliseconds`. Errors: a deadline with no bound rejects at preparation,
  replacing today's "exactly one positive" check.
- **R6:** One opaque-handle encoding remains; `Value.natural` is removed unless a Producer needs a
  value no other arm represents; the capture type is `SingularType` restricted at preparation;
  one named-value message replaces the three binding shapes; `CorrelatedContract.version` and the
  wire `EntrypointKind` are removed (the Go runtime keeps its own classification); the wire-encoding
  scalar kinds are folded or recorded as kept with the reason. Errors: a capture of an admitted type
  outside scalar, enum or message rejects at preparation.
- **R7:** `common/testing/testpilot/README.md` gains an extension section listing, for a new
  instruction, fault kind, Run Event payload and expression reference, every place that must change
  (protocol, generated Lean, `Testpilot.Authoring`, Go interpreter or evaluator, Profile Opcode,
  conformance class, retired-vocabulary gate), with the worker-stop fault kind traced through every
  place as the worked example; fn-85 R10 is its first use. Errors: no error surface.
- **R8:** Every fixture and conformance corpus entry regenerates through its generator; the
  field-mapping equivalence test passes for every pre-migration fixture; `make
  umpire-check-testpilot-protocol`, `make umpire-check-case-runtime-conformance`,
  `make umpire-check-live-tests`, `make lint-model` and `make umpire-check-regression` pass. Errors:
  a fixture whose migrated form differs beyond the declared mapping fails the equivalence test naming
  the fixture and the field.
- **R9:** Instructions run in entrypoint order by default and only when their dependencies
  succeeded; `after:` names any other dependency set within the same entrypoint, including none;
  explicit guards remain only for other conditions. Errors: an `after:` naming an unknown instruction,
  itself, an instruction on another entrypoint, or forming a cycle rejects at preparation.
- **R10:** The environment binding list, activation reservations and instruction outcome fields are
  derived at preparation and no longer written in a Case; instruction limits equal to the Profile
  defaults are omitted. Errors: a Case that still writes a derived field rejects at preparation
  naming it.
- **R11:** A Case's import closure excludes `Run`, `Verdict`, diagnostics and Run Event payloads,
  checked by a Go test over the descriptor set. Errors: an import that pulls a Run-only message into
  the Case closure fails that test naming the file.
- **R12:** Node, edge, byte, work, capture and depth ceilings live in the Profile; a Case declares
  only instruction timeouts and attempts, deadlines and correlated windows; an SEM-16 amendment is
  drafted under GOV-02. Errors: a Case bound outside the Profile's ceiling rejects at preparation as
  today.
- **R13:** `CaseProvenance` is structured (Definition IDs with fingerprints and sources, Known Gaps;
  fn-85 R8 adds abstraction claims) and readable in fixture diffs; the glossary's Case and Provenance entries are
  amended under GOV-02. Errors: no error surface; the runtime does not read provenance.
- **R14:** Program and Contract use Case-local names, and provenance maps each to its Definition ID;
  a model value is its declared spelling, with a parameterized value's canonical encoding recorded
  as a fingerprint in provenance; `typed-nexus-case.json` shrinks by at least its 244 KB of encoded
  values. Errors: a Case-local name used twice for different Definition IDs rejects at production
  naming both.
- **R15:** `Testpilot.ProtoJSON` emits fields in declaration order with identity first; field paths
  are strings in a documented grammar parsed at preparation; enum literals carry names; a comparison
  with an absent operand is false and Producers emit no presence check beside a comparison on the
  same path. Errors: a path string outside the grammar and an enum name the descriptor does not
  declare reject at preparation with the offending text.

## Early proof point

Land R1 and R2 first, purely mechanical: renames, file moves, comments and renumbering, with the
equivalence test proving every fixture unchanged in meaning. Only then start R3 to R6 and R9 to R15. If the
equivalence test cannot be written against a descriptor snapshot, stop: the "no semantic change"
guarantee needs another oracle before any structural change.

Tasks fn-87-tighten-the-testpilot-protocol-glossary.1 (the equivalence harness over a frozen
descriptor snapshot) through .4 (restructure with no new mapping step) are that proof point. If .1
cannot decode the baseline through the snapshot, or .4 needs a mapping step, stop and re-evaluate
before .5.

## Quick commands

```bash
# Equivalence oracle (every task)
go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/
# Lean protocol, authoring and fixtures
make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance
# Vocabulary gate
make umpire-check-retired-vocabulary
# Full gate (live tests need these)
CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression
go clean -cache && make lint-code GOLANGCI_LINT_FIX=false   # baseline 161
make lint-model                                              # baseline 163
```

## Planning decisions

Decided while breaking the spec into tasks (2026-09-12), from the repository and gap scans. Each
narrows or completes a requirement without changing its intent; tasks record the final choice.

- **Renames skip doomed names.** R1 does not rename declarations a later requirement deletes
  (the `Program*`/`Contract*` expression messages, the `*Ref` messages `Reference` replaces, and the
  reservation, environment and outcome declarations R10 derives); their names are retired when they
  are deleted. `InstructionRef` becomes `InstructionReference`, `ContractDeadline` becomes `Deadline`.
- **Go initialisms.** Hand-written Go keeps `InvokeRPC` for the Opcode and Driver method (staticcheck
  ST1003), as it keeps `CaseID` beside generated `GetCaseId`; the proto message, generated Go and Lean
  spell `InvokeRpc`, and the gate retires the Lean and JSON spelling `invokeRPC`.
- **Two more `Reference` arms.** A correlated condition is an existential over the admitted step's
  action, outcome, state or facts, which a literal `model_value` cannot express, and an evidence-lift
  guard needs the projected value as its operand. `Reference` gains a correlated step reference and a
  projected-value marker, each admitted in one context.
- **Context rejection category.** A reference outside its context is a `PreparationError` of category
  `unknown` with a located path, and the `static-preparation-rejection` class carries a second Case
  (EVD-18 keeps six classes).
- **Run Event payload arms.** The arms are the instruction outcome and the injected fault; diagnostic
  events keep an outcome. A payload that does not match its kind is an `INVARIANT` Run diagnostic
  raised when the event is recorded.
- **Named values.** The evidence side uses one `{field_id, Value}` message and the lift side the same
  shape over an `Expression`, replacing three unrelated shapes.
- **Scalar kinds and naturals.** Wire-encoding scalar kinds are kept if admission needs a slot's kind to
  equal the field's kind (it does today); `natural_value` is removed in favor of `unsigned_integer_value`
  unless a Producer needs a larger value.
- **Ceilings.** Every bound except instruction timeout and attempts, deadlines and correlated windows
  moves to the Profile, durations included, unless moving one changes a Verdict, in which case it stays
  as a behavior bound. A Temporal default ceiling set feeds derived and test Profiles; the Profile also
  gains instruction defaults.
- **`after` scope (R9 amended 2026-09-12).** R9 first said `after:` names cross-entrypoint
  dependencies. The runtime has none: each entrypoint is an activation-local acyclic graph, preparation
  rejects a dependency on another entrypoint's instruction ("missing, duplicate or cross-entrypoint
  dependency"), no checked-in Case uses one, and entrypoints coordinate only through Temporal and the
  instructions that wait on it (`AwaitInstruction`, awaited Slots). Supporting one would be a new
  scheduling capability, which Boundaries exclude. What the Cases do need is a non-default dependency
  set within one entrypoint (the typed Nexus Case has a second root and two diamonds), so R9 now says
  that, and a cross-entrypoint entry rejects as unsupported.
  Cross-entrypoint waiting stays with `AwaitInstruction`. Dependents that run regardless of success
  carry an explicit `true` guard.
- **Derived-field errors.** Removed fields fail strict ProtoJSON decode naming the field, which is where
  a Case that still writes one is rejected.
- **Additional rule drafts.** ART-13 (a declared binding graph) and ART-09 (opaque provenance bytes)
  contradict R10 and R13, so restatements of both are drafted under GOV-02 beside the SEM-16 and
  glossary drafts; approved text is not edited.
- **Local names and values.** One Case-wide namespace of shortest unique dotted suffixes, rule ids
  included; value spellings get a fingerprint-derived disambiguator only where two encodings of one
  definition share a spelling.
- **Absent operands.** Every comparison operator, `NOT_EQUAL` included, is false on an absent operand;
  bare absent predicates and absent inputs still reject. The rule is run over unchanged fixtures before
  Producers drop presence checks.
- **Cross-file enum accessors (decided in .4, 2026-09-12).** `protogen` trims an enum value's type
  prefix only inside the file that declares the enum, so a message whose singular enum field names an
  enum from another file of the package generated Go that did not compile. That is why `FaultInjected`
  sat beside `FaultKind`. `RunEvent.kind` must reach `RunEventKind` in `event.proto` while R11 keeps
  `RunEvent` out of the Case closure, so no file layout satisfies both. `cmd/tools/protogen` now
  rewrites such references after generation, which changes no other generated file. `Testpilot.Protocol`
  loads the Case and Run closures with one `protoc` call, because a second `#load_proto_file`
  declares the shared files twice.
- **One Expression (decided in .5, 2026-09-12).** `Expression.not` keeps the sketch's name under an
  api-linter `core::0140::reserved-words` suppression; the Lean Authoring constructor is `Expr.negate`.
  `ModelValue` moves to `value.proto` and `CorrelatedCaptureRef` becomes `CorrelatedCaptureReference`
  in `expression.proto` now, so .6's `correlated.proto` can import `expression.proto` without a cycle.
  A capture assignment names its Observation by `observation_id`. `ir` defines only the Program and
  Contract contexts; .6 adds the correlated and evidence-lift contexts with the references they admit.
  A context rejection is `unknown` at a located path; other expression errors keep their coarse paths.
  Logical depth, IR node counts, runtime work limits and Contract per-event work bounds are unchanged
  for every checked-in Case; surface and admission work grow by one message per outcome, Run and Run
  Event reference and one enum per equality, far below the admission ceilings. The expression-context
  rejection is a variant beneath `static-preparation-rejection`, and the equivalence oracle declares
  new fixtures in `Added`.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Glossary renames and retired names | .2, .3 | — |
| R2 | One file per concept, comments, dense numbers | .4 | — |
| R3 | One Expression with context checks | .5, .6 | — |
| R4 | Run Event payload oneof | .7 | — |
| R5 | Presence as optional fields; Deadline bound oneof | .6, .8 | — |
| R6 | Duplicate and dead shapes removed | .8, .9 | — |
| R7 | Extension checklist | .17 | — |
| R8 | Generators, equivalence test, gates | .1, .17 (every task keeps it green) | — |
| R9 | Entrypoint order by default; `after` | .11 | — |
| R10 | Derived declarations; default instruction limits | .12 | — |
| R11 | Case closure excludes Run-only messages | .4, .7 | — |
| R12 | Ceilings in the Profile; SEM-16 draft | .10 | — |
| R13 | Structured provenance; glossary drafts | .13 | — |
| R14 | Case-local names and short values | .14 | — |
| R15 | Declaration order, string paths, named enums, absent operands | .15, .16 | — |

## Boundaries
<!-- scope: business -->

- **No new capabilities.** These protocol gaps stay with their owners:

  | Gap | Owner |
  | --- | --- |
  | worker instructions carrying Temporal API messages (Nexus timeouts, operation-failed reply, handler error type and retry behavior, completion outcome) and one observation declaration per Case | fn-85 R10, authored on the new shapes |
  | Nexus cancel request and handler cancel reply | fn-79 |
  | correlated transitions over structured machine state (a record of fields per entity instance, not one `state` value and a fact list) | fn-85, whose machines need it |
  | a wait-for-duration instruction, if fn-85's timer realization needs one | fn-85 |
  | signals, updates, queries, child workflows, continue-as-new, activity scheduling and an activity interpreter, HTTP invocation for external Nexus callers | the first Model that needs each |
  | fault kinds beyond worker stop and resume | the first Model that needs each |

- **No change to Verdict computation** beyond R15's absent-operand rule, which is checked against
  every conformance class and live test.
- **No versioning scheme.** `FormatVersion` stays `1.0`; no compatibility shim for old Cases.
- **No edits to historical `.plans` documents** other than `UMPIRE4_ORDER.md` and the drafted
  SEM-16 and glossary amendments in R12 and R13 (and the ART-09 and ART-13 restatements the Planning
  decisions add); approved rule text keeps the names it cites.
- **Depends on fn-84**, whose Driver contract and projection lowering tasks touch the same code.

## Decision Context
<!-- scope: both — conditionally substructured -->

The protocol changes now because it has no compatibility promise and two specs are about to build on
it: fn-85 adds instructions and fn-86 migrates every hand-written Case. Cleaning afterwards would
migrate those additions a second time.

- **One expression language with context checks** over three typed ones: every operator is defined
  once and a new reference is added in one place; the cost is that context mismatches become
  preparation errors instead of unrepresentable values, which the existing located static
  preparation errors already handle.
- **Payload oneof on Run Events** over per-kind fields: the fault fields were the first per-kind
  addition and every later one would repeat them.
- **Rename to glossary words** over documenting the differences: SEM-19 forbids a word naming two
  concepts and a concept having two words.
- **Mechanical changes first**, proven by a field-mapping equivalence test, so structural
  simplifications are reviewed against an unchanged baseline.

- **Defaults over repetition**: every instruction guard in the checked-in Cases is "every
  dependency succeeded", and no limit field takes more than four values across the six Cases, so the
  wire states the exception, not the rule.
- **Resource ceilings in the Profile** (amending SEM-16): they bound the environment, not the
  behavior, and admission already checks Cases against the Profile.
- **Readable fixtures**: the user reviews fixtures as artifacts; local names, short values,
  declaration order, string paths and named enums make a Case readable without changing what the
  runtime checks.

Rejected: cross-entrypoint `after:` dependencies (entrypoints run as independent activation graphs that
coordinate through Temporal; see the amended R9 in Planning decisions); a `v2` package beside `v1` (no consumer needs both, and the generator and runtime would
carry two protocols); splitting the correlated capability into its own package (it is part of one
Contract); a generated prose summary beside each fixture instead of a readable fixture (it would be a
second artifact to review and keep in step).



