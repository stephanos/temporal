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

Umpire models expected behavior. A Query asks a bounded question about that behavior, and planning
searches for an answer. A Producer lowers checked behavior into a versioned Case containing one
bounded Program and one deterministic Contract. `PrepareCase` validates that Case against an
immutable Profile without target I/O. A prepared Case can then run repeatedly through an authorized
Host; every attempt produces one append-only Run and one Verdict. For example, a Contract can
require that a declared Nexus history Observation reaches a correlated completion within a bounded
horizon.

## How the model is organized

### Core concepts

- **Behavior Model.** Lean code under `model/` that describes expected product behavior and
  Temporal's current implementation behavior.
- **Model Definition.** A named, handwritten part of the Behavior Model, such as a state, Action,
  transition, or Property. Generated Data and Generated Views are not Model Definitions.
- **Generated Data.** Machine-produced descriptions of API and configuration fields and types. This
  data describes what information exists, not how it affects behavior.
- **Definition ID.** A stable dot-separated ID for a Model Definition, such as
  `switch.query.exact-action`. Umpire checks that an ID refers to the expected kind of
  definition. Reordering declarations or editing documentation does not change it.
- **Behavior Fingerprint.** A value computed from the behavior-affecting parts of a Model Definition.
  It changes with behavior, but not with documentation, source location, or source order.
- **Capability Contract.** A named behavior that one model component requires and another provides,
  together with the laws each provider must satisfy.
- **Implementation Link.** Lean code that explicitly connects product behavior in
  `Temporal.Feature` to corresponding implementation behavior in `Temporal.System` without merging
  their descriptions.
- **Producer.** A compiler or conforming client that creates a versioned Case. Lean is the first
  Producer, but the Case format and Go runtime do not depend on Lean.
- **Case.** Exactly one Program and one Contract, with version, provenance, stable definition
  bindings, and Known Gaps.
- **Program.** A bounded acyclic graph of typed instructions in controller, workflow, activity, or
  Nexus-handler entrypoints.
- **Contract.** A finite set of deterministic safety and bounded-liveness monitor machines over Run
  Events and declared Observations.
- **Profile.** An immutable authorization snapshot containing a descriptor Catalog, symbolic role
  policy, capabilities, and independent Program and Contract ceilings.
- **Host.** The environment-owned implementation of authorized side effects. Server and worker
  capabilities remain separate even when composed behind one Host.
- **Prepared Case.** The immutable result of static Case, Program, Contract, descriptor, and Profile
  admission. It contains no live client, credential, worker, or Run state.
- **Run Event.** One immutable, monotonically sequenced fact appended by the Executor.
- **Slot.** Private immutable single-assignment execution data. Slot opacity does not make declared
  response projections secret; only declared Observations enter Contract evidence.
- **Observation.** A declared typed value attached to a Run Event and available to the Contract.
- **Verdict.** The Evaluator's satisfied, violated, or inconclusive conclusion with rule states and
  supporting Run Event sequences.

### Where things live

- **`Umpire`.** Reusable Lean tools for authoring and checking models and producing plans. It
  contains no Temporal-specific behavior.
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

### Enforced module boundaries

- **MOD-01 — `Umpire` independence.** `Umpire.*` MUST NOT directly or transitively import
  `Temporal.*`.
- **MOD-03 — `Temporal.Feature` isolation.** `Temporal.Feature.*` MUST NOT directly or transitively
  import `Temporal.System.*`. Imports of `Temporal.Verify.*` or `Umpire.Verify.Veil` are allowed only
  for the verification consumers listed in MOD-05.
- **MOD-05 — Verification isolation.** First-party (repository-owned) Lean modules MUST NOT directly
  or transitively import `Temporal.Verify.*` or `Umpire.Verify.Veil` unless they are one of these
  opt-in consumers: `TemporalVerify`, `TemporalVeilTests`, or `Temporal.Tool.VerifyVeil`.
- **MOD-09 — `Shared` independence.** `Shared.*` MUST NOT directly or transitively import `Umpire.*`
  or `Temporal.*`.
- **MOD-10 — `Temporal.System` isolation.** `Temporal.System.*` MUST NOT directly or transitively
  import `Temporal.Feature.*`. The only exception is `Temporal.System.Nexus.ImplementationLink`.
