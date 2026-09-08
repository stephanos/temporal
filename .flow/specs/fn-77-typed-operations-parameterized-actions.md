# fn-77-typed-operations-parameterized-actions Typed operations, parameterized Actions, and field-level Properties

> HTML render lens: `.flow/artifacts/fn-77-typed-operations-parameterized-actions/spec.html` — open locally; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

Status: implementation specification under task breakdown and plan review. Nexus operation cancellation qualification is deferred to fn-79; typed operations and field-level Properties proceed without it.

[UMPIRE4_SPEC](../../.plans/UMPIRE4_SPEC.md) remains normative. This complements
[fn-78 typed temporal authoring and checked scoped monitoring](fn-78-typed-temporal-authoring-and-checked.md):
this spec owns operation/value/field semantics; fn-78 owns temporal obligations and evidence admission.


`Nexus3/Nexus.lean` describes lifecycle Actions and
state/outcome/fact requirements. Its Producer
separately names RPCs, constructs request assignments, and compares concrete history fields.
The author cannot yet express fine-grained request/response behavior in the same checked model.

An author must be able to reference a typed API operation or SDK command, provide modeled arguments,
describe Target-owned response/error alternatives, and write Properties relating those fields to
state, other fields, and earlier captured values. Integration must execute and observe that meaning,
not invent the missing product requirements downstream.

Public API contracts are product behavior. `Temporal.Feature` may import generated `Temporal.API`
structure and interpret it; SEM-03 forbids generated structure from defining behavior by itself.
Referencing an RPC declaration is not performing an RPC or importing its handler implementation.
The runtime restrictions on raw evidence, arbitrary callbacks, and undeclared effects remain intact.


### Current foundations and gaps

- `Temporal.API` already declares typed
  `Method Request Response` references. Reuse them instead of hand-maintained RPC-name strings.
- `API/Types.lean` contains generated messages, enums, and
  oneofs. `API/Proto.lean` also uses structural `MessageRef`
  values and digest/size `Bytes` summaries. These are not complete concrete recursive/byte values.
  Importing those types alone does not establish full-fidelity field evaluation or runtime encoding.
- `Property/Language.lean` has typed literal predicates
  and separate guard/expectation contexts, but explicitly excludes cross-field operands. Extend its
  checked data and semantics; do not silently reinterpret existing literal comparisons.
- The existing Case Program and Contract have field paths, assignments, projections, and captures.
  They are a lowering target to reuse, not an additional public language for model requirements.
- The completed fn-68 success Producer and its negative fixtures are the compatibility baseline.
  The DSL experiment did not test parameterized RPC Actions, arbitrary field access, or their
  production correspondence; its results cannot substitute for this spec's acceptance evidence.

## Architecture & Data Models
<!-- scope: technical -->

### Ownership

| Owner | Responsibility |
| --- | --- |
| API generator and `Temporal.API` | Structural operation/message/field declarations, schema provenance, and supported exact value representations. |
| `Umpire` checked semantic interfaces | Parameterized Actions/outcomes, typed field expressions, finite domains, validation, canonicalization, and Property meaning. |
| `Temporal.Feature` | Product operation meaning, permitted state changes and results, and independently authored field-level requirements. |
| `Temporal.System` and Implementation Links | Implementation mechanisms, mapping observed requests/results/events into model values, and correspondence obligations. |
| Temporal Producer | Checked request construction, declared response/evidence projection, faithful field-expression lowering, and model provenance. |
| Testpilot/Driver/environment | Generic prepared execution, authorized clients/capabilities, endpoint binding, and transport. No hidden product assertions. |

Extend existing modules and the fn-75 semantic seam. Preserve AUT-07's Property/Behavior/Query
owners and fn-71's Program/Contract context separation. Do not require a new package for every
conceptual interface or duplicate schema definitions by hand.


### ACT-1 — Typed operation references and model instances

An operation declaration exposes its stable identity, typed request signature, typed return/error
signature, and interaction kind. A model Action instance binds that declaration to concrete modeled
arguments. The Target determines admitted result/error alternatives and resulting state from the
prior state and arguments; selecting an Action never selects its outcome (SEM-07).

For RPC-backed operations, reference the generated method declaration and check its method identity,
request/response schema, and supported interaction shape. Equal request types do not make two RPCs
interchangeable. Reject forged, wrong-method, incompatible-schema, and unsupported streaming bindings.
Unary RPCs are the initial executable scope; retain streaming metadata and reject unsupported use.

