# Lean test suite structure

## Status

Approved on 2026-08-25.

## Goal

Split every Lean regression suite currently larger than 300 lines into cohesive concern modules
while retaining the project's elaboration-based testing model. Each original test root remains a
stable import-only facade or is replaced by the final owner-specific aggregate already approved by
the Temporal semantic-layout migration.

The refactor covers these seven suites:

- `model/Temporal/Umpire/ConfigTests.lean`
- `model/Umpire/CoreTests.lean`
- `model/Umpire/Behavior/Tests.lean`
- `model/Umpire/Property/Tests.lean`
- `model/Umpire/Planning/Tests.lean`
- `model/Umpire/Query/Tests.lean`
- `model/TemporalUmpireTests.lean`

All existing explanatory comments remain attached to the declarations or assertions they explain.
Production DSL behavior and public APIs do not change.

## Shared conventions

An existing `Tests.lean` or `CoreTests.lean` root becomes an import-only facade. Concern modules
import a sibling `Fixtures` module rather than the facade, preventing import cycles. Fixture modules
expose only base vocabulary, constructors, contexts, and operations used by more than one concern;
derived variants remain private to their consumer.

Assertions remain anonymous `example` declarations. Closed computations continue using
`native_decide`, while direct theorem applications remain ordinary proofs. `rfl` is used only when
definitional equality is itself the contract under test.

Declaration namespaces remain stable where the semantic-layout migration does not already require a
namespace move. Each new file receives a short module comment. Existing comments are moved
verbatim, not rewritten.

Semantic source strings are fixture data. A mechanical file split does not rewrite them merely to
match a leaf filename. The active synthetic-vocabulary and Temporal ownership migrations may change
them when their approved semantic contract requires it; the subsequent split preserves the value
established by that migration.

## Umpire Core

```text
model/Umpire/CoreTests.lean
model/Umpire/CoreTests/
├── Fixtures.lean
├── Composition.lean
├── Validation.lean
├── KernelSoundness.lean
├── Canonicalization.lean
└── Trace.lean
```

`Fixtures` owns declaration constructors, test laws and witnesses, the baseline kernel, providers,
connector, target, and error projection. Composition owns successful baseline and alternate-model
composition. Validation owns declaration, capability, provider, connector, and law failures. Kernel
soundness owns incomplete-kernel rejection and proof obligations. Canonicalization owns ordering,
digest sensitivity, documentation behavior, and serializer availability. Trace owns exact semantic
trace representation.

All 31 current assertions remain.

## Umpire Behavior

```text
model/Umpire/Behavior/Tests.lean
model/Umpire/Behavior/Tests/
├── Fixtures.lean
├── Admission.lean
├── Validation.lean
├── Canonicalization.lean
└── Narrowing.lean
```

`Fixtures` owns the base context, semantic values, traces, occurrences, exact witness, constrained
declaration, and admission helper. Admission covers allowed, forbidden, ordering, adjacency, and
action-exact versus trace-exact behavior. Validation covers authoring errors, unsatisfiability,
schedule contradictions, and occurrence guards. Canonicalization owns ordering and digest behavior.
Narrowing owns the constraint-refinement law.

The reflexive canonical self-comparison is removed because it cannot detect a regression. The other
23 assertions and all five explanatory comments remain.

## Umpire Property

```text
model/Umpire/Property/Tests.lean
model/Umpire/Property/Tests/
├── Fixtures.lean
├── Evaluation.lean
├── Validation.lean
├── LogicalTime.lean
└── Canonicalization.lean
```

`Fixtures` owns the common semantic vocabulary, context, clauses, base property, positive trace,
and result helpers. Evaluation covers successful checking, focused uniqueness failure, boundaries,
the evaluator theorem, hidden observations, and result evidence. Validation owns malformed
declarations and authoring modes. Logical time owns its property variants and trace constructor.
Canonicalization owns ordering and digest sensitivity.

The negative uniqueness assertion evaluates the existing `uniquenessProperty`. The reflexive
canonical self-comparison is removed. The resulting suite contains 24 meaningful assertions,
including the existing direct theorem proof and its unchanged comment.

## Umpire Planning

```text
model/Umpire/Planning/Tests.lean
model/Umpire/Planning/Tests/
├── Fixtures.lean
├── Outcomes.lean
├── Artifacts.lean
└── Enumeration.lean
model/Umpire/Planning/VisibilityTests.lean
```

`Fixtures` owns the deterministic model, checked query, incremental kernel, and planning runner.
Outcomes covers query forms, invalid declarations, absence, exhaustion, and unsatisfiable behavior.
Artifacts covers witness construction, optional occurrences, byte stability, and semantic identity.
Enumeration owns cursor laziness and instrumentation. `VisibilityTests.lean` remains separate and
continues importing only the public `Umpire.Planning` facade.

