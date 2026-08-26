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
- **SCP-02 — Domain-neutral core.** The reusable Umpire library MUST remain free of Temporal
  vocabulary, dependencies, and fixtures.
- **SCP-03 — One model language.** All semantic model code MUST be written in Lean and live under
  `model/`.
- **SCP-04 — Focused complement.** Umpire SHOULD complement, rather than replace, specialized unit,
  race, persistence, schema, authorization, performance, and handler tests.

############ TODO review ############

## Semantic authority

- **SEM-01 — Lean authority.** The Lean model MUST be the sole authority for behavioral meaning;
  generated artifacts, Go code, runtimes, evidence mappings, and checker adapters MUST NOT redefine
  it.
- **SEM-02 — Canonical declarations.** Canonical Feature and System declarations MUST be the only
  sources of Temporal behavioral meaning within the Lean model.
- **SEM-03 — Structural inputs.** Generated API and dynamic-configuration declarations MUST remain
  structural inputs until handwritten Lean declarations assign semantic meaning.
- **SEM-04 — Separate languages.** Property, Behavior, Query, and Observation MUST remain distinct
  typed languages with distinct responsibilities.
- **SEM-05 — Pure properties.** Properties MUST be pure, portable, capability-scoped claims over
  semantic traces and MUST NOT depend on implementation evidence sources.
- **SEM-06 — Declarative behavior.** Behavior MUST constrain admissible semantic trace spaces; it
  MUST NOT become a procedural RPC or runtime script.
- **SEM-07 — Target-owned outcomes.** Authors MUST request actions, while target semantics determines
  their outcomes and resulting states.
- **SEM-08 — Explicit refinement.** Feature product meaning and System implementation meaning MUST
  meet through an explicit refinement, never through declaration order or implicit selection.
- **SEM-09 — Bounded progress.** Progress properties MUST use an explicit bound and declared semantic
  unit; finite execution MUST NOT claim unbounded liveness.

## Model architecture

- **MOD-01 — Dependency direction.** `Umpire.*` MUST NOT import `Temporal.*`.
- **MOD-02 — Semantic altitude.** `Temporal.Feature` MUST own product-visible meaning, while
  `Temporal.System` MUST own implementation mechanisms, configuration interpretation, evidence
  mappings, and execution semantics.
- **MOD-03 — Feature isolation.** `Temporal.Feature.*` MUST NOT import `Temporal.System.*`,
  `Temporal.Verify.*`, or Veil.
- **MOD-04 — Refinement leaves.** Feature and System modules MUST remain independently understandable
  and testable; only focused refinement leaves MAY compose them.
- **MOD-05 — Verification isolation.** Ordinary Umpire and Temporal facades, tools, tests, and builds
  MUST exclude expert verification modules and Veil.
- **MOD-06 — Deep modules.** Umpire modules SHOULD hide substantial checking, planning, artifact,
  observation, and verification machinery behind small, cohesive interfaces.
- **MOD-07 — Component seams.** Components MUST have narrow responsibilities and communicate through
  explicit contracts rather than each other's internal representations.
- **MOD-08 — Isolated testability.** Each component MUST be testable with fixtures or domain-neutral
  examples without requiring the complete Umpire pipeline or a running Temporal cluster.

## Authoring

- **AUT-01 — Approachable authoring.** A Temporal engineer with Lean basics SHOULD be able to author
  ordinary Feature and System declarations without assembling proof, provider, connector,
  canonicalization, digest, or planner plumbing.
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
- **AUT-07 — Single authoring path.** Property, Behavior, and Query MUST remain the only public
  semantic authoring path; compatibility facades MUST NOT create a second interface.

## Planning and bounds

- **PLN-01 — Explicit bounds.** Behavior, search, execution, observation, and minimization MUST each
  have explicit typed bounds.
- **PLN-02 — Deterministic selection.** Identical declarations, semantic inputs, bounds, strategy,
  and seed MUST produce identical selected plans and semantic identities.
- **PLN-03 — Honest completeness.** A complete search MUST fail rather than silently truncate.
- **PLN-04 — Honest exhaustion.** Budget exhaustion MUST remain distinct from proof that no trace or
  counterexample exists.
- **PLN-05 — Honest satisfiability.** An empty behavior MUST report `unsatisfiable`, never success by
  vacuity.
