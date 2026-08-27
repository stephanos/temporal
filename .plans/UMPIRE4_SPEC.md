# Umpire 4 development rules

This file is the authoritative list of Umpire 4 development rules. Supporting designs cite these
rule IDs. MUST, MUST NOT, SHOULD, and MAY state how strictly each rule applies.

## Governance

- **GOV-01 — Stable rule IDs.** New rules MUST receive new IDs. Existing IDs MUST NOT be renumbered
  or reused, even after a rule is retired.
- **GOV-02 — Human approval.** A human MUST approve any deliberate exception to these rules.

Supporting documents MUST use the terms below with the same meaning. Key terms are capitalized when
used in their Umpire-specific sense. Exact Lean names appear in backticks.

## How Umpire works

Umpire describes expected behavior, chooses a concrete Test, runs it against Temporal, and compares
what happened with what the model expected. For Nexus, a Property can say that closing a Workflow
cancels its running Nexus operation at most once. A Query chooses a Model Trace, a Test Plan records
what to run, and Run Evaluation checks the collected Evidence.

## How the model is organized

### Core concepts

- **Behavior Model.** Lean code under `model/` that describes expected product behavior and the
  behavior of Temporal's current implementation.
- **Model Definition.** One named, handwritten piece of the Behavior Model, such as a state, Action,
  transition, or Property. Generated Data and Generated Views are not Model Definitions.
- **Generated Data.** Machine-produced descriptions of API and configuration fields and types. It
  says what information exists, not how that information affects behavior.
- **Definition ID.** A stable dot-separated ID for a Model Definition, such as
  `workflow-nexus.property.caller-closure`. Umpire checks that an ID refers to the expected kind of
  definition. Reordering declarations or editing documentation does not change it.
- **Behavior Fingerprint.** A value generated from the parts of a Model Definition that affect
  behavior. It changes when behavior changes, but not when documentation, source location, or source
  order changes.
- **Capability Contract.** A named behavior that one model component needs and another provides,
  together with the rules the provider must follow.
- **Implementation Link.** Lean code that connects product behavior under `Temporal.Feature` to the
  corresponding implementation behavior under `Temporal.System` without merging the two
  descriptions.

### Where things live

- **`Umpire`.** Reusable Lean tools for writing and checking models and producing plans. It contains
  no Temporal-specific behavior.
- **`Temporal.Feature`.** Product behavior visible to users and SDKs. It should remain true if the
  implementation changes.
- **`Temporal.System`.** How the current Temporal implementation, configuration, and runtime behave.
- **`Temporal.API`.** Generated API field and type information from Protobuf and gRPC definitions.
- **`Temporal.DynamicConfig`.** Generated configuration field and type information.

### Purpose and scope

- **SCP-01 — Temporal-driven scope.** Umpire MUST add only capabilities required by a concrete
  Temporal modeling, Regression, Exploration, Run Evaluation, or verification use case.
- **SCP-02 — Reusable core.** Reusable `Umpire` code MUST NOT contain Temporal-specific names,
  dependencies, or fixtures.
- **SCP-03 — One model language.** All Behavior Model code MUST be written in Lean and live under
  `model/`.
- **SCP-04 — Focused complement.** Umpire SHOULD complement specialized unit, race, persistence,
  schema, authorization, performance, and handler tests rather than replace them.

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

### Enforced module boundaries

- **MOD-01 — `Umpire` independence.** `Umpire.*` MUST NOT directly or transitively import
  `Temporal.*`.
- **MOD-03 — `Temporal.Feature` isolation.** `Temporal.Feature.*` MUST NOT directly or transitively
  import `Temporal.System.*`, `Temporal.Verify.*`, or `Umpire.Verify.Veil`. The only exceptions are
  the verification consumers listed in MOD-05.
- **MOD-05 — Verification isolation.** Ordinary first-party modules MUST NOT directly or
  transitively import `Temporal.Verify.*` or `Umpire.Verify.Veil`. The only opt-in consumers are
  `TemporalVerify`, `TemporalVeilTests`, `Temporal.Tool.VerifyVeil`, and
  `Temporal.Feature.Nexus.Experimental.CallerClosure.VeilTests`.
