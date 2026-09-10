# Umpire public API

Umpire is the reusable, Temporal-independent Lean library for semantic modeling, finite planning,
reviewed promotion, and Case production. For the cross-library map and Go runtime boundary, see the
[model architecture](../ARCHITECTURE.md).

## Imports and modules

Most consumers import the umbrella facade:

```lean
import Umpire
```

Focused public imports are available by responsibility:

| Import | Public responsibility |
| --- | --- |
| `Umpire.Core` | Stable definitions, traces, capabilities, laws, and finite kernels. |
| `Umpire.Model` | Finite-machine and expert Model authoring plus checked composition. |
| `Umpire.Model.Check` | Checked Model/Machine access, pure admission, and relation-indexed finite planning. |
| `Umpire.Property` | The authored Property language: fields, clauses, and their ordinary-Lean constructors. |
| `Umpire.Property.Check` | Property admission, canonicalization, and the checked trace view. |
| `Umpire.Property.Evaluate` | Pure evaluation of a checked Property over an admitted trace view. |
| `Umpire.Property.Elab` | Located-diagnostic elaboration of an authored Property. |
| `Umpire.Scenario` | The authored Scenario language: setup roles, occurrences, and trace shape. |
| `Umpire.Scenario.Check` | Scenario admission, canonicalization, and trace admission. |
| `Umpire.Query` | Bounded questions over a checked Model, Properties, and Scenarios. |
| `Umpire.Space` | Checked finite axes, fault intents and their lowering, and atomic point compilation. |
| `Umpire.Exploration` | Bounded finite selection, pinned precedence, and process-local sessions. |
| `Umpire.Observation` | Offline evidence mappings and accepted semantic traces. |
| `Umpire.ImplementationLink` | Checked correspondence between independent semantic Models. |
| `Umpire.Search` | Deterministic incremental planning over checked Queries. |
| `Umpire.Promotion` | Exact review-only source compilation from an unchanged planned Query. |
| `Umpire.Artifact` | Retained model-planning and offline-analysis artifact codecs. |
| `Umpire.Json` | Ordered JSON construction for codec owners. |
| `Umpire.Case` | Umpire provenance and temporary aliases for generated Testpilot protocol types. |
| `Umpire.Case.Compiler` | Generated Case assembly, source-bound producer diagnostics, and Umpire provenance. |
| `Umpire.SemanticInventory` | Explicit opt-in catalogs consuming semantic-owner contracts for documentation. |

Implementation modules remain behind these facades. Reusable Umpire modules cannot import the
domain-specific Temporal modules; the complete import graph is enforced by `make lint-model`.

## Model ownership and semantic imports

Ordinary authors keep `import Umpire.Model` (or `import Umpire`). The Model facade includes
finite-table/machine adapters and syntax-aware authoring. `Umpire.Property` and `Umpire.Scenario`
are the authored language modules themselves rather than facades: an author who elaborates imports
`Umpire.Property.Elab` or `Umpire.Scenario.Elab`, and one who evaluates imports
`Umpire.Property.Evaluate`. Query keeps its `Umpire.Query` facade.

Library code that consumes checked semantics imports `Umpire.Model.Check`. Property and
Scenario semantic implementations use that surface; Query uses their semantic modules, and
Planning uses Query's semantic module. Their complete import closures exclude `Umpire.Model.Elab`
and `Lean.Elab.Term`, including paths through external modules. `make lint-model` enforces this
boundary from compiled imports as well as the source inventory.

| Model owner | Responsibility |
| --- | --- |
| `Types` | Model specs, behavior tables, and inert source-reference/diagnostic data. |
| `Canonical` | Pure behavior-table construction, canonical encodings, and fingerprint inputs. |
| `Check` | Private checked/draft construction, composition validation, typed diagnostic selection, and checked Machine/planning access. |
| `Elab` | Syntax source-reference capture and mapping the same admission diagnostic to a located elaboration error. |
| `Table` | Finite-table/machine validation and proof-carrying adapters. |

