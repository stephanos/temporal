# Umpire semantic authoring languages

Status: interview-refined design contract; implementation is split across three Flow specs.

Implementation allocation:

- `fn-3-umpire-semantic-authoring-and-planning`: shared vocabulary, capabilities, Property,
  Behavior, Query, bounded planning, `DrivePlan`, and `ExperimentSpec`;
- `fn-4-umpire-observation-and-semantic-verdicts`: evidence interpretation, qualified traces,
  derivations, and property verdicts; depends on `fn-3`; and
- `fn-5-umpire-discovery-promotion-and-artifact`: glossary, promotion, and artifact evolution;
  depends on `fn-3` and `fn-4`.

This file is the shared architectural contract. Testable implementation requirements live in the
three specs and are not duplicated here.

## 1. Purpose

Umpire needs approachable authoring without collapsing four different jobs into one large DSL:

1. define reusable semantic properties;
2. describe exact or exploratory behavior;
3. ask a planner or runner a precise question; and
4. interpret execution evidence as semantic observations.

The current Lean regression slice proves that a pure compiler can produce a deterministic,
inspectable `ExperimentSpec`. It also exposes an authoring problem: `Regression` and `ModelTarget`
mix behavior, properties, model bindings, and inspection concerns. This design separates those
concerns while retaining a single Lean-owned semantic authority.

The first audience is Umpire and Lean model engineers. The concepts should remain teachable to
Temporal feature engineers, but the first release is not an application-developer testing DSL.
Syntax is experimental; the four language responsibilities and stable semantic identities are the
compatibility boundary.

The desired user experience is:

```text
property          behavior             query
what must hold    what may happen      what to solve or run
     |                  |                    |
     +------------------+--------------------+
                        |
                        v
                 planner / compiler
                        |
                 DrivePlan / ExperimentSpec
                        |
                        v
                     runtime
                        |
                    raw evidence
                        |
                        v
              observation interpretation
                        |
                  semantic trace
                        |
                        v
                 property results
```

The first release stops before the runtime box: synthetic evidence exercises interpretation and
property checking offline. Live execution later plugs into the same generated artifact and
observation boundary.

## 2. Design basis

This design synthesizes the following existing documents:

- `UMPIRE.md`: separate observation from action, keep one semantic source, and qualify every claim
  by evidence and bounds;
- `UMPIRE_COMPONENTS.md`: connect independently useful tools through versioned artifacts;
- `UMPIRE_CHATS.md`: treat regressions and exploration as selections from a Lean-owned semantic
  space, while keeping runtime evidence separate;
- `UMPIRE_EXAMPLE.md`: use compositional model capabilities and explicit cross-domain connectors;
- `UMPIRE_EXM.md`: prefer small model-derived traces and narrow conformance mappings over universal
  whole-server trace reconstruction;
- `UMPIRE_UX.md`: hide mechanics behind deep modules and avoid one universal builder; and
- `UMPIRE_VISION.md`: reuse the same semantics for deterministic regressions, exploration, faults,
  local execution, CI, remote profiles, and black-box observation.

`UMPIRE_LEAN.md` is intentionally not a design source here. Umpire3 is not a dependency, reuse
source, or behavioral oracle for this design.

## 3. Decisions

1. Lean remains the sole authority for behavioral semantics.
2. There are four small authoring languages: Property, Behavior, Query, and Observation.
3. They share one typed semantic vocabulary but do not share responsibilities.
4. Properties are pure and reusable across model compositions.
5. Properties declare required capabilities rather than belonging nominally to one domain.
6. Behavior is a constrained semantic trace space. `actionsExactly` fixes controllable action
   order; `traceExactly` is the singleton case.
7. There is no separate author-facing Drive DSL. A concrete `DrivePlan` is a generated artifact.
8. Query syntax makes existential, universal, exploratory, and execution claims explicit.
9. Raw evidence is interpreted separately into qualified semantic traces before properties run.
10. Missing, ambiguous, conflicting, or unsupported evidence never becomes success.
11. All search and execution is explicitly bounded. Unbounded temporal claims are outside the
    initial language.
12. Public vocabulary has stable namespaced identities and generates `model/GLOSSARY.md`.
13. Portable declarations are typed, inspectable data with a Lean denotation. Opaque Lean escape
    hatches cannot participate in planning, persisted artifacts, or promotion.
14. Capability records are the artifact-level contract. Lean type classes may provide authoring
    convenience, but requirements remain explicit after elaboration.
15. Property, Behavior, Query, and Observation lower to separate typed internal forms. They share
    vocabulary and semantic types, not one universal instruction tree.
16. Exploratory behavior is the default. Exact controllable actions and an exact complete semantic
    trace are distinct restrictions.
17. The existing combined `Regression` structures are replaced cleanly when the new languages land;
    they do not remain as a second public authoring path.
18. The first trusted planner is a deterministic, lazy, bounded Lean enumerator behind a replaceable
    planner interface.

## 4. Goals

- Let a property be written once and checked by model exploration or against qualified live
  evidence.
- Let a behavior range from broadly exploratory to one exact semantic action order.
- Let a planner connect properties and behaviors in either direction.
- Support cross-domain properties without creating a global untyped state bag.
- Preserve the distinction between requested actions, realized actions, semantic transitions, raw
  evidence, semantic observations, and property verdicts.
- Produce deterministic, inspectable artifacts suitable for replay, minimization, promotion, and
  generated developer views.
