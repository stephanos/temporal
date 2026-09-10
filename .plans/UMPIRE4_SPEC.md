# Umpire 4 specification

This document defines Umpire 4's terms and architecture and is the authoritative list of its
development rules. Supporting designs cite rules by ID. MUST, MUST NOT, SHOULD, and MAY indicate
requirement strength.

## Governance

- **GOV-01 — Stable rule IDs.** New rules MUST receive new IDs. Existing IDs MUST NOT be renumbered
  or reused, even after a rule is retired.
- **GOV-02 — Human approval.** A human MUST approve any deliberate exception to these rules.

The term definitions in this document are normative. Supporting documents MUST use those terms
consistently. Capitalized terms have Umpire-specific meanings. Exact Lean names appear in backticks.

## How Umpire works

Umpire models expected behavior. A Query asks a bounded question about that behavior, and Search
answers it. A Producer lowers checked behavior into a versioned Case containing one bounded Program
and one deterministic Contract. Testpilot's `Prepare` validates that Case against an immutable
Profile without target I/O. A prepared Case can then run repeatedly through an authorized Driver;
every attempt produces one append-only Run and one Verdict. For example, a Contract can require that
a declared Nexus history Observation reaches a correlated completion within a bounded Deadline.

## How the model is organized

### Core concepts

- **Behavior Model.** Lean code under `model/` that describes expected product behavior and
  Temporal's current implementation behavior.
- **Model Definition.** A named, handwritten part of the Behavior Model, such as a state, Action,
  step, or Property. Generated Data and Generated Views are not Model Definitions.
- **Generated Data.** Machine-produced descriptions of API and configuration fields and types. This
  data describes what information exists, not how it affects behavior.
- **Definition ID (`Umpire.DefinitionId`).** A stable dot-separated ID for a Model Definition, such
  as `switch.query.exact-action`. Umpire checks that an ID refers to the expected kind of
  definition. Reordering declarations or editing documentation does not change it.
- **Behavior Fingerprint (`Umpire.BehaviorFingerprint`).** A value computed from the
  behavior-affecting parts of a Model Definition. It changes with behavior, but not with
  documentation, source location, or source order.
- **Capability (`Umpire.Capability`).** A named behavior one model component requires and another
  supplies. A `Umpire.Provider` supplies one, a `Umpire.Law` states what every supplier must
  satisfy, a `Umpire.LawProof` discharges one, a `Umpire.Meaning` fixes the interpretation, and a
  `Umpire.Connector` joins two domains explicitly.
- **Known Gap (`Umpire.KnownGap`).** A missing or unsupported Capability, input, interpretation, or
  claim. A Known Gap limits what a Case or Run can prove. A hole in a mapping's source domain is a
  `Umpire.UnmappedSource`, which is not a Known Gap.
- **Implementation Link (`Umpire.ImplementationLink`).** Lean code that explicitly connects product
  behavior in `Temporal.Feature` to corresponding implementation behavior in `Temporal.System`
  without merging their descriptions.
- **Producer.** A compiler or conforming client that creates a versioned Case. Lean is the first
  Producer, but the Case format and Go runtime do not depend on Lean. `Umpire.Case.Compiler` is
  Umpire's own Producer surface.
- **Case.** Exactly one Program and one Contract, with version, stable identity, and generic opaque
  provenance.
- **Provenance (`Umpire.Provenance`).** The producer-owned identity bytes inside a Case:
  `Umpire.Provenance.DefinitionBinding` rows and `Umpire.Provenance.KnownGap` rows that tie the Case
  back to the Model Definitions it came from. The runtime reads none of it.
- **Program.** A bounded acyclic graph of typed instructions in controller, workflow, activity, or
  Nexus-handler entrypoints. One instruction kind is an Opcode.
- **Contract.** A finite set of deterministic safety and bounded-liveness Rules over Run Events and
  declared Observations.
- **Profile.** An immutable authorization and environment snapshot containing a descriptor Catalog,
  symbolic role policy, physical environment bindings, Opcodes, and independent Program and
  Contract ceilings. A binding authorizes no Opcode by itself.
- **Driver.** The environment-owned implementation of authorized side effects. Server and worker
  authority remain separate even when composed behind one Driver.
- **Prepared Case.** The immutable result of static Case, Program, Contract, descriptor, and Profile
  admission. Symbolic resources are resolved from the snapshotted Profile, while its source Case
  remains symbolic. It contains no live client, credential, worker, or Run state.
- **Run Event.** One immutable, monotonically sequenced fact appended by the Executor.
- **Slot.** Private immutable single-assignment execution data. Slot opacity does not make declared
  response projections secret; only declared Observations enter Contract evidence.
- **Observation.** A declared typed value attached to a Run Event and available to the Contract.
- **Verdict.** The three-valued conclusion `satisfied`, `violated`, or `inconclusive`, with rule
  states and supporting Run Event sequences. The model-side answer a Query endpoint produces is
  `Umpire.PropertyEndpointAnswer`, in the same three values.

### Where things live