- **MOD-11 — Executable enforcement.** `make lint-model` MUST enforce MOD-01, MOD-03, MOD-05,
  MOD-09, and MOD-10 across the complete first-party Lean import graph.
- **MOD-12 — Public Go facade.** The public execution sequence MUST be exactly
  `PrepareCase(case, profile)` followed by `PreparedCase.Run(ctx, host)`. Scheduler, Recorder, Slot
  storage, and Monitor-factory construction MUST remain internal.
- **MOD-13 — Temporal authority split.** `tools/umpire/temporal/server` MUST supply the authorized
  descriptor catalog and transport prepared unary method/request pairs, returning raw typed
  responses and protocol status. `tools/umpire/temporal/worker` MUST own SDK workflow, activity, and
  Nexus-handler execution plus reserved activation delivery. Neither side may assume the other's
  authority; internal execution owns request construction and response projection.
- **MOD-14 — Internal execution boundary.** Production packages outside the root facade and
  verification package MUST NOT import `tools/umpire/internal/execution`; Host adapters depend on
  the root facade.

### Module design

- **MOD-02 — Product and system ownership.** `Temporal.Feature` MUST own product-visible behavior.
  `Temporal.System` MUST own implementation mechanisms, configuration interpretation, Evidence
  mappings, and Execution behavior.
- **MOD-04 — Independent product and system modules.** `Temporal.Feature.*` and
  `Temporal.System.*` modules MUST be understandable and testable on their own. Only focused
  Implementation Link modules MAY connect them, subject to MOD-10.
- **MOD-06 — Small public interfaces.** `Umpire.*` modules SHOULD hide checking, planning, Artifact,
  observation, and verification machinery behind small, cohesive interfaces.
- **MOD-07 — Clear component boundaries.** Components MUST have narrow responsibilities and
  communicate through explicit contracts rather than each other's internal representations.
- **MOD-08 — Isolated testability.** Each component MUST be testable with fixtures or generic
  examples without the complete `Umpire` pipeline or a running Temporal cluster.

## Model authoring and traces

### Trace concepts

- **Action.** Something a Test asks the model to do, such as closing a Workflow. An Action request
  neither chooses its Model Outcome nor proves that the Action occurred at runtime.
- **Model Outcome.** The next state and Model Facts that the Target produces for an Action. It is an
  expected model result, not a runtime result or Stage Status.
- **Model Fact.** A claim made by the model, such as “the Nexus operation received a cancellation.”
  Logs, spans, RPCs, and records are Evidence used to determine whether the claim was true during a
  Run.
- **Model Trace.** A starting state and a sequence of Actions, Model Outcomes, state changes, and
  Model Facts. It contains no runtime Evidence.
- **Scenario.** A named set of possible Model Traces. It defines available variations and faults but
  does not select a trace.
- **Known Gap.** A missing or unsupported Capability Contract, input, interpretation, or claim. A
  Known Gap limits what an Artifact or Result can prove.
- **Behavior (`Umpire.Behavior`).** Lean data that defines the Model Traces a Scenario allows. It
  neither evaluates Properties nor determines whether a trace occurred at runtime.
- **Target (`Umpire.CheckedTarget`).** A validated Behavior Model shared by Properties, Behaviors,
  and Queries. It is ready for planning and evaluation.
- **Property (`Umpire.Property`).** A reusable pass/fail rule over Model Traces. For example, closing
  a Workflow cancels its running Nexus operation at most once.
- **Query (`Umpire.Query`).** A request, within explicit Limits, to verify a Property, find a
  satisfying witness or violating counterexample, or select a Model Trace for Exploration.
- **Unsatisfiable.** A Behavior that allows no Model Traces. This is an error, not a passing Test.

### Model languages

- **SEM-04 — Separate languages.** Each Lean authoring language—including `Umpire.Property`,
  `Umpire.Behavior`, `Umpire.Query`, and `Umpire.Observation`—MUST be separate and have a distinct
  purpose.
- **SEM-05 — Pure `Umpire.Property`.** `Umpire.Property` declarations MUST use only Model Traces and
  Capability Contracts. They MUST NOT depend on implementation Evidence.