- **MOD-09 — `Shared` independence.** `Shared.*` MUST NOT directly or transitively import `Umpire.*`
  or `Temporal.*`.
- **MOD-10 — `Temporal.System` isolation.** `Temporal.System.*` MUST NOT directly or transitively
  import `Temporal.Feature.*`. The only exception is `Temporal.System.Nexus.Refinement`.
- **MOD-11 — Executable enforcement.** `make lint-model` MUST enforce MOD-01, MOD-03, MOD-05,
  MOD-09, and MOD-10 across the complete first-party Lean import graph.

### Module design

- **MOD-02 — Product and system ownership.** `Temporal.Feature` MUST own product-visible behavior.
  `Temporal.System` MUST own implementation mechanisms, configuration interpretation, Evidence
  mappings, and Execution behavior.
- **MOD-04 — Focused mappings.** `Temporal.Feature.*` and `Temporal.System.*` modules MUST remain
  understandable and testable on their own. Only focused Implementation Link modules MAY connect
  them, subject to MOD-10.
- **MOD-06 — Small public interfaces.** `Umpire.*` modules SHOULD hide checking, planning, Artifact,
  observation, and verification machinery behind small, cohesive interfaces.
- **MOD-07 — Clear component boundaries.** Components MUST have narrow responsibilities and
  communicate through explicit contracts rather than each other's internal representations.
- **MOD-08 — Isolated testability.** Each component MUST be testable with fixtures or generic
  examples without the complete `Umpire` pipeline or a running Temporal cluster.

## Model authoring and traces

### Key concepts

- **Action.** Something a Test asks the model to do, such as closing a Workflow. Requesting an
  Action does not choose its Model Outcome or prove that it happened at runtime.
- **Model Outcome.** The result the Checked Model produces for an Action, including the next state
  and any Model Facts. It is an expected model result, not a runtime result or Stage Status.
- **Model Fact.** A fact expressed in the model, such as “the Nexus operation received a
  cancellation.” Logs, spans, RPCs, and records are Evidence used to decide whether the fact
  occurred during a Run.
- **Model Trace.** A starting state followed by Actions, Model Outcomes, state changes, and Model
  Facts. It contains no runtime Evidence.
- **Scenario.** A named set of possible Model Traces. It describes the available variation and
  faults but does not choose one trace.
- **Known Gap.** A Capability Contract, input, interpretation, or claim that is missing or
  unsupported. A Known Gap limits what an Artifact or Result can prove.
- **Behavior (`Umpire.Behavior`).** Lean data describing which Model Traces a Scenario allows. It
  does not decide whether a trace is correct or occurred at runtime.
- **Checked Model (`Umpire.CheckedTarget`).** A model that has passed validation and is ready for
  planning and evaluation.
- **Property (`Umpire.Property`).** A reusable pass/fail rule over Model Traces. For example, closing
  a Workflow cancels its running Nexus operation at most once.
- **Query (`Umpire.Query`).** A request to find a Model Trace that demonstrates or violates a
  Property within explicit Limits.
- **Unsatisfiable.** A Behavior that allows no Model Traces. This is an error, not a passing Test.

### Model languages

- **SEM-04 — Separate languages.** `Umpire.Property`, `Umpire.Behavior`, `Umpire.Query`,
  `Umpire.Observation`, and the other Lean authoring languages MUST remain separate and have separate
  jobs.
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
  to write ordinary Model Definitions without understanding Umpire's internal plumbing.
- **AUT-02 — Explicit meaning.** Authoring interfaces MUST make states, Actions, outcomes, relations,
  Limits, faults, Capability Contracts, Known Gaps, and unsupported cases explicit.
- **AUT-03 — Checked declarations.** Public declarations MUST be checked before planning or
  Execution. Failures SHOULD report errors at the relevant source location.