- Make authoring vocabulary searchable and explainable.
- Let representative examples read primarily as domain intent: capability requirements, bounds,
  and qualification stay visible, while compiler plumbing and artifact fields remain hidden.
- Make artifacts equally useful to human reviewers and deterministic automation.
- Demonstrate reuse by authoring a second small scenario without changing the public language or
  compiler concepts.

## 5. Non-goals

- A second semantic authority in Go, JSON, Gherkin, or generated code.
- An exact Go goroutine, RPC, or distributed-event scheduler.
- Unbounded liveness proofs in the first implementation.
- Arbitrary callbacks in portable behaviors or properties.
- Inferring correctness directly from requested actions or runtime receipts.
- Hiding truncation, unsupported vocabulary, evidence gaps, or cleanup failures.
- Replacing specialized unit, race, persistence, schema, authorization, performance, or exact
  handler tests.
- Implementing a live runtime as part of the first language slice.
- A general Temporal application-testing language for SDK or workflow authors.
- A compatibility facade that permanently exposes both `Regression` and the new languages.
- An upstream Temporal compatibility commitment for the experimental syntax.

## 6. Shared semantic vocabulary

The languages share a Lean-owned catalog of typed semantic declarations:

- model modules, interfaces, compositions, and targets;
- resources, entities, states, finite attributes, and relations;
- inputs, outputs, actions, faults, and semantic observations;
- properties, behaviors, queries, and observation mappings; and
- stable identities, documentation metadata, provenance, and semantic digests.

The catalog is vocabulary, not an authoring language. Mechanical Protobuf declarations and dynamic
configuration catalogs may supply structure, but they do not acquire behavioral meaning until an
authored semantic declaration interprets them.

The four languages use separate typed internal forms. They may reuse small common semantic types
such as identities, finite values, expressions, and trace positions, but there is no universal AST
whose variants mix authoring, planning, execution, and evidence responsibilities.

Every public entry has a stable namespaced identity and a kind. For example:

```text
workflow.state.running
nexus.action.requestCancel
nexus.observation.cancelDelivered
workflow-nexus.relation.ownsOperation
nexus.property.cancelIsUnique
```

Wrong-kind references fail compilation even if their textual names happen to match.

Every capability also has a stable identity, versioned contract, and checked laws. A persisted
declaration records the capabilities it consumes; it never relies on reconstructing Lean instance
search. Documentation text and source ordering do not change semantic identity, while a change to a
consumed capability contract does.

## 7. Capability-scoped composition

A property or behavior does not belong nominally to a single domain such as `Nexus`. Instead, it
declares the semantic capabilities it requires. A model target or composition provides those
capabilities.

The artifact-level representation is a capability record. Lean type classes may elaborate concise
authoring syntax into that explicit record:

```lean
def cancelIsUnique
    [HasCancellation M] :
    TraceProperty M :=
  ...

def callerCloseHonorsCancellation
    [HasWorkflowLifecycle M]
    [HasNexusCancellation M]
    [HasOwnership WorkflowRun NexusOperation M] :
    TraceProperty M :=
  ...
```

The approachable surface hides the carrier type:

```lean
property callerCloseHonorsCancellation where
  requires Workflow.Lifecycle
  requires Nexus.Cancellation
  requires Workflow.ownsNexusOperation

  whenever Workflow.callerClosed
  eventuallyWithin cancelBudget Nexus.cancelDelivered
```

A query selects a checked composition such as `WorkflowNexus`. Compilation fails if the selected
target cannot provide every required capability or connector relation. This supports cross-domain
properties without permitting arbitrary access to unrelated model internals.

Evaluation exposes a capability-limited view of the semantic trace. The full trace may exist inside
the planner or checker, but a property cannot inspect vocabulary outside its declared requirements.
Post-hoc validation of unrestricted access is not sufficient.

If two composed capabilities provide competing meanings for one identity or relationship,
composition fails unless an explicit connector selects or reconciles them. Neither declaration
order nor type-class search order chooses silently.

Cross-domain composition must exist in the semantic model before authoring. A property cannot invent
a runtime `Combine` operation or an unproved relationship between two domains.

## 8. Property DSL

### 8.1 Responsibility

The Property DSL defines pure predicates over semantic states, transitions, relations, inputs,
outputs, or finite traces. It does not drive the system and does not name runtime evidence sources.

Initial property kinds are:

- state invariants;
- transition precondition/postcondition contracts;
- identity and relation properties;
- semantic input/output contracts;
- finite trace ordering properties; and
- bounded progress or quiescence properties.

A portable property is a typed data declaration, not an opaque Lean function. Lean defines its
denotation over a semantic trace and an executable evaluator that produces a structured verdict.
The evaluator must be shown to agree with the denotation for the supported portable core.

The common semantic trace contains an initial semantic state followed by typed steps. Each step
retains the selected semantic action, model-owned outcome, resulting state, and emitted semantic
observations. A property receives only the view admitted by its capability requirements.

Example:

```lean
property cancelIsUnique where
  requires Nexus.Cancellation

  always cancelIntents.perOperation ≤ 1

property honoredDelivery where
  requires Nexus.Cancellation

  whenever cancelRequested
  eventuallyWithin cancelBudget cancelDelivered
```

`eventuallyWithin` is a finite semantic bound, not an unbounded liveness claim and not an implicit
wall-clock sleep. Every bound has a declared unit, such as semantic transitions, selected actions,
observation positions, or model-defined logical time. A named budget resolves to a typed finite
value; mixing units fails checking. The property result records the expanded value and unit.