- **SEM-06 — Declarative `Umpire.Behavior`.** `Umpire.Behavior` declarations MUST constrain
  allowed Model Traces. They MUST NOT become step-by-step RPC or runtime scripts.
- **SEM-07 — Model-owned outcomes.** Authors MUST request Actions, while `Umpire.CheckedTarget`
  determines their Model Outcomes and resulting states.
- **SEM-09 — Bounded progress.** A Property claiming that something eventually happens MUST state a
  Limit and unit. A finite Execution MUST NOT prove an unlimited “eventually” claim.

### Authoring

- **AUT-01 — Approachable authoring.** A Temporal engineer with basic Lean knowledge SHOULD be able
  to write ordinary Model Definitions without understanding Umpire's implementation details.
- **AUT-02 — Explicit meaning.** Authoring interfaces MUST make states, Actions, Model Outcomes,
  relations, Limits, faults, Capability Contracts, Known Gaps, and unsupported cases explicit.
- **AUT-03 — Checked declarations.** Public declarations MUST be checked before planning or
  Execution. Failures SHOULD report errors at the relevant source location.
- **AUT-04 — Stable IDs.** Every public Model Definition MUST have a stable, dot-separated Definition
  ID that is checked against the expected definition kind. Source order and documentation MUST NOT
  affect it.
- **AUT-05 — Cross-language data.** Anything used in planning, Artifacts, promotion, or
  cross-language Execution MUST be serializable data that Lean can interpret. It MUST NOT depend on
  in-process callbacks.
- **AUT-06 — Explicit composition.** Competing providers MUST be selected explicitly, and
  cross-domain relationships MUST be connected explicitly. Declaration order and Lean's automatic
  instance search MUST NOT choose behavior.
- **AUT-07 — Single authoring path.** `Umpire.Property`, `Umpire.Behavior`, and `Umpire.Query` MUST
  be the only public languages for declaring Properties, Scenarios, and Queries.
  `Umpire.CheckedTarget` is their shared model representation, not an authoring language. Wrappers
  MUST NOT provide another way to define behavior.
- **AUT-08 — Finite Target adapter.** Authors SHOULD use the proof-carrying
  `Umpire.FiniteMachine` adapter when a complete finite Target has enumerators that define its
  authoritative behavior. The adapter derives membership relations, completeness support, and
  exact finite planning. Authors MUST still provide ordered semantic domains, encoders, enumerators,
  evidence that enumerated values stay within those domains, and evidence that every enumerated
  Action is executable. As an expert alternative, authors MAY construct `Umpire.TransitionKernel`
  directly for Targets whose authority is specified independently. Both paths MUST produce an
  `Umpire.AuthoredTarget` and pass it to `Umpire.checkTarget`. `Umpire.FiniteMachine` MUST NOT
  introduce another Behavior, Property, Query, Scenario, or macro language.

## Planning, Limits, and Artifacts

### Planning and Artifact concepts

- **Limit.** A typed limit for one stage, such as 100 search candidates or 30 execution steps. It
  does not limit any other stage.
- **Limit Reached.** A Stage Status reported when a stage reaches a Limit before answering its
  question. It proves neither a negative answer nor that the search was exhaustive.
- **Exhaustive Search.** A search that checks every candidate allowed by the exact Behavior and
  Limits. If it finds no candidate, it proves absence only within those Limits.
- **Test.** A deterministic Model Trace selected from a Scenario and packaged with its Properties
  and Limits as a Test Plan.
- **Artifact.** Immutable, versioned, inspectable data exchanged across components, languages, and
  processes. Artifacts cannot define model behavior.
- **Case Artifact.** Canonical ProtoJSON for one Case. It is deterministic Producer output and is
  admitted before any Host I/O.
- **Artifact Checksum.** A reproducible checksum over all Artifact content in canonical order,
  excluding the checksum field itself. It identifies one exact Artifact; it is not a Definition ID
  or Behavior Fingerprint.
- **Generated View.** A deterministic representation of an Artifact, such as a Go test or
  documentation. It is bound to the source Artifact Checksum and cannot define behavior.