- **AUT-04 — Stable IDs.** Every public Model Definition MUST have a stable, dot-separated Definition
  ID that is checked against the expected definition kind. Source order and documentation MUST NOT
  affect it.
- **AUT-05 — Portable data.** Anything used for portable planning, Artifacts, promotion, or
  cross-language Execution MUST be serializable data that Lean can interpret. It MUST NOT depend on
  in-process callbacks.
- **AUT-06 — Explicit composition.** Competing providers and cross-domain relationships MUST be
  connected explicitly. Declaration order and Lean's automatic instance search MUST NOT choose
  behavior.
- **AUT-07 — Single authoring path.** `Umpire.Property`, `Umpire.Behavior`, and `Umpire.Query` MUST
  remain the only public scenario and question languages. `Umpire.CheckedTarget` is their shared
  semantic-model substrate, not another language; wrappers MUST NOT introduce another way to define
  behavior.

## Planning, Limits, and Artifacts

### Key concepts

- **Limit.** A typed limit for one stage, such as 100 search candidates or 30 execution steps. A
  Limit for one stage does not limit another.
- **Limit Reached.** A Stage Status reported when a Limit is reached before the stage answers its
  question. It proves neither that the answer is no nor that the search was exhaustive.
- **Exhaustive Search.** A search that checks every candidate allowed by the exact Behavior and
  Limits. Finding no candidate proves absence only within those Limits.
- **Test.** One deterministic Model Trace selected from a Scenario and packaged as a Test Plan with
  its Properties and Limits.
- **Artifact.** Versioned data that does not change and can be inspected. Components, languages, and
  processes exchange Artifacts, but Artifacts cannot define model behavior.
- **Artifact Checksum.** A reproducible checksum of all Artifact content after it has been put in a
  stable order, excluding the checksum field itself. It identifies one exact Artifact; it is not a
  Definition ID or Behavior Fingerprint.
- **Generated View.** A generated view of an Artifact, such as a Go test or documentation. It is
  bound to the source Artifact Checksum and cannot define behavior.
- **Execution Plan (`Umpire.DrivePlan`).** Generated instructions for attempting one selected Model
  Trace. It is not Evidence that Execution occurred.
- **Test Plan (`Umpire.ExperimentSpec`).** A portable file containing everything needed to attempt
  one bounded Test. It describes the intended Test, not what happened.

### Planning and Limits

- **PLN-01 — Explicit Limits.** Deciding whether a Behavior allows a trace, search, Execution,
  observation, and failure reduction MUST each have explicit typed Limits.
- **PLN-02 — Deterministic selection.** Identical Model Definitions, model inputs, Limits, strategy,
  and seed MUST produce identical Execution Plans and Artifact Checksums.
- **PLN-03 — Honest completeness.** An Exhaustive Search MUST fail rather than silently stop early.
- **PLN-04 — Honest limits.** Limit Reached MUST remain distinct from proof that no trace or
  counterexample exists.
- **PLN-05 — Honest `Umpire.Behavior` satisfiability.** A checked `Umpire.Behavior` that admits no
  Model Trace MUST report `unsatisfiable`, never a passing Test.
- **PLN-06 — Generated Execution Plan.** A `Umpire.DrivePlan` MUST be generated instructions, not an
  authoring language or Evidence that Execution occurred.

### Artifacts

- **ART-01 — Versioned formats.** Persisted Artifacts MUST use explicit, versioned formats that can be
  inspected and always serialize the same content in the same way.
- **ART-02 — Model binding.** Artifacts MUST carry Definition IDs, Behavior Fingerprints, their own
  Artifact Checksums, source information, Known Gaps, and enough compatibility data to reject stale
  readers.
- **ART-03 — Portable Test Plan.** `Umpire.ExperimentSpec` MUST contain complete,
  environment-independent instructions for one bounded Test. It MUST NOT claim that any requested
  Action, fault, outcome, or observation occurred.
- **ART-04 — Safe format changes.** Readers MUST reject unknown major versions and unknown fields
  that could affect behavior. Changing the meaning of old data requires a named migration that
  always produces the same result.
