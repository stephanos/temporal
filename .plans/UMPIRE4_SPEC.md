# Umpire 4 development rules

This document is the normative index of Umpire 4 development rules. Supporting design documents
cite these rule IDs when refining or applying them. The terms MUST, MUST NOT, SHOULD, and MAY are
normative.

## Governance

- **GOV-01 — Stable index.** New rules MUST receive new IDs; existing IDs MUST NOT be renumbered or
  reused, including after a rule is retired.
- **GOV-02 — Human deviation.** A human MUST approve any change that violates a rule or deliberately
  plans to violate one.

## Purpose and scope

- **SCP-01 — Temporal-driven scope.** Umpire MUST solve modeling, regression, exploration,
  conformance, and verification problems demonstrated by Temporal rather than hypothetical users.
- **SCP-02 — Domain-neutral core.** The reusable `Umpire` library MUST remain free of Temporal
  vocabulary, dependencies, and fixtures.
- **SCP-03 — One model language.** All semantic model code MUST be written in Lean and live under
  `model/`.
- **SCP-04 — Focused complement.** Umpire SHOULD complement, rather than replace, specialized unit,
  race, persistence, schema, authorization, performance, and handler tests.

############ TODO review ############

## Ubiquitous language

These definitions determine how terms are used in this normative index. Supporting Umpire 4
documents use them consistently and may add detail without changing their meaning. Lean modules,
namespaces, and types are always referenced by fully qualified names in backticks.

- **Action.** A semantic request recognized by a selected `Umpire.CheckedTarget`. An authored or
  planned action requests a transition; it neither chooses the model outcome nor proves that a
  runtime realized the request.
- **Artifact.** Immutable, versioned, inspectable data exchanged across a component, language, or
  process seam. Portability does not give an artifact semantic authority.
- **Bound.** An explicit, typed, phase-local limit with a value and semantic unit. A bound on one
  phase does not implicitly bound another phase.
- **Budget exhaustion.** A phase outcome indicating that an effort bound was reached before the
  phase established its claim. It proves neither absence nor completeness.
- **Canonical declaration.** A checked handwritten declaration under `Temporal.Feature` or
  `Temporal.System` that owns Temporal behavioral meaning for its semantic identity. Structural
  inputs, projections, adapters, and checker views are not canonical declarations.
- **Canonical replay.** Re-evaluation of a trace or counterexample through the referenced
  `Umpire.CheckedTarget`, `Umpire.Behavior`, and `Umpire.Property` declarations with matching
  semantic identities, digests, and bounds.
- **Capability.** A named semantic contract, including its required laws, that a declaration
  requires or a target composition provides.
- **Complete search.** A search with checked completeness evidence that every candidate admitted by
  the exact behavior bounds was considered. Finding no candidate establishes only absence within
  those bounds.
- **Conformance.** Interpretation of raw evidence into a semantic trace followed by evaluation of
  the applicable `Umpire.Property` declarations. Conformance determines what a run establishes; it
  does not perform execution.
- **Evidence.** Recorded information about execution. Raw evidence consists of implementation facts
  and receipts; semantic evidence is their identity-bound, ordered, closure-checked interpretation
  under `Umpire.Observation`.
- **Evidence derivation.** The inspectable justification for one established semantic observation,
  including the mapping, evidence identities, bindings, ordering facts, and closure evidence used.
- **Execution.** A bounded attempt to realize a `Umpire.ExperimentSpec` in an environment. Execution
  reports attempts, realized outcomes, raw evidence, divergence, infrastructure failures, and
  cleanup; it does not decide property satisfaction.
- **Exploration.** Model-owned bounded selection from a declared semantic space to find useful
  experiments or counterexamples. Exploration is exhaustive only when it completes a declared
  finite space with checked completeness evidence.
- **Fault intent.** An authored request to apply a fault at a semantic occurrence. A fault intent is
  not a realized fault without a matching realization receipt.
- **Lean model.** The semantic code under `model/`, comprising the domain-neutral `Umpire` library
  and Temporal-owned declarations. It is the sole authority for behavioral meaning.