Model SDK commands and semantic events as distinct typed operation kinds. In the Nexus workflow
path, SDK command submission, service acknowledgement, history confirmation, and eventual resolution
are different boundaries. A history-fetch RPC is observation work and does not cause completion.
A standalone Nexus operation RPC must not replace a workflow SDK command merely because names or
payload fields resemble each other. Any modeled correspondence between them requires an explicit
Implementation Link.

An operation declaration defines structural connectivity, not product semantics. Authors still
declare valid/invalid modeled inputs, state-dependent transitions, and requirements independently.
Do not derive every Property from the transition being checked. Protocol-valid but product-invalid
inputs remain modelable when testing their expected rejection; type admission must not erase them.
Represent transport failure, application rejection, successful return, pending work, and later
semantic resolution distinctly where the operation needs them. A transport timeout is not evidence
that the operation failed or did not execute, and a successful call is not proof of eventual success.

### ACT-2 — Schema-aware field fidelity

Expose generated, typed field references for the selected operation's complete declared schema.
Distinguish access to structural metadata from access to concrete field values. The supported
executable fragment must preserve the exact values and presence distinctions observed by the
requested Property; unsupported access rejects with the responsible field/clause and reason.

At minimum, implement nested message access, scalar values, explicit presence, oneof selection,
enum values, bounded repeated-field selection/cardinality, and keyed map lookup for the qualifying
examples. Presence follows the actual descriptor: do not invent absence for an implicit-presence
scalar or collapse an absent optional/message value into a present default. Preserve repeated order,
oneof discriminants, numeric enum identity, and integer signedness/range. Map normalization must
follow decoded Protobuf map semantics and serialize deterministically, not depend on list order.

Byte comparisons require concrete bytes, not equality of digest/size summaries. Recursive field
access requires an exact admitted value representation with a declared depth/work bound; exhausted
depth is unsupported/incomplete, never a fabricated empty subtree. Provide these capabilities through
the existing generator or a checked structural-value layer; keep descriptive metadata useful to its
existing consumers. Never hand-edit generated files or create hand-maintained copies of API schemas.

“Fidelity” here means the schema-defined value semantics needed by the Property, not preservation
of incidental wire encoding order. Report support for unknown enum values, recursive values, bytes,
floating-point values, and other special forms explicitly. If a requested operation is unsupported,
reject it rather than substitute text, hashes, silent truncation, or approximate equality. Floating-
point comparisons, if supported, must specify NaN and signed-zero behavior consistently across Lean
and Go; this spec does not require unused numeric operators to be implemented.

### ACT-5 — Finite exploration without false full-domain claims

Full schema access does not imply exhaustive exploration of every possible string, byte sequence,
list, or recursive request. Authors declare finite input domains, bounds, or a named abstraction
for each modeled variation. Do not silently choose “typical” values or restrict a domain in a binding.
Extend the existing proof-carrying finite adapters to parameterized instances rather than replacing
the authoritative Target or weakening domain soundness/completeness.

Results record exactly which field values/domains were explored and which dimensions were fixed,
abstracted, or outside the claim. Representative cases prove only results over those cases. Any
stronger abstraction claim needs explicit transition and Property preservation evidence; replay of
one concrete witness cannot prove an abstract universal claim or validate excluded behaviors.

Runtime monitoring may compare concrete values outside an exploration sample only when the declared
runtime scope and checked lowering admit them. That does not extend the finite query's verification
claim. Otherwise report unsupported/out-of-scope evidence explicitly. Keep semantic domain bounds
separate from search work and runtime resource ceilings.


### Delivery and dependencies

| Stage | Deliver | Order |
| --- | --- | --- |
| O1 — Operation and value contracts | Generated operation references, separate command/event kinds, exact supported value/field access, and explicit unsupported cases. | Uses the delivered fn-75 checked Target seam. |
| O2 — Parameterized semantics | Action/outcome instances, field operands/captures, finite domains, canonical meaning, and independent Properties. | After O1; extends the delivered fn-78 scoped semantics with typed operands. |
| O3 — Checked concrete lowering | Request assignments, declared result/evidence projections, coverage maps, field-expression correspondence, and generic portable capability. | After O2; consumes fn-71/fn-72 and fn-78's generic temporal compiler. |
| O4 — Field-level qualification | Both authored examples, mutations, identity/compatibility evidence, and real Driver qualification. | After O3; composes field and scoped temporal correspondence. |