### 8.2 Purity rules

A property may reference only semantic vocabulary supplied by its declared capabilities. It may not
reference:

- logs, spans, RPC method names, poll loops, or concrete storage records;
- environment profiles or evidence channels;
- action realizers or requested fault controls;
- mutable planner or coverage state; or
- opaque callbacks.

An expert-only opaque Lean predicate may exist below the portable language, but it cannot be
planned, serialized, promoted, used from another language, or presented as part of the portable DSL.

Property evaluation does not return a bare Boolean. A structured verdict identifies the responsible
property clause, relevant trace span, evaluated bound, semantic provenance, and any observation
derivations used to qualify the trace.

Human statements, documentation, generated tests, and support views are projections of the checked
property declaration, not independently authored semantics.

## 9. Behavior DSL

### 9.1 Responsibility

The Behavior DSL defines a set of admissible semantic traces. It says what may, must, or must not
happen; it does not say whether those traces are correct.

Exploration is the ordinary authoring mode. A behavior declares typed symbolic resource roles and
semantic setup constraints; the query and selected target bind those roles to concrete values or
search for bindings. This keeps behavior declarations reusable instead of baking one fixture into
each declaration.

It owns:

- initial semantic setup and symbolic resources;
- allowed, required, and forbidden semantic actions;
- named action occurrences and occurrence bounds;
- choices and finite variation axes;
- partial and total semantic ordering constraints;
- optional and repeated actions;
- semantic fault intents and fault scopes; and
- explicit planning bounds and omissions.

Example:

```lean
behavior callerClosure where
  requires Workflow.Lifecycle
  requires Nexus.Cancellation
  requires Workflow.ownsNexusOperation

  given operation.isStarted
  allow callerClose, requestCancel, retryCancel

  let cancel := require requestCancel
  let close := require callerClose
  cancel before close

  occurs retryCancel atMost 2
  within 5 actions
```

### 9.2 Constraint meanings

The surface must distinguish similar-looking constraints precisely:

- `allow a`: `a` may occur subject to its occurrence bound;
- `require a`: introduce a named required occurrence of `a`;
- `forbid a`: no occurrence of `a` is admissible;
- `occurs a exactly|atLeast|atMost n`: constrain the number of occurrences;
- `x before y`: named occurrence `x` precedes named occurrence `y`, while other allowed actions may
  interleave;
- `sequence [a, b, c]`: require one occurrence of each in that relative order, while other allowed
  actions may interleave;
- `adjacent [a, b]`: require the named semantic occurrences with no semantic action between them;
  and
- `actionsExactly [a, b, c]`: admit only that total sequence of controllable semantic actions while
  still allowing model-owned outcomes to vary; and
- `traceExactly witness`: admit one fully selected semantic trace, including its setup, choices,
  actions, outcomes, states, and observations.

Ordering is over semantic actions, not implementation events. Exact semantic order does not promise
exact network, scheduler, storage, or goroutine order.

Ordering constraints form a directed acyclic graph. Incomparable occurrences express semantic
concurrency. For each generated `DrivePlan`, the planner selects and records one deterministic
linear extension. Runtime-level concurrency remains outside the semantic ordering claim.

### 9.3 Exact restrictions are not a separate DSL

Mathematically, an exploratory behavior denotes a set of traces. `actionsExactly` narrows the
controllable action schedule but may still denote several traces when the model owns choices or
outcomes. `traceExactly` denotes a singleton. Tightening either form adds constraints and can only
remove traces.

```lean
behavior callerClosureRegression :=
  callerClosure |> actionsExactly [requestCancel, callerClose]

behavior promotedCallerClosureRegression :=
  callerClosure |> traceExactly selectedWitness
```

This supports a continuous workflow:

```text
broad behavior space
    -> selected witness or counterexample
    -> minimized constraints
    -> exact or tightly constrained pinned regression
```

There is no translation into a second Drive language during promotion. Promotion retains the exact
trace together with its properties, target composition, expanded bounds, source query, semantic
digests, selection reason, and provenance.

### 9.4 Faults

The Behavior DSL may request semantic faults such as one retryable delivery failure. A fault request
is not evidence that the fault occurred. The generated plan declares a required control capability,
and execution must produce a realization receipt linked to the intercepted occurrence. Missing or
misdirected realization is execution divergence.

## 10. Query DSL

### 10.1 Responsibility

The Query DSL connects a behavior, one or more properties, a compatible model target, and a search or
execution mode. It makes the quantifier and claim strength explicit.

Property-led and behavior-led questions are equally first-class. A user may ask the planner to find
paths that satisfy or challenge a property, or select paths from a behavior and check chosen
properties. The planner never infers the quantifier from an ambiguous bag of ingredients.

### 10.2 Model-only queries

Universal bounded verification:

```lean
verify cancelIsUnique
  over callerClosure
  on WorkflowNexus
  within nexusSmall
```

This checks every admissible trace inside complete finite bounds. It returns either
`verifiedWithinBounds` or a counterexample. Exhaustive mode is available only when every relevant
type is finitely enumerable and the planner can publish complete bounds and explored counts. It
fails rather than truncating. An empty behavior returns `unsatisfiable`; it does not verify every
property by vacuity unless a future query form opts into that claim explicitly.

Existential witness search:

```lean
find witness
  of honoredDelivery
  in callerClosure
  on WorkflowNexus
  shortest
```