- **`Umpire`.** Reusable Lean tools for authoring and checking models, answering Queries, and
  producing Cases. It contains no Temporal-specific behavior. Its owners are `Umpire.Model`,
  `Umpire.Property`, `Umpire.Scenario`, `Umpire.Query`, `Umpire.Search`, `Umpire.Evidence`,
  `Umpire.Case`, `Umpire.Provenance`, `Umpire.Variations`, `Umpire.Exploration`,
  `Umpire.Promotion`, `Umpire.Artifact`, `Umpire.ImplementationLink`, and `Umpire.Inventory`.
- **`Testpilot`.** The canonical name for running behavior through Temporal and Workers. The
  Testpilot protobuf closure rooted at
  `proto/internal/temporal/server/api/testpilot/v1/case.proto` is the wire authority;
  `Testpilot.Protocol` exposes its generated Lean
  declarations, `Testpilot.Authoring` constructs them through a context-safe producer-neutral API,
  and `Testpilot.ProtoJSON` owns the single library-backed serialization policy. The shared Go
  runtime admits and executes Cases through a caller-owned Driver and evaluates their Contracts.
  `Temporal.Testpilot` supplies the Lean Producers.
- **`Shared`.** Semantics both a Lean Producer and the Go runtime must agree on, such as
  `Shared.CorrelatedObligation` and `Shared.CorrelatedProjection`. It imports neither `Umpire` nor
  `Temporal`.
- **`Temporal.Feature`.** Product behavior visible to users and SDKs, independent of the current
  implementation.
- **`Temporal.System`.** Behavior of the current Temporal implementation, configuration, and
  runtime.
- **`Temporal.API`.** Generated API field and type information from Protobuf and gRPC definitions.
- **`Temporal.DynamicConfig`.** Generated configuration field and type information.

### Purpose and scope

- **SCP-01 — Temporal-driven scope.** Umpire MUST include only capabilities required by a concrete
  Temporal use case in modeling, Regression, Exploration, Case execution, or verification.
- **SCP-02 — Reusable core.** Reusable `Umpire` code MUST NOT contain Temporal-specific names,
  dependencies, or fixtures.
- **SCP-03 — Lean Behavior Model.** All Behavior Model code MUST be written in Lean and live under
  `model/`.
- **SCP-04 — Complement specialized tests.** Umpire SHOULD complement specialized unit, race,
  persistence, schema, authorization, performance, and handler tests rather than replace them.

### Source of truth

- **SEM-01 — Model authority.** For behavior covered by Umpire, the Behavior Model MUST be the only
  source of truth. Generated code, Artifacts, runtimes, Evidence mappings, and checker adapters MUST
  NOT add or override behavior.
- **SEM-02 — Model Definitions.** Handwritten `Temporal.Feature` and `Temporal.System` Model
  Definitions MUST be the only sources of product and implementation behavior.
- **SEM-03 — Generated Data.** Generated `Temporal.API` and `Temporal.DynamicConfig` declarations
  MUST NOT define behavior until a Model Definition interprets them.
- **SEM-08 — Explicit Implementation Link.** A dedicated Implementation Link MUST connect
  `Temporal.Feature` product behavior to `Temporal.System` implementation behavior. Declaration
  order and implicit selection MUST NOT create that connection.
- **SEM-16 — Case authority.** One admitted Case MUST be authoritative for its exact bounded Program
  and Contract. Runtime code MUST NOT add scenario behavior, verification clauses, implicit retry,
  or undeclared evidence.
- **SEM-17 — Evaluator authority.** The prepared Contract MUST supply the Monitor used during
  execution and MUST use the same transition semantics for offline evaluation. Expiry is evaluated
  before transitions at every event, bounded captures are rule-local and Run-local, and a proven
  violation MUST remain authoritative despite later cleanup or operational failure.
- **SEM-18 — Producer neutrality.** Lean MUST produce deterministic Cases for model-owned behavior,
  but any conforming client MAY author a Case. A non-Lean Case is not thereby a Behavior Model
  declaration or a claim about any other Case.
- **SEM-19 — One word per concept.** *(drafted by fn-82; awaiting GOV-02 approval.)* Each concept in
  this document MUST have exactly one word, and that word MUST be spelled the same in Lean, in the
  Testpilot protobuf schema, in Go, in the command syntax, and in prose. A second word for a concept
  that already has one is a rename, not a synonym: the old word MUST be removed in the same change
  that introduces the new one. A word MUST NOT name two concepts.
- **SEM-20 — Compound-only retirement.** *(drafted by fn-82; awaiting GOV-02 approval.)* A retired
  word MUST enter the retired-vocabulary gate when, and only when, it is a compound identifier, a
  module path, an executable or Make target name, a macro name, or a snake_case keyword. Bare
  English words such as `Target`, `Behavior`, or `Step` MUST NOT be gated, because the gate also
  matches their lowercase form and would reject ordinary prose. A retired bare keyword MUST instead
  be rejected by the command macro that used to accept it, with a located error naming its
  replacement.

### Enforced module boundaries

- **MOD-01 — `Umpire` independence.** `Umpire.*` MUST NOT directly or transitively import
  `Temporal.*`.
- **MOD-03 — `Temporal.Feature` isolation.** `Temporal.Feature.*` MUST NOT directly or transitively
  import `Temporal.System.*`. Once optional verification exists, imports of it are allowed only for
  the consumers MOD-05 lists.
