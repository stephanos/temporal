# Umpire 4 development rules

This document is the normative index of Umpire 4 development rules. Supporting design documents
cite these rule IDs when refining or applying them. The terms MUST, MUST NOT, SHOULD, and MAY are
normative.

## Governance

- **GOV-01 — Stable index.** New rules MUST receive new IDs; existing IDs MUST NOT be renumbered or
  reused, including after a rule is retired.
- **GOV-02 — Human deviation.** A human MUST approve any change that violates a rule or deliberately
  plans to violate one.

The ubiquitous language is grouped with the rules that govern each part of the domain. These
definitions determine how terms are used in this normative index. Supporting Umpire 4 documents use
them consistently and may add detail without changing their meaning. Lean modules, namespaces, and
types are always referenced by fully qualified names in backticks. Defined Ubiquitous Language terms
are capitalized when used as nouns outside their defining entry; ordinary descriptive and adjectival
uses remain lowercase.

## Model ownership and architecture

### Ubiquitous language

- **Lean Model.** The semantic code under `model/`, comprising the domain-neutral `Umpire` library
  and Temporal-owned declarations. It is the sole authority for behavioral meaning.
- **Semantic Definition.** A checked handwritten declaration under `Temporal.Feature` or
  `Temporal.System` that owns Temporal behavioral meaning for its Semantic Identity. Generated
  Structures, Projections, adapters, and checker views are not Semantic Definitions.
- **Generated Structure.** Mechanically generated information, such as `Temporal.API` or
  `Temporal.DynamicConfig` declarations, that has no behavioral meaning until a Semantic Definition
  interprets it.
- **`Umpire`.** The domain-neutral Lean library and namespace that owns reusable semantic authoring,
  checking, planning, artifact, observation, refinement, and verification machinery.
- **`Temporal.API`.** The generated Temporal namespace for mechanical Protobuf and gRPC structure.
  It owns no product or implementation meaning.
- **`Temporal.DynamicConfig`.** The generated Temporal namespace for mechanical configuration
  structure. It owns no configuration interpretation or behavioral effect.
- **`Temporal.Feature`.** The Temporal namespace that owns product-visible states, actions,
  outcomes, relations, and `Umpire.Property`, `Umpire.Behavior`, and `Umpire.Query` declarations
  whose meaning survives an implementation rewrite.
- **`Temporal.System`.** The Temporal namespace that owns implementation mechanisms, configuration
  interpretation, `Umpire.Observation` declarations, execution semantics, and refinements.
############ TODO review ############
- **Semantic Identity.** A stable, namespaced, kind-checked name for a declaration or selected
  semantic product. It is independent of declaration order and documentation.
- **Semantic Digest.** A deterministic digest of meaning-bearing canonical content, used to detect
  semantic change or stale composition. It is distinct from source location, documentation, and an
  Artifact format version.
- **Capability.** A named semantic contract, including its required laws, that a declaration
  requires or a target composition provides.
- **Refinement.** An explicit checked correspondence from `Temporal.System` implementation meaning
  to `Temporal.Feature` product meaning. Refinement relates the two declarations without allowing
  either to redefine the other.

### Purpose and scope

- **SCP-01 — Temporal-driven scope.** Umpire MUST solve modeling, Regression, Exploration,
  Conformance, and verification problems demonstrated by Temporal rather than hypothetical users.
- **SCP-02 — Domain-neutral core.** The reusable `Umpire` library MUST remain free of Temporal
  vocabulary, dependencies, and fixtures.
- **SCP-03 — One model language.** All semantic model code MUST be written in Lean and live under
  `model/`.
- **SCP-04 — Focused complement.** Umpire SHOULD complement, rather than replace, specialized unit,
  race, persistence, schema, authorization, performance, and handler tests.

### Semantic authority

- **SEM-01 — Lean authority.** The Lean Model MUST be the sole authority for behavioral meaning;
  generated Artifacts, Go code, runtimes, evidence mappings, and checker adapters MUST NOT redefine
  it.
