# Umpire 4 specification

This document defines Umpire 4's terms and architecture and is the authoritative list of the rules
every part of Umpire shares, whatever language a Model is written in. Supporting designs cite rules
by ID. MUST, MUST NOT, SHOULD, and MAY indicate requirement strength.

A front end's own specification adds the rules that bind only that front end: its module graph, its
authoring surface, and its optional checkers. It may add rules; it MUST NOT relax one stated here.
Front end rules share this document's ID space (GOV-01), and the IDs this document leaves to front
ends are listed under "Front end rules". Where a rule below has a restatement or amendment still
awaiting GOV-02 approval, that is marked, and the drafted text is kept with the front end that
drafted it.

## Governance

- **GOV-01 — Stable rule IDs.** New rules MUST receive new IDs. Existing IDs MUST NOT be renumbered
  or reused, even after a rule is retired.
- **GOV-02 — Human approval.** A human MUST approve any deliberate exception to these rules.

The term definitions in this document are normative. Supporting documents MUST use those terms
consistently. Capitalized terms have Umpire-specific meanings.

## How Umpire works

Umpire models expected behavior. A Query asks a bounded question about that behavior, and Search
answers it. A Producer lowers checked behavior into a versioned Case containing one bounded Program
and one deterministic Contract. Testpilot's `Prepare` validates that Case against an immutable
Profile without target I/O. A prepared Case can then run repeatedly through an authorized Driver;
every attempt produces one append-only Run and one Verdict. For example, a Contract can require that
a declared Nexus history Observation reaches a correlated completion within a bounded Deadline.

## How the model is organized

### Core concepts

- **Behavior Model.** The Model Definitions that describe expected product behavior and Temporal's
  current implementation behavior. They live under `model/` and are written in a front end.
- **Front end.** A language and toolchain Model Definitions are authored in, with its own
  specification. A front end's Models reproduce the answers every other front end gives for the
  Models both declare: tables, Definition IDs, refinement rows, Behavior Fingerprints and Case
  bytes. *(drafted 2026-09-30; awaiting GOV-02 approval.)*
- **Model IR.** The language-neutral form of a Model: its finite types, pure step functions as
  expression trees, actions and machines, with the source position of every node. Its schema is
  `proto/internal/temporal/server/api/umpire/v1/ir.proto`, its evaluation rules are
  `model/scalav2/SEMANTICS.md`, and an interpreter derives every table, identity and fingerprint
  from it without running front end code. *(drafted 2026-09-30; awaiting GOV-02 approval.)*
- **Model Definition.** A named, handwritten part of the Behavior Model, such as a state, Action,
  step, or Property. Generated Data and Generated Views are not Model Definitions.
- **Generated Data.** Machine-produced descriptions of API and configuration fields and types. This
  data describes what information exists, not how it affects behavior.
- **Definition ID.** A stable dot-separated ID for a Model Definition, such as
  `switch.query.exact-action`. Umpire checks that an ID refers to the expected kind of definition.
  Reordering declarations or editing documentation does not change it.
- **Behavior Fingerprint.** A value computed from the behavior-affecting parts of a Model
  Definition. It changes with behavior, but not with documentation, source location, or source
  order.
- **Capability.** A named behavior one model component requires and another supplies. A provider
  supplies one, a law states what every supplier must satisfy, a meaning fixes its interpretation,
  and a connector joins two domains explicitly.
- **Known Gap.** A missing or unsupported Capability, input, interpretation, or claim. A Known Gap
  limits what a Case or Run can prove. A hole in a mapping's source domain is an unmapped source,
  which is not a Known Gap.
- **Implementation Link.** An explicit connection from product behavior to the corresponding
  implementation behavior, without merging their descriptions. Declaration order and implicit
  selection never create one.
- **Producer.** A compiler or conforming client that creates a versioned Case. The Case format and
  the Go runtime depend on no front end.
- **Case.** Exactly one Program and one Contract, with version, stable identity, and generic opaque
  provenance. *(A restatement awaits GOV-02 approval: structured provenance the runtime does not
  read.)*
- **Provenance.** The producer-owned rows inside a Case that tie it back to the Model Definitions it
  came from: Definition bindings with their Behavior Fingerprints and kinds, sources, Known Gaps,
  correlated rule bindings, and the local-name and model-value fingerprint rows that map each
  Case-local name and short value spelling to what it stands for. Each kind of row is its own list,
  in the order the Producer lists them. The runtime reads none of it.
