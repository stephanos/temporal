# Separate Lean Target semantics from authoring machinery

## Goal & Context
<!-- scope: business -->

Semantic consumers need the checked Target and its kernel obligations, but currently inherit Target authoring machinery. `Umpire.Target.Language` combines `CheckedTarget`, finite planning evidence, composition validation, occurrence diagnostics, canonical projections, and `Lean.Elab.Term` integration. `Umpire.Property.Language` imports the entire `Umpire.Target` facade; Query and Planning inherit that dependency.

Separate these responsibilities so model evaluation and planning can depend on an intentional semantic interface. Preserve Target, Property, Behavior, and Query as their existing semantic owners, and preserve the established finite-table and finite-machine authoring improvements. This is architecture-review item 6's Target seam, an independent track alongside the ongoing fn-68 Nexus3 demonstration.

## Architecture & Data Models
<!-- scope: technical -->

The dependency boundary has four responsibilities:

- The Target semantic interface owns the checked Target, its authoritative kernel access, resolved setups, semantic identity, and finite planning availability tied to that exact kernel. Existing `TransitionKernel` and its soundness, completeness, domain, and closure contracts remain the kernel authority.
- Pure Target admission owns composition validation and construction of checked values. It shares that authority with the semantic representation without exposing an unchecked construction API. Splitting modules must respect Lean's private-constructor visibility; a public raw constructor or user-supplied validity flag is not an acceptable shortcut.
- Target projection/serialization owns behavior-description construction and canonical encodings. Serialization helpers can be pure dependencies below admission where needed to derive checked fingerprints; they must not depend on the authoring frontend. Stored canonical metadata or behavior descriptions may remain available on checked values. This work does not require removing the existing `Lean.Data.Json` dependency from Core or redesigning persistence.
- Target authoring owns authored composition, source-occurrence collection, located diagnostics, syntax capture, and elaborator integration. The elaborator invokes the same pure checking authority and maps its errors to syntax locations; it does not add a second validator.

Expose a narrow semantic import, conventionally `Umpire.Target.Semantics`, and retain `Umpire.Target` as the ordinary authoring facade. Existing `Umpire.Target.Language` imports may remain a compatibility facade; semantic consumers must not use that broad facade. Property, Behavior, Query, and Planning semantic implementation modules use the narrow surface or the smaller dependency they actually need. Their transitive imports must not reintroduce Target's elaboration frontend through an authoring convenience import.

Do not indiscriminately wrap kernel fields. Kernel access is a legitimate expert semantic interface. Add semantic accessors or lemmas only where migrating an actual consumer reveals coupling to checked-value assembly or incidental finite-list representation. Keep the relation-indexed proof obligations visible and keep finite enumeration construction inside the existing `FiniteTable`, `ValidatedFiniteTable`, `FiniteMachine`, and `ValidatedFiniteModel` owners.

## API Contracts
<!-- scope: technical -->

`CheckedTarget` remains parameterized by its law statement and Setup, State, Action, Outcome, and Observation types. Its kernel's authoritative initial and step relations, resolved setups, Definition ID, Behavior Fingerprint, and optional planning evidence retain their meaning. `FinitePlanningCapability` remains indexed by the authoritative step relation and retains both action soundness and action completeness.

Preserve the public authoring entrypoints and their result distinctions: `composeTarget` returns `Except DefinitionError CheckedTarget`; `checkTarget` returns `Except AuthoringDiagnostic CheckedTarget`; `checkedTarget` retains its checked extraction contract; and `elaborateTarget` returns a checked value in `TermElabM` or raises the located authoring diagnostic. Preserve the existing namespace-qualified APIs exposed by `Umpire.Target`, including finite-machine/table adapters. Moving declarations between modules does not authorize semantic changes.

`CheckedTarget.withEquivalentKernel` remains available as the expert enumerator-replacement seam. Its obligations continue to establish matching metadata and semantic domains, equal authoritative initial/step relations, and an unchanged behavior description. Replacement planning evidence remains indexed by the replacement relation. Preserve its existing handling of absent planning evidence; do not infer completeness from the mere existence of finite descriptions.

Canonical Target metadata, behavior descriptions, DefinitionError JSON, authoring diagnostic JSON, and fingerprint computation have one implementation each. Preserve existing externally observable bytes for unchanged inputs. Source occurrences remain nonsemantic; neither syntax objects nor occurrence identities enter checked behavior or fingerprint inputs.

## Edge Cases & Constraints
<!-- scope: technical -->

Preserve validation of malformed and duplicate identities, wrong definition kinds, capability conflicts, missing law witnesses, missing/incomplete kernels, and missing/incomplete or invalidly encoded Behavior Domains. Located diagnostics retain the original/offending occurrence distinction, including fallback behavior when no captured syntax occurrence matches.