- **SEM-02 — Semantic Definitions.** Checked handwritten `Temporal.Feature` and `Temporal.System`
  declarations MUST be the only sources of Temporal behavioral meaning within the Lean Model.
- **SEM-03 — Generated Structure.** Generated `Temporal.API` and `Temporal.DynamicConfig`
  declarations MUST remain Generated Structures until Semantic Definitions assign behavioral
  meaning.
- **SEM-08 — Explicit Refinement.** `Temporal.Feature` product meaning and `Temporal.System`
  implementation meaning MUST meet through an explicit Refinement, never through declaration order
  or implicit selection.

### Enforced module boundaries

- **MOD-01 — `Umpire` independence.** `Umpire.*` MUST NOT reach `Temporal.*` through first-party
  imports.
- **MOD-03 — `Temporal.Feature` isolation.** `Temporal.Feature.*` MUST NOT reach
  `Temporal.System.*`, `Temporal.Verify.*`, or `Umpire.Verify.Veil` through first-party imports,
  except for an exact MOD-05 verification consumer.
- **MOD-05 — Verification isolation.** Ordinary first-party modules MUST NOT reach
  `Temporal.Verify.*` or `Umpire.Verify.Veil`. The only opt-in consumers are `TemporalVerify`,
  `TemporalVeilTests`, `Temporal.Tool.VerifyVeil`, and
  `Temporal.Feature.Nexus.Experimental.CallerClosure.VeilTests`.
- **MOD-09 — `Shared` independence.** `Shared.*` MUST NOT reach `Umpire.*` or `Temporal.*` through
  first-party imports.
- **MOD-10 — `Temporal.System` isolation.** `Temporal.System.*` MUST NOT reach
  `Temporal.Feature.*` through first-party imports except from
  `Temporal.System.Nexus.Refinement`.
- **MOD-11 — Executable enforcement.** `make lint-model` MUST enforce MOD-01, MOD-03, MOD-05,
  MOD-09, and MOD-10 over the complete first-party Lean import graph.

### Module design

- **MOD-02 — Semantic altitude.** `Temporal.Feature` MUST own product-visible meaning, while
  `Temporal.System` MUST own implementation mechanisms, configuration interpretation, evidence
  mappings, and execution semantics.
- **MOD-04 — Refinement leaves.** `Temporal.Feature.*` and `Temporal.System.*` modules MUST remain
  independently understandable and testable; only focused Refinement leaves MAY relate their
  meanings, with import composition governed by MOD-10.
- **MOD-06 — Deep modules.** `Umpire.*` modules SHOULD hide substantial checking, planning, artifact,
  observation, and verification machinery behind small, cohesive interfaces.
- **MOD-07 — Component seams.** Components MUST have narrow responsibilities and communicate through
  explicit contracts rather than each other's internal representations.
- **MOD-08 — Isolated testability.** Each component MUST be testable with fixtures or domain-neutral
  examples without requiring the complete `Umpire` pipeline or a running Temporal cluster.

## Semantic authoring and traces

### Ubiquitous language

- **Action.** A semantic request recognized by a selected `Umpire.CheckedTarget`. An authored or
  planned action requests a transition; it neither chooses the Model Outcome nor proves that a
  runtime realized the request.
- **Model Outcome.** The `Umpire.CheckedTarget`-owned response to an Action. The same target
  transition determines the resulting state and Semantic Observations. A model outcome is distinct
  from a Phase Outcome or runtime Realization.
- **Semantic Observation.** A target-owned fact present in a pure Semantic Trace or established from
  raw Evidence by `Umpire.Observation`. It is not a raw log, span, RPC, record, or receipt.
- **Semantic Trace.** A pure initial state and ordered sequence of selected Actions, Model Outcomes,
  resulting states, and Semantic Observations. Runtime Evidence and Qualification are absent.