The generic fn-78 delivery is complete; its cancellation work was transferred to deferred fn-79.
Reuse its projection, obligation, and portable Contract semantics. Parameterized variants add typed
operands and captures without duplicating those owners. Delivered fn-68/71/72/73/74/75/78 are
compatibility inputs, not new outstanding spec dependencies. Serialize overlapping source and Lean
verification work with fn-76 without treating that scheduling constraint as a feature dependency.

This is not a prerequisite for fn-73 or the first fn-70 canary. Reuse environment binding and Driver
work from their owners; do not expand this spec into new transport, activation, or deployment policy.

## API Contracts
<!-- scope: technical -->

### ACT-3 — Typed field-level Properties

Extend the existing closed Property expression vocabulary with typed operands referencing admitted
request fields, outcome fields, prior/resulting model state, semantic event fields, and explicitly
captured earlier model values. Support presence, equality/inequality between compatible operands,
declared scalar ordering, oneof tests, and Boolean composition needed by the qualifying examples.
Preserve the existing literal-only predicate interpretation for existing declarations.

Make value availability explicit. Preconditions may inspect prior state and requested arguments;
result requirements may inspect the same immutable request together with the modeled result and
resulting state. Cross-step comparisons bind a named occurrence and capture typed values under an
explicit operation key. Reject future references, unbound captures, wrong occurrence/key selection,
incompatible operand types, and field access through an unselected oneof or unestablished presence.
Absence may be tested explicitly; missing evidence cannot otherwise make a comparison true.

Properties may express selected field values, relationships between request and response, and
request/state or cross-occurrence equality. Unmentioned fields are unconstrained by that Property;
do not infer an exhaustive field contract from a partial conjunction. Transition/frame constraints
belong to the authoritative model and must say which state is preserved where it matters.

Authoring remains ordinary typed Lean or focused syntax elaborating to checked, serializable data.
Do not install arbitrary Lean closures into Cases or introduce a second evaluator for a convenient
syntax form. Field-level temporal requirements reuse the scoped key/clock/closure rules of the DSL
evolution spec; this spec adds operands and captures, not another temporal language.

### ACT-4 — Requirements remain in the model

Keep product assertions in `Temporal.Feature` Properties and implementation-specific assertions
in the appropriate `Temporal.System` model. Integration defines how concrete data establishes those
modeled values. Every runtime product assertion must trace to a checked Property/clause; every
additional projection/correlation condition must be identified as an Implementation Link obligation.

For example, a declared requirement that a command addresses a particular modeled operation
belongs in the model. Matching a history event's scheduled-event reference to the bound operation
is part of establishing its concrete evidence. The latter cannot silently become a new requirement
about fields the author never declared. Environment assignment of a namespace or task queue is a
binding, while any semantic relationship involving it must be declared and preserved under binding.

Require a coverage map for the selected Case: modeled input fields to Program construction,
modeled result/event fields to declared Observations, and each requested clause to its exact
lowering. Missing mappings or unsupported clauses reject the whole requested Case before Driver I/O.
The runtime Contract still reads declared Observations and Run Events, not private Slots or arbitrary
raw API objects. Capture only admitted data under explicit work/size/capture limits.

### ACT-6 — Identity, compatibility, and execution

Bind operation references and field expressions to checked method/command identities and schema
provenance. Field identity must survive source-order changes and avoid dependence on generated Lean
spelling alone. Operation templates have stable Definition IDs; parameter values belong to canonical
Action instances and trace/Case identity. Behavior-affecting domains, field relationships, and
abstractions participate in Behavior Fingerprints.

Preserve unchanged declarations, IDs, canonical bytes, and existing fixtures where semantics are
unchanged. Version new encoded expressions/values and reject unknown meaning-bearing fields or
stale schemas. An added unreferenced schema field must not silently acquire product meaning; any
compatibility allowance must be checked. Named migrations own intentional semantic/format changes.

Prove expression evaluation and request/result lowering preserve the declared field meanings.
Generated structural metadata is not a proof of behavioral fidelity. Use the existing generic
Testpilot codec, Prepare/Run facade, and shared Driver; scope any necessary generic protocol extension
to unsupported value/expression capability demonstrated by this spec. Keep endpoint/credential and
symbolic environment binding outside the model. Known Gaps disclose support limits and cannot waive
a requested field requirement.