- **Planning Artifact (`Umpire.ExperimentSpec`).** Model-planning data retained for scenario-neutral
  catalog and reviewed-promotion use. It is not accepted by the Case Runtime.

### Planning and Limit rules

- **PLN-01 — Explicit Limits.** Separate, explicit, typed Limits MUST govern each stage: checking
  whether a Behavior allows a Model Trace, search, Execution, Observation Evaluation, and failure
  reduction.
- **PLN-02 — Deterministic selection.** Identical Model Definitions, model inputs, Limits, strategy,
  and seed MUST produce identical Execution Plans and Artifact Checksums.
- **PLN-03 — Exhaustive means complete.** A search declared Exhaustive MUST fail if it cannot check
  every candidate.
- **PLN-04 — Limit Reached is inconclusive.** Limit Reached MUST NOT be treated as proof that no
  trace or counterexample exists.
- **PLN-05 — Unsatisfiable is an error.** A checked `Umpire.Behavior` that admits no Model Trace MUST
  report `unsatisfiable`, never a passing Test.
- **PLN-06 — Generated Execution Plan.** A `Umpire.DrivePlan` MUST contain generated instructions.
  It MUST NOT be an authoring language or Evidence that Execution occurred.

### Artifact rules

- **ART-01 — Versioned formats.** Persisted Artifacts MUST use versioned, inspectable formats and
  deterministic serialization.
- **ART-02 — Model binding.** Artifacts MUST carry Definition IDs, Behavior Fingerprints, their own
  Artifact Checksums, source information, Known Gaps, and enough compatibility data for stale readers
  to reject them.
- **ART-03 — Retired: executable Test Plan.** `Umpire.ExperimentSpec` is no longer a runtime input;
  ART-09 defines the replacement Case Artifact.
- **ART-04 — Safe format changes.** Readers MUST reject unknown major versions and unknown fields
  that could affect behavior. Changing the meaning of old data requires a named, deterministic
  migration.
- **ART-05 — Retired: same Test Plan.** Superseded by immutable prepared Case identity under ART-10.
- **ART-06 — Retired: executable trace closure.** Superseded by complete Program admission under
  ART-09 and immutable Run closure under EVD-15.
- **ART-07 — Generated Views.** The same source Artifact MUST always produce the same Generated View.
  Generated Go tests and documentation MUST be bound to their source Artifact Checksums and MUST NOT
  be editable sources of model behavior.
- **ART-08 — Retired: closed portable evaluation.** Superseded by the Case Contract under ART-09
  and SEM-17.
- **ART-09 — Closed Case format.** A Case MUST contain exactly one versioned Program and one
  Contract, stable IDs and provenance, explicit Known Gaps, typed roles, paths, Slots,
  Observations, independent limits, and no callback, client, credential, endpoint, or executable.
  Unknown versions, fields, enum values, instructions, paths, types, crossed references, or
  out-of-policy resources MUST reject before Host I/O.
- **ART-10 — Immutable preparation.** `PrepareCase` MUST snapshot all admitted Case, Catalog,
  Profile, Program, and Contract data. A Prepared Case MUST be safe for isolated sequential and
  concurrent Runs and MUST expose no mutation path into prepared state.
- **ART-11 — Deterministic Case fixtures.** Lean-produced Case data MUST compare byte-for-byte.
  Runtime values MAY be compared through a named closed projection only when every excluded dynamic
  field is validated structurally. Generic normalization or ignore lists are forbidden.
- **ART-12 — Transactional fixture ownership.** Fixture generation MUST build and validate the
  complete managed tree under a temporary root before comparison or publication. Verification and
  reviewed promotion MUST be separate actions; ordinary tests MUST invoke neither Lean nor rewrite
  fixtures.

## Case execution and verification

### Runtime concepts

- **Execution.** One bounded attempt to interpret a prepared Program through an authorized Host.
- **Run.** The authoritative append-only record of one attempted Program execution, including
  declared Observations, independent cleanup status, diagnostics, and its immutable Verdict copy.
- **Executor.** The internal generic interpreter that schedules a Program, owns Slot state and
  effect handles, records Run Events, and performs bounded cleanup.