- **MOD-05 — Verification isolation.** First-party (repository-owned) Lean modules MUST NOT directly
  or transitively import `Temporal.Verify` or `Umpire.Verify.Veil` unless they are one of the
  declared opt-in consumers. No such module exists in the tree, so `ModelLint` reserves nothing for
  them; the reservation returns with the modules.
  *(planned: fn-24-lean-native-verification-receipts-and)*
- **MOD-09 — `Shared` independence.** `Shared.*` MUST NOT directly or transitively import `Umpire.*`
  or `Temporal.*`.
- **MOD-10 — `Temporal.System` isolation.** `Temporal.System.*` MUST NOT directly or transitively
  import `Temporal.Feature.*`. The only exception is `Temporal.System.Nexus.ImplementationLink`.
- **MOD-11 — Executable enforcement.** `make lint-model` MUST enforce MOD-01, MOD-03, MOD-09, and
  MOD-10 across the complete first-party Lean import graph, and MOD-05 once its modules exist.
- **MOD-12 — Public Testpilot facade.** The public execution sequence MUST be exactly
  `testpilot.Prepare(case, profile)` followed by `PreparedCase.Run(ctx, driver)`. Scheduler, Recorder, Slot
  storage, and Monitor-factory construction MUST remain internal.
- **MOD-13 — Temporal authority split.** `common/testing/testpilot/temporal/server` MUST supply the authorized
  descriptor catalog and transport prepared unary method/request pairs, returning raw typed
  responses and protocol status. `common/testing/testpilot/temporal/worker` MUST own SDK workflow, activity, and
  Nexus-handler execution plus reserved activation delivery. Neither side may assume the other's
  authority; internal execution owns request construction and response projection.
- **MOD-14 — Internal execution boundary.** Production packages outside Testpilot and its private
  verification package MUST NOT import `common/testing/testpilot/internal/execution`; Driver adapters
  depend only on the public Testpilot facade.
- **MOD-15 — Resolvable glossary.** *(drafted by fn-82; awaiting GOV-02 approval.)* Every dotted
  Lean name this document cites in backticks MUST name a module, a namespace, or a declaration that
  exists in `model/`, unless the rule citing it is marked planned and the Flow spec that owns
  delivering it is open. A Go test under `tools/umpire` MUST enforce this against an index built by
  scanning the model tree. Whether each term is defined once and used consistently remains a review
  judgment; the test covers only the mechanical half.

### Module design

- **MOD-02 — Product and system ownership.** `Temporal.Feature` MUST own product-visible behavior.
  `Temporal.System` MUST own implementation mechanisms, configuration interpretation, Evidence
  mappings, and runtime behavior.
- **MOD-04 — Independent product and system modules.** `Temporal.Feature.*` and
  `Temporal.System.*` modules MUST be understandable and testable on their own. Only focused
  Implementation Link modules MAY connect them, subject to MOD-10.
- **MOD-06 — Small public interfaces.** `Umpire.*` modules SHOULD hide checking, Search, Artifact,
  Evidence, and verification machinery behind small, cohesive interfaces.
- **MOD-07 — Clear component boundaries.** Components MUST have narrow responsibilities and
  communicate through explicit contracts rather than each other's internal representations.
- **MOD-08 — Isolated testability.** Each component MUST be testable with fixtures or generic
  examples without the complete `Umpire` pipeline or a running Temporal cluster.

## Model authoring and traces

### Trace concepts

- **Model (`Umpire.Model`).** The checked behavior an author writes. `Umpire.DraftModel` is the
  unchecked author record, `Umpire.checkModel` admits it, and `Umpire.CheckedModel` is the result
  every Property, Scenario, and Query shares. `Umpire.ModelSpec` is the declaration data behind it.
- **Machine (`Umpire.Machine`).** The transition relation of a Model together with the proofs that
  it is the authority for that Model's behavior.
- **Table (`Umpire.FiniteTable`).** The finite row form of a Machine. `Umpire.CheckedTable` is an
  admitted one and `Umpire.TableModelSpec` is its author record.
- **Vocabulary (`Umpire.Vocabulary`).** The enumerated finite domains of a Model: its states,
  Actions, Model Outcomes, and Facts, in a canonical order.
- **Action.** Something an author asks the Model to do, such as closing a Workflow. Requesting an
  Action neither chooses its Model Outcome nor proves that the Action occurred at runtime.
- **Model Outcome.** The result the Model produces for an Action. It is an expected model result,
  not a runtime result.
- **Step (`Umpire.Step`).** One model step and what it produced: an `outcome`, a `state`, and a list
  of `facts`.
- **Fact.** A claim the Model makes at a Step, such as “the Nexus operation received a
  cancellation.” Logs, spans, RPCs, and records are Evidence used to decide whether the claim held
  during a Run. Facts are the `facts` of a `Umpire.Step`; the Fact domain is a Model's own type.
- **Trace (`Umpire.Scenario.Trace`).** A starting state and a sequence of Steps. It contains no
  runtime Evidence. One position inside a Trace is a `Umpire.ModelCoordinate`; SEM-19 retires that
  name to `TraceAddress`, which the tree has not taken yet.