`checkModel` returns
`Except LocatedError CheckedModel`; `model` extracts using checker-success evidence,
retaining its existing default proof. `elabModel` invokes the same pure admission authority
and reports its diagnostic in `TermElabM`. No unchecked constructor is exposed. Occurrence data
selects diagnostic locations, including the fallback when no captured syntax matches, without
entering semantic identity or fingerprint inputs.

Expert consumers can use `CheckedModel.machine` directly. `CheckedModel.withEquivalentMachine`
requires matching metadata and domains, equal authoritative initial/step relations, and unchanged
behavior description. Its optional planning evidence belongs to the replacement step relation;
omitting it leaves planning unavailable even if the original Model had finite planning. A finite
behavior table alone supplies no action-completeness proof. Enumeration order and duplicate
results remain the enumerator's responsibility, independently of canonical behavior identity.

## Semantic model lifecycle

A model maintainer defines a checked Model once. Ordinary authors then define independent
Properties and Scenarios, combine them in a bounded Query, and plan or explore only through that
checked Model. Model-owned steps decide outcomes; authoring order and instance search do
not select behavior.

```text
DraftModel ── checkModel ──▶ CheckedModel
                                     ├── Property
                                     ├── Scenario
                                     └── Query ──▶ Planning / Space / Exploration
```

The finite-machine adapter is the ordinary route for fully enumerable Models. Direct
`Machine` construction remains the expert route when authoritative propositions are
specified independently. Both routes converge before Property, Scenario, or Query checking.

`FiniteMachine.modelSpec` and `FiniteMachine.draftModel` assemble the ordinary finite
Model without deriving its evidence. The author still owns every ordered domain, encoder,
enumerator, closure proof, Action-executability proof, provider, connector, source, and stable ID.
`checkModel` remains the semantic admission boundary. Property, Scenario, Query, and Observation
constructors follow the same raw input → typed `check` result → explicitly proof-backed `checked`
shape; no constructor infers Model outcomes or checker success.

`FiniteTable` keeps ordered typed catalogs, setup alternatives, transition alternatives, Model
Outcomes, and Model Facts explicit, then validates domain closure before constructing the ordinary
finite Model. `DefinitionFamily`, `Property`, `Scenario`, `Query`, and
`Limits` reduce repeated structure while delegating to the existing language-owned checkers.
Their `checked` operations require explicit proof of checker success; the ordinary `check`
operations return the existing typed `Except` results.

Version-two Property data adds typed Boolean predicates, same-step branch groups, and guarded bounded
temporal clauses. A branch group is the `branches` clause; a bounded clause becomes guarded by
carrying an optional guard on `eventuallyWithin` or `neverWithin`. Boolean composition is limited to `atom`, `all`, `any`, `not`, and `oneOf` over
the field contexts admitted by the Property checker. Every applicable obligation is conjoined.
Exceptions are trigger-time applicability conditions and do not select a winning case or retract a
pending temporal obligation. Case analysis reports coverage, overlap, logical conflict, modeled
incompatibility, exhaustive completion, and limit exhaustion as separate bounded results.

`correlated_response%` is a readable spelling of `PropertyScopedClause`, admitted through the same
`property%`/`Property.check` boundary. Scoped clauses declare execution fields, an operation key,
natural bound, and a partial or final endpoint.
Projection admits causally supported, Model-authorized steps before obligation execution;
submissions and duplicate observations contribute no transition. Independent trigger windows count
only their operation's transitions, including self-loops. Runtime incompleteness leaves unresolved
windows inconclusive; a known violation remains proved.

