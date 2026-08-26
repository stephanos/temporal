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

- **SCP-01 — Focused reuse.** Umpire interfaces MUST be driven by modeling, regression, exploration,
  conformance, and verification problems demonstrated by Temporal, while the reusable Umpire
  library remains domain-neutral.
- **SCP-02 — One model language.** All semantic model code MUST be written in Lean and live under
  `model/`.

## Semantic authority

- **SEM-01 — Lean authority.** Canonical Lean Feature and System declarations MUST be the sole
  authority for behavioral meaning; generated artifacts, Go code, runtimes, evidence mappings, and
  checker adapters MUST NOT redefine it.
- **SEM-02 — Structural inputs.** Generated API and dynamic-configuration declarations MUST remain
  structural inputs until handwritten Lean declarations assign semantic meaning.
- **SEM-03 — Separate languages.** Property, Behavior, Query, and Observation MUST remain distinct
  typed languages with distinct responsibilities.
- **SEM-04 — Pure properties.** Properties MUST be pure, portable, capability-scoped claims over
  semantic traces and MUST NOT depend on implementation evidence sources.
- **SEM-05 — Declarative behavior.** Behavior MUST constrain admissible semantic trace spaces, with
  authors requesting actions and target semantics determining outcomes and resulting states.
- **SEM-06 — Explicit refinement.** Feature product meaning and System implementation meaning MUST
  meet through an explicit refinement, never through declaration order or implicit selection.

## Model architecture

- **MOD-01 — Dependency direction.** `Umpire.*` MUST NOT import `Temporal.*` or contain
  Temporal-owned vocabulary or fixtures.
- **MOD-02 — Import isolation.** `Temporal.Feature.*` MUST NOT import `Temporal.System.*`,
  `Temporal.Verify.*`, or Veil; ordinary Umpire and Temporal facades MUST exclude expert verification
  modules.
- **MOD-03 — Deep component seams.** Components MUST hide substantial behavior behind small,
  cohesive interfaces, have narrow responsibilities, and communicate through explicit contracts
  rather than internal representations.

## Authoring

- **AUT-01 — Approachable authoring.** A Temporal engineer with Lean basics SHOULD be able to author
  ordinary Feature and System declarations without assembling proof, provider, connector,
  canonicalization, digest, or planner plumbing.
- **AUT-02 — Explicit meaning.** Authoring interfaces MUST keep meaning-bearing states, actions,
  outcomes, relations, bounds, faults, capabilities, omissions, and unsupported cases explicit.
- **AUT-03 — Checked declarations.** Public declarations MUST be checked before planning or execution,
  and failures SHOULD produce precise source-located diagnostics.
- **AUT-04 — Inspectable declarations.** Anything used for portable planning, artifacts, promotion,
  or cross-language execution MUST be inspectable data with a stable, namespaced, kind-checked
  identity and a Lean denotation, not an opaque callback.

## Planning and bounds

- **PLN-01 — Explicit bounds.** Behavior, search, execution, observation, and minimization MUST each
  have explicit typed bounds.
- **PLN-02 — Deterministic selection.** Identical declarations, semantic inputs, bounds, strategy,
  and seed MUST produce identical selected plans and semantic identities.
- **PLN-03 — Honest search.** Complete search MUST fail rather than truncate, budget exhaustion MUST
  remain distinct from absence, and an empty behavior MUST report `unsatisfiable` rather than
  success by vacuity.
- **PLN-04 — Generated intent.** A DrivePlan MUST be generated execution intent, not an authoring
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

## Execution and evidence

- **EVD-01 — Thin runtime.** Runtime and CLI code MUST bind and execute model-produced artifacts
  without independently interpreting Temporal product semantics.
- **EVD-02 — Phase separation.** Execution MUST report what happened; evidence interpretation and
  property evaluation MUST separately decide what that establishes, and their outcomes MUST NOT
  imply one another.
- **EVD-03 — Qualified evidence.** Raw evidence MUST be normalized, identity-bound, causally ordered,
  checked for source closure and gaps, and translated into semantic observations before properties
  consume it.
- **EVD-04 — Fail closed.** Missing, ambiguous, conflicting, stale, causally unrelated, or unsupported
  evidence MUST NOT establish success or absence.
- **EVD-05 — Realization receipts.** A requested action or fault MUST NOT count as realized without a
  receipt linked to the intended semantic occurrence.
- **EVD-06 — Distributed ordering.** Semantic conclusions MUST rely on declared semantic, causal, or
  source-local ordering rather than synchronized wall clocks.
- **EVD-07 — Complete lifecycle.** Every run MUST retain attempts, realized outcomes, evidence,
  omissions, divergence, infrastructure failures, and cleanup results.

## Exploration, replay, and promotion

- **EXP-01 — Shared semantics.** Regression execution, model checking, exploration, fuzzing, replay,
  and canary selection MUST reuse the same model declarations and properties.
- **EXP-02 — Model-owned exploration.** The model MUST own exploration spaces, mutation operators,
  semantic coverage, candidate scoring, and selection policy; orchestration MAY execute and persist
  the resulting batches.
- **EXP-03 — Budget isolation.** Time- or budget-bounded runtime fuzzing MUST NOT claim completeness,
  and known regressions MUST run independently of exploratory budgets.
- **EXP-04 — Reviewed promotion.** A discovered failure MUST be reproducible, semantically minimized,
  and canonically replayed before a human reviews its promotion into a permanent Lean regression.

## Optional formal verification

- **VER-01 — Lean-native default.** Lean-native checking MUST remain the default verification path;
  optional checkers MUST NOT become a second semantic authority.
- **VER-02 — Opt-in isolation.** Veil and other expert checker adapters MUST remain opt-in under
  focused verification modules and MUST stay out of ordinary imports, builds, tools, artifacts, and
  production runtime paths.
- **VER-03 — Checked correspondence.** Every checker view MUST have an explicit checked
  correspondence to an existing canonical target and property, and every counterexample MUST replay
  through canonical semantics before supporting a semantic violation or promotion.
- **VER-04 — Honest receipts.** Verification receipts MUST expose source and semantic identities,
  assumptions, bounds, omissions, provenance, and trust class; kernel proofs, reconstructed proofs,
  trusted solvers, bounded search, testing, and concrete replay MUST remain distinct claim classes.

## CLI and qualification

- **CLI-01 — Code location.** Umpire CLI code MUST live under `tools/umpire` or be imported from
  `temporal/tools/common`.
- **CLI-02 — Thin, inspectable interface.** User-facing tools MAY select declarations and tighten
  declared bounds, but MUST NOT invent behavior or broaden model-declared bounds; named objects
  SHOULD have coherent list and explain surfaces.
- **QLF-01 — Operational profiles.** Environment profiles MAY supply non-semantic bindings, but each
  non-local profile MUST explicitly own authorization, rate and concurrency limits, cleanup,
  isolation, rollout policy, and blast-radius controls.
- **QLF-02 — Qualified claims.** Every qualified claim MUST expose its environment, evidence profile,
  bounds, trust, omissions, cleanup outcome, and semantic digests.