- **Model outcome.** The `Umpire.CheckedTarget`-owned response to a semantic action. The same target
  transition determines the resulting state and semantic observations. A model outcome is distinct
  from a phase outcome or runtime realization.
- **Omission.** An explicit declaration that a capability, input, interpretation, or claim is absent
  or unsupported. An omission narrows what an artifact or result can establish.
- **Phase outcome.** The status reported by one lifecycle phase, such as planning, execution,
  observation, property evaluation, or verification. A phase outcome implies no other phase's
  outcome unless an explicit rule says otherwise.
- **Projection.** A deterministic, digest-bound developer view derived from a semantic artifact. A
  projection is not an independently editable source of meaning.
- **Qualification.** Evaluation of admitted results under a named environment and evidence profile,
  preserving bounds, trust, omissions, authority, and cleanup status in the resulting claim.
- **Realization.** Runtime evidence, bound to the intended semantic occurrence, that a requested
  action or fault actually occurred. Selection, planning, or request dispatch alone is not
  realization.
- **Refinement.** An explicit checked correspondence from `Temporal.System` implementation meaning
  to `Temporal.Feature` product meaning. Refinement relates the two declarations without allowing
  either to redefine the other.
- **Regression.** A permanent named `Umpire.Query` retained to detect recurrence of known behavior
  independently of exploratory budgets.
- **Result.** The qualified interpretation of a run, retaining its distinct execution, observation,
  property, omission, and cleanup outcomes. A result is not synonymous with any one phase outcome.
- **Run.** One environment-specific execution of one `Umpire.ExperimentSpec`, retaining all action
  and fault attempts, receipts, evidence, failures, and cleanup outcomes.
- **Scenario.** A named space of possible semantic traces defined by `Umpire.Behavior` and,
  optionally, `Umpire.Space`. A scenario does not select a concrete trace.
- **Semantic digest.** A deterministic digest of meaning-bearing canonical content, used to detect
  semantic change or stale composition. It is distinct from source location, documentation, and an
  artifact format version.
- **Semantic identity.** A stable, namespaced, kind-checked name for a declaration or selected
  semantic product. It is independent of declaration order and documentation.
- **Semantic observation.** A target-owned fact present in a pure semantic trace or established from
  raw evidence by `Umpire.Observation`. It is not a raw log, span, RPC, record, or receipt.
- **Semantic trace.** A pure initial state and ordered sequence of selected actions, model outcomes,
  resulting states, and semantic observations. Runtime evidence and qualification are absent.
- **Structural input.** Generated mechanical information, such as API or dynamic-configuration
  declarations, that has no behavioral meaning until a canonical declaration interprets it.
- **`Temporal.Feature`.** The Temporal namespace that owns product-visible states, actions,
  outcomes, relations, properties, and scenarios whose meaning survives an implementation rewrite.
- **`Temporal.System`.** The Temporal namespace that owns implementation mechanisms, configuration
  interpretation, evidence mappings, execution semantics, and refinements.
- **Test.** One concrete deterministic semantic trace selected from a scenario, compiled with its
  `Umpire.Property` declarations and bounds into a `Umpire.ExperimentSpec`.
- **Trust class.** The kind of assurance supporting a claim, such as kernel proof, reconstructed
  proof, trusted solver, bounded search, testing, or concrete replay. Different trust classes are
  not interchangeable.
- **Unsatisfiable.** A checked `Umpire.Behavior` whose constraints admit no semantic trace. It is an
  explicit failure outcome, not success by vacuity.
- **`Umpire`.** The domain-neutral Lean library and namespace that owns reusable semantic authoring,
  checking, planning, artifact, observation, refinement, and verification machinery.
- **`Umpire.Behavior`.** The typed language that constrains admissible semantic trace spaces without
  deciding whether a trace is correct or whether runtime execution occurred.
- **`Umpire.CheckedTarget`.** A validated composition of semantic vocabulary, capabilities, laws,
  providers, connectors, and the authoritative transition kernel used by planning and evaluation.