- **Scenario.** A named space of possible Semantic Traces. Its `Umpire.Behavior` constrains
  admissible traces, while model-owned variation and fault declarations may parameterize the space.
  A scenario does not select a concrete trace.
- **Omission.** An explicit declaration that a Capability, input, interpretation, or claim is absent
  or unsupported. An omission narrows what an Artifact or Result can establish.
- **`Umpire.Behavior`.** The typed language that constrains admissible semantic trace spaces without
  deciding whether a trace is correct or whether runtime execution occurred.
- **`Umpire.CheckedTarget`.** A validated composition of semantic vocabulary, Capabilities, laws,
  providers, connectors, and the authoritative transition kernel used by planning and evaluation.
- **`Umpire.Property`.** The typed language for pure, portable, capability-scoped claims over
  Semantic Traces. It contains no implementation evidence sources or runtime controls.
- **`Umpire.Query`.** The typed language that combines checked `Umpire.Behavior` and
  `Umpire.Property` declarations, a compatible `Umpire.CheckedTarget`, a claim, Bounds, and planning
  policy into a bounded question.
- **Unsatisfiable.** A checked `Umpire.Behavior` whose constraints admit no Semantic Trace. It is an
  explicit failure outcome, not success by vacuity.

### Semantic languages

- **SEM-04 — Separate languages.** `Umpire.Property`, `Umpire.Behavior`, `Umpire.Query`,
  `Umpire.Observation`, and other Lean DSLs MUST remain distinct typed languages with distinct
  responsibilities.
- **SEM-05 — Pure `Umpire.Property`.** `Umpire.Property` declarations MUST be pure, portable,
  capability-scoped claims over Semantic Traces and MUST NOT depend on implementation evidence
  sources.
- **SEM-06 — Declarative `Umpire.Behavior`.** `Umpire.Behavior` declarations MUST constrain
  admissible Semantic Trace spaces; they MUST NOT become procedural RPC or runtime scripts.
- **SEM-07 — Target-owned outcomes.** Authors MUST request Actions, while `Umpire.CheckedTarget`
  semantics determine their Model Outcomes and resulting states.
- **SEM-09 — Bounded progress.** Progress claims in `Umpire.Property` MUST use an explicit Bound and
  declared semantic unit; finite Execution MUST NOT claim unbounded liveness.

### Authoring

- **AUT-01 — Approachable authoring.** A Temporal engineer with Lean basics SHOULD be able to author
  ordinary `Temporal.Feature` and `Temporal.System` declarations without assembling proof, provider,
  connector, canonicalization, digest, or planner plumbing.
- **AUT-02 — Explicit meaning.** Authoring interfaces MUST keep meaning-bearing states, actions,
  outcomes, relations, Bounds, faults, Capabilities, Omissions, and unsupported cases explicit.
- **AUT-03 — Checked declarations.** Public declarations MUST be checked before planning or execution,
  and failures SHOULD produce precise source-located diagnostics.
- **AUT-04 — Stable identities.** Every public semantic declaration MUST have a stable, namespaced,
  kind-checked Semantic Identity that is independent of source ordering and documentation.
- **AUT-05 — Portable data.** Anything used for portable planning, Artifacts, promotion, or
  cross-language Execution MUST be inspectable data with a Lean denotation, not an opaque callback.
- **AUT-06 — Explicit composition.** Competing providers and cross-domain relationships MUST be
  connected explicitly; declaration order and type-class search MUST NOT select semantics.
- **AUT-07 — Single authoring path.** `Umpire.Property`, `Umpire.Behavior`, and `Umpire.Query` MUST
  remain the only public semantic authoring path; compatibility facades MUST NOT create a second
  interface.

## Planning, bounds, and artifacts

### Ubiquitous language

- **Bound.** An explicit, typed, phase-local limit with a value and semantic unit. A bound on one
  phase does not implicitly bound another phase.
