# Umpire semantic model and authoring languages

Status: architecture contract for the domain-neutral Lean library under `model/Umpire` and its
Temporal adapters under `model/Temporal`.

The module seams, author roles, import rules, and optional-verification placement are governed by
[`UMPIRE4_SPEC_MODEL_ARCH.md`](UMPIRE4_SPEC_MODEL_ARCH.md). This document owns the semantic
languages and their denotations. Where the documents overlap, the model-architecture spec controls
source placement and import isolation.

This document does not describe the Go Umpire2 runtime or the independent Umpire3 implementation.
See [`UMPIRE.md`](UMPIRE.md) and [`UMPIRE3.md`](UMPIRE3.md) for those systems. Shared product goals
remain in [`UMPIRE_VISION.md`](UMPIRE_VISION.md).

## Purpose

The model provides one Lean-owned semantic authority for reusable properties, constrained behavior,
bounded questions, executable experiment artifacts, and evidence interpretation. It separates four
authoring jobs:

1. **Property** defines what must hold.
2. **Behavior** defines which semantic traces are admissible.
3. **Query** states what the planner or runner must establish.
4. **Observation** defines how raw evidence establishes semantic observations.

These languages share typed vocabulary but lower to separate internal forms. They do not form one
universal instruction tree.

```text
property + behavior + query + checked target
                    |
                    v
             bounded planner
                    |
             ExperimentSpec
                    |
        +-----------+-----------+
        |                       |
        v                       v
  model exploration       runtime execution
                                    |
                              raw evidence
                                    |
                         observation interpretation
                                    |
                         qualified semantic trace
                                    |
                            property verdicts
```

The implemented current-model path ends at deterministic `ExperimentSpec` inspection. Runtime
execution, live evidence qualification, replay, promotion, and deployment qualification are
separate integration work and must preserve the same semantic identities.

## Core decisions

- Lean is the sole authority for behavioral meaning in this model pipeline.
- Properties, behaviors, queries, and observations have separate responsibilities and types.
- Properties are pure, portable, and capability-scoped; they never mention evidence sources.
- Behavior denotes a constrained semantic trace space, not a procedural RPC script.
- Scenario authoring records requested actions; target semantics owns outcomes.
- A `DrivePlan` is generated execution intent, not an author-facing language or proof of execution.
- Raw evidence is interpreted separately before properties evaluate it.
- Missing, ambiguous, conflicting, stale, or unsupported evidence never becomes success.
- Every search, execution, observation, and minimization phase has explicit typed bounds.
- Public vocabulary and persisted artifacts use stable identities, versions, provenance, semantic
  digests, deterministic ordering, and explicit omissions.
- Capability records are the artifact contract. Lean type classes may provide concise authoring but
  cannot hide consumed requirements.
- Portable declarations are inspectable data with a Lean denotation. Opaque callbacks cannot
  participate in portable planning, artifacts, promotion, or cross-language reuse.
- Cross-domain composition is checked in the semantic model before authoring; runtime code cannot
  invent an unproved `Combine` operation.
- Property, Behavior, and Query are the only public scenario and question languages. A checked
  Target is their shared semantic-model substrate, not a fourth language; obsolete combined
  regression structures and compatibility facades must not remain as a second interface.
- Umpire remains Temporal-agnostic while its interfaces are selected and deepened around problems
  demonstrated by Temporal.
- `Temporal.Feature` owns product meaning and `Temporal.System` owns implementation meaning. Both
  are ordinary authoring surfaces; mixed claims meet through an explicit refinement.
- Ordinary Temporal authors state domain meaning, bounds, and evidence requirements without
  assembling Umpire proof, checking, canonicalization, or planner plumbing.
- Veil is available only through focused generic support and expert adapters under
  `Temporal.Verify`; ordinary Umpire and Temporal facades never expose it.
- Umpire3 is neither a dependency nor a semantic oracle for this model.

## Shared vocabulary and composition

The catalog contains typed declarations for:

- model modules, interfaces, compositions, and targets;
- resources, entities, states, finite attributes, and relations;
- inputs, outputs, actions, faults, and semantic observations;
- properties, behaviors, queries, and observation mappings; and
- stable identities, documentation metadata, provenance, and semantic digests.