Counterexample search:

```lean
find counterexample
  to cancelIsUnique
  in callerClosure
  on WorkflowNexus
  using coverageGuided
  budget 100
```

A budgeted search that finds nothing returns `budgetExhausted`, not a universal success claim.

### 10.3 Execution queries

```lean
run callerClosure
  on WorkflowNexus
  selecting pairwise
  budget 20
  checking [cancelIsUnique, honoredDelivery]
  using local
```

The planner selects concrete traces, compiles each into a `DrivePlan` and `ExperimentSpec`, and asks
the runtime to realize them. Property results apply only to traces that were both realized and
sufficiently observed.

An exact regression uses the same query language:

```lean
run (callerClosure |> actionsExactly [requestCancel, callerClose])
  on WorkflowNexus
  checking [cancelIsUnique, honoredDelivery]
  using local
```

Every selected experiment retains separate planning, execution, observation, and per-property
results. A strict query summary cannot report success when any required trace diverged or any
required property is `unknown`, `conflict`, or `unsupported`. Supported property results remain
visible even when another property cannot be evaluated.

### 10.4 Strategy and bounds

The first backend is a deterministic, lazy, bounded Lean enumerator supporting the strategies needed
by the first slice. Later strategies may include pairwise, t-wise, seeded random, transition
coverage, relation coverage, outcome coverage, and coverage-guided selection. Strategy and budget
are query policy, not behavior or property semantics, and later backends must implement the same
result contract.

Phase-specific bounds remain distinct:

- behavior bounds constrain semantic traces;
- search bounds constrain planning effort and completeness;
- execution bounds constrain actions, concurrency, and deadlines;
- observation bounds constrain source closure and evidence volume; and
- minimization bounds constrain reduction attempts.

There is no universal `Bounds` value that silently mixes these meanings.

Named typed profiles may provide concise defaults for these phases. A checked query and every
persisted artifact expand each value and unit. Query identity covers resolved declaration
identities, consumed semantic digests, expanded bounds, strategy, seed, and target composition;
incidental source ordering and documentation text do not affect it.

## 11. Generated DrivePlan and ExperimentSpec

The runtime executes a generated concrete artifact rather than evaluating behavior constraints
directly.

A `DrivePlan` records:

- the selected semantic action occurrences and deterministic linear extension of their partial
  order;
- grounded and still-symbolic resources and identity bindings;
- selected choices, variants, and requested faults;
- required drive capabilities and semantic preconditions;
- action, observation, cleanup, and execution bounds;
- expected semantic observation checkpoints;
- the source behavior, query, target, model identity, and selection reason; and
- explicit omissions.

An `ExperimentSpec` is the portable, environment-independent envelope consumed by execution,
checking, replay, and generation. It embeds or references the `DrivePlan` plus property identities,
observation requirements, format version, semantic identity, and provenance.

The `DrivePlan` is inspectable and replayable but is not normally authored. Low-level framework
tests may construct it directly as an escape hatch; such tests are below the ordinary DSL and must
declare that their plan was not produced from a checked behavior.

Human readability and stable machine consumption have equal priority. A `DrivePlan` records what
the runtime should attempt; it never upgrades that request into evidence that an action, fault, or
model outcome occurred.

## 12. Observation DSL

### 12.1 Responsibility

The Observation DSL separately defines how raw execution evidence establishes semantic observations.
Properties consume semantic observations; they never consume raw logs, spans, RPCs, or storage
records.

A semantic observation belongs to the shared vocabulary:

```lean
observation Nexus.cancelDelivered where
  intent : CancelIntentId
  operation : NexusOperationId
  attempt : Nat
```

A profile-specific mapping interprets raw evidence:

```lean
interpret Nexus.cancelDelivered from TemporalHistory where
  match NexusOperationCancelDelivered
  bind intent from cancellationId
  bind operation from operationId
  order by eventId
  source closes at historyEnd
```

Another mapping may establish the same observation from trusted in-process events. Both must produce
the same semantic meaning.

Authors define small typed mapping rules. The compiler combines them into one checked interpretation
plan for an evidence profile. Overlapping rules, incompatible bindings, wrong-kind output, and
ordering conflicts fail before evidence is processed; first-match order is never semantic.

### 12.2 Mapping contract

Each mapping declares:

- accepted evidence source and environment profile;
- normalization and sensitive-field disposition;
- symbolic-to-runtime identity and relation bindings;
- causal or source-local ordering requirements;
- source-closure condition and gap detection;
- duplicate, ambiguity, and conflict handling;
- semantic observations it may establish; and
- mapping version and provenance.

For every evidence field, the mapping also declares whether interpretation retains, redacts, hashes,
or rejects the value. A portable semantic trace contains only approved normalized data. Raw evidence
is a separately controlled artifact and is not copied wholesale into an `ExperimentSpec` or result.

### 12.3 Interpretation pipeline

```text
raw evidence
  -> normalize typed records
  -> bind symbolic and runtime identities
  -> establish source-local and causal order
  -> verify source closure and detect gaps
  -> construct a qualified semantic trace
  -> evaluate pure properties
```

The interpreter produces a qualified wrapper around the pure `SemanticTrace`. Qualification records
source closure, gap analysis, completeness, mapping identity, provenance, and derivations. A property
still receives only its capability-limited semantic view; the evaluator gates the call before the
property runs.

Every established observation carries a compact derivation linking the mapping version, matched
evidence identities, bindings, ordering facts, and closure evidence. Property verdicts reference the
derivations they consume.