- **Budget Exhaustion.** A Phase Outcome indicating that an effort Bound was reached before the
  phase established its claim. It proves neither absence nor completeness.
- **Complete Search.** A search with checked completeness evidence that every candidate admitted by
  the exact `Umpire.Behavior` Bounds was considered. Finding no candidate establishes only absence
  within those bounds.
- **Test.** One concrete deterministic Semantic Trace selected by a `Umpire.Query` from a Scenario
  and compiled with its `Umpire.Property` declarations and Bounds into a
  `Umpire.ExperimentSpec`.
- **Artifact.** Immutable, versioned, inspectable data exchanged across a component, language, or
  process seam. Portability does not give an artifact semantic authority.
- **Projection.** A deterministic, digest-bound developer view derived from a semantic Artifact. A
  projection is not an independently editable source of meaning.
- **`Umpire.DrivePlan`.** Generated deterministic execution intent for one selected Semantic Trace.
  It is neither an authoring language nor Evidence of Execution.
- **`Umpire.ExperimentSpec`.** The portable, environment-independent envelope containing complete
  bounded execution intent, `Umpire.Property` identities, `Umpire.Observation` requirements,
  provenance, and semantic bindings. It records what a runtime should attempt, not what occurred.

### Planning and bounds

- **PLN-01 — Explicit Bounds.** `Umpire.Behavior` admission, search, Execution, observation, and
  minimization MUST each have explicit typed Bounds.
- **PLN-02 — Deterministic selection.** Identical declarations, semantic inputs, Bounds, strategy,
  and seed MUST produce identical selected plans and Semantic Identities.
- **PLN-03 — Honest completeness.** A Complete Search MUST fail rather than silently truncate.
- **PLN-04 — Honest exhaustion.** Budget Exhaustion MUST remain distinct from proof that no trace or
  counterexample exists.
- **PLN-05 — Honest `Umpire.Behavior` satisfiability.** A checked `Umpire.Behavior` that admits no
  Semantic Trace MUST report `unsatisfiable`, never success by vacuity.
- **PLN-06 — Generated `Umpire.DrivePlan` intent.** A `Umpire.DrivePlan` MUST be generated execution
  intent, not an authoring language or Evidence that Execution occurred.

### Artifacts

- **ART-01 — Versioned seams.** Persisted Artifacts MUST be explicit, versioned, deterministic, and
  inspectable component boundaries.
- **ART-02 — Semantic binding.** Semantic Artifacts MUST carry Semantic Identities, Semantic Digests,
  provenance, explicit Omissions, and enough compatibility data to reject stale consumers.
- **ART-03 — Portable `Umpire.ExperimentSpec`.** `Umpire.ExperimentSpec` MUST describe complete,
  environment-independent, bounded execution intent without claiming that any requested Action,
  Fault Intent, Model Outcome, or Semantic Observation occurred.
- **ART-04 — Strict evolution.** Readers MUST reject unknown major versions and meaning-bearing
  unknown fields; semantic changes to old data require named deterministic migrations.
- **ART-05 — Same experiment.** Local, CI, staging, black-box, and canary Execution MUST consume the
  same semantic `Umpire.ExperimentSpec` rather than environment-specific copies of its meaning.
- **ART-06 — Complete traces.** An executable Semantic Trace MUST include its semantic setup,
  participant programs, runtime-resolved symbolic references, Actions, Fault Intents, ordering,
  Semantic Observations,
  termination, and cleanup obligations.
- **ART-07 — Derived Projections.** Generated Go tests and documentation MUST be deterministic,
  digest-bound Projections of Lean-owned Artifacts, never independently editable semantic sources.

## Execution, evidence, and conformance

### Ubiquitous language

- **Execution.** A bounded attempt to realize a `Umpire.ExperimentSpec` in an environment. Execution
  reports attempts, realized Model Outcomes, raw Evidence, divergence, infrastructure failures, and
  cleanup; it does not decide `Umpire.Property` satisfaction.