Mechanical Protobuf declarations and dynamic-configuration catalogs provide structure. They do not
acquire product meaning until handwritten Lean declarations interpret them.

Every public entry has a namespaced identity and kind, for example:

```text
workflow.state.running
nexus.action.requestCancel
nexus.observation.cancelDelivered
workflow-nexus.relation.ownsOperation
nexus.property.cancelIsUnique
```

Wrong-kind references fail checking even when their text matches. Documentation and source ordering
do not change semantic identity; a change to a consumed contract does.

Properties and behaviors declare the capabilities they require. A checked target provides those
capabilities and their laws. Evaluation exposes only the capability-limited trace view admitted by
the declaration. Competing providers for one identity or relation require an explicit connector;
declaration order and type-class search order never choose silently.

```lean
def callerCloseHonorsCancellation
    [HasWorkflowLifecycle M]
    [HasNexusCancellation M]
    [HasOwnership WorkflowRun NexusOperation M] :
    TraceProperty M :=
  ...
```

## Property language

A property is inspectable typed data with a pure denotation over semantic states, transitions,
relations, inputs, outputs, or finite traces. The portable core supports:

- state invariants;
- transition precondition/postcondition contracts;
- identity and relation properties;
- semantic input/output contracts;
- finite trace ordering; and
- bounded progress or quiescence.

```lean
property honoredDelivery where
  requires Nexus.Cancellation

  whenever cancelRequested
  eventuallyWithin cancelBudget cancelDelivered
```

`eventuallyWithin` uses a declared semantic unit such as transitions, actions, observation
positions, or model-defined logical time. It is not an unbounded liveness claim or an implicit
wall-clock delay. Persisted queries expand named bounds to exact values and units.

A property cannot reference logs, spans, RPC names, storage records, environment profiles, action
realizers, requested fault controls, planner state, or coverage state. Human statements, generated
tests, and support views are projections of the checked declaration rather than independent rules.

Evaluation returns a structured verdict naming the responsible clause, relevant trace span,
expanded bound, semantic provenance, and consumed observation derivations. A portable evaluator
must agree with its Lean denotation. Expert-only opaque predicates remain below the portable
language and cannot be planned, serialized, promoted, or presented as portable declarations.

## Behavior language

A behavior defines admissible semantic traces. It owns symbolic resources, semantic setup,
allowed/required/forbidden actions, named occurrences, finite variation, partial ordering,
occurrence bounds, fault intents, and explicit omissions. It does not decide whether a trace is
correct.

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

Constraint meanings are precise:

| Form | Meaning |
| --- | --- |
| `allow a` | `a` may occur within its occurrence bound |
| `require a` | Introduce a named required occurrence of `a` |
| `forbid a` | No occurrence of `a` is admissible |
| `occurs a exactly/atLeast/atMost n` | Constrain occurrence count |
| `x before y` | Require order while permitting unrelated interleaving |
| `sequence [a, b, c]` | Require relative order while permitting unrelated interleaving |
| `adjacent [a, b]` | Permit no semantic action between the named occurrences |
| `actionsExactly [a, b, c]` | Fix controllable action order while target-owned outcomes may vary |
| `traceExactly witness` | Admit one complete setup, action, outcome, state, and observation trace |

Ordering is over semantic actions, not network, scheduler, storage, or goroutine events. The planner
selects and records a deterministic linear extension of each valid partial order. Adding constraints
can only narrow the denoted trace space.

A semantic fault request declares a required control capability. Execution must return a realization
receipt linked to the intercepted occurrence. A request is never evidence that the fault happened;
missing or misdirected realization is execution divergence.

## Query and planning

A query combines a checked behavior, properties, compatible target, quantifier, strategy, and
bounds. It supports:

- universal verification within complete finite bounds;
- existential witness search;
- counterexample search; and
- selection of experiments for an execution profile.

Complete search fails instead of truncating. An empty behavior is `unsatisfiable`, not universal
success by vacuity. A budgeted search that finds nothing is `budgetExhausted`, not verified.

Phase-specific bounds remain distinct:

| Phase | Bound controls |
| --- | --- |
| Behavior | Admissible semantic traces |
| Search | Planning effort and completeness |
| Execution | Actions, concurrency, and deadlines |
| Observation | Source closure and evidence volume |
| Minimization | Reduction attempts |

The first planner is a deterministic, lazy, bounded Lean enumerator behind a replaceable interface.
Strategies and seeds are query policy, not property or behavior semantics. Query identity covers
resolved declarations, consumed semantic digests, expanded bounds, target composition, strategy,
and seed.

### Planning results

- `found`: a witness, counterexample, or selected experiment exists;
- `noSuchTraceWithinCompleteBounds`: complete search found none;
- `budgetExhausted`: incomplete search stopped without a result;
- `unsatisfiable`: behavior constraints admit no trace; and
- `invalid`: vocabulary, capabilities, types, or bounds are malformed.

## DrivePlan and ExperimentSpec

The runtime consumes generated artifacts rather than evaluating behavior constraints directly.

A `DrivePlan` records selected semantic occurrences, their deterministic order, resources and
bindings, choices, variants, requested faults, required drive capabilities, preconditions, bounds,
observation checkpoints, source identities, selection reason, and omissions.

An `ExperimentSpec` is the portable environment-independent envelope. It embeds or references the
plan and adds property identities, observation requirements, format version, semantic identity,
provenance, and digests. It records what a runtime should attempt; it never claims the attempt,
fault, outcome, or observation occurred.

Artifact readers reject unknown major versions and meaning-bearing unknown fields. Named,
deterministic migrations are required when old data needs transformation. Readers never infer a new
meaning for an old field.

## Observation and verdicts

The Observation language maps raw implementation evidence to shared semantic observations.
Properties consume only the resulting qualified semantic trace.

Each mapping declares:

- evidence source and supported environment profile;
- normalization and sensitive-field disposition;
- symbolic/runtime identity and relation bindings;
- causal or source-local ordering requirements;
- source-closure and gap detection;
- duplicate, ambiguity, and conflict handling;
- observations it may establish; and
- mapping identity, version, and provenance.

Overlapping rules, incompatible bindings, wrong-kind output, and ordering conflicts fail before
processing evidence. First-match order is never semantic. Every retained field is explicitly kept,
redacted, hashed, or rejected.

```text
raw evidence
  -> normalize typed records
  -> bind identities and relations
  -> establish source-local and causal order
  -> verify closure and detect gaps
  -> construct a qualified semantic trace
  -> evaluate pure properties
```

Each established observation retains a compact derivation linking its mapping, evidence identities,
bindings, ordering facts, and closure evidence. Multiple compatible semantic traces initially yield
`unknown` with their missing discriminator; incompatible facts yield `conflict`.

Execution, observation, and property outcomes stay separate:

| Phase | Outcomes |
| --- | --- |
| Execution | `realized`, `diverged`, `unsupported`, `infrastructureFailed` |
| Property | `satisfied`, `violated`, `unknown`, `conflict`, `unsupported` |

Properties evaluate independently when their required observations are available, but a strict
aggregate cannot succeed while a required experiment diverged or a required property is unknown,
conflicting, or unsupported. `realized` never implies `satisfied`, and missing evidence never
implies absence.

## Artifact and component boundaries

Components communicate through explicit versioned artifacts rather than importing one another's
internal representations.

| Artifact | Responsibility |
| --- | --- |
| API catalog | Mechanical Protobuf structure and field disposition |
| Config catalog | Keys, types, defaults, precedence, scope, and classification |
| Semantic catalog | Lean-owned vocabulary, targets, properties, observations, and digests |
| Regression/space | Named behavior and exploration declarations |
| ExperimentSpec | Environment-independent bounded execution intent |
| ExperimentRun | One environment-specific realization with controls and cleanup |
| Raw evidence | Typed facts, receipts, source positions, causality, and omissions |
| Semantic evidence | Lean-owned interpretation of raw facts |
| Result | Qualified property and phase outcomes |
| Replay bundle | Spec, run, evidence, result, bounds, and provenance |
| Verification receipt | Checker target, trust mode, proof or counterexample, and provenance |

The stable component responsibilities are:

| Component | Boundary |
| --- | --- |
| API importer | Descriptor sets to structural Lean declarations and catalog |
| Config importer | Dynamic-config declarations to typed catalog and fixtures |
| Authoring languages | Lean declarations to checked semantic catalog |
| Experiment compiler | Checked query and bounds to `ExperimentSpec` |
| Go/docs projection | Stable regression catalog to non-semantic developer views |
| Execution runtime | `ExperimentSpec` and environment to run plus raw evidence |
| Conformance | Spec, run, and raw evidence to semantic evidence and result |
| Exploration | Scenario space, strategy, bounds, and coverage to selected specs |
| SDK participant | Participant program to SDK behavior and observations |
| Replay/promotion | Failing bundle to minimized replay and reviewed regression |
| Formal checking | Model target and bounds to receipt or counterexample |
| Qualification | Spec and authorized profile to environment-qualified result |

The current implementation integrates structural import, authoring, finite planning, and
deterministic `umpire-experiment/v1` inspection. Go runtimes and richer Umpire3 assurance machinery
are useful independent baselines, but they are not integrated until they consume this artifact and
preserve its semantic identities. Implementation status and task sequencing belong in Flow-Next.

## Package architecture

The reusable `Umpire` Lake library is domain-neutral and exposes independently importable vertical
modules:

```text
                 +-> Umpire.Property -+
Umpire.Core -----+-> Umpire.Behavior -+-> Umpire.Query -> Umpire.Artifact -> Umpire.Planning
                 +-> Umpire.Search ---+
```

`Property`, `Behavior`, and `Search` depend only on Core. Query is the first layer that combines
properties and behavior. Artifact owns portable plans/specs; Planning owns enumeration and private
completion authority. Callers cannot manufacture a verified result.

Public facade modules such as `Umpire.Property` hide cohesive implementation directories. The
generic switch example lives under `Umpire.Examples`. No production or test file under
`model/Umpire` may contain Temporal-owned vocabulary, identities, fixtures, or imports.
The ordinary `Umpire` aggregate does not import optional `Umpire.Verify.Veil` machinery.

Temporal modules are classified by semantic altitude:

- `Temporal.API` and `Temporal.DynamicConfig` own generated mechanical catalogs;
- `Temporal.Feature` owns product-visible behavior and checked feature targets;
- `Temporal.System` owns concrete mechanisms, configuration interpretation, evidence mappings,
  execution adapters, and refinement adapters;
- `Temporal.Tool` owns ordinary inspection and developer tooling without behavioral authority; and
- `Temporal.Verify` owns expert-only checker views, Veil declarations, checked bindings,
  correspondence proofs, and verification entry points.

Feature and System describe semantic altitude, not author expertise. Regular Temporal engineers may
author both through the same approachable Umpire interfaces. Feature models do not import concrete
System mechanisms, and base System mechanisms do not redefine Feature properties. A
`Temporal.System.<Family>.Refinement` leaf imports the relevant Feature and System interfaces and
owns their mapping. Ordinary tooling may import both sides but never imports `Temporal.Verify`.

The classification test is whether a claim survives a complete rewrite of Temporal internals while
externally observable behavior remains the same. Such claims belong in Feature. Concrete handlers,
tasks, persistence, configuration resolution, evidence sources, and other implementation choices
belong in System. Mixed concerns split into a Feature property, System mechanism, explicit
refinement, and observation mapping.

Configuration follows the same seam: generic typed resolution and provenance live under
`Temporal.System.Configuration`; Callback and Matching own their domain-specific interpretations.
Feature may expose an abstract semantic configuration choice only when it changes product-visible
meaning; a refinement maps the resolved System value to that choice.

`Temporal.lean`, ordinary Temporal model tests, and ordinary developer tools exclude
`Temporal.Verify`. Optional family verification enters through a dedicated aggregate such as
`TemporalVerify.lean` and a focused build/test command.
The repository root `Makefile` remains the public build and verification surface.

## Optional Veil checking

Veil is an optional Lean-native capability for selected model families whose inductive invariants,
interference reasoning, symbolic search, or SMT-assisted proof provide value beyond finite planning.
It does not replace Property, Behavior, Query, Planning, Observation, Artifact, or canonical replay.

