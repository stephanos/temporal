# Separate Lean Target semantics from authoring machinery

> HTML render lens: `.flow/artifacts/fn-75-separate-lean-target-semantics-from/spec.html` — open locally; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

Semantic consumers need the checked Target and its kernel obligations, but currently inherit Target authoring machinery. `Umpire.Target.Language` combines `CheckedTarget`, finite planning evidence, composition validation, occurrence diagnostics, canonical projections, and `Lean.Elab.Term` integration. `Umpire.Property.Language` imports the entire `Umpire.Target` facade; Query and Planning inherit that dependency.

Separate these responsibilities so model evaluation and planning can depend on an intentional semantic interface. Preserve Target, Property, Behavior, and Query as their existing semantic owners, and preserve the established finite-table and finite-machine authoring improvements. This is architecture-review item 6's Target seam. The delivered fn-68 Nexus3 demonstration and fn-78 scoped semantics are compatibility baselines; fn-74's completed Go activation and diagnostic changes remain outside this refactor.

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
- No model-to-Case lowering or additional fn-68 Nexus3 work; preserve its delivered finite-table APIs and consumers.
- No SemanticInventory outcome/gap contract extraction; that is the separate inventory dependency-direction spec.
- No standalone Testpilot protocol, environment binding, Go driver extraction, or activation interpreter changes; those are separate specs.
- No blanket extraction of all Core JSON helpers, general-purpose finite proof library, or fingerprint migration.

## Decision Context
<!-- scope: both -->

The architecture review identifies a concrete dependency leak, not a need to replace Target/Property/Behavior/Query. Narrowing imports and moving implementation ownership gives semantic consumers a stable boundary while preserving mature checked adapters. Keeping pure canonical helpers usable below admission avoids circular imports without making syntax or elaboration part of semantic checking. Preserving constructor authority matters more than forcing an arbitrary number of files.

The existing `withEquivalentKernel` contract demonstrates an intentional expert seam and remains useful. Only demonstrated representation coupling warrants a new lemma or accessor; speculative wrappers would make the interface larger without reducing proof burden.

This spec has no hard dependency on the other architecture specs. The fn-76 inventory extraction may touch some of the same import sites; coordinate those edits without absorbing its contracts or introducing a dependency solely for overlap. fn-77 consumes the resulting seam without adding typed-operation scope here. fn-70 remains independently deliverable, and fn-79 Nexus operation cancellation remains deferred.

## Implementation constraints and early proof

Capture the current trust inventories and canonical fixture hashes before the first extraction edit. Audit moved load-bearing declarations and the existing `checkedTarget` default argument separately; preserve its established public call shape without importing elaborator machinery through the semantic surface. Keep private checked assembly and its pure admission authority together, or use only proof-carrying checked operations across owners. A raw constructor bridge is prohibited. Pure occurrence and diagnostic data may remain below the frontend where admission needs them; syntax objects and elaboration remain above it.

The first task proves that the checked semantic surface can compile independently while the ordinary authoring facade still accepts existing callers. If private visibility or default-argument imports prevent that, revise the ownership split before migrating consumers; do not weaken R1 or public checked construction.

Import enforcement must inspect actual reachable dependency metadata, including external wrappers that indirectly import `Lean.Elab.Term`. First-party source reconciliation remains distinct from external dependency traversal. Pin the semantic roots precisely, including Property/Behavior/Query semantic implementations and Planning's Artifact/Types path; authoring facades and dedicated authoring tests remain permitted consumers of elaboration. Tests cover direct and bridged forbidden imports, external wrappers, pure imports, allowed authoring, cycles, deterministic diagnostics, and missing owned metadata.

Preserve existing comments with their owners, exact canonical fixtures, and source-span/fallback diagnostics. Keep unavailable planning unavailable; replacement tests cover both available and absent planning, enumeration order, and insufficient equivalence proofs. No performance improvement is claimed. Existing finite enumeration complexity and bounded planner memory remain the acceptance baseline.

Relevant prior constraints: checked input authority (`.flow/memory/bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05.md`), behavior-neutral extraction (`.flow/memory/bug/integration/behavior-neutral-refactors-must-not-2026-09-04.md`), and unchanged canonical defaults (`.flow/memory/bug/integration/default-empty-extensions-must-preserve-2026-09-05.md`). Broad generated API drift verification and CI expansion remain excluded per `.flow/memory/declined/generated-api-drift-verification.md`; existing applicable staleness checks remain required.

## Requirement coverage

| Requirement | Tasks | Verification |
| --- | --- | --- |
| R1 | .1, .2 | Positive semantic import plus actual transitive import graph and bridge regressions |
| R2 | .2 | Property, Behavior, Query, Planning and Artifact dependency closure |
| R3 | .1, .3 | Private constructor rejection and preserved checked authoring APIs |
| R4 | .1, .3 | Pure projection owner, syntax separation and unchanged located diagnostics |
| R5 | .1, .3 | Pre-edit fixture hashes and unchanged canonical/fingerprint tests |
| R6 | .1, .3 | Finite adapters, available/unavailable planning and kernel replacement tests |
| R7 | .1, .2, .3 | Pre/post trust inventories, focused checks and combined build/lint gates |

Tasks execute sequentially: ownership extraction, semantic-consumer/import enforcement, then compatibility/trust qualification and documentation. Each task preserves a buildable ordinary authoring facade; the final task owns combined verification. Existing verified repository lint debt is reported against the captured baseline under the Lean guidelines, never represented as a clean lint pass.