- **Program.** A bounded acyclic graph of typed instructions in controller, workflow, activity, or
  Nexus-handler entrypoints.
- **Opcode.** The kind of one Program instruction. A Profile authorizes a set of Opcodes; nothing
  else authorizes what a Program may do.
- **Contract.** A finite set of deterministic safety and bounded-liveness Rules over Run Events and
  declared Observations.
- **Rule.** One state machine inside a Contract, with an initial state, finite transitions, terminal
  satisfied or violated states, and, when it is a bounded-liveness Rule, one Deadline. *(An
  amendment awaits GOV-02 approval: a Rule MAY declare typed instance values and a list of Rule
  instances, each evaluated as its own state machine with its own verdict.)*
- **Profile.** An immutable authorization and environment snapshot containing a descriptor Catalog,
  symbolic role policy, physical environment bindings, Opcodes, and independent Program and
  Contract ceilings. A binding authorizes no Opcode by itself. *(A restatement awaits GOV-02
  approval: the Profile also holds instruction defaults and the correlated resource ceilings every
  Case is admitted against, and a Case declares none of those ceilings.)*
- **Driver.** The environment-owned implementation of authorized side effects. Server and worker
  authority remain separate even when composed behind one Driver.
- **Prepared Case.** The immutable result of static Case, Program, Contract, descriptor, and Profile
  admission. Symbolic resources are resolved from the snapshotted Profile, while its source Case
  remains symbolic. It contains no live client, credential, worker, or Run state.
- **Run Event.** One immutable, monotonically sequenced fact appended by the Executor.
- **Slot.** Private immutable single-assignment execution data. Slot opacity does not make declared
  response projections secret; only declared Observations enter Contract evidence.
- **Observation.** A declared typed value attached to a Run Event and available to the Contract. On
  the model side, a machine's evidence names the observation that confirms each Fact a step
  records: a recorded event the realization catalogs, or an observation the Model declares for a
  value that is read rather than recorded. A timer marked unobservable is one no observation
  confirms, and every Case whose path fires it carries a Known Gap saying so.
- **Verdict.** The three-valued conclusion a Run reaches: `satisfied`, `violated`, or
  `inconclusive`, with Rule states and supporting Run Event sequences.

### Where things live

- **`model/`.** Every front end's Behavior Model and implementation: the reference Model and parity
  oracle, the Go and Scala 3 implementations of the model layer, and `model/scalav2`, which lifts
  the Scala Models into the Model IR and interprets it in Go.
- **Testpilot.** The canonical name for running behavior through Temporal and Workers. The Testpilot
  protobuf closure rooted at `proto/internal/temporal/server/api/testpilot/v1/case.proto` is the
  wire authority. The shared Go runtime under `common/testing/testpilot` admits and executes Cases
  through a caller-owned Driver and evaluates their Contracts.

### Purpose and scope

- **SCP-01 — Temporal-driven scope.** Umpire MUST include only capabilities required by a concrete
  Temporal use case in modeling, Regression, Exploration, Case execution, or verification.
- **SCP-02 — Reusable core.** Reusable Umpire framework code MUST NOT contain Temporal-specific names,
  dependencies, or fixtures.
- **SCP-03 — Behavior Model location.** All Behavior Model code MUST live under `model/` and be
  written in a front end. *(Restated 2026-09-30; awaiting GOV-02 approval. The approved text names
  the single front end that existed when it was written.)*
- **SCP-04 — Complement specialized tests.** Umpire SHOULD complement specialized unit, race,
  persistence, schema, authorization, performance, and handler tests rather than replace them.

### Source of truth

- **SEM-01 — Model authority.** For behavior covered by Umpire, the Behavior Model MUST be the only
  source of truth. Generated code, Artifacts, runtimes, Evidence mappings, and checker adapters MUST
  NOT add or override behavior.
