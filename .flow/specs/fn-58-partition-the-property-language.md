# Partition the Property language implementation

## Overview

Partition Property authoring vocabulary, typed checking and canonicalization, checked trace projection, and clause evaluation into cohesive internal modules behind the unchanged `Umpire.Property` facade. Preserve every public name, type, diagnostic, theorem, trace rule, Limit, fingerprint, and evaluation result.

## Goal & Context
<!-- scope: business -->

The Property language implementation currently combines four distinct responsibilities in one large module. Model authors should continue using one focused facade, while maintainers gain smaller modules whose interfaces align with authoring, checking, trace projection, and evaluation concerns.

Property authors, Observation verdicts, Planning, inspection tools, and Temporal models see no source or semantic change. The refactor improves navigation, focused compilation, and isolated testing without adding another Property language.

## Architecture & Data Models
<!-- scope: technical -->

Four modules form one acyclic downward import chain:

1. `Umpire.Property.Language` owns authoring vocabulary and inert helpers: trace fields, value constraints, patterns, Limits, clauses, declarations, authoring sentinels, and the exact-pattern constructor.
2. `Umpire.Property.Check` owns typed errors, capability and check contexts, resolved and checked types, canonical rendering, `checkProperty`, and the explicit-proof `checkedProperty` convenience.
3. `Umpire.Property.Trace` owns the Property-specific Model Coordinate adapter, capability-limited trace steps and views, and checked-property trace projection.
4. `Umpire.Property.Evaluation` owns constraint and pattern meaning, occurrences, clause interpreters and agreement theorems, spans, results, and `evaluateProperty`.

All declarations remain in namespace `Umpire`; only physical ownership and import direction change. The existing `Umpire.Property` facade imports the four modules and remains the sole normal consumer surface. No standalone Constructors module is added because the existing helpers belong beside their principal types or checker.

## API Contracts
<!-- scope: technical -->

- `import Umpire.Property` exposes every existing Property declaration with unchanged fully qualified names and types. Behavior and Query authoring declarations remain absent from this narrow facade.
- `PropertyPattern.exact`, `checkProperty`, `checkedProperty`, `CheckedProperty.traceView`, `PropertyTraceField.valueAt?`, `evaluateProperty`, and public agreement theorems retain their exact interfaces and behavior.
- Property checking preserves typed error fields and rendering, capability view, resolved Limits, canonical metadata, validation order, source fallback, and Behavior Fingerprint inputs.
- Property trace projection preserves capability filtering and the existing strict coordinate and prior-state semantics.
- Evaluation preserves the denotational and executable meaning of every clause, Bool/Prop agreement, result spans, evaluated Limits, and semantic provenance.
- The package-level facade check retains the existing direct Language import contract while aggregating Check, Trace, and Evaluation.

## Edge Cases & Constraints
<!-- scope: technical -->

- Opaque or malformed declarations, duplicate IDs, unknown or wrong-kind references, missing or unknown capabilities, duplicate profiles, invalid clauses, invalid field and unit combinations, missing named Limits, and missing logical time retain typed fail-closed diagnostics and precedence.
- Equivalent reordering of definitions, providers, meanings, and clauses preserves canonical identity. Documentation and source changes remain fingerprint-neutral; semantic clause, reference, capability, and Limit changes still alter it.
- Capability-hidden values remain absent from the trace view and cannot satisfy a Property.
- Empty matches or triggers, missing, malformed, or regressing logical time, same-position ordering, strict ordering, inclusive bounded windows, and zero-distance eventuality retain current truth and failure behavior.
- All seven clause meanings, occurrence construction, span selection, and semantic provenance remain exact. Public agreement theorems retain their statements and axiom inventories.
- Internal modules import only downward, reusable Property code does not import Temporal, and public consumers do not migrate to child modules.
- At ten times declaration or trace length, algorithms, allocations, and asymptotic complexity remain unchanged; no cache, extra traversal, alternate representation, or adapter is introduced.
- Checking and evaluation remain pure total Lean code. No runtime, I/O, credential, concurrency, recovery, dependency, axiom, or compiler-trust surface is added.
- Existing comments and docstrings move intact. Architecture documentation describes internal ownership without changing public author guidance.

## Approach

1. Extract Property-owned trace projection and executable/denotational evaluation while temporarily retaining checking in Language.
2. Extract typed checking and canonicalization, then complete the final Language-to-Check-to-Trace-to-Evaluation import chain.
3. Strengthen facade and typed diagnostic characterization, document internal ownership, audit trust and comments, and run aggregate gates.