- **Scenario (`Umpire.Scenario`).** A named, constrained set of Traces. It defines available
  variations and faults but selects no single Trace, and it neither evaluates Properties nor
  determines whether a Trace occurred at runtime. `Umpire.CheckedScenario` is an admitted one.
- **Property (`Umpire.Property`).** A reusable pass/fail rule over Traces, built from
  `Umpire.PropertyClause` clauses. For example, closing a Workflow cancels its running Nexus
  operation at most once. Guarded alternatives inside one clause are branches.
  `Umpire.CheckedProperty` is an admitted one.
- **Correlated (`Umpire.Property.Correlated`).** A rule tracked separately per operation and
  correlated by an explicit key, so one operation's obligations never discharge another's.
  `Shared.CorrelatedObligation` is the semantics the Lean Producer and the Go runtime share.
- **Unsatisfiable.** A Scenario that allows no Trace. This is an error, not a passing answer.

### Model languages

- **SEM-04 — Separate languages.** Each Lean authoring language—including `Umpire.Property`,
  `Umpire.Scenario`, `Umpire.Query`, and `Umpire.Evidence`—MUST be separate and have a distinct
  purpose.
- **SEM-05 — Pure `Umpire.Property`.** `Umpire.Property` declarations MUST use only Traces and
  Capability Contracts. They MUST NOT depend on implementation Evidence.
- **SEM-06 — Declarative `Umpire.Scenario`.** `Umpire.Scenario` declarations MUST constrain
  allowed Traces. They MUST NOT become step-by-step RPC or runtime scripts.
- **SEM-07 — Model-owned outcomes.** Authors MUST request Actions, while `Umpire.CheckedModel`
  determines their Model Outcomes and resulting states.
- **SEM-09 — Bounded progress.** A Property claiming that something eventually happens MUST state a
  Limit and unit. A finite Run MUST NOT prove an unlimited “eventually” claim.

### Authoring

- **AUT-01 — Approachable authoring.** A Temporal engineer with basic Lean knowledge SHOULD be able
  to write ordinary Model Definitions without understanding Umpire's implementation details.
- **AUT-02 — Explicit meaning.** Authoring interfaces MUST make states, Actions, Model Outcomes,
  relations, Limits, faults, Capability Contracts, Known Gaps, and unsupported cases explicit.
- **AUT-03 — Checked declarations.** Public declarations MUST be checked before Search or a Run. Failures SHOULD report errors at the relevant source location.
- **AUT-04 — Stable IDs.** Every public Model Definition MUST have a stable, dot-separated Definition
  ID that is checked against the expected definition kind. Source order and documentation MUST NOT
  affect it.
- **AUT-05 — Cross-language data.** Anything used in Search, Artifacts, Promotion, or
  cross-language Runs MUST be serializable data that Lean can interpret. It MUST NOT depend on
  in-process callbacks.
- **AUT-06 — Explicit composition.** Competing providers MUST be selected explicitly, and
  cross-domain relationships MUST be connected explicitly. Declaration order and Lean's automatic
  instance search MUST NOT choose behavior.
- **AUT-07 — Single authoring path.** `Umpire.Property`, `Umpire.Scenario`, and `Umpire.Query` MUST
  be the only public languages for declaring Properties, Scenarios, and Queries.
  `Umpire.CheckedModel` is their shared model representation, not an authoring language. Wrappers
  MUST NOT provide another way to define behavior.
- **AUT-08 — Finite Model adapter.** Authors SHOULD use the proof-carrying
  `Umpire.FiniteMachine` adapter when a complete finite Model has enumerators that define its
  authoritative behavior. The adapter derives membership relations, completeness support, and
  exact finite Search. Authors MUST still provide ordered semantic domains, encoders, enumerators,
  evidence that enumerated values stay within those domains, and evidence that every enumerated
  Action is executable. As an expert alternative, authors MAY construct `Umpire.Machine`
  directly for Models whose authority is specified independently. Both paths MUST produce an
  `Umpire.DraftModel` and pass it to `Umpire.checkModel`. `Umpire.FiniteMachine` MUST NOT
  introduce another Property, Query, Scenario, or macro language.
- **AUT-09 — Macro-derived finite domains.** *(drafted by fn-80; approved 2026-09-10 under GOV-02.)*
  AUT-08's "author-provided" includes a domain a command macro derives from the author's
  own declarations: the ordered domains, encoders and enumerators an authoring macro elaborates from
  the constructors of an enum-like inductive the author named are author-provided, not inferred. The
  macro MUST derive them from declarations the author wrote and MUST NOT admit a spelling the author
  did not declare. Completeness, domain membership and Action executability MUST still be discharged
  against the same authorities AUT-08 names; a macro MUST NOT weaken or assume them.

## Search, Limits, and Artifacts

### Search and Artifact concepts

- **Limit (`Umpire.Limit`).** A typed ceiling on one stage, in one `Umpire.LimitUnit`: `steps`,
  `actions`, `logicalTime`, `search`, or `plans`. It limits no other stage. `Umpire.Limits` is the
  set a Query carries.
