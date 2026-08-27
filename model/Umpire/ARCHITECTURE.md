# Umpire public API

Umpire is the reusable, Temporal-independent library for composing finite semantic targets,
checking portable properties and behaviors, planning bounded queries, and producing portable model
artifacts. For the cross-library map, see the [model architecture](../ARCHITECTURE.md).

## Imports and modules

Most consumers should import the umbrella facade:

```lean
import Umpire
```

Focused imports are available when a consumer needs a smaller surface:

| Import | Public responsibility |
| --- | --- |
| `Umpire.Core` | Semantic vocabulary, capabilities, laws, and finite kernels. |
| `Umpire.Target` | Target authoring, checked composition, and canonical target projections. |
| `Umpire.Property` | Portable property authoring, checking, and evaluation. |
| `Umpire.Behavior` | Setup and trace-shape constraints. |
| `Umpire.Query` | Checked combinations of targets, properties, behaviors, bounds, and policies. |
| `Umpire.Artifact` | Portable drive plans and experiment specifications. |
| `Umpire.Planning` | Deterministic incremental planning over checked queries. |
| `Umpire.Observation` | Checked evidence mappings, qualification, derivations, dispositions, semantic verdicts, and strict aggregation. |

`Umpire.Target.Language`, `Umpire.Property.Language`, `Umpire.Behavior.Language`,
`Umpire.Query.Language`, `Umpire.Observation.Language`, `Umpire.Observation.Qualification`, and
`Umpire.Planning.Engine` implement their public facades and should not normally be imported
directly.

## API lifecycle

The public API deliberately separates authoring from checked values:

```text
AuthoredTarget ── checkTarget / elaborateTarget ──▶ CheckedTarget
PropertyDeclaration ─ checkProperty ─▶ CheckedProperty
BehaviorDeclaration ─ checkBehavior ─▶ CheckedBehavior
CheckedTarget + QueryDeclaration ─ checkQuery ─▶ CheckedQuery
CheckedQuery ─ derive planner kernel ─▶ plan ─▶ PlannerRun ─▶ ExperimentSpec?
```

Target is the checked semantic substrate consumed by the distinct Property, Behavior, and Query
languages; it is not another scenario language. Checked types freeze canonical metadata and
semantic digests, and Planning accepts checked values rather than raw author input.

## Core and Target APIs

`Umpire.Core` defines the vocabulary shared by every other module. `Umpire.Target` owns target
authoring, validation, canonicalization, and checked composition.

Important value types:

- `DeclarationId` identifies semantic declarations. Public identities are expected to be
  namespaced; construct them with `DeclarationId.of`.
- `DeclarationMetadata` describes a state, action, outcome, observation, relation, capability,
  provider, law, connector, target, or kernel.
- `SemanticValue` pairs a declaration identity with a canonical string value.
- `SemanticTrace` and `SemanticTraceStep` represent pure model traces.
- `TransitionResult` represents one model-owned transition result.
- `TypedBound` associates a bound with an explicit `BoundUnit`.

Target composition uses:

- `TransitionKernel` — finite initial-state and transition enumerators accompanied by soundness and
  completeness proofs.
- `CapabilityProvider` — meanings and law witnesses supplied by one capability.
- `CapabilityConnector` — explicit reconciliation of meanings supplied by multiple providers.
- `TargetDefinition` — the ordinary semantic vocabulary and transition-kernel input.
- `TargetComposition` — a sealed builder that collects explicit provider and connector choices.
- `AuthoredTarget` — the sealed result of `AuthoredTarget.make`, plus optional Target-owned finite
  planning evidence and compiler-only occurrences.
- `TargetDeclaration` — the lower-level typed composition used by Target maintainers.
- `CheckedTarget` — validated and canonicalized target.

The ordinary checked entry point is:

```lean
checkTarget :
  AuthoredTarget LawStatement Setup State Action Outcome Observation →
  Except AuthoringDiagnostic
    (CheckedTarget LawStatement Setup State Action Outcome Observation)
```

`elaborateTarget` emits the same typed failure at the captured Lean source occurrence.
`checkedTarget` produces the value for a declaration that compiles as valid while hiding extraction
and proof-relation re-ascription. Both `AuthoredTarget` and `CheckedTarget` have sealed constructors;
ordinary maintainers use `TargetComposition.provide`, `TargetComposition.connect`, and
`AuthoredTarget.make`. `composeTarget` remains the lower-level
`Except DeclarationError` expert seam. `canonicalCheckedTargetJson` returns the checked target's
canonical projection; compiler-only occurrence spans never enter it.

## Property API

`Umpire.Property` describes portable claims over capability-limited semantic traces.

