# Umpire property test structure

## Status

Superseded by `2026-08-25-lean-test-suite-structure-design.md`.

## Goal

Make the Umpire property regression tests easier to navigate and maintain while retaining the
project's existing Lean testing model: pure fixtures and compile-time assertions elaborated as part
of the `UmpireTests` library.

The refactor must preserve intended regression coverage and all existing comments, except for the
explicit removal of one vacuous assertion and the narrowing of one negative assertion to its
existing focused fixture. It must not introduce a custom test framework or change the production
`Umpire.Property` API.

## Module structure

`Umpire/Property/Tests.lean` becomes an import-only test facade:

```text
Umpire/Property/
├── Tests.lean
└── Tests/
    ├── Fixtures.lean
    ├── Evaluation.lean
    ├── Validation.lean
    ├── LogicalTime.lean
    └── Canonicalization.lean
```

The facade imports the four assertion modules. `UmpireTests.lean` continues importing only
`Umpire.Property.Tests`, so the aggregate test boundary and existing build commands remain stable.

Each module begins with a short module comment describing its concern. Test declarations remain
under `Umpire.PropertyTests`; shared fixtures live under `Umpire.PropertyTests.Fixtures` and are
opened explicitly by the assertion modules.

## Shared fixture boundary

`Fixtures.lean` imports `Umpire.Property` and exposes only the common vocabulary and base values
used by multiple assertion modules:

- declaration identifiers, metadata constructors, meanings, and the property-checking context;
- reusable pattern and semantic-value constructors;
- the base clauses, portable property, authored property, and positive trace;
- `evaluationOf` and `errorKindOf` result helpers.

Derived or malformed declarations remain private to the module that tests them. In particular,
negative traces, reordered inputs, logical-time traces, invalid properties, and digest mutations do
not become shared fixture API. This keeps the fixture module deep enough to remove setup noise
without coupling unrelated test cases through every variation.

## Test responsibilities

### Evaluation

`Evaluation.lean` covers successful checking and evaluation, the focused uniqueness failure,
same-position boundaries, evaluator/theorem agreement, filtering of hidden observations, and
evaluation-result evidence.

The negative uniqueness assertion evaluates `uniquenessProperty`, rather than the complete portable
property, so an unrelated failing clause cannot satisfy the expected negative result. The existing
`uniquenessProperty` fixture is retained and used.

### Validation

`Validation.lean` covers undeclared references, incompatible bound units, missing logical-time
sources, and opaque declarations. Assertions continue checking the precise `PropertyErrorKind`.

### Logical time

`LogicalTime.lean` owns logical-time property variants and their trace constructor. It covers valid
time, absent time, malformed values, and decreasing time for both eventual and quiescent clauses.

### Canonicalization

`Canonicalization.lean` covers input-order invariance and digest sensitivity to constructor,
reference, and bound changes. The reflexive canonicalization assertion is removed because comparing
an expression with itself using `rfl` cannot detect a regression. No golden canonical JSON string is
added unless the serialized representation is explicitly intended to be a compatibility contract.

## Assertion style

Closed computational regressions remain anonymous `example` declarations proved with
`native_decide`. Direct theorem application remains an ordinary proof. `rfl` is used only when
definitional equality is itself the contract under test, not as a self-comparison smoke check.

Short related assertions may remain adjacent, but each derived fixture appears immediately before
the assertions that consume it. Existing explanatory comments move with their declarations and are
not rewritten.

## Build integration

This refactor does not change `lakefile.toml`, the default targets, or the root Makefile. Adding a
package `testDriver` for conventional `lake test` support is a separate project-level improvement
because it must aggregate both `UmpireTests` and `TemporalUmpireTests`.

## Verification

From the repository root:

```sh
(cd model && mise exec -- lake build UmpireTests)
make umpire-check-regression
git diff --check
```

Verification must also confirm that `UmpireTests.lean` still imports the property test facade, every
new assertion module is imported by that facade, the vacuous self-comparison is gone, and the
focused uniqueness property is exercised.

## Non-goals

- No production behavior or public API changes.
- No custom test runner, assertion DSL, property-based testing dependency, or runtime IO tests.
- No reorganization of the Behavior, Query, Planning, Core, or Temporal test modules.
- No Lake or Make workflow changes.