- **PLN-06 — Generated intent.** A DrivePlan MUST be generated execution intent, not an authoring
  language or evidence that execution occurred.

## Artifacts

- **ART-01 — Versioned seams.** Persisted artifacts MUST be explicit, versioned, deterministic, and
  inspectable component boundaries.
- **ART-02 — Semantic binding.** Semantic artifacts MUST carry stable identities, semantic digests,
  provenance, explicit omissions, and enough compatibility data to reject stale consumers.
- **ART-03 — Portable experiment.** ExperimentSpec MUST describe complete, environment-independent,
  bounded execution intent without claiming that any requested action, fault, outcome, or
  observation occurred.
- **ART-04 — Strict evolution.** Readers MUST reject unknown major versions and meaning-bearing
  unknown fields; semantic changes to old data require named deterministic migrations.
- **ART-05 — Same experiment.** Local, CI, staging, black-box, and canary execution MUST consume the
  same semantic ExperimentSpec rather than environment-specific copies of its meaning.
- **ART-06 — Complete traces.** An executable trace MUST include its semantic setup, participant
  programs, runtime-resolved symbolic references, actions, faults, ordering, observations,
  termination, and cleanup obligations.
- **ART-07 — Derived projections.** Generated Go tests and documentation MUST be deterministic,
  digest-bound projections of Lean-owned artifacts, never independently editable semantic sources.

## Execution and evidence

- **EVD-01 — Thin runtime.** Runtime and CLI code MUST bind and execute model-produced artifacts
  without independently interpreting Temporal product semantics.
- **EVD-02 — Separate conformance.** Execution MUST report what happened; evidence interpretation and
  property evaluation MUST separately decide what that establishes.
- **EVD-03 — Qualified evidence.** Raw evidence MUST be normalized, identity-bound, causally ordered,
  checked for source closure and gaps, and translated into semantic observations before properties
  consume it.
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
  and canary selection MUST reuse the same model declarations and properties.
- **EXP-02 — Model-owned exploration.** The model MUST own exploration spaces, mutation operators,
  semantic coverage, candidate scoring, and selection policy; orchestration MAY execute and persist
  the resulting batches.
- **EXP-03 — Honest fuzzing.** Time- or budget-bounded runtime fuzzing MUST NOT claim exhaustive
  coverage or completeness.
- **EXP-04 — Pinned regressions.** Known regressions MUST run independently of exploratory budgets.
- **EXP-05 — Reviewed promotion.** A discovered failure MUST be reproducible, semantically minimized,
  and canonically replayed before a human reviews its promotion into a permanent Lean regression.

## Optional formal verification

- **VER-01 — Lean-native default.** Lean-native checking MUST remain the default verification path.
- **VER-02 — Family opt-in.** Optional checker integration MUST be adopted explicitly per model
  family and property.
- **VER-03 — Checked correspondence.** Every checker view MUST have an explicit checked
  correspondence to an existing canonical target and property.
- **VER-04 — Honest receipts.** Verification receipts MUST expose source and semantic identities,
  assumptions, bounds, omissions, provenance, and trust class.
- **VER-05 — Canonical replay.** A checker counterexample MUST replay through canonical Umpire
  semantics before it can support a semantic violation or promoted regression.
- **VER-06 — Distinct trust.** Kernel proofs, reconstructed proofs, trusted solvers, bounded search,
  testing, and concrete replay MUST remain distinct claim classes.

## CLI and qualification

- **CLI-01 — Code location.** Umpire CLI code MUST live under `tools/umpire` or be imported from
  `temporal/tools/common`.
- **CLI-02 — Thin interface.** User-facing tools MAY select declarations and tighten declared bounds,
  but MUST NOT invent behavior or broaden model-declared bounds.
- **CLI-03 — Inspectability.** Named properties, scenarios, tests, explorations, checks, artifacts,
  and results SHOULD have coherent list and explain surfaces.
- **QLF-01 — Operational bindings.** Environment profiles MAY bind endpoints, credentials,
  namespaces, authority, resources, and adapters only when those bindings do not change semantic
  meaning.
- **QLF-02 — Environment controls.** Each non-local environment MUST explicitly own authorization,
  rate and concurrency limits, cleanup, isolation, rollout policy, and blast-radius controls.
- **QLF-03 — Qualified claims.** Every qualified claim MUST expose its environment, evidence profile,
  bounds, trust, omissions, cleanup outcome, and semantic digests.