The main authoring types are:

- `PropertyPattern`
- `ValueConstraint`
- `PropertyBound`
- `PropertyClause`
- `PropertyDeclaration`
- `PropertyCheckContext`

`PropertyClause` supports state invariants, transition contracts, identity relations,
input/output relationships, ordering, bounded eventuality, and bounded quiescence.

Main entry points:

```lean
PropertyCheckContext.ofTarget
checkProperty
evaluateProperty
```

`checkProperty` returns either `PropertyError` or `CheckedProperty`. `evaluateProperty` reduces an
unrestricted semantic trace to the checked property's admitted capability view before evaluating
its clauses.

## Behavior API

`Umpire.Behavior` constrains setup and trace shape without assigning target outcomes.

Important types:

- `ResourceRole`, `RoleBinding`
- `SetupConstraint`
- `NamedOccurrence`, `OccurrenceBound`, `OccurrenceOrder`
- `AuthoredExactTrace`
- `BehaviorDeclaration`
- `CheckedBehavior`
- `BehaviorTrace`

Convenience constructors include:

```lean
OccurrenceBound.exactly
OccurrenceBound.atLeast
OccurrenceBound.atMost
```

The main entry point is:

```lean
checkBehavior :
  BehaviorCheckContext →
  BehaviorDeclaration →
  Except BehaviorError CheckedBehavior
```

Ordinary callers derive the context with `BehaviorCheckContext.ofTarget target`; direct declaration
contexts remain confined to Behavior's focused lower-level fixtures.

Checking validates identities and references, canonicalizes constraints, rejects contradictions,
and records whether the described behavior space is statically unsatisfiable. It does not select a
target or enumerate a trace.

`CheckedBehavior.assignOccurrences` canonically attributes selected action positions to authored
required occurrences. Behavior admission and Artifact linear extensions cross this same seam.

## Observation API

`Umpire.Observation` describes, checks, and applies bounded mappings from typed synthetic evidence
to qualified semantic traces. Import the complete public surface with:

```lean
import Umpire.Observation
```

The offline lifecycle is:

```text
EvidenceProfileDeclaration + ObservationMappingDeclaration + CheckedTarget
  ── ObservationCheckContext.ofTarget / checkObservation ──▶ CheckedObservationPlan
CheckedObservationPlan + synthetic EvidenceBundle
  ── qualifyEvidence ──▶ QualificationResult
CheckedQuery + CheckedProperty + QualificationResult
  ── evaluateQualifiedProperty ──▶ SemanticPropertyVerdict
CheckedQuery + property verdicts
  ── summarizeQueryVerdicts ──▶ StrictQuerySummary
```

`checkObservation` compiles the closed expression grammar, declared ordering and closures, semantic
outputs, field dispositions, and positive `evidence-records` bound into one canonical
`CheckedObservationPlan`. `qualifyEvidence` then either returns one complete `QualifiedTrace` with
a derivation for every semantic coordinate, or one closed diagnostic without exposing a partial
trace. The raw `EvidenceBundle` is consumed only during qualification and is not retained in the
qualified trace or verdicts.

Every consumed field has exactly one disposition:

- `retain` may preserve its approved normalized value;
- `redact` may preserve only a contribution marker;
- `hash` may preserve only a deterministic token under the mapping's named, versioned synthetic
  digest policy;
- `reject` prevents qualification when that field is present and cannot be read by a mapping.

Qualification statuses are `qualified`, `unknown`, `conflict`, and `unsupported`. Property verdicts
are independently `satisfied`, `violated`, `unknown`, `conflict`, or `unsupported`; qualification
failure never becomes a Property violation. Strict aggregation is `satisfied` only for exactly one
resolved satisfied verdict per required property over the same query, trace, and evidence bound. It
is `violated` only when that result set is structurally complete and resolved but contains a
violation. Missing, duplicate, unexpected, divergent, wrong-query, cross-trace, cross-bound, or
unresolved results make the summary `incomplete`.

### Future adapter handoff

The exact typed input a future runtime adapter would have to provide is one complete
`EvidenceBundle`:

- `profile` and `profileVersion` select the checked evidence schema;
- `records` contain `id`, the same `profile` and `profileVersion`, `kind`, one-based `sequence`, and
  `causalParents`; each record's `fields` pair a field identity with a typed text, natural, or
  Boolean value plus optional `digestPolicy` and `reportedDigestToken`, while `bindingFacts` pair a
  binding identity with a typed value and `faultTarget` optionally identifies the intended target;
- `closures` pair each required evidence kind with its final sequence;
- `compatibleAlternatives` preserve alternative identities and their evidence identities instead
  of selecting one silently;