Empty domains, nondeterministic transition enumerations, duplicate or reordered authored entries, and unavailable finite planning retain their current meanings and validation behavior. Module extraction must not reorder initial states, transition results, observations, or planner candidates, broaden Limits, or strengthen a bounded claim.

Keep comments and relevant public documentation with their owners. No new libraries, Lean toolchain changes, axioms, `sorry`, or additional compiler-trust dependencies are authorized. Compare before/after axiom inventories for changed load-bearing declarations; new semantic lemmas must fit the existing approved trust boundary.

This refactor introduces no runtime service, state store, or concurrency mechanism. It must preserve existing finite enumeration complexity and planner memory behavior; it does not promise a particular compilation-speed improvement.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A semantic-only import exposes the checked Target/kernel and finite planning contracts without importing Target's elaboration frontend or `Lean.Elab.Term`, directly or transitively. Dedicated positive import tests and a negative import-graph regression enforce the boundary, including rejection of a bridge module that reintroduces the forbidden dependency. Errors: a forbidden dependency fails the existing import/lint gate with an identifiable dependency path.
- **R2:** Property, Behavior, Query, and Planning semantic consumers use the narrow dependency surface and compile with unchanged public semantic types and behavior. Their normal semantic import closure satisfies R1. Errors: no new error surface beyond their existing admission and evaluation errors; legitimate authoring consumers remain allowed to import the authoring facade.
- **R3:** Pure admission remains the single authority for constructing checked Targets, and the full authoring facade preserves its checked APIs and finite adapters. Import tests continue to reject direct access to private checked/authored constructors. Errors: existing Target validation rejection cases still reject with the same typed categories; no raw assembly bridge bypasses those checks.
- **R4:** Syntax capture and located diagnostics live in the authoring owner; canonical projections/serialization have a separate pure owner, with no authoring dependency below the semantic boundary. Existing authoring diagnostic regression tests retain selected source spans and canonical error output. Errors: missing captured locations retain the existing fallback, and all malformed authoring inputs remain rejected.
- **R5:** Existing Target canonical metadata and Behavior Fingerprint fixtures compare byte-for-byte without rewriting expected bytes to accommodate the refactor. Canonicalization/mutation tests retain the distinction between semantic changes and changes to documentation, source locations, occurrence metadata, or declaration order. Errors: any unexplained byte or fingerprint drift fails verification.
- **R6:** Finite-table/machine checks, checked kernel replacement, and planning/evaluation regressions establish unchanged authoritative relations, enumeration behavior, and completeness obligations. Any added accessor/lemma has a concrete migrated consumer and keeps representation details within its owner. Errors: unavailable planning remains unavailable, invalid finite tables still reject, and replacement kernels without the existing equivalence proofs remain unconstructible through the replacement API.
- **R7:** Changed trust-bearing declarations pass an explicit before/after transitive axiom audit with no expanded trust; focused Target/import/Property/Planning tests, all affected model consumers, and the repository's applicable build and lint gates pass. Use the existing `umpire-build-model`, `lint-model`, and required `lint-code` gates; run generator staleness checks if an owned generated surface changes. Errors: placeholders, new unsupported assumptions, import cycles, or unverified gate failures prevent a completion claim and are reported precisely.

## Boundaries
<!-- scope: business -->

- No new modeling DSL, replacement semantic owner, or broad authoring redesign.
- No requirement to hide legitimate expert `TransitionKernel` use behind forwarding wrappers.
- No model-to-Case lowering or completion of fn-68 Nexus3 work; preserve and coordinate its in-progress finite-table APIs and consumers.
- No SemanticInventory outcome/gap contract extraction; that is the separate inventory dependency-direction spec.
- No standalone Testpilot protocol, environment binding, Go driver extraction, or activation interpreter changes; those are separate specs.
- No blanket extraction of all Core JSON helpers, general-purpose finite proof library, or fingerprint migration.

## Decision Context
<!-- scope: both -->

The architecture review identifies a concrete dependency leak, not a need to replace Target/Property/Behavior/Query. Narrowing imports and moving implementation ownership gives semantic consumers a stable boundary while preserving mature checked adapters. Keeping pure canonical helpers usable below admission avoids circular imports without making syntax or elaboration part of semantic checking. Preserving constructor authority matters more than forcing an arbitrary number of files.

The existing `withEquivalentKernel` contract demonstrates an intentional expert seam and remains useful. Only demonstrated representation coupling warrants a new lemma or accessor; speculative wrappers would make the interface larger without reducing proof burden.

This spec has no hard dependency on the other five architecture specs or fn-68 completion. The inventory extraction may touch some of the same import sites, so reconcile concurrent import edits without absorbing its contracts. Coordinate against fn-68's current checked finite APIs and preserve source compatibility so the Nexus3 demonstration can continue independently.