- **Deadline.** The bounded-time ceiling one Contract Rule declares, counted in the Run Events that
  Rule has evaluated since its last transition, or in elapsed milliseconds on the host that
  produced the Run. A Deadline bounds a Rule; a Limit bounds a stage.
- **Limit Reached.** The result reported when a stage reaches a Limit before answering its
  question. It proves neither a negative answer nor that the search was exhaustive.
- **Exhaustive Search.** A search that checks every candidate the exact Scenario and Limits allow.
  If it finds no candidate, it proves absence only within those Limits.
- **Query (`Umpire.Query`).** A bounded question about a Model, within explicit Limits. Its
  `Umpire.Query.Form` is `verify`, `find`, `findViolation`, or `pick`; its `Umpire.Query.Ending`
  says which Traces count as complete. `Umpire.CheckedQuery` is an admitted one.
- **Search (`Umpire.Search`).** Answering a Query. `Umpire.SearchView` is the enumerable view of a
  Model it walks, `Umpire.SearchStats` records the work it spent, and `Umpire.PlanResult` is what it
  returns.
- **Plan (`Umpire.Plan`).** The planned model-level test a Search chose: a `Umpire.Plan.Steps`
  sequence retained for scenario-neutral catalog and reviewed-promotion use. It is generated
  instructions, not an authoring language, and Testpilot does not accept it.
- **Variations (`Umpire.Variations`).** A finite space of authored variations over one Model.
  `Umpire.VariationSpace` is the declared space Exploration draws candidates from.
- **Artifact (`Umpire.Artifact`).** Immutable, versioned, inspectable data exchanged across
  components, languages, and processes. Artifacts cannot define model behavior.
- **Artifact Checksum.** A reproducible checksum over all Artifact content in canonical order,
  excluding the checksum field itself. It identifies one exact Artifact; it is not a Definition ID
  or Behavior Fingerprint.
- **Generated View.** A deterministic representation of an Artifact, such as a Go test or
  documentation. It is bound to the source Artifact Checksum and cannot define behavior.
- **Inventory (`Umpire.Inventory`).** The generated status and gap catalog, published as
  `model/INVENTORY.md`. It reports what the tree contains; it defines nothing.

### Search and Limit rules

- **PLN-01 — Explicit Limits.** Separate, explicit, typed Limits MUST govern each stage: checking
  whether a Scenario allows a Trace, Search, a Run, Evidence evaluation, and failure
  reduction.
- **PLN-02 — Deterministic selection.** Identical Model Definitions, model inputs, Limits, strategy,
  and seed MUST produce identical Plans and Artifact Checksums.
- **PLN-03 — Exhaustive means complete.** A search declared Exhaustive MUST fail if it cannot check
  every candidate.
- **PLN-04 — Limit Reached is inconclusive.** Limit Reached MUST NOT be treated as proof that no
  trace or counterexample exists.
- **PLN-05 — Unsatisfiable is an error.** A checked `Umpire.Scenario` that admits no Trace MUST
  report `unsatisfiable`, never a passing answer.
- **PLN-06 — Generated Plan.** A `Umpire.Plan.Steps` MUST contain generated instructions. It MUST
  NOT be an authoring language or Evidence that a Run occurred.

### Artifact rules

- **ART-01 — Versioned formats.** Persisted Artifacts MUST use versioned, inspectable formats and
  deterministic serialization.
- **ART-02 — Model binding.** Artifacts MUST carry Definition IDs, Behavior Fingerprints, their own
  Artifact Checksums, source information, Known Gaps, and enough compatibility data for stale readers
  to reject them.
- **ART-04 — Safe format changes.** Readers MUST reject unknown major versions and unknown fields
  that could affect behavior. Changing the meaning of old data requires a named, deterministic
  migration.
- **ART-07 — Generated Views.** The same source Artifact MUST always produce the same Generated View.
  Generated Go tests and documentation MUST be bound to their source Artifact Checksums and MUST NOT
  be editable sources of model behavior.
- **ART-09 — Closed Case format.** A Case MUST contain exactly one versioned Program and one
  Contract, stable IDs, generic opaque provenance, typed roles, paths, Slots, Observations,
  independent limits, and no callback, client, credential, endpoint, or executable. Umpire
  Producers MUST retain their explicit Known Gaps in producer-owned provenance bytes.
  Unknown versions, fields, enum values, instructions, paths, types, crossed references, or
  out-of-policy resources MUST reject before Driver I/O.
- **ART-10 — Immutable preparation.** `testpilot.Prepare` MUST snapshot all admitted Case, Catalog,
  Profile, Program, and Contract data. A Prepared Case MUST be safe for isolated sequential and
  concurrent Runs and MUST expose no mutation path into prepared state.
- **ART-11 — Deterministic Case fixtures.** Lean-produced Case data MUST compare byte-for-byte.
  Runtime values MAY be compared through a named closed projection only when every excluded dynamic
  field is validated structurally. Generic normalization or ignore lists are forbidden.
- **ART-12 — Transactional fixture ownership.** Fixture generation MUST build and validate the
  complete managed tree under a temporary root before comparison or publication. Verification and
  reviewed promotion MUST be separate actions; ordinary tests MUST invoke neither Lean nor rewrite
  fixtures.