If available evidence is compatible with multiple semantic traces, the initial design returns
`unknown` together with the alternatives and missing discriminator. Evaluating a property over all
compatible traces is a possible later feature with explicit quantifier semantics, not an initial
correctness shortcut. Incompatible established facts return `conflict` instead.

## 13. Results and failure semantics

Planning, execution, observation, and property checking return separate result families.

### 13.1 Planning results

- `found`: a witness, counterexample, or selected experiment was produced;
- `noSuchTraceWithinCompleteBounds`: complete search proved no matching trace exists within its
  published bounds;
- `budgetExhausted`: incomplete search stopped without a result;
- `unsatisfiable`: behavior constraints admit no trace; and
- `invalid`: vocabulary, capabilities, types, or bounds are malformed.

`unsatisfiable` remains distinct from universal verification so an empty behavior cannot silently
prove every property.

### 13.2 Execution results

- `realized`: requested semantic actions were realized with required receipts;
- `diverged`: the selected semantic plan was not realized;
- `unsupported`: the environment lacks a required drive capability; and
- `infrastructureFailed`: allocation, transport, persistence, artifact, or cleanup mechanics failed.

An exact plan that diverges is not a property counterexample because its intended semantic trace did
not occur.

### 13.3 Observation and property results

- `satisfied`: a complete qualified semantic trace satisfies the property;
- `violated`: a complete qualified semantic trace violates the property;
- `unknown`: evidence is incomplete, unclosed, gapped, or ambiguous;
- `conflict`: evidence sources establish incompatible semantic facts; and
- `unsupported`: no selected mapping or evidence profile can establish required vocabulary.

Properties are evaluated independently when their required observations are available. Supported
verdicts remain visible if another property is `unknown` or `unsupported`; the aggregate query result
is nevertheless incomplete. Results are a matrix indexed by selected experiment and property, with
phase outcomes retained rather than collapsed into one Boolean.

`realized` never implies `satisfied`. `budgetExhausted` never implies verified. Missing evidence never
implies absence.

### 13.4 Authoring and infrastructure errors

Unknown vocabulary, wrong-kind references, missing capabilities, invalid bounds, cycles, incompatible
targets, and malformed mappings fail before environment allocation. Infrastructure errors retain any
partial plan, run, evidence, and cleanup artifacts already produced; they do not become semantic
verdicts.

Every authoring diagnostic has a stable kind, declaration identity, source path, offending value,
and related identities. Human explanations are renderings of that structured value rather than an
API contract encoded only in prose.

## 14. Vocabulary glossary and index

Lean declarations are authoritative. `model/GLOSSARY.md` is a deterministic, checked-in human view
generated from public vocabulary metadata. The repository's top-level Makefile owns regeneration
and freshness checks; no model-local Makefile is introduced or extended for this work.

Example declaration metadata:

```lean
vocabulary action nexus.callerForceClose where
  summary := "Request closure of the caller owning a Nexus operation."
  kind := .action
  model := Nexus
  aliases := []
```

The generated entry includes:

- stable identity, kind, summary, and longer definition;
- Lean declaration and source location;
- required and provided capabilities;
- model targets in which the term is available;
- properties, behaviors, queries, and mappings that reference it;
- aliases, deprecations, and replacement identities; and
- related vocabulary.

The glossary covers framework concepts as well as model vocabulary: property, behavior, query,
observation, evidence, plan, target, resource, state, action, relation, output, fault, and result.
Mechanical Protobuf inventory remains in the API catalog and is not copied into the semantic
glossary.

Proposed discovery surface:

```text
umpire glossary list
umpire glossary explain nexus.callerForceClose
umpire glossary check
umpire property list|explain
umpire behavior list|explain
umpire query list|explain
```

The same catalog may emit a machine-readable index for IDEs, generated documentation, test wrappers,
and compatibility checks. Generated Markdown and machine views are projections, not semantic
authority.

Generation fails on duplicate or wrong-kind identities, broken references, alias cycles, missing
deprecation replacements, stale checked-in output, internal inconsistency, or nondeterministic
ordering.

## 15. Determinism, identity, and evolution

Every public declaration and persisted artifact has:

- a format version;
- stable namespaced identity;
- source identity and provenance;
- a semantic digest covering consumed contracts;
- explicit bounds and omissions; and
- deterministic collection and field ordering.

Renames require an explicit alias or deprecation record. A semantic change changes the relevant
semantic digest. Proof-only or documentation-only changes may preserve it when the consumed contract
is unchanged.

Artifact readers reject unknown major format versions and accept only explicitly compatible minor
additions. When old data requires transformation, a deterministic named migration records the source
and destination versions. Readers never ignore meaning-bearing fields or infer a new meaning for an
old field.

Portable behavior and property declarations use serializable combinators. Opaque Lean or Go
callbacks may exist as framework escape hatches, but they cannot participate in deterministic
planning, minimization, portable artifact generation, or cross-language reuse unless replaced by a
stable declared semantic operation.

## 16. Verification strategy

### 16.1 Language and planner tests

Small synthetic model families verify that:

- capability requirements accept valid compositions and reject missing interfaces;
- conflicting capability providers are rejected unless an explicit connector reconciles them;
- cross-domain properties use only their declared capabilities;
- unknown and wrong-kind vocabulary fails;
- behavior constraints reject cycles, impossible bounds, and unsatisfiable combinations;
- `actionsExactly [a, b]` fixes the controllable action schedule without fixing model outcomes;
- `traceExactly witness` denotes one complete semantic trace;
- adding a constraint can only narrow a behavior space;
- complete search never truncates silently;
- an unsatisfiable behavior does not report universal verification;
- budgeted selection is deterministic for a fixed strategy and seed;
- witnesses and counterexamples replay through canonical semantics; and
- a promoted witness remains a member of its source behavior space.

### 16.2 Observation tests

Fixture-only tests verify:

- different evidence profiles produce equivalent semantic observations when they describe the same
  behavior;
- source closure, sequence gaps, ambiguity, duplicate evidence, and conflicts fail closed;
- every established observation retains an auditable derivation;
- unrelated evidence cannot discharge a causally scoped obligation;
- requested, attempted, applied, committed, and aborted operations remain distinct;
- sensitive fields are rejected or redacted according to mapping policy; and
- every `unsupported`, `unknown`, and `conflict` path is stable and inspectable.

Observation mappings receive independent mutations so a property and its adapter do not silently
share the same mistake.

Portable property evaluators additionally have agreement tests or proofs connecting executable
verdicts to their Lean denotation. Negative declaration fixtures, deterministic golden artifacts,
and replay tests exercise boundaries without using one layer as another layer's oracle.

### 16.3 End-to-end and mutation tests

The first end-to-end slice uses one capability-scoped Nexus property, one exploratory caller-closure
behavior, one witness or counterexample query, the same behavior restricted to one exact action
schedule, and one promoted exact trace. It checks deterministic planning, `ExperimentSpec`
generation, synthetic evidence interpretation, property evaluation, promotion, and glossary
generation without requiring a live Temporal environment.

Model, property, planner, observation, and later implementation mutations must each make the
appropriate layer fail for the intended reason. Branch or case coverage remains diagnostic evidence,
not proof of correctness.

### 16.4 Artifact checks

Catalogs, glossary, plans, experiment specs, traces, and results are regenerated and compared
byte-for-byte. Stale semantic identities, undeclared omissions, unsupported vocabulary, or
nondeterministic ordering fail verification.

## 17. First bounded implementation slice

The language design is broader than the first implementation. The first slice should prove the
boundaries with minimal surface:

1. a small public vocabulary, explicit capability records, and generated glossary;
2. capability interfaces sufficient for Nexus cancellation, caller lifetime, and one checked
   Workflow–Nexus connector property, including missing-connector rejection;
3. portable state, transition, relation, and finite-trace property combinators with typed bounds,
   Lean denotation, executable evaluator, and structured verdicts;
4. symbolic behavior setup, allow/require/forbid, named partial ordering, occurrence bounds,
   `actionsExactly`, and `traceExactly`;
5. model-only `verify`, `find witness`, and `find counterexample` queries, plus compilation of one
   exact-action execution query without runtime realization;
6. a deterministic lazy Lean planner, `DrivePlan`, and `ExperimentSpec` compilation;
7. one synthetic observation source, composable evidence mapping fixture, qualified trace, and
   auditable observation derivations;
8. promotion of a selected trace to a complete exact pinned regression; and
9. one second small scenario authored without changing the public language or compiler concepts.

Pairwise, t-wise, coverage-guided search, live runtime execution, multiple evidence profiles,
minimization, generated Go wrappers, and remote qualification should follow only after this slice
demonstrates readable authoring and correct artifacts.

## 18. Relationship to the current Lean regression slice

The existing types are useful compiler substrate but not the final authoring surface:

- `ModelTarget` contributes model identity, resources, action projections, property observations,
  and provenance to the semantic catalog and target interfaces;
- `Regression` should split into Behavior plus Query rather than continue accumulating fields;
- `ExpectedProperties` becomes property references in an execution query;
- `ExperimentSpec` remains the environment-independent compiled artifact, expanded only as concrete
  runtime and evidence needs are demonstrated;
- canonical JSON, deterministic compilation, structured errors, and inspectability remain required;
  and
- the bounded Nexus pilot becomes one exact query derived from a broader behavior space.

Replacement preserves the useful compiler invariants and tests while introducing the new languages
on small synthetic models. When the new languages land, the old `Regression`,
`ExpectedProperties`, and callback-bearing `ModelTarget` authoring structures are removed rather
than retained as a compatibility facade or second public path.

## 19. Alternatives considered

### One combined DSL

Combining setup, behavior, properties, planning, and evidence looks concise for one scenario but
creates a shallow, ever-growing interface. It prevents independent property reuse, obscures claim
strength, and mixes semantic truth with runtime observability. Rejected.

### Separate Behavior and Drive DSLs

A procedural Drive DSL makes exact scripts obvious but duplicates action vocabulary, ordering,
bounds, and identity. Promotion and replay require translation between semantic intent languages.
Rejected for ordinary authoring. A generated `DrivePlan` remains necessary as an artifact and
framework escape hatch.

### One untyped global trace vocabulary

Global names make cross-domain properties easy to write but allow meaningless references and defer
errors until planning or execution. Rejected in favor of required semantic capabilities and checked
model composition.

### Profile-specific properties

Letting properties mention logs, history, spans, or in-process facts makes live checking direct but
duplicates the same semantic claim for every environment. Rejected in favor of separate observation
mappings.

### Two authoring facades over one algebra

An exploratory facade and an exact facade may be added later as syntax sugar if usability testing
justifies them. Both must lower to the same Behavior constraint algebra and must not introduce new
semantics.