- **Run.** One environment-specific Execution of one `Umpire.ExperimentSpec`, retaining all Action
  and Fault Intent attempts, receipts, Evidence, failures, and cleanup outcomes.
- **Fault Intent.** An authored request to apply a fault at a semantic occurrence. A fault intent is
  not a realized fault without a matching Realization receipt.
- **Realization.** Runtime Evidence, bound to the intended semantic occurrence, that a requested
  Action or Fault Intent actually occurred. Selection, planning, or request dispatch alone is not
  realization.
- **Evidence.** Recorded information about execution. Raw evidence consists of implementation facts
  and receipts; semantic evidence is their identity-bound, ordered, closure-checked interpretation
  under `Umpire.Observation`.
- **Evidence Derivation.** The inspectable justification for one established Semantic Observation,
  including the mapping, evidence identities, bindings, ordering facts, and closure evidence used.
- **`Umpire.Observation`.** The typed language that maps raw evidence into qualified semantic
  Observations while retaining identity, ordering, closure, conflict, and derivation information.
- **Conformance.** Interpretation of raw Evidence into a Semantic Trace followed by evaluation of
  the applicable `Umpire.Property` declarations. Conformance determines what a Run establishes; it
  does not perform Execution.
- **Phase Outcome.** The status reported by one lifecycle phase, such as planning, Execution,
  `Umpire.Observation` interpretation, `Umpire.Property` evaluation, or verification. A phase
  outcome implies no other phase's outcome unless an explicit rule says otherwise.
- **Result.** The qualified interpretation of a Run, retaining distinct Execution,
  `Umpire.Observation` interpretation, `Umpire.Property` evaluation, Omission, and cleanup outcomes.
  A result is not synonymous with any one Phase Outcome.

### Execution and evidence

- **EVD-01 — Thin runtime.** Runtime and CLI code MUST bind and execute model-produced Artifacts
  without independently interpreting Temporal product semantics.
- **EVD-02 — Separate Conformance.** Execution MUST report what happened; Evidence interpretation and
  `Umpire.Property` evaluation MUST separately decide what that establishes.
- **EVD-03 — Qualified Evidence.** Raw Evidence MUST be normalized, identity-bound, causally ordered,
  checked for source closure and gaps, and translated into Semantic Observations before
  `Umpire.Property` declarations consume it.
- **EVD-04 — Fail closed.** Missing, ambiguous, conflicting, stale, causally unrelated, or unsupported
  Evidence MUST NOT establish success or absence.
- **EVD-05 — Independent outcomes.** Authoring, planning, Execution, `Umpire.Observation`
  interpretation, `Umpire.Property` evaluation, and verification outcomes MUST remain distinct and
  MUST NOT imply one another.
- **EVD-06 — Realization receipts.** A requested Action or Fault Intent MUST NOT count as realized
  without a receipt linked to the intended semantic occurrence.
- **EVD-07 — Distributed ordering.** Semantic conclusions MUST rely on declared semantic, causal, or
  source-local ordering rather than synchronized wall clocks.
- **EVD-08 — Complete lifecycle.** Every Run MUST retain attempts, realized Model Outcomes, Evidence,
  Omissions, divergence, infrastructure failures, and cleanup results.
- **EVD-09 — Evidence Derivations.** Every established Semantic Observation MUST retain an Evidence
  Derivation linking it to its mapping, evidence identities, bindings, ordering facts, and closure
  evidence.

## Exploration, replay, and promotion

### Ubiquitous language

- **Exploration.** Model-owned bounded selection from a declared semantic space to find useful
  experiments or counterexamples. Exploration is exhaustive only when it completes a declared
  finite space with checked completeness evidence.
- **Regression.** A permanent named `Umpire.Query` retained to detect recurrence of known behavior
  independently of exploratory budgets.