## Edge Cases & Constraints
<!-- scope: technical -->

Run focused Lean semantic/elaboration tests and transitive axiom audits. Change the owning API
generator when generated declarations change, regenerate its complete owned surface, and run its
fixture/staleness and downstream checks. Use independent field-value fixtures against the actual
Go codec, admission, and evaluator; a shared renderer must not serve as its own correctness oracle.
Run `make umpire-build-model`, `make lint-model`, `make lint-code`, and affected generation/regression
gates discovered from the current Makefile. Go tests include `-tags test_dep`, with `integration`
only for integration tests. Preserve existing comments and public compatibility where applicable.

Do not add third-party libraries, change the Lean toolchain, or introduce custom axioms,
placeholders, or additional compiler-trust dependencies. The implementation is bounded by declared
message sizes, nesting, collection sizes, live captures, and work budgets. Exercise tenfold request
variation and payload/collection loads to verify bounded rejection and Run isolation; growing field
domains can multiply finite search space, and this spec promises no symbolic scalability.

No implicit retries or recovery service are introduced. A crash or transport interruption supplies
neither a successful response nor evidence of nonexecution; existing Run closure and evidence
policies remain authoritative. Credentials and raw sensitive data are not copied into model
definitions or receipts merely because a request type exposes a field.

## Acceptance Criteria
<!-- scope: both -->

### Qualifying examples

Use two distinct examples rather than disguising an SDK command as an RPC:

1. A unary WorkflowService call already used by a Producer, such as `StartWorkflowExecution`,
   referenced by its generated declaration. Supply typed modeled request arguments and explicit
   finite domains; author a nested-field constraint and a request/state or request/result relation.
   Choose actual product requirements from the model's use case rather than asserting a fictitious
   response field or API guarantee. Include a concrete byte-bearing or optional field fixture to
   test the fidelity boundary independently of the live call.
2. The existing workflow-owned Nexus start/completion command/event path. Relate the selected
   operation identity to its modeled scheduled-operation identity, retain typed completion fields,
   and compose a cross-occurrence requirement with the scoped bounded-response Property. Exercise
   two operations so mismatched identities cannot pass accidentally. Cancellation command,
   confirmation, and resolution qualification are deferred to [fn-79](fn-79-deferred-nexus-operation-cancellation.md).


- **R1:** An author references a generated RPC declaration directly in a checked parameterized Action, with no copied method string/schema in the ordinary model. Distinct SDK-command and event declarations work without pretending to be unary RPCs. Wrong method/request pairing and unsupported streaming reject. Errors: Wrong method/request pairing, forged identities, incompatible schemas, and unsupported streaming reject under ACT-1.

- **R2:** Generator/value-layer fixtures preserve nested values, supported presence, oneofs, unknown enum policy, repeated order, map lookup, integer boundaries, and concrete bytes. An explicit default differs from absence only where the descriptor permits it. Unsupported recursive depth, value forms, or operators reject with a source-owned diagnostic. Errors: Unsupported value forms, recursion bounds, and operators reject under ACT-2; presence follows the descriptor.

- **R3:** Independent Properties compare request/state, request/result, and captured cross-step fields through the same checked expression semantics. Negative tests cover unbound/future/wrong-operation references, type mismatches, missing presence, wrong oneof branch, and differing byte values with equal length. Errors: Invalid availability, types, presence, oneofs, and occurrence keys reject under ACT-3; missing evidence never satisfies a comparison.

- **R4:** Mutating a selected request or response field causes the appropriate declared clause to fail or evidence admission to reject. Corrupting correlation fails the link rather than inventing a product violation; removing a required projection cannot yield satisfaction. Field omissions remain visible in the coverage map. Errors: Missing mappings reject before I/O; field mutations violate the declared clause or reject evidence, and correlation failures remain link failures.

- **R5:** Finite domains enumerate exactly their declared parameterized Action instances and preserve Target-owned alternatives. A representative sample never produces a full-schema exhaustive claim. Runtime inputs outside the sample follow the separately declared runtime scope; incomplete search and unsupported abstractions remain distinct. Errors: Out-of-scope runtime values and unsupported abstractions are reported explicitly; incomplete search never yields an exhaustive claim.

- **R6:** Typed request construction and field/capture expressions lower to an actual admitted Case, with checked correspondence and Lean/Go tests using independently derived expected values. Model evaluation, online monitoring, and offline replay agree for positive, violated, incomplete, and malformed evidence cases. Errors: Malformed or incomplete evidence retains its declared admission/verdict meaning across all three evaluators.