- **`Umpire.DrivePlan`.** Generated deterministic execution intent for one selected semantic trace.
  It is neither an authoring language nor evidence of execution.
- **`Umpire.ExperimentSpec`.** The portable, environment-independent envelope containing complete
  bounded execution intent, properties, observation requirements, provenance, and semantic
  bindings. It records what a runtime should attempt, not what occurred.
- **`Umpire.Observation`.** The typed language that maps raw evidence into qualified semantic
  observations while retaining identity, ordering, closure, conflict, and derivation information.
- **`Umpire.Property`.** The typed language for pure, portable, capability-scoped claims over
  semantic traces. It contains no implementation evidence sources or runtime controls.
- **`Umpire.Query`.** The typed language that combines checked `Umpire.Behavior` and
  `Umpire.Property` declarations, a compatible `Umpire.CheckedTarget`, a claim, bounds, and planning
  policy into a bounded question.
- **`Umpire.Space`.** The typed language for finite variation axes, choices, requested fault intents,
  and semantic coverage goals applied to scenarios.

## Semantic authority

- **SEM-01 — Lean authority.** The Lean model MUST be the sole authority for behavioral meaning;
  generated artifacts, Go code, runtimes, evidence mappings, and checker adapters MUST NOT redefine
  it.
- **SEM-02 — Canonical declarations.** Canonical `Temporal.Feature` and `Temporal.System`
  declarations MUST be the only sources of Temporal behavioral meaning within the Lean model.
- **SEM-03 — Structural inputs.** Generated model code, like API and dynamic-configuration
  declarations, MUST remain structural inputs until handwritten Lean declarations assign semantic
  meaning.
- **SEM-04 — Separate languages.** `Umpire.Property`, `Umpire.Behavior`, `Umpire.Query`,
  `Umpire.Observation`, and other Lean DSLs MUST remain distinct typed languages with distinct
  responsibilities.
- **SEM-05 — Pure `Umpire.Property`.** `Umpire.Property` declarations MUST be pure, portable,
  capability-scoped claims over semantic traces and MUST NOT depend on implementation evidence
  sources.
- **SEM-06 — Declarative `Umpire.Behavior`.** `Umpire.Behavior` declarations MUST constrain
  admissible semantic trace spaces; they MUST NOT become procedural RPC or runtime scripts.
- **SEM-07 — Target-owned outcomes.** Authors MUST request actions, while `Umpire.CheckedTarget`
  semantics determine their outcomes and resulting states.
- **SEM-08 — Explicit refinement.** `Temporal.Feature` product meaning and `Temporal.System`
  implementation meaning MUST meet through an explicit refinement, never through declaration order
  or implicit selection.
- **SEM-09 — Bounded progress.** Progress claims in `Umpire.Property` MUST use an explicit bound and
  declared semantic unit; finite execution MUST NOT claim unbounded liveness.

## Model architecture

- **MOD-01 — Dependency direction.** `Umpire.*` MUST NOT import `Temporal.*`.
- **MOD-02 — Semantic altitude.** `Temporal.Feature` MUST own product-visible meaning, while
  `Temporal.System` MUST own implementation mechanisms, configuration interpretation, evidence
  mappings, and execution semantics.
- **MOD-03 — `Temporal.Feature` isolation.** `Temporal.Feature.*` MUST NOT import `Temporal.System.*`,
  `Temporal.Verify.*`, or `Umpire.Verify.Veil`.
- **MOD-04 — Refinement leaves.** `Temporal.Feature` and `Temporal.System` modules MUST remain
  independently understandable and testable; only focused refinement leaves MAY compose them.
- **MOD-05 — Verification isolation.** Ordinary `Umpire` and `Temporal` facades, tools, tests, and
  builds MUST exclude `Temporal.Verify` and `Umpire.Verify.Veil`.
- **MOD-06 — Deep modules.** `Umpire` modules SHOULD hide substantial checking, planning, artifact,
  observation, and verification machinery behind small, cohesive interfaces.
- **MOD-07 — Component seams.** Components MUST have narrow responsibilities and communicate through
  explicit contracts rather than each other's internal representations.