A Veil-owning family logically owns:

1. its canonical target, properties, and semantic identities under the ordinary Feature or System
   model;
2. a handwritten Veil declaration under `Temporal.Verify.<Family>` in the primary Lake project; and
3. a checked binding under the same expert adapter between Veil states, actions, transitions,
   properties, and the canonical model.

Generic optional Veil machinery lives under `Umpire.Verify.Veil` and contains no Temporal
vocabulary. Family-specific views, declarations, mappings, and correspondence proofs live under
`Temporal.Verify`, not under `Umpire`, `Temporal.Feature`, or base `Temporal.System` modules.

Umpire never generates Veil source from Go, JSON, templates, or `ExperimentSpec`. Lean
metaprogramming may remove local boilerplate, but authored declarations remain inspectable and
source-bound. Ordinary Umpire and Temporal imports must remain usable without importing or
compiling Veil modules.

The binding records canonical and Veil source identities and digests, state/action mappings, the
claimed relation, assumptions, bounds, exclusions, unsupported vocabulary, and trust mode. Partial
bindings expose omissions and cannot claim equivalence from matching fixtures alone.

Results distinguish established, violated, unknown, unsupported, and invalid. They also preserve
kernel proof, reconstructed solver proof, trusted solver, bounded symbolic search, testing, and
concrete replay as different trust classes. Timeout, solver unavailability, incomplete search, stale
digests, and replay disagreement never become success.

Every Veil counterexample must replay through the canonical Umpire transition kernel before it can
support a semantic violation or promoted regression. Verification receipts reference rather than
duplicate `ExperimentSpec` and remain offline build/test artifacts; Veil never enters production
request paths or server binaries. The normal model build and regression gate do not compile or run
`Temporal.Verify`; a separate focused verification gate owns Veil's toolchain, cost, and retained
trust evidence.

## Verification contract

Focused checks must cover:

- capability composition, connector conflicts, and undeclared access;
- property denotation/evaluator agreement and typed bounds;
- behavior contradictions, exactness, monotonic narrowing, and deterministic planning;
- complete-search, unsatisfiable, exhausted, invalid, and anti-forgery outcomes;
- canonical artifacts, stable identities, semantic digests, and deterministic ordering;
- evidence closure, gaps, ambiguity, conflict, causality, field disposition, and derivations;
- independent model, mapping, property, and implementation mutations;
- package import direction and absence of Temporal vocabulary under `model/Umpire`;
- absence of Veil imports from the ordinary `Umpire` and `Temporal` aggregates, Feature, base
  System, and ordinary Tool modules;
- direct elaboration and source-binding of optional declarations under `Temporal.Verify`;
- checked correspondence between each checker view and its canonical Feature or System model;
- stale Veil binding rejection, honest trust classes, and canonical counterexample replay; and
- both the domain-neutral switch and Temporal Nexus scenario through the same public interfaces.

Use `make umpire-check-regression` as the stable ordinary repository command; it excludes optional
Veil adapters. A separate focused command owns `Temporal.Verify` when a family adopts Veil. Current
source and generated fixtures, not status prose in this document, determine implementation truth.

## Non-goals

- A second semantic authority in Go, JSON, Gherkin, generated tests, or checker-neutral IR.
- A monolithic model of all Temporal behavior.
- Exact Go goroutine, network, storage, or distributed-event scheduling.
- Unbounded liveness claims from finite execution.
- Arbitrary callbacks in portable declarations.
- Inferring correctness from requested actions, control receipts, coverage, or metadata.
- Hiding bounds, truncation, unsupported vocabulary, evidence gaps, or cleanup failure.
- Replacing specialized unit, race, persistence, schema, authorization, performance, or handler
  tests.
- Putting Temporal-specific checker views or Veil bindings under `Umpire`.
- Treating Feature as the approachable layer and System as an expert-only layer.
- Importing Veil from `Temporal.Feature`, base `Temporal.System`, ordinary `Temporal.Tool`, or the
  ordinary `Temporal` aggregate.
- Making Veil mandatory, generating Veil source, or importing Umpire3 semantics.