- **ART-05 — Same experiment.** Local, CI, staging, black-box, and canary Execution MUST consume the
  same Test Plan rather than environment-specific copies that change its model behavior.
- **ART-06 — Complete traces.** An executable trace MUST include its model setup, participant
  programs, references whose concrete IDs are learned at runtime, Actions, faults, ordering,
  observations, termination, and cleanup obligations.
- **ART-07 — Generated Views.** The same source Artifact MUST always produce the same Generated View.
  Generated Go tests and documentation MUST be bound to their source Artifact Checksums and MUST NOT
  be editable sources of model behavior.

## Execution, Evidence, and Run Evaluation

### Key concepts

- **Execution.** The bounded process of attempting a Test Plan in an environment. It records what
  happened but does not decide whether a Property passed.
- **Run.** The record of one Execution in one environment, including Action and fault attempts,
  receipts, Evidence, failures, and cleanup.
- **Fault Request.** A request to apply a fault at a specific point in a Model Trace. It does not
  prove that the fault occurred.
- **Execution Receipt.** Runtime Evidence tied to a requested Action or fault that confirms it
  actually occurred. Planning or sending a request is not an Execution Receipt.
- **Evidence.** Logs, traces, responses, records, and receipts collected during a Run. Umpire checks
  their source, identity, order, and completeness before using them to establish Model Facts.
- **Evidence Link.** An auditable explanation of why Umpire accepted one Model Fact, including the
  Evidence, bindings, and ordering facts it used.
- **Observation Rules (`Umpire.Observation`).** Lean rules that translate raw Evidence into Model
  Facts while retaining source, order, completeness, conflict, and Evidence Link information.
- **Run Evaluation.** Translation of raw Evidence into a Model Trace followed by checking its
  Properties. It decides what a Run proves but does not perform Execution.
- **Stage Status.** The status of one stage, such as planning, Execution, observation, Property
  checking, or verification. One Stage Status says nothing about another unless a rule explicitly
  connects them.
- **Result.** The complete interpreted report for a Run. It keeps Execution, observation, Property,
  Known Gap, and cleanup statuses separate.

### Execution and Evidence

- **EVD-01 — Thin runtime.** Runtime and CLI code MUST only fill in environment-specific values and
  execute model-produced Artifacts. They MUST NOT independently decide Temporal product behavior.
- **EVD-02 — Separate Run Evaluation.** Execution MUST report what happened. Evidence interpretation
  and Property evaluation MUST separately decide what that proves.
- **EVD-03 — Checked Evidence.** Before a Property uses raw Evidence, Umpire MUST normalize it,
  verify its source and identity, order it causally, check it for missing records, and translate it
  into Model Facts.
- **EVD-04 — Fail closed.** Missing, ambiguous, conflicting, outdated, unsupported, or causally
  unrelated Evidence MUST NOT establish success or absence.
- **EVD-05 — Independent statuses.** Authoring, planning, Execution, Observation Rules, Property
  evaluation, and verification MUST report separate Stage Status values. One MUST NOT imply another.
- **EVD-06 — Execution Receipts.** A requested Action or fault MUST NOT count as having occurred
  without an Execution Receipt linked to the intended point in the Model Trace.
- **EVD-07 — Distributed ordering.** Conclusions about model behavior MUST rely on order declared by
  the model, cause and effect, or records from one source. They MUST NOT rely on synchronized wall
  clocks.
- **EVD-08 — Complete lifecycle.** Every Run MUST retain attempts, actual outcomes, Evidence, Known
  Gaps, divergence, infrastructure failures, and cleanup results.
- **EVD-09 — Evidence Links.** Every accepted Model Fact MUST retain an Evidence Link to its mapping,
  Evidence records, bindings, ordering facts, and completeness checks.

## Exploration, replay, and promotion

### Key concepts

- **Exploration.** Model-owned selection from a declared space to find useful Tests or
  counterexamples. It is exhaustive only when it completes a declared finite space within its
  Limits.