- **MOD-08 — Isolated testability.** Each component MUST be testable with fixtures or domain-neutral
  examples without requiring the complete Umpire pipeline or a running Temporal cluster.

## Authoring

- **AUT-01 — Approachable authoring.** A Temporal engineer with Lean basics SHOULD be able to author
  ordinary `Temporal.Feature` and `Temporal.System` declarations without assembling proof, provider,
  connector, canonicalization, digest, or planner plumbing.
- **AUT-02 — Explicit meaning.** Authoring interfaces MUST keep meaning-bearing states, actions,
  outcomes, relations, bounds, faults, capabilities, omissions, and unsupported cases explicit.
- **AUT-03 — Checked declarations.** Public declarations MUST be checked before planning or execution,
  and failures SHOULD produce precise source-located diagnostics.
- **AUT-04 — Stable identities.** Every public semantic declaration MUST have a stable, namespaced,
  kind-checked identity that is independent of source ordering and documentation.
- **AUT-05 — Portable data.** Anything used for portable planning, artifacts, promotion, or
  cross-language execution MUST be inspectable data with a Lean denotation, not an opaque callback.
- **AUT-06 — Explicit composition.** Competing providers and cross-domain relationships MUST be
  connected explicitly; declaration order and type-class search MUST NOT select semantics.
- **AUT-07 — Single authoring path.** `Umpire.Property`, `Umpire.Behavior`, and `Umpire.Query` MUST
  remain the only public semantic authoring path; compatibility facades MUST NOT create a second
  interface.

## Planning and bounds

- **PLN-01 — Explicit bounds.** `Umpire.Behavior` admission, search, execution, observation, and
  minimization MUST each have explicit typed bounds.
- **PLN-02 — Deterministic selection.** Identical declarations, semantic inputs, bounds, strategy,
  and seed MUST produce identical selected plans and semantic identities.
- **PLN-03 — Honest completeness.** A complete search MUST fail rather than silently truncate.
- **PLN-04 — Honest exhaustion.** Budget exhaustion MUST remain distinct from proof that no trace or
  counterexample exists.
- **PLN-05 — Honest `Umpire.Behavior` satisfiability.** A checked `Umpire.Behavior` that admits no
  semantic trace MUST report `unsatisfiable`, never success by vacuity.
- **PLN-06 — Generated `Umpire.DrivePlan` intent.** A `Umpire.DrivePlan` MUST be generated execution
  intent, not an authoring language or evidence that execution occurred.

## Artifacts

- **ART-01 — Versioned seams.** Persisted artifacts MUST be explicit, versioned, deterministic, and
  inspectable component boundaries.
- **ART-02 — Semantic binding.** Semantic artifacts MUST carry stable identities, semantic digests,
  provenance, explicit omissions, and enough compatibility data to reject stale consumers.
- **ART-03 — Portable `Umpire.ExperimentSpec`.** `Umpire.ExperimentSpec` MUST describe complete,
  environment-independent, bounded execution intent without claiming that any requested action,
  fault, outcome, or observation occurred.
- **ART-04 — Strict evolution.** Readers MUST reject unknown major versions and meaning-bearing
  unknown fields; semantic changes to old data require named deterministic migrations.
- **ART-05 — Same experiment.** Local, CI, staging, black-box, and canary execution MUST consume the
  same semantic `Umpire.ExperimentSpec` rather than environment-specific copies of its meaning.
- **ART-06 — Complete traces.** An executable trace MUST include its semantic setup, participant
  programs, runtime-resolved symbolic references, actions, faults, ordering, observations,
  termination, and cleanup obligations.
- **ART-07 — Derived projections.** Generated Go tests and documentation MUST be deterministic,
  digest-bound projections of Lean-owned artifacts, never independently editable semantic sources.

## Execution and evidence

- **EVD-01 — Thin runtime.** Runtime and CLI code MUST bind and execute model-produced artifacts
  without independently interpreting Temporal product semantics.
- **EVD-02 — Separate conformance.** Execution MUST report what happened; evidence interpretation and
  `Umpire.Property` evaluation MUST separately decide what that establishes.