`Shared.SemanticData`, `Shared.ScopedProjection`, and `Shared.ScopedObligation` own inert table data,
causal admission, and bounded countdown execution. Umpire's checked facades retain their semantic
proofs. `Umpire.Case.Scoped.lower` binds the generated wire decode to the checked projection and
Property, carrying exact clause/source provenance. `Testpilot.Scoped` interprets the closed table
capability without importing Umpire callbacks. Go admission rejects unsupported/stale capabilities
and incompatible resource ceilings before Driver execution; mutable evidence and windows belong to
one Run. The scoped fixture corpus exercises both offline parity and real public-facade recording.
Nexus operation cancellation remains deferred to fn-79; generic scoped support does not admit a
cancellation Case or supply an operation cancellation capability.

Query validity reports satisfiability, trigger exercise, answer, and search completeness separately.
An impossible scenario, an unexercised nonempty scenario, an unresolved prefix, and exhausted search
cannot become universal verification. Finite Model terminal declarations are conjunctive and never
inferred from deadlock. `Scenario` adds typed allow/forbid, occurrence, ordering, and adjacency
constraints through the existing canonical checker; ordering permits intervening allowed actions,
while adjacency and exactness deliberately impose stronger constraints.

All public declarations carry stable Definition IDs, source locations, and behavior fingerprints.
Limits are stage-specific. Exhaustion and limit-reached outcomes remain distinct, and a planning
artifact never proves that runtime work occurred.

Checked Queries may carry a default-empty `KnownGapSet`. Planning composes authored and phase-owned
sets once before traversal and returns a typed conflict before artifact publication. Known Gaps are
nonbehavioral: they are excluded from Query behavior fingerprints, cannot establish success, and do
not imply that absent limitations were discovered. Case lowering copies the exact composed rows;
runtime interpretation remains outside the semantic authoring path.

## Promotion API

`Umpire.Promotion` is scenario-neutral. `compilePromotionSource` accepts an unchanged checked Query,
its target-indexed planner kernel, a complete planning anchor, fresh source identities, and exact
expected bytes. It replans and rechecks the target-owned trace before returning one opaque
`CompiledPromotionSource`.

The module performs no runtime reproduction, reduction, replay, publication, or installation. Its
source template imports only generic Umpire modules and cannot receive caller-selected imports or a
namespace. The focused `Umpire.PromotionTests` build protects that boundary.

## Testpilot protocol and Umpire provenance

The checked-in `.proto` closure rooted at
`proto/internal/temporal/server/api/testpilot/v1/case.proto` owns the closed wire vocabulary.
`Testpilot.Protocol` exposes its generated Lean declarations, including Case, Program, Contract,
Run, values, paths, expressions, instructions, monitors, and Verdict data. Producers construct
those generated values through the context-safe `Testpilot.Authoring` facade.

`Umpire.Case` owns only Umpire-specific definition bindings, Behavior Fingerprints, sources, and
Known Gaps. `Umpire.Case.Provenance` encodes that data into opaque `producerData`; Testpilot and Go
do not interpret its contents. Temporary `Umpire.Case` type aliases delegate directly to the
generated declarations and are removed when downstream imports use `Testpilot.Protocol` directly.
The compatibility codec delegates directly to `Testpilot.ProtoJSON` and is covered by an
equivalent-output regression under `Testpilot.Tests` until its callers migrate.

The generated protocol contains no client, credential, worker, callback, endpoint, filesystem
access, executable hook, or runtime registry.

### Slots and Observations

A Slot is immutable single-assignment data used by later instructions. Slot storage is private to
one execution and is never exposed as a public scheduler API or recorded automatically.

An Observation is a declared typed projection on a Run Event. Contracts may inspect Observations
and selected Run Event fields. They cannot inspect arbitrary Slots or raw request and response
payloads. This opacity limits evidence authority; it does not classify every response field as
secret.

### Program instructions

Version one supports a closed set of generic instructions: authorized unary RPC invocation, Slot
await, Nexus completion, SDK Nexus start and await, workflow/activity finish, and Nexus response.
Every node declares its context, dependencies, optional guard, typed outcome schema, optional
response projections, activation reservations, and exact bounds. Unsupported context/opcode pairs
reject during preparation.