- **R7:** The real shared Driver runs the unary example and the Nexus command/event example under the existing public facade; no RPC substitution, model-specific runtime branch, endpoint rewriting, or callback-based Property interpretation is added. Preserve the completed Nexus3 success path and its fixture/rejection tests. Errors: No error surface beyond R1–R6 and the existing Driver rejection/closure contract.

- **R8:** A property edit changes one semantic declaration and its derived Contract, without handwritten duplicate assertions. Schema/field changes trigger deterministic regeneration and compatibility checks. Equivalent surface spellings preserve meaning/identity; semantic parameter/domain changes affect the appropriate fingerprints/artifacts. Errors: Stale schema and unknown meaning-bearing encoded fields reject; intentional incompatible changes require a named migration.

- **R9:** Changed theorem axiom inventories preserve the approved baseline; focused generator, semantic, negative elaboration, finite planning, cross-language, and functional checks pass, along with affected build/lint/staleness gates. Record actual supported forms and remaining Known Gaps. Errors: Failed gates or unapproved axioms prevent qualification; Known Gaps cannot waive requested clauses.

## Boundaries
<!-- scope: business -->

General streaming execution, all-RPC semantic coverage, a full Protobuf implementation rewrite,
arbitrary quantified field logic, unrestricted callbacks, full LTL, and Veil adoption are outside
scope. The selected examples must nevertheless exercise real typed field semantics end to end;
renaming string paths or moving handwritten assertions into a new adapter does not satisfy this spec.

## Decision Context
<!-- scope: both -->

Generated API declarations provide structural fidelity; independently authored model Properties define behavior. This lets developers describe field relationships directly while integration supplies checked construction, observation, and correspondence. Finite domains keep exploration claims precise without limiting schema access to summary values. Reuse the existing checked Property language and shared Driver instead of adding a second evaluator or handwritten API schemas.

Concrete integration consumes the delivered fn-71/fn-72 interfaces; temporal qualification consumes
fn-78's delivered generic monitoring. This spec does not resume fn-79 or block the fn-70 canary.

### Resolved implementation choices

Generated method references are checked against generator-owned structural declarations covering
the full method identity, request/response descriptor closure, and streaming flags. Public phantom
types and matching message names alone do not establish a valid binding. This needs no external
trust store or new authorization framework.

Use checked structural values alongside the existing descriptive Bytes/MessageRef representations.
Shared schema/value contracts must preserve the Umpire-to-Temporal import boundary. Concrete recursive
values consume explicit depth and work bounds; descriptor traversal tracks visited identities.
Implicit-presence scalars read descriptor defaults; optional/message/oneof presence remains explicit.
Canonical decoded maps have one entry per typed key in deterministic order. Raw duplicate map entries
follow protobuf last-value semantics only at the codec boundary. Properties do not normalize values.
Open enums retain unknown numeric values. Unsupported closed-enum cases reject explicitly. Integer
ranges are checked before protobuf lowering; concrete bytes remain exact. Floating-point metadata
remains discoverable, while unused floating-point operators may reject with a source diagnostic.
No unsupported-form policy may waive a qualifying example's requested clause.

Parameterized instances use the existing checked Target and finite adapters with a versioned exact
value carrier. Any serialized bridge must have typed reversible encoding and field-denotation
preservation; string equality does not replace typed evaluation. Existing Atom and literal encodings
remain byte-identical when the extension is absent. Finite domains and runtime admission scopes are
separate checked declarations with separate reported claims.

Cross-occurrence captures select an exact named earlier occurrence identity or ordinal under an
explicit key. Selection is deterministic and cannot silently mean the latest matching event.
Captured values are immutable per key and trigger; ambiguity, future references, and wrong keys
reject. Presence and oneof guards refine only the Boolean branches where their facts hold. Missing
evidence remains unresolved or rejected, never equality of absent operands.

Prove field-path and operand denotation, then request/result lowering and capture/scoped composition
against the actual emitted portable Contract. Metadata checks, codec round trips, and shared-renderer
goldens cannot substitute for these relationships or independently expected Lean/Go fixtures.

Focused generator fixtures, complete schema-input invalidation, and compatibility checks remain
required. Broad generated-API drift verification and CI expansion remain declined; see
[the recorded decision](../memory/declined/generated-api-drift-verification.md).