All 10 assertions and both visibility guards remain.

## Umpire Query

```text
model/Umpire/Query/Tests.lean
model/Umpire/Query/Tests/
├── Fixtures.lean
├── Visibility.lean
├── Forms.lean
├── Completeness.lean
├── Validation.lean
└── Identity.lean
```

`Fixtures` owns the shared model, checked property and behavior, completeness profile, declaration,
and error projection. Visibility imports only `Umpire.Query` so it remains a genuine public-facade
test. Forms covers quantifier and claim semantics. Completeness covers finite-domain requirements
and partial completeness. Validation owns exact-trace and bound errors. Identity covers canonical
projection and semantic identity.

All 10 assertions, the visibility guard, and existing comments remain.

## Temporal configuration

The existing semantic-layout migration owns the physical and namespace cutover. The 609-line legacy
suite is not split in place.

```text
model/Temporal/System/Configuration/Tests.lean
model/Temporal/System/Configuration/Tests/
├── Fixtures.lean
├── Validation.lean
├── Resolution.lean
└── Catalog.lean
model/Temporal/System/Callback/ConfigurationTests.lean
```

The shared Configuration facade imports validation, resolution, and catalog concerns. Its fixture
module contains only generic checked-use and view construction shared across those concerns.
Validation owns checked-use, override, and setting-structure failures. Resolution owns deterministic
resolution, context isolation, typed reads, immutable views, and mixed Callback/Matching provenance.
Catalog owns constrained defaults, fixture conformance, and opaque-default replacement and drift.

Callback tests remain one cohesive module because the owner split reduces them below the large-suite
threshold. They own address decoding and policy, consumer projections, routing, admission, dispatch,
timeouts, captured snapshots, and projection failures.

All 30 legacy assertions move exactly once. This follow-up evaluates the final shared
Configuration suite after `fn-10` completes and splits it only if the ownership migration leaves it
above the large-suite threshold. The already-cohesive Callback suite remains unchanged.

## Temporal model aggregate

The mixed `TemporalUmpireTests.lean` root disappears through the already-approved Feature/System/Tool
cutover rather than becoming another facade.

```text
model/Temporal/Feature/Nexus/CallerClosureTests.lean
model/Temporal/System/Configuration/Tests.lean
model/Temporal/System/Callback/ConfigurationTests.lean
model/Temporal/Tool/InspectTests.lean
model/Umpire/Examples/SwitchTests.lean
model/TemporalModelTests.lean
```

Feature assertions move to `CallerClosureTests`. Reusable Switch assertions move to their existing
owner. CLI success, repeatability, unknown-scenario, and failed-scenario assertions move to
`InspectTests`. `TemporalModelTests.lean` is import-only and does not import `UmpireTests` or reusable
test fixtures.

All 41 current assertions remain. The invalid-arity assertion required by the semantic-layout spec
is added by its owning Tool task, not by this mechanical preservation count.

## Sequencing and task ownership

This design becomes a new Flow-Next follow-up spec whose prerequisite is
`fn-10-temporal-semantic-model-layout-and`. It does not amend, narrow, or expand any `fn-10` task.
Planning begins from the completed post-`fn-10` layout so paths, namespaces, and ownership are not
specified against an intermediate migration state.

Each of the seven original large suites is evaluated in that final layout. Core, Behavior, Property,
Planning, and Query are expected to remain large and receive independent split tasks. Configuration
receives a split task only if its final shared owner suite remains above the threshold. The former
mixed Temporal aggregate is expected to have become smaller owner-specific modules; those modules
are accepted without another split when they are below the threshold.

Every resulting split task is implemented by a fresh sub-agent that edits only its assigned suite
and does not commit. Independent suite tasks may run concurrently. The root agent reviews every
diff, checks cross-suite conflicts, and runs the full regression gate after all suite tasks converge.

## Verification

Each worker runs `mise exec -- lake env lean` for every leaf it creates, followed by the narrowest
aggregate build that reaches the suite. The root integration pass runs from the repository root:

```sh
(cd model && mise exec -- lake build UmpireTests TemporalModelTests)
make umpire-check-regression
git diff --check
```

Inventory checks confirm preserved assertion and comment counts, facade coverage of every leaf,
absence of child-to-facade import cycles, removal of both vacuous canonical assertions, focused use
of `uniquenessProperty`, and absence of forbidden Temporal vocabulary below `model/Umpire/**`.

## Non-goals

- No production DSL behavior or public API changes.
- No custom test framework, assertion DSL, new dependency, runtime IO suite, or property-testing
  framework.
- No split of test modules already below the threshold after semantic ownership is applied.
- No Lake, Make, CI, or documentation workflow change beyond the existing semantic-layout cutover.
- No commits by implementation agents or the coordinating agent.