Programs are acyclic. Descriptor paths have a bounded grammar with explicit presence, oneof,
repeated fanout, and literal map-key selection. There is no general JSONPath or arbitrary expression
language.

### Contract semantics

Each rule has one initial state, finite transitions, and terminal satisfied or violated states.
Bounded-liveness rules have explicit horizons. The Evaluator checks expiry before transitions for
every Run Event kind, so a matching event at the deadline cannot revive an expired rule.

Captures copy only declared values, are bounded by count, bytes, and work, and are isolated per rule
and per Run. A transition may cite the matching event as support; Verdicts retain exact supporting
event sequences. Pending rules close inconclusive. A proved violation has precedence over later
operational and cleanup failures.

## Case production

Each Lean Producer owns its checked semantic lowering into generated protocol values through
`Testpilot.Authoring`. Umpire-backed Producers pass those values to `Umpire.Case.Compiler`, which
validates source-bound property rows, preserves typed unsupported-lowering errors, converts Known
Gaps in order, attaches exact opaque provenance, and performs final generated Case assembly. Its
input contains generated protocol values and introduces no parallel wire representation. A
Testpilot-only synthetic Producer can assemble a generated Case directly.

### Typed field lowering

A Case whose Properties name concrete request and evidence fields is lowered in both directions, and
each direction has one owner.

`Umpire.Case.Coverage` owns the request side. A requested coverage map binds each modeled input
field to the exact entrypoint and instruction whose request assignment constructs it, with the exact
modeled value. `Coverage.check` runs against the Program the Case actually produced and rejects the
whole requested Case before any Driver I/O when a mapping names an unknown instruction, a field the
Program never constructs from an exact value, a different value than the model declares, a presence
read (which describes a payload rather than constructing one), or result coordinates, which are
covered by declared Observations and never by a request assignment.

`Umpire.Case.Observed` owns the evidence side. `Observed.pathOf` derives the runtime read path a
modeled operand's coordinates describe, rooted at the declared Observation's own message, mirroring
`Coverage.targetPath` in the other direction. Because every field path in a derived Contract rule is
`Observed.pathOf` applied to the same `PropertyFieldPath` the Property compares, moving a coordinate
in the Property moves the runtime read with it, and a field the Property stops naming stops being
read. Reads it cannot derive — a wrong oneof group, an unselected member, a presence read, a
repeated element, a keyed map lookup, a cardinality — reject by name rather than resolving to an
approximate path.

Operation identity is versioned separately from the model values it carries.
`Umpire.Operation.Canonical.rpcSchema` names the selected operation by method full name, both
payload signature roots and streaming shape, and reaches the descriptor closure through
`Canonical.closure`, a bounded structural fold, so a proto revision that preserves method and
message names but changes the descriptor closure changes the identity. That encoding is versioned
`parameterized-v2-` under the named migration `parameterized-operation-identity-v2`; the digest is
bounded, so it retains what `Canonical.rpcSchema_inj` states rather than claiming injectivity over
an unbounded schema space. Parameter values belong to canonical Action instances and enter Scenario
Fingerprints through the domain's canonical meaning, which records the explored dimension, the
coverage claim, every sample, and the declared runtime scope.

`Testpilot.ProtoJSON.canonical` delegates canonical Case encoding to `Protobuf.Json`;
`Umpire.Case.ProtoJSON.canonical` is only a temporary forwarding compatibility name. Testpilot's
`Prepare` owns static admission, including Program and Contract closure, types, paths, instruction
contexts, limits, identity, scope, and environment policy, before Driver I/O. Temporal-specific
Producer declarations live outside Umpire.

## Runtime handoff

The Lean package stops at canonical Case data. It does not open a Driver, schedule instructions,
create workers, collect runtime credentials, or choose a Monitor. Testpilot performs:

```text
testpilot.Prepare(case, profile)
PreparedCase.Run(ctx, driver)
```

Static preparation snapshots the admitted Case and Profile without Driver I/O. The prepared Contract
creates the private Run-local Monitor. The internal Executor owns scheduling, recording, Slots,
effect handles, cancellation, and cleanup. Alternate Drivers are the environment extension seam.

Exact Case 1.0 is the sole format. Resource-bearing Programs declare symbolic text resources; their
IDs identify namespace, task-queue, and named Nexus endpoint relationships but contain no
physical names or transport addresses. The Profile owns physical binding values. `Prepare` resolves
them into immutable private prepared data and includes the complete binding fingerprint in Prepared
Case identity while preserving the source Case bytes. `Run` checks the Driver's matching identity,
calls its no-I/O `Validate` hook before Monitor creation, and calls `Open` only after validation.
This environment identity does not alter Behavior Fingerprints, Contract meaning, or producer-owned
provenance.

## Artifact and generated-view boundaries

Semantic owners import the neutral `Umpire.OutcomeClassification` vocabulary and
`Umpire.KnownGap` carry contracts. `OutcomeClassification` contains only classifier descriptors,
matchers, list operations and propositions, and projection-sentinel descriptors over Lean's `Init`
foundation. Concrete classifiers and their exhaustive `ExactlyOne` proofs stay with Planning,
Artifact runtime, Observation, Implementation Link, and Verdict. Implementation Link also owns its
projection-only `not-evaluated` sentinel; it is not an outcome constructor.

`KnownGapCarryMapping` belongs to `Umpire.KnownGap`. Observation owns the lossy admission mapping
from code and optional subject to an Evidence Gap; kind and detail are absent. Result Artifact owns
exact carry of kind, code, subject, and detail. The inventory consumes these owner declarations and
owns only catalog descriptors, lineage, scope, source shapes, and catalog validation.

Catalog consumers explicitly import `Umpire.SemanticInventory` or its focused modules; `import
Umpire` does not aggregate the inventory. `Umpire.SemanticInventory.Types` retains relocated
qualified names through ordinary imports for explicit inventory consumers. Production Umpire
modules outside the inventory cannot reach it directly or transitively, including through facades,
helpers, external modules, or test fixtures. `make lint-model` enforces this direction and the
neutral module's foundation-only imports while allowing dedicated inventory consumers such as the
Planning Known Gap catalog tests.

Planning artifacts, the semantic inventory, and Generated Views remain deterministic projections
owned by their existing modules and generators. They do not execute Cases or define Contract
results. The Case conformance tree is separately owned by the Case renderer and transactional Go
publisher.

Lean Case fixtures compare byte-for-byte. Runtime results use a named closed stable projection only
for stable fields, while excluded Run IDs, event timing and identities, causal links, activation
identities, support references, and diagnostics are structurally validated. There is no generic
normalization or ignored-field registry.

## API invariants

- Reusable Umpire code is Temporal-independent.
- Public semantic declarations are checked before planning or Case lowering.
- The generated Testpilot Case, Program, Contract, and Run vocabularies are finite, versioned, and
  bounded by their `.proto` schema.
- A Program contains no clients, credentials, callbacks, or arbitrary executable code.
- A Contract is the sole authority for live and offline Verdict semantics.
- Slots are private execution state; Observations are the declared evidence surface.
- Preparation is static and immutable; one Prepared Case supports isolated concurrent Runs.
- Resource-bearing Case 1.0 Programs use complete symbolic bindings; resource-free Programs may have an empty environment.
- Run disposition, cleanup status, and Verdict remain independent.
- Generated data and views cannot create behavior.
- Promotion remains generic and review-only.

## Superseded runtime history

The pre-fn-64 portable plan, resident executor, caller-specific adapter, and separate Run Evaluation
path were removed. Historical documents label those interfaces explicitly as superseded. They are
not compatibility surfaces of `Umpire.Case`.