### Delivery units

Eleven cohesive tasks separate contracts with distinct proof and execution boundaries. Foundational
negative tests and proofs belong to each task; the final task combines evidence rather than postponing
correctness work. Source overlap may require serial implementation despite independent graph edges.

| Task | Deliverable | Dependencies |
| --- | --- | --- |
| 1 | Generated checked operation/schema bindings and original trust/fixture baseline | — |
| 2 | Exact bounded structural values | 1 |
| 3 | Typed field references and checked access with denotation proof | 2 |
| 4 | Parameterized instances and separate finite/runtime domains | 1, 2 |
| 5 | Same-step field Property semantics | 3, 4 |
| 6 | Keyed occurrence captures composed with scoped obligations | 5 |
| 7 | Portable typed field execution capability and independent codec/evaluator fixtures | 3, 5, 6 |
| 8 | Checked whole-Case field/capture lowering and atomic coverage admission | 4, 5, 6, 7 |
| 9 | Generated unary authored example and real Driver qualification | 8 |
| 10 | Two-operation Nexus authored example and real Driver qualification | 8, 9 |
| 11 | Combined compatibility, original trust comparison, bounded load evidence, and documentation | 9, 10 |

Task 10 consumes the shared qualification setup established by task 9. Both examples must record
named live passing results through public Prepare/Run; skipped tests or synthetic execution do not
satisfy R7. The unary example should use generated StartWorkflowExecution and independently relate
submitted nested fields to correlated started history/state. Confirm actual descriptors and model
requirements before authoring it: a Start acknowledgement does not prove eventual success, and the
response must not be assigned a fictitious workflow ID field.


## Requirement coverage

| Requirement | Tasks |
| --- | --- |
| R1 | fn-77-typed-operations-parameterized-actions.1, fn-77-typed-operations-parameterized-actions.4, fn-77-typed-operations-parameterized-actions.9, fn-77-typed-operations-parameterized-actions.10 |
| R2 | fn-77-typed-operations-parameterized-actions.2, fn-77-typed-operations-parameterized-actions.3, fn-77-typed-operations-parameterized-actions.7, fn-77-typed-operations-parameterized-actions.9, fn-77-typed-operations-parameterized-actions.11 |
| R3 | fn-77-typed-operations-parameterized-actions.3, fn-77-typed-operations-parameterized-actions.5, fn-77-typed-operations-parameterized-actions.6, fn-77-typed-operations-parameterized-actions.9, fn-77-typed-operations-parameterized-actions.10 |
| R4 | fn-77-typed-operations-parameterized-actions.8, fn-77-typed-operations-parameterized-actions.9, fn-77-typed-operations-parameterized-actions.10 |
| R5 | fn-77-typed-operations-parameterized-actions.4, fn-77-typed-operations-parameterized-actions.6, fn-77-typed-operations-parameterized-actions.9, fn-77-typed-operations-parameterized-actions.10, fn-77-typed-operations-parameterized-actions.11 |
| R6 | fn-77-typed-operations-parameterized-actions.6, fn-77-typed-operations-parameterized-actions.7, fn-77-typed-operations-parameterized-actions.8, fn-77-typed-operations-parameterized-actions.9, fn-77-typed-operations-parameterized-actions.10 |
| R7 | fn-77-typed-operations-parameterized-actions.9, fn-77-typed-operations-parameterized-actions.10 |
| R8 | fn-77-typed-operations-parameterized-actions.1, fn-77-typed-operations-parameterized-actions.2, fn-77-typed-operations-parameterized-actions.4, fn-77-typed-operations-parameterized-actions.5, fn-77-typed-operations-parameterized-actions.7, fn-77-typed-operations-parameterized-actions.8, fn-77-typed-operations-parameterized-actions.9, fn-77-typed-operations-parameterized-actions.10, fn-77-typed-operations-parameterized-actions.11 |
| R9 | fn-77-typed-operations-parameterized-actions.1, fn-77-typed-operations-parameterized-actions.2, fn-77-typed-operations-parameterized-actions.3, fn-77-typed-operations-parameterized-actions.4, fn-77-typed-operations-parameterized-actions.5, fn-77-typed-operations-parameterized-actions.6, fn-77-typed-operations-parameterized-actions.7, fn-77-typed-operations-parameterized-actions.8, fn-77-typed-operations-parameterized-actions.11 |