- **Regression.** A permanent named `Umpire.Query` retained to detect recurrence of known behavior
  independently of Exploration Limits.
- **Exact Replay.** Re-evaluation of a Model Trace or counterexample using the exact referenced
  Definition IDs, Behavior Fingerprints, Properties, Behavior, and Limits.

### Rules

- **EXP-01 — Shared model.** Regression Execution, model checking, Exploration, fuzzing, Exact
  Replay, and canary selection MUST reuse the same Model Definitions and
  `Umpire.Property` declarations.
- **EXP-02 — Model-owned Exploration.** The model MUST define what can be explored, how inputs may be
  varied, what counts as coverage, how candidates are scored, and how they are selected.
  Orchestration MAY execute and store the resulting batches.
- **EXP-03 — Honest fuzzing.** Runtime fuzzing stopped by a time or work Limit MUST NOT claim
  exhaustive coverage.
- **EXP-04 — Pinned Regressions.** Known Regressions MUST run independently of Exploration Limits.
- **EXP-05 — Reviewed promotion.** A discovered failure MUST be reproducible, minimized in model
  terms, and reproduced by Exact Replay before a human reviews its promotion into a permanent Lean
  Regression.

## Verification, interfaces, and Claim Assessment

### Key concepts

- **`Temporal.Verify`.** Optional Temporal-specific checker integration. It does not define behavior.
- **`Umpire.Verify.Veil`.** Optional reusable Veil checker integration. Ordinary models and runtimes
  do not import it.
- **Assurance Method.** How a claim was supported, such as a kernel proof, reconstructed proof,
  trusted solver, search within Limits, Test, or concrete replay. These methods are not
  interchangeable.
- **Claim Assessment.** A statement of what a Result proves in a named environment under a specific
  Evidence policy. It includes the Limits, Assurance Method, Known Gaps, trusted sources, and cleanup
  status.

### Optional formal verification

- **VER-01 — Lean-native default.** Lean-native checking MUST remain the default verification path.
- **VER-02 — Explicit opt-in.** Each model family and `Umpire.Property` declaration MUST explicitly
  opt in to optional checker integration.
- **VER-03 — Checked link.** Every checker view MUST have an explicit checked link to an existing
  `Umpire.CheckedTarget` and `Umpire.Property` declaration.
- **VER-04 — Honest receipts.** Verification receipts MUST expose source information, Definition
  IDs, Behavior Fingerprints, assumptions, Limits, Known Gaps, and Assurance Method.
- **VER-05 — Exact Replay.** A checker counterexample MUST pass Exact Replay through the Behavior
  Model before it can support a model violation or promoted Regression.
- **VER-06 — Distinct trust.** Kernel proofs, reconstructed proofs, trusted solvers, search within
  Limits, Tests, and concrete replay MUST remain distinct Assurance Methods.

### CLI and Claim Assessment

- **CLI-01 — Code location.** Umpire CLI code MUST live under `tools/umpire` or be imported from
  `temporal/tools/common`.
- **CLI-02 — Thin interface.** User-facing tools MAY select declarations and tighten declared Limits,
  but MUST NOT invent `Umpire.Behavior` declarations or broaden model-declared Limits.
- **CLI-03 — Inspectability.** User-facing tools SHOULD provide consistent commands to list and
  explain named Properties, Scenarios, Tests, Explorations, checks, Artifacts, and Results.
- **QLF-01 — Environment settings.** Environment profiles MAY provide endpoints, credentials,
  namespaces, permissions, resources, and adapters only when they do not change modeled behavior.
- **QLF-02 — Environment controls.** Each non-local environment MUST explicitly own authorization,
  rate and concurrency limits, cleanup, isolation, rollout policy, and limits on possible impact.
- **QLF-03 — Complete claims.** Every Claim Assessment MUST expose its environment, Evidence policy,
  Limits, Assurance Method, Known Gaps, cleanup outcome, and Behavior Fingerprints.