- **ART-13 — Explicit environment binding.** Exact Case 1.0 is the only admitted and generated
  format. A resource-free Program MAY have an empty environment; every Program that uses a physical
  resource MUST declare a complete closed graph of symbolic text bindings. The Case owns only symbolic IDs and references; the Profile owns their
  physical namespace, task-queue, and named Nexus endpoint values. Symbolic endpoint IDs are not
  transport addresses. Credentials, gRPC targets, callback authorities, SDK clients, and lifecycle
  configuration remain Driver inputs.
- **ART-14 — Binding identity.** Preparation and Driver identity MUST include the same deterministic
  fingerprint of the complete immutable Profile binding snapshot. Reordering equal bindings MUST
  preserve the fingerprint; changing any binding, including one unused by the Case, MUST change it.
  Environment rebinding MUST NOT change source Case bytes, Behavior Fingerprints, Contract bytes, or
  producer provenance meaning.

## Case execution and verification

### Runtime concepts

- **Run.** The authoritative append-only record of one bounded attempt to interpret a Prepared Case
  through an authorized Driver, including declared Observations, independent cleanup status,
  diagnostics, and its immutable Verdict copy.
- **Executor.** The internal generic interpreter that schedules a Program, owns Slot state and
  effect handles, records Run Events, and performs bounded cleanup.
- **Evaluator.** The verification component that creates one fresh Monitor per Run or evaluates a
  closed Run offline using the same Contract transition semantics.
- **Monitor.** One private Run-local Contract state machine. It returns Continue or Stop but cannot
  dispatch work or mutate a Run. `Testpilot.Correlated.Monitor` is the Lean Monitor for a
  Correlated capability; `Shared.CorrelatedObligation.Monitor` is the semantics it and the Go
  runtime share.
- **Evidence (`Umpire.Evidence`).** Offline evaluation of captured runtime records against a Model.
  `Umpire.Evidence.Reading` is one admitted record and `Umpire.Evidence.PropertyStatus` is what a
  Property's clauses came to under it.
- **Projection (`Umpire.Case.Projection`).** Reading declared Run values into model Steps and
  fields. It is the only seam between a Run and a Model; nothing else in the tree is called a
  projection.
- **Run disposition.** `completed`, `stopped_by_monitor`, or `incomplete`; it is independent of the
  cleanup status and Verdict.
- **Cleanup status.** `succeeded`, `failed`, or `timed_out`; cleanup failure never erases a proved
  violation.

### Runtime rules

- **EVD-01 — Thin runtime.** Runtime and CLI code MUST only prepare Cases, bind authorized Driver
  capabilities, and execute admitted Programs. It MUST NOT independently decide scenario or product
  behavior.
- **EVD-04 — Fail closed.** Missing, ambiguous, conflicting, outdated, unsupported, or causally
  unrelated Evidence MUST NOT establish success or absence.
- **EVD-05 — Independent statuses.** Authoring, Search, a Run, Evidence evaluation, Implementation
  Link application, Property evaluation, and verification MUST each report their own status. A
  status for one stage MUST NOT imply the status of another.
- **EVD-07 — Distributed ordering.** Conclusions about model behavior MUST rely only on
  model-declared order, causal relationships, or record order within one source. They MUST NOT rely
  on synchronized wall clocks.
- **EVD-08 — Complete lifecycle.** Every Run MUST retain attempted instructions, actual generic
  outcomes, declared Observations, diagnostics, disposition, cleanup status, and Verdict.
- **EVD-11 — Generic scheduling.** The Executor MUST dispatch only admitted instruction/context
  pairs, honor dependencies and guards, enforce each bound, preserve single-assignment Slots, and
  add no scenario-specific branch.
- **EVD-12 — Deterministic evaluation.** Monitor transitions MUST process appended events
  synchronously in order, apply expiry before transitions on every event kind, charge declared work
  and capture limits, and fail closed on missing, ambiguous, conflicting, or unsupported values.
- **EVD-13 — Exact support.** Every rule conclusion MUST retain its terminal state and exact
  supporting Run Event sequences. Contracts MUST inspect declared Observations and event fields,
  never private Slots or arbitrary raw payloads.
- **EVD-14 — Safety stop and cleanup.** A proved safety violation MUST stop new controller dispatch
  and activation reservations, cancel and drain owned work within bounds, and then execute cleanup
  through a fresh bounded context. Cleanup remains independent from the Verdict.
- **EVD-15 — Immutable closure.** After `Run` returns, late completions, quarantine release, Driver
  diagnostics, caller mutation, and another Run MUST NOT change either returned Run or Verdict.
- **EVD-16 — Activation cancellation.** Cancellation MUST address reserved activation handles and
  already-started SDK commands at activation scope, including delivery that races Stop.
- **EVD-17 — Server/worker composition.** Server Drivers MAY transport only authorized prepared unary
  method/request pairs and return their raw typed response and protocol status. Internal execution
  MUST construct requests, apply declared response projections, assign Slots, and emit Observations.
  Worker Drivers MUST use Temporal SDK APIs for workflow, activity, and Nexus-handler entrypoints.
  Runtime Cases never supply credentials or transport metadata.