- **Evaluator.** The verification component that creates one fresh Monitor per Run or evaluates a
  closed Run offline using the same Contract transition semantics.
- **Monitor.** One private Run-local Contract state machine. It returns Continue or Stop but cannot
  dispatch work or mutate a Run.
- **Run disposition.** `completed`, `stopped_by_monitor`, or `incomplete`; it is independent of the
  cleanup status and Verdict.
- **Cleanup status.** `succeeded`, `failed`, or `timed_out`; cleanup failure never erases a proved
  violation.

### Runtime rules

- **EVD-01 — Thin runtime.** Runtime and CLI code MUST only prepare Cases, bind authorized Host
  capabilities, and execute admitted Programs. It MUST NOT independently decide scenario or product
  behavior.
- **EVD-02 — Retired: separate legacy Run Evaluation.** Superseded by the authoritative Contract
  Evaluator under SEM-17 and EVD-12.
- **EVD-03 — Retired: legacy Evidence normalization.** The Case Contract consumes only immutable
  Run Events and declared typed Observations under EVD-12.
- **EVD-04 — Fail closed.** Missing, ambiguous, conflicting, outdated, unsupported, or causally
  unrelated Evidence MUST NOT establish success or absence.
- **EVD-05 — Independent statuses.** Authoring, planning, Execution, Observation Evaluation,
  Implementation Link application, Property evaluation, and verification MUST report separate Stage
  Status values. A status for one stage MUST NOT imply the status of another.
- **EVD-06 — Retired: legacy Execution Receipts.** Run Event source identity, causal references,
  outcome, and declared Observations replace the retired receipt scheme.
- **EVD-07 — Distributed ordering.** Conclusions about model behavior MUST rely only on
  model-declared order, causal relationships, or record order within one source. They MUST NOT rely
  on synchronized wall clocks.
- **EVD-08 — Complete lifecycle.** Every Run MUST retain attempted instructions, actual generic
  outcomes, declared Observations, diagnostics, disposition, cleanup status, and Verdict.
- **EVD-09 — Retired: legacy Evidence Links.** Contract support is represented by exact supporting
  Run Event sequences under EVD-13.
- **EVD-10 — Retired: strict portable interpretation.** Superseded by Contract rules EVD-12 through
  EVD-14.
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
- **EVD-15 — Immutable closure.** After `Run` returns, late completions, quarantine release, Host
  diagnostics, caller mutation, and another Run MUST NOT change either returned Run or Verdict.
- **EVD-16 — Activation cancellation.** Cancellation MUST address reserved activation handles and
  already-started SDK commands at activation scope, including delivery that races Stop.
- **EVD-17 — Server/worker composition.** Server Hosts MAY transport only authorized prepared unary
  method/request pairs and return their raw typed response and protocol status. Internal execution
  MUST construct requests, apply declared response projections, assign Slots, and emit Observations.
  Worker Hosts MUST use Temporal SDK APIs for workflow, activity, and Nexus-handler entrypoints.
  Runtime Cases never supply credentials or transport metadata.
- **EVD-18 — Facade conformance.** Regression MUST exercise exactly the satisfied, violated,
  inconclusive, static-preparation-rejection, cleanup-failure-after-proved-violation, and
  cross-Run-isolation facade classes while leaving focused concurrency, cancellation, path,
  cardinality, fuzz, and lifecycle tests independent.

## Exploration, replay, and promotion

### Exploration concepts

- **Exploration.** Model-owned selection from a declared space to find useful Tests or
  counterexamples. It is exhaustive only when it covers the declared finite space within its
  Limits.
- **Regression.** A permanent named `Umpire.Query` retained to detect recurrence of known behavior
  independently of Exploration Limits.
- **Exact Replay.** Re-evaluation of a Model Trace or counterexample against the Behavior Model,
  using exactly the referenced Definition IDs, Behavior Fingerprints, Properties, Behavior, and
  Limits.

### Exploration rules

- **EXP-01 — Shared model.** Regression Execution, model checking, Exploration, fuzzing, Exact
  Replay, and canary selection MUST reuse the same Model Definitions and
  `Umpire.Property` declarations.