### One meaning of exact

Using one `exactly` operator for both a controllable action order and a complete semantic execution
looks concise but hides whether model outcomes may vary. Rejected in favor of distinct
`actionsExactly` and `traceExactly` restrictions in the same Behavior language.

### Unrestricted property functions

Raw Lean predicates maximize expressiveness but cannot be inspected, planned, serialized, promoted,
or checked for undeclared capability access. Rejected for the portable core. An explicitly
non-portable expert escape hatch remains below the ordinary DSL.

## 20. Implementation allocation

The interview split implementation requirements into three Flow specs instead of duplicating a
large acceptance list in this shared design:

1. `fn-3-umpire-semantic-authoring-and-planning` owns vocabulary, capabilities, Property, Behavior,
   Query, bounded planning, `DrivePlan`, `ExperimentSpec`, clean replacement of `Regression`, the
   first cross-domain scenario, and the second-scenario reuse proof.
2. `fn-4-umpire-observation-and-semantic-verdicts` owns the Observation language, qualified traces,
   composable mapping plans, field disposition, derivations, ambiguity, independent property
   verdicts, and strict summaries. It depends on `fn-3`.
3. `fn-5-umpire-discovery-promotion-and-artifact` owns the checked-in glossary, discovery commands,
   complete regression promotion, artifact compatibility, and named migrations. It depends on
   `fn-3` and `fn-4`.

Together they must demonstrate the complete offline semantic loop without a live Temporal server.
No individual spec may weaken the shared purity, qualification, determinism, identity, or
provenance rules to make its local implementation easier.

The package extraction accepted in section 21 is a focused prerequisite follow-up to `fn-3`. Before
work begins on `fn-4` or `fn-5`, their task graphs must depend on that extraction so new declarations
do not extend the namespace being removed.

## Resolved via Project Docs

- The existing regression slice already targets Lean/Umpire tool developers, adds no Temporal
  server behavior, and compiles an environment-independent artifact without starting Temporal
  (`.flow/specs/fn-1-lean-regression-dsl-and-nexus.md:13`).
- The completed slice intentionally established only the authoring/compiler seam before broader
  exploration, execution, and evidence work
  (`.flow/specs/fn-1-lean-regression-dsl-and-nexus.md:23`).
- Nexus operations are cross-boundary, potentially asynchronous operations with status, result,
  callback, and cancellation behavior (`docs/architecture/nexus.md:3`), while cancellation delivery
  is durably retried until success, permanent failure, or timeout (`docs/architecture/nexus.md:310`).
- This repository is the Temporal server; ordinary Workflow, Activity, and Worker authoring belongs
  in supported SDK languages rather than this model DSL (`README.md:64`).

## Resolved via Codebase

- `Umpire.Core` owns the reusable semantic trace/kernel vocabulary and checked target composition.
- `Umpire.Property`, `Umpire.Behavior`, and `Umpire.Search` are independent siblings over Core;
  `Umpire.Query` is the first module that combines the three concerns.
- `Umpire.Artifact` owns the portable `DrivePlan` and `ExperimentSpec`, while `Umpire.Planning`
  consumes checked queries and produces those artifacts with private completion authority.
- `Umpire.Examples.Switch` is the domain-neutral reuse example. The Nexus caller-closure adapter and
  shared inspector live at `Temporal.Umpire.NexusCallerClosure` and `Temporal.Umpire.Inspect`.
- `UmpireTests`, `TemporalUmpireTests`, and `temporal-umpire-inspect` are the only focused Lake
  regression targets. The repository's top-level Makefile remains the stable verification surface.
- The checked-in target-state fixtures retain the pre-move inspector bytes except for exactly two
  source-path substitutions: `Temporal/Experiment/SwitchScenario.lean` to
  `Umpire/Examples/Switch.lean`, and `Temporal/Experiment/NexusCallerClosure.lean` to
  `Temporal/Umpire/NexusCallerClosure.lean`. No other pre/post migration delta was accepted.

## 21. Lean package and namespace architecture

This section records the accepted package split as of 2026-08-25. It refines code ownership without
changing the language semantics defined above.

### 21.1 One Umpire library with independently importable modules

Reusable authoring and planning abstractions live in one Lake library named `Umpire`. The library
uses vertical modules for each DSL rather than horizontal syntax, validation, evaluation, and
serialization layers. Ordinary authors import only the language they need:

```lean
import Umpire.Property
import Umpire.Behavior
import Umpire.Query
```

Public declarations remain concise under the root `Umpire` namespace, such as
`Umpire.PropertyDeclaration` and `Umpire.CheckedBehavior`. The module owns the interface without
forcing every public type into a second namespace such as `Umpire.Property.Declaration`.

The module split is physical as well as logical. Each substantial language owns a directory behind
its stable public facade:

```text
Umpire/
  Property.lean              # imports Umpire.Property.Language
  Property/
    Language.lean
    Tests.lean
    ImportTests.lean
  Behavior.lean              # imports Umpire.Behavior.Language
  Behavior/
    Language.lean
    Tests.lean
    ImportTests.lean
  Query.lean                 # imports Umpire.Query.Language
  Query/
    Language.lean
    Tests.lean
  Planning.lean              # imports Umpire.Planning.Engine
  Planning/
    Engine.lean
    Tests.lean
    VisibilityTests.lean
```