- **EVD-18 — Facade conformance.** Regression MUST exercise exactly the satisfied, violated,
  inconclusive, static-preparation-rejection, cleanup-failure-after-proved-violation, and
  cross-Run-isolation facade classes while leaving focused concurrency, cancellation, path,
  cardinality, fuzz, and lifecycle tests independent.
- **EVD-19 — Static Driver validation.** After complete Driver identity agreement and before Monitor
  creation or `Driver.Open`, `PreparedCase.Run` MUST call the Driver's no-I/O validation hook over
  immutable prepared metadata. Validation failure MUST create no Session, Run, Verdict, worker
  registration, or target effect. Temporal validation MUST compare binding references, not merely
  their currently resolved text. The Driver MUST obtain physical resource names solely from the
  immutable Profile binding snapshot.
- **EVD-20 — Driver-realized faults.** *(drafted by fn-80; approved 2026-09-10 under GOV-02.)*
  A deliberate outage MUST be a declared instruction of the version-one instruction table, MUST name
  a role the Program declares, and MUST be authorized by a Profile capability like any other
  instruction. A Driver MUST record exactly one `RUN_EVENT_KIND_FAULT_INJECTED` Run Event per
  realized instruction, carrying the role and the kind it realized, and MUST record none for an
  instruction it refused or could not complete; such an instruction is a failed outcome plus a
  Driver invariant diagnostic. A requested fault proves nothing until the Run carries that event
  for it. A Program that declares a fault MUST hold resources no other Run shares, so an outage it
  asks for cannot reach another Run.
- **EVD-21 — Deadline units.** *(drafted by fn-80; approved 2026-09-10 under GOV-02.)*
  A bounded-liveness rule MUST declare exactly one positive Deadline bound. `rule_events` counts the
  Run Events the rule evaluated since its last transition and is the bound a conclusion may rest on,
  because it counts only what the Run recorded. `elapsed_milliseconds` remains admitted and is
  host-clock dependent, so a Case whose verdict must not depend on the machine that produced it
  SHOULD declare the event count instead; EVD-07 already forbids resting a conclusion on
  synchronized wall clocks. Both bounds MUST be ticked through one shared helper, so the online and
  the offline evaluation of the same Run answer identically. The counter MUST reset on each
  transition into a new state, MUST stop once the rule is terminal, and MUST freeze with every other
  rule effect once execution becomes incomplete, so no expiry is ever concluded from a truncated Run.
  A Correlated rule counts admitted operation steps and MUST NOT fall back to either.

## Exploration, replay, and promotion

### Exploration concepts

- **Exploration (`Umpire.Exploration`).** Model-owned selection from a declared `Umpire.Variations`
  space to find useful Plans or counterexamples. It is exhaustive only when it covers the declared
  finite space within its Limits.
- **Regression.** A permanent named `Umpire.Query` retained to detect recurrence of known behavior
  independently of Exploration Limits.
- **Promotion (`Umpire.Promotion`).** Re-answering a Query against the Behavior Model using exactly
  the referenced Definition IDs, Behavior Fingerprints, Properties, Scenario, and Limits, so a
  discovered failure can be reviewed and kept as a Regression.

### Exploration rules

- **EXP-01 — Shared model.** Regression Runs, model checking, Exploration, fuzzing, promotion
  replay, and canary selection MUST reuse the same Model Definitions and
  `Umpire.Property` declarations.
- **EXP-02 — Model-owned Exploration.** The model MUST define the exploration space, allowed input
  variations, coverage criteria, candidate scoring, and selection rules. Orchestration MAY execute
  and store the resulting batches.
- **EXP-03 — Bounded fuzzing is not exhaustive.** Runtime fuzzing stopped by a time or work Limit
  MUST NOT claim exhaustive coverage.
- **EXP-04 — Pinned Regressions.** Known Regressions MUST run independently of Exploration Limits.
- **EXP-05 — Reviewed promotion.** Before human review for promotion to a permanent Lean Regression,
  a discovered failure MUST be reproduced at runtime, minimized in model terms, and re-answered
  against the Behavior Model through `Umpire.Promotion` using its exact referenced identities.

## Verification, CLI, and claims

### Verification and claim concepts

Nothing in this section exists in the tree. Every rule below is a design commitment whose owning
spec is named, and each is enforceable only once that spec delivers it.

- **`Temporal.Verify`.** Optional Temporal-specific checker integration. It does not define
  behavior. *(planned: fn-24-lean-native-verification-receipts-and)*
- **`Umpire.Verify.Veil`.** Optional reusable Veil checker integration. Ordinary models and runtimes
  do not import it. *(planned: fn-25-optional-callerclosure-veil-binding-and)*

### Optional verification rules

- **VER-01 — Lean-native default.** Lean-native checking MUST be the default verification path.
  *(planned: fn-24-lean-native-verification-receipts-and)*
- **VER-02 — Explicit opt-in.** Each model family and each `Umpire.Property` declaration MUST opt in
  explicitly to optional checker integration.
  *(planned: fn-25-optional-callerclosure-veil-binding-and)*