- **EXP-02 — Model-owned Exploration.** The model MUST define the exploration space, allowed input
  variations, coverage criteria, candidate scoring, and selection rules. Orchestration MAY execute
  and store the resulting batches.
- **EXP-03 — Bounded fuzzing is not exhaustive.** Runtime fuzzing stopped by a time or work Limit
  MUST NOT claim exhaustive coverage.
- **EXP-04 — Pinned Regressions.** Known Regressions MUST run independently of Exploration Limits.
- **EXP-05 — Reviewed promotion.** Before human review for promotion to a permanent Lean Regression,
  a discovered failure MUST be reproduced at runtime, minimized in model terms, and validated by
  Exact Replay.

## Verification, CLI, and claims

### Verification and claim concepts

- **`Temporal.Verify`.** Optional Temporal-specific checker integration. It does not define behavior.
- **`Umpire.Verify.Veil`.** Optional reusable Veil checker integration. Ordinary models and runtimes
  do not import it.
- **Assurance Method.** The basis for a claim, such as a kernel proof, reconstructed proof, trusted
  solver, search within Limits, Test execution, or concrete replay. These methods are not
  interchangeable.
- **Claim Assessment.** A statement of what a Result proves in a named environment under a specific
  Evidence policy. It includes the Limits, Assurance Method, Known Gaps, trusted sources, and cleanup
  status.

### Optional verification rules

- **VER-01 — Lean-native default.** Lean-native checking MUST be the default verification path.
- **VER-02 — Explicit opt-in.** Each model family and each `Umpire.Property` declaration MUST opt in
  explicitly to optional checker integration.
- **VER-03 — Checked link.** Every representation used by an optional checker MUST have an explicit,
  checked link to an `Umpire.CheckedTarget` and an `Umpire.Property` declaration.
- **VER-04 — Complete verification receipts.** Verification receipts MUST expose source
  information, Definition IDs, Behavior Fingerprints, assumptions, Limits, Known Gaps, and an
  Assurance Method.
- **VER-05 — Exact Replay.** A checker counterexample MUST be validated by Exact Replay against the
  Behavior Model before it can support a claimed model violation or promotion to a Regression.
- **VER-06 — Distinct trust.** Kernel proofs, reconstructed proofs, trusted solvers, search within
  Limits, Test executions, and concrete replay MUST be distinct Assurance Methods.

### CLI, environment, and claim rules

- **CLI-01 — Code location.** Umpire CLI code MUST either live under `tools/umpire` or be imported
  from `temporal/tools/common`.
- **CLI-02 — Thin interface.** User-facing tools MAY select declarations and tighten declared
  Limits. They MUST NOT invent `Umpire.Behavior` declarations or broaden model-declared Limits.
- **CLI-03 — Inspectability.** User-facing tools SHOULD provide consistent commands to list and
  explain named Properties, Scenarios, Tests, Explorations, verification checks, Artifacts, and
  Results.
- **CLI-04 — Case Runtime command scope.** Case fixture commands MAY build Lean Producer tools and
  atomically promote reviewed deterministic fixtures. Ordinary execution exposes no replacement
  resident service, scenario-specific adapter, or public Monitor selector.
- **QLF-01 — Environment settings.** Environment profiles MAY provide endpoints, credentials,
  namespaces, permissions, resources, and adapters, provided they do not change modeled behavior.
- **QLF-02 — Environment controls.** Each non-local environment MUST explicitly own its authorization
  controls, rate and concurrency limits, cleanup and isolation responsibilities, rollout policy,
  and limits on possible impact.
- **QLF-03 — Complete claims.** Every Claim Assessment MUST expose its environment, Evidence policy,
  Limits, Assurance Method, Known Gaps, cleanup outcome, and Behavior Fingerprints.
- **QLF-04 — Retired: per-Test local decisions.** The legacy Local Canary `pass`, `fail`, and
  `inconclusive` decision rule is superseded by the separate Case Runtime statuses under QLF-05.
- **QLF-05 — Per-Run decisions.** A Case Runtime decision MUST retain Run disposition, cleanup
  status, and Verdict separately. A satisfied Verdict does not hide operational or cleanup failure;
  a proved violated Verdict remains violated after later cleanup failure; every unresolved Contract
  rule closes inconclusive.