The facade is the external seam: callers continue to write `import Umpire.Property`, for example,
and do not need to know how that language is arranged internally. Core, Search, and Artifact remain
shared support modules outside a DSL directory until they acquire multiple cohesive internals. This
keeps the packages vertical; there is no cross-language `Syntax`, `Validation`, or `Evaluation`
directory.

The old `Temporal.Experiment.*` namespace and module tree are removed in the same change. No aliases,
re-exporting compatibility facade, or second authoring path remains.

### 21.2 Module dependency direction

The public modules form this acyclic dependency order:

1. `Umpire.Core` owns declaration identity, vocabulary metadata, semantic values and traces,
   transition kernels, capability composition, and checked targets.
2. `Umpire.Property` depends only on Core and owns property declarations, checking, denotation,
   evaluation, diagnostics, and canonical representation.
3. `Umpire.Behavior` depends only on Core and owns setup, action, ordering, occurrence, exact-action,
   and exact-trace constraints together with checking, diagnostics, and canonical representation.
4. `Umpire.Search` depends only on Core and owns shared bounds, strategies, budgets, tie breaking,
   policies, and deterministic selection metadata. These are planning concerns used by Query rather
   than a fourth authoring DSL.
5. `Umpire.Query` depends on Property, Behavior, and Search. It is the first layer allowed to combine
   checked properties with a checked behavior.
6. `Umpire.Artifact` depends on Query and owns the portable `DrivePlan` and `ExperimentSpec` data and
   their canonical representations.
7. `Umpire.Planning` depends on Query and Artifact and owns planner kernels, enumeration, private
   termination authority, planning outcomes, and results.

Small canonical JSON and ordering primitives that genuinely repeat may live in
`Umpire.Internal.Canonical`. This is an internal seam, not another authoring interface. Each DSL
continues to own the canonical meaning and structured errors of its declarations.

`Umpire` never imports `Temporal`, Nexus, or a runtime/evidence implementation. The one-way
dependency is enforceable at the Lean module graph: Temporal may import Umpire; Umpire may not import
Temporal.

### 21.3 Domain and example placement

The two existing scenarios separate according to their real dependencies:

- the domain-neutral switch reuse proof moves to `Umpire.Examples.Switch`;
- the Nexus caller-closure model and proofs move to `Temporal.Umpire.NexusCallerClosure`; and
- the Temporal scenario registry and inspector move to `Temporal.Umpire.Inspect`.

Future Temporal history, event, cluster, and in-process evidence mappings and execution adapters stay
under `Temporal.Umpire.*`. Other domains may supply their own adapters without changing the Umpire
library.

### 21.4 Authoring and execution data flow

A domain adapter declares vocabulary, laws, checked target composition, and a transition kernel
through Core. Property and Behavior check independently. Query combines their checked products with
bounded search intent. Planning consumes the checked query and kernel and emits an inspectable
`ExperimentSpec`. A later runtime may consume that artifact, but runtime types and evidence never
flow back into Property, Behavior, Query, or Planning.

The Observation language remains a separate future vertical module. `Umpire.Observation` will own
domain-independent interpretation and qualification concepts; `Temporal.Umpire.Observation.*` will
own Temporal evidence mappings. Property must never import Observation. A later `Umpire.Verdict`
module may combine a pure property with a qualified semantic trace. This extraction does not create
an empty Observation module before that interface exists.

### 21.5 Clean migration and failure semantics

The extraction is atomic and behavior-preserving. It updates every repository consumer while
preserving declaration identities, format versions, semantic digests, validation order,
deterministic planner ordering, structured error kinds, and the distinction among invalid,
unsatisfiable, budget-exhausted, and verified results.

Planner completion and finalization remain private to `Umpire.Planning`; moving declarations must
not let callers manufacture a verified result. Each DSL retains its own `Except` error family rather
than introducing a generic package-migration error.

Provenance source paths change to their truthful new locations. Canonical artifact bytes may change
where provenance records a moved path, but semantic identities and digests must not change merely
because a file moved. Stale `Temporal.Experiment` imports fail at compile time instead of resolving
through compatibility aliases.

The repository's top-level Makefile remains the only Makefile changed for model build or regression
commands. No model-local Makefile is added or extended.

### 21.6 Verification

Tests follow the new ownership:

- Umpire tests cover Core, Property, Behavior, Query, Artifact, Planning, and the generic switch
  example;
- Temporal Umpire tests cover Nexus target composition, scenario authoring, and inspector output;
- compile-time import tests prove that Property and Behavior do not expose one another or Query,
  while Query deliberately imports both;
- an import-graph check proves that no `Umpire.*` module imports `Temporal.*` or Nexus;
- existing positive and negative declaration tests move without weaker assertions;
- planner tests retain the external-forgery guard and all termination distinctions;
- golden comparisons preserve identities, digests, ordering, and artifact fields while allowing
  only truthful provenance-path changes;
- both the generic switch scenario and the Temporal Nexus scenario compile and plan through the same
  public Umpire interfaces; and
- a stale-import scan requires `Temporal.Experiment` to disappear from Lean sources and model
  documentation.

`make umpire-check-regression` remains the stable user command. Its recipe changes only in the
repository's top-level Makefile to build the renamed Lean targets and Temporal Umpire inspector.

### 21.7 Delivery order

The package extraction should land before Observation, discovery, or promotion work adds more
declarations to the old namespace. It moves and separates the implemented surface without
redesigning `SemanticValue`, adding new DSL semantics, or scaffolding future empty packages.
Generalizing the value representation remains a separate design decision.
