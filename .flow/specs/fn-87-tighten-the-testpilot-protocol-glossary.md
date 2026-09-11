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

This spec makes the protocol consistent with the glossary, structured by concept, and smaller, and
documents how it is extended. It changes no runtime semantics: every migrated Case means exactly
what the fixture it replaces meant.

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
| 11 | `RespondNexus` beside `StartNexusOperation` and `CompleteNexusOperation`; `NexusResponseKind` answers an operation, not "Nexus" | naming | R1 |
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
| `RespondNexus`, `NexusResponseKind` | `RespondNexusOperation`, `NexusOperationResponseKind` | matches the other Nexus operation instructions |
| `Value` oneof fields | one convention, every arm suffixed `_value` | consistent, and avoids language keywords |
| `*Definition` declarations | no suffix; the graph node is `InstructionNode` and its body stays `Instruction` | one convention; the Opcode is the `Instruction` oneof case |
| `source` as a path operand | `operand`; the evidence source name becomes `evidence_source`; `RunEvent.source_id` keeps its meaning | one meaning per word |
| `PROTOCOL_NON_SUCCESS` | `PROTOCOL_FAILURE` | parallel to `SDK_FAILURE` |

Names the approved rules cite stay: `rule_events` and `elapsed_milliseconds` (EVD-21),
`RUN_EVENT_KIND_FAULT_INJECTED` (EVD-20), and the facade sequence (MOD-12).

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

- **No semantic change.** Every migrated Case, conformance corpus entry and expected Verdict means
  what it meant; a Go test decodes each pre-migration fixture through a snapshot of today's
  descriptors, maps it field by field to the new protocol, and compares it with the regenerated
  fixture. A difference that is not a declared rename fails.
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
  conformance class, retired-vocabulary gate), with the Nexus instruction additions of fn-85 R10 as
  the worked example. Errors: no error surface.
- **R8:** Every fixture and conformance corpus entry regenerates through its generator; the
  field-mapping equivalence test passes for every pre-migration fixture; `make
  umpire-check-testpilot-protocol`, `make umpire-check-case-runtime-conformance`,
  `make umpire-check-live-tests`, `make lint-model` and `make umpire-check-regression` pass. Errors:
  a fixture whose migrated form differs beyond declared renames fails the equivalence test naming
  the fixture and the field.

## Early proof point

Land R1 and R2 first, purely mechanical: renames, file moves, comments and renumbering, with the
equivalence test proving every fixture unchanged in meaning. Only then start R3 to R6. If the
equivalence test cannot be written against a descriptor snapshot, stop: the "no semantic change"
guarantee needs another oracle before any structural change.

## Boundaries
<!-- scope: business -->

- **No new capabilities.** These protocol gaps stay with their owners:

  | Gap | Owner |
  | --- | --- |
  | Nexus operation timeouts, operation-failed reply, handler error type and retry behavior, completion outcome | fn-85 R10, authored on the new shapes |
  | Nexus cancel request and handler cancel reply | fn-79 |
  | correlated transitions over structured machine state (a record of fields per entity instance, not one `state` value and a fact list) | fn-85, whose machines need it |
  | a wait-for-duration instruction, if fn-85's timer realization needs one | fn-85 |
  | signals, updates, queries, child workflows, continue-as-new, activity scheduling and an activity interpreter, HTTP invocation for external Nexus callers | the first Model that needs each |
  | fault kinds beyond worker stop and resume | the first Model that needs each |

- **No change to runtime semantics**, Verdict computation, admission ceilings or the Profile.
- **Limits stay.** The four Limits messages and their ceilings are kept: SEM-16 makes a Case
  authoritative for its bounds.
- **No versioning scheme.** `FormatVersion` stays `1.0`; no compatibility shim for old Cases.
- **No edits to historical `.plans` documents** other than `UMPIRE4_ORDER.md`; glossary words the
  renames align with already exist, and approved rule text keeps the names it cites.
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

Rejected: a `v2` package beside `v1` (no consumer needs both, and the generator and runtime would
carry two protocols); moving resource ceilings from the Case into the Profile (SEM-16); splitting
the correlated capability into its own package (it is part of one Contract).