- **VER-03 — Checked link.** Every representation used by an optional checker MUST have an explicit,
  checked link to an `Umpire.CheckedModel` and an `Umpire.Property` declaration.
  *(planned: fn-25-optional-callerclosure-veil-binding-and)*
- **VER-04 — Complete verification receipts.** Verification receipts MUST expose source
  information, Definition IDs, Behavior Fingerprints, assumptions, Limits, Known Gaps, and the basis
  of the claim. *(planned: fn-24-lean-native-verification-receipts-and)*
- **VER-05 — Replayed counterexamples.** A checker counterexample MUST be re-answered against the
  Behavior Model through `Umpire.Promotion` before it can support a claimed model violation or a
  promotion to a Regression. *(planned: fn-24-lean-native-verification-receipts-and)*
- **VER-06 — Distinct trust.** Kernel proofs, reconstructed proofs, trusted solvers, search within
  Limits, Runs, and concrete replay MUST be recorded as distinct bases for a claim; they are not
  interchangeable. *(planned: fn-24-lean-native-verification-receipts-and)*

### CLI, environment, and claim rules

- **CLI-01 — Code location.** Umpire CLI code MUST either live under `tools/umpire` or be imported
  from `temporal/tools/common`.
- **CLI-02 — Thin interface.** User-facing tools MAY select declarations and tighten declared
  Limits. They MUST NOT invent `Umpire.Scenario` declarations or broaden model-declared Limits.
- **CLI-03 — Inspectability.** User-facing tools SHOULD provide consistent commands to list and
  explain named Properties, Scenarios, Queries, Plans, Explorations, verification checks, Artifacts, and
  Results.
- **CLI-04 — Testpilot command scope.** Case fixture commands MAY build Lean Producer tools and
  atomically promote reviewed deterministic fixtures. Ordinary execution exposes no replacement
  resident service, scenario-specific adapter, or public Monitor selector.
- **QLF-01 — Environment settings.** Environment profiles MAY provide endpoints, credentials,
  namespaces, permissions, resources, and adapters, provided they do not change modeled behavior.
- **QLF-02 — Environment controls.** Each non-local environment MUST explicitly own its authorization
  controls, rate and concurrency limits, cleanup and isolation responsibilities, rollout policy,
  and limits on possible impact.
- **QLF-03 — Complete claims.** Every claim made from a Run MUST expose its environment, Evidence
  policy, Limits, the basis of the claim, Known Gaps, cleanup outcome, and Behavior Fingerprints.
- **QLF-05 — Per-Run decisions.** A Testpilot decision MUST retain Run disposition, cleanup
  status, and Verdict separately. A satisfied Verdict does not hide operational or cleanup failure;
  a proved violated Verdict remains violated after later cleanup failure; every unresolved Contract
  rule closes inconclusive.

## Appendix: retired rules

These rules no longer bind. GOV-01 keeps their IDs so a design that cites one still resolves, and
so no future rule reuses the number. Each names what superseded it.

- **SEM-10 — Retired: portable interpreter seam.** Superseded by SEM-16 through SEM-18. The deleted
  portable interpreter is historical and MUST NOT be restored as an execution recommendation.
- **SEM-11 — Retired: portable plan authority.** Superseded by SEM-16. The deleted portable plan
  format has no runtime authority.
- **SEM-12 — Retired: plan-local claim scope.** Superseded by SEM-17 and SEM-18.
- **SEM-13 — Retired: independently validated model scope.** Superseded by Case provenance and
  Profile admission under SEM-16 and ART-09.
- **SEM-14 — Retired: external portable obligations.** Superseded by explicit Case Known Gaps and
  the closed Contract vocabulary under SEM-17.
- **SEM-15 — Retired: Lean portable plan compilation.** Superseded by Lean Case production under
  SEM-18.
- **ART-03 — Retired: executable Test Plan.** `Umpire.Plan` is no longer a runtime input;
  ART-09 defines the replacement Case Artifact.
- **ART-05 — Retired: same Test Plan.** Superseded by immutable prepared Case identity under ART-10.
- **ART-06 — Retired: executable trace closure.** Superseded by complete Program admission under
  ART-09 and immutable Run closure under EVD-15.
- **ART-08 — Retired: closed portable evaluation.** Superseded by the Case Contract under ART-09
  and SEM-17.
- **EVD-02 — Retired: separate legacy Run Evaluation.** Superseded by the authoritative Contract
  Evaluator under SEM-17 and EVD-12.
- **EVD-03 — Retired: legacy Evidence normalization.** The Case Contract consumes only immutable
  Run Events and declared typed Observations under EVD-12.
- **EVD-06 — Retired: legacy Execution Receipts.** Run Event source identity, causal references,
  outcome, and declared Observations replace the retired receipt scheme.
- **EVD-09 — Retired: legacy Evidence Links.** Contract support is represented by exact supporting
  Run Event sequences under EVD-13.
- **EVD-10 — Retired: strict portable interpretation.** Superseded by Contract rules EVD-12 through
  EVD-14.
- **QLF-04 — Retired: per-Test local decisions.** The legacy Local Canary `pass`, `fail`, and
  `inconclusive` decision rule is superseded by the separate Testpilot statuses under QLF-05.