## Quick commands

```bash
cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Property.ImportTests Umpire.Observation.Tests.Verdict
cd model && mise exec -- lake build UmpireTests Temporal TemporalModelTests TemporalExperimentalTests
make umpire-build-model
make umpire-check-regression
make lint-model
make lint-code
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Language, Check, Trace, and Evaluation compile as one acyclic downward internal module chain, while `import Umpire.Property` exposes every existing public declaration with unchanged names and types and continues excluding Behavior and Query authoring declarations. Errors: a missing facade symbol, newly exposed language, direct or transitive Temporal dependency, consumer migration to a child module, package-gate drift, or import cycle fails completion; there is no runtime error surface.
- **R2:** Property checking retains exact validation order, typed error fields and rendering, capability view, Limit resolution, canonical metadata, source behavior, and Behavior Fingerprint. Errors: every existing Property error kind retains its kind, owner, source path, offending value, related identities, and precedence for malformed or duplicate IDs, unknown or wrong-kind references, missing or unknown capabilities, duplicate profiles, invalid clauses, fields or units, missing named Limits, and missing logical time; failure returns no partial checked property.
- **R3:** Capability filtering, coordinate compatibility, logical-time fail-closed behavior, all clause meanings, executable/denotational agreement, trace spans, evaluated Limits, and semantic provenance remain unchanged. Errors and boundaries: hidden values, empty matches or triggers, missing, malformed, or regressing logical time, same-position and strict ordering, inclusive bounded windows, and zero-distance eventuality retain current results and evidence; no partial semantic success is introduced.
- **R4:** Existing comments and docstrings move intact, architecture documentation describes the internal ownership while preserving public guidance, and focused plus aggregate builds, regression, lint, facade, artifact, and trust checks pass. Errors: lost comment, changed theorem statement or axiom inventory, generated or artifact byte drift, warning, import-boundary violation, stale documentation, new DSL or dependency, or lint failure blocks completion.

## Early proof point

The first task must compile Trace and Evaluation independently while the public facade and complete positive, negative, hidden-capability, logical-time, agreement, span, and provenance tests remain unchanged. If clause meaning requires a new public representation or the coordinate adapter cannot remain Property-owned, reconsider the import chain before extracting the checker.

## Boundaries
<!-- scope: business -->

- No new Property, Behavior, Query, Scenario, macro, builder, coercion-driven DSL, callback, or authoring representation.
- No public child-module contract, new constructor module, adapter, compatibility layer, or exported internal helper.
- No Property declaration, error, canonical identity, fingerprint, trace, clause, Limit, result, theorem, or provenance semantic change.
- No Observation production-code, Planning, runtime, artifact schema, generated-file, persisted-byte, or package-gate change.
- No new dependency, axiom, native-decision default, cache, performance feature, or CI workflow.

## Decision Context
<!-- scope: both — conditionally substructured -->

The module chain follows data flow: inert authoring feeds deterministic checking; checked declarations project a capability-limited trace; evaluation interprets clauses over that view. The public facade hides all four. A separate Constructors module is rejected because the existing exact-pattern and proof-taking helpers are shallow when detached from their principal types and checker.

Completed ordinary-authoring and coordinate work owns `PropertyPattern.exact`, `checkedProperty`, canonical identity, and strict prior-state semantics; this plan moves those contracts intact. No dependency on the Observation evaluator or authoring partitions is required. The small documentation task should refresh its anchors after concurrent Observation documentation edits rather than coupling the implementations.

Four modules add import and navigation edges in exchange for substantially better locality and focused verification. Runtime algorithms, allocations, asymptotic performance, scalability, pure crash behavior, information-flow security, and trust remain unchanged.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Stable facade and acyclic internal modules | `.1`–`.3` | — |
| R2 | Exact checker, diagnostics, identity, and fingerprint | `.2` | — |
| R3 | Exact trace projection and clause evaluation | `.1` | — |
| R4 | Comments, docs, trust, regression, and lint compatibility | `.1`–`.3` | — |

## References

- Umpire 4 rules SEM-04, SEM-05, SEM-09, AUT-01, AUT-03, AUT-04, AUT-07, MOD-01, MOD-06 through MOD-08, and PLN-01 define the separate pure Property language, checked stable declarations, bounded progress, reusable module isolation, and explicit Limits.
- Lean Authoring Guidelines sections 2, 4, 5, and 6 govern interfaces, module documentation, trust, and verification.
- Completed ordinary-authoring and accepted-trace coordinate specs define the public helpers, canonical identity, and coordinate semantics preserved here.