- **EVD-03 — Qualified evidence.** Raw evidence MUST be normalized, identity-bound, causally ordered,
  checked for source closure and gaps, and translated into semantic observations before
  `Umpire.Property` declarations consume it.
- **EVD-04 — Fail closed.** Missing, ambiguous, conflicting, stale, causally unrelated, or unsupported
  evidence MUST NOT establish success or absence.
- **EVD-05 — Independent outcomes.** Authoring, planning, execution, observation, property, and
  verification outcomes MUST remain distinct and MUST NOT imply one another.
- **EVD-06 — Realization receipts.** A requested action or fault MUST NOT count as realized without a
  receipt linked to the intended semantic occurrence.
- **EVD-07 — Distributed ordering.** Semantic conclusions MUST rely on declared semantic, causal, or
  source-local ordering rather than synchronized wall clocks.
- **EVD-08 — Complete lifecycle.** Every run MUST retain attempts, realized outcomes, evidence,
  omissions, divergence, infrastructure failures, and cleanup results.
- **EVD-09 — Evidence derivations.** Every established semantic observation MUST retain a derivation
  linking it to its mapping, evidence identities, bindings, ordering facts, and closure evidence.

## Exploration, replay, and promotion

- **EXP-01 — Shared semantics.** Regression execution, model checking, exploration, fuzzing, replay,
  and canary selection MUST reuse the same model declarations and `Umpire.Property` declarations.
- **EXP-02 — Model-owned `Umpire.Space`.** The model MUST own `Umpire.Space` declarations, mutation
  operators, semantic coverage, candidate scoring, and selection policy; orchestration MAY execute
  and persist the resulting batches.
- **EXP-03 — Honest fuzzing.** Time- or budget-bounded runtime fuzzing MUST NOT claim exhaustive
  coverage or completeness.
- **EXP-04 — Pinned regressions.** Known regressions MUST run independently of exploratory budgets.
- **EXP-05 — Reviewed promotion.** A discovered failure MUST be reproducible, semantically minimized,
  and canonically replayed before a human reviews its promotion into a permanent Lean regression.

## Optional formal verification

- **VER-01 — Lean-native default.** Lean-native checking MUST remain the default verification path.
- **VER-02 — Family opt-in.** Optional checker integration MUST be adopted explicitly per model
  family and `Umpire.Property` declaration.
- **VER-03 — Checked correspondence.** Every checker view MUST have an explicit checked
  correspondence to an existing `Umpire.CheckedTarget` and `Umpire.Property` declaration.
- **VER-04 — Honest receipts.** Verification receipts MUST expose source and semantic identities,
  assumptions, bounds, omissions, provenance, and trust class.
- **VER-05 — Canonical replay.** A checker counterexample MUST replay through canonical `Umpire`
  semantics before it can support a semantic violation or promoted regression.
- **VER-06 — Distinct trust.** Kernel proofs, reconstructed proofs, trusted solvers, bounded search,
  testing, and concrete replay MUST remain distinct claim classes.

## CLI and qualification

- **CLI-01 — Code location.** Umpire CLI code MUST live under `tools/umpire` or be imported from
  `temporal/tools/common`.
- **CLI-02 — Thin interface.** User-facing tools MAY select declarations and tighten declared bounds,
  but MUST NOT invent `Umpire.Behavior` declarations or broaden model-declared bounds.
- **CLI-03 — Inspectability.** Named `Umpire.Property` declarations, scenarios, tests, explorations,
  checks, artifacts, and results SHOULD have coherent list and explain surfaces.
- **QLF-01 — Operational bindings.** Environment profiles MAY bind endpoints, credentials,
  namespaces, authority, resources, and adapters only when those bindings do not change semantic
  meaning.
- **QLF-02 — Environment controls.** Each non-local environment MUST explicitly own authorization,
  rate and concurrency limits, cleanup, isolation, rollout policy, and blast-radius controls.
- **QLF-03 — Qualified claims.** Every qualified claim MUST expose its environment, evidence profile,
  bounds, trust, omissions, cleanup outcome, and semantic digests.