- **Canonical Replay.** Re-evaluation of a trace or counterexample through the referenced
  `Umpire.CheckedTarget`, `Umpire.Behavior`, and `Umpire.Property` declarations with matching
  Semantic Identities, Semantic Digests, and Bounds.

### Rules

- **EXP-01 — Shared semantics.** Regression execution, model checking, Exploration, fuzzing,
  Canonical Replay, and canary selection MUST reuse the same model declarations and
  `Umpire.Property` declarations.
- **EXP-02 — Model-owned Exploration.** The model MUST own exploration spaces, mutation operators,
  semantic coverage, candidate scoring, and selection policy; orchestration MAY execute and persist
  the resulting batches.
- **EXP-03 — Honest fuzzing.** Time- or budget-bounded runtime fuzzing MUST NOT claim exhaustive
  coverage or completeness.
- **EXP-04 — Pinned Regressions.** Known Regressions MUST run independently of exploratory budgets.
- **EXP-05 — Reviewed promotion.** A discovered failure MUST be reproducible, semantically minimized,
  and canonically replayed before a human reviews its promotion into a permanent Lean Regression.

## Verification, interfaces, and qualification

### Ubiquitous language

- **`Temporal.Verify`.** The opt-in Temporal namespace for checker views, bindings, correspondence,
  and verification entry points. It does not own independent behavioral meaning.
- **`Umpire.Verify.Veil`.** The opt-in, domain-neutral checker-integration namespace. It is excluded
  from the ordinary `Umpire` facade, ordinary Temporal imports, and runtime paths.
- **Trust Class.** The kind of assurance supporting a claim, such as kernel proof, reconstructed
  proof, trusted solver, bounded search, testing, or concrete replay. Different trust classes are
  not interchangeable.
- **Qualification.** Evaluation of admitted Results under a named environment and evidence profile,
  preserving Bounds, trust, Omissions, authority, and cleanup status in the resulting claim.

### Optional formal verification

- **VER-01 — Lean-native default.** Lean-native checking MUST remain the default verification path.
- **VER-02 — Family opt-in.** Optional checker integration MUST be adopted explicitly per model
  family and `Umpire.Property` declaration.
- **VER-03 — Checked correspondence.** Every checker view MUST have an explicit checked
  correspondence to an existing `Umpire.CheckedTarget` and `Umpire.Property` declaration.
- **VER-04 — Honest receipts.** Verification receipts MUST expose source and Semantic Identities,
  assumptions, Bounds, Omissions, provenance, and Trust Class.
- **VER-05 — Canonical Replay.** A checker counterexample MUST replay through canonical `Umpire`
  semantics before it can support a semantic violation or promoted Regression.
- **VER-06 — Distinct trust.** Kernel proofs, reconstructed proofs, trusted solvers, bounded search,
  testing, and concrete replay MUST remain distinct claim classes.

### CLI and qualification

- **CLI-01 — Code location.** Umpire CLI code MUST live under `tools/umpire` or be imported from
  `temporal/tools/common`.
- **CLI-02 — Thin interface.** User-facing tools MAY select declarations and tighten declared Bounds,
  but MUST NOT invent `Umpire.Behavior` declarations or broaden model-declared Bounds.
- **CLI-03 — Inspectability.** Named `Umpire.Property` declarations, Scenarios, Tests, Explorations,
  checks, Artifacts, and Results SHOULD have coherent list and explain surfaces.
- **QLF-01 — Operational bindings.** Environment profiles MAY bind endpoints, credentials,
  namespaces, authority, resources, and adapters only when those bindings do not change semantic
  meaning.
- **QLF-02 — Environment controls.** Each non-local environment MUST explicitly own authorization,
  rate and concurrency limits, cleanup, isolation, rollout policy, and blast-radius controls.
- **QLF-03 — Qualified claims.** Every qualified claim MUST expose its environment, evidence profile,
  Bounds, trust, Omissions, cleanup outcome, and Semantic Digests.