- `missingDiscriminator` names the fact needed to distinguish those alternatives.

Such an adapter would own translation into these fields while preserving source identity,
ordering, causality, closure, ambiguity, and declared field types. It would hand the complete bundle
to `qualifyEvidence`; it would not construct a `QualifiedTrace`, choose an offline status, or
reinterpret a Property result. This release provides no such adapter: it does not start Temporal,
execute operations, collect live evidence, persist raw records, prove runtime conformance, promote
results, or admit another evidence profile. Observation also does not redefine Property meaning.

## Query and search API

`Umpire.Query` combines a checked target, checked properties, checked behavior, finite bounds, and
deterministic planning policy.

Important types:

- `QueryTarget`
- `QueryForm`
- `QueryDeclaration`
- `CheckedQuery`
- `FiniteCompletenessEvidence`
- `QueryCheckContext`
- `QueryBounds`
- `PlannerPolicy`
- `SearchStrategy`

`QueryForm` determines the meaning of the result:

- `verify` searches for bounded universal verification.
- `witness` searches for a satisfying trace.
- `counterexample` searches for a violating trace.
- `select` performs bounded exploratory selection.

The checked entry point is:

```lean
checkQuery :
  QueryCheckContext LawStatement →
  QueryDeclaration →
  Except QueryError (CheckedQuery LawStatement)
```

`QueryCheckContext.ofTarget` derives the Query view from the checked Target, including any available
finite completeness contract. An exhaustive Query rejects a Target that explicitly lacks that
capability. `QueryBounds` keeps behavior-space bounds separate from the planner's
candidate-evaluation budget.

## Planning API

`Umpire.Planning` performs deterministic, incremental enumeration of a checked target.

The principal types are:

- `IncrementalPlannerKernel`
- `FiniteKernelOrder`
- `PlanningOutcome`
- `PlanningResult`
- `PlannerRun`
- `PlannerInstrumentation`

The main entry point is:

```lean
plan :
  (query : CheckedQuery LawStatement) →
  IncrementalPlannerKernel query.target →
  PlannerRun
```

`IncrementalPlannerKernel.ofCheckedQuery?` derives indexed enumeration, bounds, soundness, and
completeness from the checked Query and Target-owned finite domain. A model maintainer supplies only
the canonical-order proofs; ordinary Query authors do not construct a planner kernel.

`IncrementalPlannerKernel` exposes indexed action, initial-state, and transition enumeration. Its
proof fields establish soundness, completeness, and canonical ordering relative to the selected
target.

`PlanningOutcome` distinguishes a selected trace, bounded verification, complete absence of a
matching trace, budget exhaustion, unsatisfiable behavior, and an invalid query. `PlannerRun`
contains the outcome, optional artifact, and instrumentation.

## Artifact API

`Umpire.Artifact` defines portable, environment-independent products of pure model planning.

- `DrivePlan` records bindings, requested actions, model-owned outcomes, resulting states,
  checkpoints, bounds, selection reason, and provenance.
- `ExperimentSpec` is the portable envelope consumed by later execution, checking, replay, and
  generation work.
- `artifactOfSelection` constructs an `ExperimentSpec` from a checked query and a selected,
  kernel-produced `BehaviorTrace`.

Canonical serialization entry points include:

```lean
canonicalCheckedTargetJson
canonicalPropertyJson
canonicalBehaviorJson
canonicalQueryJson
canonicalDrivePlanJson
canonicalExperimentSpecJson
```

Canonical error projections are available as `canonicalDeclarationErrorJson`,
`canonicalPropertyErrorJson`, `canonicalBehaviorErrorJson`, and `canonicalQueryErrorJson`.

Artifacts do not claim that a runtime action occurred or that execution evidence was collected.

## Reference example

[`Umpire.Examples.Switch`](Examples/Switch.lean) is the smallest complete example. Read its public
declarations in lifecycle order:

1. `transitionKernel`
2. `targetDeclaration`, `targetAuthoring`, and checked `target`
3. `propertyDeclaration` and `flipProperty`
4. the behavior declarations and checked behaviors
5. the query declarations and checked queries
6. `incrementalKernel` and the planner runs
7. `compiledArtifact`

The example exposes exploratory, exact-action, and exact-trace variants without depending on
Temporal-specific modules.

## API invariants

- Authored declarations remain distinguishable from checked values.
- Invalid inputs cross public boundaries as typed `Except` errors.
- Semantic identities derive from canonical projections, not source declaration order.
- Transition outcomes belong to the target model, not to behavior declarations.
- Planning is pure and performs no runtime execution, evidence collection, or promotion.
- Reusable Umpire modules must not depend on Temporal-specific modules.