- **SEM-16 — Case authority.** One admitted Case MUST be authoritative for its exact bounded Program
  and Contract. Runtime code MUST NOT add scenario behavior, verification clauses, implicit retry,
  or undeclared evidence. *(A restatement awaits GOV-02 approval: the Case is also authoritative for
  the bounds that carry its behavior -- instruction timeouts and attempts, Contract deadlines and
  correlated windows -- while resource ceilings belong to the Profile, and admission checks every
  Case's bounds and structure against them before Driver I/O.)*
- **SEM-17 — Evaluator authority.** The prepared Contract MUST supply the Monitor used during
  execution and MUST use the same transition semantics for offline evaluation. Expiry is evaluated
  before transitions at every event, bounded captures are rule-local and Run-local, and a proven
  violation MUST remain authoritative despite later cleanup or operational failure.
- **SEM-18 — Producer neutrality.** Every front end MUST produce deterministic Cases for model-owned
  behavior, but any conforming client MAY author a Case. A Case from another client is not thereby a
  Behavior Model declaration or a claim about any other Case.
- **SEM-19 — One word per concept.** *(awaiting GOV-02 approval.)* Each concept in this document MUST
  have exactly one word, and that word MUST be spelled the same in every front end, in the Model IR,
  in the Testpilot protobuf schema, in Go, in the command syntax, and in prose. A second word for a
  concept that already has one is a rename, not a synonym: the old word MUST be removed in the same
  change that introduces the new one. A word MUST NOT name two concepts.
- **SEM-20 — Compound-only retirement.** *(awaiting GOV-02 approval.)* A retired word MUST enter the
  retired-vocabulary gate when, and only when, it is a compound identifier, a module path, an
  executable or Make target name, a macro name, or a snake_case keyword. Bare English words such as
  `Target`, `Behavior`, or `Step` MUST NOT be gated, because the gate also matches their lowercase
  form and would reject ordinary prose. A retired bare keyword MUST instead be rejected by the front
  end that used to accept it, with a located error naming its replacement.

### Runtime module boundaries

- **MOD-12 — Public Testpilot facade.** The public execution sequence MUST be exactly
  `testpilot.Prepare(case, profile)` followed by `PreparedCase.Run(ctx, driver)`. Scheduler,
  Recorder, Slot storage, and Monitor-factory construction MUST remain internal.
- **MOD-13 — Temporal authority split.** `common/testing/testpilot/temporal/server` MUST supply the
  authorized descriptor catalog and transport prepared unary method/request pairs, returning raw
  typed responses and protocol status. `common/testing/testpilot/temporal/worker` MUST own SDK
  workflow, activity, and Nexus-handler execution plus reserved activation delivery. Neither side may
  assume the other's authority; internal execution owns request construction and response
  projection.
- **MOD-14 — Internal execution boundary.** Production packages outside Testpilot and its private
  verification package MUST NOT import `common/testing/testpilot/internal/execution`; Driver
  adapters depend only on the public Testpilot facade. *(A restatement awaits GOV-02 approval: Driver
  adapters also depend on the Driver-contract leaf `common/testing/testpilot/contract`, which imports
  neither internal package, and the facade re-exports every leaf type by alias.)*

### Module design

- **MOD-07 — Clear component boundaries.** Components MUST have narrow responsibilities and
  communicate through explicit contracts rather than each other's internal representations.
- **MOD-08 — Isolated testability.** Each component MUST be testable with fixtures or generic
  examples without the complete Umpire pipeline or a running Temporal cluster.

## Model authoring and traces

### Trace concepts

- **Model.** The checked behavior an author writes: its machines, their finite domains -- states,
  Actions, Model Outcomes, and Facts -- in a canonical order, and the Properties, Scenarios and
  Queries over them.
- **Machine.** The transition relation of a Model: a state of finite fields, one step function per
  action, the entity it tracks, the states it starts and ends in, its timers, and the observation
  confirming each Fact. A machine may be derived from another by restricting it to some actions or
  extending an action's rows; a derived machine keeps its source's entity, state, starts, ends and
  evidence, owns its own Action catalog and Definition IDs, and inherits no refinement.
- **Entity.** Something with identity that a machine keeps state for, with the entities it refers to
  and the key recorded data names an instance by. State belongs to machines, not entities: two
  machines may track one entity differently.
- **Party.** Who performs an action: a name a Model declares by using it on an action. `system` is
  reserved for the implementation under test, performs no declared action, and owns the timers. A
  set binds every other party; a fault is an ordinary action of the party that causes it.
- **Refinement.** A machine that refines another through a state map: a stuttering forward
  simulation decided over the two tables, so every Property declared on the abstract machine is read
  on the refining machine's paths. Both machines of a refinement are product behavior.
- **Composition.** One Model built from machines of different entities, for a claim no one of them
  can state: a state with one field per member, member actions paired into one step by sync lines,
  and every other member action stepping its member alone. A composed state is keyed by its members'
  state keys `_`-joined in field order, a member's own action as `<field>_<key>`, a synchronized
  action by its sync name, and its Definition IDs hang off the owner `compose-<name>`. Only the rows
  reachable from its starts belong to it; it answers `verify` Queries only.
- **Table.** The finite row form of a Machine: every state in catalog order, every action class, and
  one row per enabled pair.
- **Action.** Something an author asks the Model to do, such as closing a Workflow. Requesting an
  Action neither chooses its Model Outcome nor proves that the Action occurred at runtime. An action
  has a party, the entity it creates or acts on, typed finite inputs and an optional protobuf
  schema. Each member of an input domain is a class, and a constructor with finite fields is one
  class per assignment of them; a Scenario selects classes, and a step function is written over them.
- **Model Outcome.** The result the Model produces for an Action. It is an expected model result,
  not a runtime result.
- **Step.** One model step and what it produced: an outcome, a state, and a list of facts.
- **Fact.** A claim the Model makes at a Step, such as "the Nexus operation received a
  cancellation." Logs, spans, RPCs, and records are Evidence used to decide whether the claim held
  during a Run. The Fact domain is a Model's own type.
- **Trace.** A starting state and a sequence of Steps. It contains no runtime Evidence.
- **Scenario.** A named, constrained set of Traces. It defines available variations and faults but
  selects no single Trace, and it neither evaluates Properties nor determines whether a Trace
  occurred at runtime.
- **Property.** A reusable pass/fail rule over Traces, built from clauses. For example, closing a
  Workflow cancels its running Nexus operation at most once. Guarded alternatives inside one clause
  are branches.
- **Correlated.** A rule tracked separately per operation and correlated by an explicit key, so one
  operation's obligations never discharge another's. A Producer and the Go runtime share its
  semantics.
- **Unsatisfiable.** A Scenario that allows no Trace. This is an error, not a passing answer.
- **Set.** A named group of Queries by purpose -- `functional`, `canary` or `exploratory` -- that
  binds every party but `system` to `driven` (the Case performs the party's actions, using each
  class's example) or `observed` (the world performs them and the verifier reads which class
  occurred). A functional set compiles to one Case per `find` Query, once per value of its repeat
  switch; a canary set is admitted when a deployment can close every Known Gap its Cases carry; an
  exploratory set names the machine it covers, a coverage goal and a budget, and enumerates its
  coverage targets.
- **Realization.** The platform-owned binding of a Model to a runtime: each action class to an
  instruction, each observation to where it is recorded, each timer to a duration, each switch to
  its values. The Producer assembles a Case's Program and Contract from a Query's witness and the
  realization, so no Program is written per Case.
- **Abstraction Claim.** An author's claim, on one input class of an action, that every realized
  value of the class behaves alike, with the example the functional Case runs. A Case records the
  claims of the classes its path performs; an exploration tries the other members, and a divergent
  member is a counterexample that splits the class. A single-member class carries no claim.

### Model languages

- **SEM-09 — Bounded progress.** A Property claiming that something eventually happens MUST state a
  Limit and unit. A finite Run MUST NOT prove an unlimited "eventually" claim.

### Authoring

- **AUT-02 — Explicit meaning.** Authoring interfaces MUST make states, Actions, Model Outcomes,
  relations, Limits, faults, Capabilities, Known Gaps, and unsupported cases explicit.
- **AUT-03 — Checked declarations.** Public declarations MUST be checked before Search or a Run.
  Failures SHOULD report errors at the relevant source location.
- **AUT-04 — Stable IDs.** Every public Model Definition MUST have a stable, dot-separated Definition
  ID that is checked against the expected definition kind. Source order and documentation MUST NOT
  affect it.
- **AUT-05 — Cross-language data.** Anything used in Search, Artifacts, Promotion, or cross-language
  Runs MUST be serializable data every front end and the Go runtime can interpret, such as the Model
  IR or a Case. It MUST NOT depend on in-process callbacks.

## Search, Limits, and Artifacts

### Search and Artifact concepts

- **Limit.** A typed ceiling on one stage, in one unit: `steps`, `actions`, `logicalTime`, `search`,
  or `plans`. It limits no other stage. A Query carries a set of Limits.
- **Deadline.** The bounded-time ceiling one Contract Rule declares, counted in the Run Events that
  Rule has evaluated since its last transition, or in elapsed milliseconds on the host that produced
  the Run. A Deadline bounds a Rule; a Limit bounds a stage.
- **Limit Reached.** The result reported when a stage reaches a Limit before answering its question.
  It proves neither a negative answer nor that the search was exhaustive.
- **Exhaustive Search.** A search that checks every candidate the exact Scenario and Limits allow.
  If it finds no candidate, it proves absence only within those Limits.
- **Query.** A bounded question about a Model, within explicit Limits. Its form is `verify`, `find`,
  `findViolation`, or `pick`, and its ending says which Traces count as complete.
- **Search.** Answering a Query: walking the enumerable view of a Model within the Query's Limits and
  recording the work it spent.
- **Plan.** The planned model-level test a Search chose: a sequence of steps retained for
  scenario-neutral catalog and reviewed-promotion use. It is generated instructions, not an
  authoring language, and Testpilot does not accept it.
- **Variations.** A finite space of authored variations over one Model, which a compiler lowers to
  Plans.
- **Artifact.** Immutable, versioned, inspectable data exchanged across components, languages, and
  processes. Artifacts cannot define model behavior.
- **Artifact Checksum.** A reproducible checksum over all Artifact content in canonical order,
  excluding the checksum field itself. It identifies one exact Artifact; it is not a Definition ID or
  Behavior Fingerprint.
- **Generated View.** A deterministic representation of an Artifact, such as a Go test or
  documentation. It is bound to the source Artifact Checksum and cannot define behavior.
- **Inventory.** The generated status and gap catalog of a Behavior Model. It reports what the tree
  contains; it defines nothing.

### Search and Limit rules

- **PLN-01 — Explicit Limits.** Separate, explicit, typed Limits MUST govern each stage: checking
  whether a Scenario allows a Trace, Search, a Run, Evidence evaluation, and failure reduction.
- **PLN-02 — Deterministic selection.** Identical Model Definitions, model inputs, Limits, strategy,
  and seed MUST produce identical Plans and Artifact Checksums.
- **PLN-03 — Exhaustive means complete.** A search declared Exhaustive MUST fail if it cannot check
  every candidate.
- **PLN-04 — Limit Reached is inconclusive.** Limit Reached MUST NOT be treated as proof that no
  trace or counterexample exists.
- **PLN-05 — Unsatisfiable is an error.** A checked Scenario that admits no Trace MUST report
  `unsatisfiable`, never a passing answer.
- **PLN-06 — Generated Plan.** A Plan MUST contain generated instructions. It MUST NOT be an
  authoring language or Evidence that a Run occurred.

### Artifact rules

- **ART-01 — Versioned formats.** Persisted Artifacts MUST use versioned, inspectable formats and
  deterministic serialization.
- **ART-02 — Model binding.** Artifacts MUST carry Definition IDs, Behavior Fingerprints, their own
  Artifact Checksums, source information, Known Gaps, and enough compatibility data for stale
  readers to reject them.
- **ART-04 — Safe format changes.** Readers MUST reject unknown major versions and unknown fields
  that could affect behavior. Changing the meaning of old data requires a named, deterministic
  migration.
- **ART-07 — Generated Views.** The same source Artifact MUST always produce the same Generated View.
  Generated Go tests and documentation MUST be bound to their source Artifact Checksums and MUST NOT
  be editable sources of model behavior.
- **ART-09 — Closed Case format.** A Case MUST contain exactly one versioned Program and one
  Contract, stable IDs, generic opaque provenance, typed roles, paths, Slots, Observations,
  independent limits, and no callback, client, credential, endpoint, or executable. Umpire Producers
  MUST retain their explicit Known Gaps in producer-owned provenance. Unknown versions, fields, enum
  values, instructions, paths, types, crossed references, or out-of-policy resources MUST reject
  before Driver I/O. *(A restatement awaits GOV-02 approval: structured provenance rows, the bounds
  that carry the Case's behavior, and no resource ceiling.)*
- **ART-10 — Immutable preparation.** `testpilot.Prepare` MUST snapshot all admitted Case, Catalog,
  Profile, Program, and Contract data. A Prepared Case MUST be safe for isolated sequential and
  concurrent Runs and MUST expose no mutation path into prepared state.
- **ART-11 — Deterministic Case fixtures.** Producer-generated Case data MUST compare byte-for-byte.
  Runtime values MAY be compared through a named closed projection only when every excluded dynamic
  field is validated structurally. Generic normalization or ignore lists are forbidden.
- **ART-12 — Transactional fixture ownership.** Fixture generation MUST build and validate the
  complete managed tree under a temporary root before comparison or publication. Verification and
  reviewed promotion MUST be separate actions; ordinary tests MUST invoke neither a front end's
  toolchain nor rewrite fixtures.
- **ART-13 — Explicit environment binding.** Exact Case 1.0 is the only admitted and generated
  format. A resource-free Program MAY have an empty environment; every Program that uses a physical
  resource MUST declare a complete closed graph of symbolic text bindings. The Case owns only
  symbolic IDs and references; the Profile owns their physical namespace, task-queue, and named Nexus
  endpoint values. Symbolic endpoint IDs are not transport addresses. Credentials, gRPC targets,
  callback authorities, SDK clients, and lifecycle configuration remain Driver inputs. *(A
  restatement awaits GOV-02 approval: the binding graph is derived at preparation from the IDs a
  Program references, and preparation rejects one the Profile does not supply.)*
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
  dispatch work or mutate a Run.
- **Evidence.** The captured runtime records a claim about a Run rests on: logs, spans, RPCs, and
  history.
- **Projection.** Reading declared Run values into model Steps and fields. It is the only seam
  between a Run and a Model.
- **Run disposition.** `completed`, `stopped_by_monitor`, or `incomplete`; it is independent of the
  cleanup status and Verdict.
- **Cleanup status.** `succeeded`, `failed`, or `timed_out`; cleanup failure never erases a proved
  violation.

### Runtime rules

- **EVD-01 — Thin runtime.** Runtime and CLI code MUST only prepare Cases, bind the Driver authority a
  Profile's Opcodes allow, and execute admitted Programs. It MUST NOT independently decide scenario
  or product behavior.
- **EVD-04 — Fail closed.** Missing, ambiguous, conflicting, outdated, unsupported, or causally
  unrelated Evidence MUST NOT establish success or absence.
- **EVD-05 — Independent statuses.** Authoring, Search, a Run, Evidence evaluation, Implementation
  Link application, Property evaluation, and verification MUST each report their own status. A status
  for one stage MUST NOT imply the status of another.
- **EVD-07 — Distributed ordering.** Conclusions about model behavior MUST rely only on model-declared
  order, causal relationships, or record order within one source. They MUST NOT rely on synchronized
  wall clocks.
- **EVD-08 — Complete lifecycle.** Every Run MUST retain attempted instructions, actual generic
  outcomes, declared Observations, diagnostics, disposition, cleanup status, and Verdict.
- **EVD-11 — Generic scheduling.** The Executor MUST dispatch only admitted instruction/context pairs,
  honor dependencies and guards, enforce each bound, preserve single-assignment Slots, and add no
  scenario-specific branch.
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
- **EVD-20 — Driver-realized faults.** *(approved 2026-09-10 under GOV-02.)* A deliberate outage MUST
  be a declared instruction of the version-one instruction table, MUST name a role the Program
  declares, and MUST be authorized by a Profile Opcode like any other instruction. A Driver MUST
  record exactly one `RUN_EVENT_KIND_FAULT_INJECTED` Run Event per realized instruction, carrying the
  role and the kind it realized, and MUST record none for an instruction it refused or could not
  complete; such an instruction is a failed outcome plus a Driver invariant diagnostic. A requested
  fault proves nothing until the Run carries that event for it. A Program that declares a fault MUST
  hold resources no other Run shares, so an outage it asks for cannot reach another Run.
- **EVD-21 — Deadline units.** *(approved 2026-09-10 under GOV-02.)* A bounded-liveness rule MUST
  declare exactly one positive Deadline bound. `rule_events` counts the Run Events the rule evaluated
  since its last transition and is the bound a conclusion may rest on, because it counts only what
  the Run recorded. `elapsed_milliseconds` remains admitted and is host-clock dependent, so a Case
  whose verdict must not depend on the machine that produced it SHOULD declare the event count
  instead; EVD-07 already forbids resting a conclusion on synchronized wall clocks. Both bounds MUST
  be ticked through one shared helper, so the online and the offline evaluation of the same Run
  answer identically. The counter MUST reset on each transition into a new state, MUST stop once the
  rule is terminal, and MUST freeze with every other rule effect once execution becomes incomplete,
  so no expiry is ever concluded from a truncated Run. A Correlated rule counts admitted operation
  steps and MUST NOT fall back to either.

## Exploration, replay, and promotion

### Exploration concepts

- **Exploration.** Model-owned walking of one exploratory set's enumerated coverage targets: one
  candidate per target in enumeration order, each an exact-trace Query admitted and searched like any
  other, credited per target from a decisive Run along its planned witness path. It is exhausted when
  no target is pending, and it claims coverage only for targets a satisfied Run confirmed. The same
  set, budget and observation stream select the same candidates in the same order and credit the
  same targets. A violated class-member target is a counterexample the campaign retains and compiles
  through Promotion into a review-only regression source; nothing installs it.
- **Replay.** Taking one violated Run of one produced Case back through three classes of replay,
  reported apart. Semantic replay re-evaluates the recorded Run through the same prepared Contract
  offline and must reproduce the recorded Verdict. A concrete rerun prepares the same canonical Case
  under the exact recorded Profile identity and runs it fresh; two reruns decide the subject. SDK
  history replay is diagnostic only and proves nothing. A violation is compared by its key -- the
  violated rules, their terminal states and violating evidence, in Definition IDs -- never by the
  Case's identity. A reproduced subject's Query is reduced by Model-admitted prefix edits, each kept
  only when two fresh Runs reproduce the key.
- **Regression.** A permanent named Query retained to detect recurrence of known behavior
  independently of Exploration Limits.
- **Promotion.** Re-answering a Query against the Behavior Model using exactly the referenced
  Definition IDs, Behavior Fingerprints, Properties, Scenario, and Limits, so a discovered failure can
  be reviewed and kept as a Regression.

### Exploration rules

- **EXP-01 — Shared model.** Regression Runs, model checking, Exploration, fuzzing, promotion replay,
  and canary selection MUST reuse the same Model Definitions and Property declarations.
- **EXP-02 — Model-owned Exploration.** The model MUST define the exploration space, allowed input
  variations, coverage criteria, candidate scoring, and selection rules. Orchestration MAY execute and
  store the resulting batches.
- **EXP-03 — Bounded fuzzing is not exhaustive.** Runtime fuzzing stopped by a time or work Limit
  MUST NOT claim exhaustive coverage.
- **EXP-04 — Pinned Regressions.** Known Regressions MUST run independently of Exploration Limits. A
  campaign's targets are its exploratory set's alone.
- **EXP-05 — Reviewed promotion.** Before human review for promotion to a permanent Regression, a
  discovered failure MUST be reproduced at runtime, minimized in model terms, and re-answered against
  the Behavior Model through Promotion using its exact referenced identities. Reproduced means the
  recorded Verdict replays offline and two fresh Runs reproduce its key; minimized means one bounded
  sweep of Model-admitted prefix edits in which every retained edit reproduced the key twice.

## Verification, CLI, and claims

### Verification and claim concepts

- **Claim Assessment.** Deciding, offline, what one closed Run of one canonical Case supports under
  one Evaluation Profile: a declared, Temporal-free policy of a claim, an asserted trust basis, the
  Known Gap kinds that block acceptance and an ordered reason table, each reason forcing `rejected`
  or `incomplete`; with no reason holding, the subject is `accepted`. The subject is admitted
  strictly and never prepared, run or replayed. The decision is an Evaluation Receipt: canonical bytes
  named by their SHA-256, binding the Profile, the Case and recorded Run identities, the recorded
  Driver identity, the Verdict with its evidence links, the decision and every reason, the Known Gaps
  and the admission caps. A receipt is not self-authenticating and authorizes nothing.

### CLI, environment, and claim rules

- **CLI-01 — Code location.** Umpire CLI code MUST either live under `tools/umpire` or be imported
  from `temporal/tools/common`.
- **CLI-02 — Thin interface.** User-facing tools MAY select declarations and tighten declared Limits.
  They MUST NOT invent Scenario declarations or broaden model-declared Limits.
- **CLI-03 — Inspectability.** User-facing tools SHOULD provide consistent commands to list and
  explain named Properties, Scenarios, Queries, Plans, Explorations, verification checks, Artifacts,
  and Results.
- **CLI-04 — Testpilot command scope.** Case fixture commands MAY build Producer tools and atomically
  promote reviewed deterministic fixtures. Ordinary execution exposes no replacement resident service,
  scenario-specific adapter, or public Monitor selector.
- **QLF-01 — Environment settings.** Environment profiles MAY provide endpoints, credentials,
  namespaces, permissions, resources, and adapters, provided they do not change modeled behavior.
- **QLF-02 — Environment controls.** Each non-local environment MUST explicitly own its authorization
  controls, rate and concurrency limits, cleanup and isolation responsibilities, rollout policy, and
  limits on possible impact.
- **QLF-03 — Complete claims.** Every claim made from a Run MUST expose its environment, Evidence
  policy, Limits, the basis of the claim, Known Gaps, cleanup outcome, and Behavior Fingerprints.
- **QLF-05 — Per-Run decisions.** A Testpilot decision MUST retain Run disposition, cleanup status,
  and Verdict separately. A satisfied Verdict does not hide operational or cleanup failure; a proved
  violated Verdict remains violated after later cleanup failure; every unresolved Contract rule
  closes inconclusive.

## Front end rules

These IDs belong to front end specifications, which state them for their own module graph,
authoring surface and checkers: SEM-02 to SEM-08, MOD-01 to MOD-06, MOD-09 to MOD-11, MOD-15 to
MOD-18, AUT-01, AUT-06 to AUT-09, and VER-01 to VER-06.

## Appendix: retired rules

These rules no longer bind. GOV-01 keeps their IDs so a design that cites one still resolves, and so
no future rule reuses the number. Each names what superseded it.

- **SEM-10 — Retired: portable interpreter seam.** Superseded by SEM-16 through SEM-18. The deleted
  portable interpreter is historical and MUST NOT be restored as an execution recommendation.
- **SEM-11 — Retired: portable plan authority.** Superseded by SEM-16. The deleted portable plan
  format has no runtime authority.
- **SEM-12 — Retired: plan-local claim scope.** Superseded by SEM-17 and SEM-18.
- **SEM-13 — Retired: independently validated model scope.** Superseded by Case provenance and
  Profile admission under SEM-16 and ART-09.
- **SEM-14 — Retired: external portable obligations.** Superseded by explicit Case Known Gaps and the
  closed Contract vocabulary under SEM-17.
- **SEM-15 — Retired: portable plan compilation.** Superseded by Case production under SEM-18.
- **ART-03 — Retired: executable Test Plan.** A Plan is no longer a runtime input; ART-09 defines the
  replacement Case Artifact.
- **ART-05 — Retired: same Test Plan.** Superseded by immutable prepared Case identity under ART-10.
- **ART-06 — Retired: executable trace closure.** Superseded by complete Program admission under
  ART-09 and immutable Run closure under EVD-15.
- **ART-08 — Retired: closed portable evaluation.** Superseded by the Case Contract under ART-09 and
  SEM-17.
- **EVD-02 — Retired: separate legacy Run Evaluation.** Superseded by the authoritative Contract
  Evaluator under SEM-17 and EVD-12.
- **EVD-03 — Retired: legacy Evidence normalization.** The Case Contract consumes only immutable Run
  Events and declared typed Observations under EVD-12.
- **EVD-06 — Retired: legacy Execution Receipts.** Run Event source identity, causal references,
  outcome, and declared Observations replace the retired receipt scheme.
- **EVD-09 — Retired: legacy Evidence Links.** Contract support is represented by exact supporting Run
  Event sequences under EVD-13.
- **EVD-10 — Retired: strict portable interpretation.** Superseded by Contract rules EVD-12 through
  EVD-14.
- **QLF-04 — Retired: per-Test local decisions.** The legacy Local Canary `pass`, `fail`, and
  `inconclusive` decision rule is superseded by the separate Testpilot statuses under QLF-05.
